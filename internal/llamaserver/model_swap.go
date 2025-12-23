// Package llamaserver - Model Swap Manager
// Ermöglicht temporäres Wechseln zwischen Modellen (z.B. Text ↔ Vision)
// mit automatischer VRAM-Verwaltung
package llamaserver

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ModelType definiert den Modelltyp
type ModelType string

const (
	ModelTypeChat   ModelType = "chat"
	ModelTypeVision ModelType = "vision"
	ModelTypeCoder  ModelType = "coder"
)

// ModelSwapManager verwaltet das Wechseln zwischen verschiedenen Modellen
// Optimiert für GPUs mit begrenztem VRAM (z.B. RTX 3060 mit 12GB)
type ModelSwapManager struct {
	server       *Server
	mu           sync.RWMutex

	// Gespeicherte Modell-Pfade für jeden Typ
	chatModelPath   string
	visionModelPath string
	visionMmprojPath string
	coderModelPath  string

	// Aktueller Status
	currentType    ModelType
	swapping       bool
	swapStartTime  time.Time

	// Callbacks für Status-Updates (z.B. für Frontend-Animation)
	onSwapStart func(fromType, toType ModelType, estimatedSeconds int)
	onSwapComplete func(toType ModelType, success bool, durationSeconds float64)
	onSwapProgress func(percent int, message string)
}

// NewModelSwapManager erstellt einen neuen Swap-Manager
func NewModelSwapManager(server *Server) *ModelSwapManager {
	return &ModelSwapManager{
		server:      server,
		currentType: ModelTypeChat, // Default: Chat-Modell
	}
}

// SetChatModel setzt den Pfad für das Chat-Modell
func (m *ModelSwapManager) SetChatModel(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chatModelPath = path
	log.Printf("📝 Chat-Modell registriert: %s", filepath.Base(path))
}

// SetVisionModel setzt den Pfad für das Vision-Modell (LLaVA) inkl. mmproj
func (m *ModelSwapManager) SetVisionModel(modelPath, mmprojPath string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.visionModelPath = modelPath
	m.visionMmprojPath = mmprojPath
	log.Printf("👁️ Vision-Modell registriert: %s (mmproj: %s)",
		filepath.Base(modelPath), filepath.Base(mmprojPath))
}

// SetCoderModel setzt den Pfad für das Coder-Modell
func (m *ModelSwapManager) SetCoderModel(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.coderModelPath = path
	log.Printf("💻 Coder-Modell registriert: %s", filepath.Base(path))
}

// SetSwapCallbacks setzt Callbacks für Status-Updates
func (m *ModelSwapManager) SetSwapCallbacks(
	onStart func(fromType, toType ModelType, estimatedSeconds int),
	onComplete func(toType ModelType, success bool, durationSeconds float64),
	onProgress func(percent int, message string),
) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onSwapStart = onStart
	m.onSwapComplete = onComplete
	m.onSwapProgress = onProgress
}

// GetCurrentType gibt den aktuellen Modelltyp zurück
func (m *ModelSwapManager) GetCurrentType() ModelType {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentType
}

// IsSwapping prüft ob gerade ein Modellwechsel stattfindet
func (m *ModelSwapManager) IsSwapping() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.swapping
}

// GetSwapStatus gibt den aktuellen Swap-Status zurück
func (m *ModelSwapManager) GetSwapStatus() (swapping bool, currentType ModelType, elapsedSeconds float64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	elapsed := 0.0
	if m.swapping && !m.swapStartTime.IsZero() {
		elapsed = time.Since(m.swapStartTime).Seconds()
	}
	return m.swapping, m.currentType, elapsed
}

// EnsureVisionModel stellt sicher, dass das Vision-Modell geladen ist
// Wenn ein anderes Modell läuft, wird gewechselt (mit VRAM-Freigabe)
// Gibt die geschätzte Wartezeit in Sekunden zurück
func (m *ModelSwapManager) EnsureVisionModel() (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Prüfen ob Vision-Modell konfiguriert ist
	if m.visionModelPath == "" {
		return 0, fmt.Errorf("kein Vision-Modell konfiguriert")
	}

	// Prüfen ob Vision-Modell existiert
	if _, err := os.Stat(m.visionModelPath); os.IsNotExist(err) {
		return 0, fmt.Errorf("Vision-Modell nicht gefunden: %s", m.visionModelPath)
	}

	// Wenn bereits Vision-Modell geladen, nichts tun
	if m.currentType == ModelTypeVision && m.server.IsHealthy() {
		log.Printf("👁️ Vision-Modell bereits aktiv")
		return 0, nil
	}

	// Modellwechsel durchführen
	return m.switchToModelLocked(ModelTypeVision, m.visionModelPath, m.visionMmprojPath)
}

// EnsureChatModel stellt sicher, dass das Chat-Modell geladen ist
func (m *ModelSwapManager) EnsureChatModel() (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Prüfen ob Chat-Modell konfiguriert ist
	if m.chatModelPath == "" {
		return 0, fmt.Errorf("kein Chat-Modell konfiguriert")
	}

	// Wenn bereits Chat-Modell geladen, nichts tun
	if m.currentType == ModelTypeChat && m.server.IsHealthy() {
		log.Printf("📝 Chat-Modell bereits aktiv")
		return 0, nil
	}

	// Modellwechsel durchführen
	return m.switchToModelLocked(ModelTypeChat, m.chatModelPath, "")
}

// RestoreOriginalModel stellt das ursprüngliche Modell wieder her (nach Vision-Analyse)
func (m *ModelSwapManager) RestoreOriginalModel() (int, error) {
	return m.EnsureChatModel()
}

// switchToModelLocked führt den eigentlichen Modellwechsel durch (muss mit Lock aufgerufen werden)
func (m *ModelSwapManager) switchToModelLocked(targetType ModelType, modelPath, mmprojPath string) (int, error) {
	if m.swapping {
		return 0, fmt.Errorf("Modellwechsel bereits in Gang")
	}

	m.swapping = true
	m.swapStartTime = time.Now()

	// Geschätzte Dauer basierend auf Modellgröße
	estimatedSeconds := m.estimateSwapDuration(modelPath)

	// Callback: Swap startet
	if m.onSwapStart != nil {
		go m.onSwapStart(m.currentType, targetType, estimatedSeconds)
	}

	// Progress: 10% - Starte Wechsel
	if m.onSwapProgress != nil {
		go m.onSwapProgress(10, "Bereite Modellwechsel vor...")
	}

	log.Printf("🔄 Model-Swap: %s → %s (geschätzt: %ds)", m.currentType, targetType, estimatedSeconds)

	// 1. Aktuelles Modell stoppen
	if m.server.IsRunning() {
		log.Printf("🔄 Stoppe aktuelles Modell...")
		if m.onSwapProgress != nil {
			go m.onSwapProgress(20, "Stoppe aktuelles Modell...")
		}
		if err := m.server.Stop(); err != nil {
			m.swapping = false
			return 0, fmt.Errorf("Stoppen fehlgeschlagen: %w", err)
		}
		// Kurz warten für VRAM-Freigabe
		time.Sleep(1 * time.Second)
	}

	// 2. VRAM prüfen und ggf. bereinigen
	if m.onSwapProgress != nil {
		go m.onSwapProgress(40, "Prüfe VRAM...")
	}
	requiredVRAM := EstimateModelVRAM(modelPath)
	if err := m.server.EnsureVRAMAvailable(requiredVRAM); err != nil {
		log.Printf("⚠️ VRAM-Warnung: %v", err)
		// Nicht abbrechen, nur warnen
	}

	// 3. mmproj setzen wenn Vision-Modell
	if mmprojPath != "" {
		m.server.SetMmprojPath(mmprojPath)
	} else {
		m.server.SetMmprojPath("") // Clear für nicht-Vision-Modelle
	}

	// 4. Neues Modell laden
	if m.onSwapProgress != nil {
		go m.onSwapProgress(60, fmt.Sprintf("Lade %s...", filepath.Base(modelPath)))
	}
	log.Printf("🔄 Lade neues Modell: %s", filepath.Base(modelPath))

	if err := m.server.Start(modelPath); err != nil {
		m.swapping = false
		if m.onSwapComplete != nil {
			go m.onSwapComplete(targetType, false, time.Since(m.swapStartTime).Seconds())
		}
		return 0, fmt.Errorf("Laden fehlgeschlagen: %w", err)
	}

	// 5. Warten bis Server bereit ist
	if m.onSwapProgress != nil {
		go m.onSwapProgress(80, "Warte auf Server-Bereitschaft...")
	}
	maxWait := 60 // Max 60 Sekunden warten
	for i := 0; i < maxWait; i++ {
		if m.server.IsHealthy() {
			break
		}
		time.Sleep(time.Second)

		// Progress Update alle 5 Sekunden
		if i > 0 && i%5 == 0 && m.onSwapProgress != nil {
			progress := 80 + (i * 20 / maxWait)
			go m.onSwapProgress(progress, fmt.Sprintf("Laden... (%ds)", i))
		}
	}

	if !m.server.IsHealthy() {
		m.swapping = false
		if m.onSwapComplete != nil {
			go m.onSwapComplete(targetType, false, time.Since(m.swapStartTime).Seconds())
		}
		return 0, fmt.Errorf("Server nicht bereit nach %d Sekunden", maxWait)
	}

	// Erfolgreich
	duration := time.Since(m.swapStartTime).Seconds()
	m.currentType = targetType
	m.swapping = false

	if m.onSwapProgress != nil {
		go m.onSwapProgress(100, "Modell bereit!")
	}
	if m.onSwapComplete != nil {
		go m.onSwapComplete(targetType, true, duration)
	}

	log.Printf("✅ Model-Swap abgeschlossen: %s in %.1fs", targetType, duration)
	return int(duration), nil
}

// estimateSwapDuration schätzt die Dauer eines Modellwechsels
func (m *ModelSwapManager) estimateSwapDuration(modelPath string) int {
	info, err := os.Stat(modelPath)
	if err != nil {
		return 15 // Default: 15 Sekunden
	}

	sizeGB := float64(info.Size()) / (1024 * 1024 * 1024)

	// Faustregeln (mit SSD):
	// - < 3GB: ~5-8 Sekunden
	// - 3-6GB: ~8-12 Sekunden
	// - 6-10GB: ~12-18 Sekunden
	// - > 10GB: ~18-25 Sekunden
	switch {
	case sizeGB < 3:
		return 8
	case sizeGB < 6:
		return 12
	case sizeGB < 10:
		return 18
	default:
		return 25
	}
}

// WithVisionModel führt eine Funktion mit dem Vision-Modell aus und stellt danach
// das ursprüngliche Modell wieder her. Ideal für einzelne Vision-Anfragen.
func (m *ModelSwapManager) WithVisionModel(fn func() error) error {
	// Aktuellen Typ merken
	m.mu.RLock()
	previousType := m.currentType
	m.mu.RUnlock()

	// Zu Vision wechseln
	if _, err := m.EnsureVisionModel(); err != nil {
		return fmt.Errorf("Vision-Modell laden: %w", err)
	}

	// Funktion ausführen
	fnErr := fn()

	// Zurück zum vorherigen Modell wenn es nicht Vision war
	if previousType != ModelTypeVision {
		if _, err := m.EnsureChatModel(); err != nil {
			log.Printf("⚠️ Fehler beim Zurückwechseln zum Chat-Modell: %v", err)
			// Nicht als Fehler zurückgeben, fn() war evtl. erfolgreich
		}
	}

	return fnErr
}

// AutoDetectModels versucht, Modelle automatisch im ModelsDir zu finden
// Durchsucht auch Unterverzeichnisse (library/, vision/, custom/)
func (m *ModelSwapManager) AutoDetectModels(modelsDir string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if modelsDir == "" {
		return
	}

	// Sammle alle GGUF-Dateien aus modelsDir und Unterverzeichnissen
	var ggufFiles []string

	// Definiere Verzeichnisse die durchsucht werden sollen
	searchDirs := []string{
		modelsDir,
		filepath.Join(modelsDir, "library"),
		filepath.Join(modelsDir, "vision"),
		filepath.Join(modelsDir, "custom"),
	}

	for _, dir := range searchDirs {
		files, err := os.ReadDir(dir)
		if err != nil {
			continue // Verzeichnis existiert nicht - ignorieren
		}

		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(strings.ToLower(file.Name()), ".gguf") {
				continue
			}
			ggufFiles = append(ggufFiles, filepath.Join(dir, file.Name()))
		}
	}

	log.Printf("🔍 AutoDetect: %d GGUF-Dateien gefunden", len(ggufFiles))

	for _, fullPath := range ggufFiles {
		name := strings.ToLower(filepath.Base(fullPath))

		// Vision-Modell erkennen (LLaVA, MiniCPM, etc.)
		// Aber NICHT mmproj-Dateien als Hauptmodell
		if strings.Contains(name, "mmproj") {
			// mmproj für Vision merken
			if m.visionMmprojPath == "" {
				m.visionMmprojPath = fullPath
				log.Printf("👁️ mmproj auto-detected: %s", filepath.Base(fullPath))
			}
			continue
		}

		if strings.Contains(name, "llava") || strings.Contains(name, "vision") ||
		   strings.Contains(name, "minicpm") {
			if m.visionModelPath == "" {
				m.visionModelPath = fullPath
				log.Printf("👁️ Vision-Modell auto-detected: %s", filepath.Base(fullPath))
			}
			continue
		}

		// Coder-Modell erkennen
		if strings.Contains(name, "coder") || strings.Contains(name, "deepseek-coder") {
			if m.coderModelPath == "" {
				m.coderModelPath = fullPath
				log.Printf("💻 Coder-Modell auto-detected: %s", filepath.Base(fullPath))
			}
			continue
		}

		// Alles andere als Chat-Modell (erstes gefundenes)
		if m.chatModelPath == "" {
			// Bevorzuge Qwen und Llama für Chat
			if strings.Contains(name, "qwen") || strings.Contains(name, "llama") ||
			   strings.Contains(name, "mistral") || strings.Contains(name, "gemma") {
				m.chatModelPath = fullPath
				log.Printf("📝 Chat-Modell auto-detected: %s", filepath.Base(fullPath))
			}
		}
	}

	// Fallback: Wenn kein Chat-Modell gefunden, nehme erstes nicht-Vision GGUF
	if m.chatModelPath == "" {
		for _, fullPath := range ggufFiles {
			name := strings.ToLower(filepath.Base(fullPath))
			if !strings.Contains(name, "mmproj") &&
			   !strings.Contains(name, "llava") &&
			   !strings.Contains(name, "vision") {
				m.chatModelPath = fullPath
				log.Printf("📝 Chat-Modell (Fallback): %s", filepath.Base(fullPath))
				break
			}
		}
	}

	// Log gefundene Modelle
	if m.visionModelPath != "" && m.visionMmprojPath != "" {
		log.Printf("✅ Vision-Modell komplett: %s + %s",
			filepath.Base(m.visionModelPath), filepath.Base(m.visionMmprojPath))
	} else if m.visionModelPath != "" {
		log.Printf("⚠️ Vision-Modell ohne mmproj: %s (Vision wird nicht funktionieren!)",
			filepath.Base(m.visionModelPath))
	}
}

// GetConfiguredModels gibt die konfigurierten Modellpfade zurück
func (m *ModelSwapManager) GetConfiguredModels() map[ModelType]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[ModelType]string{
		ModelTypeChat:   m.chatModelPath,
		ModelTypeVision: m.visionModelPath,
		ModelTypeCoder:  m.coderModelPath,
	}
}

// HasVisionModel prüft ob ein Vision-Modell konfiguriert ist
func (m *ModelSwapManager) HasVisionModel() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.visionModelPath != "" && m.visionMmprojPath != ""
}

// GetVisionModelInfo gibt Infos zum Vision-Modell zurück
func (m *ModelSwapManager) GetVisionModelInfo() (modelPath, mmprojPath string, configured bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.visionModelPath, m.visionMmprojPath, m.visionModelPath != "" && m.visionMmprojPath != ""
}
