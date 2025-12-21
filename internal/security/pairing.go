package security

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// PairingState beschreibt den Zustand eines Pairing-Vorgangs
type PairingState int

const (
	PairingPending  PairingState = iota // Warte auf User-Bestätigung
	PairingApproved                     // User hat bestätigt
	PairingRejected                     // User hat abgelehnt
	PairingComplete                     // Pairing abgeschlossen
)

// TrustedMate repräsentiert einen vertrauten Mate
type TrustedMate struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Type         string            `json:"type"` // "writer", "mail", "web-search", "coder", etc.
	PublicKey    string            `json:"publicKey"`
	ExchangeKey  string            `json:"exchangeKey,omitempty"`  // X25519 Exchange Key vom Mate
	SharedSecret string            `json:"sharedSecret,omitempty"` // Berechneter Shared Secret (Base64)
	PairedAt     time.Time         `json:"pairedAt"`
	LastSeen     time.Time         `json:"lastSeen"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	// KI-Konfiguration für diesen Mate
	Model        string `json:"model,omitempty"`        // Spezifisches Modell für diesen Mate
	SystemPrompt string `json:"systemPrompt,omitempty"` // System-Prompt für diesen Mate
	ActiveMode   string `json:"activeMode,omitempty"`   // Aktiver Programmier-Modus (für Coder-Mates)
}

// PairingRequest repräsentiert eine eingehende Pairing-Anfrage
type PairingRequest struct {
	ID              string       `json:"requestId"` // Frontend erwartet requestId
	MateName        string       `json:"mateName"`
	MateType        string       `json:"mateType"`
	MatePublicKey   string       `json:"matePublicKey"`
	MateExchangeKey string       `json:"mateExchangeKey,omitempty"` // X25519 Exchange Key vom Mate
	PairingCode     string       `json:"pairingCode"`
	State           PairingState `json:"state"`
	CreatedAt       time.Time    `json:"createdAt"`
	ExpiresAt       time.Time    `json:"expiresAt"`
}

// PairingManager verwaltet das Mate-Pairing
type PairingManager struct {
	mu              sync.RWMutex
	keyPair         *KeyPair
	exchangeKeyPair *ExchangeKeyPair // X25519 für Key Exchange
	trustedMates    map[string]*TrustedMate
	pendingRequests map[string]*PairingRequest
	storagePath     string

	// Callback wenn neues Pairing ankommt (für UI)
	OnPairingRequest func(req *PairingRequest)
}

// NewPairingManager erstellt einen neuen PairingManager
func NewPairingManager(storagePath string) (*PairingManager, error) {
	pm := &PairingManager{
		trustedMates:    make(map[string]*TrustedMate),
		pendingRequests: make(map[string]*PairingRequest),
		storagePath:     storagePath,
	}

	// Lade oder generiere Navigator-Schlüsselpaar
	if err := pm.loadOrGenerateKeys(); err != nil {
		return nil, err
	}

	// Lade gespeicherte Trusted Mates
	if err := pm.loadTrustedMates(); err != nil {
		return nil, err
	}

	return pm, nil
}

// loadOrGenerateKeys lädt existierende Keys oder generiert neue
func (pm *PairingManager) loadOrGenerateKeys() error {
	keyFile := filepath.Join(pm.storagePath, "navigator_keys.json")

	data, err := os.ReadFile(keyFile)
	if err == nil {
		// Keys existieren, laden
		var stored struct {
			PrivateKey         string `json:"private_key"`
			ExchangePublicKey  string `json:"exchange_public_key,omitempty"`
			ExchangePrivateKey string `json:"exchange_private_key,omitempty"`
		}
		if err := json.Unmarshal(data, &stored); err != nil {
			return err
		}
		// Private Key enthält bei Ed25519 auch den Public Key
		privKey, err := base64Decode(stored.PrivateKey)
		if err != nil {
			return err
		}
		pm.keyPair = &KeyPair{
			PrivateKey: ed25519.PrivateKey(privKey),
			PublicKey:  ed25519.PrivateKey(privKey).Public().(ed25519.PublicKey),
		}

		// X25519 Exchange Key laden oder generieren
		if stored.ExchangePublicKey != "" && stored.ExchangePrivateKey != "" {
			pubKey, err := base64Decode(stored.ExchangePublicKey)
			if err != nil {
				return err
			}
			privExKey, err := base64Decode(stored.ExchangePrivateKey)
			if err != nil {
				return err
			}
			pm.exchangeKeyPair = &ExchangeKeyPair{}
			copy(pm.exchangeKeyPair.PublicKey[:], pubKey)
			copy(pm.exchangeKeyPair.PrivateKey[:], privExKey)
		} else {
			// Neuen Exchange Key generieren und speichern
			ekp, err := GenerateExchangeKeyPair()
			if err != nil {
				return err
			}
			pm.exchangeKeyPair = ekp
			// Speichern mit Exchange Key
			return pm.saveKeys()
		}
		return nil
	}

	// Neue Keys generieren
	kp, err := GenerateKeyPair()
	if err != nil {
		return err
	}
	pm.keyPair = kp

	// X25519 Exchange Key generieren
	ekp, err := GenerateExchangeKeyPair()
	if err != nil {
		return err
	}
	pm.exchangeKeyPair = ekp

	return pm.saveKeys()
}

// saveKeys speichert alle Keys
func (pm *PairingManager) saveKeys() error {
	stored := struct {
		PrivateKey         string `json:"private_key"`
		ExchangePublicKey  string `json:"exchange_public_key"`
		ExchangePrivateKey string `json:"exchange_private_key"`
	}{
		PrivateKey:         base64Encode(pm.keyPair.PrivateKey),
		ExchangePublicKey:  base64Encode(pm.exchangeKeyPair.PublicKey[:]),
		ExchangePrivateKey: base64Encode(pm.exchangeKeyPair.PrivateKey[:]),
	}
	data, _ := json.MarshalIndent(stored, "", "  ")

	if err := os.MkdirAll(pm.storagePath, 0700); err != nil {
		return err
	}
	keyFile := filepath.Join(pm.storagePath, "navigator_keys.json")
	return os.WriteFile(keyFile, data, 0600)
}

// loadTrustedMates lädt die vertrauten Mates aus der Datei
func (pm *PairingManager) loadTrustedMates() error {
	matesFile := filepath.Join(pm.storagePath, "trusted_mates.json")

	data, err := os.ReadFile(matesFile)
	if os.IsNotExist(err) {
		return nil // Keine Mates vorhanden, OK
	}
	if err != nil {
		return err
	}

	var mates []*TrustedMate
	if err := json.Unmarshal(data, &mates); err != nil {
		return err
	}

	for _, mate := range mates {
		pm.trustedMates[mate.ID] = mate
	}
	return nil
}

// saveTrustedMates speichert die vertrauten Mates
// WICHTIG: Diese Funktion erwartet, dass der Aufrufer KEINEN Lock hält!
// Wird intern von saveTrustedMatesLocked aufgerufen wenn Lock bereits gehalten
func (pm *PairingManager) saveTrustedMates() error {
	pm.mu.RLock()
	mates := make([]*TrustedMate, 0, len(pm.trustedMates))
	for _, mate := range pm.trustedMates {
		mates = append(mates, mate)
	}
	pm.mu.RUnlock()

	return pm.writeTrustedMates(mates)
}

// saveTrustedMatesLocked speichert die Mates wenn der Lock bereits gehalten wird
// WICHTIG: Aufrufer muss den Lock bereits halten!
func (pm *PairingManager) saveTrustedMatesLocked() error {
	mates := make([]*TrustedMate, 0, len(pm.trustedMates))
	for _, mate := range pm.trustedMates {
		mates = append(mates, mate)
	}
	return pm.writeTrustedMates(mates)
}

// writeTrustedMates schreibt die Mates in die Datei (ohne Lock)
func (pm *PairingManager) writeTrustedMates(mates []*TrustedMate) error {
	data, err := json.MarshalIndent(mates, "", "  ")
	if err != nil {
		return err
	}

	matesFile := filepath.Join(pm.storagePath, "trusted_mates.json")
	return os.WriteFile(matesFile, data, 0600)
}

// GetPublicKey gibt den öffentlichen Schlüssel des Navigators zurück (Ed25519)
func (pm *PairingManager) GetPublicKey() string {
	return pm.keyPair.PublicKeyToString()
}

// GetExchangePublicKey gibt den X25519 Exchange Key zurück
func (pm *PairingManager) GetExchangePublicKey() string {
	if pm.exchangeKeyPair == nil {
		return ""
	}
	return pm.exchangeKeyPair.PublicKeyToString()
}

// InitiatePairing startet einen neuen Pairing-Vorgang
func (pm *PairingManager) InitiatePairing(mateName, mateType, matePublicKey string) (*PairingRequest, error) {
	return pm.InitiatePairingWithExchangeKey(mateName, mateType, matePublicKey, "")
}

// InitiatePairingWithExchangeKey startet einen neuen Pairing-Vorgang mit X25519 Exchange Key
func (pm *PairingManager) InitiatePairingWithExchangeKey(mateName, mateType, matePublicKey, mateExchangeKey string) (*PairingRequest, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Pairing-Code berechnen
	mateKeyBytes, err := base64Decode(matePublicKey)
	if err != nil {
		return nil, fmt.Errorf("ungültiger Mate Public Key: %w", err)
	}
	pairingCode := GeneratePairingCode(mateKeyBytes, pm.keyPair.PublicKey)

	req := &PairingRequest{
		ID:              GenerateRandomID(16),
		MateName:        mateName,
		MateType:        mateType,
		MatePublicKey:   matePublicKey,
		MateExchangeKey: mateExchangeKey, // X25519 Exchange Key vom Mate
		PairingCode:     pairingCode,
		State:           PairingPending,
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(5 * time.Minute), // 5 Min Timeout
	}

	pm.pendingRequests[req.ID] = req

	// UI benachrichtigen
	if pm.OnPairingRequest != nil {
		pm.OnPairingRequest(req)
	}

	return req, nil
}

// ApprovePairing bestätigt ein Pairing
func (pm *PairingManager) ApprovePairing(requestID string) (*TrustedMate, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	req, ok := pm.pendingRequests[requestID]
	if !ok {
		return nil, fmt.Errorf("Pairing-Anfrage nicht gefunden")
	}

	if time.Now().After(req.ExpiresAt) {
		delete(pm.pendingRequests, requestID)
		return nil, fmt.Errorf("Pairing-Anfrage abgelaufen")
	}

	// Shared Secret berechnen wenn Exchange Key vorhanden
	var sharedSecretB64 string
	if req.MateExchangeKey != "" && pm.exchangeKeyPair != nil {
		mateExKeyBytes, err := base64Decode(req.MateExchangeKey)
		if err == nil && len(mateExKeyBytes) == 32 {
			var mateExKey [32]byte
			copy(mateExKey[:], mateExKeyBytes)

			// X25519 ECDH - Shared Secret berechnen
			sharedSecret, err := pm.exchangeKeyPair.ComputeSharedSecret(mateExKey)
			if err == nil {
				sharedSecretB64 = base64Encode(sharedSecret[:])
				fmt.Printf("🔐 Shared Secret berechnet für %s\n", req.MateName)
			} else {
				fmt.Printf("⚠️ Shared Secret Berechnung fehlgeschlagen: %v\n", err)
			}
		}
	}

	// Trusted Mate erstellen
	mate := &TrustedMate{
		ID:           GenerateRandomID(16),
		Name:         req.MateName,
		Type:         req.MateType,
		PublicKey:    req.MatePublicKey,
		ExchangeKey:  req.MateExchangeKey,
		SharedSecret: sharedSecretB64,
		PairedAt:     time.Now(),
		LastSeen:     time.Now(),
		Metadata:     make(map[string]string),
	}

	pm.trustedMates[mate.ID] = mate
	delete(pm.pendingRequests, requestID)

	// Speichern - saveTrustedMatesLocked weil wir bereits den Lock halten!
	if err := pm.saveTrustedMatesLocked(); err != nil {
		return nil, err
	}

	return mate, nil
}

// RejectPairing lehnt ein Pairing ab
func (pm *PairingManager) RejectPairing(requestID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, ok := pm.pendingRequests[requestID]; !ok {
		return fmt.Errorf("Pairing-Anfrage nicht gefunden")
	}

	delete(pm.pendingRequests, requestID)
	return nil
}

// IsTrusted prüft ob ein Mate vertraut ist (nach Public Key)
func (pm *PairingManager) IsTrusted(publicKey string) (*TrustedMate, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, mate := range pm.trustedMates {
		if mate.PublicKey == publicKey {
			return mate, true
		}
	}
	return nil, false
}

// GetTrustedMateByID gibt einen Mate nach ID zurück
func (pm *PairingManager) GetTrustedMateByID(mateID string) (*TrustedMate, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	mate, ok := pm.trustedMates[mateID]
	return mate, ok
}

// GetSecureChannelForMate erstellt einen SecureChannel für einen Mate
func (pm *PairingManager) GetSecureChannelForMate(mateID string) (*SecureChannel, error) {
	pm.mu.RLock()
	mate, ok := pm.trustedMates[mateID]
	pm.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("Mate %s nicht gefunden", mateID)
	}

	if mate.SharedSecret == "" {
		return nil, fmt.Errorf("Kein Shared Secret für Mate %s", mateID)
	}

	// Base64-decodieren
	sharedSecretBytes, err := base64Decode(mate.SharedSecret)
	if err != nil {
		return nil, fmt.Errorf("Shared Secret decode Fehler: %w", err)
	}

	if len(sharedSecretBytes) != 32 {
		return nil, fmt.Errorf("Ungültige Shared Secret Länge: %d", len(sharedSecretBytes))
	}

	var sessionKey [32]byte
	copy(sessionKey[:], sharedSecretBytes)

	return NewSecureChannel(sessionKey)
}

// GetTrustedMates gibt alle vertrauten Mates zurück
func (pm *PairingManager) GetTrustedMates() []*TrustedMate {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	mates := make([]*TrustedMate, 0, len(pm.trustedMates))
	for _, mate := range pm.trustedMates {
		mates = append(mates, mate)
	}
	return mates
}

// RemoveTrustedMate entfernt einen vertrauten Mate
func (pm *PairingManager) RemoveTrustedMate(mateID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, ok := pm.trustedMates[mateID]; !ok {
		return fmt.Errorf("Mate nicht gefunden")
	}

	delete(pm.trustedMates, mateID)
	return pm.saveTrustedMatesLocked()
}

// UpdateLastSeen aktualisiert den LastSeen-Zeitstempel
func (pm *PairingManager) UpdateLastSeen(mateID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if mate, ok := pm.trustedMates[mateID]; ok {
		mate.LastSeen = time.Now()
	}
}

// UpdateMateConfig aktualisiert die KI-Konfiguration eines Mates
func (pm *PairingManager) UpdateMateConfig(mateID, model, systemPrompt, activeMode string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	mate, ok := pm.trustedMates[mateID]
	if !ok {
		return fmt.Errorf("Mate nicht gefunden: %s", mateID)
	}

	mate.Model = model
	mate.SystemPrompt = systemPrompt
	mate.ActiveMode = activeMode

	return pm.saveTrustedMatesLocked()
}

// GetMateConfig gibt die KI-Konfiguration eines Mates zurück
func (pm *PairingManager) GetMateConfig(mateID string) (model, systemPrompt, activeMode string, err error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	mate, ok := pm.trustedMates[mateID]
	if !ok {
		return "", "", "", fmt.Errorf("Mate nicht gefunden: %s", mateID)
	}

	return mate.Model, mate.SystemPrompt, mate.ActiveMode, nil
}

// GetPendingRequests gibt alle ausstehenden Pairing-Anfragen zurück
func (pm *PairingManager) GetPendingRequests() []*PairingRequest {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	requests := make([]*PairingRequest, 0, len(pm.pendingRequests))
	for _, req := range pm.pendingRequests {
		if time.Now().Before(req.ExpiresAt) {
			requests = append(requests, req)
		}
	}
	return requests
}

// Sign signiert Daten mit dem Navigator-Schlüssel
func (pm *PairingManager) Sign(data []byte) []byte {
	return pm.keyPair.Sign(data)
}

// Helper-Funktionen
func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
