package security

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
)

// SecureChannel repräsentiert einen verschlüsselten Kommunikationskanal
type SecureChannel struct {
	sessionKey [32]byte
	gcm        cipher.AEAD
}

// NewSecureChannel erstellt einen neuen verschlüsselten Kanal
func NewSecureChannel(sessionKey [32]byte) (*SecureChannel, error) {
	block, err := aes.NewCipher(sessionKey[:])
	if err != nil {
		return nil, fmt.Errorf("AES Cipher Fehler: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("GCM Modus Fehler: %w", err)
	}

	return &SecureChannel{
		sessionKey: sessionKey,
		gcm:        gcm,
	}, nil
}

// Encrypt verschlüsselt Daten mit AES-256-GCM
func (sc *SecureChannel) Encrypt(plaintext []byte) ([]byte, error) {
	nonce, err := GenerateNonce()
	if err != nil {
		return nil, fmt.Errorf("Nonce-Generierung fehlgeschlagen: %w", err)
	}

	// Nonce wird dem Ciphertext vorangestellt
	ciphertext := sc.gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt entschlüsselt Daten mit AES-256-GCM
func (sc *SecureChannel) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < sc.gcm.NonceSize() {
		return nil, fmt.Errorf("Ciphertext zu kurz")
	}

	nonce := ciphertext[:sc.gcm.NonceSize()]
	ciphertext = ciphertext[sc.gcm.NonceSize():]

	plaintext, err := sc.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("Entschlüsselung fehlgeschlagen: %w", err)
	}

	return plaintext, nil
}

// EncryptToBase64 verschlüsselt und encodiert zu Base64 (für JSON/WebSocket)
func (sc *SecureChannel) EncryptToBase64(plaintext []byte) (string, error) {
	encrypted, err := sc.Encrypt(plaintext)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// DecryptFromBase64 decodiert Base64 und entschlüsselt
func (sc *SecureChannel) DecryptFromBase64(encoded string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("Base64 Decode Fehler: %w", err)
	}
	return sc.Decrypt(ciphertext)
}

// EncryptString verschlüsselt einen String
func (sc *SecureChannel) EncryptString(plaintext string) (string, error) {
	return sc.EncryptToBase64([]byte(plaintext))
}

// DecryptString entschlüsselt zu einem String
func (sc *SecureChannel) DecryptString(encoded string) (string, error) {
	plaintext, err := sc.DecryptFromBase64(encoded)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
