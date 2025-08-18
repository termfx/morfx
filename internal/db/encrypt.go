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
var globalKeyring = make(map[int][]byte)

// initKeyring initializes the keyring by deriving the active key and loading existing keys from the database.
func initKeyring(db *sql.DB, masterKey []byte, activeKeyVersion int, encryptor Encryptor) error {
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
			"INSERT INTO keys(key_version, created_at, algo, key_hash) VALUES(?, ?, ?, ?)",
			activeKeyVersion,
			time.Now().Unix(), // Use Unix timestamp for created_at
			encryptor.Algo(),
			keyHash[:16], // Store first 16 bytes of SHA256(key)
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

// encryptBlob is now a wrapper around the Encryptor interface.
func encryptBlob(key, plaintext, aad []byte, encryptor Encryptor) ([]byte, error) {
	nonce := make([]byte, encryptor.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return encryptor.Encrypt(key, nonce, plaintext, aad)
}

// decryptBlob is now a wrapper around the Encryptor interface.
func decryptBlob(key, encrypted, aad []byte, encryptor Encryptor) ([]byte, error) {
	nonceSize := encryptor.NonceSize()
	if len(encrypted) < nonceSize {
		return nil, fmt.Errorf("encrypted data too short")
	}
	nonce := encrypted[:nonceSize]
	ciphertext := encrypted[nonceSize:]
	return encryptor.Decrypt(key, nonce, ciphertext, aad)
}
