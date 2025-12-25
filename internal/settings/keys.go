package settings

// =============================================================================
// SETTINGS KEYS - Zentrale Definition aller Einstellungs-Schlüssel
// =============================================================================
// Jede Kategorie hat ihren eigenen Prefix für einfache Gruppierung

// --- Model/LLM Settings ---
const (
	KeyDefaultModel  = "model.default"  // Standard-Modell für Chat
	KeySelectedModel = "model.selected" // Aktuell ausgewähltes Modell
)

// --- UI Settings ---
const (
	KeyUITheme            = "ui.theme"              // UI-Theme (tech-dark, etc.)
	KeyDarkMode           = "ui.darkMode"           // Dark Mode aktiviert
	KeyFontSize           = "ui.fontSize"           // Schriftgröße (50-150%)
	KeyWebSearchAnimation = "ui.webSearchAnimation" // Web-Suche Animation (data-wave, orbit, radar, constellation)
)

// --- Expert Settings ---
const (
	KeySelectedExpert = "expert.selected" // Aktuell ausgewählter Experte (ID)
)

// --- LLM Provider ---
const (
	KeyActiveProvider = "llm.provider.active" // Aktiver Provider (llama-server, ollama)
)

// --- Sampling Parameters (KI-Verhalten) ---
const (
	KeySamplingTemperature   = "sampling.temperature"    // Kreativität (0.0-2.0)
	KeySamplingTopP          = "sampling.top_p"          // Nucleus Sampling
	KeySamplingTopK          = "sampling.top_k"          // Top-K Sampling
	KeySamplingMaxTokens     = "sampling.max_tokens"     // Max Ausgabe-Tokens
	KeySamplingRepeatPenalty = "sampling.repeat_penalty" // Wiederholungs-Strafe
)

// --- Vision/Chaining Settings ---
const (
	KeyVisionEnabled     = "vision.enabled"      // Vision aktiviert
	KeyVisionModel       = "vision.model"        // Vision-Modell (LLaVA etc.)
	KeyVisionModelPath   = "vision.model_path"   // Pfad zum GGUF
	KeyVisionMmprojPath  = "vision.mmproj_path"  // Pfad zur mmproj-Datei
	KeyVisionAutoRestore = "vision.auto_restore" // Nach Vision Chat-Modell wiederherstellen

	KeyChainingEnabled              = "chaining.enabled"                 // Model Chaining aktiviert
	KeyChainingVisionModel          = "chaining.vision_model"            // Vision-Modell für Chaining
	KeyChainingAnalysisModel        = "chaining.analysis_model"          // Analyse-Modell
	KeyChainingAutoSelect           = "chaining.auto_select"             // Automatische Modellwahl
	KeyChainingThinkFirst           = "chaining.think_first"             // Erst denken, dann Websuche
	KeyChainingShowIntermediateOutput = "chaining.show_intermediate_output" // Zwischenergebnisse anzeigen
)

// --- Mate Model Settings (Modell-Zuordnung für Mates) ---
const (
	KeyMateEmailModel       = "mate.model.email"       // Modell für E-Mail-Klassifizierung
	KeyMateDocumentModel    = "mate.model.document"    // Modell für Dokumentenanalyse
	KeyMateLogAnalysisModel = "mate.model.loganalysis" // Modell für Log-Analyse
	KeyMateCoderModel       = "mate.model.coder"       // Modell für Code-Assistenz
)

// --- Voice Settings (STT/TTS) ---
const (
	KeyVoiceWhisperModel = "voice.whisper.model" // Whisper-Modell (base, small, medium)
	KeyVoicePiperVoice   = "voice.piper.voice"   // Piper TTS Stimme
	KeyVoiceLanguage     = "voice.language"      // Sprache (de, en, auto)
	KeyVoiceTTSEnabled   = "voice.tts.enabled"   // TTS aktiviert
)

// --- Voice Assistant / Lauschfunktion ---
const (
	KeyVoiceAssistantEnabled  = "voice.assistant.enabled"   // Sprachassistent aktiviert (Lauschfunktion)
	KeyVoiceAssistantWakeWord = "voice.assistant.wakeword"  // Wake Word ("Hey Ewa", "Ewa", custom)
	KeyVoiceAssistantAutoStop = "voice.assistant.autostop"  // Auto-Stop nach Antwort
	KeyVoiceAssistantQuietHoursEnabled = "voice.assistant.quiet_hours.enabled" // Ruhezeiten aktiviert
	KeyVoiceAssistantQuietHoursStart   = "voice.assistant.quiet_hours.start"   // Ruhezeit Start (z.B. "22:00")
	KeyVoiceAssistantQuietHoursEnd     = "voice.assistant.quiet_hours.end"     // Ruhezeit Ende (z.B. "07:00")
)

// --- Web Search Settings ---
const (
	KeyWebSearchEnabled        = "websearch.enabled"          // Web-Suche aktiviert
	KeyWebSearchBraveAPIKey    = "websearch.brave.apikey"     // Brave Search API Key
	KeyWebSearchCustomInstance = "websearch.searxng.custom"   // Eigene SearXNG Instanz
	KeyWebSearchInstances      = "websearch.searxng.instances" // Fallback-Instanzen (JSON)
	KeyWebSearchCount          = "websearch.count"            // Monatlicher Zähler
)

// --- User Preferences ---
const (
	KeyLocale = "user.locale" // Sprache (de, en)
)

// --- Setup/Legal ---
const (
	KeyDisclaimerAccepted   = "setup.disclaimer.accepted"    // Disclaimer akzeptiert
	KeyDisclaimerAcceptedAt = "setup.disclaimer.accepted_at" // Zeitpunkt der Akzeptanz
)
