package db

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestDeriveKey(t *testing.T) {
	masterKey := make([]byte, 32)
	_, err := rand.Read(masterKey)
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	salt := make([]byte, 16)
	_, err = rand.Read(salt)
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}

	info := []byte("test-info")

	tests := []struct {
		name      string
		masterKey []byte
		salt      []byte
		info      []byte
		keyLen    int
		wantErr   bool
	}{
		{
			name:      "valid AES key derivation",
			masterKey: masterKey,
			salt:      salt,
			info:      info,
			keyLen:    32,
			wantErr:   false,
		},
		{
			name:      "valid ChaCha key derivation",
			masterKey: masterKey,
			salt:      salt,
			info:      info,
			keyLen:    32,
			wantErr:   false,
		},
		{
			name:      "different key length",
			masterKey: masterKey,
			salt:      salt,
			info:      info,
			keyLen:    16,
			wantErr:   false,
		},
		{
			name:      "empty info",
			masterKey: masterKey,
			salt:      salt,
			info:      []byte{},
			keyLen:    32,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := deriveKey(tt.masterKey, tt.salt, tt.info, tt.keyLen)
			if (err != nil) != tt.wantErr {
				t.Errorf("deriveKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(key) != tt.keyLen {
					t.Errorf("deriveKey() key length = %d, want %d", len(key), tt.keyLen)
				}

				// Derive the same key again and ensure it's identical
				key2, err := deriveKey(tt.masterKey, tt.salt, tt.info, tt.keyLen)
				if err != nil {
					t.Errorf("deriveKey() second call failed: %v", err)
				}
				if !bytes.Equal(key, key2) {
					t.Errorf("deriveKey() not deterministic")
				}

				// Derive with different salt and ensure it's different
				salt2 := make([]byte, len(tt.salt))
				copy(salt2, tt.salt)
				salt2[0] ^= 0xFF // flip bits
				key3, err := deriveKey(tt.masterKey, salt2, tt.info, tt.keyLen)
				if err != nil {
					t.Errorf("deriveKey() with different salt failed: %v", err)
				}
				if bytes.Equal(key, key3) {
					t.Errorf("deriveKey() should produce different keys with different salts")
				}
			}
		})
	}
}

func TestGetEncryptor(t *testing.T) {
	tests := []struct {
		name     string
		algo     string
		wantErr  bool
		wantType string
	}{
		{
			name:     "XChaCha20-Poly1305",
			algo:     "xchacha20poly1305",
			wantErr:  false,
			wantType: "*db.xchacha20Encryptor",
		},
		{
			name:     "AES-256-GCM",
			algo:     "aesgcm",
			wantErr:  false,
			wantType: "*db.aesGCMEncryptor",
		},
		{
			name:    "invalid algorithm",
			algo:    "INVALID-ALGO",
			wantErr: true,
		},
		{
			name:    "empty algorithm",
			algo:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encryptor, err := getEncryptor(tt.algo)
			if (err != nil) != tt.wantErr {
				t.Errorf("getEncryptor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if encryptor == nil {
					t.Errorf("getEncryptor() returned nil encryptor")
				}
				// The Algo() method returns uppercase format, not the input format
				expectedAlgo := ""
				if tt.algo == "xchacha20poly1305" {
					expectedAlgo = "XCHACHA20-POLY1305"
				} else if tt.algo == "aesgcm" {
					expectedAlgo = "AES-256-GCM"
				}
				if encryptor.Algo() != expectedAlgo {
					t.Errorf("getEncryptor() algo = %s, want %s", encryptor.Algo(), expectedAlgo)
				}
			}
		})
	}
}

func TestXChaCha20Encryptor(t *testing.T) {
	encryptor := &xchacha20Encryptor{}

	// Test basic properties
	if encryptor.Algo() != "XCHACHA20-POLY1305" {
		t.Errorf("xchacha20Encryptor.Algo() = %s, want XCHACHA20-POLY1305", encryptor.Algo())
	}
	if encryptor.AlgoKeyLen() != 32 {
		t.Errorf("xchacha20Encryptor.AlgoKeyLen() = %d, want 32", encryptor.AlgoKeyLen())
	}
	if encryptor.NonceSize() != 24 {
		t.Errorf("xchacha20Encryptor.NonceSize() = %d, want 24", encryptor.NonceSize())
	}

	// Test encryption/decryption
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	nonce := make([]byte, encryptor.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		t.Fatalf("Failed to generate nonce: %v", err)
	}

	plaintext := []byte("Hello, World!")
	aad := []byte("additional data")

	// Test encryption
	ciphertext, err := encryptor.Encrypt(key, nonce, plaintext, aad)
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	if len(ciphertext) <= len(plaintext) {
		t.Errorf("Ciphertext should be longer than plaintext due to authentication tag")
	}

	// Test decryption
	decrypted, err := encryptor.Decrypt(key, nonce, ciphertext, aad)
	if err != nil {
		t.Fatalf("Decrypt() failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("Decrypted text doesn't match original: got %s, want %s", decrypted, plaintext)
	}

	// Test with wrong key
	wrongKey := make([]byte, 32)
	_, err = rand.Read(wrongKey)
	if err != nil {
		t.Fatalf("Failed to generate wrong key: %v", err)
	}

	_, err = encryptor.Decrypt(wrongKey, nonce, ciphertext, aad)
	if err == nil {
		t.Errorf("Decrypt() should fail with wrong key")
	}

	// Test with wrong AAD
	wrongAAD := []byte("wrong additional data")
	_, err = encryptor.Decrypt(key, nonce, ciphertext, wrongAAD)
	if err == nil {
		t.Errorf("Decrypt() should fail with wrong AAD")
	}

	// Test with invalid key length
	shortKey := make([]byte, 16)
	_, err = encryptor.Encrypt(shortKey, nonce, plaintext, aad)
	if err == nil {
		t.Errorf("Encrypt() should fail with invalid key length")
	}
}

func TestAESGCMEncryptor(t *testing.T) {
	encryptor := &aesGCMEncryptor{}

	// Test basic properties
	if encryptor.Algo() != "AES-256-GCM" {
		t.Errorf("aesGCMEncryptor.Algo() = %s, want AES-256-GCM", encryptor.Algo())
	}
	if encryptor.AlgoKeyLen() != 32 {
		t.Errorf("aesGCMEncryptor.AlgoKeyLen() = %d, want 32", encryptor.AlgoKeyLen())
	}
	if encryptor.NonceSize() != 12 {
		t.Errorf("aesGCMEncryptor.NonceSize() = %d, want 12", encryptor.NonceSize())
	}

	// Test encryption/decryption
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	nonce := make([]byte, encryptor.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		t.Fatalf("Failed to generate nonce: %v", err)
	}

	plaintext := []byte("Hello, AES-GCM!")
	aad := []byte("additional data")

	// Test encryption
	ciphertext, err := encryptor.Encrypt(key, nonce, plaintext, aad)
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	if len(ciphertext) <= len(plaintext) {
		t.Errorf("Ciphertext should be longer than plaintext due to authentication tag")
	}

	// Test decryption
	decrypted, err := encryptor.Decrypt(key, nonce, ciphertext, aad)
	if err != nil {
		t.Fatalf("Decrypt() failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("Decrypted text doesn't match original: got %s, want %s", decrypted, plaintext)
	}

	// Test with wrong key
	wrongKey := make([]byte, 32)
	_, err = rand.Read(wrongKey)
	if err != nil {
		t.Fatalf("Failed to generate wrong key: %v", err)
	}

	_, err = encryptor.Decrypt(wrongKey, nonce, ciphertext, aad)
	if err == nil {
		t.Errorf("Decrypt() should fail with wrong key")
	}

	// Test with wrong AAD
	wrongAAD := []byte("wrong additional data")
	_, err = encryptor.Decrypt(key, nonce, ciphertext, wrongAAD)
	if err == nil {
		t.Errorf("Decrypt() should fail with wrong AAD")
	}

	// Test with invalid key length
	shortKey := make([]byte, 16)
	_, err = encryptor.Encrypt(shortKey, nonce, plaintext, aad)
	if err == nil {
		t.Errorf("Encrypt() should fail with invalid key length")
	}
}

func TestEncryptDecryptBlob(t *testing.T) {
	plaintext := []byte("This is a test message for blob encryption")
	aad := []byte("blob-aad")

	tests := []struct {
		name string
		algo string
	}{
		{
			name: "XChaCha20-Poly1305",
			algo: "xchacha20poly1305",
		},
		{
			name: "AES-256-GCM",
			algo: "aesgcm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encryptor, err := getEncryptor(tt.algo)
			if err != nil {
				t.Fatalf("Failed to get encryptor: %v", err)
			}

			key := make([]byte, encryptor.AlgoKeyLen())
			if _, err := rand.Read(key); err != nil {
				t.Fatalf("Failed to generate key: %v", err)
			}

			// Test encryptBlob
			encrypted, err := encryptBlob(key, plaintext, aad, encryptor)
			if err != nil {
				t.Fatalf("encryptBlob() failed: %v", err)
			}

			if len(encrypted) <= len(plaintext) {
				t.Errorf("Encrypted blob should be longer than plaintext")
			}

			// Test decryptBlob
			decrypted, err := decryptBlob(key, encrypted, aad, encryptor)
			if err != nil {
				t.Fatalf("decryptBlob() failed: %v", err)
			}

			if !bytes.Equal(plaintext, decrypted) {
				t.Errorf("Decrypted blob doesn't match original: got %s, want %s", decrypted, plaintext)
			}
		})
	}

	// Test with different encryptors to ensure they produce different results
	xchachaEncryptor, _ := getEncryptor("xchacha20poly1305")
	aesEncryptor, _ := getEncryptor("aesgcm")

	key1 := make([]byte, xchachaEncryptor.AlgoKeyLen())
	key2 := make([]byte, aesEncryptor.AlgoKeyLen())
	rand.Read(key1)
	rand.Read(key2)

	encrypted, _ := encryptBlob(key1, plaintext, aad, xchachaEncryptor)
	encrypted2, _ := encryptBlob(key2, plaintext, aad, aesEncryptor)

	// Ensure different encryptors produce different results
	if bytes.Equal(encrypted, encrypted2) {
		t.Errorf("Different encryptors should produce different encrypted blobs")
	}
}

func setupEncryptTestDB(t *testing.T) (*sql.DB, func()) {
	// Use a temporary file database instead of in-memory to avoid transaction issues
	tmpFile := t.TempDir() + "/test.db"
	db, err := sql.Open("sqlite3", tmpFile+"?_journal_mode=WAL&_timeout=5000")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Set connection pool settings to avoid locking issues
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Run migrations to create necessary tables
	err = Migrate(db)
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db, func() { db.Close() }
}

func TestInitKeyring(t *testing.T) {
	tests := []struct {
		name     string
		algo     string
		password string
		wantErr  bool
	}{
		{
			name:     "XChaCha20-Poly1305",
			algo:     "xchacha20poly1305",
			password: "test-password",
			wantErr:  false,
		},
		{
			name:     "AES-256-GCM",
			algo:     "aesgcm",
			password: "test-password",
			wantErr:  false,
		},
		{
			name:     "invalid algorithm",
			algo:     "invalid",
			password: "test-password",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, cleanup := setupEncryptTestDB(t)
			defer cleanup()

			masterKey := make([]byte, 32)
			_, err := rand.Read(masterKey)
			if err != nil {
				t.Fatalf("Failed to generate master key: %v", err)
			}
			activeKeyVersion := 1

			encryptor, err := getEncryptor(tt.algo)
			if tt.wantErr {
				if err == nil {
					t.Errorf("getEncryptor() expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("getEncryptor() failed: %v", err)
			}

			err = initKeyring(db, masterKey, activeKeyVersion, encryptor)
			if (err != nil) != tt.wantErr {
				t.Errorf("initKeyring() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify key was stored in database
				var count int
				err = db.QueryRow("SELECT COUNT(*) FROM keys WHERE key_version = ?", activeKeyVersion).Scan(&count)
				if err != nil {
					t.Errorf("Failed to query keys table: %v", err)
				}
				if count != 1 {
					t.Errorf("Expected 1 encryption key in database, got %d", count)
				}

				// Verify key is in global keyring
				if _, exists := globalKeyring[activeKeyVersion]; !exists {
					t.Errorf("Key version %d not found in global keyring", activeKeyVersion)
				}
			}
		})
	}
}

func TestKeyRotation(t *testing.T) {
	db, cleanup := setupEncryptTestDB(t)
	defer cleanup()

	// Setup master key and encryptor
	masterKey := make([]byte, 32)
	_, err := rand.Read(masterKey)
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	encryptor := &xchacha20Encryptor{}
	config := DefaultKeyRotationConfig()

	// Initialize keyring with version 1
	err = initKeyring(db, masterKey, 1, encryptor)
	if err != nil {
		t.Fatalf("Failed to initialize keyring: %v", err)
	}

	// Test key rotation
	newVersion, err := RotateKey(db, masterKey, encryptor, config)
	if err != nil {
		t.Fatalf("Failed to rotate key: %v", err)
	}

	if newVersion != 2 {
		t.Errorf("Expected new version 2, got %d", newVersion)
	}

	// Verify active key version
	activeVersion, err := GetActiveKeyVersion(db)
	if err != nil {
		t.Fatalf("Failed to get active key version: %v", err)
	}

	if activeVersion != 2 {
		t.Errorf("Expected active version 2, got %d", activeVersion)
	}

	// Verify old key is still accessible
	_, err = GetKeyForDecryption(1)
	if err != nil {
		t.Errorf("Old key should still be accessible: %v", err)
	}

	// Verify new key is accessible
	_, err = GetKeyForDecryption(2)
	if err != nil {
		t.Errorf("New key should be accessible: %v", err)
	}
}

func TestVersionedEncryption(t *testing.T) {
	db, cleanup := setupEncryptTestDB(t)
	defer cleanup()

	// Setup master key and encryptor
	masterKey := make([]byte, 32)
	_, err := rand.Read(masterKey)
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	encryptor := &xchacha20Encryptor{}

	// Initialize keyring
	err = initKeyring(db, masterKey, 1, encryptor)
	if err != nil {
		t.Fatalf("Failed to initialize keyring: %v", err)
	}

	plaintext := []byte("test data for versioned encryption")
	aad := []byte("additional authenticated data")

	// Encrypt with version information
	encrypted, err := EncryptBlobWithVersion(db, plaintext, aad, encryptor)
	if err != nil {
		t.Fatalf("Failed to encrypt with version: %v", err)
	}

	// Decrypt with version detection
	decrypted, err := DecryptBlobWithVersion(encrypted, aad, encryptor)
	if err != nil {
		t.Fatalf("Failed to decrypt with version: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("Decrypted data doesn't match original")
	}

	// Rotate key and test cross-version compatibility
	_, err = RotateKey(db, masterKey, encryptor, DefaultKeyRotationConfig())
	if err != nil {
		t.Fatalf("Failed to rotate key: %v", err)
	}

	// Old encrypted data should still be decryptable
	decrypted2, err := DecryptBlobWithVersion(encrypted, aad, encryptor)
	if err != nil {
		t.Fatalf("Failed to decrypt old data after key rotation: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted2) {
		t.Errorf("Old encrypted data not decryptable after key rotation")
	}

	// New encryption should use new key version
	newEncrypted, err := EncryptBlobWithVersion(db, plaintext, aad, encryptor)
	if err != nil {
		t.Fatalf("Failed to encrypt with new version: %v", err)
	}

	// Extract version from new encrypted data
	newVersion := int(newEncrypted[0])<<24 | int(newEncrypted[1])<<16 | int(newEncrypted[2])<<8 | int(newEncrypted[3])
	if newVersion != 2 {
		t.Errorf("Expected new encryption to use version 2, got %d", newVersion)
	}
}

func TestKeyRotationNeeded(t *testing.T) {
	db, cleanup := setupEncryptTestDB(t)
	defer cleanup()

	masterKey := make([]byte, 32)
	_, err := rand.Read(masterKey)
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	encryptor := &xchacha20Encryptor{}
	config := &KeyRotationConfig{
		RotationInterval: 1 * time.Hour,
		MaxKeyAge:        24 * time.Hour,
		RetainOldKeys:    3,
	}

	// No keys exist, rotation should be needed
	needed, err := CheckKeyRotationNeeded(db, config)
	if err != nil {
		t.Fatalf("Failed to check key rotation: %v", err)
	}
	if !needed {
		t.Error("Key rotation should be needed when no keys exist")
	}

	// Initialize keyring
	err = initKeyring(db, masterKey, 1, encryptor)
	if err != nil {
		t.Fatalf("Failed to initialize keyring: %v", err)
	}

	// Fresh key, rotation should not be needed
	needed, err = CheckKeyRotationNeeded(db, config)
	if err != nil {
		t.Fatalf("Failed to check key rotation: %v", err)
	}
	if needed {
		t.Error("Key rotation should not be needed for fresh key")
	}
}

func TestKeyCleanup(t *testing.T) {
	db, cleanup := setupEncryptTestDB(t)
	defer cleanup()

	// Verify keys table exists
	var keyCount int
	err := db.QueryRow("SELECT COUNT(*) FROM keys").Scan(&keyCount)
	if err != nil {
		t.Fatalf("Failed to query keys table: %v", err)
	}

	masterKey := make([]byte, 32)
	_, err = rand.Read(masterKey)
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	encryptor := &xchacha20Encryptor{}
	config := &KeyRotationConfig{
		RotationInterval: 1 * time.Hour,
		MaxKeyAge:        24 * time.Hour,
		RetainOldKeys:    2, // Keep only 2 keys
	}

	// Initialize and create multiple key versions
	err = initKeyring(db, masterKey, 1, encryptor)
	if err != nil {
		t.Fatalf("Failed to initialize keyring: %v", err)
	}

	// Create 4 more versions (total 5)
	for i := range 4 {
		_, err = RotateKey(db, masterKey, encryptor, config)
		if err != nil {
			t.Fatalf("Failed to rotate key iteration %d: %v", i, err)
		}
	}

	// Check that only 2 keys remain
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM keys").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query keys count: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 keys after cleanup, got %d", count)
	}

	// Verify the remaining keys are the most recent ones
	activeVersion, err := GetActiveKeyVersion(db)
	if err != nil {
		t.Fatalf("Failed to get active version: %v", err)
	}

	if activeVersion != 5 {
		t.Errorf("Expected active version 5, got %d", activeVersion)
	}
}

func TestGetEncryptionConfig(t *testing.T) {
	tests := []struct {
		name           string
		algo           string
		password       string
		encryptionMode string
		wantErr        bool
	}{
		{
			name:           "XChaCha20-Poly1305",
			algo:           "xchacha20poly1305",
			password:       "test-password",
			encryptionMode: "on",
			wantErr:        false,
		},
		{
			name:           "AES-256-GCM",
			algo:           "aesgcm",
			password:       "test-password",
			encryptionMode: "on",
			wantErr:        false,
		},
		{
			name:           "encryption off",
			algo:           "",
			password:       "",
			encryptionMode: "off",
			wantErr:        false,
		},
		{
			name:           "no master key with encryption on",
			algo:           "xchacha20poly1305",
			password:       "",
			encryptionMode: "on",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up global context for encryption config
			ctx := &Context{
				EncryptionMode:   tt.encryptionMode,
				MasterKey:        hex.EncodeToString([]byte(tt.password)),
				EncryptionAlgo:   tt.algo,
				ActiveKeyVersion: 1,
			}
			if tt.password == "" {
				ctx.MasterKey = ""
			}
			SetGlobalContext(ctx)

			mode, masterKey, derivedKey, encryptor, err := getEncryptionConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("getEncryptionConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if mode != tt.encryptionMode {
					t.Errorf("Expected mode '%s', got %s", tt.encryptionMode, mode)
				}
				if tt.encryptionMode == "on" {
					if masterKey == nil {
						t.Error("Expected non-nil master key")
					}
					if derivedKey == nil {
						t.Error("Expected non-nil derived key")
					}
					if encryptor == nil {
						t.Error("Expected non-nil encryptor")
					}
				}
			}
		})
	}
}
