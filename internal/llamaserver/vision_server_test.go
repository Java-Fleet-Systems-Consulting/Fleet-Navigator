// Package llamaserver - Unit Tests für VisionServer
package llamaserver

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDefaultVisionServerConfig prüft die Standard-Konfiguration
func TestDefaultVisionServerConfig(t *testing.T) {
	// Test mit temporärem Verzeichnis
	tmpDir := t.TempDir()

	config := DefaultVisionServerConfig(tmpDir)

	// Port sollte 2024 sein (Chat auf 2026)
	if config.Port != 2024 {
		t.Errorf("Port sollte 2024 sein, ist aber %d", config.Port)
	}

	// Host sollte localhost sein
	if config.Host != "127.0.0.1" {
		t.Errorf("Host sollte 127.0.0.1 sein, ist aber %s", config.Host)
	}

	// GPU Layers sollte 99 sein (alle auf GPU)
	if config.GPULayers != 99 {
		t.Errorf("GPULayers sollte 99 sein, ist aber %d", config.GPULayers)
	}

	// Context Size sollte 8192 sein
	if config.ContextSize != 8192 {
		t.Errorf("ContextSize sollte 8192 sein, ist aber %d", config.ContextSize)
	}

	// Idle Timeout sollte 5 Minuten sein
	if config.IdleTimeout != 5*time.Minute {
		t.Errorf("IdleTimeout sollte 5m sein, ist aber %v", config.IdleTimeout)
	}

	// Enabled sollte true sein
	if !config.Enabled {
		t.Error("Enabled sollte true sein")
	}
}

// TestNewVisionServer prüft die Erstellung eines neuen VisionServer
func TestNewVisionServer(t *testing.T) {
	config := VisionServerConfig{
		Port:        2024,
		Host:        "127.0.0.1",
		ModelPath:   "/tmp/test-model.gguf",
		MmprojPath:  "/tmp/test-mmproj.gguf",
		IdleTimeout: 5 * time.Minute,
	}

	vs := NewVisionServer(config)

	if vs == nil {
		t.Fatal("NewVisionServer sollte nicht nil zurückgeben")
	}

	if vs.config.Port != 2024 {
		t.Errorf("Port sollte 2024 sein, ist aber %d", vs.config.Port)
	}

	if vs.running {
		t.Error("Server sollte initial nicht laufen")
	}
}

// TestVisionServerGetStatus prüft den Status-Abruf
func TestVisionServerGetStatus(t *testing.T) {
	config := VisionServerConfig{
		Port:        2024,
		Host:        "127.0.0.1",
		ModelPath:   "/tmp/test-model.gguf",
		IdleTimeout: 5 * time.Minute,
	}

	vs := NewVisionServer(config)
	status := vs.GetStatus()

	// Server sollte nicht laufen
	if status.Running {
		t.Error("Server sollte nicht laufen")
	}

	// Port sollte korrekt sein
	if status.Port != 2024 {
		t.Errorf("Port sollte 2024 sein, ist aber %d", status.Port)
	}

	// Ready sollte false sein
	if status.Ready {
		t.Error("Ready sollte false sein wenn Server nicht läuft")
	}

	// IdleTimeout sollte als String vorhanden sein
	if status.IdleTimeout == "" {
		t.Error("IdleTimeout sollte nicht leer sein")
	}
}

// TestVisionServerGetConfig prüft den Config-Abruf
func TestVisionServerGetConfig(t *testing.T) {
	originalConfig := VisionServerConfig{
		Port:        2024,
		Host:        "127.0.0.1",
		ModelPath:   "/tmp/test-model.gguf",
		MmprojPath:  "/tmp/test-mmproj.gguf",
		GPULayers:   48,
		ContextSize: 4096,
		IdleTimeout: 10 * time.Minute,
		Enabled:     true,
	}

	vs := NewVisionServer(originalConfig)
	config := vs.GetConfig()

	// Alle Werte sollten übereinstimmen
	if config.Port != originalConfig.Port {
		t.Errorf("Port stimmt nicht überein: erwartet %d, bekommen %d", originalConfig.Port, config.Port)
	}
	if config.Host != originalConfig.Host {
		t.Errorf("Host stimmt nicht überein: erwartet %s, bekommen %s", originalConfig.Host, config.Host)
	}
	if config.ModelPath != originalConfig.ModelPath {
		t.Errorf("ModelPath stimmt nicht überein: erwartet %s, bekommen %s", originalConfig.ModelPath, config.ModelPath)
	}
	if config.MmprojPath != originalConfig.MmprojPath {
		t.Errorf("MmprojPath stimmt nicht überein: erwartet %s, bekommen %s", originalConfig.MmprojPath, config.MmprojPath)
	}
	if config.GPULayers != originalConfig.GPULayers {
		t.Errorf("GPULayers stimmt nicht überein: erwartet %d, bekommen %d", originalConfig.GPULayers, config.GPULayers)
	}
}

// TestVisionServerSetModelPath prüft das Setzen des Modell-Pfads
func TestVisionServerSetModelPath(t *testing.T) {
	config := VisionServerConfig{
		Port:        2024,
		Host:        "127.0.0.1",
		IdleTimeout: 5 * time.Minute,
	}

	vs := NewVisionServer(config)

	// Initial sollte ModelPath leer sein
	if vs.config.ModelPath != "" {
		t.Error("ModelPath sollte initial leer sein")
	}

	// Modell setzen
	vs.SetModelPath("/new/model.gguf", "/new/mmproj.gguf")

	// Prüfen ob gesetzt
	updatedConfig := vs.GetConfig()
	if updatedConfig.ModelPath != "/new/model.gguf" {
		t.Errorf("ModelPath sollte /new/model.gguf sein, ist aber %s", updatedConfig.ModelPath)
	}
	if updatedConfig.MmprojPath != "/new/mmproj.gguf" {
		t.Errorf("MmprojPath sollte /new/mmproj.gguf sein, ist aber %s", updatedConfig.MmprojPath)
	}
}

// TestVisionServerSetIdleTimeout prüft das Setzen des Idle-Timeouts
func TestVisionServerSetIdleTimeout(t *testing.T) {
	config := VisionServerConfig{
		Port:        2024,
		Host:        "127.0.0.1",
		IdleTimeout: 5 * time.Minute,
	}

	vs := NewVisionServer(config)

	// Neuen Timeout setzen
	vs.SetIdleTimeout(10 * time.Minute)

	// Prüfen ob gesetzt
	updatedConfig := vs.GetConfig()
	if updatedConfig.IdleTimeout != 10*time.Minute {
		t.Errorf("IdleTimeout sollte 10m sein, ist aber %v", updatedConfig.IdleTimeout)
	}
}

// TestVisionServerIsReady prüft den Ready-Status
func TestVisionServerIsReady(t *testing.T) {
	config := VisionServerConfig{
		Port:        2024,
		Host:        "127.0.0.1",
		IdleTimeout: 5 * time.Minute,
	}

	vs := NewVisionServer(config)

	// Ohne laufenden Server sollte IsReady false sein
	if vs.IsReady() {
		t.Error("IsReady sollte false sein wenn Server nicht läuft")
	}
}

// TestVisionServerStartWithoutModel prüft Start ohne Modell
func TestVisionServerStartWithoutModel(t *testing.T) {
	config := VisionServerConfig{
		Port:        2024,
		Host:        "127.0.0.1",
		IdleTimeout: 5 * time.Minute,
		// Kein ModelPath gesetzt!
	}

	vs := NewVisionServer(config)

	// Start sollte Fehler geben wenn kein Modell konfiguriert
	err := vs.Start()
	if err == nil {
		t.Error("Start ohne Modell sollte einen Fehler geben")
		// Falls doch gestartet, stoppen
		vs.Stop()
	}
}

// TestVisionServerStartWithNonExistentModel prüft Start mit nicht-existentem Modell
func TestVisionServerStartWithNonExistentModel(t *testing.T) {
	config := VisionServerConfig{
		Port:        2024,
		Host:        "127.0.0.1",
		ModelPath:   "/non/existent/model.gguf",
		IdleTimeout: 5 * time.Minute,
	}

	vs := NewVisionServer(config)

	// Start sollte Fehler geben wenn Modell nicht existiert
	err := vs.Start()
	if err == nil {
		t.Error("Start mit nicht-existentem Modell sollte einen Fehler geben")
		// Falls doch gestartet, stoppen
		vs.Stop()
	}
}

// TestVisionServerStopWhenNotRunning prüft Stop wenn nicht gestartet
func TestVisionServerStopWhenNotRunning(t *testing.T) {
	config := VisionServerConfig{
		Port:        2024,
		Host:        "127.0.0.1",
		IdleTimeout: 5 * time.Minute,
	}

	vs := NewVisionServer(config)

	// Stop sollte keinen Fehler geben auch wenn nicht gestartet
	err := vs.Stop()
	if err != nil {
		t.Errorf("Stop ohne laufenden Server sollte keinen Fehler geben, aber: %v", err)
	}
}

// TestVisionModelAutoDetection prüft die Auto-Detection von Vision-Modellen
func TestVisionModelAutoDetection(t *testing.T) {
	tmpDir := t.TempDir()
	visionDir := filepath.Join(tmpDir, "models", "vision")

	// Erstelle vision Verzeichnis
	if err := os.MkdirAll(visionDir, 0755); err != nil {
		t.Fatalf("Konnte vision-Verzeichnis nicht erstellen: %v", err)
	}

	// Erstelle Test-Dateien
	testFiles := []string{
		"llava-v1.6-mistral-7b-Q4_K_M.gguf",
		"mmproj-model-f16.gguf",
	}

	for _, file := range testFiles {
		path := filepath.Join(visionDir, file)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Konnte Test-Datei nicht erstellen: %v", err)
		}
	}

	// Scanne Verzeichnis
	entries, err := os.ReadDir(visionDir)
	if err != nil {
		t.Fatalf("Konnte vision-Verzeichnis nicht lesen: %v", err)
	}

	var foundModel, foundMmproj string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		nameLower := filepath.Base(name)

		if containsIgnoreCase(nameLower, "mmproj") {
			foundMmproj = filepath.Join(visionDir, name)
		} else if containsIgnoreCase(nameLower, "llava") || containsIgnoreCase(nameLower, "minicpm") {
			foundModel = filepath.Join(visionDir, name)
		}
	}

	if foundModel == "" {
		t.Error("LLaVA-Modell sollte gefunden werden")
	}
	if foundMmproj == "" {
		t.Error("mmproj sollte gefunden werden")
	}
}

// Hilfsfunktion für case-insensitive contains
func containsIgnoreCase(s, substr string) bool {
	return filepath.Base(s) != "" &&
		(len(s) >= len(substr) &&
		(s == substr ||
		filepath.Base(s) == substr ||
		containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalFold(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func equalFold(s, t string) bool {
	if len(s) != len(t) {
		return false
	}
	for i := 0; i < len(s); i++ {
		if toLower(s[i]) != toLower(t[i]) {
			return false
		}
	}
	return true
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}

// TestVisionServerStatusTimeout prüft TimeUntilStop Berechnung
func TestVisionServerStatusTimeout(t *testing.T) {
	config := VisionServerConfig{
		Port:        2024,
		Host:        "127.0.0.1",
		IdleTimeout: 5 * time.Minute,
	}

	vs := NewVisionServer(config)

	// Status holen
	status := vs.GetStatus()

	// IdleTimeout sollte "5m0s" sein
	if status.IdleTimeout != "5m0s" {
		t.Errorf("IdleTimeout sollte 5m0s sein, ist aber %s", status.IdleTimeout)
	}

	// TimeUntilStop sollte leer sein wenn nicht läuft
	if status.TimeUntilStop != "" {
		t.Errorf("TimeUntilStop sollte leer sein wenn Server nicht läuft, ist aber %s", status.TimeUntilStop)
	}
}
