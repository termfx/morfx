package db

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

const (
	aesKeyLen     = 32 // AES-256-GCM
	xchachaKeyLen = 32 // XChaCha20-Poly1305
	// gcmStandardNonceSize is the standard nonce size for GCM.
	gcmStandardNonceSize = 12
)

// Encryptor defines the interface for encryption and decryption operations.
type Encryptor interface {
	Encrypt(key []byte, nonce []byte, plaintext []byte, aad []byte) ([]byte, error)
	Decrypt(key []byte, nonce []byte, ciphertext []byte, aad []byte) ([]byte, error)
	NonceSize() int
	Algo() string // "XCHACHA20-POLY1305" or "AES-256-GCM"
	AlgoKeyLen() int
}

// xchacha20Encryptor implements the Encryptor interface for XChaCha20-Poly1305.
type xchacha20Encryptor struct{}

// Encrypt encrypts plaintext using XChaCha20-Poly1305.
func (e *xchacha20Encryptor) Encrypt(key []byte, nonce []byte, plaintext []byte, aad []byte) ([]byte, error) {
	if len(key) != xchachaKeyLen {
		return nil, fmt.Errorf("invalid XChaCha20-Poly1305 key length: %d", len(key))
	}
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	return aead.Seal(nil, nonce, plaintext, aad), nil
}

// Decrypt decrypts ciphertext using XChaCha20-Poly1305.
func (e *xchacha20Encryptor) Decrypt(key []byte, nonce []byte, ciphertext []byte, aad []byte) ([]byte, error) {
	if len(key) != xchachaKeyLen {
		return nil, fmt.Errorf("invalid XChaCha20-Poly1305 key length: %d", len(key))
	}
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	return aead.Open(nil, nonce, ciphertext, aad)
}

// NonceSize returns the nonce size for XChaCha20-Poly1305.
func (e *xchacha20Encryptor) NonceSize() int {
	return chacha20poly1305.NonceSizeX
}

// Algo returns the algorithm name for XChaCha20-Poly1305.
func (e *xchacha20Encryptor) Algo() string {
	return "XCHACHA20-POLY1305"
}

// AlgoKeyLen returns the key length for the specific algorithm.
func (e *xchacha20Encryptor) AlgoKeyLen() int {
	return xchachaKeyLen
}

// aesGCMEncryptor implements the Encryptor interface for AES-256-GCM.
type aesGCMEncryptor struct{}

// Encrypt encrypts plaintext using AES-256-GCM.
func (e *aesGCMEncryptor) Encrypt(key []byte, nonce []byte, plaintext []byte, aad []byte) ([]byte, error) {
	if len(key) != aesKeyLen {
		return nil, fmt.Errorf("invalid AES key length: %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Seal(nil, nonce, plaintext, aad), nil
}

// Decrypt decrypts ciphertext using AES-256-GCM.
func (e *aesGCMEncryptor) Decrypt(key []byte, nonce []byte, ciphertext []byte, aad []byte) ([]byte, error) {
	if len(key) != aesKeyLen {
		return nil, fmt.Errorf("invalid AES key length: %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ciphertext, aad)
}

// NonceSize returns the nonce size for AES-256-GCM.
func (e *aesGCMEncryptor) NonceSize() int {
	return gcmStandardNonceSize
}

// Algo returns the algorithm name for AES-256-GCM.
func (e *aesGCMEncryptor) Algo() string {
	return "AES-256-GCM"
}

// AlgoKeyLen returns the key length for the specific algorithm.
func (e *aesGCMEncryptor) AlgoKeyLen() int {
	return aesKeyLen
}

// deriveKey uses HKDF-SHA256 to derive a key from a master secret.
func deriveKey(masterKey, salt, info []byte, keyLen int) ([]byte, error) {
	hkdf := hkdf.New(sha256.New, masterKey, salt, info)
	key := make([]byte, keyLen)
	if _, err := io.ReadFull(hkdf, key); err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}
	return key, nil
}

// getEncryptor returns the appropriate Encryptor based on the MORFX_ENCRYPTION_ALGO environment variable.
func getEncryptor(algo string) (Encryptor, error) {
	switch algo {
	case "xchacha20poly1305":
		return &xchacha20Encryptor{}, nil
	case "aesgcm":
		return &aesGCMEncryptor{}, nil
	default:
		return nil, fmt.Errorf("unsupported encryption algorithm: %s", algo)
	}
}

// globalKeyring stores derived keys by their version.
var (
	globalKeyring      = make(map[int][]byte)
	globalKeyringMutex sync.RWMutex
)

// KeyRotationConfig holds configuration for key rotation
type KeyRotationConfig struct {
	RotationInterval time.Duration // How often to rotate keys
	MaxKeyAge        time.Duration // Maximum age before key is considered expired
	RetainOldKeys    int           // Number of old keys to retain for decryption
}

// DefaultKeyRotationConfig returns sensible defaults for key rotation
func DefaultKeyRotationConfig() *KeyRotationConfig {
	return &KeyRotationConfig{
		RotationInterval: 30 * 24 * time.Hour, // 30 days
		MaxKeyAge:        90 * 24 * time.Hour, // 90 days
		RetainOldKeys:    5,                   // Keep 5 old keys
	}
}

// initKeyring initializes the keyring by deriving the active key and loading existing keys from the database.
func initKeyring(db *sql.DB, masterKey []byte, activeKeyVersion int, encryptor Encryptor) error {
	globalKeyringMutex.Lock()
	defer globalKeyringMutex.Unlock()
	// Derive the active key
	salt := []byte("morfx-patches")
	info := fmt.Appendf(nil, "v%d", activeKeyVersion)
	derivedKey, err := deriveKey(masterKey, salt, info, encryptor.AlgoKeyLen())
	if err != nil {
		return fmt.Errorf("failed to derive active key for version %d: %w", activeKeyVersion, err)
	}
	globalKeyring[activeKeyVersion] = derivedKey

	// Check if the active key version exists in the database
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM keys WHERE key_version = ?", activeKeyVersion).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to query keys table: %w", err)
	}

	if count == 0 {
		// Insert the new key version into the database
		keyHash := sha256.Sum256(derivedKey)
		_, err := db.Exec(
			"INSERT INTO keys(key_version, created_at, algo, key_hash, key_material, derivation_salt, derivation_info, is_active) VALUES(?, ?, ?, ?, ?, ?, ?, ?)",
			activeKeyVersion,
			time.Now().Unix(), // Use Unix timestamp for created_at
			encryptor.Algo(),
			keyHash[:16], // Store first 16 bytes of SHA256(key)
			derivedKey,   // Store the actual key material
			salt,         // Store derivation salt
			info,         // Store derivation info
			1,            // Mark as active
		)
		if err != nil {
			return fmt.Errorf("failed to insert new key version into database: %w", err)
		}
	}

	// Load all existing keys from the database into the keyring
	rows, err := db.Query("SELECT key_version, algo, key_hash FROM keys")
	if err != nil {
		return fmt.Errorf("failed to query existing keys: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var kv int
		var algo string
		var keyHashPrefix []byte
		if err := rows.Scan(&kv, &algo, &keyHashPrefix); err != nil {
			return fmt.Errorf("failed to scan key row: %w", err)
		}

		// Re-derive the key for existing versions and verify hash
		info = fmt.Appendf(nil, "v%d", kv)
		// Assuming all keys use the same AlgoKeyLen for now, this might need to be dynamic if algo changes per key version
		existingDerivedKey, err := deriveKey(masterKey, salt, info, encryptor.AlgoKeyLen())
		if err != nil {
			return fmt.Errorf("failed to re-derive key for version %d: %w", kv, err)
		}

		reDerivedKeyHash := sha256.Sum256(existingDerivedKey)
		if !bytes.Equal(keyHashPrefix, reDerivedKeyHash[:16]) {
			return fmt.Errorf("key hash mismatch for version %d. Possible tampering or master key change.", kv)
		}
		globalKeyring[kv] = existingDerivedKey
	}

	return nil
}

// RotateKey creates a new key version and marks it as active
func RotateKey(db *sql.DB, masterKey []byte, encryptor Encryptor, config *KeyRotationConfig) (int, error) {
	globalKeyringMutex.Lock()
	defer globalKeyringMutex.Unlock()

	// Get the current maximum key version
	var maxVersion int
	err := db.QueryRow("SELECT COALESCE(MAX(key_version), 0) FROM keys").Scan(&maxVersion)
	if err != nil {
		return 0, fmt.Errorf("failed to get max key version: %w", err)
	}

	newVersion := maxVersion + 1
	salt := []byte("morfx-patches")
	info := fmt.Appendf(nil, "v%d", newVersion)

	// Derive new key
	newKey, err := deriveKey(masterKey, salt, info, encryptor.AlgoKeyLen())
	if err != nil {
		return 0, fmt.Errorf("failed to derive new key for version %d: %w", newVersion, err)
	}

	// Store new key in keyring
	globalKeyring[newVersion] = newKey

	// Insert new key into database
	keyHash := sha256.Sum256(newKey)
	now := time.Now().Unix()
	_, err = db.Exec(
		"INSERT INTO keys(key_version, created_at, algo, key_hash, key_material, is_active, derivation_salt, derivation_info) VALUES(?, ?, ?, ?, ?, ?, ?, ?)",
		newVersion,
		now,
		encryptor.Algo(),
		keyHash[:16],
		newKey, // Store the actual derived key material
		1,      // is_active as integer (1 = true)
		salt,
		info,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert new key version: %w", err)
	}

	// Mark previous keys as inactive
	_, err = db.Exec("UPDATE keys SET is_active = 0 WHERE key_version != ?", newVersion)
	if err != nil {
		return 0, fmt.Errorf("failed to deactivate old keys: %w", err)
	}

	// Clean up old keys if we exceed retention limit
	if config != nil && config.RetainOldKeys > 0 {
		err = cleanupOldKeys(db, config.RetainOldKeys)
		if err != nil {
			return newVersion, fmt.Errorf("key rotation succeeded but cleanup failed: %w", err)
		}
	}

	return newVersion, nil
}

// cleanupOldKeys removes old keys beyond the retention limit
func cleanupOldKeys(db *sql.DB, retainCount int) error {
	// Get keys to delete (keeping the most recent retainCount keys)
	rows, err := db.Query(
		"SELECT key_version FROM keys ORDER BY key_version DESC LIMIT -1 OFFSET ?",
		retainCount,
	)
	if err != nil {
		return fmt.Errorf("failed to query old keys: %w", err)
	}
	defer rows.Close()

	var versionsToDelete []int
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return fmt.Errorf("failed to scan key version: %w", err)
		}
		versionsToDelete = append(versionsToDelete, version)
	}

	// Delete old keys from database and keyring
	for _, version := range versionsToDelete {
		_, err = db.Exec("DELETE FROM keys WHERE key_version = ?", version)
		if err != nil {
			return fmt.Errorf("failed to delete key version %d: %w", version, err)
		}
		delete(globalKeyring, version)
	}

	return nil
}

// GetKeyForDecryption retrieves a key for decryption by version
func GetKeyForDecryption(keyVersion int) ([]byte, error) {
	globalKeyringMutex.RLock()
	defer globalKeyringMutex.RUnlock()

	key, exists := globalKeyring[keyVersion]
	if !exists {
		return nil, fmt.Errorf("key version %d not found in keyring", keyVersion)
	}
	return key, nil
}

// GetActiveKeyVersion returns the currently active key version
func GetActiveKeyVersion(db *sql.DB) (int, error) {
	var activeVersion int
	err := db.QueryRow("SELECT key_version FROM keys WHERE is_active = 1 ORDER BY key_version DESC LIMIT 1").Scan(&activeVersion)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("no active key found")
		}
		return 0, fmt.Errorf("failed to get active key version: %w", err)
	}
	return activeVersion, nil
}

// CheckKeyRotationNeeded determines if key rotation is needed based on config
func CheckKeyRotationNeeded(db *sql.DB, config *KeyRotationConfig) (bool, error) {
	if config == nil {
		return false, nil
	}

	// Get the most recent key creation time
	var createdAt int64
	err := db.QueryRow("SELECT created_at FROM keys WHERE is_active = 1 ORDER BY key_version DESC LIMIT 1").Scan(&createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return true, nil // No keys exist, rotation needed
		}
		return false, fmt.Errorf("failed to get key creation time: %w", err)
	}

	keyAge := time.Since(time.Unix(createdAt, 0))
	return keyAge >= config.RotationInterval, nil
}

// getEncryptionConfig returns the encryption mode, derived key, and selected encryptor.
func getEncryptionConfig() (mode string, masterKey []byte, derivedKey []byte, encryptor Encryptor, err error) {
	ctx := GetGlobalContext()

	mode = ctx.EncryptionMode
	keyVersion := ctx.ActiveKeyVersion

	if mode != "off" && ctx.MasterKey == "" {
		return "", nil, nil, nil, fmt.Errorf("encryption enabled but MORFX_MASTER_KEY is not set")
	}

	if ctx.MasterKey == "" || mode == "off" {
		return "off", nil, nil, nil, nil // encryption off or not set
	}

	masterKey, err = hex.DecodeString(ctx.MasterKey)
	if err != nil {
		return mode, nil, nil, nil, fmt.Errorf("invalid encryption master key hex: %w", err)
	}

	encryptor, err = getEncryptor(ctx.EncryptionAlgo)
	if err != nil {
		return mode, masterKey, nil, nil, err
	}

	// Derived key will be managed by initKeyring and globalKeyring
	// For now, we return the derived key for the active version for immediate use.
	// In a real scenario, you'd fetch from the keyring.
	derivedKey, err = deriveKey(masterKey, []byte("morfx-patches"), fmt.Appendf(nil, "v%d", keyVersion), encryptor.AlgoKeyLen())
	if err != nil {
		return mode, masterKey, nil, encryptor, fmt.Errorf("failed to derive key for active version %d: %w", keyVersion, err)
	}

	return mode, masterKey, derivedKey, encryptor, nil
}

// EncryptBlobWithVersion encrypts data with key version information
func EncryptBlobWithVersion(db *sql.DB, plaintext, aad []byte, encryptor Encryptor) ([]byte, error) {
	// Get active key version
	activeVersion, err := GetActiveKeyVersion(db)
	if err != nil {
		return nil, fmt.Errorf("failed to get active key version: %w", err)
	}

	// Get the key for encryption
	key, err := GetKeyForDecryption(activeVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get encryption key: %w", err)
	}

	return encryptBlobWithKeyVersion(key, plaintext, aad, encryptor, activeVersion)
}

// encryptBlobWithKeyVersion encrypts data with a specific key version
func encryptBlobWithKeyVersion(key, plaintext, aad []byte, encryptor Encryptor, keyVersion int) ([]byte, error) {
	nonce := make([]byte, encryptor.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ciphertext, err := encryptor.Encrypt(key, nonce, plaintext, aad)
	if err != nil {
		return nil, err
	}

	// Format: [version:4][nonce:N][ciphertext:...]
	result := make([]byte, 4+len(nonce)+len(ciphertext))
	// Store key version as 4-byte big-endian integer
	result[0] = byte(keyVersion >> 24)
	result[1] = byte(keyVersion >> 16)
	result[2] = byte(keyVersion >> 8)
	result[3] = byte(keyVersion)
	copy(result[4:], nonce)
	copy(result[4+len(nonce):], ciphertext)
	return result, nil
}

// encryptBlob is now a wrapper around the Encryptor interface (legacy function).
func encryptBlob(key, plaintext, aad []byte, encryptor Encryptor) ([]byte, error) {
	nonce := make([]byte, encryptor.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ciphertext, err := encryptor.Encrypt(key, nonce, plaintext, aad)
	if err != nil {
		return nil, err
	}
	// Prepend nonce to ciphertext (legacy format without version)
	result := make([]byte, len(nonce)+len(ciphertext))
	copy(result, nonce)
	copy(result[len(nonce):], ciphertext)
	return result, nil
}

// DecryptBlobWithVersion decrypts data that may contain key version information
func DecryptBlobWithVersion(encrypted, aad []byte, encryptor Encryptor) ([]byte, error) {
	// Check if this is a versioned blob (minimum 4 bytes for version + nonce)
	if len(encrypted) >= 4+encryptor.NonceSize() {
		// Try to extract version (first 4 bytes as big-endian integer)
		keyVersion := int(encrypted[0])<<24 | int(encrypted[1])<<16 | int(encrypted[2])<<8 | int(encrypted[3])

		// Validate that this looks like a reasonable key version (1-1000)
		if keyVersion > 0 && keyVersion <= 1000 {
			// Try to get the key for this version
			key, err := GetKeyForDecryption(keyVersion)
			if err == nil {
				// This appears to be a versioned blob, decrypt with versioned format
				return decryptVersionedBlob(key, encrypted, aad, encryptor)
			}
		}
	}

	// Fall back to legacy format - this requires the caller to provide the correct key
	// This is a limitation of the legacy format, but we maintain backward compatibility
	return nil, fmt.Errorf("cannot decrypt legacy format without explicit key - use DecryptBlobLegacy")
}

// DecryptBlobLegacy decrypts data in the legacy format (without version information)
func DecryptBlobLegacy(key, encrypted, aad []byte, encryptor Encryptor) ([]byte, error) {
	return decryptBlob(key, encrypted, aad, encryptor)
}

// decryptVersionedBlob decrypts data with version information
func decryptVersionedBlob(key, encrypted, aad []byte, encryptor Encryptor) ([]byte, error) {
	if len(encrypted) < 4 {
		return nil, fmt.Errorf("versioned encrypted data too short")
	}

	// Skip version (first 4 bytes) and extract nonce and ciphertext
	nonceSize := encryptor.NonceSize()
	if len(encrypted) < 4+nonceSize {
		return nil, fmt.Errorf("versioned encrypted data too short for nonce")
	}

	nonce := encrypted[4 : 4+nonceSize]
	ciphertext := encrypted[4+nonceSize:]
	return encryptor.Decrypt(key, nonce, ciphertext, aad)
}

// decryptBlob is now a wrapper around the Encryptor interface (legacy function).
func decryptBlob(key, encrypted, aad []byte, encryptor Encryptor) ([]byte, error) {
	nonceSize := encryptor.NonceSize()
	if len(encrypted) < nonceSize {
		return nil, fmt.Errorf("encrypted data too short")
	}
	nonce := encrypted[:nonceSize]
	ciphertext := encrypted[nonceSize:]
	return encryptor.Decrypt(key, nonce, ciphertext, aad)
}
