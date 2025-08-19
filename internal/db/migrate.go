package db

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// Migrate applies database schema migrations.
func Migrate(db *sql.DB) error {
	// Enable foreign key support
	_, err := db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	sqlStmt := `
	CREATE TABLE IF NOT EXISTS runs (
		id TEXT PRIMARY KEY,
		public_ulid TEXT NOT NULL UNIQUE,
		repo TEXT,
		branch TEXT,
		commit_base TEXT,
		status TEXT NOT NULL,
		started_at INTEGER NOT NULL,
		finished_at INTEGER,
		metrics_json TEXT,
		next_op_seq INTEGER DEFAULT 0,
		-- Morfx Core Spec additions
		engine_version TEXT,
		total_operations INTEGER DEFAULT 0,
		total_files INTEGER DEFAULT 0,
		total_bytes_processed INTEGER DEFAULT 0,
		total_lines_processed INTEGER DEFAULT 0,
		total_matches_found INTEGER DEFAULT 0,
		total_edits_applied INTEGER DEFAULT 0,
		total_overlaps_detected INTEGER DEFAULT 0,
		fuzzy_matches_used INTEGER DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_runs_status_started_at ON runs (status, started_at DESC);

	CREATE TABLE IF NOT EXISTS files (
		id TEXT PRIMARY KEY,
		run_id TEXT NOT NULL,
		path TEXT NOT NULL,
		lang TEXT,
		size INTEGER,
		hash_before TEXT,
		hash_after TEXT,
		status TEXT NOT NULL,
		FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_files_run_id_path ON files (run_id, path);

	CREATE TABLE IF NOT EXISTS operations (
		id TEXT PRIMARY KEY,
		run_id TEXT NOT NULL,
		file_id TEXT NOT NULL,
		seq INTEGER NOT NULL,
		kind TEXT NOT NULL,
		status TEXT NOT NULL,
		started_at INTEGER NOT NULL,
		finished_at INTEGER,
		stats_json TEXT,
		-- Morfx Core Spec additions
		language TEXT,
		query TEXT,
		operation TEXT,
		replacement TEXT,
		resolved_operation TEXT,
		code_hash_before TEXT,
		code_hash_after TEXT,
		bytes_processed INTEGER DEFAULT 0,
		lines_processed INTEGER DEFAULT 0,
		matches_found INTEGER DEFAULT 0,
		edits_applied INTEGER DEFAULT 0,
		overlaps_detected INTEGER DEFAULT 0,
		fuzzy_used BOOLEAN DEFAULT FALSE,
		fuzzy_original_query TEXT,
		fuzzy_resolved_query TEXT,
		fuzzy_confidence REAL,
		fuzzy_distance INTEGER,
		duration_ms INTEGER,
		FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE,
		FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_operations_run_id_seq ON operations (run_id, seq);
	CREATE INDEX IF NOT EXISTS idx_operations_run_id_status ON operations (run_id, status);

	CREATE TABLE IF NOT EXISTS patches (
		id TEXT PRIMARY KEY,
		op_id TEXT NOT NULL,
		file_id TEXT NOT NULL,
		algo TEXT NOT NULL,
		forward_blob BLOB,
		reverse_blob BLOB,
		bytes_added INTEGER,
		bytes_removed INTEGER,
		-- Encryption fields (Morfx Core Spec)
		enc_algo TEXT NOT NULL DEFAULT 'PLAINTEXT',
		key_version INTEGER NOT NULL DEFAULT 0,
		nonce BLOB,
		aad BLOB,
		-- Additional patch metadata
		diff_unified TEXT,
		start_byte INTEGER,
		end_byte INTEGER,
		lines_added INTEGER DEFAULT 0,
		lines_removed INTEGER DEFAULT 0,
		compression_algo TEXT DEFAULT 'NONE',
		original_size INTEGER,
		compressed_size INTEGER,
		FOREIGN KEY (op_id) REFERENCES operations(id) ON DELETE CASCADE,
		FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_patches_op_id ON patches (op_id);

	CREATE TABLE IF NOT EXISTS checkpoints (
		id TEXT PRIMARY KEY,
		run_id TEXT NOT NULL,
		name TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		meta_json TEXT,
		FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_checkpoints_run_id_name ON checkpoints (run_id, name);

	CREATE TABLE IF NOT EXISTS diagnostics (
		id TEXT PRIMARY KEY,
		op_id TEXT NOT NULL,
		file TEXT,
		line INTEGER,
		col INTEGER,
		severity TEXT NOT NULL,
		code TEXT,
		message TEXT,
		raw_json TEXT,
		FOREIGN KEY (op_id) REFERENCES operations(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_diagnostics_op_id_severity ON diagnostics (op_id, severity);

	CREATE TABLE IF NOT EXISTS keys (
		key_version INTEGER PRIMARY KEY,
		created_at INTEGER NOT NULL,
		algo TEXT NOT NULL,
		key_hash BLOB NOT NULL,
		-- Morfx Core Spec encryption additions
		key_material BLOB NOT NULL,
		rotated_at INTEGER,
		is_active INTEGER DEFAULT 1,
		derivation_salt BLOB,
		derivation_info TEXT
	);
	
	-- Create table for edit conflicts and overlaps
	CREATE TABLE IF NOT EXISTS edit_conflicts (
		id TEXT PRIMARY KEY,
		op_id TEXT NOT NULL,
		conflict_type TEXT NOT NULL,
		start_byte INTEGER NOT NULL,
		end_byte INTEGER NOT NULL,
		conflicting_op_id TEXT,
		resolution TEXT,
		created_at INTEGER NOT NULL,
		FOREIGN KEY (op_id) REFERENCES operations(id) ON DELETE CASCADE,
		FOREIGN KEY (conflicting_op_id) REFERENCES operations(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_edit_conflicts_op_id ON edit_conflicts (op_id);
	
	-- Create table for fuzzy match details
	CREATE TABLE IF NOT EXISTS fuzzy_matches (
		id TEXT PRIMARY KEY,
		op_id TEXT NOT NULL,
		original_query TEXT NOT NULL,
		resolved_query TEXT NOT NULL,
		confidence REAL NOT NULL,
		distance INTEGER NOT NULL,
		algorithm TEXT NOT NULL,
		variations_tried INTEGER DEFAULT 0,
		created_at INTEGER NOT NULL,
		FOREIGN KEY (op_id) REFERENCES operations(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_fuzzy_matches_op_id ON fuzzy_matches (op_id);
	CREATE INDEX IF NOT EXISTS idx_fuzzy_matches_confidence ON fuzzy_matches (confidence DESC);
	`

	_, err = db.Exec(sqlStmt)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	// Check for FTS5 support by attempting to create a dummy FTS5 table
	_, err = db.Exec("CREATE VIRTUAL TABLE IF NOT EXISTS _dummy_fts_test USING fts5(content);")
	if err == nil {
		// FTS5 is supported, create the actual logs table as FTS5
		_, err = db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS logs USING fts5(op_id, ts, level, text);`)
		if err != nil {
			return fmt.Errorf("failed to create FTS5 logs table: %w", err)
		}
		// Drop the dummy table
		_, err = db.Exec("DROP TABLE IF EXISTS _dummy_fts_test;")
		if err != nil {
			return fmt.Errorf("failed to drop dummy FTS5 table: %w", err)
		}
	} else if strings.Contains(err.Error(), "no such module: fts5") {
		// FTS5 is not supported, fallback to a regular table
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS logs (
				op_id TEXT NOT NULL,
				ts INTEGER NOT NULL,
				level TEXT NOT NULL,
				text TEXT,
				FOREIGN KEY (op_id) REFERENCES operations(id) ON DELETE CASCADE
			);
			CREATE INDEX IF NOT EXISTS idx_logs_op_id_ts ON logs (op_id, ts);
			CREATE INDEX IF NOT EXISTS idx_logs_ts ON logs (ts);
		`)
		if err != nil {
			return fmt.Errorf("failed to create regular logs table: %w", err)
		}
	} else {
		return fmt.Errorf("failed to check FTS5 support: %w", err)
	}

	return nil
}
