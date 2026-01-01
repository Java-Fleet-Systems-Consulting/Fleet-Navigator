// Package llamaserver - Separater Vision-Server für On-Demand Bildanalyse
// Läuft parallel zum Chat-Server auf eigenem Port
package llamaserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// VisionServerConfig enthält die Konfiguration für den Vision-Server
type VisionServerConfig struct {
	Port           int           `json:"port"`           // Port für Vision-Server (default: 8081)
	Host           string        `json:"host"`           // Host (default: 127.0.0.1)
	ModelPath      string        `json:"modelPath"`      // Pfad zum Vision-Modell (GGUF)
	MmprojPath     string        `json:"mmprojPath"`     // Pfad zum mmproj (Multimodal Projector)
	BinaryPath     string        `json:"binaryPath"`     // Pfad zur llama-server Binary
	LibraryPath    string        `json:"libraryPath"`    // Pfad zu Shared Libraries
	GPULayers      int           `json:"gpuLayers"`      // GPU Layer (-ngl)
	ContextSize    int           `json:"contextSize"`    // Context Size (-c)
	IdleTimeout    time.Duration `json:"idleTimeout"`    // Timeout nach Inaktivität (default: 5 Min)
	Enabled        bool          `json:"enabled"`        // Vision-Server aktiviert
	// Multi-GPU Support
	MainGPU        int           `json:"mainGpu"`        // --main-gpu: GPU-Index (-1 = auto)
	Backend        string        `json:"backend"`        // cuda, rocm, vulkan, cpu
}

// DefaultVisionServerConfig gibt die Standard-Konfiguration zurück
func DefaultVisionServerConfig(dataDir string) VisionServerConfig {
	binaryPath, libraryPath, _ := GetOrExtractLlamaServer(dataDir)

	return VisionServerConfig{
		Port:        2024, // Vision-Server auf Port 2024 (Chat auf 2026)
		Host:        "127.0.0.1",
		BinaryPath:  binaryPath,
		LibraryPath: libraryPath,
		GPULayers:   99,
		ContextSize: 8192, // Vision braucht weniger Context
		IdleTimeout: 5 * time.Minute,
		Enabled:     true,
		MainGPU:     -1,     // -1 = automatische Auswahl
		Backend:     "auto", // auto = beste verfügbare
	}
}

// VisionServerStatus enthält den aktuellen Status des Vision-Servers
type VisionServerStatus struct {
	Running        bool      `json:"running"`
	ModelName      string    `json:"modelName,omitempty"`
	Port           int       `json:"port"`
	LastUsed       time.Time `json:"lastUsed,omitempty"`
	IdleTimeout    string    `json:"idleTimeout"`
	TimeUntilStop  string    `json:"timeUntilStop,omitempty"`
	VRAM           string    `json:"vram,omitempty"`
	Ready          bool      `json:"ready"`          // True wenn Model geladen und bereit
	ActiveRequests int       `json:"activeRequests"` // Anzahl aktiver Anfragen
}

// VisionServer verwaltet einen separaten llama-server Prozess für Vision
type VisionServer struct {
	config          VisionServerConfig
	cmd             *exec.Cmd
	running         bool
	modelName       string
	lastUsed        time.Time
	mu              sync.RWMutex
	cancelFunc      context.CancelFunc
	idleTimer       *time.Timer
	httpClient      *http.Client
	startingUp      bool // True während Server hochfährt
	activeRequests  int  // Anzahl aktiver Anfragen (verhindert Idle-Timeout während Verarbeitung)
}

// NewVisionServer erstellt einen neuen Vision-Server Manager
func NewVisionServer(config VisionServerConfig) *VisionServer {
	return &VisionServer{
		config: config,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// GetStatus gibt den aktuellen Status zurück
func (vs *VisionServer) GetStatus() VisionServerStatus {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	status := VisionServerStatus{
		Running:        vs.running,
		ModelName:      vs.modelName,
		Port:           vs.config.Port,
		LastUsed:       vs.lastUsed,
		IdleTimeout:    vs.config.IdleTimeout.String(),
		Ready:          vs.running && !vs.startingUp,
		ActiveRequests: vs.activeRequests,
	}

	if vs.running && !vs.lastUsed.IsZero() {
		elapsed := time.Since(vs.lastUsed)
		remaining := vs.config.IdleTimeout - elapsed
		if remaining > 0 {
			status.TimeUntilStop = fmt.Sprintf("%.0fs", remaining.Seconds())
		}
	}

	return status
}

// IsRunning prüft ob der Server läuft
func (vs *VisionServer) IsRunning() bool {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	return vs.running
}

// IsReady prüft ob der Server bereit ist (läuft und Modell geladen)
func (vs *VisionServer) IsReady() bool {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	return vs.running && !vs.startingUp
}

// EnsureRunning stellt sicher, dass der Server läuft
// Startet ihn bei Bedarf und wartet bis er bereit ist
func (vs *VisionServer) EnsureRunning() error {
	vs.mu.Lock()

	// Schon am Laufen?
	if vs.running && !vs.startingUp {
		vs.lastUsed = time.Now()
		vs.resetIdleTimer()
		vs.mu.Unlock()
		return nil
	}

	// Startet gerade hoch? Warten
	if vs.startingUp {
		vs.mu.Unlock()
		return vs.waitForReady(60 * time.Second)
	}

	vs.mu.Unlock()

	// Neu starten
	if err := vs.Start(); err != nil {
		return err
	}

	return vs.waitForReady(60 * time.Second)
}

// Start startet den Vision-Server
func (vs *VisionServer) Start() error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if vs.running {
		return nil // Schon am Laufen
	}

	// Prüfe ob Modell konfiguriert
	if vs.config.ModelPath == "" {
		return fmt.Errorf("kein Vision-Modell konfiguriert")
	}

	// Backend-spezifische Binary suchen wenn Backend konfiguriert
	if vs.config.Backend != "" && vs.config.Backend != "auto" {
		dataDir := filepath.Dir(filepath.Dir(vs.config.BinaryPath)) // bin -> dataDir
		if binPath, libPath, err := GetLlamaServerForBackend(dataDir, vs.config.Backend); err == nil {
			vs.config.BinaryPath = binPath
			vs.config.LibraryPath = libPath
			log.Printf("[VisionServer] 🔧 Backend: %s, Binary: %s", vs.config.Backend, binPath)
		}
	}

	// Prüfe ob Binary vorhanden
	if vs.config.BinaryPath == "" {
		return fmt.Errorf("llama-server Binary nicht gefunden")
	}

	// === VRAM-GUARD: Automatischer Fallback auf CPU wenn VRAM zu knapp ===
	if vs.config.GPULayers > 0 {
		requiredVRAM := EstimateModelVRAM(vs.config.ModelPath)
		availableVRAM := GetAvailableVRAM()
		vramBuffer := int64(500 * 1024 * 1024) // 500 MB Sicherheitspuffer

		log.Printf("[VisionServer] 📊 VRAM-Check: benötigt ~%dMB, verfügbar %dMB",
			requiredVRAM, availableVRAM/(1024*1024))

		if availableVRAM < int64(requiredVRAM*1024*1024)+vramBuffer {
			log.Printf("[VisionServer] ⚠️ VRAM zu knapp! Wechsle auf CPU/RAM-Modus")
			log.Printf("[VisionServer] 💡 Tipp: Kleineres Vision-Modell oder mehr VRAM freigeben")
			vs.config.GPULayers = 0 // Fallback auf CPU
		} else {
			log.Printf("[VisionServer] ✅ VRAM ausreichend für GPU-Modus")
		}
	}

	vs.startingUp = true
	log.Printf("[VisionServer] 🖼️ Starte Vision-Server auf Port %d...", vs.config.Port)

	// Modellname aus Pfad extrahieren
	vs.modelName = filepath.Base(vs.config.ModelPath)

	// llama-server Argumente
	args := []string{
		"--model", vs.config.ModelPath,
		"--port", fmt.Sprintf("%d", vs.config.Port),
		"--host", vs.config.Host,
		"-ngl", fmt.Sprintf("%d", vs.config.GPULayers),
		"-c", fmt.Sprintf("%d", vs.config.ContextSize),
	}

	// Flash Attention nur bei GPU-Modus (GPULayers > 0)
	if vs.config.GPULayers > 0 {
		args = append(args, "--flash-attn")
	} else {
		log.Printf("[VisionServer] 🧠 CPU-Modus aktiv - Flash Attention deaktiviert")
	}

	// Multi-GPU: --main-gpu für spezifische GPU-Zuweisung
	if vs.config.MainGPU >= 0 {
		args = append(args, "--main-gpu", fmt.Sprintf("%d", vs.config.MainGPU))
		log.Printf("[VisionServer] 🎮 Verwende GPU #%d", vs.config.MainGPU)
	}

	// Multimodal Projector hinzufügen wenn vorhanden
	if vs.config.MmprojPath != "" {
		args = append(args, "--mmproj", vs.config.MmprojPath)
		log.Printf("[VisionServer] 📸 Mit mmproj: %s", filepath.Base(vs.config.MmprojPath))
	}

	// Context mit Cancel für sauberes Beenden
	ctx, cancel := context.WithCancel(context.Background())
	vs.cancelFunc = cancel

	// Prozess starten
	vs.cmd = exec.CommandContext(ctx, vs.config.BinaryPath, args...)

	// Umgebungsvariablen setzen
	vs.cmd.Env = vs.cmd.Environ()
	if vs.config.LibraryPath != "" {
		vs.cmd.Env = append(vs.cmd.Env, "LD_LIBRARY_PATH="+vs.config.LibraryPath)
	}

	// Im CPU-Modus: CUDA komplett deaktivieren um GPU-Speicher zu sparen
	if vs.config.GPULayers == 0 {
		vs.cmd.Env = append(vs.cmd.Env, "CUDA_VISIBLE_DEVICES=")
		log.Printf("[VisionServer] 🚫 CUDA deaktiviert - reiner CPU-Modus")
	}

	// In Hintergrund starten
	if err := vs.cmd.Start(); err != nil {
		vs.startingUp = false
		return fmt.Errorf("Vision-Server Start fehlgeschlagen: %w", err)
	}

	vs.running = true
	vs.lastUsed = time.Now()

	// Goroutine für Prozess-Überwachung
	go func() {
		err := vs.cmd.Wait()
		vs.mu.Lock()
		vs.running = false
		vs.startingUp = false
		vs.mu.Unlock()

		if err != nil && !strings.Contains(err.Error(), "killed") {
			log.Printf("[VisionServer] ❌ Prozess beendet: %v", err)
		} else {
			log.Printf("[VisionServer] 🛑 Vision-Server gestoppt")
		}
	}()

	// Health-Check in separater Goroutine
	go func() {
		if err := vs.waitForHealthy(60 * time.Second); err != nil {
			log.Printf("[VisionServer] ⚠️ Health-Check fehlgeschlagen: %v", err)
			vs.Stop()
			return
		}

		vs.mu.Lock()
		vs.startingUp = false
		vs.mu.Unlock()

		log.Printf("[VisionServer] ✅ Vision-Server bereit: %s", vs.modelName)

		// Idle-Timer starten
		vs.resetIdleTimer()
	}()

	return nil
}

// Stop beendet den Vision-Server
func (vs *VisionServer) Stop() error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if !vs.running {
		return nil
	}

	log.Printf("[VisionServer] 🛑 Stoppe Vision-Server...")

	// Idle-Timer stoppen
	if vs.idleTimer != nil {
		vs.idleTimer.Stop()
		vs.idleTimer = nil
	}

	// Prozess beenden
	if vs.cancelFunc != nil {
		vs.cancelFunc()
	}

	// Warten auf Beendigung (max 5s)
	done := make(chan struct{})
	go func() {
		if vs.cmd != nil && vs.cmd.Process != nil {
			vs.cmd.Process.Wait()
		}
		close(done)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(5 * time.Second):
		// Force kill
		if vs.cmd != nil && vs.cmd.Process != nil {
			vs.cmd.Process.Kill()
		}
	}

	vs.running = false
	vs.startingUp = false
	vs.modelName = ""

	log.Printf("[VisionServer] ✅ Vision-Server gestoppt")
	return nil
}

// SetIdleTimeout setzt das Idle-Timeout
func (vs *VisionServer) SetIdleTimeout(timeout time.Duration) {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	vs.config.IdleTimeout = timeout
	log.Printf("[VisionServer] Idle-Timeout gesetzt: %v", timeout)
}

// SetModelPath setzt den Modell-Pfad
func (vs *VisionServer) SetModelPath(modelPath, mmprojPath string) {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	vs.config.ModelPath = modelPath
	vs.config.MmprojPath = mmprojPath
	log.Printf("[VisionServer] Modell konfiguriert: %s", filepath.Base(modelPath))
}

// SetGPULayers setzt die Anzahl der GPU-Layer
// 0 = CPU only (verwendet RAM), 99 = alle Layer auf GPU
func (vs *VisionServer) SetGPULayers(layers int) {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	vs.config.GPULayers = layers
	if layers == 0 {
		log.Printf("[VisionServer] 🧠 CPU-Modus aktiviert (RAM wird verwendet)")
	} else {
		log.Printf("[VisionServer] 🎮 GPU-Modus: %d Layer auf GPU", layers)
	}
}

// GetGPULayers gibt die aktuelle GPU-Layer Konfiguration zurück
func (vs *VisionServer) GetGPULayers() int {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	return vs.config.GPULayers
}

// TouchLastUsed aktualisiert den LastUsed-Timestamp und resettet den Idle-Timer
func (vs *VisionServer) TouchLastUsed() {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	vs.lastUsed = time.Now()
	vs.resetIdleTimerLocked()
}

// resetIdleTimer setzt den Idle-Timer zurück (mit Lock)
func (vs *VisionServer) resetIdleTimer() {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	vs.resetIdleTimerLocked()
}

// resetIdleTimerLocked setzt den Idle-Timer zurück (MUSS mit Lock aufgerufen werden!)
func (vs *VisionServer) resetIdleTimerLocked() {
	// Bestehenden Timer stoppen
	if vs.idleTimer != nil {
		vs.idleTimer.Stop()
	}

	// Neuen Timer starten (nur wenn Timeout > 0)
	if vs.config.IdleTimeout > 0 {
		vs.idleTimer = time.AfterFunc(vs.config.IdleTimeout, func() {
			vs.mu.Lock()
			activeReqs := vs.activeRequests
			vs.mu.Unlock()

			// Nicht stoppen wenn noch Anfragen laufen!
			if activeReqs > 0 {
				log.Printf("[VisionServer] ⏰ Idle-Timeout erreicht, aber %d Anfrage(n) aktiv - Timer wird neu gestartet", activeReqs)
				vs.resetIdleTimer()
				return
			}

			log.Printf("[VisionServer] ⏰ Idle-Timeout erreicht (%v), stoppe Server...", vs.config.IdleTimeout)
			vs.Stop()
		})
	}
}

// waitForHealthy wartet bis der Server bereit ist
func (vs *VisionServer) waitForHealthy(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	url := fmt.Sprintf("http://%s:%d/health", vs.config.Host, vs.config.Port)

	for time.Now().Before(deadline) {
		resp, err := vs.httpClient.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("Vision-Server nicht bereit nach %v", timeout)
}

// waitForReady wartet bis der Server bereit ist (public)
func (vs *VisionServer) waitForReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if vs.IsReady() {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("Vision-Server nicht bereit nach %v", timeout)
}

// IsHealthy prüft ob der Server healthy ist
func (vs *VisionServer) IsHealthy() bool {
	if !vs.IsRunning() {
		return false
	}

	url := fmt.Sprintf("http://%s:%d/health", vs.config.Host, vs.config.Port)
	resp, err := vs.httpClient.Get(url)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// GetPort gibt den konfigurierten Port zurück
func (vs *VisionServer) GetPort() int {
	return vs.config.Port
}

// GetConfig gibt die Konfiguration zurück
func (vs *VisionServer) GetConfig() VisionServerConfig {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	return vs.config
}

// VisionAnalysisResult enthält das Ergebnis einer Bildanalyse vom Vision-Server
type VisionAnalysisResult struct {
	Description  string   `json:"description"`
	Text         string   `json:"text,omitempty"`
	DocumentType string   `json:"documentType,omitempty"`
	Objects      []string `json:"objects,omitempty"`
}

// AnalyzeImage analysiert ein Base64-kodiertes Bild
// Startet den Vision-Server automatisch wenn nötig
func (vs *VisionServer) AnalyzeImage(ctx context.Context, imageBase64 string, prompt string) (*VisionAnalysisResult, error) {
	// Server sicherstellen
	if err := vs.EnsureRunning(); err != nil {
		return nil, fmt.Errorf("Vision-Server nicht verfügbar: %w", err)
	}

	// Aktive Anfrage registrieren (verhindert Idle-Timeout während Verarbeitung)
	vs.mu.Lock()
	vs.activeRequests++
	vs.lastUsed = time.Now()
	vs.mu.Unlock()

	// Am Ende: Anfrage deregistrieren und Timer neu starten
	defer func() {
		vs.mu.Lock()
		vs.activeRequests--
		vs.lastUsed = time.Now()
		vs.mu.Unlock()
		vs.resetIdleTimer() // Timer erst nach Abschluss neu starten
	}()

	log.Printf("[VisionServer] 🔄 Bildanalyse gestartet (aktive Anfragen: %d)", vs.activeRequests)

	// Default-Prompt für Dokumentenanalyse
	if prompt == "" {
		prompt = `Analysiere dieses Bild detailliert auf Deutsch.

Falls es sich um ein Dokument handelt (Brief, Rechnung, Formular, etc.):
1. Extrahiere ALLEN lesbaren Text wörtlich
2. Identifiziere den Dokumenttyp (Rechnung, Brief, Vertrag, Formular, etc.)
3. Extrahiere wichtige Daten (Datum, Beträge, Namen, Adressen)
4. Beschreibe visuelle Elemente (Logos, Stempel, Unterschriften)

Falls es ein Foto oder Grafik ist:
1. Beschreibe was zu sehen ist
2. Identifiziere Objekte und Personen
3. Beschreibe die Szene und Kontext

Antwortformat:
- Beginne mit einer kurzen Zusammenfassung
- Liste dann alle Details auf`
	}

	// OpenAI-kompatible API verwenden
	// llama-server erwartet Bilder im "image_url" Format
	requestBody := map[string]interface{}{
		"model": vs.modelName,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": prompt,
					},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": "data:image/jpeg;base64," + imageBase64,
						},
					},
				},
			},
		},
		"max_tokens": 2048,
		"stream":     false,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("JSON-Fehler: %w", err)
	}

	// API-Request
	url := fmt.Sprintf("http://%s:%d/v1/chat/completions", vs.config.Host, vs.config.Port)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("Request-Fehler: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Längerer Timeout für Vision-Analyse (CPU-Inferenz kann 5-10 Minuten dauern!)
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API-Fehler: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API-Status %d: %s", resp.StatusCode, string(body))
	}

	// Response parsen (OpenAI-Format)
	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("Response-Parse-Fehler: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("keine Antwort vom Vision-Modell")
	}

	content := apiResp.Choices[0].Message.Content
	log.Printf("[VisionServer] ✅ Bildanalyse abgeschlossen: %d Zeichen", len(content))

	// Ergebnis zusammenstellen
	result := &VisionAnalysisResult{
		Description: content,
	}

	// Dokumenttyp erkennen (simple Heuristik)
	contentLower := strings.ToLower(content)
	switch {
	case strings.Contains(contentLower, "rechnung") || strings.Contains(contentLower, "invoice"):
		result.DocumentType = "invoice"
	case strings.Contains(contentLower, "vertrag") || strings.Contains(contentLower, "contract"):
		result.DocumentType = "contract"
	case strings.Contains(contentLower, "brief") || strings.Contains(contentLower, "letter"):
		result.DocumentType = "letter"
	case strings.Contains(contentLower, "formular") || strings.Contains(contentLower, "form"):
		result.DocumentType = "form"
	case strings.Contains(contentLower, "quittung") || strings.Contains(contentLower, "receipt"):
		result.DocumentType = "receipt"
	}

	return result, nil
}
