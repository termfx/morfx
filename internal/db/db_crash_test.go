package db

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestRollbackCrashResumeCPA(t *testing.T) {
	// This test uses the GO_WANT_HELPER_PROCESS pattern to simulate a crash.
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	// Setup: Create a dummy file and a database connection
	testFilePath := "test_crash_cpa.txt"
	originalContent := []byte("original content")
	modifiedContent := []byte("modified content")

	err := os.WriteFile(testFilePath, originalContent, 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFilePath)

	dbConn, err := Open()
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer dbConn.Close()

	err = Migrate(dbConn.DB)
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	// Simulate a run and an operation
	runID, err := BeginRun(dbConn.DB, map[string]any{"repo": "test", "branch": "main", "commit_base": "abc"})
	if err != nil {
		t.Fatalf("BeginRun failed: %v", err)
	}

	fileID := uuid.NewString()
	err = RecordFile(dbConn.DB, runID, fileID, testFilePath, "text", int64(len(originalContent)), "hash_orig", "hash_mod", "modified")
	if err != nil {
		t.Fatalf("RecordFile failed: %v", err)
	}

	opID, err := AppendOp(dbConn.DB, runID, fileID, "rewrite")
	if err != nil {
		t.Fatalf("AppendOp failed: %v", err)
	}

	// Record a patch
	patch := Patch{
		OpID:         opID,
		FileID:       fileID,
		Algo:         "binary",
		ForwardBlob:  modifiedContent,
		ReverseBlob:  originalContent,
		BytesAdded:   len(modifiedContent),
		BytesRemoved: len(originalContent),
	}
	err = RecordPatches(dbConn.DB, []Patch{patch})
	if err != nil {
		t.Fatalf("RecordPatches failed: %v", err)
	}

	// Call Rollback with CP-A crash point
	_, err = Rollback(dbConn.DB, opID, false, true)
	if err != nil && !strings.Contains(err.Error(), "exit status 137") {
		t.Fatalf("Rollback did not crash as expected: %v", err)
	}
}

func TestRollbackCrashResumeCPB(t *testing.T) {
	// This test uses the GO_WANT_HELPER_PROCESS pattern to simulate a crash.
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	// Setup: Create a dummy file and a database connection
	testFilePath := "test_crash_cpb.txt"
	originalContent := []byte("original content")
	modifiedContent := []byte("modified content")

	err := os.WriteFile(testFilePath, originalContent, 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFilePath)

	dbConn, err := Open()
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer dbConn.Close()

	err = Migrate(dbConn.DB)
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	// Simulate a run and an operation
	runID, err := BeginRun(dbConn.DB, map[string]any{"repo": "test", "branch": "main", "commit_base": "abc"})
	if err != nil {
		t.Fatalf("BeginRun failed: %v", err)
	}

	fileID := uuid.NewString()
	err = RecordFile(dbConn.DB, runID, fileID, testFilePath, "text", int64(len(originalContent)), "hash_orig", "hash_mod", "modified")
	if err != nil {
		t.Fatalf("RecordFile failed: %v", err)
	}

	opID, err := AppendOp(dbConn.DB, runID, fileID, "rewrite")
	if err != nil {
		t.Fatalf("AppendOp failed: %v", err)
	}

	// Record a patch
	patch := Patch{
		OpID:         opID,
		FileID:       fileID,
		Algo:         "binary",
		ForwardBlob:  modifiedContent,
		ReverseBlob:  originalContent,
		BytesAdded:   len(modifiedContent),
		BytesRemoved: len(originalContent),
	}
	err = RecordPatches(dbConn.DB, []Patch{patch})
	if err != nil {
		t.Fatalf("RecordPatches failed: %v", err)
	}

	// Call Rollback with CP-B crash point
	_, err = Rollback(dbConn.DB, opID, false, true)
	if err != nil && !strings.Contains(err.Error(), "exit status 137") {
		t.Fatalf("Rollback did not crash as expected: %v", err)
	}
}

func TestRollbackCrashResume(t *testing.T) {
	// Ensure .morfx directory exists and clean it up before and after
	_ = os.RemoveAll(".morfx")
	if err := os.MkdirAll(".morfx", 0o700); err != nil {
		t.Fatalf("Failed to create .morfx directory: %v", err)
	}
	defer os.RemoveAll(".morfx")

	// Setup: Create a dummy file
	testFilePath := "test_crash_resume.txt"
	originalContent := []byte("original content")

	err := os.WriteFile(testFilePath, originalContent, 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFilePath)

	// Scenario 1: Crash at CP-A (before renames)
	t.Run("Crash at CP-A", func(t *testing.T) {
		_ = os.RemoveAll(".morfx") // Clean for subtest
		cmd := exec.Command(os.Args[0], "-test.run=TestRollbackCrashResumeCPA")
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1", "MORFX_CRASH_POINT=CP-A")
		cmd.Dir = "."
		out, err := cmd.CombinedOutput()
		if e, ok := err.(*exec.ExitError); ok && e.ExitCode() == 137 {
			// Expected crash
		} else if err != nil {
			t.Fatalf("Helper process failed unexpectedly: %v\nOutput: %s", err, out)
		} else {
			t.Fatalf("Helper process did not crash as expected\nOutput: %s", out)
		}

		// Verify FS state: original file intact
		currentContent, err := os.ReadFile(testFilePath)
		if err != nil {
			t.Fatalf("Failed to read file after crash: %v", err)
		}
		if string(currentContent) != string(originalContent) {
			t.Errorf("File content mismatch after CP-A crash. Expected: %s, Got: %s", originalContent, currentContent)
		}

		// Resume: Re-run rollback, it should complete successfully
		dbConn, err := Open()
		if err != nil {
			t.Fatalf("Open failed on resume: %v", err)
		}
		defer dbConn.Close()

		// Need to get the opID from the DB, as the helper process crashed before returning it
		var opID string
		err = dbConn.QueryRow(`SELECT id FROM operations ORDER BY started_at DESC LIMIT 1`).Scan(&opID)
		if err != nil {
			t.Fatalf("Failed to get opID for resume: %v", err)
		}

		result, err := Rollback(dbConn.DB, opID, false, true)
		if err != nil {
			t.Fatalf("Rollback on resume failed: %v", err)
		}
		if len(result.RevertedOps) != 1 {
			t.Errorf("Rollback on resume did not revert expected operation. RevertedOps: %v", result.RevertedOps)
		}

		// Verify FS state after resume
		currentContent, err = os.ReadFile(testFilePath)
		if err != nil {
			t.Fatalf("Failed to read file after resume: %v", err)
		}
		if string(currentContent) != string(originalContent) {
			t.Errorf("File content mismatch after resume. Expected: %s, Got: %s", originalContent, currentContent)
		}

		// Verify DB state after resume
		var status string
		err = dbConn.QueryRow("SELECT status FROM operations WHERE id = ?", opID).Scan(&status)
		if err != nil {
			t.Fatalf("Failed to query operation status after resume: %v", err)
		}
		if status != "rolled_back" {
			t.Errorf("Expected operation status to be 'rolled_back' after resume, got: %s", status)
		}
	})

	// Scenario 2: Crash at CP-B (after renames, before COMMIT)
	t.Run("Crash at CP-B", func(t *testing.T) {
		_ = os.RemoveAll(".morfx") // Clean for subtest
		cmd := exec.Command(os.Args[0], "-test.run=TestRollbackCrashResumeCPB")
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1", "MORFX_CRASH_POINT=CP-B")
		cmd.Dir = "."
		out, err := cmd.CombinedOutput()
		if e, ok := err.(*exec.ExitError); ok && e.ExitCode() == 137 {
			// Expected crash
		} else if err != nil {
			t.Fatalf("Helper process failed unexpectedly: %v\nOutput: %s", err, out)
		} else {
			t.Fatalf("Helper process did not crash as expected\nOutput: %s", out)
		}

		// Verify FS state: file should have modified content (as rename happened)
		currentContent, err := os.ReadFile(testFilePath)
		if err != nil {
			t.Fatalf("Failed to read file after crash: %v", err)
		}
		if string(currentContent) != string(originalContent) { // Should be original content after rollback
			t.Errorf("File content mismatch after CP-B crash. Expected: %s, Got: %s", originalContent, currentContent)
		}

		// Resume: Re-run rollback, it should complete successfully
		dbConn, err := Open()
		if err != nil {
			t.Fatalf("Open failed on resume: %v", err)
		}
		defer dbConn.Close()

		// Need to get the opID from the DB, as the helper process crashed before returning it
		var opID string
		err = dbConn.QueryRow(`SELECT id FROM operations ORDER BY started_at DESC LIMIT 1`).Scan(&opID)
		if err != nil {
			t.Fatalf("Failed to get opID for resume: %v", err)
		}

		result, err := Rollback(dbConn.DB, opID, false, true)
		if err != nil {
			t.Fatalf("Rollback on resume failed: %v", err)
		}
		if len(result.RevertedOps) != 1 {
			t.Errorf("Rollback on resume did not revert expected operation. RevertedOps: %v", result.RevertedOps)
		}

		// Verify FS state after resume
		currentContent, err = os.ReadFile(testFilePath)
		if err != nil {
			t.Fatalf("Failed to read file after resume: %v", err)
		}
		if string(currentContent) != string(originalContent) {
			t.Errorf("File content mismatch after resume. Expected: %s, Got: %s", originalContent, currentContent)
		}

		// Verify DB state after resume
		var status string
		err = dbConn.QueryRow("SELECT status FROM operations WHERE id = ?", opID).Scan(&status)
		if err != nil {
			t.Fatalf("Failed to query operation status after resume: %v", err)
		}
		if status != "rolled_back" {
			t.Errorf("Expected operation status to be 'rolled_back' after resume, got: %s", status)
		}
	})
}
