package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid"
)

type RollbackResult struct {
	RevertedOps   []string
	RevertedFiles []string
	BytesWritten  int
	Duration      time.Duration
	Warnings      []string
}

func BeginRun(db *sql.DB, meta map[string]any) (string, error) {
	// Enforce retention policy before starting a new run
	if retentionErr := EnforceRetentionPolicy(db); retentionErr != nil {
		return "", fmt.Errorf("BeginRun: failed to enforce retention policy: %w", retentionErr)
	}
	runID := uuid.NewString()
	// Generate a ULID for public_ulid
	publicULID := ulid.MustNew(ulid.Timestamp(time.Now()), ulid.Monotonic(rand.Reader, 0)).String()
	startedAt := time.Now().UnixMilli() // Use millisecond precision
	metricsJSON, marshalErr := json.Marshal(meta["metrics"])
	if marshalErr != nil {
		// Default to empty JSON object if marshaling fails
		metricsJSON = []byte("{}")
	}
	_, execErr := execWithRetry(
		db,
		`INSERT INTO runs (id, public_ulid, repo, branch, commit_base, status, started_at, metrics_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?) `,
		runID,
		publicULID,
		meta["repo"],
		meta["branch"],
		meta["commit_base"],
		"started",
		startedAt,
		string(metricsJSON),
	)
	if execErr != nil {
		return "", fmt.Errorf("BeginRun insert: %w", execErr)
	}
	return runID, nil
}
func AppendOp(db *sql.DB, runID, fileID, kind string) (string, error) {
	opID := uuid.NewString()
	startedAt := time.Now().UnixMilli()
	// Fetch and increment next_op_seq for the run
	tx, beginErr := db.Begin()
	if beginErr != nil {
		return "", fmt.Errorf("AppendOp tx: %w", beginErr)
	}
	defer tx.Rollback()
	var seq int64
	row := tx.QueryRow(`SELECT next_op_seq FROM runs WHERE id = ?`, runID)
	scanErr := row.Scan(&seq)
	if scanErr != nil {
		return "", fmt.Errorf("AppendOp: failed to get next_op_seq: %w", scanErr)
	}
	seq++ // Increment for the current operation
	_, updateErr := execWithRetryTx(
		tx,
		`UPDATE runs SET next_op_seq = ? WHERE id = ?`,
		seq,
		runID,
	)
	if updateErr != nil {
		return "", fmt.Errorf("AppendOp: failed to update next_op_seq: %w", updateErr)
	}
	_, insertErr := execWithRetryTx(
		tx,
		`INSERT INTO operations (id, run_id, file_id, seq, kind, status, started_at) VALUES (?, ?, ?, ?, ?, ?, ?) `,
		opID,
		runID,
		fileID,
		seq,
		kind,
		"pending", // Initial status
		startedAt,
	)
	if insertErr != nil {
		return "", fmt.Errorf("AppendOp insert: %w", insertErr)
	}
	if commitErr := tx.Commit(); commitErr != nil {
		return "", fmt.Errorf("AppendOp commit: %w", commitErr)
	}
	return opID, nil
}

type Patch struct {
	OpID         string
	FileID       string
	Algo         string
	ForwardBlob  []byte
	ReverseBlob  []byte
	BytesAdded   int
	BytesRemoved int
}

func RecordPatches(
	db *sql.DB,
	patches []Patch,
) error {
	fmt.Printf("MASTER KEY %+v\n", os.Getenv("MORFX_MASTER_KEY"))
	mode, _, _, encryptor, encErr := getEncryptionConfig()
	if encErr != nil {
		return fmt.Errorf("RecordPatches: failed to get encryption config: %w", encErr)
	}
	tx, beginErr := db.Begin()
	if beginErr != nil {
		return fmt.Errorf("RecordPatches tx: %w", beginErr)
	}
	defer tx.Rollback()
	for _, p := range patches {
		patchID := uuid.NewString()
		var encryptedForward, encryptedReverse []byte
		var encAlgo string = "PLAINTEXT"
		var keyVersion int = 0 // Default key version
		var nonce []byte
		// Construct AAD from relevant metadata
		aad := fmt.Appendf(nil, "%s-%s-%s", p.OpID, p.FileID, p.Algo)
		if mode != "off" && encryptor != nil {
			nonce = make([]byte, encryptor.NonceSize())
			if _, randErr := rand.Read(nonce); randErr != nil {
				return fmt.Errorf("RecordPatches: failed to generate nonce: %w", randErr)
			}
			keyVersion = GetGlobalContext().ActiveKeyVersion
			currentKey, ok := globalKeyring[keyVersion]
			if !ok {
				return fmt.Errorf("RecordPatches: key for version %d not found in keyring", keyVersion)
			}
			var encryptErr error
			encryptedForward, encryptErr = encryptor.Encrypt(currentKey, nonce, p.ForwardBlob, aad)
			if encryptErr != nil {
				return fmt.Errorf("RecordPatches: failed to encrypt forward blob: %w", encryptErr)
			}
			encryptedReverse, encryptErr = encryptor.Encrypt(currentKey, nonce, p.ReverseBlob, aad)
			if encryptErr != nil {
				return fmt.Errorf("RecordPatches: failed to encrypt reverse blob: %w", encryptErr)
			}
			encAlgo = encryptor.Algo()
		} else {
			encryptedForward = p.ForwardBlob
			encryptedReverse = p.ReverseBlob
		}
		_, execErr := execWithRetryTx(
			tx,
			`INSERT INTO patches (id, op_id, file_id, algo, forward_blob, reverse_blob, bytes_added, bytes_removed, enc_algo, key_version, nonce) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) `,
			patchID,
			p.OpID,
			p.FileID,
			p.Algo,
			encryptedForward,
			encryptedReverse,
			p.BytesAdded,
			p.BytesRemoved,
			encAlgo,
			keyVersion,
			nonce,
		)
		if execErr != nil {
			return fmt.Errorf("RecordPatches insert: %w", execErr)
		}
	}
	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("RecordPatches commit: %w", commitErr)
	}
	return nil
}
func Checkpoint(db *sql.DB, runID, name string, meta map[string]any) error {
	checkpointID := uuid.NewString()
	createdAt := time.Now().UnixMilli() // Use millisecond precision
	metaJSON, marshalErr := json.Marshal(meta)
	if marshalErr != nil {
		// Default to empty JSON object if marshaling fails
		metaJSON = []byte("{}")
	}
	tx, beginErr := db.Begin()
	if beginErr != nil {
		return fmt.Errorf("Checkpoint tx: %w", beginErr)
	}
	defer tx.Rollback()
	_, execErr := execWithRetryTx(
		tx,
		`INSERT INTO checkpoints (id, run_id, name, created_at, meta_json) VALUES (?, ?, ?, ?, ?)`,
		checkpointID,
		runID,
		name,
		createdAt,
		string(metaJSON),
	)
	if execErr != nil {
		return fmt.Errorf("Checkpoint insert: %w", execErr)
	}
	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("Checkpoint commit: %w", commitErr)
	}
	return nil
}

// RecordFile records a file's metadata.
func RecordFile(
	db *sql.DB,
	runID, fileID, path, lang string,
	size int64,
	hashBefore, hashAfter string,
	status string,
) error {
	_, execErr := execWithRetry(
		db,
		`INSERT INTO files (id, run_id, path, lang, size, hash_before, hash_after, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?) `,
		fileID,
		runID,
		path,
		lang,
		size,
		hashBefore,
		hashAfter,
		status,
	)
	if execErr != nil {
		return fmt.Errorf("RecordFile insert: %w", execErr)
	}
	return nil
}
func Rollback(db *sql.DB, opOrCheckpoint string, dryRun, strict bool) (*RollbackResult, error) {
	result := &RollbackResult{
		RevertedOps:   []string{},
		RevertedFiles: []string{},
		Warnings:      []string{},
	}
	startTime := time.Now()
	tx, beginErr := db.Begin()
	if beginErr != nil {
		return nil, fmt.Errorf("Rollback tx: %w", beginErr)
	}
	defer tx.Rollback() // Rollback on error
	var runID string
	var startSeq int64 // Inclusive starting sequence for rollback
	// Try to find by op_id first
	row := tx.QueryRow(`SELECT run_id, seq FROM operations WHERE id = ?`, opOrCheckpoint)
	scanErr := row.Scan(&runID, &startSeq)
	if scanErr == nil {
		// If op_id is found, startSeq is the seq of that op. Rollback from this op onwards.
	} else if scanErr == sql.ErrNoRows {
		// Not an op_id, try checkpoint name
		var checkpointCreatedAt int64
		row = tx.QueryRow(`SELECT run_id, created_at FROM checkpoints WHERE run_id = ? AND name = ?`, runID, opOrCheckpoint)
		checkpointErr := row.Scan(&runID, &checkpointCreatedAt)
		if checkpointErr == nil {
			// Found checkpoint, now find the latest operation seq before or at this checkpoint
			var maxSeqBeforeCheckpoint sql.NullInt64
			row = tx.QueryRow(`SELECT MAX(seq) FROM operations WHERE run_id = ? AND started_at <= ?`, runID, checkpointCreatedAt)
			maxSeqErr := row.Scan(&maxSeqBeforeCheckpoint)
			if maxSeqErr != nil && maxSeqErr != sql.ErrNoRows {
				return nil, fmt.Errorf("Rollback: failed to get max seq before checkpoint: %w", maxSeqErr)
			}
			// Rollback operations *after* this checkpoint. If no ops before, startSeq remains 0.
			if maxSeqBeforeCheckpoint.Valid {
				startSeq = maxSeqBeforeCheckpoint.Int64 + 1
			} else {
				startSeq = 0 // No operations before checkpoint, so rollback all
			}
		} else if checkpointErr == sql.ErrNoRows {
			return nil, fmt.Errorf("Rollback: '%s' is neither a valid operation ID nor a checkpoint name", opOrCheckpoint)
		} else {
			return nil, fmt.Errorf("Rollback: failed to query checkpoint: %w", checkpointErr)
		}
	} else {
		return nil, fmt.Errorf("Rollback: failed to query operation: %w", scanErr)
	}
	// Fetch patches to rollback in LIFO order
	// We need to get patches for operations with seq >= startSeq and for the specific runID.
	// We also need the file path to apply the patch.
	query := `
		SELECT
			p.reverse_blob,
			f.path,
			o.id, -- operation ID to update its status later
			f.id,  -- file ID to update its status later
			p.bytes_removed, -- bytes removed by this patch (bytes added by rollback)
			p.enc_algo,
			p.key_version,
			p.nonce
		FROM patches p
		JOIN operations o ON p.op_id = o.id
		JOIN files f ON p.file_id = f.id
		WHERE o.run_id = ? AND o.seq >= ? AND o.status != 'rolled_back'
		ORDER BY o.seq DESC
	`
	rows, queryErr := tx.Query(query, runID, startSeq)
	if queryErr != nil {
		return nil, fmt.Errorf("Rollback: failed to query patches: %w", queryErr)
	}
	defer rows.Close()
	for rows.Next() {
		var reverseBlob []byte
		var filePath string
		var opIDToUpdate string
		var fileIDToUpdate string
		var bytesAddedByRollback int
		var encAlgo string
		var keyVersion int
		var nonce []byte
		if scanErr := rows.Scan(&reverseBlob, &filePath, &opIDToUpdate, &fileIDToUpdate, &bytesAddedByRollback, &encAlgo, &keyVersion, &nonce); scanErr != nil {
			return nil, fmt.Errorf("Rollback: failed to scan patch row: %w", scanErr)
		}
		// Construct AAD for decryption
		aad := fmt.Appendf(nil, "%s-%s-%s", opIDToUpdate, fileIDToUpdate, "binary") // Assuming algo is binary for now
		// Decrypt reverseBlob if encrypted
		if encAlgo != "PLAINTEXT" {
			_, _, _, encryptor, encErr := getEncryptionConfig()
			if encErr != nil {
				return nil, fmt.Errorf("Rollback: failed to get encryption config for decryption: %w", encErr)
			}
			if encryptor == nil { // Should not happen if encAlgo is not PLAINTEXT
				return nil, fmt.Errorf("Rollback: encryption enabled but no encryptor found for decryption")
			}
			currentKey, ok := globalKeyring[keyVersion]
			if !ok {
				return nil, fmt.Errorf("Rollback: key for version %d not found in keyring", keyVersion)
			}
			var decryptErr error
			reverseBlob, decryptErr = encryptor.Decrypt(currentKey, nonce, reverseBlob, aad)
			if decryptErr != nil {
				return nil, fmt.Errorf("Rollback: failed to decrypt reverse blob: %w", decryptErr)
			}
		}
		// Test Hook: CP-A (after writing all *.tmp, before renaming)
		if os.Getenv("MORFX_CRASH_POINT") == "CP-A" {
			os.Exit(137)
		}
		if !dryRun {
			// Apply reverse_blob to filePath using a temporary file and atomic rename
			// This ensures atomicity between file system changes and database commit.
			// IMPORTANT: This assumes reverse_blob contains the full content of the file
			// as it was *before* the patch was applied. If it's a diff, a more complex
			// patching mechanism (e.g., using a patch library) would be required here.
			tempFile, createErr := os.CreateTemp("", "rollback-temp-")
			if createErr != nil {
				return nil, fmt.Errorf("Rollback: failed to create temporary file: %w", createErr)
			}
			defer os.Remove(tempFile.Name()) // Clean up temp file on exit
			_, writeErr := tempFile.Write(reverseBlob)
			if writeErr != nil {
				tempFile.Close()
				return nil, fmt.Errorf("Rollback: failed to write to temporary file: %w", writeErr)
			}
			if closeErr := tempFile.Close(); closeErr != nil {
				return nil, fmt.Errorf("Rollback: failed to close temporary file: %w", closeErr)
			}
			// Atomically replace the original file with the temporary file
			if renameErr := os.Rename(tempFile.Name(), filePath); renameErr != nil {
				if strict {
					return nil, fmt.Errorf("Rollback: failed to atomically rename file %s: %w", filePath, renameErr)
				} else {
					result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to apply patch to %s: %v", filePath, renameErr))
					continue // Continue with other patches if not strict
				}
			}
		}
		// Test Hook: CP-B (after renaming tmp -> real, before COMMIT)
		if os.Getenv("MORFX_CRASH_POINT") == "CP-B" {
			os.Exit(137)
		}
		// Update file status and hash_after in the database
		// For simplicity, we'll re-calculate the hash after rollback. In a real scenario,
		// you might store the hash of the reverse_blob or the original file content.
		// For now, we'll just mark the file status as 'rolled_back' and clear hash_after.
		_, execErr := tx.Exec(`UPDATE files SET status = ?, hash_after = ? WHERE id = ?`, "rolled_back", "", fileIDToUpdate)
		if execErr != nil {
			return nil, fmt.Errorf("Rollback: failed to update file status for %s: %w", fileIDToUpdate, execErr)
		}
		result.RevertedOps = append(result.RevertedOps, opIDToUpdate)
		result.RevertedFiles = append(result.RevertedFiles, filePath)
		result.BytesWritten += bytesAddedByRollback
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("Rollback: error iterating patches: %w", rowsErr)
	}
	// Update status of rolled-back operations
	if len(result.RevertedOps) > 0 {
		// Create a string of placeholders for the IN clause
		placeholders := make([]string, len(result.RevertedOps))
		args := make([]any, len(result.RevertedOps)+1)
		for i, id := range result.RevertedOps {
			placeholders[i] = "?"
			args[i+1] = id
		}
		args[0] = "rolled_back" // The status to set
		updateQuery := fmt.Sprintf(`UPDATE operations SET status = ? WHERE id IN (%s)`, strings.Join(placeholders, ","))
		_, execErr := tx.Exec(updateQuery, args...)
		if execErr != nil {
			return nil, fmt.Errorf("Rollback: failed to update operation status: %w", execErr)
		}
	}
	if commitErr := tx.Commit(); commitErr != nil {
		return nil, fmt.Errorf("Rollback commit: %w", commitErr)
	}
	result.Duration = time.Since(startTime)
	return result, nil
}
func AppendDiagnostics(db *sql.DB, opID string, list []map[string]any) error {
	tx, beginErr := db.Begin()
	if beginErr != nil {
		return fmt.Errorf("AppendDiagnostics tx: %w", beginErr)
	}
	defer tx.Rollback()
	for _, diag := range list {
		diagID := uuid.NewString()
		_, execErr := execWithRetryTx(
			tx,
			`INSERT INTO diagnostics (id, op_id, file, line, col, severity, code, message, raw_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			diagID,
			opID,
			diag["file"],
			diag["line"],
			diag["col"],
			diag["severity"],
			diag["code"],
			diag["message"],
			diag["raw_json"],
		)
		if execErr != nil {
			return fmt.Errorf("AppendDiagnostics insert: %w", execErr)
		}
	}
	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("AppendDiagnostics commit: %w", commitErr)
	}
	return nil
}
func GetRunSummary(db *sql.DB, runID string) (map[string]any, error) {
	summary := make(map[string]any)
	// Get basic run information
	var repo, branch string
	if scanErr := db.QueryRow(`SELECT repo, branch FROM runs WHERE id = ?`, runID).Scan(&repo, &branch); scanErr != nil {
		return nil, fmt.Errorf("GetRunSummary basic info: %w", scanErr)
	}
	summary["id"] = runID
	summary["repo"] = repo
	summary["branch"] = branch
	var opCount int64
	if scanErr := db.QueryRow(`SELECT COUNT(*) FROM operations WHERE run_id = ?`, runID).Scan(&opCount); scanErr != nil {
		return nil, fmt.Errorf("GetRunSummary op_count: %w", scanErr)
	}
	summary["op_count"] = opCount
	var bytesAdded, bytesRemoved sql.NullInt64
	if scanErr := db.QueryRow(`SELECT SUM(bytes_added), SUM(bytes_removed) FROM patches WHERE file_id IN (SELECT id FROM files WHERE run_id = ?)`, runID).
		Scan(&bytesAdded, &bytesRemoved); scanErr != nil {
		return nil, fmt.Errorf("GetRunSummary bytes: %w", scanErr)
	}
	summary["bytes_added"] = bytesAdded.Int64
	summary["bytes_removed"] = bytesRemoved.Int64
	// Calculate LOC (sum bytes)
	var totalLOC sql.NullInt64
	if scanErr := db.QueryRow(`SELECT SUM(size) FROM files WHERE run_id = ?`, runID).Scan(&totalLOC); scanErr != nil {
		return nil, fmt.Errorf("GetRunSummary total_loc: %w", scanErr)
	}
	summary["total_loc"] = totalLOC.Int64
	// Calculate safe_change % and guardrail pass rate
	var opsWithErrorsOrWarnings int64
	if scanErr := db.QueryRow(`
		SELECT COUNT(DISTINCT op_id) FROM diagnostics
		WHERE op_id IN (SELECT id FROM operations WHERE run_id = ?)
		AND (severity = 'error' OR severity = 'warning')
	`, runID).Scan(&opsWithErrorsOrWarnings); scanErr != nil {
		return nil, fmt.Errorf("GetRunSummary ops_with_errors_or_warnings: %w", scanErr)
	}
	var totalOps int64
	if scanErr := db.QueryRow(`SELECT COUNT(*) FROM operations WHERE run_id = ?`, runID).Scan(&totalOps); scanErr != nil {
		return nil, fmt.Errorf("GetRunSummary total_ops: %w", scanErr)
	}
	safeChangePercent := 0.0
	guardrailPassRate := 0.0
	if totalOps > 0 {
		safeChangePercent = (float64(totalOps-opsWithErrorsOrWarnings) / float64(totalOps)) * 100.0
		guardrailPassRate = 100.0 - safeChangePercent
	}
	summary["safe_change_percent"] = safeChangePercent
	summary["guardrail_pass_rate"] = guardrailPassRate
	// Calculate duration
	var startedAt, finishedAt sql.NullInt64
	if scanErr := db.QueryRow(`SELECT started_at, finished_at FROM runs WHERE id = ?`, runID).Scan(&startedAt, &finishedAt); scanErr != nil {
		return nil, fmt.Errorf("GetRunSummary duration: %w", scanErr)
	}
	if startedAt.Valid && finishedAt.Valid {
		summary["duration_ms"] = finishedAt.Int64 - startedAt.Int64
	} else {
		summary["duration_ms"] = 0
	}
	// op_summary: per file_id/op_id: bytes ±, #diagnostics
	opSummary := []map[string]any{}
	opSummaryRows, queryErr := db.Query(`
		SELECT
			o.id, o.file_id, o.kind, o.status,
			SUM(p.bytes_added) AS total_bytes_added,
			SUM(p.bytes_removed) AS total_bytes_removed,
			COUNT(d.id) AS diagnostic_count
		FROM operations o
		LEFT JOIN patches p ON o.id = p.op_id
		LEFT JOIN diagnostics d ON o.id = d.op_id
		WHERE o.run_id = ?
		GROUP BY o.id, o.file_id, o.kind, o.status
		ORDER BY o.started_at ASC
	`, runID)
	if queryErr != nil {
		return nil, fmt.Errorf("GetRunSummary op_summary query: %w", queryErr)
	}
	defer opSummaryRows.Close()
	for opSummaryRows.Next() {
		var opID, fileID, kind, status string
		var totalBytesAdded, totalBytesRemoved sql.NullInt64
		var diagnosticCount sql.NullInt64
		if scanErr := opSummaryRows.Scan(&opID, &fileID, &kind, &status, &totalBytesAdded, &totalBytesRemoved, &diagnosticCount); scanErr != nil {
			return nil, fmt.Errorf("GetRunSummary op_summary scan: %w", scanErr)
		}
		opSummary = append(opSummary, map[string]any{
			"op_id":            opID,
			"file_id":          fileID,
			"kind":             kind,
			"status":           status,
			"bytes_added":      totalBytesAdded.Int64,
			"bytes_removed":    totalBytesRemoved.Int64,
			"diagnostic_count": diagnosticCount.Int64,
		})
	}
	if rowsErr := opSummaryRows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("GetRunSummary op_summary rows error: %w", rowsErr)
	}
	summary["op_summary"] = opSummary
	// error_hotspots: top files by severity in ('error','warning')
	errorHotspots := []map[string]any{}
	errorHotspotsRows, queryErr := db.Query(`
		SELECT
			f.path,
			COUNT(d.id) AS error_warning_count
		FROM diagnostics d
		JOIN operations o ON d.op_id = o.id
		JOIN files f ON o.file_id = f.id
		WHERE o.run_id = ? AND (d.severity = 'error' OR d.severity = 'warning')
		GROUP BY f.path
		ORDER BY error_warning_count DESC
		LIMIT 10 -- Top 10 hotspots
	`, runID)
	if queryErr != nil {
		return nil, fmt.Errorf("GetRunSummary error_hotspots query: %w", queryErr)
	}
	defer errorHotspotsRows.Close()
	for errorHotspotsRows.Next() {
		var filePath string
		var count int64
		if scanErr := errorHotspotsRows.Scan(&filePath, &count); scanErr != nil {
			return nil, fmt.Errorf("GetRunSummary error_hotspots scan: %w", scanErr)
		}
		errorHotspots = append(errorHotspots, map[string]any{
			"file_path": filePath,
			"count":     count,
		})
	}
	if rowsErr := errorHotspotsRows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("GetRunSummary error_hotspots rows error: %w", rowsErr)
	}
	summary["error_hotspots"] = errorHotspots
	return summary, nil
}

// EnforceRetentionPolicy archives old runs based on the configured retention limit.
func EnforceRetentionPolicy(db *sql.DB) error {
	ctx := GetGlobalContext()
	retentionRuns := ctx.RetentionRuns
	if retentionRuns == 0 {
		return nil // Retention policy disabled
	}
	// Get the IDs of runs that exceed the retention limit
	rows, queryErr := db.Query(`
		SELECT id FROM runs
		ORDER BY started_at DESC
		LIMIT -1 OFFSET ?
	`, retentionRuns)
	if queryErr != nil {
		return fmt.Errorf("EnforceRetentionPolicy: failed to query old runs: %w", queryErr)
	}
	defer rows.Close()
	var runIDsToArchive []string
	for rows.Next() {
		var runID string
		if scanErr := rows.Scan(&runID); scanErr != nil {
			return fmt.Errorf("EnforceRetentionPolicy: failed to scan run ID: %w", scanErr)
		}
		runIDsToArchive = append(runIDsToArchive, runID)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return fmt.Errorf("EnforceRetentionPolicy: error iterating run IDs: %w", rowsErr)
	}
	if len(runIDsToArchive) > 0 {
		// Archive (or delete) these runs
		// For now, we'll just update their status to 'archived'
		placeholders := make([]string, len(runIDsToArchive))
		args := make([]any, len(runIDsToArchive))
		for i, id := range runIDsToArchive {
			placeholders[i] = "?"
			args[i] = id
		}
		updateQuery := fmt.Sprintf(`UPDATE runs SET status = 'archived' WHERE id IN (%s)`, strings.Join(placeholders, ","))

		_, err := execWithRetry(db, updateQuery, args...)
		if err != nil {
			return fmt.Errorf("EnforceRetentionPolicy: failed to archive runs: %w", err)
		}
	}

	return nil
}

// LogEntry represents a single log entry.
type LogEntry struct {
	OpID  string
	TS    int64
	Level string
	Text  string
}

// SearchLogs searches log entries using FTS5 or LIKE queries.
func SearchLogs(db *sql.DB, query string, useFTS bool) ([]LogEntry, error) {
	var logs []LogEntry
	var rows *sql.Rows
	var err error

	if useFTS {
		// Try FTS5 query
		rows, err = db.Query("SELECT op_id, ts, level, text FROM logs WHERE logs MATCH ?", query)
		if err != nil {
			// Fallback to LIKE if FTS5 query fails (e.g., syntax error, or if table is not FTS5)
			fmt.Printf("FTS5 query failed, falling back to LIKE: %v\n", err) // TODO: Replace with proper logging
			useFTS = false
		}
	}

	if !useFTS {
		// Use LIKE query
		rows, err = db.Query("SELECT op_id, ts, level, text FROM logs WHERE text LIKE ?", "%"+query+"%")
		if err != nil {
			return nil, fmt.Errorf("failed to execute LIKE query: %w", err)
		}
	}

	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()

	for rows.Next() {
		var logEntry LogEntry
		if err := rows.Scan(&logEntry.OpID, &logEntry.TS, &logEntry.Level, &logEntry.Text); err != nil {
			return nil, fmt.Errorf("failed to scan log entry: %w", err)
		}
		logs = append(logs, logEntry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating log entries: %w", err)
	}

	return logs, nil
}
