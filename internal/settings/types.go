package settings

import "time"

// =============================================================================
// SETTINGS TYPES - Alle Datentypen für Settings
// =============================================================================

// AppSetting repräsentiert eine einzelne Anwendungseinstellung in der DB
type AppSetting struct {
	ID          int64     `json:"id"`
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description string    `json:"description,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// --- Sampling Parameters ---

// SamplingParams enthält KI-Sampling-Parameter
type SamplingParams struct {
	Temperature   float64 `json:"temperature"`   // 0.0-2.0, Default: 0.7
	TopP          float64 `json:"topP"`          // 0.0-1.0, Default: 0.9
	TopK          int     `json:"topK"`          // 1-100, Default: 40
	MaxTokens     int     `json:"maxTokens"`     // Max Ausgabe-Tokens, Default: 2048
	RepeatPenalty float64 `json:"repeatPenalty"` // 1.0-2.0, Default: 1.1
}

// DefaultSamplingParams gibt die Standard-Sampling-Parameter zurück
func DefaultSamplingParams() SamplingParams {
	return SamplingParams{
		Temperature:   0.7,
		TopP:          0.9,
		TopK:          40,
		MaxTokens:     2048,
		RepeatPenalty: 1.1,
	}
}

// --- Vision/Chaining Settings ---

// VisionSettings enthält Einstellungen für Bildanalyse
type VisionSettings struct {
	Enabled     bool   `json:"enabled"`     // Vision aktiviert
	Model       string `json:"model"`       // Vision-Modell Name
	ModelPath   string `json:"modelPath"`   // Pfad zum GGUF
	MmprojPath  string `json:"mmprojPath"`  // Pfad zur mmproj-Datei
	AutoRestore bool   `json:"autoRestore"` // Nach Vision Chat-Modell wiederherstellen
	IdleTimeout int    `json:"idleTimeout"` // Vision-Server Idle-Timeout in Sekunden (0 = 5 Min Default, -1 = kein Timeout)
	GPULayers   int    `json:"gpuLayers"`   // GPU Layer: 0 = CPU/RAM, 99 = alle auf GPU (Default)
	AutoStart   bool   `json:"autoStart"`   // Vision-Server beim App-Start hochfahren (statt On-Demand)
}

// ChainingSettings enthält Model-Chaining-Konfiguration
type ChainingSettings struct {
	Enabled              bool   `json:"enabled"`              // Chaining aktiviert
	AutoSelect           bool   `json:"autoSelect"`           // Automatische Modellwahl
	VisionModel          string `json:"visionModel"`          // Vision-Modell für Chaining
	AnalysisModel        string `json:"analysisModel"`        // Analyse-Modell
	ThinkFirst           bool   `json:"thinkFirst"`           // Erst denken, dann Websuche
	WebSearchThinkFirst  bool   `json:"webSearchThinkFirst"`  // Alias für ThinkFirst (Kompatibilität)
	ShowIntermediateOutput bool `json:"showIntermediateOutput"` // Zwischenergebnisse anzeigen
}

// --- Voice Settings ---

// VoiceSettings enthält Voice-Konfiguration (Whisper STT + Piper TTS)
type VoiceSettings struct {
	WhisperModel string `json:"whisperModel"` // z.B. "base", "small", "medium"
	PiperVoice   string `json:"piperVoice"`   // z.B. "de_DE-thorsten-medium"
	Language     string `json:"language"`     // z.B. "de" oder "auto"
	TTSEnabled   bool   `json:"ttsEnabled"`   // TTS aktiviert
}

// DefaultVoiceSettings gibt die Standard-Voice-Einstellungen zurück
func DefaultVoiceSettings() VoiceSettings {
	return VoiceSettings{
		WhisperModel: "base",
		PiperVoice:   "de_DE-kerstin-low",
		Language:     "de",
		TTSEnabled:   false,
	}
}

// --- User Preferences ---

// UserPreferences enthält Benutzer-Präferenzen
type UserPreferences struct {
	Locale   string `json:"locale"`   // Sprache (de, en)
	DarkMode bool   `json:"darkMode"` // Dark Mode aktiviert
}

// --- UI Settings ---

// UISettings enthält UI-bezogene Einstellungen
type UISettings struct {
	Theme              string `json:"theme"`              // Theme (tech-dark, etc.)
	DarkMode           bool   `json:"darkMode"`           // Dark Mode
	FontSize           int    `json:"fontSize"`           // Schriftgröße (50-150)
	WebSearchAnimation string `json:"webSearchAnimation"` // Web-Suche Animation (data-wave, orbit, radar, constellation)
}

// DefaultUISettings gibt die Standard-UI-Einstellungen zurück
func DefaultUISettings() UISettings {
	return UISettings{
		Theme:              "tech-dark",
		DarkMode:           true,
		FontSize:           100,
		WebSearchAnimation: "data-wave",
	}
}

// --- Mate Settings ---

// MateModelSettings enthält die Modell-Zuordnungen für verschiedene Mate-Typen
type MateModelSettings struct {
	EmailModel       string `json:"emailModel"`       // Für E-Mail-Klassifizierung
	DocumentModel    string `json:"documentModel"`    // Für Dokumentenanalyse
	LogAnalysisModel string `json:"logAnalysisModel"` // Für Log-Analyse
	CoderModel       string `json:"coderModel"`       // Für Code-Assistenz
}

// --- Web Search Settings ---

// WebSearchSettings enthält Web-Such-Konfiguration
type WebSearchSettings struct {
	Enabled        bool     `json:"enabled"`        // Web-Suche aktiviert
	BraveAPIKey    string   `json:"braveApiKey"`    // Brave Search API Key
	CustomInstance string   `json:"customInstance"` // Eigene SearXNG Instanz
	Instances      []string `json:"instances"`      // Fallback-Instanzen
}

// --- GPU Settings ---

// GPUAssignment beschreibt die GPU-Zuweisung für einen Server
type GPUAssignment struct {
	GPUID       int    `json:"gpuId"`       // GPU Index (-1 = CPU)
	GPUName     string `json:"gpuName"`     // GPU Name für Anzeige
	Backend     string `json:"backend"`     // cuda, rocm, vulkan, cpu
	GPULayers   int    `json:"gpuLayers"`   // 0 = CPU, 99 = alle Layer auf GPU
	VRAMLimit   int64  `json:"vramLimit"`   // VRAM-Limit in Bytes (0 = kein Limit)
}

// GPUSettings enthält Multi-GPU-Konfiguration
type GPUSettings struct {
	AutoDetect  bool          `json:"autoDetect"`  // Automatische GPU-Erkennung
	ChatGPU     GPUAssignment `json:"chatGpu"`     // GPU für Chat-Server
	VisionGPU   GPUAssignment `json:"visionGpu"`   // GPU für Vision-Server
	Strategy    string        `json:"strategy"`    // Strategie-Beschreibung
	LastDetect  string        `json:"lastDetect"`  // Zeitpunkt der letzten Erkennung
}

// DefaultGPUSettings gibt die Standard-GPU-Einstellungen zurück
func DefaultGPUSettings() GPUSettings {
	return GPUSettings{
		AutoDetect: true,
		ChatGPU: GPUAssignment{
			GPUID:     -1, // Auto-Detect
			Backend:   "auto",
			GPULayers: 99,
		},
		VisionGPU: GPUAssignment{
			GPUID:     -1,
			Backend:   "auto",
			GPULayers: 99,
		},
		Strategy: "Automatische Erkennung",
	}
}
