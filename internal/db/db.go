package db

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// execWithRetry wraps Exec with retry logic for database is locked errors.
func execWithRetry(db *sql.DB, query string, args ...any) (sql.Result, error) {
	var res sql.Result
	var err error
	maxRetries := 5
	for range maxRetries {
		res, err = db.Exec(query, args...)
		if err == nil {
			return res, nil
		}
		if strings.Contains(err.Error(), "database is locked") {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		return nil, err
	}
	return nil, fmt.Errorf(
		"execWithRetry: database is locked after %d retries: %w",
		maxRetries,
		err,
	)
}

// execWithRetryTx wraps Tx.Exec with retry logic for database is locked errors.
func execWithRetryTx(tx *sql.Tx, query string, args ...any) (sql.Result, error) {
	var res sql.Result
	var err error
	maxRetries := 5
	for range maxRetries {
		res, err = tx.Exec(query, args...)
		if err == nil {
			return res, nil
		}
		if strings.Contains(err.Error(), "database is locked") {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		return nil, err
	}
	return nil, fmt.Errorf(
		"execWithRetryTx: database is locked after %d retries: %w",
		maxRetries,
		err,
	)
}

// queryRowWithRetry wraps QueryRow with retry logic for database is locked errors.
func queryRowWithRetry(db *sql.DB, query string, args ...any) *sql.Row {
	// QueryRow does not return error until Scan, so we can't retry here
	return db.QueryRow(query, args...)
}

// QuickCheck runs PRAGMA quick_check and returns error if DB is not healthy.
func QuickCheck(db *sql.DB) error {
	row := db.QueryRow("PRAGMA quick_check;")
	var result string
	if err := row.Scan(&result); err != nil {
		return fmt.Errorf("quick_check scan error: %w", err)
	}
	if result != "ok" {
		return fmt.Errorf("quick_check failed: %s", result)
	}
	return nil
}

// RunHealthCheck performs a database health check.
func RunHealthCheck(db *sql.DB) error {
	return QuickCheck(db)
}

func getDBPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	return fmt.Sprintf("%s/.morfx/run.db", cwd), nil
}

type DBConn struct {
	*sql.DB
}

func (conn *DBConn) Close() error {
	// Run PRAGMA quick_check before closing
	if err := QuickCheck(conn.DB); err != nil {
		// TODO: Replace with proper logging
		fmt.Printf("WARNING: quick_check failed on close: %v\n", err)
	}
	return conn.DB.Close()
}

// Open opens the SQLite DB and applies required PRAGMAs.
func Open() (*DBConn, error) {
	dbPath, err := getDBPath()
	if err != nil {
		return nil, err
	}
	// Ensure .morfx/ directory exists and set permissions
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	dir := fmt.Sprintf("%s/.morfx", cwd)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create .morfx directory: %w", err)
	}
	// Set permissions for .morfx/ (ignore error if already set)
	_ = os.Chmod(dir, 0o700)
	// Create run.db if not exists, then set permissions
	f, err := os.OpenFile(dbPath, os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to create run.db: %w", err)
	}
	f.Close()
	_ = os.Chmod(dbPath, 0o600)
	db, err := sql.Open("sqlite3", fmt.Sprintf("%s?_busy_timeout=5000&_foreign_keys=ON&_journal_mode=WAL&_synchronous=NORMAL&_temp_store=MEMORY", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	// Apply migrations
	if err := Migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	// Add a final PRAGMA quick_check for sanity
	if err := QuickCheck(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("initial quick_check failed: %w", err)
	}

	// Initialize keyring if encryption is enabled
	ctx := GetGlobalContext()
	mode := ctx.EncryptionMode
	masterKeyHex := ctx.MasterKey
	keyVersion := ctx.ActiveKeyVersion

	if mode != "off" && masterKeyHex != "" {
		masterKey, err := hex.DecodeString(masterKeyHex)
		if err != nil {
			return nil, fmt.Errorf("invalid encryption master key hex: %w", err)
		}
		encryptor, err := getEncryptor(ctx.EncryptionAlgo)
		if err != nil {
			return nil, fmt.Errorf("failed to get encryptor: %w", err)
		}
		if err := initKeyring(db, masterKey, keyVersion, encryptor); err != nil {
			return nil, fmt.Errorf("failed to initialize keyring: %w", err)
		}
	}

	return &DBConn{db}, nil
}

// CheckWALSizeAndCheckpoint checks WAL size and runs checkpoint if above threshold (128 MB).
func CheckWALSizeAndCheckpoint(db *sql.DB) error {
	dbPath, err := getDBPath()
	if err != nil {
		return err
	}
	walPath := dbPath + "-wal"
	info, err := os.Stat(walPath)
	if os.IsNotExist(err) {
		return nil // WAL may not exist yet, ignore
	} else if err != nil {
		return fmt.Errorf("failed to get WAL file info: %w", err)
	}

	// Get configurable threshold from config, default to 128 MB
	thresholdMB := 10 // TODO: Add WALAutoCheckpointMB to DBConfig if needed
	thresholdBytes := int64(thresholdMB) * 1024 * 1024

	// TODO: Replace with proper logging
	// fmt.Printf("WAL size before checkpoint: %d bytes (threshold: %d bytes)\n", info.Size(), thresholdBytes)

	if info.Size() > thresholdBytes {
		_, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE);")
		if err != nil {
			return fmt.Errorf("failed to checkpoint WAL: %w", err)
		}
		// TODO: Replace with proper logging
		// fmt.Printf("WAL checkpointed. WAL size after checkpoint: %d bytes\n", info.Size())
	}
	return nil
}
