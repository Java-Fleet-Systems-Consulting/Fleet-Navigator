package settings

import (
	"time"
)

// AppSetting repräsentiert eine Anwendungseinstellung
type AppSetting struct {
	ID          int64     `json:"id"`
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description string    `json:"description,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Setting Keys Constants
const (
	// Model Selection
	KeyModelSelectionEnabled = "model.selection.enabled"
	KeyCodeModel             = "model.selection.code"
	KeyFastModel             = "model.selection.fast"
	KeyVisionModel           = "model.selection.vision"
	KeyDefaultModel          = "model.default"
	KeySelectedModel         = "model.selected"
	KeySelectedExpert        = "expert.selected"

	// UI Settings
	KeyShowWelcomeTiles = "ui.showWelcomeTiles"
	KeyShowTopBar       = "ui.showTopBar"
	KeyUITheme          = "ui.theme"

	// Vision
	KeyVisionChainingEnabled       = "vision.chaining.enabled"
	KeyVisionChainingSmartSelection = "vision.chaining.smart.selection"
	KeyAutoSwitchVisionOnImage     = "model.auto_switch.vision_on_image"

	// LLM Provider
	KeyActiveProvider = "llm.provider.active"

	// Web Search
	KeyWebSearchBraveAPIKey   = "websearch.brave.apikey"
	KeyWebSearchCustomInstance = "websearch.searxng.custom"
	KeyWebSearchInstances     = "websearch.searxng.instances"
	KeyWebSearchCount         = "websearch.count"

	// File Search
	KeyFileSearchFolders = "filesearch.folders"
	KeyFileSearchEnabled = "filesearch.enabled"

	// Mate Models
	KeyEmailModel       = "mate.model.email"
	KeyDocumentModel    = "mate.model.document"
	KeyLogAnalysisModel = "mate.model.loganalysis"
	KeyCoderModel       = "mate.model.coder"

	// Sampling Parameters (wichtig für KI-Verhalten)
	KeySamplingTemperature = "sampling.temperature"
	KeySamplingTopP        = "sampling.top_p"
	KeySamplingTopK        = "sampling.top_k"
	KeySamplingMaxTokens   = "sampling.max_tokens"
	KeySamplingRepeatPenalty = "sampling.repeat_penalty"

	// Model Chaining (wichtig für KI-Workflow)
	KeyChainingEnabled              = "chaining.enabled"
	KeyChainingAutoSelect           = "chaining.auto_select"
	KeyChainingVisionModel          = "chaining.vision_model"
	KeyChainingAnalysisModel        = "chaining.analysis_model"
	KeyChainingShowIntermediateOutput = "chaining.show_intermediate_output"
	KeyChainingWebSearchThinkFirst  = "chaining.web_search_think_first"

	// User Preferences
	KeyLocale    = "user.locale"
	KeyDarkMode  = "ui.darkMode"

	// Voice Settings
	KeyVoiceWhisperModel = "voice.whisper.model"
	KeyVoicePiperVoice   = "voice.piper.voice"
	KeyVoiceLanguage     = "voice.language"

	// Vision Model Swap (VRAM Memory Management)
	KeyVisionSwapEnabled     = "vision.swap.enabled"      // Auto-Swap für Vision aktivieren
	KeyVisionSwapModelPath   = "vision.swap.model_path"   // Pfad zum Vision-Modell (GGUF)
	KeyVisionSwapMmprojPath  = "vision.swap.mmproj_path"  // Pfad zum mmproj
	KeyVisionSwapAutoRestore = "vision.swap.auto_restore" // Nach Vision automatisch zurück zum Chat-Modell
)

// ModelSelectionSettings enthält die Model-Selection-Einstellungen
type ModelSelectionSettings struct {
	Enabled      bool   `json:"enabled"`
	DefaultModel string `json:"defaultModel"`
	CodeModel    string `json:"codeModel"`
	FastModel    string `json:"fastModel"`
	VisionModel  string `json:"visionModel"`
}

// SamplingParams enthält KI-Sampling-Parameter (wichtig für Session-übergreifende Persistenz)
type SamplingParams struct {
	Temperature   float64 `json:"temperature"`
	TopP          float64 `json:"topP"`
	TopK          int     `json:"topK"`
	MaxTokens     int     `json:"maxTokens"`
	RepeatPenalty float64 `json:"repeatPenalty"`
}

// ChainingSettings enthält Model-Chaining-Konfiguration
type ChainingSettings struct {
	Enabled                bool   `json:"enabled"`
	AutoSelect             bool   `json:"autoSelect"`
	VisionModel            string `json:"visionModel"`
	AnalysisModel          string `json:"analysisModel"`
	ShowIntermediateOutput bool   `json:"showIntermediateOutput"`
	WebSearchThinkFirst    bool   `json:"webSearchThinkFirst"` // Think First: LLM erst selbst, dann Websuche bei Unsicherheit
}

// UserPreferences enthält Benutzer-Präferenzen
type UserPreferences struct {
	Locale   string `json:"locale"`
	DarkMode bool   `json:"darkMode"`
}

// VoiceSettings enthält Voice-Konfiguration (Whisper STT + Piper TTS)
type VoiceSettings struct {
	WhisperModel string `json:"whisperModel"` // z.B. "base", "small", "medium"
	PiperVoice   string `json:"piperVoice"`   // z.B. "de_DE-thorsten-medium"
	Language     string `json:"language"`     // z.B. "de" oder "auto"
}

// VisionSwapSettings enthält Einstellungen für automatisches Vision-Model-Swapping
// Ermöglicht Bildanalyse auch auf GPUs mit begrenztem VRAM (z.B. RTX 3060 mit 12GB)
type VisionSwapSettings struct {
	Enabled     bool   `json:"enabled"`     // Auto-Swap aktiviert
	ModelPath   string `json:"modelPath"`   // Pfad zum Vision-Modell (LLaVA GGUF)
	MmprojPath  string `json:"mmprojPath"`  // Pfad zur mmproj-Datei
	AutoRestore bool   `json:"autoRestore"` // Nach Vision automatisch Chat-Modell wiederherstellen
}
