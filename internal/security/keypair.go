// Package security implementiert das Mate-Pairing-System
// Asymmetrische Verschlüsselung mit Ed25519 + X25519 + AES-256-GCM
package security

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/curve25519"
)

// KeyPair enthält Ed25519 Schlüsselpaar für Signaturen
type KeyPair struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// ExchangeKeyPair enthält X25519 Schlüsselpaar für Key Exchange
type ExchangeKeyPair struct {
	PublicKey  [32]byte
	PrivateKey [32]byte
}

// GenerateKeyPair erstellt ein neues Ed25519 Schlüsselpaar
func GenerateKeyPair() (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("Schlüsselpaar-Generierung fehlgeschlagen: %w", err)
	}
	return &KeyPair{
		PublicKey:  pub,
		PrivateKey: priv,
	}, nil
}

// GenerateExchangeKeyPair erstellt ein X25519 Schlüsselpaar für ECDH
func GenerateExchangeKeyPair() (*ExchangeKeyPair, error) {
	var priv, pub [32]byte

	_, err := io.ReadFull(rand.Reader, priv[:])
	if err != nil {
		return nil, fmt.Errorf("Zufallszahlen-Generierung fehlgeschlagen: %w", err)
	}

	// Clamp private key für X25519
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64

	curve25519.ScalarBaseMult(&pub, &priv)

	return &ExchangeKeyPair{
		PublicKey:  pub,
		PrivateKey: priv,
	}, nil
}

// Sign signiert Daten mit dem privaten Schlüssel
func (kp *KeyPair) Sign(data []byte) []byte {
	return ed25519.Sign(kp.PrivateKey, data)
}

// Verify prüft eine Signatur mit einem öffentlichen Schlüssel
func Verify(publicKey ed25519.PublicKey, data, signature []byte) bool {
	return ed25519.Verify(publicKey, data, signature)
}

// ComputeSharedSecret berechnet das gemeinsame Geheimnis via ECDH
func (ekp *ExchangeKeyPair) ComputeSharedSecret(otherPublic [32]byte) ([32]byte, error) {
	var shared [32]byte
	curve25519.ScalarMult(&shared, &ekp.PrivateKey, &otherPublic)
	return shared, nil
}

// DeriveSessionKey leitet einen AES-256 Session-Key aus dem Shared Secret ab
func DeriveSessionKey(sharedSecret [32]byte, salt []byte) [32]byte {
	h := sha256.New()
	h.Write(sharedSecret[:])
	h.Write(salt)
	var key [32]byte
	copy(key[:], h.Sum(nil))
	return key
}

// PublicKeyToString konvertiert Public Key zu Base64
func (kp *KeyPair) PublicKeyToString() string {
	return base64.StdEncoding.EncodeToString(kp.PublicKey)
}

// PublicKeyFromString parsed einen Base64 Public Key
func PublicKeyFromString(s string) (ed25519.PublicKey, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	if len(data) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("ungültige Public Key Länge")
	}
	return ed25519.PublicKey(data), nil
}

// ExchangePublicKeyToString konvertiert Exchange Public Key zu Base64
func (ekp *ExchangeKeyPair) PublicKeyToString() string {
	return base64.StdEncoding.EncodeToString(ekp.PublicKey[:])
}

// GeneratePairingCode generiert den 6-stelligen Bestätigungscode
// Beide Seiten berechnen denselben Code aus den Public Keys
func GeneratePairingCode(matePublicKey, navigatorPublicKey []byte) string {
	h := sha256.New()
	h.Write(matePublicKey)
	h.Write(navigatorPublicKey)
	hash := h.Sum(nil)

	// Erste 3 Bytes zu 6-stelliger Zahl
	num := int(hash[0])<<16 | int(hash[1])<<8 | int(hash[2])
	code := num % 1000000
	return fmt.Sprintf("%06d", code)
}

// GenerateNonce erstellt eine zufällige Nonce für AES-GCM
func GenerateNonce() ([]byte, error) {
	nonce := make([]byte, 12) // AES-GCM standard nonce size
	_, err := io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, err
	}
	return nonce, nil
}

// GenerateRandomID erstellt eine zufällige ID (hex)
func GenerateRandomID(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
