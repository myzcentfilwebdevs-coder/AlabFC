package database

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	EncPrefix        = "ENC:"
	DefaultVaultFile = "databases.vault"
	MasterKeyEnv     = "ALAB_MASTER_KEY"
)

func ResolveMasterKey() ([]byte, error) {
	raw := strings.Trim(strings.TrimSpace(os.Getenv(MasterKeyEnv)), "'\"")
	if raw == "" {
		return nil, fmt.Errorf("%s environment variable is not set", MasterKeyEnv)
	}

	if len(raw) == 64 {
		if decoded, err := hex.DecodeString(raw); err == nil && len(decoded) == 32 {
			return decoded, nil
		}
	}

	sum := sha256.Sum256([]byte(raw))
	return sum[:], nil
}

func GenerateMasterKeyHex() (string, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", err
	}
	return hex.EncodeToString(key), nil
}

func EncryptValue(plaintext string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", errors.New("master key must be 32 bytes (AES-256)")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return EncPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

func DecryptValue(encrypted string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", errors.New("master key must be 32 bytes (AES-256)")
	}

	trimmed := strings.TrimSpace(encrypted)
	if !strings.HasPrefix(trimmed, EncPrefix) {
		return "", errors.New("value is not encrypted with ENC: prefix")
	}

	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(trimmed, EncPrefix))
	if err != nil {
		return "", fmt.Errorf("invalid base64 payload: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(raw) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := raw[:nonceSize], raw[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errors.New("decryption failed: invalid master key or corrupted data")
	}

	return string(plaintext), nil
}

func SaveVault(vaultPath string, configs map[string]Config, key []byte) error {
	if vaultPath == "" {
		vaultPath = DefaultVaultFile
	}
	data, err := json.Marshal(configs)
	if err != nil {
		return err
	}
	encrypted, err := EncryptValue(string(data), key)
	if err != nil {
		return err
	}
	return os.WriteFile(vaultPath, []byte(encrypted), 0600)
}

func LoadVault(vaultPath string, key []byte) (map[string]Config, error) {
	if vaultPath == "" {
		vaultPath = DefaultVaultFile
	}
	raw, err := os.ReadFile(vaultPath)
	if err != nil {
		return nil, err
	}
	decrypted, err := DecryptValue(string(raw), key)
	if err != nil {
		return nil, err
	}
	var configs map[string]Config
	if err := json.Unmarshal([]byte(decrypted), &configs); err != nil {
		return nil, err
	}
	return configs, nil
}
