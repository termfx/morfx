package db

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// Ensure .morfx directory exists and clean it up
	_ = os.RemoveAll(".morfx")
	if err := os.MkdirAll(".morfx", 0o700); err != nil {
		t.Fatalf("Failed to create .morfx directory: %v", err)
	}

	dbConn, err := Open()
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if err := Migrate(dbConn.DB); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	cleanup := func() {
		dbConn.Close()
		os.RemoveAll(".morfx")
	}

	return dbConn.DB, cleanup
}

func TestBeginRun(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name    string
		meta    map[string]any
		wantErr bool
	}{
		{
			name: "valid metadata",
			meta: map[string]any{
				"repo":        "test-repo",
				"branch":      "main",
				"commit_base": "abc123",
				"metrics":     map[string]any{"files": 10, "lines": 1000},
			},
			wantErr: false,
		},
		{
			name: "minimal metadata",
			meta: map[string]any{
				"repo":        "test-repo",
				"branch":      "main",
				"commit_base": "def456",
			},
			wantErr: false,
		},
		{
			name: "invalid metrics",
			meta: map[string]any{
				"repo":        "test-repo",
				"branch":      "main",
				"commit_base": "ghi789",
				"metrics":     make(chan int), // unmarshalable type
			},
			wantErr: false, // Should not error, just use empty JSON
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runID, err := BeginRun(db, tt.meta)
			if (err != nil) != tt.wantErr {
				t.Errorf("BeginRun() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify runID is a valid UUID
				if _, err := uuid.Parse(runID); err != nil {
					t.Errorf("BeginRun() returned invalid UUID: %v", err)
				}

				// Verify run was inserted into database
				var count int
				err = db.QueryRow("SELECT COUNT(*) FROM runs WHERE id = ?", runID).Scan(&count)
				if err != nil {
					t.Errorf("Failed to query runs table: %v", err)
				}
				if count != 1 {
					t.Errorf("Expected 1 run in database, got %d", count)
				}
			}
		})
	}
}

func TestAppendOp(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a run first
	meta := map[string]any{
		"repo":        "test-repo",
		"branch":      "main",
		"commit_base": "abc123",
	}
	runID, err := BeginRun(db, meta)
	if err != nil {
		t.Fatalf("BeginRun failed: %v", err)
	}

	// Create file records first
	fileID1 := uuid.NewString()
	fileID2 := uuid.NewString()

	err = RecordFile(db, runID, fileID1, "file1.go", "go", 1024, "hash1", "hash2", "modified")
	if err != nil {
		t.Fatalf("RecordFile failed: %v", err)
	}

	err = RecordFile(db, runID, fileID2, "file2.py", "python", 2048, "hash3", "hash4", "created")
	if err != nil {
		t.Fatalf("RecordFile failed: %v", err)
	}

	tests := []struct {
		name    string
		runID   string
		fileID  string
		kind    string
		wantErr bool
	}{
		{
			name:    "valid operation",
			runID:   runID,
			fileID:  fileID1,
			kind:    "modify",
			wantErr: false,
		},
		{
			name:    "another valid operation",
			runID:   runID,
			fileID:  fileID2,
			kind:    "create",
			wantErr: false,
		},
		{
			name:    "invalid run ID",
			runID:   "invalid-uuid",
			fileID:  "file3.js",
			kind:    "delete",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opID, err := AppendOp(db, tt.runID, tt.fileID, tt.kind)
			if (err != nil) != tt.wantErr {
				t.Errorf("AppendOp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify opID is a valid UUID
				if _, err := uuid.Parse(opID); err != nil {
					t.Errorf("AppendOp() returned invalid UUID: %v", err)
				}

				// Verify operation was inserted into database
				var count int
				err = db.QueryRow("SELECT COUNT(*) FROM operations WHERE id = ?", opID).Scan(&count)
				if err != nil {
					t.Errorf("Failed to query ops table: %v", err)
				}
				if count != 1 {
					t.Errorf("Expected 1 operation in database, got %d", count)
				}
			}
		})
	}
}

func TestRecordPatches(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Set global context with encryption disabled for this test
	SetGlobalContext(&Context{
		DBPath:         ".morfx.db",
		EncryptionMode: "off",
	})

	// Create a run and operation first
	meta := map[string]any{
		"repo":        "test-repo",
		"branch":      "main",
		"commit_base": "abc123",
	}
	runID, err := BeginRun(db, meta)
	if err != nil {
		t.Fatalf("BeginRun failed: %v", err)
	}

	// Create file records first
	fileID1 := uuid.NewString()
	fileID2 := uuid.NewString()
	fileID3 := uuid.NewString()

	err = RecordFile(db, runID, fileID1, "test.go", "go", 1024, "hash1", "hash2", "modified")
	if err != nil {
		t.Fatalf("RecordFile failed: %v", err)
	}

	err = RecordFile(db, runID, fileID2, "test1.go", "go", 2048, "hash3", "hash4", "modified")
	if err != nil {
		t.Fatalf("RecordFile failed: %v", err)
	}

	err = RecordFile(db, runID, fileID3, "test2.go", "go", 3072, "hash5", "hash6", "modified")
	if err != nil {
		t.Fatalf("RecordFile failed: %v", err)
	}

	opID, err := AppendOp(db, runID, fileID1, "modify")
	if err != nil {
		t.Fatalf("AppendOp failed: %v", err)
	}

	tests := []struct {
		name    string
		patches []Patch
		wantErr bool
	}{
		{
			name: "single patch",
			patches: []Patch{
				{
					OpID:         opID,
					FileID:       fileID1,
					Algo:         "plain",
					ForwardBlob:  []byte("new content"),
					ReverseBlob:  []byte("old content"),
					BytesAdded:   11,
					BytesRemoved: 11,
				},
			},
			wantErr: false,
		},
		{
			name: "multiple patches",
			patches: []Patch{
				{
					OpID:         opID,
					FileID:       fileID2,
					Algo:         "plain",
					ForwardBlob:  []byte("content1"),
					ReverseBlob:  []byte("old1"),
					BytesAdded:   8,
					BytesRemoved: 4,
				},
				{
					OpID:         opID,
					FileID:       fileID3,
					Algo:         "plain",
					ForwardBlob:  []byte("content2"),
					ReverseBlob:  []byte("old2"),
					BytesAdded:   8,
					BytesRemoved: 4,
				},
			},
			wantErr: false,
		},
		{
			name:    "empty patches",
			patches: []Patch{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RecordPatches(db, tt.patches)
			if (err != nil) != tt.wantErr {
				t.Errorf("RecordPatches() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(tt.patches) > 0 {
				// Verify patches were inserted into database
				var count int
				err = db.QueryRow("SELECT COUNT(*) FROM patches WHERE op_id = ?", opID).Scan(&count)
				if err != nil {
					t.Errorf("Failed to query patches table: %v", err)
				}
				if count < len(tt.patches) {
					t.Errorf("Expected at least %d patches in database, got %d", len(tt.patches), count)
				}
			}
		})
	}
}

func TestCheckpoint(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a run first
	meta := map[string]any{
		"repo":        "test-repo",
		"branch":      "main",
		"commit_base": "abc123",
	}
	runID, err := BeginRun(db, meta)
	if err != nil {
		t.Fatalf("BeginRun failed: %v", err)
	}

	tests := []struct {
		name    string
		runID   string
		cpName  string
		meta    map[string]any
		wantErr bool
	}{
		{
			name:   "valid checkpoint",
			runID:  runID,
			cpName: "CP-A",
			meta: map[string]any{
				"files_processed": 5,
				"status":          "success",
			},
			wantErr: false,
		},
		{
			name:    "checkpoint with nil meta",
			runID:   runID,
			cpName:  "CP-B",
			meta:    nil,
			wantErr: false,
		},
		{
			name:    "invalid run ID",
			runID:   "invalid-uuid",
			cpName:  "CP-C",
			meta:    map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Checkpoint(db, tt.runID, tt.cpName, tt.meta)
			if (err != nil) != tt.wantErr {
				t.Errorf("Checkpoint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify checkpoint was inserted into database
				var count int
				err = db.QueryRow("SELECT COUNT(*) FROM checkpoints WHERE run_id = ? AND name = ?", tt.runID, tt.cpName).Scan(&count)
				if err != nil {
					t.Errorf("Failed to query checkpoints table: %v", err)
				}
				if count != 1 {
					t.Errorf("Expected 1 checkpoint in database, got %d", count)
				}
			}
		})
	}
}

func TestRecordFile(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a run first
	meta := map[string]any{
		"repo":        "test-repo",
		"branch":      "main",
		"commit_base": "abc123",
	}
	runID, err := BeginRun(db, meta)
	if err != nil {
		t.Fatalf("BeginRun failed: %v", err)
	}

	tests := []struct {
		name       string
		runID      string
		fileID     string
		path       string
		lang       string
		size       int64
		hashBefore string
		hashAfter  string
		status     string
		wantErr    bool
	}{
		{
			name:       "valid file record",
			runID:      runID,
			fileID:     "file1",
			path:       "/path/to/file1.go",
			lang:       "go",
			size:       1024,
			hashBefore: "hash1",
			hashAfter:  "hash2",
			status:     "modified",
			wantErr:    false,
		},
		{
			name:       "file with empty hashes",
			runID:      runID,
			fileID:     "file2",
			path:       "/path/to/file2.py",
			lang:       "python",
			size:       0,
			hashBefore: "",
			hashAfter:  "",
			status:     "created",
			wantErr:    false,
		},
		{
			name:       "invalid run ID",
			runID:      "invalid-uuid",
			fileID:     "file3",
			path:       "/path/to/file3.js",
			lang:       "javascript",
			size:       512,
			hashBefore: "hash3",
			hashAfter:  "hash4",
			status:     "deleted",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RecordFile(db, tt.runID, tt.fileID, tt.path, tt.lang, tt.size, tt.hashBefore, tt.hashAfter, tt.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("RecordFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify file was inserted into database
				var count int
				err = db.QueryRow("SELECT COUNT(*) FROM files WHERE run_id = ? AND id = ?", tt.runID, tt.fileID).Scan(&count)
				if err != nil {
					t.Errorf("Failed to query files table: %v", err)
				}
				if count != 1 {
					t.Errorf("Expected 1 file record in database, got %d", count)
				}
			}
		})
	}
}

func TestAppendDiagnostics(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a run and operation first
	meta := map[string]any{
		"repo":        "test-repo",
		"branch":      "main",
		"commit_base": "abc123",
	}
	runID, err := BeginRun(db, meta)
	if err != nil {
		t.Fatalf("BeginRun failed: %v", err)
	}

	// Create a file record first
	fileID := uuid.NewString()
	err = RecordFile(db, runID, fileID, "/path/to/test.go", "go", 100, "hash1", "hash2", "modified")
	if err != nil {
		t.Fatalf("RecordFile failed: %v", err)
	}

	opID, err := AppendOp(db, runID, fileID, "modify")
	if err != nil {
		t.Fatalf("AppendOp failed: %v", err)
	}

	tests := []struct {
		name        string
		opID        string
		diagnostics []map[string]any
		wantErr     bool
	}{
		{
			name: "valid diagnostics",
			opID: opID,
			diagnostics: []map[string]any{
				{
					"severity": "error",
					"message":  "syntax error",
					"line":     10,
					"col":      5,
				},
				{
					"severity": "warning",
					"message":  "unused variable",
					"line":     20,
					"col":      8,
				},
			},
			wantErr: false,
		},
		{
			name:        "empty diagnostics",
			opID:        opID,
			diagnostics: []map[string]any{},
			wantErr:     false,
		},
		{
			name: "invalid op ID",
			opID: "invalid-uuid",
			diagnostics: []map[string]any{
				{
					"severity": "error",
					"message":  "test error",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AppendDiagnostics(db, tt.opID, tt.diagnostics)
			if (err != nil) != tt.wantErr {
				t.Errorf("AppendDiagnostics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(tt.diagnostics) > 0 {
				// Verify diagnostics were inserted into database
				var count int
				err = db.QueryRow("SELECT COUNT(*) FROM diagnostics WHERE op_id = ?", tt.opID).Scan(&count)
				if err != nil {
					t.Errorf("Failed to query diagnostics table: %v", err)
				}
				if count < len(tt.diagnostics) {
					t.Errorf("Expected at least %d diagnostics in database, got %d", len(tt.diagnostics), count)
				}
			}
		})
	}
}

func TestGetRunSummary(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a run with some data
	meta := map[string]any{
		"repo":        "test-repo",
		"branch":      "main",
		"commit_base": "abc123",
		"metrics":     map[string]any{"files": 10},
	}
	runID, err := BeginRun(db, meta)
	if err != nil {
		t.Fatalf("BeginRun failed: %v", err)
	}

	// Add some operations and files
	fileID := uuid.NewString()
	err = RecordFile(db, runID, fileID, "/path/to/test.go", "go", 1024, "hash1", "hash2", "modified")
	if err != nil {
		t.Fatalf("RecordFile failed: %v", err)
	}

	_, err = AppendOp(db, runID, fileID, "modify")
	if err != nil {
		t.Fatalf("AppendOp failed: %v", err)
	}

	tests := []struct {
		name    string
		runID   string
		wantErr bool
	}{
		{
			name:    "valid run ID",
			runID:   runID,
			wantErr: false,
		},
		{
			name:    "invalid run ID",
			runID:   "invalid-uuid",
			wantErr: true,
		},
		{
			name:    "non-existent run ID",
			runID:   uuid.NewString(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary, err := GetRunSummary(db, tt.runID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRunSummary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify summary contains expected fields
				if summary["id"] != tt.runID {
					t.Errorf("Expected run ID %s, got %v", tt.runID, summary["id"])
				}
				if summary["repo"] != "test-repo" {
					t.Errorf("Expected repo 'test-repo', got %v", summary["repo"])
				}
				if summary["branch"] != "main" {
					t.Errorf("Expected branch 'main', got %v", summary["branch"])
				}
			}
		})
	}
}

func TestEnforceRetentionPolicy(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Set up global context with retention policy of 1 run
	originalCtx := GetGlobalContext()
	defer SetGlobalContext(originalCtx)

	testCtx := &Context{
		DBPath:        ".morfx.db",
		RetentionRuns: 1, // Keep only 1 run
	}
	SetGlobalContext(testCtx)

	// Create some old runs to test retention policy
	oldTime := time.Now().AddDate(0, 0, -40) // 40 days ago
	oldRunID := uuid.NewString()

	// Insert old run directly to bypass current timestamp
	_, err := db.Exec(`
		INSERT INTO runs (id, public_ulid, repo, branch, commit_base, status, started_at, metrics_json) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		oldRunID, "old-ulid", "old-repo", "main", "old-commit", "completed",
		oldTime.UnixMilli(), "{}")
	if err != nil {
		t.Fatalf("Failed to insert old run: %v", err)
	}

	// Create a recent run
	meta := map[string]any{
		"repo":        "test-repo",
		"branch":      "main",
		"commit_base": "abc123",
	}
	recentRunID, err := BeginRun(db, meta)
	if err != nil {
		t.Fatalf("BeginRun failed: %v", err)
	}

	// Count active runs before retention policy
	var activeBefore int
	err = db.QueryRow("SELECT COUNT(*) FROM runs WHERE status != 'archived'").Scan(&activeBefore)
	if err != nil {
		t.Fatalf("Failed to count active runs: %v", err)
	}

	// Test retention policy
	err = EnforceRetentionPolicy(db)
	if err != nil {
		t.Errorf("EnforceRetentionPolicy() error = %v", err)
	}

	// Count active runs after retention policy
	var activeAfter int
	err = db.QueryRow("SELECT COUNT(*) FROM runs WHERE status != 'archived'").Scan(&activeAfter)
	if err != nil {
		t.Fatalf("Failed to count active runs after retention: %v", err)
	}

	// Should have fewer active runs now (old ones archived)
	if activeAfter >= activeBefore {
		t.Errorf("Expected fewer active runs after retention policy, before: %d, after: %d", activeBefore, activeAfter)
	}

	// Check that old run is archived
	var oldRunStatus string
	err = db.QueryRow("SELECT status FROM runs WHERE id = ?", oldRunID).Scan(&oldRunStatus)
	if err != nil {
		t.Fatalf("Failed to check old run status: %v", err)
	}
	if oldRunStatus != "archived" {
		t.Errorf("Expected old run to be archived, got status: %s", oldRunStatus)
	}

	// Recent run should still be active
	var recentRunStatus string
	err = db.QueryRow("SELECT status FROM runs WHERE id = ?", recentRunID).Scan(&recentRunStatus)
	if err != nil {
		t.Fatalf("Failed to check recent run status: %v", err)
	}
	if recentRunStatus == "archived" {
		t.Errorf("Recent run should not be archived, got status: %s", recentRunStatus)
	}
}
