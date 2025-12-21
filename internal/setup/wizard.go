// Package setup implementiert den Initial-Setup-Wizard für Fleet Navigator
package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// WizardState repräsentiert den aktuellen Zustand des Setup-Wizards
type WizardState struct {
	CurrentStep    int           `json:"currentStep"`
	TotalSteps     int           `json:"totalSteps"`
	Completed      bool          `json:"completed"`
	SystemInfo     *SystemInfo   `json:"systemInfo,omitempty"`
	SelectedModel  string        `json:"selectedModel,omitempty"`
	VoiceEnabled   bool          `json:"voiceEnabled"`
	WhisperModel   string        `json:"whisperModel,omitempty"`
	PiperVoice     string        `json:"piperVoice,omitempty"`
	VisionEnabled  bool          `json:"visionEnabled"`
	VisionModel    string        `json:"visionModel,omitempty"`
	Error          string        `json:"error,omitempty"`
}

// SystemInfo enthält Informationen über das System
type SystemInfo struct {
	OS           string   `json:"os"`
	Arch         string   `json:"arch"`
	TotalRAM     int64    `json:"totalRamGB"`
	AvailableRAM int64    `json:"availableRamGB"`
	HasGPU       bool     `json:"hasGpu"`
	GPUName      string   `json:"gpuName,omitempty"`
	GPUMemory    int64    `json:"gpuMemoryGB,omitempty"`
	CPUCores     int      `json:"cpuCores"`
}

// ModelRecommendation enthält eine Modell-Empfehlung
type ModelRecommendation struct {
	ModelID     string  `json:"modelId"`
	ModelName   string  `json:"modelName"`
	SizeGB      float64 `json:"sizeGB"`
	Description string  `json:"description"`
	Recommended bool    `json:"recommended"`
	Available   bool    `json:"available"`
	Reason      string  `json:"reason,omitempty"`
	MinRAMGB    int64   `json:"minRamGB"`
	MinVRAMGB   int64   `json:"minVramGB"`
}

// SetupProgress enthält Fortschrittsinformationen
type SetupProgress struct {
	Step        string  `json:"step"`
	Message     string  `json:"message"`
	Percent     float64 `json:"percent"`
	BytesTotal  int64   `json:"bytesTotal,omitempty"`
	BytesDone   int64   `json:"bytesDone,omitempty"`
	SpeedMBps   float64 `json:"speedMBps,omitempty"`
	Error       string  `json:"error,omitempty"`
	Done        bool    `json:"done"`
}

// Service verwaltet den Setup-Wizard
type Service struct {
	dataDir     string
	state       *WizardState
	mu          sync.RWMutex
	progressCh  chan SetupProgress
}

// NewService erstellt einen neuen Setup-Service
func NewService(dataDir string) *Service {
	return &Service{
		dataDir: dataDir,
		state: &WizardState{
			CurrentStep: 0,
			TotalSteps:  5,
		},
		progressCh: make(chan SetupProgress, 100),
	}
}

// IsFirstRun prüft ob dies der erste Start ist
func (s *Service) IsFirstRun() bool {
	// Prüfe ob das Hauptverzeichnis existiert
	if _, err := os.Stat(s.dataDir); os.IsNotExist(err) {
		return true
	}

	// Prüfe ob die Setup-Complete-Datei existiert
	setupFile := filepath.Join(s.dataDir, ".setup-complete")
	if _, err := os.Stat(setupFile); os.IsNotExist(err) {
		return true
	}

	return false
}

// GetState gibt den aktuellen Wizard-Zustand zurück
func (s *Service) GetState() *WizardState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// GetSystemInfo sammelt Systeminformationen
func (s *Service) GetSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		CPUCores: runtime.NumCPU(),
	}

	// RAM ermitteln (plattformspezifisch)
	info.TotalRAM, info.AvailableRAM = getMemoryInfo()

	// GPU ermitteln
	info.HasGPU, info.GPUName, info.GPUMemory = getGPUInfo()

	s.mu.Lock()
	s.state.SystemInfo = info
	s.mu.Unlock()

	return info, nil
}

// GetModelRecommendations gibt Modell-Empfehlungen basierend auf dem System zurück
// Hardware-Klassifizierung:
// - < 8GB RAM ohne GPU: ⛔ Nicht unterstützt
// - ≥ 8GB RAM ohne GPU: 1.5B
// - ≥ 16GB RAM oder GPU 4GB: 1.5B, 3B
// - ≥ 32GB RAM oder GPU 8GB: 1.5B, 3B, 7B
// - ≥ 48GB RAM oder GPU 12GB: 1.5B, 3B, 7B, 14B
// - ≥ 64GB RAM oder GPU 24GB: Alle Modelle
func (s *Service) GetModelRecommendations() []ModelRecommendation {
	s.mu.RLock()
	sysInfo := s.state.SystemInfo
	s.mu.RUnlock()

	// Alle verfügbaren Modelle mit Hardware-Anforderungen
	models := []struct {
		ID        string
		Name      string
		SizeGB    float64
		MinRAMGB  int64 // Minimum RAM ohne GPU
		MinVRAMGB int64 // Minimum VRAM mit GPU
		DescGPU   string
		DescCPU   string
	}{
		{
			"qwen2.5-1.5b-instruct-q4_k_m.gguf",
			"Qwen 2.5 1.5B",
			1.1, 8, 2,
			"Schnell und effizient - ideal für einfache Aufgaben",
			"Einsteiger-Modell - schnell auch auf CPU",
		},
		{
			"qwen2.5-3b-instruct-q4_k_m.gguf",
			"Qwen 2.5 3B",
			2.0, 16, 4,
			"Gute Balance zwischen Qualität und Geschwindigkeit",
			"Solide Qualität - akzeptable Geschwindigkeit auf CPU",
		},
		{
			"Qwen2.5-7B-Instruct-Q4_K_M.gguf",
			"Qwen 2.5 7B",
			4.7, 32, 8,
			"Sehr gute Qualität - empfohlen für die meisten Aufgaben",
			"Hohe Qualität - auf CPU etwas langsamer",
		},
		{
			"gemma-2-9b-it-Q4_K_M.gguf",
			"Gemma 2 9B",
			5.8, 32, 10, // Ideal für 12GB GPUs
			"Exzellent für Übersetzung & Multilingual - Google DeepMind",
			"Sehr gute Qualität - starke Multilingual-Fähigkeiten",
		},
		{
			"Meta-Llama-3.1-8B-Instruct-Q4_K_M.gguf",
			"Llama 3.1 8B",
			4.9, 24, 8, // Meta's neuestes 8B - ideal für 12GB GPUs
			"Metas neuestes 8B Modell - 128K Context, schnell & multilingual",
			"Sehr gute Qualität - schnell auf CPU, 128K Context!",
		},
		{
			"phi-4-Q4_K_M.gguf",
			"Phi-4 14B",
			9.1, 48, 16, // 16GB VRAM nötig für flüssige Performance mit Context
			"Starkes Reasoning & Analyse - für 16GB+ GPUs empfohlen",
			"Exzellentes Reasoning - benötigt viel RAM/VRAM",
		},
		{
			"Qwen2.5-14B-Instruct-Q4_K_M.gguf",
			"Qwen 2.5 14B",
			9.0, 48, 16, // 16GB VRAM für flüssige Performance
			"Premium-Qualität - für anspruchsvolle Aufgaben",
			"Premium-Qualität - erfordert Geduld auf CPU",
		},
		{
			"Qwen2.5-32B-Instruct-Q4_K_M.gguf",
			"Qwen 2.5 32B",
			20.0, 64, 24,
			"Top-Qualität - für Experten und komplexe Analysen",
			"Beste Qualität - nur mit sehr viel RAM empfohlen",
		},
	}

	recommendations := make([]ModelRecommendation, 0, len(models))
	var bestIndex int = -1

	for i, m := range models {
		rec := ModelRecommendation{
			ModelID:   m.ID,
			ModelName: m.Name,
			SizeGB:    m.SizeGB,
			MinRAMGB:  m.MinRAMGB,
			MinVRAMGB: m.MinVRAMGB,
			Available: false,
		}

		if sysInfo == nil {
			// Keine Systeminfo - alle als nicht verfügbar markieren
			rec.Description = m.DescCPU
			rec.Reason = "Systeminfo nicht verfügbar"
			recommendations = append(recommendations, rec)
			continue
		}

		// Verfügbarkeit prüfen basierend auf Hardware
		if sysInfo.HasGPU && sysInfo.GPUMemory > 0 {
			// Mit GPU: VRAM ist entscheidend
			if sysInfo.GPUMemory >= m.MinVRAMGB {
				rec.Available = true
				rec.Description = m.DescGPU
				rec.Reason = fmt.Sprintf("✓ Läuft schnell auf %s (%d GB VRAM)", sysInfo.GPUName, sysInfo.GPUMemory)
				bestIndex = i // Größtes verfügbares Modell ist das beste bei GPU
			} else {
				rec.Description = m.DescGPU
				rec.Reason = fmt.Sprintf("⚠️ Benötigt %d GB VRAM (du hast %d GB)", m.MinVRAMGB, sysInfo.GPUMemory)
			}
		} else {
			// Ohne GPU: RAM ist entscheidend
			if sysInfo.TotalRAM >= m.MinRAMGB {
				rec.Available = true
				rec.Description = m.DescCPU
				rec.Reason = fmt.Sprintf("✓ Läuft mit deinen %d GB RAM", sysInfo.TotalRAM)
				// Bei CPU: Kleineres Modell bevorzugen für bessere Geschwindigkeit
				if bestIndex < 0 {
					bestIndex = i
				}
			} else {
				rec.Description = m.DescCPU
				rec.Reason = fmt.Sprintf("⚠️ Benötigt %d GB RAM (du hast %d GB)", m.MinRAMGB, sysInfo.TotalRAM)
			}
		}

		recommendations = append(recommendations, rec)
	}

	// Bestes Modell markieren
	if bestIndex >= 0 && bestIndex < len(recommendations) {
		recommendations[bestIndex].Recommended = true
		recommendations[bestIndex].Reason = "⭐ EMPFOHLEN - " + recommendations[bestIndex].Reason
	}

	// Hardware-Warnung wenn RAM < 8GB
	if sysInfo != nil && sysInfo.TotalRAM < 8 && !sysInfo.HasGPU {
		// Alle Modelle als nicht verfügbar markieren
		for i := range recommendations {
			recommendations[i].Available = false
			recommendations[i].Reason = "⛔ Mindestens 8GB RAM erforderlich"
		}
	}

	return recommendations
}

// GetVoiceOptions gibt Voice-Optionen zurück
func (s *Service) GetVoiceOptions() map[string]interface{} {
	whisperAvailable := isWhisperAvailable()
	piperAvailable := isPiperAvailable()

	return map[string]interface{}{
		"whisperAvailable": whisperAvailable,
		"piperAvailable":   piperAvailable,
		"whisperModels": []map[string]interface{}{
			{"id": "tiny", "name": "Tiny", "sizeMB": 75, "description": "Schnellste, geringste Qualität"},
			{"id": "base", "name": "Base", "sizeMB": 148, "description": "Gute Balance für schwache Hardware"},
			{"id": "small", "name": "Small", "sizeMB": 488, "description": "Bessere Qualität, langsamer"},
			{"id": "turbo-q5", "name": "Turbo Q5", "sizeMB": 574, "description": "Large-Qualität, 6x schneller - EMPFOHLEN"},
			{"id": "medium", "name": "Medium", "sizeMB": 1533, "description": "Hohe Qualität, benötigt mehr RAM"},
			{"id": "large-v3-turbo", "name": "Large V3 Turbo", "sizeMB": 1620, "description": "Beste Turbo-Qualität"},
		},
		"piperVoices": []map[string]interface{}{
			{"id": "de_DE-thorsten-medium", "name": "Thorsten", "sizeMB": 63, "description": "Männlich, neutral - Standard"},
			{"id": "de_DE-thorsten-high", "name": "Thorsten HD", "sizeMB": 90, "description": "Männlich, hohe Qualität"},
			{"id": "de_DE-kerstin-low", "name": "Kerstin", "sizeMB": 30, "description": "Weiblich, klar"},
		},
		"platformNote": getPlatformVoiceNote(),
	}
}

// SetSelectedModel setzt das ausgewählte Modell
func (s *Service) SetSelectedModel(modelID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.SelectedModel = modelID
}

// SetVoiceOptions setzt die Voice-Optionen
func (s *Service) SetVoiceOptions(enabled bool, whisperModel, piperVoice string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.VoiceEnabled = enabled
	s.state.WhisperModel = whisperModel
	s.state.PiperVoice = piperVoice
}

// GetVisionOptions gibt Vision-Optionen zurück (für Dokumentenerkennung)
func (s *Service) GetVisionOptions(sysInfo *SystemInfo) map[string]interface{} {
	// Vision-Modelle mit VRAM-Anforderungen
	// LLaVA braucht: Modell + mmproj (Vision Encoder)
	models := []map[string]interface{}{
		{
			"id":          "llava-v1.6-mistral-7b",
			"name":        "LLaVA 1.6 Mistral 7B",
			"sizeMB":      4500, // ~4.5GB Modell + 624MB mmproj
			"mmproj":      "mmproj-model-f16.gguf",
			"minVramGB":   8,
			"minRamGB":    16,
			"description": "Gute Balance - Standard für Dokumentenerkennung",
			"recommended": true,
		},
		{
			"id":          "llava-llama-3-8b",
			"name":        "LLaVA Llama 3 8B",
			"sizeMB":      5000, // ~5GB Modell + mmproj
			"mmproj":      "llava-llama-3-8b-v1_1-mmproj-f16.gguf",
			"minVramGB":   10,
			"minRamGB":    24,
			"description": "Neuer, basiert auf Llama 3 - bessere Qualität",
			"recommended": false,
		},
		{
			"id":          "minicpm-v-2.6",
			"name":        "MiniCPM-V 2.6",
			"sizeMB":      5500, // ~5.5GB
			"mmproj":      "mmproj-minicpm-v-2.6-f16.gguf",
			"minVramGB":   8,
			"minRamGB":    16,
			"description": "GPT-4V Niveau - sehr gut für OCR und Dokumente",
			"recommended": false,
		},
	}

	// Verfügbarkeit basierend auf Hardware prüfen
	for i := range models {
		minVRAM := models[i]["minVramGB"].(int)
		minRAM := models[i]["minRamGB"].(int)

		available := false
		reason := ""

		if sysInfo != nil {
			if sysInfo.HasGPU && sysInfo.GPUMemory >= int64(minVRAM) {
				available = true
				reason = fmt.Sprintf("✅ GPU: %s (%d GB)", sysInfo.GPUName, sysInfo.GPUMemory)
			} else if sysInfo.TotalRAM >= int64(minRAM) {
				available = true
				reason = fmt.Sprintf("✅ CPU-Modus (%d GB RAM)", sysInfo.TotalRAM)
			} else {
				reason = fmt.Sprintf("❌ Benötigt %d GB VRAM oder %d GB RAM", minVRAM, minRAM)
			}
		}

		models[i]["available"] = available
		models[i]["reason"] = reason
	}

	return map[string]interface{}{
		"models": models,
		"title":  "Dokumentenerkennung & Bildanalyse",
		"description": `Vision-Modelle ermöglichen Fleet Navigator, Bilder und Dokumente zu "sehen" und zu verstehen.

Mit einem Vision-Modell kannst du:
• Eingescannte Dokumente analysieren lassen (Rechnungen, Briefe, Verträge)
• Text aus Bildern extrahieren (OCR)
• Absender und Dokumenttyp automatisch erkennen
• Dringlichkeit einschätzen (z.B. Mahnungen priorisieren)
• Fotos und Screenshots beschreiben lassen

Ohne Vision-Modell funktioniert die Dokumentenerkennung NICHT.`,
		"note":       "Benötigt zusätzlich ~5 GB Speicherplatz. Empfohlen für Büro-Anwendungen.",
		"optional":   false, // Vision ist wichtig für Dokumentenerkennung
		"recommended": true,
	}
}

// SetVisionOptions setzt die Vision-Optionen
func (s *Service) SetVisionOptions(enabled bool, visionModel string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.VisionEnabled = enabled
	s.state.VisionModel = visionModel
}

// SetStep setzt den aktuellen Schritt
func (s *Service) SetStep(step int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.CurrentStep = step
}

// GetProgressChannel gibt den Progress-Channel zurück
func (s *Service) GetProgressChannel() <-chan SetupProgress {
	return s.progressCh
}

// SendProgress sendet eine Fortschrittsnachricht
func (s *Service) SendProgress(progress SetupProgress) {
	select {
	case s.progressCh <- progress:
	default:
		// Channel voll, überspringen
	}
}

// CompleteSetup markiert das Setup als abgeschlossen
func (s *Service) CompleteSetup() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Setup-Complete-Datei erstellen
	setupFile := filepath.Join(s.dataDir, ".setup-complete")
	if err := os.WriteFile(setupFile, []byte("1"), 0644); err != nil {
		return fmt.Errorf("Setup-Complete-Datei erstellen: %w", err)
	}

	s.state.Completed = true
	s.state.CurrentStep = s.state.TotalSteps

	return nil
}

// ResetSetup setzt das Setup zurück (für Neustart)
func (s *Service) ResetSetup() error {
	setupFile := filepath.Join(s.dataDir, ".setup-complete")
	if err := os.Remove(setupFile); err != nil && !os.IsNotExist(err) {
		return err
	}

	s.mu.Lock()
	s.state = &WizardState{
		CurrentStep: 0,
		TotalSteps:  5,
	}
	s.mu.Unlock()

	return nil
}
