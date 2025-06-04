package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// CryptoService handles encryption and decryption of sensitive data
type CryptoService struct {
	key []byte
}

// NewCryptoService creates a new crypto service with the encryption key from environment
func NewCryptoService() (*CryptoService, error) {
	// Get encryption key from environment
	keyString := os.Getenv("ENCRYPTION_KEY")
	if keyString == "" {
		return nil, fmt.Errorf("ENCRYPTION_KEY environment variable not set")
	}
	
	// Create a 256-bit key from the provided string
	hash := sha256.Sum256([]byte(keyString))
	
	return &CryptoService{
		key: hash[:],
	}, nil
}

// EncryptCredentials encrypts a map of credentials using AES-256-GCM
func (cs *CryptoService) EncryptCredentials(credentials map[string]string) (string, error) {
	// Marshal credentials to JSON
	credentialsJSON, err := json.Marshal(credentials)
	if err != nil {
		return "", fmt.Errorf("failed to marshal credentials: %w", err)
	}
	
	// Create AES cipher
	block, err := aes.NewCipher(cs.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}
	
	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}
	
	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	
	// Encrypt the data
	ciphertext := gcm.Seal(nonce, nonce, credentialsJSON, nil)
	
	// Encode to base64 for storage
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptCredentials decrypts credentials using AES-256-GCM
func (cs *CryptoService) DecryptCredentials(encrypted string) (map[string]string, error) {
	// Decode from base64
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}
	
	// Create AES cipher
	block, err := aes.NewCipher(cs.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	
	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	
	// Validate minimum length
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	
	// Extract nonce and encrypted data
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	
	// Decrypt the data
	credentialsJSON, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}
	
	// Unmarshal the JSON
	var credentials map[string]string
	if err := json.Unmarshal(credentialsJSON, &credentials); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}
	
	return credentials, nil
}

// EncryptString encrypts a single string value
func (cs *CryptoService) EncryptString(plaintext string) (string, error) {
	// Create AES cipher
	block, err := aes.NewCipher(cs.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}
	
	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}
	
	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	
	// Encrypt the data
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	
	// Encode to base64 for storage
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString decrypts a single string value
func (cs *CryptoService) DecryptString(encrypted string) (string, error) {
	// Decode from base64
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}
	
	// Create AES cipher
	block, err := aes.NewCipher(cs.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}
	
	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}
	
	// Validate minimum length
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	
	// Extract nonce and encrypted data
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	
	// Decrypt the data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt string: %w", err)
	}
	
	return string(plaintext), nil
}