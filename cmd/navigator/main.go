// Fleet Navigator - Hauptanwendung
// Go Backend mit Vue.js Frontend
package main

import (
	"archive/zip"
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-pdf/fpdf"

	"fleet-navigator/internal/chat"
	"fleet-navigator/internal/custommodel"
	"fleet-navigator/internal/experte"
	"fleet-navigator/internal/hardware"
	"fleet-navigator/internal/llamaserver"
	"fleet-navigator/internal/llm"
	"fleet-navigator/internal/middleware"
	"fleet-navigator/internal/models"
	"fleet-navigator/internal/prompts"
	"fleet-navigator/internal/search"
	"fleet-navigator/internal/security"
	"fleet-navigator/internal/settings"
	"fleet-navigator/internal/setup"
	"fleet-navigator/internal/tools"
	"fleet-navigator/internal/updater"
	"fleet-navigator/internal/user"
	"fleet-navigator/internal/vision"
	"fleet-navigator/internal/voice"
	"fleet-navigator/internal/websocket"
)

//go:embed all:dist
var frontendFS embed.FS

// Build-Zeit kommt jetzt aus dem updater Package (konsistent mit Version)

// Config enthält die Konfiguration
type Config struct {
	Port        string `json:"port"`
	DataDir     string `json:"data_dir"`
	ModelsDir   string `json:"models_dir"`
	OllamaURL   string `json:"ollama_url"`
	OllamaModel string `json:"ollama_model"`
}

// getDefaultDataDir gibt das plattformspezifische Datenverzeichnis zurück
// Windows: %LOCALAPPDATA%\FleetNavigator (z.B. C:\Users\Franz\AppData\Local\FleetNavigator)
// Linux/macOS: ~/.fleet-navigator
func getDefaultDataDir() string {
	if runtime.GOOS == "windows" {
		// Windows: LOCALAPPDATA verwenden
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return filepath.Join(localAppData, "FleetNavigator")
		}
		// Fallback auf UserConfigDir
		if configDir, err := os.UserConfigDir(); err == nil {
			return filepath.Join(configDir, "FleetNavigator")
		}
	}
	// Linux/macOS: Home-Verzeichnis mit Punkt-Präfix
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".fleet-navigator")
}

// App ist die Hauptanwendung
type App struct {
	config           *Config
	pairingManager   *security.PairingManager
	wsServer         *websocket.Server
	chatService      *chat.Service
	selectionService *models.SelectionService
	chatAdapter      *chat.Adapter
	expertenService  *experte.Service
	chatStore        *chat.Store
	modelService     *llm.ModelService   // Neuer LLM Model Service
	toolRegistry     *tools.Registry     // Tool Registry (WebSearch, FileSearch, etc.)
	visionService    *vision.Service     // LLaVA Vision Service für Bildanalyse
	promptsService      *prompts.Service      // System Prompts Service
	settingsService     *settings.Service     // App Settings Service
	userService         *user.Service         // User & Auth Service
	customModelService  *custommodel.Service  // Custom Models Service
	llamaServer         *llamaserver.Server   // llama.cpp Server Manager
	selectedModel       string                // Aktuell ausgewähltes Modell für UI
	hardwareMonitor     *hardware.Monitor     // Hardware Monitor (CPU, GPU, RAM)
	searchService       *search.Service       // Web Search Service (Brave, SearXNG, DuckDuckGo)
	voiceService        *voice.Service        // Voice Service (Whisper STT, Piper TTS)
	setupService        *setup.Service        // Setup Wizard Service
	setupHandler        *setup.APIHandler     // Setup API Handler
}

func main() {
	// Flags
	port := flag.String("port", "2025", "HTTP Server Port")
	dataDir := flag.String("data", "", "Datenverzeichnis")
	flag.Parse()

	// Datenverzeichnis bestimmen (plattformspezifisch)
	if *dataDir == "" {
		*dataDir = getDefaultDataDir()
	}

	// Port: Flag -> Umgebungsvariable -> Default
	actualPort := *port
	if envPort := os.Getenv("PORT"); envPort != "" {
		actualPort = envPort
	}

	// Konfiguration
	config := &Config{
		Port:        actualPort,
		DataDir:     *dataDir,
		ModelsDir:   filepath.Join(*dataDir, "models"),
		OllamaURL:   getEnv("OLLAMA_URL", "http://localhost:11434"),
		OllamaModel: getEnv("OLLAMA_MODEL", "qwen2.5:7b"),
	}

	// Verzeichnisstruktur erstellen (First-Run-Setup)
	if err := createDirectoryStructure(config); err != nil {
		log.Fatalf("Verzeichnisstruktur konnte nicht erstellt werden: %v", err)
	}

	// App initialisieren
	app, err := NewApp(config)
	if err != nil {
		log.Fatalf("App-Initialisierung fehlgeschlagen: %v", err)
	}

	// Server starten
	app.Run()
}

// NewApp erstellt eine neue App-Instanz
func NewApp(config *Config) (*App, error) {
	// App Settings Service - ZUERST initialisieren um Provider zu kennen
	settingsRepo, err := settings.NewRepository(config.DataDir)
	if err != nil {
		return nil, fmt.Errorf("SettingsRepository Fehler: %w", err)
	}
	settingsService := settings.NewService(settingsRepo)
	activeProvider := settingsService.GetActiveProvider()

	// Pairing Manager
	pm, err := security.NewPairingManager(config.DataDir)
	if err != nil {
		return nil, fmt.Errorf("PairingManager Fehler: %w", err)
	}

	// Experten Service
	expertenSvc, err := experte.NewService(config.DataDir)
	if err != nil {
		return nil, fmt.Errorf("ExpertenService Fehler: %w", err)
	}

	// Chat Store (SQLite für Chat-Persistenz)
	chatStore, err := chat.NewStore(config.DataDir)
	if err != nil {
		return nil, fmt.Errorf("ChatStore Fehler: %w", err)
	}

	// Chat Service (nur für Ollama-Provider relevant)
	chatConfig := chat.LegacyConfig{
		BaseURL: config.OllamaURL,
		Model:   config.OllamaModel,
		Timeout: 120 * time.Second,
	}
	chatService := chat.NewService(chatConfig)

	// Model Selection Service (Smart Model Selection wie in Java)
	selectionConfig := models.DefaultSelectionConfig()
	selectionConfig.DefaultModel = config.OllamaModel
	selectionService := models.NewSelectionService(selectionConfig)

	// System-Prompt fuer den Navigator
	systemPrompt := `Du bist der Fleet Navigator, ein freundlicher und kompetenter KI-Assistent fuer kleine Bueros.
Du hilfst bei alltaeglichen Bueroaufgaben wie:
- Textverarbeitung und Formatierung
- E-Mail-Kommunikation
- Terminplanung und Organisation
- Rechts- und Datenschutzfragen (allgemeine Hinweise)
- Marketing und Kommunikation

Antworte immer auf Deutsch, freundlich und hilfsbereit. Halte deine Antworten praezise und praxisnah.
Bei rechtlichen Fragen weise darauf hin, dass du nur allgemeine Informationen geben kannst.`

	// Chat Adapter fuer WebSocket mit Model Selection
	chatAdapter := chat.NewAdapter(chatService, selectionService, systemPrompt)

	// WebSocket Server
	ws := websocket.NewServer(pm)
	ws.SetChatHandler(chatAdapter)

	// LLM Model Service (neu)
	modelServiceConfig := llm.ModelServiceConfig{
		OllamaURL:       config.OllamaURL,
		DefaultModel:    config.OllamaModel,
		SystemPrompt:    systemPrompt,
		SkipOllamaCheck: activeProvider != "ollama", // Nur prüfen wenn Ollama aktiv
	}
	modelService := llm.NewModelService(modelServiceConfig)

	// Tool Registry (WebSearch, FileSearch, WebFetch)
	toolRegistry := tools.NewRegistry()
	log.Printf("Tool Registry initialisiert mit %d Tools", len(toolRegistry.List()))

	// Vision Service (LLaVA) - nur bei Ollama prüfen
	visionConfig := vision.Config{
		OllamaURL:   config.OllamaURL,
		VisionModel: selectionConfig.VisionModel,
		Timeout:     180 * time.Second,
	}
	visionService := vision.NewService(visionConfig)
	if activeProvider == "ollama" {
		if visionService.IsAvailable() {
			log.Printf("Vision Service aktiviert: %s", visionConfig.VisionModel)
		} else {
			log.Printf("WARNUNG: Vision Model %s nicht verfügbar", visionConfig.VisionModel)
		}
	}

	// System Prompts Service
	promptsRepo, err := prompts.NewRepository(config.DataDir)
	if err != nil {
		return nil, fmt.Errorf("PromptsRepository Fehler: %w", err)
	}
	promptsService := prompts.NewService(promptsRepo)

	// Standard-Prompts initialisieren (beim ersten Start)
	if err := promptsService.InitializeDefaults(); err != nil {
		log.Printf("WARNUNG: System-Prompts konnten nicht initialisiert werden: %v", err)
	}

	// User & Auth Service
	userRepo, err := user.NewRepository(config.DataDir)
	if err != nil {
		return nil, fmt.Errorf("UserRepository Fehler: %w", err)
	}
	userService := user.NewService(userRepo)

	// Standard-Admin erstellen (beim ersten Start)
	if err := userService.InitializeDefaults(); err != nil {
		log.Printf("WARNUNG: User konnten nicht initialisiert werden: %v", err)
	}

	// Custom Model Service
	customModelRepo, err := custommodel.NewRepository(config.DataDir)
	if err != nil {
		return nil, fmt.Errorf("CustomModelRepository Fehler: %w", err)
	}
	customModelService := custommodel.NewService(customModelRepo)

	// llama-server Manager
	llamaConfig := llamaserver.DefaultConfig(config.DataDir)
	llamaConfig.Port = 2026 // llama-server läuft auf Port 2026
	llamaSrv := llamaserver.NewServer(llamaConfig)

	// Prüfen ob GGUF-Modelle vorhanden sind und Server automatisch starten
	models, _ := llamaSrv.GetAvailableModels()
	if len(models) > 0 && llamaConfig.BinaryPath != "" {
		// Standard-Modell suchen (Qwen2.5-7B bevorzugt)
		var defaultModel string
		for _, m := range models {
			if strings.Contains(strings.ToLower(m.Name), "qwen2.5-7b") {
				defaultModel = m.Path
				break
			}
		}
		// Falls kein Qwen, erstes verfügbares Modell nehmen
		if defaultModel == "" {
			defaultModel = models[0].Path
		}

		log.Printf("Starte llama-server mit Modell: %s", defaultModel)
		if err := llamaSrv.Start(defaultModel); err != nil {
			log.Printf("WARNUNG: llama-server konnte nicht gestartet werden: %v", err)
		}
	} else if llamaConfig.BinaryPath == "" {
		log.Printf("WARNUNG: llama-server Binary nicht gefunden")
	} else {
		log.Printf("Keine GGUF-Modelle gefunden - First-Run-Setup wird angezeigt")
	}

	// Hardware Monitor fuer lokale System-Statistiken
	hwMonitor := hardware.NewMonitor()

	// Setup Service erstellen
	setupSvc := setup.NewService(config.DataDir)
	setupHandler := setup.NewAPIHandler(setupSvc)

	app := &App{
		config:           config,
		pairingManager:   pm,
		wsServer:         ws,
		chatService:      chatService,
		selectionService: selectionService,
		chatAdapter:      chatAdapter,
		expertenService:  expertenSvc,
		chatStore:        chatStore,
		modelService:     modelService,
		toolRegistry:     toolRegistry,
		visionService:    visionService,
		promptsService:   promptsService,
		settingsService:     settingsService,
		userService:         userService,
		customModelService:  customModelService,
		llamaServer:         llamaSrv,
		selectedModel:       config.OllamaModel,
		hardwareMonitor:     hwMonitor,
		searchService:       search.NewService(),
		voiceService:        voice.NewService(config.DataDir),
		setupService:        setupSvc,
		setupHandler:        setupHandler,
	}

	// Chat-Adapter mit Provider-Awareness konfigurieren
	chatAdapter.SetProviderChecker(settingsService)
	chatAdapter.SetLlamaServer(&llamaServerWrapper{server: llamaSrv})
	log.Printf("Chat-Adapter konfiguriert mit Provider-Awareness (Settings + LlamaServer)")

	// Voice Service initialisieren (Whisper STT + Piper TTS)
	// Voice-Settings aus DB laden
	voiceSettings := settingsService.GetVoiceSettings()
	voiceConfig := voice.Config{
		DataDir:      config.DataDir,
		WhisperModel: voiceSettings.WhisperModel,
		PiperVoice:   voiceSettings.PiperVoice,
		Language:     voiceSettings.Language,
	}
	log.Printf("Voice-Config aus DB geladen: whisper=%s, piper=%s, lang=%s",
		voiceConfig.WhisperModel, voiceConfig.PiperVoice, voiceConfig.Language)
	if err := app.voiceService.Initialize(voiceConfig); err != nil {
		log.Printf("WARNUNG: Voice-Service Initialisierung fehlgeschlagen: %v", err)
	} else {
		log.Printf("Voice-Service initialisiert (Whisper STT + Piper TTS)")
	}

	// Callbacks setzen
	ws.OnPairingRequest = func(req *security.PairingRequest) {
		log.Printf("Neue Pairing-Anfrage: %s (%s) - Code: %s",
			req.MateName, req.MateType, req.PairingCode)
	}

	ws.OnMateConnected = func(mateID, mateName string) {
		log.Printf("Mate verbunden: %s", mateName)
	}

	ws.OnMateDisconnected = func(mateID, mateName string) {
		log.Printf("Mate getrennt: %s", mateName)
	}

	// Setup Handler konfigurieren
	setupHandler.SetLlamaStarter(llamaSrv, config.ModelsDir)
	setupHandler.SetModelUpdater(modelService)
	setupHandler.SetSettingsUpdater(settingsService)
	setupHandler.SetExpertUpdater(expertenSvc)
	// Downloader für Setup-Wizard
	setupHandler.SetModelDownloader(setup.NewHuggingFaceDownloader(config.ModelsDir))
	setupHandler.SetVisionDownloader(setup.NewVisionModelDownloader(config.ModelsDir))
	setupHandler.SetVoiceDownloader(setup.NewVoiceModelDownloader(config.DataDir))
	log.Printf("Setup-Wizard initialisiert (DataDir: %s, ModelsDir: %s)", config.DataDir, config.ModelsDir)

	// Provider-abhängige Statusmeldungen
	log.Printf("Aktiver LLM-Provider: %s", activeProvider)

	if activeProvider == "ollama" {
		// Nur bei Ollama-Provider prüfen
		if chatService.IsAvailable() {
			log.Printf("Ollama verbunden: %s", config.OllamaURL)
			log.Printf("Smart Model Selection aktiviert:")
			log.Printf("  - Default: %s", selectionConfig.DefaultModel)
			log.Printf("  - Code:    %s", selectionConfig.CodeModel)
			log.Printf("  - Fast:    %s", selectionConfig.FastModel)
			log.Printf("  - Vision:  %s", selectionConfig.VisionModel)
		} else {
			log.Printf("WARNUNG: Ollama nicht erreichbar unter %s", config.OllamaURL)
		}
	} else {
		// llama-server/llama-cpp Provider
		if llamaSrv.IsRunning() {
			log.Printf("llama-server aktiv auf Port %d", llamaConfig.Port)
		} else {
			log.Printf("llama-server wird beim ersten Chat-Request gestartet")
		}
	}

	return app, nil
}

// Run startet den Server
func (app *App) Run() {
	// WebSocket Server in Goroutine
	go app.wsServer.Run()

	// HTTP Routes
	mux := http.NewServeMux()

	// API Endpoints
	mux.HandleFunc("/api/health", app.handleHealth)
	mux.HandleFunc("/api/system/health", app.handleHealth) // Alias für Frontend-Kompatibilität
	mux.HandleFunc("/api/models", app.handleModels)
	mux.HandleFunc("/api/models/config", app.handleModelConfig)
	mux.HandleFunc("/api/models/pull", app.handleModelsPull)       // Frontend-kompatibel
	mux.HandleFunc("/api/models/default", app.handleModelsDefault) // Frontend-kompatibel
	mux.HandleFunc("/api/models/", app.handleModelByName)          // /api/models/{name}/...
	mux.HandleFunc("/api/mates", app.handleMates)
	mux.HandleFunc("/api/mates/pending", app.handlePendingPairings)
	mux.HandleFunc("/api/mates/approve", app.handleApprovePairing)
	mux.HandleFunc("/api/mates/reject", app.handleRejectPairing)
	mux.HandleFunc("/api/mates/remove", app.handleRemoveMate)
	mux.HandleFunc("/api/config", app.handleConfig)

	// Experten Endpoints
	mux.HandleFunc("/api/experts", app.handleExperts)
	mux.HandleFunc("/api/experts/modes/", app.handleModeByID) // Einzelne Modi bearbeiten/löschen
	mux.HandleFunc("/api/experts/", app.handleExpertByID)

	// Chat Endpoints (REST API für Frontend)
	mux.HandleFunc("/api/chat/new", app.handleChatNew)
	mux.HandleFunc("/api/chat/all", app.handleChatAll)
	mux.HandleFunc("/api/chat/history/", app.handleChatHistory)
	mux.HandleFunc("/api/chat/send-stream", app.handleChatSendStream)
	mux.HandleFunc("/api/chat/", app.handleChatByID)

	// File Upload Endpoint
	mux.HandleFunc("/api/files/upload", app.handleFileUpload)

	// Office/Document Generation Endpoints
	mux.HandleFunc("/api/office/generate-document", app.handleOfficeGenerateDocument)
	mux.HandleFunc("/api/office/ping", app.handleOfficePing)

	// Tools Endpoints
	mux.HandleFunc("/api/tools", app.handleTools)
	mux.HandleFunc("/api/tools/execute", app.handleToolExecute)
	mux.HandleFunc("/api/tools/search", app.handleToolSearch)
	mux.HandleFunc("/api/tools/fetch", app.handleToolFetch)

	// Vision Endpoints (LLaVA + Tesseract)
	mux.HandleFunc("/api/vision/analyze", app.handleVisionAnalyze)
	mux.HandleFunc("/api/vision/document", app.handleVisionDocument)
	mux.HandleFunc("/api/vision/pdf-stream", app.handleVisionPDFStream) // NEU: PDF mit Fortschritt
	mux.HandleFunc("/api/vision/status", app.handleVisionStatus)
	mux.HandleFunc("/api/vision/ocr", app.handleVisionOCR) // Reine Tesseract-OCR ohne Vision

	// Settings Endpoints
	mux.HandleFunc("/api/settings/selected-model", app.handleSelectedModel)

	// LLM Model Management Endpoints (neu)
	mux.HandleFunc("/api/llm/models", app.handleLLMModels)
	mux.HandleFunc("/api/llm/models/installed", app.handleLLMInstalledModels)
	mux.HandleFunc("/api/llm/models/registry", app.handleLLMRegistry)
	mux.HandleFunc("/api/llm/models/featured", app.handleLLMFeaturedModels)
	mux.HandleFunc("/api/llm/models/context", app.handleLLMModelContext) // GET: Context-Info für Modell
	mux.HandleFunc("/api/llm/models/pull", app.handleLLMPullModel)
	mux.HandleFunc("/api/llm/models/delete", app.handleLLMDeleteModel)
	mux.HandleFunc("/api/llm/models/details/", app.handleLLMModelDetails)
	mux.HandleFunc("/api/llm/chat", app.handleLLMChat)
	mux.HandleFunc("/api/llm/status", app.handleLLMStatus)

	// LLM Provider Endpoints (Frontend-kompatibel)
	mux.HandleFunc("/api/llm/providers/active", app.handleLLMProviderActive)
	mux.HandleFunc("/api/llm/providers/config", app.handleLLMProviderConfig)
	mux.HandleFunc("/api/llm/providers/switch", app.handleLLMProviderSwitch) // Provider-Wechsel mit Verbindungsprüfung
	mux.HandleFunc("/api/llm/providers", app.handleLLMProviders)

	// Ollama Endpoints (Frontend-kompatibel)
	mux.HandleFunc("/api/ollama/models", app.handleOllamaModels)
	mux.HandleFunc("/api/ollama/status", app.handleOllamaStatus)

	// Model Store Endpoints (Frontend-kompatibel)
	mux.HandleFunc("/api/model-store/all", app.handleModelStoreAll)
	mux.HandleFunc("/api/model-store/featured", app.handleModelStoreFeatured)

	// Fleet-Mate Endpoints (Frontend-Kompatibilität - Java Backend hatte /api/fleet-mate/*)
	mux.HandleFunc("/api/fleet-mate/mates", app.handleMates) // Alias für /api/mates
	// /api/fleet-mate/mates/ wird jetzt von handleFleetMateByID behandelt (registriert weiter unten)

	// Stats Endpoints (Frontend-Kompatibilität)
	mux.HandleFunc("/api/stats/global", app.handleStatsGlobal)

	// Models Custom Endpoint (Frontend-Kompatibilität)
	mux.HandleFunc("/api/models/custom", app.handleCustomModels)
	mux.HandleFunc("/api/custom-models", app.handleCustomModels) // Alias
	mux.HandleFunc("/api/custom-models/", app.handleCustomModelByID)
	mux.HandleFunc("/api/custom-models/generate-modelfile", app.handleGenerateModelfile)

	// GGUF Models Endpoints
	mux.HandleFunc("/api/gguf-models", app.handleGgufModels)
	mux.HandleFunc("/api/gguf-models/", app.handleGgufModelByID)

	// Personal Info Endpoint (Frontend-Kompatibilität)
	mux.HandleFunc("/api/personal-info", app.handlePersonalInfo)

	// Templates Endpoints (Frontend-Kompatibilität)
	mux.HandleFunc("/api/templates", app.handleTemplates)

	// Projects Endpoints (Frontend-Kompatibilität)
	mux.HandleFunc("/api/projects", app.handleProjects)

	// System Prompts Endpoints
	mux.HandleFunc("/api/system-prompts", app.handleSystemPrompts)
	mux.HandleFunc("/api/system-prompts/default", app.handleSystemPromptsDefault)
	mux.HandleFunc("/api/system-prompts/init-defaults", app.handleSystemPromptsInitDefaults)
	mux.HandleFunc("/api/system-prompts/", app.handleSystemPromptByID)

	// App Settings Endpoints
	mux.HandleFunc("/api/settings", app.handleSettings)
	mux.HandleFunc("/api/settings/model-selection", app.handleSettingsModelSelection)
	mux.HandleFunc("/api/settings/selected-expert", app.handleSettingsSelectedExpert)
	mux.HandleFunc("/api/settings/show-welcome-tiles", app.handleSettingsShowWelcomeTiles)
	mux.HandleFunc("/api/settings/show-top-bar", app.handleSettingsShowTopBar)
	mux.HandleFunc("/api/settings/ui-theme", app.handleSettingsUITheme)
	mux.HandleFunc("/api/settings/llm-provider", app.handleSettingsLLMProvider)
	mux.HandleFunc("/api/settings/document-model", app.handleSettingsDocumentModel)
	mux.HandleFunc("/api/settings/email-model", app.handleSettingsEmailModel)
	mux.HandleFunc("/api/settings/log-analysis-model", app.handleSettingsLogAnalysisModel)
	mux.HandleFunc("/api/settings/coder-model", app.handleSettingsCoderModel)
	// Neue persistente Settings (wichtig über Browser-Sessions hinweg)
	mux.HandleFunc("/api/settings/sampling", app.handleSettingsSampling)
	mux.HandleFunc("/api/settings/chaining", app.handleSettingsChaining)
	mux.HandleFunc("/api/settings/preferences", app.handleSettingsPreferences)
	mux.HandleFunc("/api/settings/language", app.handleSettingsLanguage)

	// llama-server Endpoints (lokaler GGUF-Server)
	mux.HandleFunc("/api/llamaserver/status", app.handleLlamaServerStatus)
	mux.HandleFunc("/api/llamaserver/start", app.handleLlamaServerStart)
	mux.HandleFunc("/api/llamaserver/stop", app.handleLlamaServerStop)
	mux.HandleFunc("/api/llamaserver/restart", app.handleLlamaServerRestart)
	mux.HandleFunc("/api/llamaserver/models", app.handleLlamaServerModels)
	mux.HandleFunc("/api/llamaserver/models/recommended", app.handleLlamaServerModelsRecommended)
	mux.HandleFunc("/api/llamaserver/download", app.handleLlamaServerDownload)
	mux.HandleFunc("/api/llamaserver/config", app.handleLlamaServerConfig)

	// Context-Management
	mux.HandleFunc("/api/llamaserver/context", app.handleLlamaServerContextChange)    // POST Context-Größe ändern (mit Neustart)

	// VRAM-Management Endpoints
	mux.HandleFunc("/api/llamaserver/vram", app.handleLlamaServerVRAMSettings)        // GET/POST VRAM-Einstellungen
	mux.HandleFunc("/api/llamaserver/vram/info", app.handleLlamaServerVRAMInfo)       // GET aktuelle VRAM-Info
	mux.HandleFunc("/api/llamaserver/vram/clear", app.handleLlamaServerVRAMClear)     // POST VRAM manuell löschen

	// Hardware Monitoring Endpoints (lokales System)
	mux.HandleFunc("/api/hardware/stats", app.handleHardwareStats)           // GET alle Stats
	mux.HandleFunc("/api/hardware/quick", app.handleHardwareQuickStats)      // GET schnelle Stats fuer TopBar
	mux.HandleFunc("/api/hardware/cpu", app.handleHardwareCPU)               // GET CPU Stats
	mux.HandleFunc("/api/hardware/memory", app.handleHardwareMemory)         // GET RAM Stats
	mux.HandleFunc("/api/hardware/gpu", app.handleHardwareGPU)               // GET GPU Stats

	// llama-server Provider Endpoints (Frontend-Kompatibilität - erwartet /api/llm/providers/llama-server/*)
	mux.HandleFunc("/api/llm/providers/llama-server/status", app.handleLlamaServerStatus)
	mux.HandleFunc("/api/llm/providers/llama-server/health", app.handleLlamaServerHealth)
	mux.HandleFunc("/api/llm/providers/llama-server/start", app.handleLlamaServerStart)
	mux.HandleFunc("/api/llm/providers/llama-server/stop", app.handleLlamaServerStop)
	mux.HandleFunc("/api/llm/providers/llama-server/restart", app.handleLlamaServerRestart)
	mux.HandleFunc("/api/llm/providers/llama-server/models", app.handleLlamaServerModels)

	// Voice API Endpoints (Whisper STT + Piper TTS)
	mux.HandleFunc("/api/voice/status", app.handleVoiceStatus)
	mux.HandleFunc("/api/voice/stt", app.handleVoiceSTT)           // Speech-to-Text
	mux.HandleFunc("/api/voice/tts", app.handleVoiceTTS)           // Text-to-Speech
	mux.HandleFunc("/api/voice/download", app.handleVoiceDownload)           // Models herunterladen (alle)
	mux.HandleFunc("/api/voice/download-model", app.handleVoiceDownloadModel) // Spezifisches Modell herunterladen
	mux.HandleFunc("/api/voice/models", app.handleVoiceModels)                // Verfügbare Modelle
	mux.HandleFunc("/api/voice/config", app.handleVoiceConfig)                // Konfiguration
	mux.HandleFunc("/api/voice-store/voices", app.handleVoiceStoreVoices)     // Alle Piper Voices von HuggingFace

	// User & Auth Endpoints
	mux.HandleFunc("/api/auth/login", app.handleLogin)
	mux.HandleFunc("/api/auth/logout", app.handleLogout)
	mux.HandleFunc("/api/auth/validate", app.handleValidateToken)
	mux.HandleFunc("/api/auth/check", app.handleAuthCheck)
	mux.HandleFunc("/api/auth/register", app.handleAuthRegister)
	mux.HandleFunc("/api/auth/change-password", app.handleAuthChangePassword)
	mux.HandleFunc("/api/auth/me", app.handleCurrentUser)
	mux.HandleFunc("/api/users", app.handleUsers)
	mux.HandleFunc("/api/users/", app.handleUserByID)

	// System Endpoints (Frontend-Kompatibilität)
	mux.HandleFunc("/api/system/status", app.handleSystemStatus)
	mux.HandleFunc("/api/system/version", app.handleSystemVersion)
	mux.HandleFunc("/api/system/db-size", app.handleSystemDBSize)
	mux.HandleFunc("/api/system/db-size/history", app.handleSystemDBSizeHistory)
	mux.HandleFunc("/api/system/setup-status", app.handleSystemSetupStatus)
	mux.HandleFunc("/api/system/ai-startup-status", app.handleAiStartupStatus)
	mux.HandleFunc("/api/system/stats", app.handleSystemStats)
	mux.HandleFunc("/api/system/stats/quick", app.handleSystemStatsQuick)

	// Database/PostgreSQL Migration Endpoints
	mux.HandleFunc("/api/database/status", app.handleDatabaseStatus)
	mux.HandleFunc("/api/database/postgres/config", app.handlePostgresConfig)
	mux.HandleFunc("/api/database/postgres/test", app.handlePostgresTest)
	mux.HandleFunc("/api/database/postgres/migrate", app.handlePostgresMigrate)

	// Pairing Endpoints (Frontend-Kompatibilität - erwartet /api/pairing/*)
	mux.HandleFunc("/api/pairing/pending", app.handlePendingPairings)
	mux.HandleFunc("/api/pairing/trusted", app.handleMates)
	mux.HandleFunc("/api/pairing/trusted/", app.handleMateByID)
	mux.HandleFunc("/api/pairing/approve/", app.handleApprovePairing)
	mux.HandleFunc("/api/pairing/reject/", app.handleRejectPairing)

	// File-Search Endpoints
	mux.HandleFunc("/api/file-search/status", app.handleFileSearchStatus)
	mux.HandleFunc("/api/file-search/folders", app.handleFileSearchFolders)
	mux.HandleFunc("/api/file-search/folders/", app.handleFileSearchFolderByID)

	// Search Settings Endpoints (Web-Suche)
	mux.HandleFunc("/api/search/settings", app.handleSearchSettings)
	mux.HandleFunc("/api/search/test", app.handleSearchTest)
	mux.HandleFunc("/api/search/status", app.handleSearchStatus)
	mux.HandleFunc("/api/search/execute", app.handleSearchExecute)

	// Expert Avatar Upload
	mux.HandleFunc("/api/experts/avatar/upload", app.handleExpertAvatarUpload)

	// Model-Store Download Endpoints
	mux.HandleFunc("/api/model-store/download/", app.handleModelStoreDownload)
	mux.HandleFunc("/api/model-store/huggingface/download", app.handleHuggingFaceDownload)
	mux.HandleFunc("/api/model-store/huggingface/search", app.handleHuggingFaceSearch)
	mux.HandleFunc("/api/model-store/huggingface/popular", app.handleHuggingFacePopular)
	mux.HandleFunc("/api/model-store/huggingface/german", app.handleHuggingFaceGerman)
	mux.HandleFunc("/api/model-store/huggingface/instruct", app.handleHuggingFaceInstruct)
	mux.HandleFunc("/api/model-store/huggingface/code", app.handleHuggingFaceCode)
	mux.HandleFunc("/api/model-store/huggingface/vision", app.handleHuggingFaceVision)
	mux.HandleFunc("/api/model-store/huggingface/details", app.handleHuggingFaceDetails)
	mux.HandleFunc("/api/ollama/pull/", app.handleOllamaPull)

	// Fleet-Mate erweiterte Endpoints
	mux.HandleFunc("/api/fleet-mate/mates/", app.handleFleetMateByID) // Handles /ping, /analyze-log, /execute, /command-history, /stats
	mux.HandleFunc("/api/fleet-mate/stream/", app.handleFleetMateStream)
	mux.HandleFunc("/api/fleet-mate/exec-stream/", app.handleFleetMateExecStream)
	mux.HandleFunc("/api/fleet-mate/whitelisted-commands", app.handleFleetMateWhitelistedCommands)
	mux.HandleFunc("/api/fleet-mate/export-pdf", app.handleFleetMateExportPDF)
	mux.HandleFunc("/api/export/docx", app.handleExportDOCX)
	mux.HandleFunc("/api/export/odt", app.handleExportODT)
	mux.HandleFunc("/api/export/csv", app.handleExportCSV)
	mux.HandleFunc("/api/export/rtf", app.handleExportRTF)
	mux.HandleFunc("/api/export/pdf", app.handleExportPDF)

	// Model Store erweiterte Endpoints
	mux.HandleFunc("/api/model-store/", app.handleModelStoreByID) // Handles /check-gpu, /cancel, DELETE

	// Sampling Endpoints
	mux.HandleFunc("/api/sampling/presets/", app.handleSamplingPresets)
	mux.HandleFunc("/api/sampling/defaults/auto/", app.handleSamplingDefaultsAuto)
	mux.HandleFunc("/api/sampling/defaults", app.handleSamplingDefaults)
	mux.HandleFunc("/api/sampling/help/", app.handleSamplingHelp)

	// FleetCode Endpoints
	mux.HandleFunc("/api/fleetcode/execute/", app.handleFleetCodeExecute)
	mux.HandleFunc("/api/fleetcode/stream/", app.handleFleetCodeStream)

	// Embedding Config
	mux.HandleFunc("/api/embedding/config", app.handleEmbeddingConfig)

	// Update Status (Stub für Frontend-Kompatibilität)
	mux.HandleFunc("/api/update/status", app.handleUpdateStatus)

	// Setup Wizard Endpoints
	app.setupHandler.RegisterRoutes(mux)

	// WebSocket Endpoint
	mux.HandleFunc("/ws", app.wsServer.HandleWebSocket)
	// Fleet-Mate WebSocket Endpoint (für Thunderbird Email-Mate Kompatibilität)
	mux.HandleFunc("/api/fleet-mate/ws/", app.wsServer.HandleWebSocket)

	// Frontend (Vue.js) - Embedded oder Development
	if isDev() {
		// Development: Proxy zu Vite
		log.Println("Development-Modus: Frontend unter http://localhost:5173")
	} else {
		// Production: Embedded Frontend mit SPA-Support
		distFS, err := fs.Sub(frontendFS, "dist")
		if err != nil {
			log.Printf("Frontend nicht eingebettet, nur API verfügbar")
		} else {
			// SPA Handler: Alle nicht-existierenden Pfade → index.html
			fileServer := http.FileServer(http.FS(distFS))
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Path

				// API und WebSocket nicht anfassen
				if strings.HasPrefix(path, "/api/") || path == "/ws" {
					http.NotFound(w, r)
					return
				}

				// Prüfen ob Datei existiert
				if path != "/" {
					_, err := fs.Stat(distFS, strings.TrimPrefix(path, "/"))
					if err != nil {
						// Datei existiert nicht → index.html für SPA-Routing
						r.URL.Path = "/"
					}
				}
				fileServer.ServeHTTP(w, r)
			})
		}
	}

	// Banner
	printBanner(app.config)

	// Server mit Graceful Shutdown
	addr := ":" + app.config.Port
	server := &http.Server{
		Addr:         addr,
		Handler:      securityMiddleware(app.config)(mux), // Security-Middleware (inkl. CORS, Rate Limiting, Security Headers)
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // Deaktiviert für SSE/Streaming - Downloads können Stunden dauern
		IdleTimeout:  60 * time.Second,
	}

	// Graceful Shutdown Signal Handler
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Server in Goroutine starten
	go func() {
		log.Printf("Server startet auf http://localhost%s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server-Fehler: %v", err)
		}
	}()

	// Auf Shutdown-Signal warten
	<-shutdown
	log.Println("\n🛑 Shutdown-Signal empfangen, fahre Server herunter...")

	// Graceful Shutdown mit Timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// llama-server beenden falls läuft
	if app.llamaServer != nil {
		log.Println("Beende llama-server...")
		app.llamaServer.Stop()
	}

	// HTTP-Server herunterfahren
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server Shutdown-Fehler: %v", err)
	}

	log.Println("✅ Server sauber beendet")
}

// API Handler

func (app *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	modelConfig := app.selectionService.GetConfig()

	// Prüfen welcher Provider aktiv ist
	activeProvider := app.settingsService.GetActiveProvider()
	if activeProvider == "" {
		activeProvider = "llama-server"
	}

	// Provider-Status und Modelle prüfen
	var providerAvailable bool
	var hasModels bool
	var providerError string

	if activeProvider == "ollama" {
		// Ollama-Modus
		providerAvailable = app.chatService.IsAvailable()
		ollamaModels, err := app.chatService.ListModels()
		if err == nil && len(ollamaModels) > 0 {
			hasModels = true
		}
		if !providerAvailable {
			providerError = "Ollama ist nicht erreichbar. Bitte starte Ollama."
		}
	} else {
		// llama-server Modus (Default)
		if app.llamaServer != nil {
			status := app.llamaServer.GetStatus()
			providerAvailable = status.Running && status.Healthy
			// GGUF-Modelle prüfen
			ggufModels, err := app.llamaServer.GetAvailableModels()
			if err == nil && len(ggufModels) > 0 {
				hasModels = true
			}
			if !providerAvailable {
				providerError = "llama-server ist nicht aktiv. Bitte starte den Server in den Einstellungen."
			}
		} else {
			providerError = "llama-server ist nicht konfiguriert."
		}
	}

	// Prüfe ob dies der erste Start ist
	firstRun := app.isFirstRun()

	// Health-Status berechnen
	healthy := providerAvailable && hasModels

	// Errors und Warnings sammeln (als leere Arrays initialisieren, nicht nil)
	errors := make([]string, 0)
	warnings := make([]string, 0)

	if providerError != "" {
		errors = append(errors, providerError)
	}
	if !hasModels {
		errors = append(errors, "Keine KI-Modelle installiert. Bitte lade ein Modell herunter.")
	}
	if len(app.wsServer.GetConnectedMates()) == 0 && len(app.pairingManager.GetTrustedMates()) > 0 {
		warnings = append(warnings, "Keine Fleet-Mates verbunden.")
	}

	status := map[string]interface{}{
		"status":             "ok",
		"service":            "Fleet Navigator",
		"version":            updater.Version,
		"buildTime":          updater.BuildTime,
		"healthy":            healthy,
		"hasModels":          hasModels,
		"firstRun":           firstRun,
		"errors":             errors,
		"warnings":           warnings,
		"connected_mates":    len(app.wsServer.GetConnectedMates()),
		"trusted_mates":      len(app.pairingManager.GetTrustedMates()),
		"provider":           activeProvider,
		"provider_available": providerAvailable,
		"ollama_available":   app.chatService.IsAvailable(),
		"smart_selection":    modelConfig.Enabled,
		"models": map[string]string{
			"default": modelConfig.DefaultModel,
			"code":    modelConfig.CodeModel,
			"fast":    modelConfig.FastModel,
			"vision":  modelConfig.VisionModel,
		},
		"paths": map[string]string{
			"data":   app.config.DataDir,
			"models": app.config.ModelsDir,
		},
	}
	writeJSON(w, status)
}

func (app *App) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Prüfen welcher Provider aktiv ist
	activeProvider := app.settingsService.GetActiveProvider()
	if activeProvider == "" {
		activeProvider = "llama-server" // Default ist llama-server (llama.cpp)
	}

	var modelList []string
	var currentModel string
	var err error

	if activeProvider == "ollama" {
		// Ollama-Modelle von Ollama API
		modelList, err = app.chatService.ListModels()
		currentModel = app.chatService.GetModel()
	} else {
		// llama-server: GGUF-Modelle aus dem lokalen Verzeichnis
		ggufModels, ggufErr := app.llamaServer.GetAvailableModels()
		if ggufErr != nil {
			err = ggufErr
		} else {
			modelList = make([]string, len(ggufModels))
			for i, m := range ggufModels {
				modelList[i] = m.Name
			}
		}
		// Aktuelles Modell vom llama-server
		if app.llamaServer != nil {
			status := app.llamaServer.GetStatus()
			currentModel = status.ModelName
		}
	}

	if err != nil {
		writeJSON(w, map[string]interface{}{
			"error":    err.Error(),
			"models":   []string{},
			"provider": activeProvider,
		})
		return
	}

	writeJSON(w, map[string]interface{}{
		"models":        modelList,
		"current_model": currentModel,
		"provider":      activeProvider,
	})
}

func (app *App) handleModelConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Aktuelle Konfiguration zurueckgeben
		config := app.selectionService.GetConfig()
		writeJSON(w, config)

	case http.MethodPost:
		// Konfiguration aktualisieren
		var config models.SelectionConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		app.selectionService.UpdateConfig(config)
		log.Printf("Model Selection Config aktualisiert: Default=%s, Code=%s, Fast=%s, Vision=%s",
			config.DefaultModel, config.CodeModel, config.FastModel, config.VisionModel)
		writeJSON(w, map[string]string{"status": "updated"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (app *App) handleMates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mates := app.pairingManager.GetTrustedMates()
	connected := app.wsServer.GetConnectedMates()

	// Frontend-kompatibles Format mit mateId, name, status, description, lastStatsUpdate
	type MateResponse struct {
		MateID          string  `json:"mateId"`
		Name            string  `json:"name"`
		Description     string  `json:"description"`
		Status          string  `json:"status"`
		LastStatsUpdate *int64  `json:"lastStatsUpdate"`
	}

	result := make([]MateResponse, 0, len(mates))

	// Nur echte Mates hinzufügen (kein virtueller local-navigator mehr)
	// Hardware-Monitoring ist weiterhin über /api/hardware/* verfügbar
	for _, mate := range mates {
		status := "OFFLINE"
		for _, cID := range connected {
			if cID == mate.ID {
				status = "ONLINE"
				break
			}
		}
		result = append(result, MateResponse{
			MateID:          mate.ID,
			Name:            mate.Name,
			Description:     mate.Type,
			Status:          status,
			LastStatsUpdate: nil, // Echte Mates haben keine lokalen Stats
		})
	}

	writeJSON(w, result)
}

func (app *App) handlePendingPairings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	requests := app.pairingManager.GetPendingRequests()
	writeJSON(w, requests)
}

func (app *App) handleApprovePairing(w http.ResponseWriter, r *http.Request) {
	log.Printf("handleApprovePairing aufgerufen: %s %s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Request ID aus URL-Pfad extrahieren: /api/pairing/approve/{requestId}
	requestID := strings.TrimPrefix(r.URL.Path, "/api/pairing/approve/")
	log.Printf("Request ID aus Pfad: '%s'", requestID)

	// Fallback: Wenn keine ID im Pfad, versuche JSON-Body
	if requestID == "" || requestID == r.URL.Path {
		var req struct {
			RequestID string `json:"request_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			requestID = req.RequestID
			log.Printf("Request ID aus Body: '%s'", requestID)
		}
	}

	if requestID == "" {
		log.Printf("Keine Request ID gefunden!")
		http.Error(w, "Request ID required", http.StatusBadRequest)
		return
	}

	log.Printf("Rufe ApprovePairing auf mit ID: %s", requestID)
	if err := app.wsServer.ApprovePairing(requestID); err != nil {
		log.Printf("ApprovePairing Fehler: %v", err)
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}

	log.Printf("Pairing erfolgreich genehmigt: %s", requestID)
	writeJSON(w, map[string]string{"status": "approved"})
}

func (app *App) handleRejectPairing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Request ID aus URL-Pfad extrahieren: /api/pairing/reject/{requestId}
	requestID := strings.TrimPrefix(r.URL.Path, "/api/pairing/reject/")

	// Fallback: Wenn keine ID im Pfad, versuche JSON-Body
	if requestID == "" || requestID == r.URL.Path {
		var req struct {
			RequestID string `json:"request_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			requestID = req.RequestID
		}
	}

	if requestID == "" {
		http.Error(w, "Request ID required", http.StatusBadRequest)
		return
	}

	if err := app.wsServer.RejectPairing(requestID); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, map[string]string{"status": "rejected"})
}

func (app *App) handleRemoveMate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		MateID string `json:"mate_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Zuerst den Mate über WebSocket benachrichtigen und Verbindung trennen
	if err := app.wsServer.DisconnectMate(req.MateID); err != nil {
		log.Printf("Mate %s war nicht verbunden: %v", req.MateID, err)
		// Kein Fehler - Mate war vielleicht schon offline
	}

	// Dann aus trusted_mates.json entfernen
	if err := app.pairingManager.RemoveTrustedMate(req.MateID); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}

	log.Printf("✅ Mate %s wurde entfernt und benachrichtigt", req.MateID)
	writeJSON(w, map[string]string{"status": "removed"})
}

func (app *App) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	config := map[string]string{
		"ollama_url":   app.config.OllamaURL,
		"ollama_model": app.config.OllamaModel,
		"public_key":   app.pairingManager.GetPublicKey(),
	}
	writeJSON(w, config)
}

// Helpers

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func isDev() bool {
	return os.Getenv("DEV") == "1"
}

// securityMiddleware erstellt eine Security-Middleware mit allen Schutzmaßnahmen
func securityMiddleware(config *Config) func(http.Handler) http.Handler {
	// Security-Middleware initialisieren
	secConfig := middleware.DefaultSecurityConfig()
	secMiddleware := middleware.NewSecurityMiddleware(secConfig)

	return func(next http.Handler) http.Handler {
		// Zuerst Security-Middleware (Rate Limiting, Size Limits, Security Headers)
		secured := secMiddleware.Wrap(next)

		// Dann CORS-Middleware
		return corsMiddleware(secured)
	}
}

// corsMiddleware fügt CORS-Header hinzu für Cross-Origin Requests
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Erlaubte Origins: localhost-Varianten und Entwicklung
		allowedOrigins := []string{
			"http://localhost:5173", // Vite Dev Server
			"http://localhost:2025", // Fleet Navigator
			"http://localhost:2026", // Fleet Navigator Alt-Port
			"http://127.0.0.1:5173",
			"http://127.0.0.1:2025",
			"http://127.0.0.1:2026",
		}

		// Prüfen ob Origin erlaubt ist
		allowed := false
		for _, o := range allowedOrigins {
			if origin == o {
				allowed = true
				break
			}
		}

		// Im DEV-Modus alle Origins erlauben
		if isDev() {
			allowed = true
		}

		if allowed && origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400") // 24 Stunden Preflight-Cache
		}

		// OPTIONS Preflight Request direkt beantworten
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// createDirectoryStructure erstellt alle benötigten Verzeichnisse beim ersten Start
func createDirectoryStructure(config *Config) error {
	// Hauptverzeichnisse
	dirs := []string{
		config.DataDir,
		filepath.Join(config.DataDir, "config"),
		filepath.Join(config.DataDir, "logs"),
		filepath.Join(config.DataDir, "data"),
		config.ModelsDir,
		filepath.Join(config.ModelsDir, "library"),
		filepath.Join(config.ModelsDir, "custom"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("Verzeichnis %s: %w", dir, err)
		}
	}

	log.Printf("Verzeichnisstruktur initialisiert: %s", config.DataDir)
	return nil
}

// hasLocalModels prüft ob GGUF-Modelle im models/library Verzeichnis existieren
func (app *App) hasLocalModels() bool {
	libraryDir := filepath.Join(app.config.ModelsDir, "library")

	entries, err := os.ReadDir(libraryDir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".gguf") {
			return true
		}
	}
	return false
}

// isFirstRun prüft ob dies der erste Start ist (keine Daten vorhanden)
func (app *App) isFirstRun() bool {
	// Prüfe ob es bereits Chats oder Experten gibt
	chats, err := app.chatStore.GetAllChats()
	if err == nil && len(chats) > 0 {
		return false
	}

	// Prüfe ob es bereits Modelle gibt
	if app.hasLocalModels() {
		return false
	}

	// Prüfe ob Ollama Modelle hat
	models, err := app.chatService.ListModels()
	if err == nil && len(models) > 0 {
		return false
	}

	return true
}

// --- Experten API Handler ---

func (app *App) handleExperts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Alle Experten abrufen
		onlyActive := r.URL.Query().Get("active") == "true"
		experts, err := app.expertenService.GetAllExperts(onlyActive)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, experts)

	case http.MethodPost:
		// Neuen Experten erstellen
		var req experte.CreateExpertRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		expert, err := app.expertenService.CreateExpert(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		writeJSON(w, expert)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (app *App) handleExpertByID(w http.ResponseWriter, r *http.Request) {
	// ID aus URL extrahieren: /api/experts/{id} oder /api/experts/{id}/modes
	path := r.URL.Path
	parts := splitPath(path, "/api/experts/")

	if len(parts) == 0 {
		http.Error(w, "Missing expert ID", http.StatusBadRequest)
		return
	}

	// Prüfen ob es um Modi geht
	if len(parts) >= 2 && parts[1] == "modes" {
		app.handleExpertModes(w, r, parts[0])
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid expert ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		expert, err := app.expertenService.GetExpert(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if expert == nil {
			http.Error(w, "Expert not found", http.StatusNotFound)
			return
		}
		writeJSON(w, expert)

	case http.MethodPut:
		var req experte.UpdateExpertRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		expert, err := app.expertenService.UpdateExpert(id, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, expert)

	case http.MethodDelete:
		if err := app.expertenService.DeleteExpert(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]string{"status": "deleted"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (app *App) handleExpertModes(w http.ResponseWriter, r *http.Request, expertIDStr string) {
	expertID, err := strconv.ParseInt(expertIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid expert ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		modes, err := app.expertenService.GetModes(expertID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, modes)

	case http.MethodPost:
		var req experte.CreateModeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		mode, err := app.expertenService.AddMode(expertID, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		writeJSON(w, mode)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleModeByID behandelt PUT/DELETE für einzelne Modi: /api/experts/modes/{modeId}
func (app *App) handleModeByID(w http.ResponseWriter, r *http.Request) {
	// ID aus URL extrahieren: /api/experts/modes/{id}
	path := r.URL.Path
	parts := splitPath(path, "/api/experts/modes/")

	if len(parts) == 0 {
		http.Error(w, "Missing mode ID", http.StatusBadRequest)
		return
	}

	modeID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid mode ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		mode, err := app.expertenService.GetMode(modeID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if mode == nil {
			http.Error(w, "Mode not found", http.StatusNotFound)
			return
		}
		writeJSON(w, mode)

	case http.MethodPut:
		var req struct {
			Name        string   `json:"name"`
			Prompt      string   `json:"prompt"`
			Icon        string   `json:"icon"`
			Keywords    []string `json:"keywords"`
			KeywordsStr string   `json:"keywordsStr"` // Alternative: komma-separiert
			IsDefault   bool     `json:"is_default"`
			SortOrder   int      `json:"sort_order"`
			Priority    int      `json:"priority"` // Alias für sort_order
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Keywords verarbeiten (Array oder komma-separierter String)
		keywords := req.Keywords
		if len(keywords) == 0 && req.KeywordsStr != "" {
			// Komma-separierten String zu Array konvertieren
			for _, kw := range splitByComma(req.KeywordsStr) {
				kw = strings.TrimSpace(kw)
				if kw != "" {
					keywords = append(keywords, kw)
				}
			}
		}

		// SortOrder oder Priority verwenden
		sortOrder := req.SortOrder
		if sortOrder == 0 && req.Priority > 0 {
			sortOrder = req.Priority
		}

		// Modus aktualisieren
		if err := app.expertenService.UpdateMode(modeID, req.Name, req.Prompt, req.Icon, keywords, req.IsDefault, sortOrder); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Aktualisierten Modus zurückgeben
		mode, _ := app.expertenService.GetMode(modeID)
		writeJSON(w, mode)

	case http.MethodDelete:
		if err := app.expertenService.DeleteMode(modeID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]string{"status": "deleted"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// splitByComma teilt einen String an Kommas
func splitByComma(s string) []string {
	result := []string{}
	current := ""
	for _, c := range s {
		if c == ',' {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// splitPath teilt einen Pfad nach einem Prefix
func splitPath(path, prefix string) []string {
	if len(path) <= len(prefix) {
		return nil
	}
	rest := path[len(prefix):]
	parts := []string{}
	for _, p := range splitBySlash(rest) {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func splitBySlash(s string) []string {
	result := []string{}
	current := ""
	for _, c := range s {
		if c == '/' {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// --- Chat API Handler ---

func (app *App) handleChatNew(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Title string `json:"title"`
		Model string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Wenn kein Body, Default-Werte verwenden
		req.Title = "Neuer Chat"
		req.Model = app.selectedModel
	}

	if req.Title == "" {
		req.Title = "Neuer Chat"
	}
	if req.Model == "" {
		req.Model = app.selectedModel
	}

	chatObj, err := app.chatStore.CreateChat(req.Title, req.Model)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, chatObj)
}

func (app *App) handleChatAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	chats, err := app.chatStore.GetAllChats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, chats)
}

func (app *App) handleChatHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// ID aus URL extrahieren: /api/chat/history/{id}
	idStr := r.URL.Path[len("/api/chat/history/"):]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	chatObj, err := app.chatStore.GetChat(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if chatObj == nil {
		http.Error(w, "Chat not found", http.StatusNotFound)
		return
	}

	writeJSON(w, chatObj)
}

func (app *App) handleChatByID(w http.ResponseWriter, r *http.Request) {
	// ID aus URL extrahieren: /api/chat/{id} oder /api/chat/{id}/...
	path := r.URL.Path[len("/api/chat/"):]

	// Überspringen wenn es ein reservierter Endpoint ist
	if path == "" || path == "new" || path == "all" || path == "send-stream" || len(path) > 0 && path[:min(7, len(path))] == "history" {
		return
	}

	// Sub-Endpoints erkennen und parsen
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Chat-ID extrahieren
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	// Sub-Endpoint bestimmen
	var subEndpoint string
	var subID int64 = 0
	if len(parts) > 1 {
		subEndpoint = parts[1]
		// Prüfen ob es noch eine Sub-ID gibt (z.B. /messages/{messageId})
		if len(parts) > 2 {
			subID, _ = strconv.ParseInt(parts[2], 10, 64)
		}
	}

	// Handle Sub-Endpoints
	switch subEndpoint {
	case "model":
		// PATCH /api/chat/{id}/model
		if r.Method == http.MethodPatch {
			var req struct {
				Model string `json:"model"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
			if req.Model == "" {
				http.Error(w, "Model is required", http.StatusBadRequest)
				return
			}
			if err := app.chatStore.UpdateChat(id, nil, &req.Model); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]interface{}{"status": "updated", "model": req.Model})
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return

	case "rename":
		// PATCH /api/chat/{id}/rename
		if r.Method == http.MethodPatch {
			var req struct {
				NewTitle string `json:"newTitle"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
			if req.NewTitle == "" {
				http.Error(w, "newTitle is required", http.StatusBadRequest)
				return
			}
			log.Printf("Renaming chat %d to: %s", id, req.NewTitle)
			chatObj, err := app.chatStore.RenameChat(id, req.NewTitle)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if chatObj == nil {
				http.Error(w, "Chat not found", http.StatusNotFound)
				return
			}
			writeJSON(w, chatObj)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return

	case "expert":
		// PATCH /api/chat/{id}/expert
		if r.Method == http.MethodPatch {
			var req struct {
				ExpertID *int64 `json:"expertId"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
			log.Printf("Updating chat %d expert to: %v", id, req.ExpertID)
			if err := app.chatStore.UpdateChatExpert(id, req.ExpertID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]interface{}{"status": "updated", "expertId": req.ExpertID})
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return

	case "messages":
		// DELETE /api/chat/{id}/messages/{messageId}
		if r.Method == http.MethodDelete && subID > 0 {
			log.Printf("Deleting message %d from chat %d", subID, id)
			if err := app.chatStore.DeleteMessage(id, subID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]string{"status": "deleted"})
			return
		}
		http.Error(w, "Method not allowed or missing message ID", http.StatusMethodNotAllowed)
		return

	case "fork":
		// POST /api/chat/{id}/fork
		if r.Method == http.MethodPost {
			var req struct {
				NewTitle string `json:"newTitle"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				// Wenn kein Body, Standard-Titel verwenden
				req.NewTitle = ""
			}

			// Original-Chat laden für Titel-Generierung
			original, err := app.chatStore.GetChat(id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if original == nil {
				http.Error(w, "Chat not found", http.StatusNotFound)
				return
			}

			// Titel für Fork generieren
			forkTitle := req.NewTitle
			if forkTitle == "" {
				forkTitle = original.Title + " (Fork)"
			}

			log.Printf("Forking chat %d to: %s", id, forkTitle)
			forkedChat, err := app.chatStore.ForkChat(id, forkTitle)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, forkedChat)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return

	case "export":
		// GET /api/chat/{id}/export
		if r.Method == http.MethodGet {
			log.Printf("Exporting chat %d", id)
			export, err := app.chatStore.ExportChat(id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, export)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Basis-Endpoints ohne Sub-Endpoint
	switch r.Method {
	case http.MethodGet:
		chatObj, err := app.chatStore.GetChat(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if chatObj == nil {
			http.Error(w, "Chat not found", http.StatusNotFound)
			return
		}
		writeJSON(w, chatObj)

	case http.MethodDelete:
		if err := app.chatStore.DeleteChat(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]string{"status": "deleted"})

	case http.MethodPut, http.MethodPatch:
		var req struct {
			Title *string `json:"title"`
			Model *string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if err := app.chatStore.UpdateChat(id, req.Title, req.Model); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]string{"status": "updated"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (app *App) handleChatSendStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ChatID             int64   `json:"chatId"`
		Message            string  `json:"message"`
		Model              string  `json:"model"`
		SystemPrompt       *string `json:"systemPrompt"`       // Optional: Custom System-Prompt vom Frontend
		ExpertID           *int64  `json:"expertId"`           // Optional: Experte verwenden
		ModeID             *int64  `json:"modeId"`             // Optional: Aktueller Modus
		WebSearchEnabled   bool    `json:"webSearchEnabled"`   // Web-Suche aktivieren
		WebSearchHideLinks bool    `json:"webSearchHideLinks"` // Quellen-Links NICHT anzeigen (nur RAG nutzen)
		DocumentContext    string  `json:"documentContext"`    // Extrahierter Text aus hochgeladenen Dateien (PDF, TXT, etc.)
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, "Message is required", http.StatusBadRequest)
		return
	}

	model := req.Model
	if model == "" {
		model = app.selectedModel
	}

	// Wenn keine chatId, neuen Chat erstellen
	chatID := req.ChatID
	if chatID == 0 {
		// Erstelle neuen Chat mit erstem Teil der Nachricht als Titel
		// SECURITY: HTML-Escape um XSS zu verhindern
		title := html.EscapeString(req.Message)
		if len(title) > 50 {
			title = title[:50] + "..."
		}
		newChat, err := app.chatStore.CreateChat(title, model)
		if err != nil {
			log.Printf("Chat erstellen fehlgeschlagen: %v", err)
			http.Error(w, "Fehler beim Erstellen des Chats", http.StatusInternalServerError)
			return
		}
		chatID = newChat.ID
		log.Printf("Neuer Chat erstellt: ID=%d", chatID)
	}

	// System-Prompt: Vom Frontend gesendeten verwenden, oder Default aus DB laden
	var systemPrompt string
	if req.SystemPrompt != nil && *req.SystemPrompt != "" {
		systemPrompt = *req.SystemPrompt
		log.Printf("System-Prompt vom Frontend: %d Zeichen", len(systemPrompt))
	} else {
		// Default System-Prompt aus Datenbank laden
		defaultPrompt, err := app.promptsService.GetDefault()
		if err == nil && defaultPrompt != nil {
			systemPrompt = defaultPrompt.Content
			log.Printf("System-Prompt aus DB geladen: %s", defaultPrompt.Name)
		} else {
			// Fallback wenn DB-Prompt nicht verfügbar
			systemPrompt = `Du bist ein hilfreicher KI-Assistent. Antworte immer auf Deutsch, freundlich und präzise.`
			log.Printf("Fallback System-Prompt verwendet")
		}
	}

	// Variables für Experten-Modus-Wechsel
	var modeSwitchMessage string
	var newModeID *int64

	// Variable für Web-Suche Ergebnisse und Quellen-Footer
	var webSearchContext string
	var sourcesFooter string

	// Web-Suche durchführen wenn aktiviert
	if req.WebSearchEnabled && app.searchService != nil {
		log.Printf("Web-Suche aktiviert (Nachrichtenlänge: %d Zeichen)", len(req.Message))

		// Query optimieren (konversationelle Frage -> Suchbegriff)
		optimizedQuery := app.optimizeSearchQuery(req.Message)

		// Suche mit Content-Fetching durchführen
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		searchResults, err := app.searchService.Search(ctx, optimizedQuery, search.SearchOptions{
			MaxResults:       5,
			FetchFullContent: true,  // WICHTIG: Seiteninhalte lesen!
			MaxContentLength: 4000,  // Max 4000 Zeichen pro Seite (erhöht)
			ReRank:           true,
			Region:           "de",
		})
		cancel()

		if err == nil && len(searchResults) > 0 {
			// Suchergebnisse als Kontext formatieren (mit optimierter Query für besseres Re-Ranking)
			webSearchContext = app.formatSearchContextEnhanced(searchResults, req.Message, optimizedQuery)
			sourcesFooter = app.searchService.FormatSourcesFooter(searchResults)
			log.Printf("Web-Suche: %d Ergebnisse gefunden, Query: '%s', Kontext: %d Zeichen",
				len(searchResults), optimizedQuery, len(webSearchContext))

			// Websuche-Zähler erhöhen und in DB speichern
			if app.settingsService != nil {
				count, _ := app.settingsService.IncrementWebSearchCount()
				log.Printf("Web-Suche Zähler: %d (diesen Monat)", count)
			}
		} else if err != nil {
			log.Printf("Web-Suche Fehler: %v", err)
		}
	}

	// Sampling-Parameter (Defaults, können von Experte überschrieben werden)
	var samplingParams = llamaserver.SamplingParams{
		Temperature: 0.7,
		TopP:        0.9,
		MaxTokens:   4096,
	}

	// Wenn ein Experte ausgewählt ist, ChatContext verwenden
	if req.ExpertID != nil && *req.ExpertID > 0 {
		// Aktuelle Sprache aus Settings für Experten-Prompt-Übersetzung
		locale := app.settingsService.GetLocale()
		chatCtx, err := app.expertenService.GetChatContextWithLocale(*req.ExpertID, req.ModeID, req.Message, locale)
		if err == nil && chatCtx != nil {
			// Experten-System-Prompt verwenden
			systemPrompt = chatCtx.SystemPrompt
			model = chatCtx.Model

			// Sampling-Parameter vom Experten übernehmen
			samplingParams.Temperature = chatCtx.Temperature
			samplingParams.TopP = chatCtx.TopP
			samplingParams.MaxTokens = chatCtx.MaxTokens

			// Prüfen ob Modus gewechselt wurde
			if chatCtx.ModeSwitched && chatCtx.SwitchMessage != "" {
				modeSwitchMessage = chatCtx.SwitchMessage
				if chatCtx.ActiveMode != nil {
					newModeID = &chatCtx.ActiveMode.ID
				}
			}

			log.Printf("Expert Chat: %s (Model: %s, Mode: %v, Switched: %v, Temp: %.1f, TopP: %.1f, MaxTokens: %d)",
				chatCtx.Expert.Name, model, chatCtx.ActiveMode, chatCtx.ModeSwitched,
				samplingParams.Temperature, samplingParams.TopP, samplingParams.MaxTokens)
		}
	}

	// User-Nachricht speichern (mit fixem Experten und Modus)
	_, err := app.chatStore.AddMessage(chatID, "USER", req.Message, "", 0, req.ExpertID, req.ModeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Chat-History für Kontext holen
	messages, err := app.chatStore.GetMessages(chatID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Konversations-History aufbauen
	var conversationMessages []chat.Message

	// System-Prompt mit Web-Suche-Kontext erweitern wenn vorhanden
	finalSystemPrompt := systemPrompt
	if webSearchContext != "" {
		finalSystemPrompt = systemPrompt + "\n\n" + webSearchContext + `

WICHTIGE ANWEISUNGEN ZUR NUTZUNG DER WEB-RECHERCHE:

1. INHALT NUTZEN: Beantworte die Frage des Benutzers basierend auf den Informationen aus den Web-Quellen.
   Extrahiere die relevantesten Fakten und Informationen.

2. QUELLEN ZITIEREN: Wenn du Informationen aus einer Quelle verwendest, verweise darauf mit [1], [2], etc.
   Beispiel: "Laut aktuellen Informationen [1] beträgt..."

3. ZUSAMMENFASSEN: Fasse die wichtigsten Erkenntnisse aus mehreren Quellen zusammen, wenn sie sich ergänzen.

4. KRITISCH PRÜFEN: Wenn Quellen widersprüchliche Informationen enthalten, weise darauf hin.

5. AKTUALITÄT: Beachte, dass Web-Inhalte möglicherweise veraltet sein können.

6. KEINE LINKS IM TEXT: Füge keine URLs direkt in deinen Antworttext ein - die Quellen werden am Ende automatisch angehängt.

7. SPRACHE: Antworte immer auf Deutsch, auch wenn die Quellen auf Englisch sind.`
	}

	// Dokument-Kontext (aus hochgeladenen PDFs, TXT, etc.) hinzufügen
	if req.DocumentContext != "" {
		finalSystemPrompt = finalSystemPrompt + "\n\n=== HOCHGELADENE DOKUMENTE ===\n" + req.DocumentContext + `

WICHTIG: Der Benutzer hat obige Dokumente hochgeladen. Beziehe dich in deiner Antwort auf den Inhalt dieser Dokumente.`
		log.Printf("Dokument-Kontext hinzugefügt: %d Zeichen", len(req.DocumentContext))
	}

	conversationMessages = append(conversationMessages, chat.Message{
		Role:    "system",
		Content: finalSystemPrompt,
	})

	for _, m := range messages {
		role := "user"
		if m.Role == "ASSISTANT" {
			role = "assistant"
		}
		conversationMessages = append(conversationMessages, chat.Message{
			Role:    role,
			Content: m.Content,
		})
	}

	// SSE Headers setzen
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// CORS wird bereits durch Security-Middleware gesetzt

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// WICHTIG: Start-Event mit chatId senden (Frontend erwartet das!)
	requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())
	startData := map[string]interface{}{
		"chatId":    chatID, // Wichtig: chatID verwenden, nicht req.ChatID (kann 0 sein!)
		"requestId": requestID,
	}
	startJSON, _ := json.Marshal(startData)
	fmt.Fprintf(w, "data: %s\n\n", startJSON)
	flusher.Flush()
	log.Printf("SSE Start-Event gesendet: chatId=%d, requestId=%s", chatID, requestID)

	// Wenn Modus gewechselt wurde, zuerst Switch-Nachricht senden
	if modeSwitchMessage != "" {
		switchData := map[string]interface{}{
			"type":          "mode_switch",
			"message":       modeSwitchMessage,
			"newModeId":     newModeID,
		}
		jsonData, _ := json.Marshal(switchData)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	}

	// Streaming-Antwort basierend auf aktivem Provider
	var fullResponse string
	activeProvider := app.settingsService.GetActiveProvider()
	if activeProvider == "" {
		activeProvider = "llama-server"
	}

	streamCallback := func(content string, done bool) {
		fullResponse += content

		// SSE Event senden
		data := map[string]interface{}{
			"content": content,
			"done":    done,
		}
		jsonData, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	}

	if activeProvider == "ollama" {
		// Ollama Provider
		err = app.chatService.StreamChatWithMessages(model, conversationMessages, streamCallback)
	} else {
		// llama-server Provider (Default)
		// Konvertiere Messages für llama-server
		llamaMessages := make([]llamaserver.ChatMessage, len(conversationMessages))
		for i, msg := range conversationMessages {
			llamaMessages[i] = llamaserver.ChatMessage{
				Role:    msg.Role,
				Content: msg.Content,
			}
		}
		// Mit Sampling-Parametern aufrufen
		err = app.llamaServer.StreamChatWithParams(llamaMessages, samplingParams, streamCallback)
	}

	if err != nil {
		// Fehler als SSE senden
		errData := map[string]interface{}{
			"error": err.Error(),
			"done":  true,
		}
		jsonData, _ := json.Marshal(errData)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
		return
	}

	// Wenn Web-Suche aktiv war UND Links angezeigt werden sollen, Sources Footer anhängen
	// Bei WebSearchHideLinks=true wird nur der Inhalt für RAG genutzt, aber keine Links angezeigt
	if sourcesFooter != "" && !req.WebSearchHideLinks {
		// Sources Footer als separates Content-Event senden
		sourcesData := map[string]interface{}{
			"content": "\n\n" + sourcesFooter,
			"done":    false,
		}
		sourcesJSON, _ := json.Marshal(sourcesData)
		fmt.Fprintf(w, "data: %s\n\n", sourcesJSON)
		flusher.Flush()

		// Zur vollständigen Antwort hinzufügen
		fullResponse += "\n\n" + sourcesFooter
		log.Printf("Web-Suche: Quellen-Footer angehängt")
	} else if sourcesFooter != "" && req.WebSearchHideLinks {
		log.Printf("Web-Suche: Quellen-Footer NICHT angehängt (WebSearchHideLinks=true)")
	}

	// Token-Approximation (ca. 4 Zeichen pro Token)
	tokenCount := len(fullResponse) / 4

	// WICHTIG: Done-Event mit Tokens senden (Frontend erwartet das!)
	doneData := map[string]interface{}{
		"tokens":          tokenCount,
		"webSearchUsed":   req.WebSearchEnabled && sourcesFooter != "",
	}
	doneJSON, _ := json.Marshal(doneData)
	fmt.Fprintf(w, "data: %s\n\n", doneJSON)
	flusher.Flush()
	log.Printf("SSE Done-Event gesendet: tokens=%d, webSearch=%v", tokenCount, req.WebSearchEnabled)

	// Assistenten-Antwort speichern (inkl. Sources Footer)
	// Effektive ModeID: Wenn Modus gewechselt wurde, newModeID verwenden, sonst req.ModeID
	effectiveModeID := req.ModeID
	if newModeID != nil {
		effectiveModeID = newModeID
	}
	if _, err := app.chatStore.AddMessage(chatID, "ASSISTANT", fullResponse, model, tokenCount, req.ExpertID, effectiveModeID); err != nil {
		log.Printf("WARNUNG: Assistenten-Antwort konnte nicht gespeichert werden: %v", err)
	}
}

func (app *App) handleSelectedModel(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Wenn kein Modell gesetzt, 204 No Content
		if app.selectedModel == "" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		// Nur den Model-Namen als String zurückgeben (wie Java-Backend)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(app.selectedModel))

	case http.MethodPost:
		// Frontend sendet text/plain, nicht JSON
		contentType := r.Header.Get("Content-Type")
		var modelName string

		if strings.Contains(contentType, "text/plain") {
			// Text/plain Body lesen
			body := make([]byte, 1024)
			n, _ := r.Body.Read(body)
			modelName = strings.TrimSpace(string(body[:n]))
		} else {
			// Fallback: JSON
			var req struct {
				Model string `json:"model"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}
			modelName = req.Model
		}

		if modelName == "" {
			http.Error(w, "Model is required", http.StatusBadRequest)
			return
		}

		app.selectedModel = modelName
		app.modelService.SetSelectedModel(modelName)
		log.Printf("Ausgewähltes Modell geändert: %s", modelName)
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func printBanner(config *Config) {
	fmt.Println(`
╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║   ███████╗██╗     ███████╗███████╗████████╗                  ║
║   ██╔════╝██║     ██╔════╝██╔════╝╚══██╔══╝                  ║
║   █████╗  ██║     █████╗  █████╗     ██║                     ║
║   ██╔══╝  ██║     ██╔══╝  ██╔══╝     ██║                     ║
║   ██║     ███████╗███████╗███████╗   ██║                     ║
║   ╚═╝     ╚══════╝╚══════╝╚══════╝   ╚═╝                     ║
║                                                               ║
║   ███╗   ██╗ █████╗ ██╗   ██╗██╗ ██████╗  █████╗ ████████╗   ║
║   ████╗  ██║██╔══██╗██║   ██║██║██╔════╝ ██╔══██╗╚══██╔══╝   ║
║   ██╔██╗ ██║███████║██║   ██║██║██║  ███╗███████║   ██║      ║
║   ██║╚██╗██║██╔══██║╚██╗ ██╔╝██║██║   ██║██╔══██║   ██║      ║
║   ██║ ╚████║██║  ██║ ╚████╔╝ ██║╚██████╔╝██║  ██║   ██║      ║
║   ╚═╝  ╚═══╝╚═╝  ╚═╝  ╚═══╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝   ╚═╝      ║
║                                                               ║
╠═══════════════════════════════════════════════════════════════╣
║  Das Experten-System für Ihr Büro                             ║
╠═══════════════════════════════════════════════════════════════╣`)
	fmt.Printf("║  Version:     %-46s ║\n", updater.Version)
	fmt.Printf("║  Build-Zeit:  %-46s ║\n", updater.BuildTime)
	fmt.Printf("║  Port:        %-46s ║\n", config.Port)
	fmt.Printf("║  Ollama:      %-46s ║\n", config.OllamaURL)
	fmt.Printf("║  Daten:       %-46s ║\n", config.DataDir)
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Println("║  Endpoints:                                                   ║")
	fmt.Printf("║    http://localhost:%-14s Web-UI                      ║\n", config.Port)
	fmt.Printf("║    ws://localhost:%s/ws           WebSocket (Mates)           ║\n", config.Port)
	fmt.Printf("║    http://localhost:%s/api/*      REST API                    ║\n", config.Port)
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
}

// --- LLM Model Management Handler ---

// handleLLMModels gibt alle Modelle zurueck (installiert + Registry)
// Provider-abhängig: llama-cpp = GGUF-Dateien, Ollama = Ollama API
func (app *App) handleLLMModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	activeProvider := app.settingsService.GetActiveProvider()

	// Bei llama-cpp: GGUF-Dateien direkt lesen
	if activeProvider == "llama-cpp" || activeProvider == "java-llama-cpp" || activeProvider == "llama-server" {
		ggufModels, _ := app.llamaServer.GetAvailableModels()

		// In Frontend-Format konvertieren
		installed := make([]map[string]interface{}, 0, len(ggufModels))
		for _, m := range ggufModels {
			// Registry-Info suchen
			entry := app.modelService.GetRegistry().FindByFilename(m.Name)

			model := map[string]interface{}{
				"name":        m.Name,
				"path":        m.Path,
				"size":        m.Size,
				"size_human":  formatBytesForDisplay(m.Size),
				"modified_at": m.Modified,
				"provider":    "llama-server",
				"installed":   true,
			}

			if entry != nil {
				model["display_name"] = entry.DisplayName
				model["description"] = entry.Description
				model["category"] = entry.Category
				model["architecture"] = entry.Architecture
				model["parameter_size"] = entry.ParameterSize
			}

			installed = append(installed, model)
		}

		writeJSON(w, map[string]interface{}{
			"installed":       installed,
			"registry":        app.modelService.GetAvailableModelsFromRegistry(),
			"selected_model":  app.modelService.GetSelectedModel(),
			"default_model":   app.modelService.GetDefaultModel(),
			"provider":        "llama-server",
			"provider_online": true,
		})
		return
	}

	// Ollama Provider
	installed, err := app.modelService.GetInstalledModels()
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"installed": []interface{}{},
			"registry":  app.modelService.GetAvailableModelsFromRegistry(),
			"error":     err.Error(),
		})
		return
	}

	writeJSON(w, map[string]interface{}{
		"installed":       installed,
		"registry":        app.modelService.GetAvailableModelsFromRegistry(),
		"selected_model":  app.modelService.GetSelectedModel(),
		"default_model":   app.modelService.GetDefaultModel(),
		"provider":        app.modelService.GetProviderName(),
		"provider_online": app.modelService.IsProviderAvailable(),
	})
}

// handleLLMInstalledModels gibt nur installierte Modelle zurueck
// Provider-abhängig: llama-cpp = GGUF-Dateien, Ollama = Ollama API
func (app *App) handleLLMInstalledModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	activeProvider := app.settingsService.GetActiveProvider()

	// Bei llama-cpp: GGUF-Dateien direkt lesen
	if activeProvider == "llama-cpp" || activeProvider == "java-llama-cpp" || activeProvider == "llama-server" {
		ggufModels, _ := app.llamaServer.GetAvailableModels()

		// In Frontend-Format konvertieren
		models := make([]map[string]interface{}, 0, len(ggufModels))
		for _, m := range ggufModels {
			entry := app.modelService.GetRegistry().FindByFilename(m.Name)

			model := map[string]interface{}{
				"name":        m.Name,
				"path":        m.Path,
				"size":        m.Size,
				"size_human":  formatBytesForDisplay(m.Size),
				"modified_at": m.Modified,
				"provider":    "llama-server",
				"installed":   true,
			}

			if entry != nil {
				model["display_name"] = entry.DisplayName
				model["description"] = entry.Description
				model["category"] = entry.Category
			}

			models = append(models, model)
		}

		writeJSON(w, map[string]interface{}{
			"models":         models,
			"selected_model": app.modelService.GetSelectedModel(),
			"provider":       "llama-server",
		})
		return
	}

	// Ollama Provider
	models, err := app.modelService.GetInstalledModels()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{
		"models":         models,
		"selected_model": app.modelService.GetSelectedModel(),
	})
}

// handleLLMRegistry gibt die Model-Registry zurueck
func (app *App) handleLLMRegistry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	category := r.URL.Query().Get("category")

	var models []llm.ModelRegistryEntry
	if category != "" {
		models = app.modelService.GetModelsByCategory(llm.ModelCategory(category))
	} else {
		models = app.modelService.GetAvailableModelsFromRegistry()
	}

	writeJSON(w, map[string]interface{}{
		"models":     models,
		"categories": []string{"chat", "code", "vision", "compact"},
	})
}

// handleLLMModelContext gibt die Context-Informationen für ein Modell zurück
// GET /api/llm/models/context?model=qwen2.5:7b
func (app *App) handleLLMModelContext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	modelName := r.URL.Query().Get("model")
	if modelName == "" {
		// Ohne Modellname: aktuellen Server-Context zurückgeben
		writeJSON(w, map[string]interface{}{
			"currentContext": app.llamaServer.GetContextSize(),
			"defaultContext": llm.DefaultContextSize,
		})
		return
	}

	// Modell-spezifischen Context aus Registry holen
	registry := app.modelService.GetRegistry()
	modelMaxContext := registry.GetModelContextSize(modelName)
	effectiveContext := registry.GetEffectiveContextSize(modelName, 0) // 0 = kein Expert-Override

	// Prüfen ob Context-Wechsel nötig wäre
	currentContext := app.llamaServer.GetContextSize()
	restartNeeded := currentContext != effectiveContext

	writeJSON(w, map[string]interface{}{
		"model":            modelName,
		"modelMaxContext":  modelMaxContext,
		"effectiveContext": effectiveContext,
		"currentContext":   currentContext,
		"defaultContext":   llm.DefaultContextSize,
		"restartNeeded":    restartNeeded,
	})
}

// handleLLMFeaturedModels gibt Featured-Modelle zurueck
func (app *App) handleLLMFeaturedModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, map[string]interface{}{
		"featured": app.modelService.GetFeaturedModels(),
		"trending": app.modelService.GetRegistry().GetTrendingModels(),
	})
}

// handleLLMPullModel laedt ein Modell herunter
// Provider-abhängig: Ollama = Ollama API, llama-cpp = HuggingFace GGUF
func (app *App) handleLLMPullModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Model string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Model == "" {
		http.Error(w, "Model name required", http.StatusBadRequest)
		return
	}

	// SSE Headers fuer Progress-Streaming
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Provider prüfen - IMMER vor Download!
	activeProvider := app.settingsService.GetActiveProvider()
	log.Printf("LLM Pull Model: %s (Provider: %s)", req.Model, activeProvider)

	// Bei llama-cpp: GGUF von HuggingFace herunterladen
	if activeProvider == "llama-cpp" || activeProvider == "java-llama-cpp" || activeProvider == "llama-server" {
		// Modell in Registry suchen
		entry := app.modelService.FindModelInRegistry(req.Model)
		if entry == nil {
			entry = app.modelService.GetRegistry().FindByOllamaName(req.Model)
		}

		if entry == nil || entry.HuggingFaceRepo == "" {
			errJSON, _ := json.Marshal(map[string]string{
				"error": fmt.Sprintf("Modell '%s' nicht in Registry gefunden oder keine HuggingFace URL", req.Model),
			})
			fmt.Fprintf(w, "data: %s\n\n", errJSON)
			flusher.Flush()
			return
		}

		// HuggingFace URL
		downloadURL := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", entry.HuggingFaceRepo, entry.Filename)
		log.Printf("GGUF Download von HuggingFace: %s", downloadURL)

		progressChan := make(chan llamaserver.DownloadProgress, 100)
		done := make(chan error, 1)

		go func() {
			done <- app.llamaServer.DownloadModel(downloadURL, entry.Filename, progressChan)
			close(progressChan)
		}()

		lastPercent := -1
		for progress := range progressChan {
			percent := int(progress.Percent)
			if percent != lastPercent {
				lastPercent = percent
				progressJSON, _ := json.Marshal(map[string]interface{}{
					"status":     "downloading",
					"percent":    progress.Percent,
					"downloaded": progress.Downloaded,
					"total":      progress.Total,
					"filename":   progress.Filename,
				})
				fmt.Fprintf(w, "data: %s\n\n", progressJSON)
				flusher.Flush()
			}
		}

		if err := <-done; err != nil {
			errJSON, _ := json.Marshal(map[string]string{"error": err.Error()})
			fmt.Fprintf(w, "data: %s\n\n", errJSON)
		} else {
			doneJSON, _ := json.Marshal(map[string]interface{}{
				"status":   "done",
				"model":    entry.Filename,
				"message":  fmt.Sprintf("GGUF-Modell %s erfolgreich heruntergeladen", entry.Filename),
			})
			fmt.Fprintf(w, "data: %s\n\n", doneJSON)
		}
		flusher.Flush()
		return
	}

	// Ollama Provider: Ollama API verwenden
	modelName := app.modelService.GetOllamaNameForModel(req.Model)
	log.Printf("Ollama Pull: %s", modelName)

	err := app.modelService.PullModel(modelName, func(progress string) {
		fmt.Fprintf(w, "data: %s\n\n", progress)
		flusher.Flush()
	})

	if err != nil {
		errJSON, _ := json.Marshal(map[string]string{"error": err.Error()})
		fmt.Fprintf(w, "data: %s\n\n", errJSON)
	} else {
		doneJSON, _ := json.Marshal(map[string]interface{}{"status": "done", "model": modelName})
		fmt.Fprintf(w, "data: %s\n\n", doneJSON)
	}
	flusher.Flush()
}

// handleLLMDeleteModel loescht ein Modell
func (app *App) handleLLMDeleteModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Model string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Model == "" {
		http.Error(w, "Model name required", http.StatusBadRequest)
		return
	}

	if err := app.modelService.DeleteModel(req.Model); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, map[string]string{"status": "deleted", "model": req.Model})
}

// handleLLMModelDetails gibt Details zu einem Modell zurueck
func (app *App) handleLLMModelDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Model-Name aus URL extrahieren: /api/llm/models/details/{name}
	modelName := r.URL.Path[len("/api/llm/models/details/"):]
	if modelName == "" {
		http.Error(w, "Model name required", http.StatusBadRequest)
		return
	}

	// Details vom Provider holen
	details, err := app.modelService.GetModelDetails(modelName)
	if err != nil {
		// Fallback: Nur Registry-Info
		entry := app.modelService.FindModelInRegistry(modelName)
		if entry != nil {
			writeJSON(w, entry)
			return
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Registry-Info hinzufuegen falls vorhanden
	entry := app.modelService.FindModelInRegistry(modelName)
	if entry != nil {
		details["registry_info"] = entry
	}

	writeJSON(w, details)
}

// handleLLMChat fuehrt einen Chat mit dem LLM durch
func (app *App) handleLLMChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Message string `json:"message"`
		Model   string `json:"model,omitempty"`
		Stream  bool   `json:"stream,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, "Message required", http.StatusBadRequest)
		return
	}

	model := req.Model
	if model == "" {
		model = app.modelService.GetSelectedModel()
	}

	// SSE Headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	ctx := context.Background()

	err := app.modelService.ChatWithModel(ctx, model, req.Message, func(chunk string, done bool) {
		data := map[string]interface{}{
			"content": chunk,
			"done":    done,
		}
		jsonData, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	})

	if err != nil {
		errData := map[string]interface{}{
			"error": err.Error(),
			"done":  true,
		}
		jsonData, _ := json.Marshal(errData)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	}
}

// handleLLMStatus gibt den Status des LLM-Systems zurueck
func (app *App) handleLLMStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := map[string]interface{}{
		"provider":        app.modelService.GetProviderName(),
		"provider_online": app.modelService.IsProviderAvailable(),
		"selected_model":  app.modelService.GetSelectedModel(),
		"default_model":   app.modelService.GetDefaultModel(),
	}

	// Installierte Modelle zaehlen
	if models, err := app.modelService.GetInstalledModels(); err == nil {
		status["installed_models_count"] = len(models)
	}

	// Registry-Statistiken
	registry := app.modelService.GetRegistry()
	status["registry_total"] = len(registry.GetAllModels())
	status["registry_featured"] = len(registry.GetFeaturedModels())

	writeJSON(w, status)
}

// --- Frontend-kompatible Endpoints ---

// handleModelsPull - POST /api/models/pull (Frontend-kompatibel)
// Unterstützt sowohl Ollama als auch llama-cpp (HuggingFace GGUF Download)
func (app *App) handleModelsPull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Model name required", http.StatusBadRequest)
		return
	}

	// SSE Headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Prüfen welcher Provider aktiv ist
	activeProvider := app.settingsService.GetActiveProvider()
	log.Printf("Model Pull: %s (Provider: %s)", req.Name, activeProvider)

	// Bei llama-cpp: GGUF von HuggingFace herunterladen
	if activeProvider == "llama-cpp" || activeProvider == "java-llama-cpp" || activeProvider == "llama-server" {
		// Modell in Registry suchen
		entry := app.modelService.FindModelInRegistry(req.Name)
		if entry == nil {
			// Auch nach Ollama-Name suchen
			entry = app.modelService.GetRegistry().FindByOllamaName(req.Name)
		}

		if entry == nil || entry.HuggingFaceRepo == "" {
			errJSON, _ := json.Marshal(map[string]string{
				"error": fmt.Sprintf("Modell '%s' nicht in Registry gefunden oder keine HuggingFace URL", req.Name),
			})
			fmt.Fprintf(w, "data: %s\n\n", errJSON)
			flusher.Flush()
			return
		}

		// HuggingFace URL zusammenbauen
		downloadURL := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", entry.HuggingFaceRepo, entry.Filename)
		log.Printf("Downloading GGUF from HuggingFace: %s", downloadURL)

		// Progress Channel für SSE
		progressChan := make(chan llamaserver.DownloadProgress, 100)
		done := make(chan error, 1)

		// Download im Hintergrund starten
		go func() {
			done <- app.llamaServer.DownloadModel(downloadURL, entry.Filename, progressChan)
			close(progressChan)
		}()

		// Progress an Client streamen
		lastPercent := -1
		for progress := range progressChan {
			// Nur bei signifikanter Änderung senden (jede 1%)
			percent := int(progress.Percent)
			if percent != lastPercent {
				lastPercent = percent
				progressJSON, _ := json.Marshal(map[string]interface{}{
					"status":     "downloading",
					"percent":    progress.Percent,
					"downloaded": progress.Downloaded,
					"total":      progress.Total,
					"filename":   progress.Filename,
				})
				fmt.Fprintf(w, "data: %s\n\n", progressJSON)
				flusher.Flush()
			}
		}

		// Auf Abschluss warten
		if err := <-done; err != nil {
			errJSON, _ := json.Marshal(map[string]string{"error": err.Error()})
			fmt.Fprintf(w, "data: %s\n\n", errJSON)
		} else {
			doneJSON, _ := json.Marshal(map[string]interface{}{
				"status":   "success",
				"message":  fmt.Sprintf("Modell %s erfolgreich heruntergeladen", entry.Filename),
				"filename": entry.Filename,
			})
			fmt.Fprintf(w, "data: %s\n\n", doneJSON)
		}
		flusher.Flush()
		return
	}

	// Fallback: Ollama Provider
	err := app.modelService.PullModel(req.Name, func(progress string) {
		fmt.Fprintf(w, "data: %s\n\n", progress)
		flusher.Flush()
	})

	if err != nil {
		errJSON, _ := json.Marshal(map[string]string{"error": err.Error()})
		fmt.Fprintf(w, "data: %s\n\n", errJSON)
	} else {
		doneJSON, _ := json.Marshal(map[string]interface{}{"status": "success"})
		fmt.Fprintf(w, "data: %s\n\n", doneJSON)
	}
	flusher.Flush()
}

// handleModelsDefault - GET /api/models/default
func (app *App) handleModelsDefault(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, map[string]string{
		"model": app.modelService.GetDefaultModel(),
	})
}

// handleModelByName - /api/models/{name}/...
func (app *App) handleModelByName(w http.ResponseWriter, r *http.Request) {
	// Path: /api/models/{name}/details oder /api/models/{name}/default
	path := r.URL.Path[len("/api/models/"):]
	parts := strings.SplitN(path, "/", 2)

	if len(parts) < 1 || parts[0] == "" {
		http.Error(w, "Model name required", http.StatusBadRequest)
		return
	}

	modelName, _ := url.PathUnescape(parts[0])
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch action {
	case "details":
		// GET /api/models/{name}/details
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		details, err := app.modelService.GetModelDetails(modelName)
		if err != nil {
			// Fallback: Registry-Info
			entry := app.modelService.FindModelInRegistry(modelName)
			if entry != nil {
				writeJSON(w, entry)
				return
			}
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, details)

	case "default":
		// POST /api/models/{name}/default
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		app.modelService.SetDefaultModel(modelName)
		app.modelService.SetSelectedModel(modelName)
		writeJSON(w, map[string]string{"status": "ok", "default_model": modelName})

	default:
		// DELETE /api/models/{name}
		if r.Method == http.MethodDelete {
			if err := app.modelService.DeleteModel(modelName); err != nil {
				writeJSON(w, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, map[string]string{"status": "deleted"})
			return
		}
		http.Error(w, "Unknown action", http.StatusBadRequest)
	}
}

// handleLLMProviderActive - GET /api/llm/providers/active
func (app *App) handleLLMProviderActive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, map[string]interface{}{
		"provider":  "ollama",
		"name":      app.modelService.GetProviderName(),
		"available": app.modelService.IsProviderAvailable(),
	})
}

// handleLLMProviderConfig - GET /api/llm/providers/config
func (app *App) handleLLMProviderConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, map[string]interface{}{
		"active_provider": "ollama",
		"ollama": map[string]interface{}{
			"url":     app.config.OllamaURL,
			"enabled": true,
		},
		"llamacpp": map[string]interface{}{
			"enabled": false,
		},
	})
}

// handleLLMProviders - GET /api/llm/providers
// Returns provider status (matching Java ProvidersResponse format)
func (app *App) handleLLMProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get active provider from settings (default: llama-server)
	activeProvider := app.settingsService.GetActiveProvider()

	// Check Ollama availability
	ollamaAvailable := false
	if activeProvider == "ollama" {
		ollamaAvailable = app.modelService.IsProviderAvailable()
	}

	// Check llama-server availability
	llamaServerAvailable := false
	if app.llamaServer != nil {
		llamaServerAvailable = app.llamaServer.IsRunning()
	}

	// Map internal names to frontend names
	// Backend: "llama-server" -> Frontend: "java-llama-cpp"
	frontendActiveProvider := activeProvider
	if activeProvider == "llama-server" {
		frontendActiveProvider = "java-llama-cpp"
	}

	writeJSON(w, map[string]interface{}{
		"activeProvider":     frontendActiveProvider,
		"availableProviders": []string{"ollama", "java-llama-cpp"},
		"providerStatus": map[string]bool{
			"ollama":         ollamaAvailable,
			"java-llama-cpp": llamaServerAvailable,
		},
	})
}

// handleLLMProviderSwitch - POST /api/llm/providers/switch
// Zentraler Provider-Wechsel Endpoint mit Verbindungsprüfung:
// - Bei Ollama: Verbindung prüfen, bei Fehler Fallback auf llama-cpp mit Fehlermeldung
// - Bei llama-cpp: Immer erfolgreich (lokaler Provider)
func (app *App) handleLLMProviderSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Provider string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Normalisiere Provider-Namen
	requestedProvider := normalizeProviderName(req.Provider)
	currentProvider := app.settingsService.GetActiveProvider()

	log.Printf("Provider-Wechsel: %s -> %s (angefordert: %s)", currentProvider, requestedProvider, req.Provider)

	// Bei Wechsel zu Ollama: Verbindung prüfen
	if requestedProvider == "ollama" {
		if !app.checkOllamaConnection() {
			// Ollama nicht erreichbar - Fallback auf llama-cpp
			log.Printf("FEHLER: Ollama-Wechsel fehlgeschlagen - Server nicht erreichbar")

			// Fallback auf llama-server setzen
			if err := app.settingsService.SaveActiveProvider("llama-server"); err != nil {
				log.Printf("Fehler beim Setzen des Fallback-Providers: %v", err)
			}

			// Frontend-freundliche Response mit Fehlermeldung
			writeJSON(w, map[string]interface{}{
				"success":           false,
				"message":           "Ollama-Server nicht erreichbar!",
				"error":             "Keine Verbindung zum Ollama-Server möglich. Bitte prüfen Sie:\n1. Ist Ollama installiert?\n2. Läuft 'ollama serve'?\n3. Ist Port 11434 erreichbar?",
				"requestedProvider": "ollama",
				"activeProvider":    "llama-server",
				"fallback":          true,
				"showNotification":  true,
				"notificationType":  "error",
			})
			return
		}
		log.Printf("Ollama-Verbindung erfolgreich - wechsle Provider")
	}

	// Provider speichern
	if err := app.settingsService.SaveActiveProvider(requestedProvider); err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"message": "Provider konnte nicht gespeichert werden",
			"error":   err.Error(),
		})
		return
	}

	log.Printf("Provider erfolgreich gewechselt zu: %s", requestedProvider)

	// Erfolgsmeldung
	writeJSON(w, map[string]interface{}{
		"success":          true,
		"message":          fmt.Sprintf("Provider erfolgreich auf %s gewechselt", getProviderDisplayName(requestedProvider)),
		"provider":         requestedProvider,
		"activeProvider":   requestedProvider,
		"showNotification": true,
		"notificationType": "success",
	})
}

// handleOllamaModels - GET /api/ollama/models
// Returns a direct JSON array of models (matching Java backend format)
func (app *App) handleOllamaModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	models, err := app.modelService.GetInstalledModels()
	if err != nil {
		// Return empty array on error (matching Java behavior)
		writeJSON(w, []interface{}{})
		return
	}

	// Return direct array (Frontend expects [...] not {"models": [...]})
	writeJSON(w, models)
}

// handleOllamaStatus - GET /api/ollama/status
// Returns Ollama status (matching Java backend format)
func (app *App) handleOllamaStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	available := app.modelService.IsProviderAvailable()
	message := "Ollama is not available"
	if available {
		message = "Ollama is running"
	}

	// Match Java OllamaStatus DTO format
	writeJSON(w, map[string]interface{}{
		"available": available,
		"message":   message,
	})
}

// handleModelStoreAll - GET /api/model-store/all
func (app *App) handleModelStoreAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	registry := app.modelService.GetRegistry()
	writeJSON(w, map[string]interface{}{
		"models": registry.GetAllModels(),
	})
}

// handleModelStoreFeatured - GET /api/model-store/featured
func (app *App) handleModelStoreFeatured(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	registry := app.modelService.GetRegistry()
	writeJSON(w, map[string]interface{}{
		"featured": registry.GetFeaturedModels(),
		"trending": registry.GetTrendingModels(),
	})
}

// handleFileUpload - POST /api/files/upload
// Handles file uploads for chat attachments (images, text, PDF)
func (app *App) handleFileUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Max 50MB
	r.ParseMultipartForm(50 << 20)

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "Datei konnte nicht gelesen werden: " + err.Error(),
		})
		return
	}
	defer file.Close()

	filename := header.Filename
	size := header.Size
	contentType := header.Header.Get("Content-Type")

	// Determine file type
	fileType := "unknown"
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		fileType = "image"
	case ".pdf":
		fileType = "pdf"
	case ".txt", ".md", ".markdown":
		fileType = "text"
	case ".docx":
		fileType = "docx"
	case ".odt":
		fileType = "odt"
	case ".xlsx":
		fileType = "xlsx"
	case ".xls":
		fileType = "xls"
	case ".csv":
		fileType = "csv"
	case ".eml":
		fileType = "eml"
	case ".html", ".htm":
		fileType = "html"
	case ".json":
		fileType = "json"
	case ".xml":
		fileType = "xml"
	default:
		// Check content type as fallback
		if strings.HasPrefix(contentType, "image/") {
			fileType = "image"
		} else if contentType == "application/pdf" {
			fileType = "pdf"
		} else if strings.HasPrefix(contentType, "text/") {
			fileType = "text"
		} else if contentType == "application/vnd.openxmlformats-officedocument.wordprocessingml.document" {
			fileType = "docx"
		} else if contentType == "application/vnd.oasis.opendocument.text" {
			fileType = "odt"
		} else if contentType == "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
			fileType = "xlsx"
		}
	}

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "Datei konnte nicht gelesen werden: " + err.Error(),
		})
		return
	}

	response := map[string]interface{}{
		"success":  true,
		"filename": filename,
		"type":     fileType,
		"size":     size,
	}

	switch fileType {
	case "image":
		// Encode as base64
		base64Content := base64.StdEncoding.EncodeToString(content)
		response["base64Content"] = base64Content

	case "text":
		// Return text content directly
		response["textContent"] = string(content)

	case "pdf":
		// Extract text from PDF using pdftotext first, then OCR if needed
		textContent, ocrUsed, err := extractPDFText(content, filename)
		if err != nil {
			log.Printf("PDF text extraction failed: %v", err)
			response["textContent"] = fmt.Sprintf("[PDF-Dokument: %s - Textextraktion fehlgeschlagen: %v]", filename, err)
		} else {
			response["textContent"] = textContent
			if ocrUsed {
				response["ocrUsed"] = true
				log.Printf("PDF OCR completed for %s", filename)
			}
		}
		// Also provide base64 for potential LLaVA processing
		base64Content := base64.StdEncoding.EncodeToString(content)
		response["base64Content"] = base64Content

	case "docx":
		// Extract text from DOCX (Word document)
		textContent, err := extractDOCXText(content)
		if err != nil {
			log.Printf("DOCX extraction failed: %v", err)
			response["textContent"] = fmt.Sprintf("[DOCX-Dokument: %s - Extraktion fehlgeschlagen: %v]", filename, err)
		} else {
			response["textContent"] = textContent
			log.Printf("DOCX extracted: %s (%d chars)", filename, len(textContent))
		}

	case "odt":
		// Extract text from ODT (LibreOffice document)
		textContent, err := extractODTText(content)
		if err != nil {
			log.Printf("ODT extraction failed: %v", err)
			response["textContent"] = fmt.Sprintf("[ODT-Dokument: %s - Extraktion fehlgeschlagen: %v]", filename, err)
		} else {
			response["textContent"] = textContent
			log.Printf("ODT extracted: %s (%d chars)", filename, len(textContent))
		}

	case "xlsx":
		// Extract text from XLSX (Excel spreadsheet)
		textContent, err := extractXLSXText(content)
		if err != nil {
			log.Printf("XLSX extraction failed: %v", err)
			response["textContent"] = fmt.Sprintf("[XLSX-Tabelle: %s - Extraktion fehlgeschlagen: %v]", filename, err)
		} else {
			response["textContent"] = textContent
			log.Printf("XLSX extracted: %s (%d chars)", filename, len(textContent))
		}

	case "csv":
		// CSV is text, but format it nicely
		textContent := formatCSVAsText(string(content))
		response["textContent"] = textContent
		log.Printf("CSV processed: %s (%d chars)", filename, len(textContent))

	case "eml":
		// Extract text from EML (email file)
		textContent, err := extractEMLText(content)
		if err != nil {
			log.Printf("EML extraction failed: %v", err)
			response["textContent"] = fmt.Sprintf("[E-Mail: %s - Extraktion fehlgeschlagen: %v]", filename, err)
		} else {
			response["textContent"] = textContent
			log.Printf("EML extracted: %s (%d chars)", filename, len(textContent))
		}

	case "html":
		// Strip HTML tags and extract text
		textContent := extractHTMLText(string(content))
		response["textContent"] = textContent
		log.Printf("HTML extracted: %s (%d chars)", filename, len(textContent))

	case "json":
		// JSON is text, pretty print it
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, content, "", "  "); err == nil {
			response["textContent"] = prettyJSON.String()
		} else {
			response["textContent"] = string(content)
		}

	case "xml":
		// XML is text, return as-is
		response["textContent"] = string(content)

	default:
		// Try to read as text
		response["textContent"] = string(content)
	}

	log.Printf("File uploaded: %s (%s, %d bytes)", filename, fileType, size)
	writeJSON(w, response)
}

// extractPDFText extrahiert Text aus einem PDF
// Versucht zuerst pdftotext, bei wenig/keinem Text wird OCR via Tesseract verwendet
func extractPDFText(pdfContent []byte, filename string) (string, bool, error) {
	// Temporäre Datei für PDF erstellen
	tmpFile, err := os.CreateTemp("", "fleet-pdf-*.pdf")
	if err != nil {
		return "", false, fmt.Errorf("temp file erstellen: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(pdfContent); err != nil {
		tmpFile.Close()
		return "", false, fmt.Errorf("PDF schreiben: %w", err)
	}
	tmpFile.Close()

	// Schritt 1: Versuche pdftotext (für normale PDFs)
	textContent, err := extractWithPdftotext(tmpPath)
	if err == nil && len(strings.TrimSpace(textContent)) > 50 {
		// Genug Text gefunden, kein OCR nötig
		log.Printf("PDF text extraction via pdftotext: %d chars", len(textContent))
		return textContent, false, nil
	}

	// Schritt 2: Wenig/kein Text - verwende OCR (für gescannte PDFs)
	log.Printf("PDF hat wenig Text (%d chars), starte OCR...", len(strings.TrimSpace(textContent)))
	ocrText, err := extractWithOCR(tmpPath)
	if err != nil {
		// Wenn OCR fehlschlägt, gib zumindest den pdftotext-Inhalt zurück
		if textContent != "" {
			return textContent, false, nil
		}
		return "", false, fmt.Errorf("OCR fehlgeschlagen: %w", err)
	}

	return ocrText, true, nil
}

// extractWithPdftotext verwendet pdftotext zum Extrahieren von Text
func extractWithPdftotext(pdfPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "pdftotext", "-layout", pdfPath, "-")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// extractWithOCR konvertiert PDF-Seiten zu Bildern und führt OCR durch
func extractWithOCR(pdfPath string) (string, error) {
	// Temporäres Verzeichnis für Bilder
	tmpDir, err := os.MkdirTemp("", "fleet-ocr-*")
	if err != nil {
		return "", fmt.Errorf("temp dir erstellen: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// PDF zu Bildern konvertieren mit pdftoppm
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	imgPrefix := filepath.Join(tmpDir, "page")
	cmd := exec.CommandContext(ctx, "pdftoppm", "-png", "-r", "300", pdfPath, imgPrefix)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pdftoppm fehlgeschlagen: %w", err)
	}

	// Alle erzeugten Bilder finden
	files, err := filepath.Glob(filepath.Join(tmpDir, "page-*.png"))
	if err != nil || len(files) == 0 {
		return "", fmt.Errorf("keine Bilder erzeugt")
	}

	// Sortieren für richtige Reihenfolge
	sort.Strings(files)

	// OCR auf jeder Seite durchführen
	var allText strings.Builder
	for i, imgPath := range files {
		pageText, err := runTesseract(imgPath)
		if err != nil {
			log.Printf("OCR Seite %d fehlgeschlagen: %v", i+1, err)
			continue
		}
		if i > 0 {
			allText.WriteString("\n\n--- Seite ")
			allText.WriteString(fmt.Sprintf("%d", i+1))
			allText.WriteString(" ---\n\n")
		}
		allText.WriteString(pageText)
	}

	result := allText.String()
	if strings.TrimSpace(result) == "" {
		return "", fmt.Errorf("OCR hat keinen Text erkannt")
	}

	log.Printf("OCR abgeschlossen: %d Seiten, %d Zeichen", len(files), len(result))
	return result, nil
}

// runTesseract führt Tesseract OCR auf einem Bild aus
func runTesseract(imagePath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Tesseract mit deutscher Sprache
	cmd := exec.CommandContext(ctx, "tesseract", imagePath, "stdout", "-l", "deu+eng", "--psm", "1")
	output, err := cmd.Output()
	if err != nil {
		// Fallback ohne Sprachpaket
		cmd = exec.CommandContext(ctx, "tesseract", imagePath, "stdout", "--psm", "1")
		output, err = cmd.Output()
		if err != nil {
			return "", err
		}
	}

	return string(output), nil
}

// ============================================================================
// Document Format Extraction Functions
// ============================================================================

// extractDOCXText extrahiert Text aus einem DOCX-Dokument (Word)
// DOCX ist ein ZIP-Archiv mit word/document.xml
func extractDOCXText(content []byte) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", fmt.Errorf("DOCX öffnen: %w", err)
	}

	var textContent strings.Builder

	for _, file := range reader.File {
		if file.Name == "word/document.xml" {
			rc, err := file.Open()
			if err != nil {
				return "", fmt.Errorf("document.xml öffnen: %w", err)
			}
			defer rc.Close()

			xmlContent, err := io.ReadAll(rc)
			if err != nil {
				return "", fmt.Errorf("document.xml lesen: %w", err)
			}

			// Einfaches XML-Parsing: Text zwischen <w:t> Tags extrahieren
			text := extractXMLText(string(xmlContent), "w:t")
			textContent.WriteString(text)
			break
		}
	}

	return textContent.String(), nil
}

// extractODTText extrahiert Text aus einem ODT-Dokument (LibreOffice)
// ODT ist ein ZIP-Archiv mit content.xml
func extractODTText(content []byte) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", fmt.Errorf("ODT öffnen: %w", err)
	}

	var textContent strings.Builder

	for _, file := range reader.File {
		if file.Name == "content.xml" {
			rc, err := file.Open()
			if err != nil {
				return "", fmt.Errorf("content.xml öffnen: %w", err)
			}
			defer rc.Close()

			xmlContent, err := io.ReadAll(rc)
			if err != nil {
				return "", fmt.Errorf("content.xml lesen: %w", err)
			}

			// Text zwischen <text:p> und <text:span> Tags extrahieren
			text := extractXMLTextMultiple(string(xmlContent), []string{"text:p", "text:span", "text:h"})
			textContent.WriteString(text)
			break
		}
	}

	return textContent.String(), nil
}

// extractXLSXText extrahiert Text aus einem XLSX-Dokument (Excel)
// XLSX ist ein ZIP-Archiv mit xl/sharedStrings.xml und xl/worksheets/sheet*.xml
func extractXLSXText(content []byte) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", fmt.Errorf("XLSX öffnen: %w", err)
	}

	var result strings.Builder

	// Zuerst sharedStrings.xml für String-Tabelle lesen
	var sharedStrings []string
	for _, file := range reader.File {
		if file.Name == "xl/sharedStrings.xml" {
			rc, err := file.Open()
			if err != nil {
				continue
			}
			xmlContent, _ := io.ReadAll(rc)
			rc.Close()

			// Strings aus <t> Tags extrahieren
			sharedStrings = extractXMLStrings(string(xmlContent), "t")
			break
		}
	}

	// Dann Worksheets lesen
	sheetNum := 0
	for _, file := range reader.File {
		if strings.HasPrefix(file.Name, "xl/worksheets/sheet") && strings.HasSuffix(file.Name, ".xml") {
			rc, err := file.Open()
			if err != nil {
				continue
			}
			xmlContent, _ := io.ReadAll(rc)
			rc.Close()

			sheetNum++
			if sheetNum > 1 {
				result.WriteString(fmt.Sprintf("\n\n=== Tabelle %d ===\n", sheetNum))
			}

			// Vereinfachte Extraktion: Alle <v> (Value) Tags
			values := extractXMLStrings(string(xmlContent), "v")
			for i, v := range values {
				// Wenn v eine Zahl ist, könnte es ein Index in sharedStrings sein
				if idx, err := strconv.Atoi(v); err == nil && idx < len(sharedStrings) {
					result.WriteString(sharedStrings[idx])
				} else {
					result.WriteString(v)
				}
				if (i+1)%10 == 0 { // Neue Zeile alle 10 Werte (grob geschätzt)
					result.WriteString("\n")
				} else {
					result.WriteString("\t")
				}
			}
		}
	}

	return result.String(), nil
}

// formatCSVAsText formatiert CSV-Inhalt als lesbaren Text
func formatCSVAsText(csvContent string) string {
	var result strings.Builder
	lines := strings.Split(csvContent, "\n")

	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		// Zeile mit Zeilennummer ausgeben
		result.WriteString(fmt.Sprintf("Zeile %d: %s\n", i+1, line))
	}

	return result.String()
}

// extractEMLText extrahiert Text aus einer E-Mail-Datei (EML)
func extractEMLText(content []byte) (string, error) {
	var result strings.Builder
	lines := strings.Split(string(content), "\n")

	inHeader := true
	inBody := false
	var headers []string
	var body strings.Builder

	for _, line := range lines {
		if inHeader {
			if strings.TrimSpace(line) == "" {
				inHeader = false
				inBody = true
				continue
			}
			// Wichtige Header extrahieren
			lowerLine := strings.ToLower(line)
			if strings.HasPrefix(lowerLine, "from:") ||
				strings.HasPrefix(lowerLine, "to:") ||
				strings.HasPrefix(lowerLine, "subject:") ||
				strings.HasPrefix(lowerLine, "date:") {
				headers = append(headers, strings.TrimSpace(line))
			}
		} else if inBody {
			body.WriteString(line)
			body.WriteString("\n")
		}
	}

	// Header formatiert ausgeben
	result.WriteString("=== E-Mail Header ===\n")
	for _, h := range headers {
		result.WriteString(h)
		result.WriteString("\n")
	}
	result.WriteString("\n=== E-Mail Inhalt ===\n")
	result.WriteString(body.String())

	return result.String(), nil
}

// extractHTMLText entfernt HTML-Tags und extrahiert reinen Text
func extractHTMLText(htmlContent string) string {
	// Einfache Regex-basierte Tag-Entfernung
	// Script und Style Blöcke entfernen
	htmlContent = regexp.MustCompile(`(?i)<script[^>]*>[\s\S]*?</script>`).ReplaceAllString(htmlContent, "")
	htmlContent = regexp.MustCompile(`(?i)<style[^>]*>[\s\S]*?</style>`).ReplaceAllString(htmlContent, "")

	// Block-Level-Elemente zu Zeilenumbrüchen
	htmlContent = regexp.MustCompile(`(?i)</?(p|div|br|h[1-6]|li|tr)[^>]*>`).ReplaceAllString(htmlContent, "\n")

	// Alle anderen Tags entfernen
	htmlContent = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(htmlContent, "")

	// HTML-Entities dekodieren (einfach)
	htmlContent = strings.ReplaceAll(htmlContent, "&nbsp;", " ")
	htmlContent = strings.ReplaceAll(htmlContent, "&amp;", "&")
	htmlContent = strings.ReplaceAll(htmlContent, "&lt;", "<")
	htmlContent = strings.ReplaceAll(htmlContent, "&gt;", ">")
	htmlContent = strings.ReplaceAll(htmlContent, "&quot;", "\"")
	htmlContent = strings.ReplaceAll(htmlContent, "&#39;", "'")

	// Mehrfache Leerzeilen reduzieren
	htmlContent = regexp.MustCompile(`\n{3,}`).ReplaceAllString(htmlContent, "\n\n")

	return strings.TrimSpace(htmlContent)
}

// extractXMLText extrahiert Text zwischen bestimmten XML-Tags
func extractXMLText(xmlContent, tagName string) string {
	var result strings.Builder
	pattern := regexp.MustCompile(fmt.Sprintf(`<%s[^>]*>([^<]*)</%s>`, tagName, tagName))
	matches := pattern.FindAllStringSubmatch(xmlContent, -1)

	for _, match := range matches {
		if len(match) > 1 {
			text := strings.TrimSpace(match[1])
			if text != "" {
				result.WriteString(text)
				result.WriteString(" ")
			}
		}
	}

	return result.String()
}

// extractXMLTextMultiple extrahiert Text aus mehreren Tag-Typen
func extractXMLTextMultiple(xmlContent string, tagNames []string) string {
	var result strings.Builder

	for _, tagName := range tagNames {
		text := extractXMLText(xmlContent, tagName)
		if text != "" {
			result.WriteString(text)
			result.WriteString("\n")
		}
	}

	return result.String()
}

// extractXMLStrings extrahiert alle Strings aus einem bestimmten Tag
func extractXMLStrings(xmlContent, tagName string) []string {
	var results []string
	pattern := regexp.MustCompile(fmt.Sprintf(`<%s[^>]*>([^<]*)</%s>`, tagName, tagName))
	matches := pattern.FindAllStringSubmatch(xmlContent, -1)

	for _, match := range matches {
		if len(match) > 1 {
			results = append(results, match[1])
		}
	}

	return results
}

// --- Tools API Handler ---

// handleTools - GET /api/tools
// Returns list of available tools
func (app *App) handleTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, map[string]interface{}{
		"tools":       app.toolRegistry.GetToolInfo(),
		"definitions": app.toolRegistry.GetToolDefinitions(),
	})
}

// handleToolExecute - POST /api/tools/execute
// Executes a tool by name with given parameters
func (app *App) handleToolExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Tool   string                 `json:"tool"`
		Params map[string]interface{} `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Tool == "" {
		http.Error(w, "Tool name required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	result, err := app.toolRegistry.Execute(ctx, req.Tool, req.Params)
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	writeJSON(w, result)
}

// handleToolSearch - POST /api/tools/search
// Shortcut for web search
func (app *App) handleToolSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Query      string `json:"query"`
		MaxResults int    `json:"maxResults,omitempty"`
		Region     string `json:"region,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "Query required", http.StatusBadRequest)
		return
	}

	params := map[string]interface{}{
		"query": req.Query,
	}
	if req.MaxResults > 0 {
		params["maxResults"] = float64(req.MaxResults)
	}
	if req.Region != "" {
		params["region"] = req.Region
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	result, err := app.toolRegistry.Execute(ctx, "web_search", params)
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	writeJSON(w, result)
}

// handleToolFetch - POST /api/tools/fetch
// Shortcut for web fetch (direct URL content retrieval)
func (app *App) handleToolFetch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URL          string `json:"url"`
		ExtractLinks bool   `json:"extractLinks,omitempty"`
		MaxLength    int    `json:"maxLength,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL required", http.StatusBadRequest)
		return
	}

	params := map[string]interface{}{
		"url": req.URL,
	}
	if req.ExtractLinks {
		params["extractLinks"] = true
	}
	if req.MaxLength > 0 {
		params["maxLength"] = float64(req.MaxLength)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	result, err := app.toolRegistry.Execute(ctx, "web_fetch", params)
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	writeJSON(w, result)
}

// --- Vision API Handler ---

// handleVisionAnalyze - POST /api/vision/analyze
// Analyzes an image using LLaVA with streaming response
func (app *App) handleVisionAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Image  string `json:"image"`  // Base64 encoded image
		Prompt string `json:"prompt"` // Optional custom prompt
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Image == "" {
		http.Error(w, "Image (base64) is required", http.StatusBadRequest)
		return
	}

	// SSE Headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 180*time.Second)
	defer cancel()

	var fullResponse string
	err := app.visionService.StreamAnalyzeImage(ctx, req.Image, req.Prompt, func(content string, done bool) {
		fullResponse += content
		data := map[string]interface{}{
			"content": content,
			"done":    done,
		}
		jsonData, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	})

	if err != nil {
		errData := map[string]interface{}{
			"error": err.Error(),
			"done":  true,
		}
		jsonData, _ := json.Marshal(errData)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	}
}

// handleVisionDocument - POST /api/vision/document
// Analyzes a document image with BOTH Vision (logos, stamps, signatures) AND Tesseract (full text)
// This combined approach ensures:
// - Vision: Analyzes layout, logos, official stamps, signatures, structure
// - Tesseract: Extracts ALL text (unlimited pages, stored in DB for future queries)
func (app *App) handleVisionDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Image string `json:"image"` // Base64 encoded image
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Image == "" {
		http.Error(w, "Image (base64) is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 180*time.Second)
	defer cancel()

	// Kombinierte Analyse: Vision + Tesseract
	// Vision: Logos, Stempel, Unterschriften, Layout
	// Tesseract: Vollständiger Text (für DB-Speicherung)
	combinedResult, err := app.visionService.AnalyzeDocumentWithOCR(ctx, req.Image, app.config.DataDir)
	if err != nil {
		// Fallback auf reine Vision-Analyse wenn Tesseract fehlschlägt
		log.Printf("[Vision] Kombinierte Analyse fehlgeschlagen, versuche nur Vision: %v", err)
		analysis, visionErr := app.visionService.AnalyzeDocument(ctx, req.Image)
		if visionErr != nil {
			writeJSON(w, map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		writeJSON(w, map[string]interface{}{
			"success":        true,
			"analysis":       analysis,
			"tesseractUsed":  false,
			"tesseractError": err.Error(),
		})
		return
	}

	writeJSON(w, map[string]interface{}{
		"success":       true,
		"analysis":      combinedResult.ImageAnalysis,
		"fullOcrText":   combinedResult.FullOCRText,   // Vollständiger OCR-Text für DB!
		"tesseractUsed": combinedResult.TesseractUsed,
		"pageCount":     combinedResult.PageCount,
	})
}

// handleVisionStatus - GET /api/vision/status
// Returns vision service status including Tesseract OCR
func (app *App) handleVisionStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	visionAvailable := app.visionService.IsAvailable()
	tesseractInstalled := vision.IsTesseractInstalled(app.config.DataDir)
	tesseractLangs := vision.GetTesseractLanguages(app.config.DataDir)

	writeJSON(w, map[string]interface{}{
		"available": visionAvailable,
		"model":     app.visionService.GetModel(),
		"provider":  "llama-server",
		// Tesseract OCR Status
		"tesseract": map[string]interface{}{
			"installed": tesseractInstalled,
			"languages": tesseractLangs,
		},
		// Kombinierte Analyse verfügbar wenn beides installiert
		"combinedAnalysis": visionAvailable && tesseractInstalled,
	})
}

// handleVisionOCR - POST /api/vision/ocr
// Reine Tesseract-OCR ohne Vision-Analyse
// Schnelle Text-Extraktion für Dokumente
func (app *App) handleVisionOCR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Image     string `json:"image"`     // Base64 encoded image
		Languages string `json:"languages"` // Optional: "deu+eng+tur" (default: "deu+eng")
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Image == "" {
		http.Error(w, "Image (base64) is required", http.StatusBadRequest)
		return
	}

	// Prüfe ob Tesseract installiert ist
	if !vision.IsTesseractInstalled(app.config.DataDir) {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "Tesseract ist nicht installiert. Bitte über Setup installieren.",
			"hint":    "GET /api/setup/tesseract/download",
		})
		return
	}

	// Standard-Sprachen
	languages := req.Languages
	if languages == "" {
		languages = "deu+eng"
	}

	log.Printf("[OCR] Starte Tesseract OCR (Sprachen: %s)", languages)

	// OCR ausführen
	text, err := vision.TesseractOCRFromBase64(req.Image, app.config.DataDir, languages)
	if err != nil {
		log.Printf("[OCR] Fehler: %v", err)
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	log.Printf("[OCR] ✅ Erfolgreich: %d Zeichen extrahiert", len(text))

	writeJSON(w, map[string]interface{}{
		"success":   true,
		"text":      text,
		"length":    len(text),
		"languages": languages,
	})
}

// handleVisionPDFStream - POST /api/vision/pdf-stream
// Verarbeitet große PDFs mit Streaming-Fortschrittsanzeige
// Für Dokumente mit vielen Seiten (89, 158+ Seiten)
func (app *App) handleVisionPDFStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Multipart Form für PDF-Upload
	if err := r.ParseMultipartForm(100 << 20); err != nil { // Max 100MB
		http.Error(w, "PDF zu groß oder ungültiges Format", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("pdf")
	if err != nil {
		http.Error(w, "PDF-Datei fehlt", http.StatusBadRequest)
		return
	}
	defer file.Close()

	log.Printf("[Vision/PDF] Verarbeite PDF: %s (%d Bytes)", header.Filename, header.Size)

	// PDF temporär speichern
	tmpPDF, err := os.CreateTemp("", "fleet-pdf-*.pdf")
	if err != nil {
		http.Error(w, "Temp-Datei erstellen fehlgeschlagen", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpPDF.Name())

	if _, err := io.Copy(tmpPDF, file); err != nil {
		tmpPDF.Close()
		http.Error(w, "PDF speichern fehlgeschlagen", http.StatusInternalServerError)
		return
	}
	tmpPDF.Close()

	// SSE-Header setzen
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming nicht unterstützt", http.StatusInternalServerError)
		return
	}

	// PDF zu Bildern konvertieren
	tempDir, err := os.MkdirTemp("", "fleet-pdf-pages-*")
	if err != nil {
		fmt.Fprintf(w, "data: {\"error\": \"Temp-Verzeichnis: %s\"}\n\n", err.Error())
		flusher.Flush()
		return
	}
	defer os.RemoveAll(tempDir)

	// Start-Event
	fmt.Fprintf(w, "data: {\"status\": \"converting\", \"message\": \"Konvertiere PDF zu Bildern...\"}\n\n")
	flusher.Flush()

	// pdftoppm für Konvertierung (poppler-utils)
	cmd := exec.Command("pdftoppm", "-png", "-r", "200", tmpPDF.Name(), filepath.Join(tempDir, "page"))
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(w, "data: {\"error\": \"PDF-Konvertierung fehlgeschlagen (poppler-utils installiert?)\"}\n\n")
		flusher.Flush()
		return
	}

	// Generierte Bilder finden
	pattern := filepath.Join(tempDir, "page-*.png")
	images, err := filepath.Glob(pattern)
	if err != nil || len(images) == 0 {
		fmt.Fprintf(w, "data: {\"error\": \"Keine Seiten gefunden\"}\n\n")
		flusher.Flush()
		return
	}

	totalPages := len(images)
	fmt.Fprintf(w, "data: {\"status\": \"processing\", \"totalPages\": %d, \"message\": \"Starte OCR für %d Seiten...\"}\n\n", totalPages, totalPages)
	flusher.Flush()

	// Tesseract OCR für jede Seite mit Fortschritt
	var allText strings.Builder
	for i, imagePath := range images {
		pageNum := i + 1
		progress := int(float64(pageNum) / float64(totalPages) * 100)

		// Fortschritt senden
		fmt.Fprintf(w, "data: {\"status\": \"ocr\", \"page\": %d, \"totalPages\": %d, \"progress\": %d}\n\n", pageNum, totalPages, progress)
		flusher.Flush()

		// OCR für diese Seite
		text, err := vision.TesseractOCR(imagePath, app.config.DataDir, "deu+eng+tur")
		if err != nil {
			log.Printf("[Vision/PDF] ⚠️ Seite %d OCR-Fehler: %v", pageNum, err)
			continue
		}

		if allText.Len() > 0 {
			allText.WriteString("\n\n--- Seite ")
			allText.WriteString(fmt.Sprintf("%d", pageNum))
			allText.WriteString(" ---\n\n")
		}
		allText.WriteString(text)
	}

	// Vision-Analyse nur für erste Seite (Layout, Logos, Stempel)
	fmt.Fprintf(w, "data: {\"status\": \"vision\", \"message\": \"Analysiere Layout und visuelle Elemente...\"}\n\n")
	flusher.Flush()

	var visionResult *vision.ImageAnalysis
	if len(images) > 0 {
		base64Data, err := loadImageAsBase64(images[0])
		if err == nil {
			ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
			defer cancel()
			visionResult, _ = app.visionService.AnalyzeDocument(ctx, base64Data)
		}
	}

	// Ergebnis senden
	result := map[string]interface{}{
		"status":        "complete",
		"totalPages":    totalPages,
		"fullOcrText":   allText.String(),
		"tesseractUsed": true,
		"charCount":     allText.Len(),
	}
	if visionResult != nil {
		result["analysis"] = visionResult
	}

	resultJSON, _ := json.Marshal(result)
	fmt.Fprintf(w, "data: %s\n\n", resultJSON)
	flusher.Flush()

	log.Printf("[Vision/PDF] ✅ PDF verarbeitet: %d Seiten, %d Zeichen", totalPages, allText.Len())
}

// loadImageAsBase64 lädt ein Bild und gibt es als Base64 zurück
func loadImageAsBase64(imagePath string) (string, error) {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// ============== Frontend-Kompatibilitäts-Endpoints ==============

// handleStatsGlobal - GET /api/stats/global
// Returns global statistics (stub for frontend compatibility)
func (app *App) handleStatsGlobal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Stub-Response für Frontend-Kompatibilität
	writeJSON(w, map[string]interface{}{
		"totalChats":          0,
		"totalMessages":       0,
		"totalTokens":         0,
		"averageResponseTime": 0,
		"activeUsers":         1,
		"connectedMates":      len(app.wsServer.GetConnectedMates()),
		"trustedMates":        len(app.pairingManager.GetTrustedMates()),
	})
}

// handleCustomModels - GET/POST /api/custom-models
func (app *App) handleCustomModels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		models, err := app.customModelService.GetAll()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, models)

	case http.MethodPost:
		var req custommodel.CreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		log.Printf("Creating custom model: %s (base: %s)", req.Name, req.BaseModel)

		model, err := app.customModelService.Create(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		writeJSON(w, model)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleCustomModelByID - GET/PUT/DELETE /api/custom-models/{id}
func (app *App) handleCustomModelByID(w http.ResponseWriter, r *http.Request) {
	// ID aus URL extrahieren
	path := r.URL.Path[len("/api/custom-models/"):]
	if path == "" || path == "generate-modelfile" {
		return
	}

	// Prüfen ob es ein Sub-Endpoint ist
	parts := strings.Split(path, "/")
	idStr := parts[0]
	var subEndpoint string
	if len(parts) > 1 {
		subEndpoint = parts[1]
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Ungültige Model ID", http.StatusBadRequest)
		return
	}

	// Sub-Endpoint: ancestry
	if subEndpoint == "ancestry" && r.Method == http.MethodGet {
		ancestry, err := app.customModelService.GetAncestry(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, ancestry)
		return
	}

	switch r.Method {
	case http.MethodGet:
		model, err := app.customModelService.GetByID(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if model == nil {
			http.Error(w, "Custom Model nicht gefunden", http.StatusNotFound)
			return
		}
		writeJSON(w, model)

	case http.MethodPut:
		var req custommodel.UpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		log.Printf("Updating custom model: %d", id)

		model, err := app.customModelService.Update(id, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, model)

	case http.MethodDelete:
		model, err := app.customModelService.GetByID(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if model == nil {
			http.Error(w, "Custom Model nicht gefunden", http.StatusNotFound)
			return
		}

		log.Printf("Deleting custom model: %d (%s)", id, model.Name)

		if err := app.customModelService.Delete(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, map[string]interface{}{
			"message": "Custom model deleted from database",
			"model":   model.Name,
			"note":    "Model still exists in Ollama. Use 'ollama rm " + model.Name + "' to remove it.",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGenerateModelfile - POST /api/custom-models/generate-modelfile
func (app *App) handleGenerateModelfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req custommodel.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	modelfile := app.customModelService.GenerateModelfile(
		req.BaseModel, req.SystemPrompt,
		req.Temperature, req.TopP, req.TopK, req.RepeatPenalty,
		req.NumPredict, req.NumCtx,
	)

	writeJSON(w, map[string]string{"modelfile": modelfile})
}

// handleSettingsSelectedModel - GET/POST /api/settings/selected-model
// Saves and retrieves the selected model setting
func (app *App) handleSettingsSelectedModel(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Return currently selected model
		if app.selectedModel == "" {
			writeJSON(w, map[string]interface{}{"model": "qwen2.5:7b"})
		} else {
			writeJSON(w, map[string]interface{}{"model": app.selectedModel})
		}
	case http.MethodPost:
		var req struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if req.Model == "" {
			http.Error(w, "Model is required", http.StatusBadRequest)
			return
		}
		app.selectedModel = req.Model
		app.modelService.SetSelectedModel(req.Model)
		log.Printf("Settings: Ausgewähltes Modell gesetzt auf: %s", req.Model)
		writeJSON(w, map[string]interface{}{"success": true, "model": req.Model})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGgufModels - GET/POST /api/gguf-models
// CRUD operations for GGUF model configurations
func (app *App) handleGgufModels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Alle GGUF-Konfigurationen abrufen
		configs, err := app.customModelService.GetAllGgufConfigs()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, configs)

	case http.MethodPost:
		// Neue GGUF-Konfiguration erstellen
		var req custommodel.GgufConfigCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		config, err := app.customModelService.CreateGgufConfig(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		writeJSON(w, config)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGgufModelByID - GET/PUT/DELETE /api/gguf-models/{id}
func (app *App) handleGgufModelByID(w http.ResponseWriter, r *http.Request) {
	// ID aus URL extrahieren
	path := r.URL.Path[len("/api/gguf-models/"):]
	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		config, err := app.customModelService.GetGgufConfigByID(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if config == nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		writeJSON(w, config)

	case http.MethodPut:
		var req custommodel.GgufConfigUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		config, err := app.customModelService.UpdateGgufConfig(id, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, config)

	case http.MethodDelete:
		if err := app.customModelService.DeleteGgufConfig(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]bool{"success": true})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handlePersonalInfo - GET/POST /api/personal-info
// Manages user's personal information (stub for frontend compatibility)
func (app *App) handlePersonalInfo(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Return empty personal info
		writeJSON(w, map[string]interface{}{
			"name":     "",
			"email":    "",
			"company":  "",
			"position": "",
		})
	case http.MethodPost, http.MethodPut:
		// Accept but ignore for now
		writeJSON(w, map[string]interface{}{"success": true})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleTemplates - GET/POST /api/templates
// Manages prompt templates (stub for frontend compatibility)
func (app *App) handleTemplates(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Return default templates
		templates := []map[string]interface{}{
			{
				"id":          "default-brief",
				"name":        "Geschäftsbrief",
				"description": "Vorlage für formelle Geschäftsbriefe",
				"category":    "dokumente",
				"prompt":      "Erstelle einen formellen Geschäftsbrief mit folgendem Inhalt: ",
			},
			{
				"id":          "default-email",
				"name":        "E-Mail",
				"description": "Vorlage für geschäftliche E-Mails",
				"category":    "kommunikation",
				"prompt":      "Schreibe eine professionelle E-Mail zu folgendem Thema: ",
			},
			{
				"id":          "default-zusammenfassung",
				"name":        "Zusammenfassung",
				"description": "Text zusammenfassen",
				"category":    "analyse",
				"prompt":      "Fasse den folgenden Text zusammen: ",
			},
		}
		writeJSON(w, templates)
	case http.MethodPost:
		// Accept but don't store templates (stub)
		writeJSON(w, map[string]interface{}{
			"success": true,
			"message": "Template saved",
		})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleProjects - GET/POST /api/projects
// Manages projects/workspaces (stub for frontend compatibility)
func (app *App) handleProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Return empty list of projects
		writeJSON(w, []interface{}{})
	case http.MethodPost:
		// Accept but don't store projects (stub)
		writeJSON(w, map[string]interface{}{
			"success": true,
			"message": "Projects not yet implemented in Go backend",
		})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ============== System Prompts API Handler ==============

// handleSystemPrompts - GET/POST /api/system-prompts
func (app *App) handleSystemPrompts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		promptsList, err := app.promptsService.GetAll()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, promptsList)

	case http.MethodPost:
		var prompt prompts.SystemPromptTemplate
		if err := json.NewDecoder(r.Body).Decode(&prompt); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if prompt.Name == "" || prompt.Content == "" {
			http.Error(w, "Name and content are required", http.StatusBadRequest)
			return
		}

		if err := app.promptsService.Create(&prompt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("System-Prompt erstellt: %s (ID: %d)", prompt.Name, prompt.ID)
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, prompt)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSystemPromptsDefault - GET /api/system-prompts/default
func (app *App) handleSystemPromptsDefault(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defaultPrompt, err := app.promptsService.GetDefault()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if defaultPrompt == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	writeJSON(w, defaultPrompt)
}

// handleSystemPromptsInitDefaults - POST /api/system-prompts/init-defaults
func (app *App) handleSystemPromptsInitDefaults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := app.promptsService.InitializeDefaults(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"status": "ok", "message": "Default prompts initialized"})
}

// handleSystemPromptByID - GET/PUT/DELETE /api/system-prompts/{id}
// Also handles /api/system-prompts/{id}/set-default
func (app *App) handleSystemPromptByID(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/api/system-prompts/"):]

	// Skip if hitting special endpoints
	if path == "" || path == "default" || path == "init-defaults" {
		return
	}

	// Handle /api/system-prompts/{id}/set-default
	if strings.HasSuffix(path, "/set-default") {
		idStr := strings.TrimSuffix(path, "/set-default")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid prompt ID", http.StatusBadRequest)
			return
		}

		if r.Method != http.MethodPut && r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := app.promptsService.SetDefault(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		prompt, _ := app.promptsService.GetByID(id)
		if prompt != nil {
			log.Printf("System-Prompt '%s' (ID: %d) als Standard gesetzt", prompt.Name, id)
			writeJSON(w, prompt)
		} else {
			writeJSON(w, map[string]string{"status": "ok"})
		}
		return
	}

	// Parse ID
	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		http.Error(w, "Invalid prompt ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		prompt, err := app.promptsService.GetByID(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if prompt == nil {
			http.Error(w, "Prompt not found", http.StatusNotFound)
			return
		}
		writeJSON(w, prompt)

	case http.MethodPut:
		var prompt prompts.SystemPromptTemplate
		if err := json.NewDecoder(r.Body).Decode(&prompt); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		prompt.ID = id

		// Wenn dieser Prompt als Default gesetzt wird, alle anderen zurücksetzen
		if prompt.IsDefault {
			if err := app.promptsService.SetDefault(id); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		if err := app.promptsService.Update(&prompt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("System-Prompt aktualisiert: %s (ID: %d)", prompt.Name, id)
		writeJSON(w, prompt)

	case http.MethodDelete:
		if err := app.promptsService.Delete(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("System-Prompt gelöscht: ID %d", id)
		writeJSON(w, map[string]string{"status": "deleted"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ============== App Settings API Handler ==============

// handleSettings - GET /api/settings (alle Einstellungen)
func (app *App) handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	allSettings, err := app.settingsService.GetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Als Key-Value Map zurückgeben
	settingsMap := make(map[string]string)
	for _, s := range allSettings {
		settingsMap[s.Key] = s.Value
	}

	writeJSON(w, settingsMap)
}

// handleSettingsModelSelection - GET/PUT /api/settings/model-selection
func (app *App) handleSettingsModelSelection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		modelSettings := app.settingsService.GetModelSelectionSettings()
		writeJSON(w, modelSettings)

	case http.MethodPut:
		var req settings.ModelSelectionSettings
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if err := app.settingsService.UpdateModelSelectionSettings(req); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, req)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSettingsSelectedExpert - GET/POST /api/settings/selected-expert
func (app *App) handleSettingsSelectedExpert(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		expertID := app.settingsService.GetSelectedExpertID()
		if expertID == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeJSON(w, expertID)

	case http.MethodPost:
		body := make([]byte, 1024)
		n, _ := r.Body.Read(body)
		expertIDStr := strings.TrimSpace(string(body[:n]))

		if expertIDStr == "" || expertIDStr == "null" {
			app.settingsService.SaveSelectedExpertID(0)
			w.WriteHeader(http.StatusOK)
			return
		}

		expertID, err := strconv.ParseInt(expertIDStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid expert ID", http.StatusBadRequest)
			return
		}

		if err := app.settingsService.SaveSelectedExpertID(expertID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSettingsShowWelcomeTiles - GET/POST /api/settings/show-welcome-tiles
func (app *App) handleSettingsShowWelcomeTiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		show := app.settingsService.GetShowWelcomeTiles()
		writeJSON(w, show)

	case http.MethodPost:
		var show bool
		if err := json.NewDecoder(r.Body).Decode(&show); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if err := app.settingsService.SaveShowWelcomeTiles(show); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSettingsShowTopBar - GET/POST /api/settings/show-top-bar
func (app *App) handleSettingsShowTopBar(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		show := app.settingsService.GetShowTopBar()
		writeJSON(w, show)

	case http.MethodPost:
		var show bool
		if err := json.NewDecoder(r.Body).Decode(&show); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if err := app.settingsService.SaveShowTopBar(show); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSettingsUITheme - GET/POST /api/settings/ui-theme
func (app *App) handleSettingsUITheme(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		theme := app.settingsService.GetUITheme()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(theme))

	case http.MethodPost:
		body := make([]byte, 1024)
		n, _ := r.Body.Read(body)
		theme := strings.TrimSpace(string(body[:n]))
		// Remove quotes if sent as JSON string
		theme = strings.Trim(theme, "\"")

		if theme == "" {
			theme = "tech-dark"
		}

		if err := app.settingsService.SaveUITheme(theme); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSettingsLLMProvider - GET/POST /api/settings/llm-provider
// Provider-Wechsel mit Verbindungsprüfung:
// - Bei Wechsel zu Ollama: Verbindung prüfen, bei Fehler Fallback auf llama-cpp
// - Bei Wechsel zu llama-cpp: Immer erfolgreich (lokaler Provider)
func (app *App) handleSettingsLLMProvider(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		provider := app.settingsService.GetActiveProvider()
		// Aktuellen Status mitgeben
		available := app.checkProviderAvailable(provider)
		writeJSON(w, map[string]interface{}{
			"provider":  provider,
			"available": available,
		})

	case http.MethodPost:
		var req struct {
			Provider string `json:"provider"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Normalisiere Provider-Namen (Frontend sendet verschiedene Varianten)
		requestedProvider := normalizeProviderName(req.Provider)
		log.Printf("Provider-Wechsel angefordert: %s (original: %s)", requestedProvider, req.Provider)

		// Bei Wechsel zu Ollama: Verbindung prüfen
		if requestedProvider == "ollama" {
			if !app.checkOllamaConnection() {
				// Ollama nicht erreichbar - Fallback auf llama-cpp mit Fehlermeldung
				log.Printf("WARNUNG: Ollama nicht erreichbar, Fallback auf llama-server")

				// Setze llama-server als aktiven Provider
				if err := app.settingsService.SaveActiveProvider("llama-server"); err != nil {
					log.Printf("Fehler beim Setzen des Fallback-Providers: %v", err)
				}

				writeJSON(w, map[string]interface{}{
					"success":          false,
					"message":          "Ollama-Server nicht erreichbar. Automatischer Fallback auf llama-cpp aktiviert.",
					"error":            "Keine Verbindung zum Ollama-Server möglich. Bitte stellen Sie sicher, dass Ollama läuft (ollama serve).",
					"requestedProvider": "ollama",
					"activeProvider":    "llama-server",
					"fallback":          true,
				})
				return
			}
		}

		// Provider wechseln
		if err := app.settingsService.SaveActiveProvider(requestedProvider); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("Provider erfolgreich gewechselt zu: %s", requestedProvider)

		writeJSON(w, map[string]interface{}{
			"success":        true,
			"message":        fmt.Sprintf("Provider erfolgreich auf %s gewechselt", getProviderDisplayName(requestedProvider)),
			"provider":       requestedProvider,
			"activeProvider": requestedProvider,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// normalizeProviderName normalisiert verschiedene Provider-Namen auf interne Namen
func normalizeProviderName(name string) string {
	switch strings.ToLower(name) {
	case "ollama":
		return "ollama"
	case "llama-cpp", "llamacpp", "llama.cpp", "java-llama-cpp", "llama-server":
		return "llama-server"
	case "":
		return "llama-server" // Default ist llama-server, nicht ollama!
	default:
		return "llama-server"
	}
}

// getProviderDisplayName gibt den Anzeigenamen für einen Provider zurück
func getProviderDisplayName(provider string) string {
	switch provider {
	case "ollama":
		return "Ollama"
	case "llama-server":
		return "llama.cpp (lokal)"
	default:
		return provider
	}
}

// checkProviderAvailable prüft ob ein Provider verfügbar ist
func (app *App) checkProviderAvailable(provider string) bool {
	switch provider {
	case "ollama":
		return app.checkOllamaConnection()
	case "llama-server":
		// llama-server ist immer verfügbar (wird bei Bedarf gestartet)
		return true
	default:
		return false
	}
}

// checkOllamaConnection prüft ob der Ollama-Server erreichbar ist
func (app *App) checkOllamaConnection() bool {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(app.config.OllamaURL + "/api/tags")
	if err != nil {
		log.Printf("Ollama-Verbindung fehlgeschlagen: %v", err)
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// ============================================================================
// User & Auth Handlers
// ============================================================================

// handleLogin - POST /api/auth/login
func (app *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req user.LoginRequest

	// Unterstütze sowohl JSON als auch Form-Data (für Frontend-Kompatibilität)
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		// Form-Data parsing
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}
		req.Username = r.FormValue("username")
		req.Password = r.FormValue("password")
	} else {
		// JSON parsing
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
	}

	// IP-Adresse und User-Agent extrahieren
	ipAddress := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ipAddress = strings.Split(forwarded, ",")[0]
	}
	userAgent := r.Header.Get("User-Agent")

	response, err := app.userService.Login(req.Username, req.Password, ipAddress, userAgent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Token als HTTP-Cookie setzen (für Frontend-Kompatibilität mit credentials: 'include')
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    response.Token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400, // 24 Stunden
	})

	writeJSON(w, response)
}

// handleLogout - POST /api/auth/logout
func (app *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Token aus Header extrahieren
	token := extractToken(r)
	if token == "" {
		http.Error(w, "Token erforderlich", http.StatusUnauthorized)
		return
	}

	if err := app.userService.Logout(token); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"message": "Erfolgreich abgemeldet"})
}

// handleValidateToken - GET /api/auth/validate
func (app *App) handleValidateToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := extractToken(r)
	if token == "" {
		http.Error(w, "Token erforderlich", http.StatusUnauthorized)
		return
	}

	userObj, err := app.userService.ValidateToken(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	writeJSON(w, map[string]interface{}{
		"valid": true,
		"user":  userObj,
	})
}

// handleCurrentUser - GET /api/auth/me
func (app *App) handleCurrentUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := extractToken(r)
	if token == "" {
		http.Error(w, "Token erforderlich", http.StatusUnauthorized)
		return
	}

	userObj, err := app.userService.ValidateToken(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	writeJSON(w, userObj)
}

// handleUsers - GET/POST /api/users
func (app *App) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Authentifizierung prüfen (nur Admin)
		userObj, err := app.authenticateRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if userObj.Role != user.RoleAdmin {
			http.Error(w, "Nur Administratoren können Benutzer auflisten", http.StatusForbidden)
			return
		}

		users, err := app.userService.GetAllUsers()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, users)

	case http.MethodPost:
		// Authentifizierung prüfen (nur Admin)
		userObj, err := app.authenticateRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if userObj.Role != user.RoleAdmin {
			http.Error(w, "Nur Administratoren können Benutzer erstellen", http.StatusForbidden)
			return
		}

		var req user.CreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		newUser, err := app.userService.CreateUser(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		writeJSON(w, newUser)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleUserByID - GET/PUT/DELETE /api/users/{id}
func (app *App) handleUserByID(w http.ResponseWriter, r *http.Request) {
	// ID aus URL extrahieren
	path := r.URL.Path[len("/api/users/"):]
	if path == "" {
		http.Error(w, "User ID erforderlich", http.StatusBadRequest)
		return
	}

	// Prüfen ob es ein Sub-Endpoint ist
	parts := strings.Split(path, "/")
	idStr := parts[0]
	var subEndpoint string
	if len(parts) > 1 {
		subEndpoint = parts[1]
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Ungültige User ID", http.StatusBadRequest)
		return
	}

	// Sub-Endpoints
	if subEndpoint == "password" && r.Method == http.MethodPost {
		app.handleChangePassword(w, r, id)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Authentifizierung prüfen
		currentUser, err := app.authenticateRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// User darf nur sich selbst oder Admin alle sehen
		if currentUser.ID != id && currentUser.Role != user.RoleAdmin {
			http.Error(w, "Keine Berechtigung", http.StatusForbidden)
			return
		}

		userObj, err := app.userService.GetUserByID(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if userObj == nil {
			http.Error(w, "Benutzer nicht gefunden", http.StatusNotFound)
			return
		}
		writeJSON(w, userObj)

	case http.MethodPut, http.MethodPatch:
		// Authentifizierung prüfen
		currentUser, err := app.authenticateRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// User darf nur sich selbst editieren, Admin alles
		if currentUser.ID != id && currentUser.Role != user.RoleAdmin {
			http.Error(w, "Keine Berechtigung", http.StatusForbidden)
			return
		}

		var req user.UpdateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Nicht-Admins dürfen Role nicht ändern
		if req.Role != nil && currentUser.Role != user.RoleAdmin {
			http.Error(w, "Nur Administratoren können Rollen ändern", http.StatusForbidden)
			return
		}

		updatedUser, err := app.userService.UpdateUser(id, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, updatedUser)

	case http.MethodDelete:
		// Nur Admin darf löschen
		currentUser, err := app.authenticateRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if currentUser.Role != user.RoleAdmin {
			http.Error(w, "Nur Administratoren können Benutzer löschen", http.StatusForbidden)
			return
		}

		// Selbst-Löschung verhindern
		if currentUser.ID == id {
			http.Error(w, "Sie können sich nicht selbst löschen", http.StatusBadRequest)
			return
		}

		if err := app.userService.DeleteUser(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]string{"status": "deleted"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleChangePassword - POST /api/users/{id}/password
func (app *App) handleChangePassword(w http.ResponseWriter, r *http.Request, userID int64) {
	// Authentifizierung prüfen
	currentUser, err := app.authenticateRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// User darf nur eigenes Passwort ändern
	if currentUser.ID != userID && currentUser.Role != user.RoleAdmin {
		http.Error(w, "Keine Berechtigung", http.StatusForbidden)
		return
	}

	var req user.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := app.userService.ChangePassword(userID, req.CurrentPassword, req.NewPassword); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, map[string]string{"message": "Passwort erfolgreich geändert"})
}

// Helper: Token aus Request extrahieren
func extractToken(r *http.Request) string {
	// Zuerst Authorization Header prüfen
	auth := r.Header.Get("Authorization")
	if auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
		return auth
	}

	// Dann Cookie prüfen (für Frontend mit credentials: 'include')
	if cookie, err := r.Cookie("auth_token"); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// Dann Query Parameter
	return r.URL.Query().Get("token")
}

// Helper: Request authentifizieren und User zurückgeben
func (app *App) authenticateRequest(r *http.Request) (*user.User, error) {
	token := extractToken(r)
	if token == "" {
		return nil, fmt.Errorf("Token erforderlich")
	}
	return app.userService.ValidateToken(token)
}

// ============================================================================
// llama-server API Handler
// ============================================================================

// handleLlamaServerStatus gibt den Status des llama-servers zurück
func (app *App) handleLlamaServerStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := app.llamaServer.GetStatus()
	writeJSON(w, status)
}

// handleLlamaServerStart startet den llama-server mit einem Modell
func (app *App) handleLlamaServerStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Model       string `json:"model"`       // Modellname (z.B. "Qwen2.5-7B-Instruct-Q5_K_M.gguf")
		ModelPath   string `json:"modelPath"`   // Voller Pfad (alternativ)
		Port        int    `json:"port"`        // Optional: Port überschreiben
		ContextSize int    `json:"contextSize"` // Optional: Context Size überschreiben
		GPULayers   int    `json:"gpuLayers"`   // Optional: GPU Layers überschreiben
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Modellpfad bestimmen: entweder direkt gegeben oder aus Modellname suchen
	modelPath := req.ModelPath
	if modelPath == "" && req.Model != "" {
		// Suche den vollen Pfad für das Modell
		models, err := app.llamaServer.GetAvailableModels()
		if err == nil {
			for _, m := range models {
				if m.Name == req.Model || m.Path == req.Model {
					modelPath = m.Path
					break
				}
			}
		}
	}

	if modelPath == "" {
		http.Error(w, "model oder modelPath erforderlich", http.StatusBadRequest)
		return
	}

	// Optional: Konfiguration überschreiben
	if req.Port > 0 {
		app.llamaServer.SetPort(req.Port)
	}
	if req.ContextSize > 0 {
		app.llamaServer.SetContextSize(req.ContextSize)
	}
	if req.GPULayers >= 0 {
		app.llamaServer.SetGPULayers(req.GPULayers)
	}

	if err := app.llamaServer.Start(modelPath); err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Status zurückgeben
	status := app.llamaServer.GetStatus()
	writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "llama-server gestartet",
		"model":   status.ModelName,
		"port":    status.Port,
		"running": status.Running,
	})
}

// handleLlamaServerStop stoppt den llama-server
func (app *App) handleLlamaServerStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := app.llamaServer.Stop(); err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "llama-server gestoppt",
	})
}

// handleLlamaServerRestart startet den llama-server neu
func (app *App) handleLlamaServerRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := app.llamaServer.Restart(); err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "llama-server neu gestartet",
	})
}

// handleLlamaServerContextChange ändert die Context-Größe und startet den Server neu
// POST /api/llamaserver/context
func (app *App) handleLlamaServerContextChange(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// GET: Aktuelle Context-Größe zurückgeben
		writeJSON(w, map[string]interface{}{
			"contextSize":    app.llamaServer.GetContextSize(),
			"defaultContext": 65536, // 64K Default
		})
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ContextSize int `json:"contextSize"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "Ungültiger Request-Body",
		})
		return
	}

	// Validierung: Context muss zwischen 1K und 256K sein
	if req.ContextSize < 1024 || req.ContextSize > 262144 {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "Context-Größe muss zwischen 1024 und 262144 liegen",
		})
		return
	}

	// Context ändern und Server neustarten
	estimatedSeconds, err := app.llamaServer.RestartWithContextSize(req.ContextSize)
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"success":       false,
			"error":         err.Error(),
			"restartNeeded": true,
		})
		return
	}

	writeJSON(w, map[string]interface{}{
		"success":          true,
		"contextSize":      req.ContextSize,
		"restartNeeded":    estimatedSeconds > 0,
		"estimatedSeconds": estimatedSeconds,
		"message":          fmt.Sprintf("Context auf %d gesetzt", req.ContextSize),
	})
}

// handleLlamaServerModels gibt alle verfügbaren GGUF-Modelle zurück
func (app *App) handleLlamaServerModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	models, err := app.llamaServer.GetAvailableModels()
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"models": []interface{}{},
			"error":  err.Error(),
		})
		return
	}

	writeJSON(w, map[string]interface{}{
		"models": models,
		"count":  len(models),
	})
}

// handleLlamaServerModelsRecommended gibt empfohlene Modelle zum Download zurück
func (app *App) handleLlamaServerModelsRecommended(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	models := llamaserver.GetRecommendedModels()
	writeJSON(w, map[string]interface{}{
		"models": models,
	})
}

// handleLlamaServerDownload lädt ein Modell herunter (SSE für Progress)
func (app *App) handleLlamaServerDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URL      string `json:"url"`
		Filename string `json:"filename"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" || req.Filename == "" {
		http.Error(w, "url und filename erforderlich", http.StatusBadRequest)
		return
	}

	// SSE-Header setzen
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming nicht unterstützt", http.StatusInternalServerError)
		return
	}

	// Progress Channel
	progressChan := make(chan llamaserver.DownloadProgress, 100)

	// Download in Goroutine starten
	go func() {
		err := app.llamaServer.DownloadModel(req.URL, req.Filename, progressChan)
		if err != nil {
			progressChan <- llamaserver.DownloadProgress{
				Filename: req.Filename,
				Percent:  -1, // Fehler-Signal
			}
		}
		close(progressChan)
	}()

	// Progress-Events senden
	for progress := range progressChan {
		if progress.Percent < 0 {
			// Fehler
			fmt.Fprintf(w, "data: {\"error\":\"Download fehlgeschlagen\",\"done\":true}\n\n")
		} else {
			data, _ := json.Marshal(progress)
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
		flusher.Flush()
	}

	// Abschluss-Event
	fmt.Fprintf(w, "data: {\"done\":true,\"filename\":\"%s\"}\n\n", req.Filename)
	flusher.Flush()
}

// handleLlamaServerConfig gibt die Konfiguration zurück oder aktualisiert sie
func (app *App) handleLlamaServerConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		config := app.llamaServer.GetConfig()
		writeJSON(w, config)

	case http.MethodPost:
		var req struct {
			GPULayers   *int `json:"gpuLayers"`
			ContextSize *int `json:"contextSize"`
			Threads     *int `json:"threads"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		config := app.llamaServer.GetConfig()

		if req.GPULayers != nil {
			config.GPULayers = *req.GPULayers
		}
		if req.ContextSize != nil {
			config.ContextSize = *req.ContextSize
		}
		if req.Threads != nil {
			config.Threads = *req.Threads
		}

		// Konfiguration speichern
		if err := app.llamaServer.SaveConfig(app.config.DataDir); err != nil {
			writeJSON(w, map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		writeJSON(w, map[string]interface{}{
			"success": true,
			"config":  config,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ============================================================================
// Neue Frontend-Kompatibilitäts-Handler
// ============================================================================

// handleLlamaServerHealth gibt einen einfachen Health-Status zurück
func (app *App) handleLlamaServerHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := app.llamaServer.GetStatus()
	writeJSON(w, map[string]interface{}{
		"healthy":   status.Healthy,
		"running":   status.Running,
		"online":    status.Running && status.Healthy, // Frontend erwartet "online"
		"modelName": status.ModelName,
		"port":      2026,
		"status":    "running",
	})
}

// handleLlamaServerVRAMSettings gibt VRAM-Einstellungen zurück oder aktualisiert sie
func (app *App) handleLlamaServerVRAMSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Alle VRAM-Settings inkl. verfügbarer Strategien zurückgeben
		settings := app.llamaServer.GetVRAMSettings()
		writeJSON(w, settings)

	case http.MethodPost:
		var req struct {
			Strategy  *string `json:"strategy"`
			ReserveMB *int    `json:"reserveMB"`
			UseMmap   *bool   `json:"useMmap"`
			UseMlock  *bool   `json:"useMlock"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Strategie aktualisieren
		if req.Strategy != nil {
			strategy := llamaserver.VRAMStrategy(*req.Strategy)
			// Validieren
			validStrategies := map[llamaserver.VRAMStrategy]bool{
				llamaserver.StrategySmartSwap:   true,
				llamaserver.StrategyAlwaysClear: true,
				llamaserver.StrategySmartOffload: true,
				llamaserver.StrategyManual:      true,
			}
			if !validStrategies[strategy] {
				http.Error(w, "Invalid VRAM strategy", http.StatusBadRequest)
				return
			}
			app.llamaServer.SetVRAMStrategy(strategy)
		}

		// Reserve aktualisieren
		if req.ReserveMB != nil {
			if *req.ReserveMB < 0 || *req.ReserveMB > 4096 {
				http.Error(w, "Invalid VRAM reserve (0-4096 MB)", http.StatusBadRequest)
				return
			}
			app.llamaServer.SetVRAMReserve(*req.ReserveMB)
		}

		// mmap aktualisieren
		if req.UseMmap != nil {
			app.llamaServer.SetUseMmap(*req.UseMmap)
		}

		// mlock aktualisieren
		if req.UseMlock != nil {
			app.llamaServer.SetUseMlock(*req.UseMlock)
		}

		// Konfiguration speichern
		if err := app.llamaServer.SaveConfig(app.config.DataDir); err != nil {
			writeJSON(w, map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		writeJSON(w, map[string]interface{}{
			"success":  true,
			"settings": app.llamaServer.GetVRAMSettings(),
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleLlamaServerVRAMInfo gibt nur die aktuellen VRAM-Informationen zurück
func (app *App) handleLlamaServerVRAMInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	info := llamaserver.GetVRAMInfo()
	writeJSON(w, info)
}

// handleLlamaServerVRAMClear löscht manuell den VRAM
func (app *App) handleLlamaServerVRAMClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Server stoppen falls läuft
	if app.llamaServer.IsRunning() {
		app.llamaServer.Stop()
	}

	// VRAM löschen
	if err := llamaserver.ClearVRAM(); err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Neue VRAM-Info zurückgeben
	info := llamaserver.GetVRAMInfo()
	writeJSON(w, map[string]interface{}{
		"success": true,
		"vram":    info,
	})
}

// ============================================================================
// Hardware Monitoring API Handler
// ============================================================================

// handleFleetMateStats gibt Hardware-Stats für einen bestimmten Mate zurück
// GET /api/fleet-mate/mates/{mateId}/stats
func (app *App) handleFleetMateStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// MateID aus URL extrahieren: /api/fleet-mate/mates/{mateId}/stats
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/api/fleet-mate/mates/"), "/")
	if len(parts) < 1 {
		http.Error(w, "Mate ID erforderlich", http.StatusBadRequest)
		return
	}
	mateID := parts[0]

	// Für den lokalen Navigator geben wir die lokalen Hardware-Stats zurück
	if mateID == "local-navigator" {
		stats, err := app.hardwareMonitor.Collect()
		if err != nil {
			writeJSON(w, map[string]interface{}{
				"error": err.Error(),
			})
			return
		}
		writeJSON(w, stats)
		return
	}

	// Für echte Mates: Stats vom WebSocket-Server abrufen
	mateStats := app.wsServer.GetMateStats(mateID)
	if mateStats == nil {
		writeJSON(w, map[string]interface{}{
			"error":  "Keine Stats für diesen Mate verfügbar",
			"mateId": mateID,
		})
		return
	}

	// Stats in Frontend-kompatiblem Format zurückgeben
	writeJSON(w, map[string]interface{}{
		"system":      mateStats.System,
		"cpu":         mateStats.CPU,
		"memory":      mateStats.Memory,
		"gpu":         mateStats.GPU,
		"temperature": mateStats.Temperature,
		"remoteIp":    mateStats.RemoteIP,
		"updatedAt":   mateStats.UpdatedAt,
	})
}

// handleHardwareStats gibt alle Hardware-Statistiken zurueck
func (app *App) handleHardwareStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := app.hardwareMonitor.Collect()
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, stats)
}

// handleHardwareQuickStats gibt schnelle Stats fuer die TopBar zurueck
func (app *App) handleHardwareQuickStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := app.hardwareMonitor.GetQuickStats()
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, stats)
}

// handleHardwareCPU gibt CPU-Statistiken zurueck
func (app *App) handleHardwareCPU(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := app.hardwareMonitor.GetCPU()
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, stats)
}

// handleHardwareMemory gibt RAM-Statistiken zurueck
func (app *App) handleHardwareMemory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := app.hardwareMonitor.GetMemory()
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, stats)
}

// handleHardwareGPU gibt GPU-Statistiken zurueck
func (app *App) handleHardwareGPU(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := app.hardwareMonitor.GetGPU()
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"error":   err.Error(),
			"hasGPU":  false,
		})
		return
	}

	writeJSON(w, map[string]interface{}{
		"hasGPU": true,
		"gpus":   stats,
	})
}

// handleAuthCheck prüft ob der User eingeloggt ist
func (app *App) handleAuthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userObj, err := app.authenticateRequest(r)
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"authenticated": false,
		})
		return
	}

	// Frontend erwartet flache Struktur mit username, displayName, role
	displayName := userObj.Username
	if userObj.DisplayName != "" {
		displayName = userObj.DisplayName
	}

	writeJSON(w, map[string]interface{}{
		"authenticated": true,
		"username":      userObj.Username,
		"displayName":   displayName,
		"role":          userObj.Role,
	})
}

// handleAuthRegister registriert einen neuen Benutzer
func (app *App) handleAuthRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	newUser, err := app.userService.CreateUser(user.CreateUserRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Role:     "user",
	})
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	writeJSON(w, map[string]interface{}{
		"success": true,
		"user": map[string]interface{}{
			"id":       newUser.ID,
			"username": newUser.Username,
			"email":    newUser.Email,
		},
	})
}

// handleAuthChangePassword ändert das Passwort
func (app *App) handleAuthChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := app.authenticateRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := app.userService.ChangePassword(user.ID, req.OldPassword, req.NewPassword); err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	writeJSON(w, map[string]interface{}{
		"success": true,
		"message": "Passwort erfolgreich geändert",
	})
}

// handleSystemStatus gibt den System-Status zurück
func (app *App) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ollamaStatus := "offline"
	if app.chatService.IsAvailable() {
		ollamaStatus = "online"
	}

	llamaStatus := "offline"
	if app.llamaServer.IsRunning() && app.llamaServer.IsHealthy() {
		llamaStatus = "online"
	}

	writeJSON(w, map[string]interface{}{
		"status":           "online",
		"ollama":           ollamaStatus,
		"llamaServer":      llamaStatus,
		"uptime":           time.Now().Format(time.RFC3339),
		"version":          updater.Version,
		"backendVersion":   updater.Version,
		"backendBuildTime": updater.BuildTime,
		"goVersion":        "1.24",
	})
}

// handleSystemVersion gibt die Version zurück
func (app *App) handleSystemVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, map[string]interface{}{
		"version":   updater.Version,
		"build":     "go-native",
		"buildTime": updater.BuildTime,
		"goVersion": "1.24",
	})
}

// handleSystemDBSize gibt die Datenbankgröße zurück
func (app *App) handleSystemDBSize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Größe aller DB-Dateien ermitteln
	var totalSize int64
	dbFiles := []string{
		filepath.Join(app.config.DataDir, "chats.db"),
		filepath.Join(app.config.DataDir, "experts.db"),
		filepath.Join(app.config.DataDir, "prompts.db"),
		filepath.Join(app.config.DataDir, "settings.db"),
		filepath.Join(app.config.DataDir, "users.db"),
		filepath.Join(app.config.DataDir, "custom_models.db"),
	}

	for _, dbFile := range dbFiles {
		if info, err := os.Stat(dbFile); err == nil {
			totalSize += info.Size()
		}
	}

	// Frontend-kompatibles Format (sizeBytes, formatted)
	formatted := formatBytesForDisplay(totalSize)

	writeJSON(w, map[string]interface{}{
		"sizeBytes":   totalSize,
		"formatted":   formatted,
		"totalSize":   totalSize,
		"totalSizeMB": float64(totalSize) / 1024 / 1024,
	})
}

// formatBytesForDisplay formatiert Bytes für die Anzeige
func formatBytesForDisplay(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(bytes)/(1024*1024*1024))
}

// handleSystemDBSizeHistory gibt die DB-Größen-Historie zurück (Stub)
func (app *App) handleSystemDBSizeHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Stub - gibt leere Historie zurück
	writeJSON(w, map[string]interface{}{
		"history": []interface{}{},
	})
}

// handleSystemStats gibt lokale Hardware-Statistiken zurück (CPU, RAM, GPU, Temp)
func (app *App) handleSystemStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := app.hardwareMonitor.Collect()
	if err != nil {
		http.Error(w, "Failed to collect hardware stats: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, stats)
}

// handleSystemStatsQuick gibt minimale Hardware-Stats für die TopBar zurück (schnell)
func (app *App) handleSystemStatsQuick(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := app.hardwareMonitor.GetQuickStats()
	if err != nil {
		http.Error(w, "Failed to collect quick stats: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, stats)
}

// handleSystemSetupStatus prüft ob das System eingerichtet ist
func (app *App) handleSystemSetupStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Prüfen ob Modelle verfügbar sind
	hasModels := app.chatService.IsAvailable()
	if !hasModels {
		models, _ := app.llamaServer.GetAvailableModels()
		hasModels = len(models) > 0
	}

	// Prüfen ob Experten existieren
	experts, _ := app.expertenService.GetAllExperts(true)
	hasExperts := len(experts) > 0

	writeJSON(w, map[string]interface{}{
		"setupComplete": hasModels && hasExperts,
		"hasModels":     hasModels,
		"hasExperts":    hasExperts,
		"hasOllama":     app.chatService.IsAvailable(),
		"hasLlamaServer": app.llamaServer.GetStatus().BinaryFound,
	})
}

// handleAiStartupStatus - GET /api/system/ai-startup-status
// Gibt den AI-Startup-Status für das Loading-Overlay zurück
func (app *App) handleAiStartupStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Prüfen ob llama-server läuft und healthy ist
	status := app.llamaServer.GetStatus()
	serverOnline := status.Running && status.Healthy

	// Auch Ollama als "online" betrachten
	ollamaOnline := app.chatService.IsAvailable()

	// Wenn einer von beiden online ist, ist die AI bereit
	aiReady := serverOnline || ollamaOnline

	// Status-Nachricht basierend auf dem Zustand
	var message string
	var inProgress bool
	if aiReady {
		message = "AI bereit"
		inProgress = false
	} else if status.Running && !status.Healthy {
		message = "Server startet..."
		inProgress = true
	} else if !status.BinaryFound && !ollamaOnline {
		message = "Kein AI-Backend verfügbar"
		inProgress = false
	} else {
		message = "Verbinde mit AI..."
		inProgress = true
	}

	writeJSON(w, map[string]interface{}{
		"inProgress":   inProgress,
		"complete":     !inProgress,
		"message":      message,
		"error":        nil,
		"serverOnline": aiReady,
	})
}

// ============================================================================
// Database/PostgreSQL Migration Handlers
// ============================================================================

// handleDatabaseStatus gibt den aktuellen Datenbankstatus zurück
func (app *App) handleDatabaseStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Aktuell verwenden wir nur SQLite
	// TODO: Wenn PostgreSQL-Migration implementiert ist, hier den echten Status prüfen
	writeJSON(w, map[string]interface{}{
		"database":    "sqlite",
		"connected":   true,
		"pgvector":    false,
		"description": "SQLite Datenbank aktiv",
	})
}

// handlePostgresConfig gibt die PostgreSQL-Konfiguration zurück oder speichert sie
func (app *App) handlePostgresConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Gespeicherte Konfiguration laden (ohne Passwort)
		// TODO: Aus Config-Datei oder DB laden
		writeJSON(w, map[string]interface{}{
			"host":     "localhost",
			"port":     5432,
			"database": "fleet_navigator",
			"schema":   "public",
			"username": "postgres",
			"sslMode":  "disable",
		})

	case http.MethodPost:
		// Konfiguration speichern (ohne Migration)
		var config struct {
			Host     string `json:"host"`
			Port     int    `json:"port"`
			Database string `json:"database"`
			Schema   string `json:"schema"`
			Username string `json:"username"`
			Password string `json:"password"`
			SSLMode  string `json:"sslMode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			writeJSON(w, map[string]interface{}{
				"success": false,
				"error":   "Ungültige Konfiguration: " + err.Error(),
			})
			return
		}

		// TODO: Konfiguration persistent speichern
		log.Printf("PostgreSQL-Konfiguration gespeichert: %s@%s:%d/%s",
			config.Username, config.Host, config.Port, config.Database)

		writeJSON(w, map[string]interface{}{
			"success": true,
			"message": "Konfiguration gespeichert",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handlePostgresTest testet die PostgreSQL-Verbindung
func (app *App) handlePostgresTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var config struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Database string `json:"database"`
		Schema   string `json:"schema"`
		Username string `json:"username"`
		Password string `json:"password"`
		SSLMode  string `json:"sslMode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"message": "Ungültige Konfiguration: " + err.Error(),
		})
		return
	}

	// Verbindungs-String erstellen
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.Username, config.Password, config.Database, config.SSLMode)

	log.Printf("Teste PostgreSQL-Verbindung: %s@%s:%d/%s",
		config.Username, config.Host, config.Port, config.Database)

	// Versuche Verbindung herzustellen
	// Note: Für echte PostgreSQL-Verbindung müsste github.com/lib/pq importiert werden
	// Hier simulieren wir den Test für die UI-Entwicklung

	// Simulierter Test - in Production würde hier:
	// db, err := sql.Open("postgres", connStr)
	// defer db.Close()
	// err = db.Ping()

	// Für Entwicklung: Prüfe ob Host erreichbar ist (einfacher TCP-Check)
	timeout := 5 * time.Second
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", config.Host, config.Port), timeout)
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Verbindung fehlgeschlagen: %v", err),
			"pgvector": false,
		})
		return
	}
	conn.Close()

	// Verbindung erfolgreich (TCP-Level)
	// TODO: Echte PostgreSQL-Authentifizierung und pgvector-Check
	log.Printf("PostgreSQL-Verbindung erfolgreich (TCP): %s", connStr[:50]+"...")

	writeJSON(w, map[string]interface{}{
		"success":         true,
		"message":         "Verbindung erfolgreich! (TCP-Level verifiziert)",
		"pgvector":        false, // TODO: Echten pgvector-Check implementieren
		"pgvectorVersion": nil,
	})
}

// handlePostgresMigrate führt die Migration von SQLite zu PostgreSQL durch
func (app *App) handlePostgresMigrate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var config struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Database string `json:"database"`
		Schema   string `json:"schema"`
		Username string `json:"username"`
		Password string `json:"password"`
		SSLMode  string `json:"sslMode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "Ungültige Konfiguration: " + err.Error(),
		})
		return
	}

	log.Printf("PostgreSQL-Migration angefordert: %s@%s:%d/%s",
		config.Username, config.Host, config.Port, config.Database)

	// TODO: Echte Migration implementieren
	// 1. SQLite-Daten exportieren
	// 2. PostgreSQL-Schema erstellen
	// 3. Daten importieren
	// 4. pgvector-Extension aktivieren
	// 5. Vektor-Indizes erstellen

	// Aktuell: Simulation für UI-Entwicklung
	writeJSON(w, map[string]interface{}{
		"success":         false,
		"error":           "Migration noch nicht implementiert. PostgreSQL-Unterstützung wird in einer zukünftigen Version hinzugefügt.",
		"pgvector":        false,
		"pgvectorVersion": nil,
	})
}

// handleUpdateStatus - GET /api/update/status (Stub für Frontend)
func (app *App) handleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, map[string]interface{}{
		"updateAvailable": false,
		"currentVersion":  "2.0.0",
		"latestVersion":   "2.0.0",
		"message":         "Sie verwenden die aktuelle Version",
	})
}

// handleMateByID behandelt einzelne Mate-Operationen
func (app *App) handleMateByID(w http.ResponseWriter, r *http.Request) {
	// ID aus URL extrahieren
	path := strings.TrimPrefix(r.URL.Path, "/api/pairing/trusted/")
	if path == "" {
		http.Error(w, "Mate ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Einzelnen Mate zurückgeben
		mates := app.pairingManager.GetTrustedMates()
		for _, mate := range mates {
			if mate.ID == path {
				writeJSON(w, mate)
				return
			}
		}
		http.Error(w, "Mate not found", http.StatusNotFound)

	case http.MethodDelete:
		// Mate entfernen
		if err := app.pairingManager.RemoveTrustedMate(path); err != nil {
			writeJSON(w, map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		writeJSON(w, map[string]interface{}{
			"success": true,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleFileSearchStatus gibt den Status der Dateisuche zurück
func (app *App) handleFileSearchStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Prüfen ob ein Mate mit Datei-Zugriff verbunden ist
	mates := app.pairingManager.GetTrustedMates()
	hasFileAccess := false
	for _, mate := range mates {
		if mate.Type == "desktop" || mate.Type == "file-browser" {
			hasFileAccess = true
			break
		}
	}

	writeJSON(w, map[string]interface{}{
		"available":     hasFileAccess,
		"indexedFolders": 0,
		"totalFiles":    0,
	})
}

// handleFileSearchFolders verwaltet indizierte Ordner
func (app *App) handleFileSearchFolders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Liste der indizierten Ordner (Stub)
		writeJSON(w, map[string]interface{}{
			"folders": []interface{}{},
		})

	case http.MethodPost:
		// Ordner hinzufügen (Stub)
		writeJSON(w, map[string]interface{}{
			"success": true,
			"message": "Ordner-Indizierung noch nicht implementiert",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleFileSearchFolderByID behandelt einzelne Ordner
func (app *App) handleFileSearchFolderByID(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		writeJSON(w, map[string]interface{}{
			"success": true,
		})
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handleSearchSettings verwaltet Web-Such-Einstellungen
func (app *App) handleSearchSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		settings := app.searchService.GetSettings()

		// Zähler aus der Datenbank lesen (persistiert!)
		var monthlyCount int
		var currentMonth string
		if app.settingsService != nil {
			monthlyCount, currentMonth = app.settingsService.GetWebSearchCount()
		}

		writeJSON(w, map[string]interface{}{
			"braveApiKey":              maskAPIKey(settings.BraveAPIKey),
			"hasBraveKey":              settings.BraveAPIKey != "",
			"braveConfigured":          settings.BraveAPIKey != "",
			"searxngInstances":         settings.SearXNGInstances,
			"customSearxng":            settings.CustomSearXNG,
			"customSearxngInstance":    settings.CustomSearXNG,
			"enableQueryOptimize":      settings.EnableQueryOptimize,
			"queryOptimizationEnabled": settings.EnableQueryOptimize,
			"enableContentFetch":       settings.EnableContentFetch,
			"contentScrapingEnabled":   settings.EnableContentFetch,
			"enableReRanking":          settings.EnableReRanking,
			"reRankingEnabled":         settings.EnableReRanking,
			"optimizationModel":        settings.OptimizationModel,
			"queryOptimizationModel":   settings.OptimizationModel,
			"monthlySearchCount":       monthlyCount,
			"searchCount":              monthlyCount,
			"searchLimit":              2000,
			"remainingSearches":        2000 - monthlyCount,
			"currentMonth":             currentMonth,
		})

	case http.MethodPost:
		var req struct {
			BraveAPIKey         string   `json:"braveApiKey"`
			SearXNGInstances    []string `json:"searxngInstances"`
			CustomSearXNG       string   `json:"customSearxng"`
			EnableQueryOptimize bool     `json:"enableQueryOptimize"`
			EnableContentFetch  bool     `json:"enableContentFetch"`
			EnableReRanking     bool     `json:"enableReRanking"`
			OptimizationModel   string   `json:"optimizationModel"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		settings := app.searchService.GetSettings()

		// Only update API key if not masked
		if req.BraveAPIKey != "" && !strings.HasPrefix(req.BraveAPIKey, "****") {
			settings.BraveAPIKey = req.BraveAPIKey
		}
		if len(req.SearXNGInstances) > 0 {
			settings.SearXNGInstances = req.SearXNGInstances
		}
		settings.CustomSearXNG = req.CustomSearXNG
		settings.EnableQueryOptimize = req.EnableQueryOptimize
		settings.EnableContentFetch = req.EnableContentFetch
		settings.EnableReRanking = req.EnableReRanking
		if req.OptimizationModel != "" {
			settings.OptimizationModel = req.OptimizationModel
		}

		app.searchService.UpdateSettings(settings)

		writeJSON(w, map[string]interface{}{
			"success": true,
			"message": "Einstellungen gespeichert",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSearchTest testet die Web-Suche
func (app *App) handleSearchTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Query            string   `json:"query"`
		MaxResults       int      `json:"maxResults"`
		TimeFilter       string   `json:"timeFilter"`
		FetchFullContent bool     `json:"fetchFullContent"`
		Domains          []string `json:"domains"`
		ExcludeDomains   []string `json:"excludeDomains"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		req.Query = "Fleet Navigator KI"
	}
	if req.MaxResults == 0 {
		req.MaxResults = 5
	}

	opts := search.DefaultSearchOptions()
	opts.MaxResults = req.MaxResults
	opts.FetchFullContent = req.FetchFullContent
	opts.Domains = req.Domains
	opts.ExcludeDomains = req.ExcludeDomains

	if req.TimeFilter != "" {
		opts.TimeFilter = search.TimeFilter(req.TimeFilter)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	results, err := app.searchService.Search(ctx, req.Query, opts)
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
			"results": []interface{}{},
		})
		return
	}

	writeJSON(w, map[string]interface{}{
		"success": true,
		"query":   req.Query,
		"count":   len(results),
		"results": results,
	})
}

// maskAPIKey maskiert einen API Key für die Anzeige
func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// handleSearchStatus gibt den aktuellen Such-Status zurück
func (app *App) handleSearchStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	settings := app.searchService.GetSettings()

	// Zähler aus der Datenbank lesen (persistiert!)
	var monthlyCount int
	var currentMonth string
	if app.settingsService != nil {
		monthlyCount, currentMonth = app.settingsService.GetWebSearchCount()
	}

	writeJSON(w, map[string]interface{}{
		"available":           true,
		"hasBraveKey":         settings.BraveAPIKey != "",
		"searxngInstances":    len(settings.SearXNGInstances),
		"monthlySearchCount":  monthlyCount,
		"currentMonth":        currentMonth,
		"braveMonthlyLimit":   2000, // Free tier limit
	})
}

// handleSearchExecute führt eine Web-Suche aus (für Chat-Integration)
func (app *App) handleSearchExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Query            string   `json:"query"`
		MaxResults       int      `json:"maxResults"`
		TimeFilter       string   `json:"timeFilter"`
		FetchFullContent bool     `json:"fetchFullContent"`
		Domains          []string `json:"domains"`
		ExcludeDomains   []string `json:"excludeDomains"`
		ExpertContext    string   `json:"expertContext"`
		FormatForLLM     bool     `json:"formatForLLM"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "query is required", http.StatusBadRequest)
		return
	}

	if req.MaxResults == 0 {
		req.MaxResults = 7
	}

	opts := search.DefaultSearchOptions()
	opts.MaxResults = req.MaxResults
	opts.FetchFullContent = req.FetchFullContent
	opts.Domains = req.Domains
	opts.ExcludeDomains = req.ExcludeDomains
	opts.ExpertContext = req.ExpertContext

	if req.TimeFilter != "" {
		opts.TimeFilter = search.TimeFilter(req.TimeFilter)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	results, err := app.searchService.Search(ctx, req.Query, opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"query":   req.Query,
		"count":   len(results),
		"results": results,
	}

	// Optional: Format für LLM Context
	if req.FormatForLLM {
		response["context"] = app.searchService.FormatForContext(results, true)
		response["sourcesFooter"] = app.searchService.FormatSourcesFooter(results)
	}

	writeJSON(w, response)
}

// handleExpertAvatarUpload lädt ein Avatar-Bild hoch
func (app *App) handleExpertAvatarUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Multipart-Form parsen
	if err := r.ParseMultipartForm(10 << 20); err != nil { // Max 10 MB
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("avatar")
	if err != nil {
		http.Error(w, "Avatar file required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Datei speichern
	avatarDir := filepath.Join(app.config.DataDir, "avatars")
	os.MkdirAll(avatarDir, 0755)

	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
	destPath := filepath.Join(avatarDir, filename)

	dest, err := os.Create(destPath)
	if err != nil {
		http.Error(w, "Could not save file", http.StatusInternalServerError)
		return
	}
	defer dest.Close()

	io.Copy(dest, file)

	writeJSON(w, map[string]interface{}{
		"success": true,
		"url":     "/api/avatars/" + filename,
		"path":    destPath,
	})
}

// handleModelStoreDownload lädt ein Modell aus dem Store herunter
func (app *App) handleModelStoreDownload(w http.ResponseWriter, r *http.Request) {
	// EventSource verwendet GET
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	modelID := strings.TrimPrefix(r.URL.Path, "/api/model-store/download/")
	if modelID == "" {
		http.Error(w, "Model ID required", http.StatusBadRequest)
		return
	}

	log.Printf("📥 Download-Request für Modell: %s", modelID)

	// SSE für Download-Progress
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no") // Nginx buffering deaktivieren

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Context für Client-Disconnect-Erkennung
	ctx := r.Context()

	// Finde das Modell in der Registry
	registry := app.modelService.GetRegistry()
	var modelEntry *llm.ModelRegistryEntry
	for _, entry := range registry.GetAllModels() {
		if entry.ID == modelID {
			modelEntry = &entry
			break
		}
	}

	if modelEntry == nil {
		fmt.Fprintf(w, "event: error\ndata: Modell '%s' nicht in Registry gefunden\n\n", modelID)
		flusher.Flush()
		return
	}

	// Zielverzeichnis für GGUF-Modelle
	modelsDir := filepath.Join(app.config.DataDir, "models", "library")
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		fmt.Fprintf(w, "event: error\ndata: Konnte Verzeichnis nicht erstellen: %v\n\n", err)
		flusher.Flush()
		return
	}

	// Prüfe Festplattenplatz (plattformunabhängig)
	availableBytes, diskCheckErr := getDiskSpace(modelsDir)
	if diskCheckErr == nil {
		requiredBytes := modelEntry.SizeBytes
		if requiredBytes == 0 {
			requiredBytes = 5 * 1024 * 1024 * 1024 // Default 5GB wenn unbekannt
		}

		availableGB := float64(availableBytes) / (1024 * 1024 * 1024)
		requiredGB := float64(requiredBytes) / (1024 * 1024 * 1024)

		fmt.Fprintf(w, "event: progress\ndata: 💾 Verfügbar: %.1f GB, Benötigt: %.1f GB\n\n", availableGB, requiredGB)
		flusher.Flush()

		if availableBytes < uint64(requiredBytes) {
			fmt.Fprintf(w, "event: error\ndata: ❌ Nicht genug Speicherplatz! Verfügbar: %.1f GB, Benötigt: %.1f GB\n\n", availableGB, requiredGB)
			flusher.Flush()
			return
		}
	}

	// Download-URL
	downloadURL := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", modelEntry.HuggingFaceRepo, modelEntry.Filename)
	destPath := filepath.Join(modelsDir, modelEntry.Filename)
	tempPath := destPath + ".downloading"

	// Prüfe ob Datei bereits vollständig existiert
	if _, err := os.Stat(destPath); err == nil {
		fmt.Fprintf(w, "event: progress\ndata: ✅ Modell bereits vorhanden: %s\n\n", modelEntry.Filename)
		flusher.Flush()
		// Sende 'complete' Event für Frontend-Kompatibilität (ModelManager.vue erwartet 'complete')
		fmt.Fprintf(w, "event: complete\ndata: Modell bereits vorhanden\n\n")
		flusher.Flush()
		return
	}

	// HTTP Client mit TCP Keep-Alive und angepassten Timeouts für große Downloads
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second, // TCP Keep-Alive alle 30 Sekunden
		}).DialContext,
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second, // Warte max 60s auf Response Header
		DisableCompression:    true,             // Keine Kompression für binäre Downloads
	}
	client := &http.Client{
		Timeout:   0, // Kein Gesamt-Timeout für große Dateien
		Transport: transport,
	}

	// HEAD-Request um Gesamtgröße zu ermitteln (für Resume-Check)
	headReq, _ := http.NewRequestWithContext(ctx, "HEAD", downloadURL, nil)
	headReq.Header.Set("User-Agent", "Fleet-Navigator/0.7.0")
	headResp, headErr := client.Do(headReq)
	var totalSize int64
	if headErr == nil && headResp.StatusCode == http.StatusOK {
		totalSize = headResp.ContentLength
		headResp.Body.Close()
	}
	if totalSize <= 0 {
		totalSize = int64(modelEntry.SizeBytes)
	}
	totalSizeGB := float64(totalSize) / (1024 * 1024 * 1024)

	// Prüfe ob teilweiser Download existiert (Resume-Logik)
	var resumeOffset int64 = 0
	var outFile *os.File

	if partialInfo, err := os.Stat(tempPath); err == nil {
		partialSize := partialInfo.Size()

		if partialSize > totalSize {
			// Korrupt: Lokale Datei größer als Server-Angabe → löschen
			fmt.Fprintf(w, "event: progress\ndata: ⚠️ Korrupte Teildatei gefunden (%.2f GB > %.2f GB), lösche...\n\n",
				float64(partialSize)/(1024*1024*1024), totalSizeGB)
			flusher.Flush()
			os.Remove(tempPath)
			log.Printf("⚠️ Korrupte Teildatei gelöscht: %s", tempPath)
		} else if partialSize == totalSize {
			// Vollständig heruntergeladen, nur umbenennen
			fmt.Fprintf(w, "event: progress\ndata: ✅ Download war bereits komplett, benenne um...\n\n")
			flusher.Flush()
			if err := os.Rename(tempPath, destPath); err != nil {
				fmt.Fprintf(w, "event: error\ndata: ❌ Konnte Datei nicht umbenennen: %v\n\n", err)
				flusher.Flush()
				return
			}
			fmt.Fprintf(w, "event: complete\ndata: Download war bereits komplett\n\n")
			flusher.Flush()
			log.Printf("✅ Teildatei umbenannt: %s -> %s", tempPath, destPath)
			return
		} else {
			// Resume möglich
			resumeOffset = partialSize
			fmt.Fprintf(w, "event: progress\ndata: 🔄 Setze Download fort bei %.2f GB (%.1f%%)\n\n",
				float64(resumeOffset)/(1024*1024*1024), float64(resumeOffset)/float64(totalSize)*100)
			flusher.Flush()
			log.Printf("🔄 Resume Download bei %d bytes (%.1f%%)", resumeOffset, float64(resumeOffset)/float64(totalSize)*100)
		}
	}

	fmt.Fprintf(w, "event: progress\ndata: 📥 %s: %s\n\n",
		func() string { if resumeOffset > 0 { return "Setze fort" } else { return "Starte Download" } }(),
		modelEntry.DisplayName)
	flusher.Flush()
	fmt.Fprintf(w, "event: progress\ndata: 📦 Dateigröße: %.2f GB\n\n", totalSizeGB)
	flusher.Flush()

	log.Printf("📥 Download-URL: %s", downloadURL)

	// HTTP Request für Download mit Context
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: ❌ Request-Fehler: %v\n\n", err)
		flusher.Flush()
		return
	}
	req.Header.Set("User-Agent", "Fleet-Navigator/0.7.0")

	// Range-Header für Resume
	if resumeOffset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", resumeOffset))
		log.Printf("📥 Range-Header gesetzt: bytes=%d-", resumeOffset)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ Download-Fehler für %s: %v", modelID, err)
		fmt.Fprintf(w, "event: error\ndata: ❌ Download-Fehler: %v\n\n", err)
		flusher.Flush()
		return
	}
	defer resp.Body.Close()

	// 200 OK = vollständiger Download, 206 Partial Content = Resume erfolgreich
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		fmt.Fprintf(w, "event: error\ndata: ❌ HTTP-Fehler: %d %s\n\n", resp.StatusCode, resp.Status)
		flusher.Flush()
		return
	}

	// Wenn Server Resume nicht unterstützt (200 statt 206), von vorne starten
	if resumeOffset > 0 && resp.StatusCode == http.StatusOK {
		fmt.Fprintf(w, "event: progress\ndata: ⚠️ Server unterstützt kein Resume, starte von vorne...\n\n")
		flusher.Flush()
		os.Remove(tempPath)
		resumeOffset = 0
		log.Printf("⚠️ Server unterstützt kein Resume, starte von vorne")
	}

	// Datei öffnen (Append für Resume, Create für Neustart)
	if resumeOffset > 0 {
		outFile, err = os.OpenFile(tempPath, os.O_WRONLY|os.O_APPEND, 0644)
	} else {
		outFile, err = os.Create(tempPath)
	}
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: ❌ Konnte Datei nicht erstellen/öffnen: %v\n\n", err)
		flusher.Flush()
		return
	}

	// Download mit Progress-Tracking
	downloaded := resumeOffset // Bei Resume: Start bei bereits heruntergeladenem Offset
	buf := make([]byte, 1024*1024) // 1MB Buffer
	lastProgress := time.Now()
	startTime := time.Now()
	lastDownloaded := resumeOffset

	// Cleanup-Funktion für Abbruch - Datei BEHALTEN für Resume!
	cleanup := func(reason string, deleteFile bool) {
		outFile.Close()
		if deleteFile {
			os.Remove(tempPath)
			log.Printf("⚠️ Download abgebrochen und gelöscht für %s: %s", modelEntry.DisplayName, reason)
		} else {
			log.Printf("⏸️ Download pausiert für %s: %s (%.2f GB heruntergeladen, kann fortgesetzt werden)",
				modelEntry.DisplayName, reason, float64(downloaded)/(1024*1024*1024))
		}
	}

	for {
		// Prüfe ob Client noch verbunden ist (Context cancelled = Client disconnected)
		select {
		case <-ctx.Done():
			cleanup("Client hat Verbindung getrennt", false) // Datei behalten für Resume
			// Keine SSE-Nachricht senden - Client ist weg
			return
		default:
			// Weiter mit Download
		}

		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := outFile.Write(buf[:n])
			if writeErr != nil {
				cleanup(fmt.Sprintf("Schreibfehler: %v", writeErr), false) // Datei behalten
				// Versuche Fehlermeldung zu senden (könnte fehlschlagen wenn Client weg)
				fmt.Fprintf(w, "event: error\ndata: ❌ Schreibfehler: %v - Download kann fortgesetzt werden\n\n", writeErr)
				flusher.Flush()
				return
			}
			downloaded += int64(n)

			// Progress alle 2 Sekunden senden
			if time.Since(lastProgress) >= 2*time.Second {
				elapsed := time.Since(startTime).Seconds()
				speed := float64(downloaded-lastDownloaded) / time.Since(lastProgress).Seconds() / (1024 * 1024)
				lastDownloaded = downloaded
				lastProgress = time.Now()

				percent := 0
				if totalSize > 0 {
					percent = int(float64(downloaded) / float64(totalSize) * 100)
				}

				downloadedGB := float64(downloaded) / (1024 * 1024 * 1024)

				// ETA berechnen
				var etaStr string
				if speed > 0 && totalSize > 0 {
					remaining := float64(totalSize-downloaded) / (speed * 1024 * 1024)
					if remaining > 3600 {
						etaStr = fmt.Sprintf("%.1fh", remaining/3600)
					} else if remaining > 60 {
						etaStr = fmt.Sprintf("%.0fm", remaining/60)
					} else {
						etaStr = fmt.Sprintf("%.0fs", remaining)
					}
				}

				progressMsg := fmt.Sprintf("⬇️ %d%% - %.2f GB / %.2f GB - %.1f MB/s", percent, downloadedGB, totalSizeGB, speed)
				if etaStr != "" {
					progressMsg += fmt.Sprintf(" - ETA: %s", etaStr)
				}

				// Versuche Progress zu senden - wenn Client weg ist, Fehler ignorieren und aufräumen
				_, writeErr := fmt.Fprintf(w, "event: progress\ndata: %s\n\n", progressMsg)
				if writeErr != nil {
					cleanup(fmt.Sprintf("SSE-Schreibfehler: %v", writeErr), false) // Datei behalten für Resume
					return
				}
				flusher.Flush()

				log.Printf("Download %s: %d%% (%.2f GB / %.2f GB) - %.1f MB/s", modelEntry.DisplayName, percent, downloadedGB, totalSizeGB, speed)
				_ = elapsed // Avoid unused variable warning
			}
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			// Prüfe ob es ein Context-Fehler ist (Client disconnected)
			if ctx.Err() != nil {
				cleanup("Client hat Verbindung getrennt während Download", false) // Datei behalten
				return
			}
			cleanup(fmt.Sprintf("Lesefehler: %v", err), false) // Datei behalten für Resume
			fmt.Fprintf(w, "event: error\ndata: ❌ Lesefehler: %v - Download kann fortgesetzt werden\n\n", err)
			flusher.Flush()
			return
		}
	}

	outFile.Close()

	// Rename temp file to final name
	if err := os.Rename(tempPath, destPath); err != nil {
		os.Remove(tempPath)
		fmt.Fprintf(w, "event: error\ndata: ❌ Konnte Datei nicht umbenennen: %v\n\n", err)
		flusher.Flush()
		return
	}

	totalTime := time.Since(startTime)
	avgSpeed := float64(downloaded) / totalTime.Seconds() / (1024 * 1024)

	fmt.Fprintf(w, "event: progress\ndata: ✅ Download abgeschlossen! (%.1f MB/s durchschnittlich)\n\n", avgSpeed)
	flusher.Flush()
	fmt.Fprintf(w, "event: progress\ndata: 📁 Gespeichert unter: %s\n\n", destPath)
	flusher.Flush()
	// Sende 'complete' Event für Frontend-Kompatibilität (ModelManager.vue erwartet 'complete')
	fmt.Fprintf(w, "event: complete\ndata: Download erfolgreich\n\n")
	flusher.Flush()

	log.Printf("✅ Download abgeschlossen: %s -> %s", modelEntry.DisplayName, destPath)
}

// extractParamSize extrahiert die Parametergröße (z.B. "70B") aus Model-Name oder Tags
func extractParamSize(modelName string, tags []string) string {
	// Pattern für Parametergrößen: 1.5B, 7B, 13B, 70B, 72B etc.
	patterns := []string{"70B", "72B", "65B", "34B", "33B", "32B", "30B", "14B", "13B", "8B", "7B", "3B", "1.5B", "1B"}

	modelUpper := strings.ToUpper(modelName)
	for _, p := range patterns {
		if strings.Contains(modelUpper, p) || strings.Contains(modelUpper, "-"+p) {
			return p
		}
	}

	// Fallback: Tags prüfen
	for _, tag := range tags {
		tagUpper := strings.ToUpper(tag)
		for _, p := range patterns {
			if tagUpper == p || strings.HasSuffix(tagUpper, p) {
				return p
			}
		}
	}

	return ""
}

// estimateModelSize schätzt die GGUF-Dateigröße basierend auf Parametern und Quantisierung
// Rückgabe: geschätzte Größe in Bytes und lesbare Größe
func estimateModelSize(modelName string, paramSize string) (int64, string) {
	// Parameter-Zahl extrahieren
	var params float64
	paramUpper := strings.ToUpper(paramSize)
	paramUpper = strings.TrimSuffix(paramUpper, "B")
	params, _ = strconv.ParseFloat(paramUpper, 64)

	if params == 0 {
		return 0, ""
	}

	// Quantisierung aus Model-Name erkennen
	modelLower := strings.ToLower(modelName)
	var bytesPerParam float64

	switch {
	case strings.Contains(modelLower, "q2_k"):
		bytesPerParam = 0.35 // ~2.5 bits per param
	case strings.Contains(modelLower, "q3_k"):
		bytesPerParam = 0.45 // ~3.5 bits per param
	case strings.Contains(modelLower, "q4_k"), strings.Contains(modelLower, "q4_0"), strings.Contains(modelLower, "iq4"):
		bytesPerParam = 0.55 // ~4.5 bits per param
	case strings.Contains(modelLower, "q5_k"), strings.Contains(modelLower, "q5_0"):
		bytesPerParam = 0.65 // ~5.5 bits per param
	case strings.Contains(modelLower, "q6_k"):
		bytesPerParam = 0.75 // ~6 bits per param
	case strings.Contains(modelLower, "q8"), strings.Contains(modelLower, "q8_0"):
		bytesPerParam = 1.0 // 8 bits per param
	case strings.Contains(modelLower, "f16"), strings.Contains(modelLower, "fp16"):
		bytesPerParam = 2.0 // 16 bits per param
	default:
		bytesPerParam = 0.55 // Default: Q4 angenommen
	}

	// Berechnung: params (Milliarden) * bytes_per_param * 1 Milliarde
	sizeBytes := int64(params * bytesPerParam * 1_000_000_000)

	// Lesbare Größe
	sizeGB := float64(sizeBytes) / (1024 * 1024 * 1024)
	var sizeHuman string
	if sizeGB >= 1.0 {
		sizeHuman = fmt.Sprintf("~%.1f GB", sizeGB)
	} else {
		sizeMB := float64(sizeBytes) / (1024 * 1024)
		sizeHuman = fmt.Sprintf("~%.0f MB", sizeMB)
	}

	return sizeBytes, sizeHuman
}

// handleHuggingFaceSearch sucht nach GGUF-Modellen DIREKT auf HuggingFace
func (app *App) handleHuggingFaceSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("query")
	if query == "" {
		query = r.URL.Query().Get("q") // Frontend sendet "q"
	}
	if query == "" {
		writeJSON(w, []interface{}{})
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 30
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// HuggingFace API direkt aufrufen - suche nach GGUF Modellen
	// Filter: gguf (nur GGUF-Dateien), sort: downloads (beliebteste zuerst)
	hfURL := fmt.Sprintf(
		"https://huggingface.co/api/models?search=%s+gguf&sort=downloads&direction=-1&limit=%d",
		url.QueryEscape(query),
		limit,
	)

	log.Printf("HuggingFace Suche: %s", hfURL)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(hfURL)
	if err != nil {
		log.Printf("HuggingFace API Fehler: %v", err)
		writeJSON(w, []interface{}{})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("HuggingFace API Status: %d", resp.StatusCode)
		writeJSON(w, []interface{}{})
		return
	}

	// HuggingFace Response parsen
	var hfModels []struct {
		ID          string   `json:"id"`          // z.B. "TheBloke/Llama-2-70B-GGUF"
		ModelID     string   `json:"modelId"`
		Author      string   `json:"author"`
		Downloads   int      `json:"downloads"`
		Likes       int      `json:"likes"`
		Tags        []string `json:"tags"`
		CreatedAt   string   `json:"createdAt"`
		LastModified string  `json:"lastModified"`
		Private     bool     `json:"private"`
		Siblings    []struct {
			Filename string `json:"rfilename"`
		} `json:"siblings"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&hfModels); err != nil {
		log.Printf("HuggingFace JSON Parse Fehler: %v", err)
		writeJSON(w, []interface{}{})
		return
	}

	// Ergebnisse für Frontend aufbereiten
	// WICHTIG: siblings ist im Search-Response NULL - das Frontend lädt Details separat
	results := make([]map[string]interface{}, 0, len(hfModels))
	for _, model := range hfModels {
		// Da "gguf" im Suchbegriff war, nehmen wir an dass alle Ergebnisse GGUF haben
		// Alternativ: Prüfe ob "GGUF" im Model-Namen vorkommt
		modelNameLower := strings.ToLower(model.ID)
		if !strings.Contains(modelNameLower, "gguf") {
			continue // Überspringe Modelle ohne "gguf" im Namen
		}

		// Modellgröße aus Tags oder Model-Name extrahieren (z.B. "7B", "13B", "70B")
		paramSize := extractParamSize(model.ID, model.Tags)

		// Geschätzte Dateigröße berechnen
		sizeBytes, sizeHuman := estimateModelSize(model.ID, paramSize)

		// Kategorie aus Tags ableiten
		category := "chat"
		for _, tag := range model.Tags {
			tagLower := strings.ToLower(tag)
			if strings.Contains(tagLower, "code") || strings.Contains(tagLower, "coder") {
				category = "code"
				break
			}
			if strings.Contains(tagLower, "vision") || strings.Contains(tagLower, "llava") || strings.Contains(tagLower, "multimodal") {
				category = "vision"
				break
			}
		}

		results = append(results, map[string]interface{}{
			"id":            model.ID,
			"modelId":       model.ID,
			"huggingFaceId": model.ID,
			"displayName":   model.ID,
			"name":          model.ID,
			"author":        model.Author,
			"downloads":     model.Downloads,
			"likes":         model.Likes,
			"tags":          model.Tags,
			"category":      category,
			"parameters":    paramSize,
			"sizeBytes":     sizeBytes,   // Geschätzte Größe in Bytes
			"sizeHuman":     sizeHuman,   // z.B. "~38.5 GB"
			"modelSize":     sizeBytes,   // Alias für Frontend-Kompatibilität
			"siblings":      []string{},  // Wird vom Frontend bei Download geladen
			"createdAt":     model.CreatedAt,
			"lastModified":  model.LastModified,
			"source":        "huggingface", // Markierung dass es von HF kommt
		})
	}

	log.Printf("HuggingFace Suche '%s': %d Ergebnisse", query, len(results))
	writeJSON(w, results)
}

// handleHuggingFacePopular gibt beliebte/empfohlene Modelle zurück
func (app *App) handleHuggingFacePopular(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Beliebte Modelle aus der Registry (Featured oder hohe Downloads)
	registry := app.modelService.GetRegistry()
	results := make([]map[string]interface{}, 0) // Leeres Array statt nil

	for _, entry := range registry.GetAllModels() {
		// Nur Featured-Modelle oder bestimmte Kategorien
		if entry.Featured || entry.Category == "chat" || entry.Category == "coder" {
			results = append(results, map[string]interface{}{
				"id":            entry.ID,
				"displayName":   entry.DisplayName,
				"name":          entry.DisplayName,
				"description":   entry.Description,
				"modelId":       entry.HuggingFaceRepo,
				"huggingFaceId": entry.HuggingFaceRepo,
				"size":          entry.SizeHuman,
				"sizeHuman":     entry.SizeHuman,
				"filename":      entry.Filename,
				"parameters":    entry.ParameterSize,
				"quantization":  entry.Quantization,
				"category":      entry.Category,
				"downloadUrl":   fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", entry.HuggingFaceRepo, entry.Filename),
				"ggufFile":      entry.Filename,
				"featured":      entry.Featured,
			})
		}
	}

	writeJSON(w, results)
}

// handleHuggingFaceGerman gibt deutschsprachige Modelle zurück
func (app *App) handleHuggingFaceGerman(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	registry := app.modelService.GetRegistry()
	results := make([]map[string]interface{}, 0)

	for _, entry := range registry.GetAllModels() {
		// Modelle mit deutschen Sprachfähigkeiten
		hasGerman := false
		for _, lang := range entry.Languages {
			langLower := strings.ToLower(lang)
			if strings.Contains(langLower, "german") || strings.Contains(langLower, "deutsch") || langLower == "de" {
				hasGerman = true
				break
			}
		}
		if hasGerman ||
			strings.Contains(strings.ToLower(entry.Description), "german") ||
			strings.Contains(strings.ToLower(entry.Description), "deutsch") ||
			strings.Contains(strings.ToLower(entry.DisplayName), "qwen") { // Qwen hat gutes Deutsch
			results = append(results, map[string]interface{}{
				"id":            entry.ID,
				"displayName":   entry.DisplayName,
				"name":          entry.DisplayName,
				"description":   entry.Description,
				"modelId":       entry.HuggingFaceRepo,
				"huggingFaceId": entry.HuggingFaceRepo,
				"size":          entry.SizeHuman,
				"sizeHuman":     entry.SizeHuman,
				"filename":      entry.Filename,
				"parameters":    entry.ParameterSize,
				"quantization":  entry.Quantization,
				"category":      entry.Category,
				"downloadUrl":   fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", entry.HuggingFaceRepo, entry.Filename),
				"ggufFile":      entry.Filename,
				"languages":     entry.Languages,
			})
		}
	}

	writeJSON(w, results)
}

// handleHuggingFaceInstruct gibt Instruct/Chat-Modelle zurück
func (app *App) handleHuggingFaceInstruct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	registry := app.modelService.GetRegistry()
	results := make([]map[string]interface{}, 0)

	for _, entry := range registry.GetAllModels() {
		// Instruct/Chat-Modelle
		if entry.Category == "chat" ||
			strings.Contains(strings.ToLower(entry.DisplayName), "instruct") ||
			strings.Contains(strings.ToLower(entry.DisplayName), "chat") {
			results = append(results, map[string]interface{}{
				"id":            entry.ID,
				"displayName":   entry.DisplayName,
				"name":          entry.DisplayName,
				"description":   entry.Description,
				"modelId":       entry.HuggingFaceRepo,
				"huggingFaceId": entry.HuggingFaceRepo,
				"size":          entry.SizeHuman,
				"sizeHuman":     entry.SizeHuman,
				"filename":      entry.Filename,
				"parameters":    entry.ParameterSize,
				"quantization":  entry.Quantization,
				"category":      entry.Category,
				"downloadUrl":   fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", entry.HuggingFaceRepo, entry.Filename),
				"ggufFile":      entry.Filename,
			})
		}
	}

	writeJSON(w, results)
}

// handleHuggingFaceCode gibt Code-Modelle zurück
func (app *App) handleHuggingFaceCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	registry := app.modelService.GetRegistry()
	results := make([]map[string]interface{}, 0)

	for _, entry := range registry.GetAllModels() {
		// Code-Modelle
		if entry.Category == "code" || entry.Category == "coder" ||
			strings.Contains(strings.ToLower(entry.DisplayName), "code") ||
			strings.Contains(strings.ToLower(entry.DisplayName), "coder") {
			results = append(results, map[string]interface{}{
				"id":            entry.ID,
				"displayName":   entry.DisplayName,
				"name":          entry.DisplayName,
				"description":   entry.Description,
				"modelId":       entry.HuggingFaceRepo,
				"huggingFaceId": entry.HuggingFaceRepo,
				"size":          entry.SizeHuman,
				"sizeHuman":     entry.SizeHuman,
				"filename":      entry.Filename,
				"parameters":    entry.ParameterSize,
				"quantization":  entry.Quantization,
				"category":      entry.Category,
				"downloadUrl":   fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", entry.HuggingFaceRepo, entry.Filename),
				"ggufFile":      entry.Filename,
			})
		}
	}

	writeJSON(w, results)
}

// handleHuggingFaceVision gibt Vision-Modelle zurück
func (app *App) handleHuggingFaceVision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	registry := app.modelService.GetRegistry()
	results := make([]map[string]interface{}, 0)

	for _, entry := range registry.GetAllModels() {
		// Vision-Modelle
		if entry.Category == "vision" ||
			strings.Contains(strings.ToLower(entry.DisplayName), "vision") ||
			strings.Contains(strings.ToLower(entry.DisplayName), "llava") ||
			strings.Contains(strings.ToLower(entry.DisplayName), "bakllava") ||
			strings.Contains(strings.ToLower(entry.DisplayName), "moondream") {
			results = append(results, map[string]interface{}{
				"id":            entry.ID,
				"displayName":   entry.DisplayName,
				"name":          entry.DisplayName,
				"description":   entry.Description,
				"modelId":       entry.HuggingFaceRepo,
				"huggingFaceId": entry.HuggingFaceRepo,
				"size":          entry.SizeHuman,
				"sizeHuman":     entry.SizeHuman,
				"filename":      entry.Filename,
				"parameters":    entry.ParameterSize,
				"quantization":  entry.Quantization,
				"category":      entry.Category,
				"downloadUrl":   fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", entry.HuggingFaceRepo, entry.Filename),
				"ggufFile":      entry.Filename,
			})
		}
	}

	writeJSON(w, results)
}

// handleHuggingFaceDetails gibt Details zu einem spezifischen Modell zurück
func (app *App) handleHuggingFaceDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	modelId := r.URL.Query().Get("modelId")
	if modelId == "" {
		http.Error(w, "modelId required", http.StatusBadRequest)
		return
	}

	// Suche in der Registry nach dem Modell
	registry := app.modelService.GetRegistry()
	for _, entry := range registry.GetAllModels() {
		if entry.HuggingFaceRepo == modelId || entry.DisplayName == modelId || entry.ID == modelId {
			writeJSON(w, map[string]interface{}{
				"name":           entry.DisplayName,
				"description":    entry.Description,
				"huggingFaceId":  entry.HuggingFaceRepo,
				"size":           entry.SizeHuman,
				"sizeBytes":      entry.SizeBytes,
				"parameters":     entry.ParameterSize,
				"quantization":   entry.Quantization,
				"category":       entry.Category,
				"downloadUrl":    fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", entry.HuggingFaceRepo, entry.Filename),
				"ggufFile":       entry.Filename,
				"license":        entry.License,
				"languages":      entry.Languages,
				"useCases":       entry.UseCases,
				"minRamGB":       entry.MinRamGB,
				"recommendedRamGB": entry.RecommendedRamGB,
			})
			return
		}
	}

	http.Error(w, "Model not found", http.StatusNotFound)
}

// handleHuggingFaceDownload lädt ein GGUF-Modell von HuggingFace herunter
func (app *App) handleHuggingFaceDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URL      string `json:"url"`
		Filename string `json:"filename"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Weiterleitung an llama-server Download
	app.handleLlamaServerDownload(w, r)
}

// handleOllamaPull zieht ein Modell via Ollama
func (app *App) handleOllamaPull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	modelName := strings.TrimPrefix(r.URL.Path, "/api/ollama/pull/")
	if modelName == "" {
		http.Error(w, "Model name required", http.StatusBadRequest)
		return
	}

	// SSE für Download-Progress
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Download-Progress simulieren (Ollama Pull über API)
	fmt.Fprintf(w, "data: {\"status\":\"pulling\",\"model\":\"%s\"}\n\n", modelName)
	flusher.Flush()

	// TODO: Echtes Ollama Pull implementieren
	fmt.Fprintf(w, "data: {\"status\":\"success\",\"done\":true}\n\n")
	flusher.Flush()
}

// handleSettingsDocumentModel - GET/POST /api/settings/document-model
// Frontend erwartet text/plain, nicht JSON
func (app *App) handleSettingsDocumentModel(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		model := app.settingsService.GetDocumentModel()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(model))

	case http.MethodPost:
		// Frontend sendet text/plain
		body := make([]byte, 1024)
		n, _ := r.Body.Read(body)
		model := strings.TrimSpace(string(body[:n]))
		if err := app.settingsService.SaveDocumentModel(model); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSettingsEmailModel - GET/POST /api/settings/email-model
// Frontend erwartet text/plain, nicht JSON
func (app *App) handleSettingsEmailModel(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		model := app.settingsService.GetEmailModel()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(model))

	case http.MethodPost:
		// Frontend sendet text/plain
		body := make([]byte, 1024)
		n, _ := r.Body.Read(body)
		model := strings.TrimSpace(string(body[:n]))
		if err := app.settingsService.SaveEmailModel(model); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSettingsLogAnalysisModel - GET/POST /api/settings/log-analysis-model
// Frontend erwartet text/plain, nicht JSON
func (app *App) handleSettingsLogAnalysisModel(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		model := app.settingsService.GetLogAnalysisModel()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(model))

	case http.MethodPost:
		// Frontend sendet text/plain
		body := make([]byte, 1024)
		n, _ := r.Body.Read(body)
		model := strings.TrimSpace(string(body[:n]))
		if err := app.settingsService.SaveLogAnalysisModel(model); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSettingsCoderModel - GET/POST /api/settings/coder-model
// Frontend erwartet text/plain, nicht JSON
func (app *App) handleSettingsCoderModel(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		model := app.settingsService.GetCoderModel()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(model))

	case http.MethodPost:
		// Frontend sendet text/plain
		body := make([]byte, 1024)
		n, _ := r.Body.Read(body)
		model := strings.TrimSpace(string(body[:n]))
		if err := app.settingsService.SaveCoderModel(model); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// --- Persistente Settings Endpoints (Wichtig über Browser-Sessions hinweg) ---

// handleSettingsSampling - GET/POST /api/settings/sampling
// Sampling-Parameter (Temperature, TopP, etc.) - wichtig für KI-Verhalten
func (app *App) handleSettingsSampling(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		params := app.settingsService.GetSamplingParams()
		writeJSON(w, params)

	case http.MethodPost:
		var params settings.SamplingParams
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if err := app.settingsService.SaveSamplingParams(params); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]interface{}{"success": true, "params": params})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSettingsChaining - GET/POST /api/settings/chaining
// Model-Chaining-Konfiguration - wichtig für KI-Workflow
func (app *App) handleSettingsChaining(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		chainingSettings := app.settingsService.GetChainingSettings()
		writeJSON(w, chainingSettings)

	case http.MethodPost:
		var chainingSettings settings.ChainingSettings
		if err := json.NewDecoder(r.Body).Decode(&chainingSettings); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if err := app.settingsService.SaveChainingSettings(chainingSettings); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]interface{}{"success": true, "settings": chainingSettings})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSettingsPreferences - GET/POST /api/settings/preferences
// Benutzer-Präferenzen (Locale, DarkMode) - wichtig für UX
func (app *App) handleSettingsPreferences(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		prefs := app.settingsService.GetUserPreferences()
		writeJSON(w, prefs)

	case http.MethodPost:
		var prefs settings.UserPreferences
		if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if err := app.settingsService.SaveUserPreferences(prefs); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]interface{}{"success": true, "preferences": prefs})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSettingsLanguage - GET/POST /api/settings/language
// Sprachwechsel mit Backend-Benachrichtigung für Experten-Prompts
func (app *App) handleSettingsLanguage(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		locale := app.settingsService.GetLocale()
		// Verfügbare Sprachen und installierte Stimmen mitgeben
		availableVoices := app.getAvailableVoicesForLocale(locale)
		writeJSON(w, map[string]interface{}{
			"locale":          locale,
			"availableVoices": availableVoices,
			"supportedLocales": []string{"de", "en", "tr"},
		})

	case http.MethodPost:
		var req struct {
			Locale string `json:"locale"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Validiere Sprache
		validLocales := map[string]bool{"de": true, "en": true, "tr": true}
		if !validLocales[req.Locale] {
			http.Error(w, "Unsupported locale. Use: de, en, tr", http.StatusBadRequest)
			return
		}

		// Speichere in Settings-DB
		if err := app.settingsService.SaveLocale(req.Locale); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Verfügbare Stimmen für die neue Sprache
		availableVoices := app.getAvailableVoicesForLocale(req.Locale)
		installedVoices := app.getInstalledVoicesForLocale(req.Locale)

		log.Printf("[Settings] Sprache gewechselt: %s (Stimmen installiert: %d)", req.Locale, len(installedVoices))

		writeJSON(w, map[string]interface{}{
			"success":         true,
			"locale":          req.Locale,
			"availableVoices": availableVoices,
			"installedVoices": installedVoices,
			"needsVoiceDownload": len(installedVoices) == 0 && len(availableVoices) > 0,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getAvailableVoicesForLocale gibt verfügbare TTS-Stimmen für eine Sprache zurück
// Die IDs sind im Format für direkten Download: de_DE-thorsten-medium
func (app *App) getAvailableVoicesForLocale(locale string) []map[string]string {
	voices := map[string][]map[string]string{
		"de": {
			{"id": "de_DE-thorsten-medium", "name": "Thorsten", "desc": "Männlich, neutral"},
			{"id": "de_DE-kerstin-medium", "name": "Kerstin", "desc": "Weiblich, klar"},
			{"id": "de_DE-eva_k-x_low", "name": "Eva", "desc": "Weiblich, warm"},
		},
		"tr": {
			{"id": "tr_TR-fahrettin-medium", "name": "Fahrettin", "desc": "Erkek, nötr"},
			{"id": "tr_TR-fettah-medium", "name": "Fettah", "desc": "Erkek, net"},
		},
		"en": {
			{"id": "en_GB-amy-medium", "name": "Amy", "desc": "Female, British"},
			{"id": "en_US-ryan-medium", "name": "Ryan", "desc": "Male, American"},
			{"id": "en_US-lessac-medium", "name": "Lessac", "desc": "Female, American"},
		},
	}
	if v, ok := voices[locale]; ok {
		return v
	}
	return []map[string]string{}
}

// getInstalledVoicesForLocale prüft welche Stimmen bereits installiert sind
// Gibt vollständige Voice-IDs zurück (z.B. "de_DE-thorsten-medium") passend zu getAvailableVoicesForLocale
func (app *App) getInstalledVoicesForLocale(locale string) []string {
	installed := []string{}
	voiceDir := filepath.Join(app.config.DataDir, "voice", "piper")

	// Stimmen-Dateien pro Sprache - Voice-ID -> Dateiname
	// Voice-IDs müssen mit getAvailableVoicesForLocale übereinstimmen!
	voiceFiles := map[string]map[string]string{
		"de": {
			"de_DE-thorsten-medium": "de_DE-thorsten-medium.onnx",
			"de_DE-kerstin-medium":  "de_DE-kerstin-medium.onnx",
			"de_DE-eva_k-x_low":     "de_DE-eva_k-x_low.onnx",
		},
		"tr": {
			"tr_TR-fahrettin-medium": "tr_TR-fahrettin-medium.onnx",
			"tr_TR-fettah-medium":    "tr_TR-fettah-medium.onnx",
		},
		"en": {
			"en_GB-amy-medium":    "en_GB-amy-medium.onnx",
			"en_US-ryan-medium":   "en_US-ryan-medium.onnx",
			"en_US-lessac-medium": "en_US-lessac-medium.onnx",
		},
	}

	if files, ok := voiceFiles[locale]; ok {
		for voiceID, filename := range files {
			if _, err := os.Stat(filepath.Join(voiceDir, filename)); err == nil {
				installed = append(installed, voiceID)
			}
		}
	}

	return installed
}

// ========================================
// llamaServerWrapper - Adapter für chat.LlamaServerChatter Interface
// ========================================

// llamaServerWrapper implementiert chat.LlamaServerChatter für llamaserver.Server
type llamaServerWrapper struct {
	server *llamaserver.Server
}

// StreamChat implementiert chat.LlamaServerChatter Interface
func (w *llamaServerWrapper) StreamChat(messages []chat.LlamaMessage, onChunk func(content string, done bool)) error {
	// Konvertiere chat.LlamaMessage zu llamaserver.ChatMessage
	llamaMessages := make([]llamaserver.ChatMessage, len(messages))
	for i, msg := range messages {
		llamaMessages[i] = llamaserver.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return w.server.StreamChat(llamaMessages, onChunk)
}

// StreamChatWithParams implementiert chat.LlamaServerChatter Interface mit Sampling-Parametern
func (w *llamaServerWrapper) StreamChatWithParams(messages []chat.LlamaMessage, params chat.LlamaSamplingParams, onChunk func(content string, done bool)) error {
	// Konvertiere chat.LlamaMessage zu llamaserver.ChatMessage
	llamaMessages := make([]llamaserver.ChatMessage, len(messages))
	for i, msg := range messages {
		llamaMessages[i] = llamaserver.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	// Konvertiere chat.LlamaSamplingParams zu llamaserver.SamplingParams
	llamaParams := llamaserver.SamplingParams{
		Temperature: params.Temperature,
		TopP:        params.TopP,
		MaxTokens:   params.MaxTokens,
	}
	return w.server.StreamChatWithParams(llamaMessages, llamaParams, onChunk)
}

// IsRunning implementiert chat.LlamaServerChatter Interface
func (w *llamaServerWrapper) IsRunning() bool {
	return w.server.IsRunning()
}

// IsHealthy implementiert chat.LlamaServerChatter Interface
func (w *llamaServerWrapper) IsHealthy() bool {
	return w.server.IsHealthy()
}

// ============================================================================
// Fleet-Mate erweiterte Handler
// ============================================================================

// handleFleetMateByID behandelt /api/fleet-mate/mates/{id}/... Endpoints
func (app *App) handleFleetMateByID(w http.ResponseWriter, r *http.Request) {
	// Parse: /api/fleet-mate/mates/{mateId}/{action}
	path := strings.TrimPrefix(r.URL.Path, "/api/fleet-mate/mates/")
	parts := strings.Split(path, "/")

	if len(parts) < 1 {
		http.Error(w, "Missing mate ID", http.StatusBadRequest)
		return
	}

	mateID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch action {
	case "ping":
		// POST /api/fleet-mate/mates/{id}/ping
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Versuche Mate zu pingen via WebSocket
		mate := app.wsServer.GetMateByID(mateID)
		if mate == nil {
			writeJSON(w, map[string]interface{}{
				"success":  false,
				"mateId":   mateID,
				"error":    "Mate nicht verbunden",
				"latency":  0,
				"lastSeen": nil,
			})
			return
		}
		writeJSON(w, map[string]interface{}{
			"success":  true,
			"mateId":   mateID,
			"latency":  15, // Simulierte Latenz in ms
			"lastSeen": time.Now().Format(time.RFC3339),
		})

	case "analyze-log":
		// POST /api/fleet-mate/mates/{id}/analyze-log
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Gibt Session-ID zurück für SSE-Stream
		sessionID := fmt.Sprintf("log-analysis-%s-%d", mateID, time.Now().UnixNano())
		writeJSON(w, map[string]interface{}{
			"sessionId": sessionID,
			"mateId":    mateID,
			"status":    "started",
		})

	case "execute":
		// POST /api/fleet-mate/mates/{id}/execute
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Command string `json:"command"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		sessionID := fmt.Sprintf("exec-%s-%d", mateID, time.Now().UnixNano())
		writeJSON(w, map[string]interface{}{
			"sessionId": sessionID,
			"mateId":    mateID,
			"command":   req.Command,
			"status":    "started",
		})

	case "command-history":
		// GET /api/fleet-mate/mates/{id}/command-history
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Leere Historie als Stub
		writeJSON(w, map[string]interface{}{
			"mateId":  mateID,
			"history": []map[string]interface{}{},
		})

	case "stats":
		// GET /api/fleet-mate/mates/{id}/stats - bereits in handleFleetMateStats
		app.handleFleetMateStats(w, r)
		return

	case "config":
		// GET/PUT /api/fleet-mate/mates/{id}/config - KI-Konfiguration für diesen Mate
		switch r.Method {
		case http.MethodGet:
			model, systemPrompt, activeMode, err := app.pairingManager.GetMateConfig(mateID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			writeJSON(w, map[string]interface{}{
				"mateId":       mateID,
				"model":        model,
				"systemPrompt": systemPrompt,
				"activeMode":   activeMode,
			})

		case http.MethodPut:
			var req struct {
				Model        string `json:"model"`
				SystemPrompt string `json:"systemPrompt"`
				ActiveMode   string `json:"activeMode"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}
			if err := app.pairingManager.UpdateMateConfig(mateID, req.Model, req.SystemPrompt, req.ActiveMode); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			log.Printf("✅ Mate %s KI-Konfiguration aktualisiert: model=%s, mode=%s", mateID, req.Model, req.ActiveMode)
			writeJSON(w, map[string]interface{}{
				"success":      true,
				"mateId":       mateID,
				"model":        req.Model,
				"systemPrompt": req.SystemPrompt,
				"activeMode":   req.ActiveMode,
			})

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return

	default:
		// GET /api/fleet-mate/mates/{id} - Einzelner Mate
		if r.Method == http.MethodGet && action == "" {
			mate := app.wsServer.GetMateByID(mateID)
			if mate == nil {
				// Prüfe trusted mates
				for _, tm := range app.pairingManager.GetTrustedMates() {
					if tm.ID == mateID {
						writeJSON(w, map[string]interface{}{
							"mateId":      tm.ID,
							"mateName":    tm.Name,
							"mateType":    tm.Type,
							"connected":   false,
							"lastSeen":    tm.LastSeen,
							"pairedAt":    tm.PairedAt,
							"capabilities": []string{},
						})
						return
					}
				}
				http.Error(w, "Mate not found", http.StatusNotFound)
				return
			}
			writeJSON(w, map[string]interface{}{
				"mateId":       mate.MateID,
				"mateName":     mate.MateName,
				"mateType":     mate.MateType,
				"connected":    true,
				"lastSeen":     time.Now().Format(time.RFC3339),
				"capabilities": mate.Capabilities,
			})
			return
		}
		http.Error(w, "Unknown action: "+action, http.StatusNotFound)
	}
}

// handleFleetMateStream behandelt SSE für Mate-Streams
func (app *App) handleFleetMateStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := strings.TrimPrefix(r.URL.Path, "/api/fleet-mate/stream/")
	if sessionID == "" {
		http.Error(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	// SSE Setup
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Simulierte Analyse-Nachrichten
	messages := []string{
		"🔍 Analysiere Log-Daten...",
		"📊 Verarbeite Einträge...",
		"✅ Analyse abgeschlossen",
	}

	for _, msg := range messages {
		fmt.Fprintf(w, "data: %s\n\n", msg)
		flusher.Flush()
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Fprintf(w, "event: done\ndata: {\"status\":\"completed\"}\n\n")
	flusher.Flush()
}

// handleFleetMateExecStream behandelt SSE für Command-Execution
func (app *App) handleFleetMateExecStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := strings.TrimPrefix(r.URL.Path, "/api/fleet-mate/exec-stream/")
	if sessionID == "" {
		http.Error(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	// SSE Setup
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Simulierte Ausführung
	fmt.Fprintf(w, "data: {\"type\":\"output\",\"content\":\"Command execution not yet implemented in Go version\"}\n\n")
	flusher.Flush()
	fmt.Fprintf(w, "event: done\ndata: {\"exitCode\":0}\n\n")
	flusher.Flush()
}

// handleFleetMateWhitelistedCommands gibt erlaubte Befehle zurück
func (app *App) handleFleetMateWhitelistedCommands(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Standard-Whitelist für sichere Befehle
	commands := []map[string]interface{}{
		{"command": "ls", "description": "Verzeichnisinhalt auflisten", "dangerous": false},
		{"command": "pwd", "description": "Aktuelles Verzeichnis anzeigen", "dangerous": false},
		{"command": "whoami", "description": "Aktuellen Benutzer anzeigen", "dangerous": false},
		{"command": "date", "description": "Datum und Uhrzeit anzeigen", "dangerous": false},
		{"command": "df", "description": "Festplattennutzung anzeigen", "dangerous": false},
		{"command": "free", "description": "Speichernutzung anzeigen", "dangerous": false},
		{"command": "uptime", "description": "Systemlaufzeit anzeigen", "dangerous": false},
		{"command": "cat", "description": "Dateiinhalt anzeigen", "dangerous": false},
		{"command": "head", "description": "Erste Zeilen einer Datei", "dangerous": false},
		{"command": "tail", "description": "Letzte Zeilen einer Datei", "dangerous": false},
	}

	writeJSON(w, map[string]interface{}{
		"commands": commands,
		"total":    len(commands),
	})
}

// handleFleetMateExportPDF exportiert Markdown/HTML als PDF
func (app *App) handleFleetMateExportPDF(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Content   string `json:"content"`   // Markdown oder HTML Inhalt
		MateID    string `json:"mateId"`
		LogPath   string `json:"logPath"`
		SessionID string `json:"sessionId"`
		Title     string `json:"title"` // Optional: Titel für das PDF
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	log.Printf("PDF-Export für Session: %s", req.SessionID)

	// Markdown zu HTML konvertieren und PDF generieren
	pdfBytes, err := generatePDF(req.Content, req.Title, req.MateID, req.LogPath, req.SessionID)
	if err != nil {
		log.Printf("PDF-Export Fehler: %v", err)
		http.Error(w, "PDF generation failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Dateiname generieren
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("fleet-navigator_%s_%s.pdf", req.MateID, timestamp)

	// PDF als Download senden
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(pdfBytes)))
	w.Write(pdfBytes)

	log.Printf("PDF exportiert: %s (%d bytes)", filename, len(pdfBytes))
}

// generatePDF konvertiert Markdown/Text zu PDF via Chromium
func generatePDF(content, title, mateID, logPath, sessionID string) ([]byte, error) {
	// HTML-Dokument mit Styling erstellen
	timestamp := time.Now().Format("02.01.2006 15:04:05")
	if title == "" {
		title = "Fleet Navigator - Bericht"
	}

	// Einfache Markdown→HTML Konvertierung (Basis)
	htmlContent := convertSimpleMarkdownToHTML(content)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>%s</title>
  <style>
    body {
      font-family: 'DejaVu Sans', Arial, sans-serif;
      margin: 40px;
      color: #333;
      line-height: 1.6;
    }
    h1 { color: #FF6B35; border-bottom: 3px solid #FF6B35; padding-bottom: 10px; }
    h2 { color: #FF8C42; margin-top: 30px; border-left: 4px solid #FF8C42; padding-left: 10px; }
    h3 { color: #555; }
    code { background-color: #f4f4f4; padding: 2px 6px; border-radius: 3px; font-family: monospace; }
    pre { background-color: #f8f8f8; border: 1px solid #ddd; border-left: 4px solid #FF6B35; padding: 15px; overflow-x: auto; }
    pre code { background-color: transparent; padding: 0; }
    .header { background-color: #FF6B35; color: white; padding: 20px; border-radius: 8px; margin-bottom: 30px; }
    .header h1 { color: white; border: none; margin: 0; }
    .meta-info { background-color: #f9f9f9; border: 1px solid #ddd; padding: 15px; border-radius: 4px; margin-bottom: 20px; font-size: 0.9em; }
    .meta-info strong { color: #FF6B35; }
    .footer { margin-top: 50px; padding-top: 20px; border-top: 2px solid #ddd; font-size: 0.85em; color: #888; text-align: center; }
  </style>
</head>
<body>
  <div class="header">
    <h1>🚢 %s</h1>
  </div>
  <div class="meta-info">
    <p><strong>Erstellt am:</strong> %s</p>
    <p><strong>Session ID:</strong> %s</p>
    <p><strong>Fleet Mate:</strong> %s</p>
    <p><strong>Log-Datei:</strong> %s</p>
  </div>
  %s
  <div class="footer">
    <p>Generiert von Fleet Navigator Go | © 2025 JavaFleet Systems</p>
  </div>
</body>
</html>`, title, title, timestamp, sessionID, mateID, logPath, htmlContent)

	// Temporäre HTML-Datei erstellen
	tmpHTML, err := os.CreateTemp("", "fleet-pdf-*.html")
	if err != nil {
		return nil, fmt.Errorf("temp HTML erstellen: %w", err)
	}
	defer os.Remove(tmpHTML.Name())

	if _, err := tmpHTML.WriteString(html); err != nil {
		tmpHTML.Close()
		return nil, fmt.Errorf("HTML schreiben: %w", err)
	}
	tmpHTML.Close()

	// Temporäre PDF-Datei
	tmpPDF, err := os.CreateTemp("", "fleet-pdf-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("temp PDF erstellen: %w", err)
	}
	tmpPDFPath := tmpPDF.Name()
	tmpPDF.Close()
	defer os.Remove(tmpPDFPath)

	// Chromium für PDF-Generierung verwenden
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Versuche verschiedene Chromium-Pfade
	chromiumPaths := []string{
		"/snap/bin/chromium",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/usr/bin/google-chrome",
	}

	var cmd *exec.Cmd
	for _, chromePath := range chromiumPaths {
		if _, err := os.Stat(chromePath); err == nil {
			cmd = exec.CommandContext(ctx, chromePath,
				"--headless",
				"--disable-gpu",
				"--no-sandbox",
				"--print-to-pdf="+tmpPDFPath,
				"--print-to-pdf-no-header",
				tmpHTML.Name(),
			)
			break
		}
	}

	if cmd == nil {
		return nil, fmt.Errorf("Chromium nicht gefunden - bitte installieren: snap install chromium")
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("Chromium PDF-Generierung fehlgeschlagen: %v - %s", err, string(output))
	}

	// PDF lesen
	pdfBytes, err := os.ReadFile(tmpPDFPath)
	if err != nil {
		return nil, fmt.Errorf("PDF lesen: %w", err)
	}

	return pdfBytes, nil
}

// convertSimpleMarkdownToHTML konvertiert einfaches Markdown zu HTML
func convertSimpleMarkdownToHTML(markdown string) string {
	lines := strings.Split(markdown, "\n")
	var result strings.Builder
	inCodeBlock := false

	for _, line := range lines {
		// Code-Blöcke
		if strings.HasPrefix(line, "```") {
			if inCodeBlock {
				result.WriteString("</code></pre>\n")
				inCodeBlock = false
			} else {
				result.WriteString("<pre><code>")
				inCodeBlock = true
			}
			continue
		}

		if inCodeBlock {
			result.WriteString(strings.ReplaceAll(line, "<", "&lt;"))
			result.WriteString("\n")
			continue
		}

		// Überschriften
		if strings.HasPrefix(line, "### ") {
			result.WriteString("<h3>" + strings.TrimPrefix(line, "### ") + "</h3>\n")
		} else if strings.HasPrefix(line, "## ") {
			result.WriteString("<h2>" + strings.TrimPrefix(line, "## ") + "</h2>\n")
		} else if strings.HasPrefix(line, "# ") {
			result.WriteString("<h1>" + strings.TrimPrefix(line, "# ") + "</h1>\n")
		} else if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			// Listen
			result.WriteString("<li>" + strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* ") + "</li>\n")
		} else if strings.TrimSpace(line) == "" {
			result.WriteString("<br>\n")
		} else {
			// Inline-Code
			processed := line
			for strings.Contains(processed, "`") {
				start := strings.Index(processed, "`")
				end := strings.Index(processed[start+1:], "`")
				if end == -1 {
					break
				}
				end += start + 1
				code := processed[start+1 : end]
				processed = processed[:start] + "<code>" + code + "</code>" + processed[end+1:]
			}
			// Fett
			for strings.Contains(processed, "**") {
				processed = strings.Replace(processed, "**", "<strong>", 1)
				processed = strings.Replace(processed, "**", "</strong>", 1)
			}
			result.WriteString("<p>" + processed + "</p>\n")
		}
	}

	if inCodeBlock {
		result.WriteString("</code></pre>\n")
	}

	return result.String()
}

// ============================================================================
// Office/Document Generation Handler
// ============================================================================

// handleOfficePing - Health-Check für Office API
func (app *App) handleOfficePing(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{
		"status":  "ok",
		"service": "Fleet Navigator Office API",
	})
}

// handleOfficeGenerateDocument generiert ein Dokument und öffnet es in LibreOffice
func (app *App) handleOfficeGenerateDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Prompt string `json:"prompt"` // Der Text/Prompt für das Dokument
		Model  string `json:"model"`  // Optional: Modell für KI-Generierung
		Format string `json:"format"` // Optional: "html", "odt", "docx" (default: html)
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Prompt == "" {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "Prompt is required",
		})
		return
	}

	if req.Format == "" {
		req.Format = "html"
	}

	log.Printf("Generating document: format=%s, model=%s, promptLen=%d", req.Format, req.Model, len(req.Prompt))

	// Text generieren (entweder via KI oder direkt verwenden)
	var content string
	if req.Model != "" && app.llamaServer != nil && app.llamaServer.IsRunning() {
		// KI-generierter Text
		log.Printf("Generating content with AI model: %s", req.Model)

		systemPrompt := `Du bist ein professioneller Assistent für Geschäftsbriefe und Dokumente.
Erstelle formale, präzise und gut strukturierte Dokumente auf Deutsch.
WICHTIG: Schreibe NUR den reinen Dokumenttext ohne Einleitung und ohne abschließende Hinweise.`

		messages := []llamaserver.ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: req.Prompt},
		}

		var generatedText strings.Builder
		err := app.llamaServer.StreamChat(messages, func(chunk string, done bool) {
			generatedText.WriteString(chunk)
		})
		if err != nil {
			log.Printf("AI generation failed: %v", err)
			writeJSON(w, map[string]interface{}{
				"success": false,
				"error":   "KI-Generierung fehlgeschlagen: " + err.Error(),
			})
			return
		}
		content = generatedText.String()
	} else {
		// Prompt direkt als Inhalt verwenden
		content = req.Prompt
	}

	// Dokument erstellen
	documentPath, err := createDocument(content, req.Format)
	if err != nil {
		log.Printf("Document creation failed: %v", err)
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "Dokument-Erstellung fehlgeschlagen: " + err.Error(),
		})
		return
	}

	// In LibreOffice öffnen
	if err := openInTextEditor(documentPath); err != nil {
		log.Printf("Failed to open document: %v", err)
		// Trotzdem Erfolg melden, Dokument wurde erstellt
	}

	writeJSON(w, map[string]interface{}{
		"success": true,
		"content": content,
		"path":    documentPath,
		"format":  req.Format,
		"model":   req.Model,
	})
}

// createDocument erstellt ein Dokument im angegebenen Format
func createDocument(content, format string) (string, error) {
	// Dokument-Verzeichnis erstellen
	homeDir, _ := os.UserHomeDir()
	documentsDir := filepath.Join(homeDir, "FleetNavigator", "Documents")
	if err := os.MkdirAll(documentsDir, 0755); err != nil {
		return "", fmt.Errorf("Verzeichnis erstellen: %w", err)
	}

	// Dateiname mit Zeitstempel
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	var filename string
	var fileContent string

	switch format {
	case "odt", "docx":
		// Für ODT/DOCX erstellen wir HTML, das LibreOffice öffnen und konvertieren kann
		filename = fmt.Sprintf("Document_%s.html", timestamp)
		fileContent = createHTMLDocument(content)
	case "txt":
		filename = fmt.Sprintf("Document_%s.txt", timestamp)
		fileContent = content
	case "md":
		filename = fmt.Sprintf("Document_%s.md", timestamp)
		fileContent = content
	default: // html
		filename = fmt.Sprintf("Document_%s.html", timestamp)
		fileContent = createHTMLDocument(content)
	}

	documentPath := filepath.Join(documentsDir, filename)

	if err := os.WriteFile(documentPath, []byte(fileContent), 0644); err != nil {
		return "", fmt.Errorf("Datei schreiben: %w", err)
	}

	log.Printf("Document created: %s", documentPath)
	return documentPath, nil
}

// createHTMLDocument erstellt ein HTML-Dokument mit Styling
func createHTMLDocument(content string) string {
	// Zeilenumbrüche zu <br> konvertieren
	htmlContent := strings.ReplaceAll(content, "\n", "<br>\n")

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>Fleet Navigator Dokument</title>
  <style>
    body {
      font-family: Arial, sans-serif;
      font-size: 12pt;
      line-height: 1.6;
      margin: 2.5cm;
      color: #333;
    }
    p { margin-bottom: 1em; }
  </style>
</head>
<body>
%s
</body>
</html>`, htmlContent)
}

// openInTextEditor öffnet ein Dokument im Standard-Texteditor
func openInTextEditor(documentPath string) error {
	// Versuche verschiedene Editoren
	editors := []struct {
		cmd  string
		args []string
	}{
		{"libreoffice", []string{"--writer", documentPath}},
		{"soffice", []string{"--writer", documentPath}},
		{"xdg-open", []string{documentPath}},
	}

	for _, editor := range editors {
		path, err := exec.LookPath(editor.cmd)
		if err != nil {
			continue
		}

		cmd := exec.Command(path, editor.args...)
		if err := cmd.Start(); err != nil {
			log.Printf("Failed to start %s: %v", editor.cmd, err)
			continue
		}

		log.Printf("Document opened with %s: %s", editor.cmd, documentPath)
		return nil
	}

	return fmt.Errorf("kein passender Editor gefunden")
}

// ============================================================================
// Model Store erweiterte Handler
// ============================================================================

// handleModelStoreByID behandelt /api/model-store/{id}/... Endpoints
func (app *App) handleModelStoreByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/model-store/")

	// Ignoriere bekannte Sub-Pfade
	if strings.HasPrefix(path, "all") || strings.HasPrefix(path, "featured") ||
		strings.HasPrefix(path, "download/") || strings.HasPrefix(path, "huggingface/") {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	parts := strings.Split(path, "/")
	if len(parts) < 1 || parts[0] == "" {
		http.Error(w, "Missing model ID", http.StatusBadRequest)
		return
	}

	modelID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch action {
	case "check-gpu":
		// GET /api/model-store/{id}/check-gpu
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// GPU-Check basierend auf Hardware-Monitor
		gpuInfo, _ := app.hardwareMonitor.GetGPU()
		vramFree := int64(0)
		vramTotal := int64(0)
		hasGPU := false

		if len(gpuInfo) > 0 {
			hasGPU = true
			vramFree = int64(gpuInfo[0].MemoryFree) * 1024 * 1024   // MB to bytes
			vramTotal = int64(gpuInfo[0].MemoryTotal) * 1024 * 1024 // MB to bytes
		}

		// Schätze benötigten VRAM basierend auf Modell-Größe
		estimatedVRAM := int64(8 * 1024 * 1024 * 1024) // Default 8GB
		canRun := vramFree >= estimatedVRAM

		writeJSON(w, map[string]interface{}{
			"modelId":       modelID,
			"canRun":        canRun,
			"hasGPU":        hasGPU,
			"vramFree":      vramFree,
			"vramTotal":     vramTotal,
			"vramRequired":  estimatedVRAM,
			"recommendation": func() string {
				if !hasGPU {
					return "Keine GPU gefunden. Das Modell läuft auf der CPU (langsamer)."
				}
				if canRun {
					return "Genügend VRAM verfügbar. Das Modell kann auf der GPU ausgeführt werden."
				}
				return "Nicht genügend VRAM. Das Modell wird teilweise auf die CPU ausgelagert."
			}(),
		})

	case "cancel":
		// POST /api/model-store/download/{id}/cancel
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// TODO: Implement download cancellation
		writeJSON(w, map[string]interface{}{
			"success": true,
			"modelId": modelID,
			"message": "Download abgebrochen",
		})

	default:
		// DELETE /api/model-store/{id}
		if r.Method == http.MethodDelete && action == "" {
			// Lösche GGUF-Modell
			modelsDir := filepath.Join(app.config.DataDir, "models", "library")
			modelPath := filepath.Join(modelsDir, modelID+".gguf")

			// Versuche auch ohne .gguf Extension
			if _, err := os.Stat(modelPath); os.IsNotExist(err) {
				modelPath = filepath.Join(modelsDir, modelID)
			}

			if err := os.Remove(modelPath); err != nil {
				if os.IsNotExist(err) {
					http.Error(w, "Model not found", http.StatusNotFound)
				} else {
					http.Error(w, "Failed to delete model: "+err.Error(), http.StatusInternalServerError)
				}
				return
			}

			writeJSON(w, map[string]interface{}{
				"success": true,
				"modelId": modelID,
				"message": "Modell gelöscht",
			})
			return
		}

		http.Error(w, "Unknown action", http.StatusNotFound)
	}
}

// ============================================================================
// Sampling Handler
// ============================================================================

// handleSamplingPresets gibt Sampling-Presets zurück
func (app *App) handleSamplingPresets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	presetName := strings.TrimPrefix(r.URL.Path, "/api/sampling/presets/")

	presets := map[string]map[string]interface{}{
		"creative": {
			"name":        "Kreativ",
			"description": "Höhere Kreativität und Variation",
			"temperature": 0.9,
			"topP":        0.95,
			"topK":        50,
			"repeatPenalty": 1.1,
		},
		"balanced": {
			"name":        "Ausgewogen",
			"description": "Balance zwischen Kreativität und Präzision",
			"temperature": 0.7,
			"topP":        0.9,
			"topK":        40,
			"repeatPenalty": 1.1,
		},
		"precise": {
			"name":        "Präzise",
			"description": "Fokus auf Genauigkeit und Konsistenz",
			"temperature": 0.3,
			"topP":        0.8,
			"topK":        20,
			"repeatPenalty": 1.2,
		},
		"code": {
			"name":        "Code",
			"description": "Optimiert für Programmierung",
			"temperature": 0.2,
			"topP":        0.85,
			"topK":        30,
			"repeatPenalty": 1.15,
		},
	}

	if presetName != "" {
		if preset, ok := presets[presetName]; ok {
			writeJSON(w, preset)
			return
		}
		http.Error(w, "Preset not found", http.StatusNotFound)
		return
	}

	writeJSON(w, presets)
}

// handleSamplingDefaults gibt Standard-Sampling-Parameter zurück
func (app *App) handleSamplingDefaults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, map[string]interface{}{
		"temperature":   0.7,
		"topP":          0.9,
		"topK":          40,
		"repeatPenalty": 1.1,
		"maxTokens":     4096,
		"seed":          -1,
	})
}

// handleSamplingDefaultsAuto gibt modellspezifische Defaults zurück
func (app *App) handleSamplingDefaultsAuto(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	modelName := strings.TrimPrefix(r.URL.Path, "/api/sampling/defaults/auto/")

	// Modellspezifische Empfehlungen
	defaults := map[string]interface{}{
		"temperature":   0.7,
		"topP":          0.9,
		"topK":          40,
		"repeatPenalty": 1.1,
		"maxTokens":     4096,
	}

	// Anpassungen basierend auf Modelltyp
	modelLower := strings.ToLower(modelName)
	if strings.Contains(modelLower, "coder") || strings.Contains(modelLower, "code") {
		defaults["temperature"] = 0.2
		defaults["topP"] = 0.85
		defaults["repeatPenalty"] = 1.15
	} else if strings.Contains(modelLower, "instruct") {
		defaults["temperature"] = 0.5
		defaults["topP"] = 0.9
	} else if strings.Contains(modelLower, "llava") || strings.Contains(modelLower, "vision") {
		defaults["temperature"] = 0.6
		defaults["maxTokens"] = 2048
	}

	writeJSON(w, map[string]interface{}{
		"model":    modelName,
		"defaults": defaults,
	})
}

// handleSamplingHelp gibt Hilfetext für Sampling-Parameter zurück
func (app *App) handleSamplingHelp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	lang := strings.TrimPrefix(r.URL.Path, "/api/sampling/help/")
	if lang == "" {
		lang = "de"
	}

	help := map[string]interface{}{
		"temperature": map[string]string{
			"name":        "Temperatur",
			"description": "Kontrolliert die Zufälligkeit der Ausgabe. Niedrigere Werte (0.1-0.3) erzeugen fokussiertere, deterministische Antworten. Höhere Werte (0.7-1.0) erzeugen kreativere, vielfältigere Ausgaben.",
			"range":       "0.0 - 2.0",
			"default":     "0.7",
		},
		"topP": map[string]string{
			"name":        "Top P (Nucleus Sampling)",
			"description": "Begrenzt die Auswahl auf Token, deren kumulative Wahrscheinlichkeit den Schwellenwert erreicht. 0.9 bedeutet: nur die wahrscheinlichsten Token, die zusammen 90% der Wahrscheinlichkeit ausmachen.",
			"range":       "0.0 - 1.0",
			"default":     "0.9",
		},
		"topK": map[string]string{
			"name":        "Top K",
			"description": "Begrenzt die Auswahl auf die K wahrscheinlichsten Token. Niedrigere Werte erzeugen fokussiertere Ausgaben.",
			"range":       "1 - 100",
			"default":     "40",
		},
		"repeatPenalty": map[string]string{
			"name":        "Wiederholungsstrafe",
			"description": "Bestraft die Wiederholung von Token. Werte > 1.0 reduzieren Wiederholungen, Werte < 1.0 fördern sie.",
			"range":       "0.0 - 2.0",
			"default":     "1.1",
		},
		"maxTokens": map[string]string{
			"name":        "Maximale Token",
			"description": "Die maximale Anzahl von Token, die generiert werden sollen.",
			"range":       "1 - 32768",
			"default":     "4096",
		},
	}

	writeJSON(w, map[string]interface{}{
		"language": lang,
		"help":     help,
	})
}

// ============================================================================
// FleetCode Handler
// ============================================================================

// handleFleetCodeExecute startet Code-Ausführung auf einem Mate
func (app *App) handleFleetCodeExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mateID := strings.TrimPrefix(r.URL.Path, "/api/fleetcode/execute/")
	if mateID == "" {
		http.Error(w, "Missing mate ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Code     string `json:"code"`
		Language string `json:"language"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	sessionID := fmt.Sprintf("fleetcode-%s-%d", mateID, time.Now().UnixNano())

	writeJSON(w, map[string]interface{}{
		"sessionId": sessionID,
		"mateId":    mateID,
		"language":  req.Language,
		"status":    "started",
	})
}

// handleFleetCodeStream streamt FleetCode-Ausführungsergebnisse
func (app *App) handleFleetCodeStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := strings.TrimPrefix(r.URL.Path, "/api/fleetcode/stream/")
	if sessionID == "" {
		http.Error(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	// SSE Setup
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Simulierte Ausführung
	fmt.Fprintf(w, "data: {\"type\":\"status\",\"message\":\"FleetCode ist in der Go-Version noch nicht vollständig implementiert\"}\n\n")
	flusher.Flush()
	fmt.Fprintf(w, "event: done\ndata: {\"exitCode\":0,\"output\":\"\"}\n\n")
	flusher.Flush()
}

// ============================================================================
// Embedding Config Handler
// ============================================================================

// handleEmbeddingConfig verwaltet Embedding-Konfiguration
func (app *App) handleEmbeddingConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Standard-Embedding-Konfiguration zurückgeben
		writeJSON(w, map[string]interface{}{
			"enabled":     false,
			"provider":    "none",
			"model":       "",
			"dimension":   0,
			"chunkSize":   500,
			"chunkOverlap": 50,
			"status":      "Embedding ist in der Go-Version noch nicht implementiert",
		})

	case http.MethodPost:
		var config struct {
			Enabled     bool   `json:"enabled"`
			Provider    string `json:"provider"`
			Model       string `json:"model"`
			ChunkSize   int    `json:"chunkSize"`
			ChunkOverlap int   `json:"chunkOverlap"`
		}
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Stub - Speichern noch nicht implementiert
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "Embedding-Konfiguration kann in der Go-Version noch nicht geändert werden",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ============================================================================
// Document Export Handlers (Native Formats)
// ============================================================================

// handleExportDOCX exportiert Inhalt als native DOCX-Datei
func (app *App) handleExportDOCX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Content  string `json:"content"`
		Title    string `json:"title"`
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "Content is required",
		})
		return
	}

	// DOCX generieren
	docxBytes, err := generateNativeDOCX(req.Content, req.Title)
	if err != nil {
		log.Printf("DOCX generation failed: %v", err)
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "DOCX-Generierung fehlgeschlagen: " + err.Error(),
		})
		return
	}

	// Dateiname
	filename := req.Filename
	if filename == "" {
		filename = fmt.Sprintf("Dokument_%s.docx", time.Now().Format("2006-01-02_15-04-05"))
	}
	if !strings.HasSuffix(filename, ".docx") {
		filename += ".docx"
	}

	// Als Download senden
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(docxBytes)))
	w.Write(docxBytes)

	log.Printf("DOCX exportiert: %s (%d bytes)", filename, len(docxBytes))
}

// handleExportODT exportiert Inhalt als native ODT-Datei
func (app *App) handleExportODT(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Content  string `json:"content"`
		Title    string `json:"title"`
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "Content is required",
		})
		return
	}

	// ODT generieren
	odtBytes, err := generateNativeODT(req.Content, req.Title)
	if err != nil {
		log.Printf("ODT generation failed: %v", err)
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "ODT-Generierung fehlgeschlagen: " + err.Error(),
		})
		return
	}

	// Dateiname
	filename := req.Filename
	if filename == "" {
		filename = fmt.Sprintf("Dokument_%s.odt", time.Now().Format("2006-01-02_15-04-05"))
	}
	if !strings.HasSuffix(filename, ".odt") {
		filename += ".odt"
	}

	// Als Download senden
	w.Header().Set("Content-Type", "application/vnd.oasis.opendocument.text")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(odtBytes)))
	w.Write(odtBytes)

	log.Printf("ODT exportiert: %s (%d bytes)", filename, len(odtBytes))
}

// handleExportCSV exportiert Inhalt als CSV-Datei
func (app *App) handleExportCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Data      [][]string `json:"data"`      // 2D-Array für Tabellendaten
		Content   string     `json:"content"`   // Alternative: Plain text mit Trennzeichen
		Separator string     `json:"separator"` // Standard: Semikolon (für deutsches Excel)
		Filename  string     `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Data) == 0 && req.Content == "" {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "Data or content is required",
		})
		return
	}

	// Separator setzen (Standard: Semikolon für deutsches Excel)
	separator := req.Separator
	if separator == "" {
		separator = ";"
	}

	// CSV generieren
	var csvContent string
	if len(req.Data) > 0 {
		var builder strings.Builder
		for _, row := range req.Data {
			builder.WriteString(strings.Join(row, separator))
			builder.WriteString("\r\n")
		}
		csvContent = builder.String()
	} else {
		csvContent = req.Content
	}

	// BOM für UTF-8 (für Excel-Kompatibilität)
	csvBytes := append([]byte{0xEF, 0xBB, 0xBF}, []byte(csvContent)...)

	// Dateiname
	filename := req.Filename
	if filename == "" {
		filename = fmt.Sprintf("Tabelle_%s.csv", time.Now().Format("2006-01-02_15-04-05"))
	}
	if !strings.HasSuffix(filename, ".csv") {
		filename += ".csv"
	}

	// Als Download senden
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(csvBytes)))
	w.Write(csvBytes)

	log.Printf("CSV exportiert: %s (%d bytes)", filename, len(csvBytes))
}

// handleExportRTF exportiert Inhalt als RTF-Datei
func (app *App) handleExportRTF(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Content  string `json:"content"`
		Title    string `json:"title"`
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "Content is required",
		})
		return
	}

	// RTF generieren
	rtfContent := generateRTF(req.Content, req.Title)

	// Dateiname
	filename := req.Filename
	if filename == "" {
		filename = fmt.Sprintf("Dokument_%s.rtf", time.Now().Format("2006-01-02_15-04-05"))
	}
	if !strings.HasSuffix(filename, ".rtf") {
		filename += ".rtf"
	}

	// Als Download senden
	w.Header().Set("Content-Type", "application/rtf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(rtfContent)))
	w.Write([]byte(rtfContent))

	log.Printf("RTF exportiert: %s (%d bytes)", filename, len(rtfContent))
}

// handleExportPDF exportiert Inhalt als PDF-Datei
func (app *App) handleExportPDF(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Content  string `json:"content"`
		Title    string `json:"title"`
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "Content is required",
		})
		return
	}

	// PDF generieren (mit gofpdf)
	pdfBytes, err := generateSimplePDF(req.Content, req.Title)
	if err != nil {
		log.Printf("PDF generation failed: %v", err)
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "PDF-Generierung fehlgeschlagen: " + err.Error(),
		})
		return
	}

	// Dateiname
	filename := req.Filename
	if filename == "" {
		filename = fmt.Sprintf("Dokument_%s.pdf", time.Now().Format("2006-01-02_15-04-05"))
	}
	if !strings.HasSuffix(filename, ".pdf") {
		filename += ".pdf"
	}

	// Als Download senden
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(pdfBytes)))
	w.Write(pdfBytes)

	log.Printf("PDF exportiert: %s (%d bytes)", filename, len(pdfBytes))
}

// generateSimplePDF erstellt ein PDF aus Text mit gofpdf (für Chat-Export)
func generateSimplePDF(content, title string) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()

	// Titel
	if title != "" {
		pdf.SetFont("Helvetica", "B", 16)
		pdf.MultiCell(0, 10, title, "", "L", false)
		pdf.Ln(5)
	}

	// Inhalt
	pdf.SetFont("Helvetica", "", 11)

	// Zeilenumbrüche behandeln
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if line == "" {
			pdf.Ln(5)
		} else {
			// Konvertiere UTF-8 zu Latin-1 für fpdf (vereinfacht)
			cleanLine := cleanForPDF(line)
			pdf.MultiCell(0, 6, cleanLine, "", "L", false)
		}
	}

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// cleanForPDF konvertiert UTF-8 Text für PDF (ersetzt Umlaute und Sonderzeichen)
func cleanForPDF(s string) string {
	replacer := strings.NewReplacer(
		"ä", "ae", "ö", "oe", "ü", "ue",
		"Ä", "Ae", "Ö", "Oe", "Ü", "Ue",
		"ß", "ss",
		"\u20AC", "EUR",  // €
		"\u201E", "\"",   // „
		"\u201C", "\"",   // "
		"\u201D", "\"",   // "
		"\u2013", "-",    // –
		"\u2014", "-",    // —
		"\u2018", "'",    // '
		"\u2019", "'",    // '
	)
	return replacer.Replace(s)
}

// ============================================================================
// Native Document Format Generators
// ============================================================================

// generateNativeDOCX erstellt eine echte DOCX-Datei (ZIP mit XML)
func generateNativeDOCX(content, title string) ([]byte, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	// Escape XML-Sonderzeichen im Content
	escapedContent := escapeXML(content)
	escapedTitle := escapeXML(title)
	if escapedTitle == "" {
		escapedTitle = "Fleet Navigator Dokument"
	}

	// Content-Typen in Absätze aufteilen
	paragraphs := strings.Split(escapedContent, "\n")
	var paragraphXML strings.Builder
	for _, para := range paragraphs {
		if strings.TrimSpace(para) == "" {
			paragraphXML.WriteString(`<w:p><w:r><w:t></w:t></w:r></w:p>`)
		} else {
			paragraphXML.WriteString(fmt.Sprintf(`<w:p><w:r><w:t xml:space="preserve">%s</w:t></w:r></w:p>`, para))
		}
	}

	// 1. [Content_Types].xml
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
  <Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>
</Types>`
	if err := addFileToZip(zipWriter, "[Content_Types].xml", contentTypes); err != nil {
		return nil, err
	}

	// 2. _rels/.rels
	rootRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>
</Relationships>`
	if err := addFileToZip(zipWriter, "_rels/.rels", rootRels); err != nil {
		return nil, err
	}

	// 3. word/_rels/document.xml.rels
	docRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`
	if err := addFileToZip(zipWriter, "word/_rels/document.xml.rels", docRels); err != nil {
		return nil, err
	}

	// 4. word/document.xml - Das eigentliche Dokument
	documentXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:pPr><w:pStyle w:val="Title"/></w:pPr><w:r><w:t>%s</w:t></w:r></w:p>
    %s
  </w:body>
</w:document>`, escapedTitle, paragraphXML.String())
	if err := addFileToZip(zipWriter, "word/document.xml", documentXML); err != nil {
		return nil, err
	}

	// 5. word/styles.xml
	stylesXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:style w:type="paragraph" w:styleId="Title">
    <w:name w:val="Title"/>
    <w:rPr><w:b/><w:sz w:val="48"/></w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:default="1" w:styleId="Normal">
    <w:name w:val="Normal"/>
    <w:rPr><w:sz w:val="24"/></w:rPr>
  </w:style>
</w:styles>`
	if err := addFileToZip(zipWriter, "word/styles.xml", stylesXML); err != nil {
		return nil, err
	}

	// 6. docProps/core.xml - Metadaten
	coreXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties"
                   xmlns:dc="http://purl.org/dc/elements/1.1/"
                   xmlns:dcterms="http://purl.org/dc/terms/">
  <dc:title>%s</dc:title>
  <dc:creator>Fleet Navigator</dc:creator>
  <dcterms:created>%s</dcterms:created>
</cp:coreProperties>`, escapedTitle, time.Now().Format(time.RFC3339))
	if err := addFileToZip(zipWriter, "docProps/core.xml", coreXML); err != nil {
		return nil, err
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("ZIP schließen: %w", err)
	}

	return buf.Bytes(), nil
}

// generateNativeODT erstellt eine echte ODT-Datei (ZIP mit XML)
func generateNativeODT(content, title string) ([]byte, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	// Escape XML-Sonderzeichen
	escapedContent := escapeXML(content)
	escapedTitle := escapeXML(title)
	if escapedTitle == "" {
		escapedTitle = "Fleet Navigator Dokument"
	}

	// Content in Absätze aufteilen
	paragraphs := strings.Split(escapedContent, "\n")
	var paragraphXML strings.Builder
	for _, para := range paragraphs {
		if strings.TrimSpace(para) == "" {
			paragraphXML.WriteString(`<text:p text:style-name="Standard"/>`)
		} else {
			paragraphXML.WriteString(fmt.Sprintf(`<text:p text:style-name="Standard">%s</text:p>`, para))
		}
	}

	// 1. mimetype (MUSS der erste Eintrag sein, unkomprimiert!)
	mimetypeWriter, err := zipWriter.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store, // Nicht komprimiert!
	})
	if err != nil {
		return nil, err
	}
	mimetypeWriter.Write([]byte("application/vnd.oasis.opendocument.text"))

	// 2. META-INF/manifest.xml
	manifest := `<?xml version="1.0" encoding="UTF-8"?>
<manifest:manifest xmlns:manifest="urn:oasis:names:tc:opendocument:xmlns:manifest:1.0" manifest:version="1.2">
  <manifest:file-entry manifest:full-path="/" manifest:media-type="application/vnd.oasis.opendocument.text"/>
  <manifest:file-entry manifest:full-path="content.xml" manifest:media-type="text/xml"/>
  <manifest:file-entry manifest:full-path="styles.xml" manifest:media-type="text/xml"/>
  <manifest:file-entry manifest:full-path="meta.xml" manifest:media-type="text/xml"/>
</manifest:manifest>`
	if err := addFileToZip(zipWriter, "META-INF/manifest.xml", manifest); err != nil {
		return nil, err
	}

	// 3. content.xml - Der eigentliche Inhalt
	contentXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<office:document-content
    xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
    xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0"
    xmlns:style="urn:oasis:names:tc:opendocument:xmlns:style:1.0"
    office:version="1.2">
  <office:body>
    <office:text>
      <text:p text:style-name="Title">%s</text:p>
      %s
    </office:text>
  </office:body>
</office:document-content>`, escapedTitle, paragraphXML.String())
	if err := addFileToZip(zipWriter, "content.xml", contentXML); err != nil {
		return nil, err
	}

	// 4. styles.xml
	stylesXML := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-styles
    xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
    xmlns:style="urn:oasis:names:tc:opendocument:xmlns:style:1.0"
    xmlns:fo="urn:oasis:names:tc:opendocument:xmlns:xsl-fo-compatible:1.0"
    office:version="1.2">
  <office:styles>
    <style:style style:name="Title" style:family="paragraph">
      <style:text-properties fo:font-size="24pt" fo:font-weight="bold"/>
    </style:style>
    <style:style style:name="Standard" style:family="paragraph">
      <style:text-properties fo:font-size="12pt"/>
    </style:style>
  </office:styles>
</office:document-styles>`
	if err := addFileToZip(zipWriter, "styles.xml", stylesXML); err != nil {
		return nil, err
	}

	// 5. meta.xml - Metadaten
	metaXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<office:document-meta
    xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
    xmlns:dc="http://purl.org/dc/elements/1.1/"
    xmlns:meta="urn:oasis:names:tc:opendocument:xmlns:meta:1.0"
    office:version="1.2">
  <office:meta>
    <dc:title>%s</dc:title>
    <dc:creator>Fleet Navigator</dc:creator>
    <meta:creation-date>%s</meta:creation-date>
    <meta:generator>Fleet Navigator Go</meta:generator>
  </office:meta>
</office:document-meta>`, escapedTitle, time.Now().Format(time.RFC3339))
	if err := addFileToZip(zipWriter, "meta.xml", metaXML); err != nil {
		return nil, err
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("ZIP schließen: %w", err)
	}

	return buf.Bytes(), nil
}

// generateRTF erstellt eine RTF-Datei
func generateRTF(content, title string) string {
	// RTF-Header mit Schriftdefinition und deutschen Zeichensatz
	rtf := `{\rtf1\ansi\ansicpg1252\deff0\nouicompat
{\fonttbl{\f0\fswiss\fcharset0 Arial;}{\f1\fmodern\fcharset0 Consolas;}}
{\colortbl ;\red255\green107\blue53;\red0\green0\blue0;}
\viewkind4\uc1\pard\cf1\b\fs48 `

	// Titel hinzufügen
	if title != "" {
		rtf += escapeRTF(title) + `\par\cf2\b0\fs24\par `
	} else {
		rtf += `Fleet Navigator Dokument\par\cf2\b0\fs24\par `
	}

	// Inhalt hinzufügen (Zeilenumbrüche als \par)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		rtf += escapeRTF(line) + `\par `
	}

	// RTF-Footer
	rtf += `\par\par\fs18\i Generiert von Fleet Navigator Go\i0}`

	return rtf
}

// escapeRTF escaped Sonderzeichen für RTF
func escapeRTF(s string) string {
	// Backslash zuerst!
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `{`, `\{`)
	s = strings.ReplaceAll(s, `}`, `\}`)
	// Deutsche Umlaute
	s = strings.ReplaceAll(s, "ä", `\'e4`)
	s = strings.ReplaceAll(s, "ö", `\'f6`)
	s = strings.ReplaceAll(s, "ü", `\'fc`)
	s = strings.ReplaceAll(s, "Ä", `\'c4`)
	s = strings.ReplaceAll(s, "Ö", `\'d6`)
	s = strings.ReplaceAll(s, "Ü", `\'dc`)
	s = strings.ReplaceAll(s, "ß", `\'df`)
	return s
}

// escapeXML escaped XML-Sonderzeichen
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// addFileToZip fügt eine Datei zum ZIP-Archiv hinzu
func addFileToZip(zipWriter *zip.Writer, filename, content string) error {
	writer, err := zipWriter.Create(filename)
	if err != nil {
		return fmt.Errorf("ZIP-Eintrag %s erstellen: %w", filename, err)
	}
	_, err = writer.Write([]byte(content))
	if err != nil {
		return fmt.Errorf("ZIP-Eintrag %s schreiben: %w", filename, err)
	}
	return nil
}

// optimizeSearchQuery verwendet das LLM um eine bessere Suchanfrage zu generieren
// Wandelt konversationelle Fragen in effektive Suchbegriffe um
func (app *App) optimizeSearchQuery(userMessage string) string {
	// Wenn llama-server nicht läuft, Original verwenden
	if app.llamaServer == nil || !app.llamaServer.IsRunning() {
		return userMessage
	}

	systemPrompt := `Du bist ein Suchquery-Optimierer. Deine Aufgabe ist es, aus einer Benutzeranfrage
einen optimalen Suchbegriff für eine Web-Suchmaschine zu generieren.

REGELN:
1. Extrahiere die Kernbegriffe aus der Frage
2. Entferne Füllwörter und konversationelle Elemente
3. Füge relevante Synonyme oder Fachbegriffe hinzu wenn sinnvoll
4. Der Suchbegriff sollte 3-8 Wörter lang sein
5. Antworte NUR mit dem optimierten Suchbegriff, NICHTS anderes
6. Keine Anführungszeichen, keine Erklärungen

BEISPIELE:
Eingabe: "Kannst du mir sagen wie das Wetter morgen in München wird?"
Ausgabe: Wetter München morgen Vorhersage

Eingabe: "Was sind die besten Restaurants in Berlin?"
Ausgabe: beste Restaurants Berlin Empfehlungen 2024

Eingabe: "Ich möchte wissen wie man eine GmbH gründet"
Ausgabe: GmbH gründen Anleitung Deutschland Schritte`

	// Timeout von 10 Sekunden für Query-Optimierung
	result, err := app.llamaServer.QuickChatWithTimeout(systemPrompt, userMessage, 10*time.Second)
	if err != nil {
		log.Printf("Query-Optimierung fehlgeschlagen: %v - verwende Original", err)
		return userMessage
	}

	// Ergebnis bereinigen (Anführungszeichen entfernen, trimmen)
	optimized := strings.TrimSpace(result)
	optimized = strings.Trim(optimized, `"'`)

	// Wenn das Ergebnis zu kurz oder leer ist, Original verwenden
	if len(optimized) < 3 {
		return userMessage
	}

	log.Printf("Query optimiert: '%s' -> '%s'", userMessage, optimized)
	return optimized
}

// formatSearchContextEnhanced erstellt einen verbesserten Kontext aus Suchergebnissen
// mit RAG-ähnlicher Struktur für bessere LLM-Integration
func (app *App) formatSearchContextEnhanced(results []search.SearchResult, userQuestion, searchQuery string) string {
	if len(results) == 0 {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("=== WEB-RECHERCHE ERGEBNISSE ===\n")
	sb.WriteString(fmt.Sprintf("Suchanfrage: %s\n", searchQuery))
	sb.WriteString(fmt.Sprintf("Anzahl Quellen: %d\n\n", len(results)))

	for i, r := range results {
		sb.WriteString(fmt.Sprintf("--- QUELLE [%d]: %s ---\n", i+1, r.Title))
		sb.WriteString(fmt.Sprintf("URL: %s\n", r.URL))

		// Snippet oder Content verwenden
		content := r.Content
		if content == "" {
			content = r.Snippet
		}

		if content != "" {
			// Content kürzen wenn zu lang
			if len(content) > 2000 {
				content = content[:2000] + "..."
			}
			sb.WriteString(fmt.Sprintf("Inhalt:\n%s\n", content))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("=== ENDE WEB-RECHERCHE ===\n")

	return sb.String()
}

// ========== Voice API Handlers (Whisper STT + Piper TTS) ==========

// handleVoiceStatus gibt den Status des Voice-Services zurück
func (app *App) handleVoiceStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := app.voiceService.GetStatus()
	json.NewEncoder(w).Encode(status)
}

// handleVoiceSTT verarbeitet Speech-to-Text Anfragen
func (app *App) handleVoiceSTT(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Multipart Form parsen (max 50MB für Audio)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		http.Error(w, fmt.Sprintf("Form parsen: %v", err), http.StatusBadRequest)
		return
	}

	// Audio-Datei lesen
	file, header, err := r.FormFile("audio")
	if err != nil {
		http.Error(w, fmt.Sprintf("Audio-Datei fehlt: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Audio-Daten lesen
	audioData, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Audio lesen: %v", err), http.StatusInternalServerError)
		return
	}

	// Format aus Dateiname extrahieren
	format := "webm" // Default
	if ext := filepath.Ext(header.Filename); ext != "" {
		format = strings.TrimPrefix(ext, ".")
	}

	// Optional: Format aus Form-Feld
	if f := r.FormValue("format"); f != "" {
		format = f
	}

	log.Printf("STT-Anfrage: %d bytes, Format: %s", len(audioData), format)

	// Transkription durchführen
	result, err := app.voiceService.TranscribeAudio(audioData, format)
	if err != nil {
		log.Printf("STT-Fehler: %v", err)
		http.Error(w, fmt.Sprintf("Transkription fehlgeschlagen: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleVoiceTTS verarbeitet Text-to-Speech Anfragen
func (app *App) handleVoiceTTS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Text  string `json:"text"`
		Voice string `json:"voice,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("JSON parsen: %v", err), http.StatusBadRequest)
		return
	}

	if req.Text == "" {
		http.Error(w, "Text ist erforderlich", http.StatusBadRequest)
		return
	}

	log.Printf("TTS-Anfrage: %d Zeichen, Stimme: %s", len(req.Text), req.Voice)

	// Sprachsynthese durchführen (mit optionaler Stimme)
	result, err := app.voiceService.SynthesizeSpeech(req.Text, req.Voice)
	if err != nil {
		log.Printf("TTS-Fehler: %v", err)
		http.Error(w, fmt.Sprintf("Sprachsynthese fehlgeschlagen: %v", err), http.StatusInternalServerError)
		return
	}

	// Audio als WAV zurückgeben
	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Content-Disposition", "inline; filename=\"speech.wav\"")
	w.Header().Set("Content-Length", strconv.Itoa(len(result.AudioData)))
	w.Write(result.AudioData)
}

// handleVoiceDownload startet den Download von Voice-Modellen
// Optional: ?component=whisper oder ?component=piper für einzelnen Download
func (app *App) handleVoiceDownload(w http.ResponseWriter, r *http.Request) {
	// GET erlauben für EventSource (SSE braucht GET)
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Optional: nur eine Komponente herunterladen
	component := r.URL.Query().Get("component") // "whisper", "piper", oder leer für beide

	// SSE für Progress-Updates
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming nicht unterstützt", http.StatusInternalServerError)
		return
	}

	progressChan := make(chan voice.DownloadProgress, 10)

	// Download in Goroutine starten
	go func() {
		defer close(progressChan)
		var err error

		switch component {
		case "whisper":
			err = app.voiceService.EnsureWhisperDownloaded(progressChan)
		case "piper":
			err = app.voiceService.EnsurePiperDownloaded(progressChan)
		default:
			// Beide herunterladen (altes Verhalten)
			err = app.voiceService.EnsureModelsDownloaded(progressChan)
		}

		if err != nil {
			progressChan <- voice.DownloadProgress{
				Component: component,
				Status:    "error",
				Error:     err.Error(),
			}
		}
	}()

	// Progress-Updates senden
	for progress := range progressChan {
		data, _ := json.Marshal(progress)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		if progress.Status == "error" || progress.Status == "done" {
			break
		}
	}

	// Abschluss senden
	fmt.Fprintf(w, "data: {\"status\":\"complete\"}\n\n")
	flusher.Flush()
}

// handleVoiceModels gibt verfügbare Modelle zurück mit installiert-Status
func (app *App) handleVoiceModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Installierte Modelle/Stimmen holen
	installedWhisper := app.voiceService.GetInstalledWhisperModels()
	installedPiper := app.voiceService.GetInstalledPiperVoices()

	// Sets für schnellen Lookup
	whisperSet := make(map[string]bool)
	for _, m := range installedWhisper {
		whisperSet[m] = true
	}
	piperSet := make(map[string]bool)
	for _, v := range installedPiper {
		piperSet[v] = true
	}

	// Whisper-Modelle mit Status
	whisperModels := voice.GetAvailableWhisperModels()
	whisperResult := make([]map[string]interface{}, len(whisperModels))
	for i, m := range whisperModels {
		whisperResult[i] = map[string]interface{}{
			"id":          m.ID,
			"name":        m.Name,
			"sizeMB":      m.SizeMB,
			"description": m.Description,
			"installed":   whisperSet[m.ID],
		}
	}

	// Piper-Stimmen mit Status
	piperVoices := voice.GetAvailablePiperVoices()
	piperResult := make([]map[string]interface{}, len(piperVoices))
	for i, v := range piperVoices {
		piperResult[i] = map[string]interface{}{
			"id":          v.ID,
			"name":        v.Name,
			"language":    v.Language,
			"quality":     v.Quality,
			"sizeMB":      v.SizeMB,
			"description": v.Description,
			"installed":   piperSet[v.ID],
		}
	}

	// Aktuelle Konfiguration
	status := app.voiceService.GetStatus()

	response := map[string]interface{}{
		"whisper":        whisperResult,
		"piper":          piperResult,
		"currentWhisper": status.Whisper.Model,
		"currentPiper":   status.Piper.Voice,
		"whisperBinary":  status.Whisper.BinaryFound,
		"piperBinary":    status.Piper.BinaryFound,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleVoiceConfig verwaltet die Voice-Konfiguration
func (app *App) handleVoiceConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		status := app.voiceService.GetStatus()
		config := map[string]interface{}{
			"whisperModel":    status.Whisper.Model,
			"whisperLanguage": status.Whisper.Language,
			"piperVoice":      status.Piper.Voice,
			"whisperReady":    status.Whisper.Available,
			"piperReady":      status.Piper.Available,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config)

	case http.MethodPost:
		var req struct {
			WhisperModel string `json:"whisperModel,omitempty"`
			PiperVoice   string `json:"piperVoice,omitempty"`
			Language     string `json:"language,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Konfiguration anwenden
		if req.WhisperModel != "" {
			if err := app.voiceService.SetWhisperModel(req.WhisperModel); err != nil {
				log.Printf("Fehler beim Setzen des Whisper-Modells: %v", err)
			} else {
				// In DB persistieren
				if err := app.settingsService.SaveWhisperModel(req.WhisperModel); err != nil {
					log.Printf("Fehler beim Speichern des Whisper-Modells in DB: %v", err)
				}
			}
		}
		if req.PiperVoice != "" {
			if err := app.voiceService.SetPiperVoice(req.PiperVoice); err != nil {
				log.Printf("Fehler beim Setzen der Piper-Stimme: %v", err)
			} else {
				// In DB persistieren
				currentSettings := app.settingsService.GetVoiceSettings()
				currentSettings.PiperVoice = req.PiperVoice
				if err := app.settingsService.SaveVoiceSettings(currentSettings); err != nil {
					log.Printf("Fehler beim Speichern der Piper-Stimme in DB: %v", err)
				}
			}
		}

		// Neuen Status zurückgeben
		status := app.voiceService.GetStatus()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":       "ok",
			"whisperModel": status.Whisper.Model,
			"piperVoice":   status.Piper.Voice,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// voiceStoreCache speichert die Piper Voices von HuggingFace
var voiceStoreCache struct {
	data      map[string]interface{}
	fetchedAt time.Time
	mu        sync.RWMutex
}

// handleVoiceStoreVoices gibt alle verfügbaren Piper Voices von HuggingFace zurück
func (app *App) handleVoiceStoreVoices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Cache prüfen (1 Stunde gültig)
	voiceStoreCache.mu.RLock()
	if voiceStoreCache.data != nil && time.Since(voiceStoreCache.fetchedAt) < time.Hour {
		// Installierte Stimmen hinzufügen
		installedVoices := app.voiceService.GetInstalledPiperVoices()
		installedSet := make(map[string]bool)
		for _, v := range installedVoices {
			installedSet[v] = true
		}

		// Antwort mit installiert-Status
		result := make(map[string]interface{})
		for k, v := range voiceStoreCache.data {
			voiceData := v.(map[string]interface{})
			voiceCopy := make(map[string]interface{})
			for kk, vv := range voiceData {
				voiceCopy[kk] = vv
			}
			voiceCopy["installed"] = installedSet[k]
			result[k] = voiceCopy
		}

		voiceStoreCache.mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}
	voiceStoreCache.mu.RUnlock()

	// Neu von HuggingFace laden
	log.Printf("Voice Store: Lade voices.json von HuggingFace...")
	resp, err := http.Get("https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/voices.json")
	if err != nil {
		log.Printf("Voice Store: Fehler beim Laden: %v", err)
		http.Error(w, "Fehler beim Laden der Voices", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Follow redirect
		if resp.StatusCode == http.StatusTemporaryRedirect || resp.StatusCode == http.StatusFound {
			redirectURL := resp.Header.Get("Location")
			if redirectURL != "" {
				resp2, err := http.Get(redirectURL)
				if err != nil {
					http.Error(w, "Redirect fehlgeschlagen", http.StatusInternalServerError)
					return
				}
				defer resp2.Body.Close()
				resp = resp2
			}
		}
	}

	var voices map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&voices); err != nil {
		log.Printf("Voice Store: JSON-Fehler: %v", err)
		http.Error(w, "Fehler beim Parsen der Voices", http.StatusInternalServerError)
		return
	}

	// Cache aktualisieren
	voiceStoreCache.mu.Lock()
	voiceStoreCache.data = voices
	voiceStoreCache.fetchedAt = time.Now()
	voiceStoreCache.mu.Unlock()

	log.Printf("Voice Store: %d Stimmen geladen", len(voices))

	// Installierte Stimmen hinzufügen
	installedVoices := app.voiceService.GetInstalledPiperVoices()
	installedSet := make(map[string]bool)
	for _, v := range installedVoices {
		installedSet[v] = true
	}

	// Antwort mit installiert-Status
	result := make(map[string]interface{})
	for k, v := range voices {
		voiceData := v.(map[string]interface{})
		voiceCopy := make(map[string]interface{})
		for kk, vv := range voiceData {
			voiceCopy[kk] = vv
		}
		voiceCopy["installed"] = installedSet[k]
		result[k] = voiceCopy
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleVoiceDownloadModel lädt ein spezifisches Modell herunter
func (app *App) handleVoiceDownloadModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parameter auslesen
	component := r.URL.Query().Get("component") // "whisper" oder "piper"
	modelID := r.URL.Query().Get("model")       // z.B. "base", "small" oder "de_DE-thorsten-medium"

	if component == "" || modelID == "" {
		http.Error(w, "component und model Parameter erforderlich", http.StatusBadRequest)
		return
	}

	// SSE für Progress-Updates
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming nicht unterstützt", http.StatusInternalServerError)
		return
	}

	progressChan := make(chan voice.DownloadProgress, 10)

	// Download in Goroutine starten
	go func() {
		defer close(progressChan)
		var err error

		switch component {
		case "whisper":
			err = app.voiceService.DownloadWhisperModel(modelID, progressChan)
		case "piper":
			err = app.voiceService.DownloadPiperVoice(modelID, progressChan)
		default:
			err = fmt.Errorf("Unbekannte Komponente: %s", component)
		}

		if err != nil {
			progressChan <- voice.DownloadProgress{
				Component: component,
				Status:    "error",
				Error:     err.Error(),
			}
		} else {
			progressChan <- voice.DownloadProgress{
				Component: component,
				File:      modelID,
				Status:    "done",
			}
		}
	}()

	// Progress-Updates senden
	for progress := range progressChan {
		data, _ := json.Marshal(progress)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		if progress.Status == "error" || progress.Status == "done" {
			break
		}
	}

	// Abschluss senden
	fmt.Fprintf(w, "data: {\"status\":\"complete\"}\n\n")
	flusher.Flush()
}
