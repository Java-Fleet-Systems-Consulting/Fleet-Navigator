// Package setup - HTTP Handler für Setup-Wizard API
package setup

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// i18n translations for setup messages
var setupTranslations = map[string]map[string]string{
	"de": {
		"checking_engine":        "Prüfe KI-Engine...",
		"engine_installed":       "KI-Engine bereits installiert",
		"engine_ready":           "KI-Engine bereit!",
		"starting_engine":        "Starte KI-Engine...",
		"downloading_model":      "Lade KI-Modell herunter...",
		"start_download":         "Starte Download: %s",
		"resume_download":        "Setze Download fort: %s",
		"downloading":            "Lade %s herunter...",
		"downloading_speed":      "Lade %s herunter... (%.1f MB/s)",
		"download_failed":        "Download fehlgeschlagen",
		"no_model_selected":      "Kein Modell ausgewählt",
		"downloader_not_config":  "Model-Downloader nicht konfiguriert",
		"streaming_not_supported": "Streaming nicht unterstützt",
		"downloading_llama":      "Lade llama-server %s herunter...",
		"downloading_llama_speed": "Lade llama-server herunter... (%.1f MB/s)",
		"getting_version":        "Ermittle neueste Version...",
		"gpu_detected":           "GPU erkannt: %s",
		"no_gpu_cpu_mode":        "Keine GPU - CPU-Modus",
		"detecting_hardware":     "Erkenne Hardware...",
		"nvidia_gpu_detected":    "NVIDIA GPU erkannt → Lade CUDA-Version",
		"amd_intel_gpu_detected": "AMD/Intel GPU erkannt → Lade Vulkan-Version",
		"no_gpu_detected":        "Keine unterstützte GPU → Lade CPU-Version",
		"model_not_found":        "Modell '%s' nicht gefunden auf HuggingFace",
		"download_complete":      "Download abgeschlossen!",
		"extracting":             "Entpacke...",
		"setting_permissions":    "Setze Berechtigungen...",
		"starting_server":        "Starte Server...",
		"server_ready":           "Server bereit!",
		"llama_installed":        "llama-server erfolgreich installiert!",
		"vision_downloading":     "Lade Vision-Modell herunter...",
		"vision_downloading_mmproj": "Lade Vision-Hilfsmodell herunter...",
		"vision_encoder":         "Lade Vision-Encoder: %s",
		"vision_model":           "Lade Vision-Modell: %s",
		"vision_installed":       "Vision-Modell erfolgreich installiert",
		"unknown_vision_model":   "Unbekanntes Vision-Modell: %s",
	},
	"en": {
		"checking_engine":        "Checking AI engine...",
		"engine_installed":       "AI engine already installed",
		"engine_ready":           "AI engine ready!",
		"starting_engine":        "Starting AI engine...",
		"downloading_model":      "Downloading AI model...",
		"start_download":         "Starting download: %s",
		"resume_download":        "Resuming download: %s",
		"downloading":            "Downloading %s...",
		"downloading_speed":      "Downloading %s... (%.1f MB/s)",
		"download_failed":        "Download failed",
		"no_model_selected":      "No model selected",
		"downloader_not_config":  "Model downloader not configured",
		"streaming_not_supported": "Streaming not supported",
		"downloading_llama":      "Downloading llama-server %s...",
		"downloading_llama_speed": "Downloading llama-server... (%.1f MB/s)",
		"getting_version":        "Getting latest version...",
		"gpu_detected":           "GPU detected: %s",
		"no_gpu_cpu_mode":        "No GPU - CPU mode",
		"detecting_hardware":     "Detecting hardware...",
		"nvidia_gpu_detected":    "NVIDIA GPU detected → Loading CUDA version",
		"amd_intel_gpu_detected": "AMD/Intel GPU detected → Loading Vulkan version",
		"no_gpu_detected":        "No supported GPU → Loading CPU version",
		"model_not_found":        "Model '%s' not found on HuggingFace",
		"download_complete":      "Download complete!",
		"extracting":             "Extracting...",
		"setting_permissions":    "Setting permissions...",
		"starting_server":        "Starting server...",
		"server_ready":           "Server ready!",
		"llama_installed":        "llama-server successfully installed!",
		"vision_downloading":     "Downloading Vision model...",
		"vision_downloading_mmproj": "Downloading Vision helper model...",
		"vision_encoder":         "Downloading Vision encoder: %s",
		"vision_model":           "Downloading Vision model: %s",
		"vision_installed":       "Vision model successfully installed",
		"unknown_vision_model":   "Unknown Vision model: %s",
	},
	"tr": {
		"checking_engine":        "Yapay zeka motoru kontrol ediliyor...",
		"engine_installed":       "Yapay zeka motoru zaten kurulu",
		"engine_ready":           "Yapay zeka motoru hazır!",
		"starting_engine":        "Yapay zeka motoru başlatılıyor...",
		"downloading_model":      "Yapay zeka modeli indiriliyor...",
		"start_download":         "İndirme başlatılıyor: %s",
		"resume_download":        "İndirme devam ettiriliyor: %s",
		"downloading":            "%s indiriliyor...",
		"downloading_speed":      "%s indiriliyor... (%.1f MB/s)",
		"download_failed":        "İndirme başarısız",
		"no_model_selected":      "Model seçilmedi",
		"downloader_not_config":  "Model indiricisi yapılandırılmadı",
		"streaming_not_supported": "Akış desteklenmiyor",
		"downloading_llama":      "llama-server %s indiriliyor...",
		"downloading_llama_speed": "llama-server indiriliyor... (%.1f MB/s)",
		"getting_version":        "En son sürüm alınıyor...",
		"gpu_detected":           "GPU algılandı: %s",
		"no_gpu_cpu_mode":        "GPU yok - CPU modu",
		"detecting_hardware":     "Donanım algılanıyor...",
		"nvidia_gpu_detected":    "NVIDIA GPU algılandı → CUDA sürümü yükleniyor",
		"amd_intel_gpu_detected": "AMD/Intel GPU algılandı → Vulkan sürümü yükleniyor",
		"no_gpu_detected":        "Desteklenen GPU yok → CPU sürümü yükleniyor",
		"model_not_found":        "'%s' modeli HuggingFace'de bulunamadı",
		"download_complete":      "İndirme tamamlandı!",
		"extracting":             "Çıkartılıyor...",
		"setting_permissions":    "İzinler ayarlanıyor...",
		"starting_server":        "Sunucu başlatılıyor...",
		"server_ready":           "Sunucu hazır!",
		"llama_installed":        "llama-server başarıyla kuruldu!",
		"vision_downloading":     "Vision modeli indiriliyor...",
		"vision_downloading_mmproj": "Vision yardımcı modeli indiriliyor...",
		"vision_encoder":         "Vision kodlayıcı indiriliyor: %s",
		"vision_model":           "Vision modeli indiriliyor: %s",
		"vision_installed":       "Vision modeli başarıyla kuruldu",
		"unknown_vision_model":   "Bilinmeyen Vision modeli: %s",
	},
	"fr": {
		"checking_engine":        "Vérification du moteur IA...",
		"engine_installed":       "Moteur IA déjà installé",
		"engine_ready":           "Moteur IA prêt !",
		"starting_engine":        "Démarrage du moteur IA...",
		"downloading_model":      "Téléchargement du modèle IA...",
		"start_download":         "Démarrage du téléchargement : %s",
		"resume_download":        "Reprise du téléchargement : %s",
		"downloading":            "Téléchargement de %s...",
		"downloading_speed":      "Téléchargement de %s... (%.1f Mo/s)",
		"download_failed":        "Échec du téléchargement",
		"no_model_selected":      "Aucun modèle sélectionné",
		"downloader_not_config":  "Téléchargeur de modèle non configuré",
		"streaming_not_supported": "Streaming non pris en charge",
		"downloading_llama":      "Téléchargement de llama-server %s...",
		"downloading_llama_speed": "Téléchargement de llama-server... (%.1f Mo/s)",
		"getting_version":        "Récupération de la dernière version...",
		"gpu_detected":           "GPU détecté : %s",
		"no_gpu_cpu_mode":        "Pas de GPU - Mode CPU",
		"detecting_hardware":     "Détection du matériel...",
		"nvidia_gpu_detected":    "GPU NVIDIA détecté → Chargement version CUDA",
		"amd_intel_gpu_detected": "GPU AMD/Intel détecté → Chargement version Vulkan",
		"no_gpu_detected":        "Pas de GPU supporté → Chargement version CPU",
		"model_not_found":        "Modèle '%s' non trouvé sur HuggingFace",
		"download_complete":      "Téléchargement terminé !",
		"extracting":             "Extraction...",
		"setting_permissions":    "Configuration des permissions...",
		"starting_server":        "Démarrage du serveur...",
		"server_ready":           "Serveur prêt !",
		"llama_installed":        "llama-server installé avec succès !",
		"vision_downloading":     "Téléchargement du modèle Vision...",
		"vision_downloading_mmproj": "Téléchargement du modèle auxiliaire Vision...",
		"vision_encoder":         "Téléchargement de l'encodeur Vision : %s",
		"vision_model":           "Téléchargement du modèle Vision : %s",
		"vision_installed":       "Modèle Vision installé avec succès",
		"unknown_vision_model":   "Modèle Vision inconnu : %s",
	},
}

// t translates a message key to the specified language
func t(lang, key string) string {
	if translations, ok := setupTranslations[lang]; ok {
		if msg, ok := translations[key]; ok {
			return msg
		}
	}
	// Fallback to English, then German
	if msg, ok := setupTranslations["en"][key]; ok {
		return msg
	}
	if msg, ok := setupTranslations["de"][key]; ok {
		return msg
	}
	return key
}

// tf translates a message key with format arguments
func tf(lang, key string, args ...interface{}) string {
	return fmt.Sprintf(t(lang, key), args...)
}

// getLang extracts language from request (query param or Accept-Language header)
func getLang(r *http.Request) string {
	// First check query parameter
	if lang := r.URL.Query().Get("lang"); lang != "" {
		if _, ok := setupTranslations[lang]; ok {
			return lang
		}
	}
	// Then check Accept-Language header
	acceptLang := r.Header.Get("Accept-Language")
	if acceptLang != "" {
		// Parse first language code (e.g., "de-DE,de;q=0.9,en;q=0.8" -> "de")
		parts := strings.Split(acceptLang, ",")
		if len(parts) > 0 {
			lang := strings.Split(strings.TrimSpace(parts[0]), "-")[0]
			lang = strings.Split(lang, ";")[0]
			if _, ok := setupTranslations[lang]; ok {
				return lang
			}
		}
	}
	// Default to German for backwards compatibility
	return "de"
}

// LangSetter Interface für Komponenten die Sprache unterstützen
type LangSetter interface {
	SetLang(lang string)
}

// ModelDownloader Interface für Model-Downloads
type ModelDownloader interface {
	DownloadModel(modelID string, progressCh chan<- SetupProgress) error
}

// VoiceDownloader Interface für Voice-Downloads
type VoiceDownloader interface {
	DownloadWhisper(model string, progressCh chan<- SetupProgress) error
	DownloadPiper(voice string, progressCh chan<- SetupProgress) error
}

// VisionDownloader Interface für Vision-Model-Downloads
type VisionDownloader interface {
	DownloadVisionModel(modelID string, progressCh chan<- SetupProgress) error
}

// LlamaServerStarter Interface zum Starten des llama-servers
type LlamaServerStarter interface {
	Start(modelPath string) error
	Stop() error
	IsRunning() bool
	IsHealthy() bool
}

// ModelUpdater Interface zum Aktualisieren des ausgewählten Modells (zur Laufzeit)
type ModelUpdater interface {
	SetSelectedModel(model string)
	SetDefaultModel(model string)
}

// SettingsUpdater Interface zum Speichern in Settings-Datenbank
type SettingsUpdater interface {
	SaveSelectedModel(model string) error
}

// ExpertUpdater Interface zum Aktualisieren der Experten-Modelle
type ExpertUpdater interface {
	UpdateAllExpertsModel(newModel string) error
	SetDefaultModel(model string)
}

// APIHandler enthält die HTTP-Handler für das Setup
type APIHandler struct {
	service          *Service
	modelDownloader  ModelDownloader
	voiceDownloader  VoiceDownloader
	visionDownloader VisionDownloader
	llamaStarter     LlamaServerStarter
	modelsDir        string
	modelUpdater     ModelUpdater
	settingsUpdater  SettingsUpdater
	expertUpdater    ExpertUpdater
}

// NewAPIHandler erstellt einen neuen API-Handler
func NewAPIHandler(service *Service) *APIHandler {
	return &APIHandler{
		service: service,
	}
}

// SetModelDownloader setzt den Model-Downloader
func (h *APIHandler) SetModelDownloader(downloader ModelDownloader) {
	h.modelDownloader = downloader
}

// SetVoiceDownloader setzt den Voice-Downloader
func (h *APIHandler) SetVoiceDownloader(downloader VoiceDownloader) {
	h.voiceDownloader = downloader
}

// SetVisionDownloader setzt den Vision-Downloader
func (h *APIHandler) SetVisionDownloader(downloader VisionDownloader) {
	h.visionDownloader = downloader
}

// SetLlamaStarter setzt den llama-server Starter
func (h *APIHandler) SetLlamaStarter(starter LlamaServerStarter, modelsDir string) {
	h.llamaStarter = starter
	h.modelsDir = modelsDir
}

// SetModelUpdater setzt den Model-Updater für Laufzeit-Updates
func (h *APIHandler) SetModelUpdater(updater ModelUpdater) {
	h.modelUpdater = updater
}

// SetSettingsUpdater setzt den Settings-Updater für DB-Persistenz
func (h *APIHandler) SetSettingsUpdater(updater SettingsUpdater) {
	h.settingsUpdater = updater
}

// SetExpertUpdater setzt den Expert-Updater für Modell-Updates
func (h *APIHandler) SetExpertUpdater(updater ExpertUpdater) {
	h.expertUpdater = updater
}

// HandleStatus GET /api/setup/status
func (h *APIHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"isFirstRun": h.service.IsFirstRun(),
		"state":      h.service.GetState(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleSystemInfo GET /api/setup/system-info
func (h *APIHandler) HandleSystemInfo(w http.ResponseWriter, r *http.Request) {
	info, err := h.service.GetSystemInfo()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// HandleModelRecommendations GET /api/setup/model-recommendations
func (h *APIHandler) HandleModelRecommendations(w http.ResponseWriter, r *http.Request) {
	// Erst Systeminfo sammeln falls noch nicht geschehen
	if h.service.GetState().SystemInfo == nil {
		h.service.GetSystemInfo()
	}

	recommendations := h.service.GetModelRecommendations()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recommendations)
}

// HandleVoiceOptions GET /api/setup/voice-options
func (h *APIHandler) HandleVoiceOptions(w http.ResponseWriter, r *http.Request) {
	options := h.service.GetVoiceOptions()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}

// HandleSetStep POST /api/setup/step
func (h *APIHandler) HandleSetStep(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Step int `json:"step"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Ungültiger Request", http.StatusBadRequest)
		return
	}

	h.service.SetStep(req.Step)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"state":   h.service.GetState(),
	})
}

// HandleSelectModel POST /api/setup/select-model
func (h *APIHandler) HandleSelectModel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ModelID string `json:"modelId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Ungültiger Request", http.StatusBadRequest)
		return
	}

	h.service.SetSelectedModel(req.ModelID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"modelId": req.ModelID,
	})
}

// HandleSelectVoice POST /api/setup/select-voice
func (h *APIHandler) HandleSelectVoice(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled      bool   `json:"enabled"`
		WhisperModel string `json:"whisperModel"`
		PiperVoice   string `json:"piperVoice"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Ungültiger Request", http.StatusBadRequest)
		return
	}

	h.service.SetVoiceOptions(req.Enabled, req.WhisperModel, req.PiperVoice)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// HandleDownloadModel GET /api/setup/download-model (SSE)
func (h *APIHandler) HandleDownloadModel(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	modelID := r.URL.Query().Get("modelId")
	if modelID == "" {
		state := h.service.GetState()
		modelID = state.SelectedModel
	}

	if modelID == "" {
		http.Error(w, t(lang, "no_model_selected"), http.StatusBadRequest)
		return
	}

	// SSE Setup
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, t(lang, "streaming_not_supported"), http.StatusInternalServerError)
		return
	}

	// Progress Channel
	progressCh := make(chan SetupProgress, 10)

	// Download in Goroutine starten
	go func() {
		defer close(progressCh)

		if h.modelDownloader == nil {
			progressCh <- SetupProgress{
				Step:    "model",
				Message: t(lang, "downloader_not_config"),
				Error:   t(lang, "downloader_not_config"),
				Done:    true,
			}
			return
		}

		// Set language if downloader supports it
		if langSetter, ok := h.modelDownloader.(LangSetter); ok {
			langSetter.SetLang(lang)
		}

		err := h.modelDownloader.DownloadModel(modelID, progressCh)
		if err != nil {
			progressCh <- SetupProgress{
				Step:    "model",
				Message: t(lang, "download_failed"),
				Error:   err.Error(),
				Done:    true,
			}
		}
	}()

	// Progress streamen
	for progress := range progressCh {
		data, _ := json.Marshal(progress)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		if progress.Done {
			break
		}
	}
}

// HandleDownloadVoice GET /api/setup/download-voice (SSE)
func (h *APIHandler) HandleDownloadVoice(w http.ResponseWriter, r *http.Request) {
	component := r.URL.Query().Get("component") // "whisper" oder "piper"
	modelID := r.URL.Query().Get("modelId")

	if component == "" {
		http.Error(w, "Komponente fehlt (whisper/piper)", http.StatusBadRequest)
		return
	}

	// SSE Setup
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
	progressCh := make(chan SetupProgress, 10)

	// Download in Goroutine starten
	go func() {
		defer close(progressCh)

		if h.voiceDownloader == nil {
			progressCh <- SetupProgress{
				Step:    component,
				Message: "Voice-Downloader nicht konfiguriert",
				Error:   "Voice-Downloader nicht verfügbar",
				Done:    true,
			}
			return
		}

		var err error
		if component == "whisper" {
			err = h.voiceDownloader.DownloadWhisper(modelID, progressCh)
		} else if component == "piper" {
			err = h.voiceDownloader.DownloadPiper(modelID, progressCh)
		}

		if err != nil {
			progressCh <- SetupProgress{
				Step:    component,
				Message: "Download fehlgeschlagen",
				Error:   err.Error(),
				Done:    true,
			}
		}
	}()

	// Progress streamen
	for progress := range progressCh {
		data, _ := json.Marshal(progress)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		if progress.Done {
			break
		}
	}
}

// HandleComplete POST /api/setup/complete
func (h *APIHandler) HandleComplete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.CompleteSetup(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Variablen für Setup-Status
	startedModel := ""

	state := h.service.GetState()

	// Ausgewähltes Modell für zukünftige Starts speichern
	displayModelName := ""
	if state.SelectedModel != "" {
		configPath := filepath.Join(h.service.dataDir, "default-model.txt")
		modelPath := filepath.Join(h.modelsDir, "library", state.SelectedModel)
		if err := os.WriteFile(configPath, []byte(modelPath), 0644); err != nil {
			log.Printf("[Setup] WARNUNG: Modell-Config konnte nicht gespeichert werden: %v", err)
		} else {
			log.Printf("[Setup] ✅ Default-Modell gespeichert: %s", modelPath)
		}

		// Anzeigename für UI ableiten
		displayModelName = deriveDisplayName(state.SelectedModel)
		log.Printf("[Setup] Display-Modellname: %s", displayModelName)

		// WICHTIG: Laufende Services aktualisieren!
		if h.modelUpdater != nil {
			h.modelUpdater.SetSelectedModel(displayModelName)
			h.modelUpdater.SetDefaultModel(displayModelName)
			log.Printf("[Setup] ✅ ModelService aktualisiert: %s", displayModelName)
		}
		if h.settingsUpdater != nil {
			if err := h.settingsUpdater.SaveSelectedModel(displayModelName); err != nil {
				log.Printf("[Setup] WARNUNG: Settings konnten nicht aktualisiert werden: %v", err)
			} else {
				log.Printf("[Setup] ✅ Settings-DB aktualisiert: %s", displayModelName)
			}
		}
		// Alle Experten auf das neue Modell aktualisieren
		if h.expertUpdater != nil {
			h.expertUpdater.SetDefaultModel(displayModelName)
			if err := h.expertUpdater.UpdateAllExpertsModel(displayModelName); err != nil {
				log.Printf("[Setup] WARNUNG: Experten konnten nicht aktualisiert werden: %v", err)
			} else {
				log.Printf("[Setup] ✅ Experten auf Modell %s aktualisiert", displayModelName)
			}
		}
	}

	// WICHTIG: Prüfe ob llama-server existiert, wenn nicht -> herunterladen
	binDir := filepath.Join(h.service.dataDir, "bin")
	var binaryName string
	if runtime.GOOS == "windows" {
		binaryName = "llama-server.exe"
	} else {
		binaryName = "llama-server"
	}
	serverPath := filepath.Join(binDir, binaryName)

	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		log.Printf("[Setup] llama-server nicht gefunden, starte automatischen Download...")

		// Synchroner Download von llama-server
		downloader := NewLlamaServerDownloader(binDir)

		// Einfacher Progress-Channel der nur loggt
		progressCh := make(chan SetupProgress, 10)
		go func() {
			for p := range progressCh {
				if p.Message != "" {
					log.Printf("[Setup] llama-server: %s (%.0f%%)", p.Message, p.Percent)
				}
			}
		}()

		if err := downloader.Download(progressCh); err != nil {
			log.Printf("[Setup] FEHLER beim llama-server Download: %v", err)
		} else {
			log.Printf("[Setup] ✅ llama-server erfolgreich heruntergeladen!")
		}
	} else {
		log.Printf("[Setup] ✅ llama-server bereits vorhanden: %s", serverPath)
	}

	// Server-Start wird jetzt über /api/setup/start-llama-server gemacht (mit Fortschrittsanzeige)
	// Hier nur prüfen ob alles bereit ist
	if state.SelectedModel != "" {
		modelPath := filepath.Join(h.modelsDir, "library", state.SelectedModel)
		if _, err := os.Stat(modelPath); err == nil {
			startedModel = state.SelectedModel
			log.Printf("[Setup] Modell bereit: %s", modelPath)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":        true,
		"completed":      true,
		"message":        "Setup erfolgreich abgeschlossen!",
		"llamaStarted":   false, // Server wird separat gestartet
		"startedModel":   startedModel,
		"selectedModel":  state.SelectedModel,      // GGUF Dateiname
		"displayModel":   displayModelName,          // Anzeigename für UI
	})
}

// deriveDisplayName leitet einen Anzeigenamen aus dem GGUF-Dateinamen ab
func deriveDisplayName(ggufFilename string) string {
	// qwen2.5-1.5b-instruct-q4_k_m.gguf -> Qwen 2.5 1.5B
	// Qwen2.5-7B-Instruct-Q5_K_M.gguf -> Qwen 2.5 7B
	name := strings.TrimSuffix(ggufFilename, ".gguf")
	name = strings.ToLower(name)

	// Bekannte Muster erkennen
	if strings.Contains(name, "qwen2.5") || strings.Contains(name, "qwen2") {
		if strings.Contains(name, "1.5b") {
			return "Qwen 2.5 1.5B"
		} else if strings.Contains(name, "3b") {
			return "Qwen 2.5 3B"
		} else if strings.Contains(name, "7b") {
			return "Qwen 2.5 7B"
		}
	}

	// Fallback: Dateiname vereinfachen
	name = strings.ReplaceAll(name, "-instruct", "")
	name = strings.ReplaceAll(name, "-q4_k_m", "")
	name = strings.ReplaceAll(name, "-q5_k_m", "")
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")

	return name
}

// HandleStartLlamaServer GET /api/setup/start-llama-server (SSE)
// Startet den llama-server und streamt den Ladefortschritt
func (h *APIHandler) HandleStartLlamaServer(w http.ResponseWriter, r *http.Request) {
	// SSE Headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE nicht unterstützt", http.StatusInternalServerError)
		return
	}

	sendProgress := func(message string, percent float64, done bool, err string) {
		data := map[string]interface{}{
			"message": message,
			"percent": percent,
			"done":    done,
		}
		if err != "" {
			data["error"] = err
		}
		jsonData, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	}

	state := h.service.GetState()
	if state.SelectedModel == "" {
		sendProgress("Kein Modell ausgewählt", 0, true, "Kein Modell ausgewählt")
		return
	}

	modelPath := filepath.Join(h.modelsDir, "library", state.SelectedModel)
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		sendProgress("Modell nicht gefunden", 0, true, "Modell nicht gefunden: "+state.SelectedModel)
		return
	}

	if h.llamaStarter == nil {
		sendProgress("llama-server nicht konfiguriert", 0, true, "llama-server nicht konfiguriert")
		return
	}

	// Server bereits laufend? -> Erst stoppen um VRAM freizugeben
	if h.llamaStarter.IsRunning() {
		sendProgress("Stoppe vorherigen Server...", 2, false, "")
		log.Printf("[Setup] Stoppe laufenden llama-server...")
		if err := h.llamaStarter.Stop(); err != nil {
			log.Printf("[Setup] WARNUNG: Stop fehlgeschlagen: %v", err)
		}
		// Warten damit VRAM freigegeben wird
		time.Sleep(2 * time.Second)
		sendProgress("VRAM wird freigegeben...", 4, false, "")
	}

	sendProgress("Starte KI-Engine...", 5, false, "")

	// Server starten
	if err := h.llamaStarter.Start(modelPath); err != nil {
		sendProgress("Start fehlgeschlagen", 0, true, err.Error())
		return
	}

	sendProgress("Lade Modell in Speicher...", 10, false, "")

	// Warte auf Server-Bereitschaft und tracke VRAM-Fortschritt
	startTime := time.Now()
	maxWaitTime := 180 * time.Second // Max 3 Minuten für große Modelle
	checkInterval := 500 * time.Millisecond

	// Initiales VRAM messen (Baseline)
	initialVRAM := getVRAMUsedMB()
	expectedVRAMMB := estimateModelVRAM(state.SelectedModel)

	log.Printf("[Setup] Server-Start: Initial VRAM=%dMB, erwartet zusätzlich ~%dMB", initialVRAM, expectedVRAMMB)

	for {
		elapsed := time.Since(startTime)
		if elapsed > maxWaitTime {
			sendProgress("Timeout beim Laden", 90, true, "Server antwortet nicht nach 3 Minuten")
			return
		}

		// Prüfe ob Server bereit ist
		if h.llamaStarter.IsHealthy() {
			sendProgress("KI-Engine bereit!", 100, true, "")
			log.Printf("[Setup] ✅ llama-server bereit nach %v", elapsed)
			return
		}

		// VRAM-Fortschritt berechnen
		currentVRAM := getVRAMUsedMB()
		vramDelta := currentVRAM - initialVRAM

		var percent float64
		if expectedVRAMMB > 0 && vramDelta > 0 {
			percent = float64(vramDelta) / float64(expectedVRAMMB) * 80 // Max 80% für VRAM-Laden
			if percent > 80 {
				percent = 80
			}
		} else {
			// Fallback: Zeit-basierter Fortschritt
			percent = float64(elapsed.Seconds()) / 60 * 50 // 50% nach 60 Sekunden
			if percent > 50 {
				percent = 50
			}
		}

		// Mindestens 10% nach Start
		percent = 10 + percent

		message := fmt.Sprintf("Lade Modell... (%d MB geladen)", vramDelta)
		if vramDelta <= 0 {
			message = "Lade Modell in Speicher..."
		}

		sendProgress(message, percent, false, "")
		time.Sleep(checkInterval)
	}
}

// getVRAMUsedMB gibt den aktuell genutzten VRAM in MB zurück
func getVRAMUsedMB() int64 {
	cmd := exec.Command("nvidia-smi", "--query-gpu=memory.used", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	val, _ := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	return val
}

// estimateModelVRAM schätzt den VRAM-Bedarf basierend auf Modellgröße
func estimateModelVRAM(modelID string) int64 {
	modelID = strings.ToLower(modelID)
	switch {
	case strings.Contains(modelID, "1.5b"):
		return 1500 // ~1.5 GB
	case strings.Contains(modelID, "3b"):
		return 2500 // ~2.5 GB
	case strings.Contains(modelID, "7b"):
		return 5000 // ~5 GB
	case strings.Contains(modelID, "9b"), strings.Contains(modelID, "gemma"):
		return 6000 // ~6 GB (Gemma 2 9B)
	case strings.Contains(modelID, "phi-4"):
		return 9500 // ~9.5 GB (Phi-4 14B)
	case strings.Contains(modelID, "14b"):
		return 9500 // ~9.5 GB
	case strings.Contains(modelID, "32b"):
		return 21000 // ~21 GB
	default:
		return 5000 // Fallback
	}
}

// HandleStartLlamaServerAsync POST /api/setup/start-llama-server-async
// Startet den llama-server im Hintergrund und gibt sofort 200 OK zurück.
// Der Server-Start läuft asynchron - ideal für Setup-Wizard, damit UI nicht blockiert.
func (h *APIHandler) HandleStartLlamaServerAsync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	state := h.service.GetState()
	if state.SelectedModel == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Kein Modell ausgewählt",
		})
		return
	}

	modelPath := filepath.Join(h.modelsDir, "library", state.SelectedModel)
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Modell nicht gefunden: " + state.SelectedModel,
		})
		return
	}

	if h.llamaStarter == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "llama-server nicht konfiguriert",
		})
		return
	}

	// Server im Hintergrund starten (nicht blockierend!)
	go func() {
		log.Printf("[Setup] Async Server-Start für: %s", state.SelectedModel)

		// Falls Server läuft, erst stoppen
		if h.llamaStarter.IsRunning() {
			log.Printf("[Setup] Stoppe laufenden llama-server...")
			if err := h.llamaStarter.Stop(); err != nil {
				log.Printf("[Setup] WARNUNG: Stop fehlgeschlagen: %v", err)
			}
			time.Sleep(2 * time.Second) // VRAM freigeben
		}

		// Server starten
		if err := h.llamaStarter.Start(modelPath); err != nil {
			log.Printf("[Setup] ❌ Async Server-Start fehlgeschlagen: %v", err)
			return
		}

		// Warten bis Server bereit ist (im Hintergrund)
		startTime := time.Now()
		maxWait := 180 * time.Second
		for {
			if time.Since(startTime) > maxWait {
				log.Printf("[Setup] ⚠️ Server-Start Timeout nach 3 Minuten")
				return
			}
			if h.llamaStarter.IsHealthy() {
				log.Printf("[Setup] ✅ llama-server bereit nach %v", time.Since(startTime))
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()

	// Sofort OK zurückgeben - Server startet im Hintergrund
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Server-Start im Hintergrund gestartet",
		"model":   state.SelectedModel,
	})
}

// HandleReset POST /api/setup/reset (nur für Entwicklung)
func (h *APIHandler) HandleReset(w http.ResponseWriter, r *http.Request) {
	if err := h.service.ResetSetup(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Setup zurückgesetzt",
	})
}

// HandleSkipSetup POST /api/setup/skip
func (h *APIHandler) HandleSkipSetup(w http.ResponseWriter, r *http.Request) {
	// Erstelle nur die Verzeichnisse ohne Downloads
	dataDir := h.service.dataDir

	dirs := []string{
		dataDir,
		filepath.Join(dataDir, "config"),
		filepath.Join(dataDir, "logs"),
		filepath.Join(dataDir, "data"),
		filepath.Join(dataDir, "models"),
		filepath.Join(dataDir, "models", "library"),
		filepath.Join(dataDir, "models", "custom"),
		filepath.Join(dataDir, "voice"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0700); err != nil {
			http.Error(w, fmt.Sprintf("Verzeichnis erstellen: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Setup als übersprungen markieren
	if err := h.service.CompleteSetup(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Setup übersprungen. Du kannst Modelle später im Model Manager herunterladen.",
	})
}

// RegisterRoutes registriert alle Setup-API-Routen
func (h *APIHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/setup/status", h.HandleStatus)
	mux.HandleFunc("/api/setup/system-info", h.HandleSystemInfo)
	mux.HandleFunc("/api/setup/model-recommendations", h.HandleModelRecommendations)
	mux.HandleFunc("/api/setup/voice-options", h.HandleVoiceOptions)
	mux.HandleFunc("/api/setup/step", h.HandleSetStep)
	mux.HandleFunc("/api/setup/select-model", h.HandleSelectModel)
	mux.HandleFunc("/api/setup/select-voice", h.HandleSelectVoice)
	mux.HandleFunc("/api/setup/download-model", h.HandleDownloadModel)
	mux.HandleFunc("/api/setup/download-voice", h.HandleDownloadVoice)
	mux.HandleFunc("/api/setup/vision-options", h.HandleVisionOptions)
	mux.HandleFunc("/api/setup/vision-settings", h.HandleVisionSettings)
	mux.HandleFunc("/api/setup/download-vision", h.HandleDownloadVision)
	mux.HandleFunc("/api/setup/complete", h.HandleComplete)
	mux.HandleFunc("/api/setup/reset", h.HandleReset)
	mux.HandleFunc("/api/setup/skip", h.HandleSkipSetup)
	// Neue Routen für llama-server Setup
	mux.HandleFunc("/api/setup/create-directories", h.HandleCreateDirectories)
	mux.HandleFunc("/api/setup/download-llama-server", h.HandleDownloadLlamaServer)
	mux.HandleFunc("/api/setup/llama-server-status", h.HandleLlamaServerStatus)
	mux.HandleFunc("/api/setup/start-llama-server", h.HandleStartLlamaServer)
	mux.HandleFunc("/api/setup/start-llama-server-async", h.HandleStartLlamaServerAsync)
}

// ---- Model Downloader Adapter ----

// HuggingFaceDownloader implementiert ModelDownloader für HuggingFace
type HuggingFaceDownloader struct {
	modelsDir string
	client    *http.Client
	lang      string // Language for progress messages
}

// SetLang sets the language for progress messages
func (d *HuggingFaceDownloader) SetLang(lang string) {
	d.lang = lang
}

// getLangOrDefault returns the set language or "de" as default
func (d *HuggingFaceDownloader) getLangOrDefault() string {
	if d.lang == "" {
		return "de"
	}
	return d.lang
}

// NewHuggingFaceDownloader erstellt einen neuen HuggingFace Downloader
func NewHuggingFaceDownloader(modelsDir string) *HuggingFaceDownloader {
	return &HuggingFaceDownloader{
		modelsDir: modelsDir,
		client: &http.Client{
			Timeout: 0, // Kein Timeout für große Downloads
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Redirects folgen (Standard)
				if len(via) >= 10 {
					return fmt.Errorf("zu viele Redirects")
				}
				return nil
			},
		},
	}
}

// isVisionModel prüft ob ein Modell ein Vision-Modell ist (benötigt mmproj)
func isVisionModel(modelID string) bool {
	lowerID := strings.ToLower(modelID)
	return strings.Contains(lowerID, "llava") ||
		strings.Contains(lowerID, "minicpm-v") ||
		strings.Contains(lowerID, "moondream") ||
		strings.Contains(lowerID, "bakllava")
}

// getMMProjURL gibt die mmproj-Download-URL für ein Vision-Modell zurück
// Gibt leeren String zurück wenn kein Vision-Modell oder mmproj nicht bekannt
func getMMProjURL(modelID string) (mmprojURL string, mmprojFilename string) {
	lowerID := strings.ToLower(modelID)

	switch {
	case strings.Contains(lowerID, "llava-v1.6-mistral-7b") || strings.Contains(lowerID, "llava-1.6-mistral"):
		return "https://huggingface.co/cjpais/llava-1.6-mistral-7b-gguf/resolve/main/mmproj-model-f16.gguf",
			"mmproj-model-f16.gguf"
	case strings.Contains(lowerID, "llava-v1.6-vicuna-13b"):
		return "https://huggingface.co/cjpais/llava-v1.6-vicuna-13b-gguf/resolve/main/mmproj-model-f16.gguf",
			"mmproj-model-f16.gguf"
	case strings.Contains(lowerID, "llava-llama-3"):
		return "https://huggingface.co/xtuner/llava-llama-3-8b-v1_1-gguf/resolve/main/llava-llama-3-8b-v1_1-mmproj-f16.gguf",
			"mmproj-llava-llama-3-f16.gguf"
	case strings.Contains(lowerID, "minicpm-v-2.6") || strings.Contains(lowerID, "minicpm-v"):
		return "https://huggingface.co/openbmb/MiniCPM-V-2_6-gguf/resolve/main/mmproj-model-f16.gguf",
			"mmproj-minicpm-v-f16.gguf"
	case strings.Contains(lowerID, "moondream"):
		return "https://huggingface.co/vikhyatk/moondream2/resolve/main/moondream2-mmproj-f16.gguf",
			"moondream2-mmproj-f16.gguf"
	case strings.Contains(lowerID, "bakllava"):
		return "https://huggingface.co/mys/ggml_bakllava-1/resolve/main/mmproj-model-f16.gguf",
			"mmproj-bakllava-f16.gguf"
	case strings.Contains(lowerID, "llava-phi-3") || strings.Contains(lowerID, "llava-phi3"):
		return "https://huggingface.co/xtuner/llava-phi-3-mini-gguf/resolve/main/llava-phi-3-mini-mmproj-f16.gguf",
			"mmproj-llava-phi3-f16.gguf"
	}
	return "", ""
}

// DownloadModel lädt ein Modell von HuggingFace herunter
// Bei Vision-Modellen wird automatisch auch die mmproj-Datei mitgeladen
func (d *HuggingFaceDownloader) DownloadModel(modelID string, progressCh chan<- SetupProgress) error {
	// HuggingFace URL basierend auf Modellgröße
	// 1.5B und 3B: Offizielles Qwen Repo (kleingeschrieben)
	// 7B, 14B, 32B: bartowski Repo (Single-File GGUFs)
	var url string

	// Vision-Modelle haben eigene Repos
	if isVisionModel(modelID) {
		url = d.getVisionModelURL(modelID)
	} else {
		modelLower := strings.ToLower(modelID)
		switch {
		// Meta Llama Modelle
		case strings.Contains(modelLower, "llama-3.1-8b"):
			url = "https://huggingface.co/bartowski/Meta-Llama-3.1-8B-Instruct-GGUF/resolve/main/" + modelID
		case strings.Contains(modelLower, "llama-3.1-70b"):
			url = "https://huggingface.co/bartowski/Meta-Llama-3.1-70B-Instruct-GGUF/resolve/main/" + modelID
		case strings.Contains(modelLower, "llama"):
			// Generischer Llama Fallback
			url = "https://huggingface.co/bartowski/Meta-Llama-3.1-8B-Instruct-GGUF/resolve/main/" + modelID
		// Google Gemma
		case strings.Contains(modelLower, "gemma"):
			url = "https://huggingface.co/bartowski/gemma-2-9b-it-GGUF/resolve/main/" + modelID
		// Microsoft Phi-4
		case strings.Contains(modelLower, "phi-4"):
			url = "https://huggingface.co/bartowski/phi-4-GGUF/resolve/main/" + modelID
		// Qwen Modelle nach Größe
		case strings.Contains(modelLower, "1.5b"):
			url = "https://huggingface.co/Qwen/Qwen2.5-1.5B-Instruct-GGUF/resolve/main/" + modelID
		case strings.Contains(modelLower, "qwen") && strings.Contains(modelLower, "3b"):
			url = "https://huggingface.co/Qwen/Qwen2.5-3B-Instruct-GGUF/resolve/main/" + modelID
		case strings.Contains(modelLower, "qwen") && strings.Contains(modelLower, "7b"):
			url = "https://huggingface.co/bartowski/Qwen2.5-7B-Instruct-GGUF/resolve/main/" + modelID
		case strings.Contains(modelLower, "qwen") && strings.Contains(modelLower, "14b"):
			url = "https://huggingface.co/bartowski/Qwen2.5-14B-Instruct-GGUF/resolve/main/" + modelID
		case strings.Contains(modelLower, "qwen") && strings.Contains(modelLower, "32b"):
			url = "https://huggingface.co/bartowski/Qwen2.5-32B-Instruct-GGUF/resolve/main/" + modelID
		default:
			// Fallback: Versuche bartowski mit dem Modellnamen
			url = "https://huggingface.co/bartowski/Qwen2.5-7B-Instruct-GGUF/resolve/main/" + modelID
		}
	}

	destPath := filepath.Join(d.modelsDir, "library", modelID)

	lang := d.getLangOrDefault()
	progressCh <- SetupProgress{
		Step:    "model",
		Message: tf(lang, "start_download", modelID),
		Percent: 0,
	}

	// HTTP Request mit User-Agent
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("Request erstellen: %w", err)
	}
	req.Header.Set("User-Agent", "Fleet-Navigator/0.7.0")

	log.Printf("[Setup] Download URL: %s", url)

	resp, err := d.client.Do(req)
	if err != nil {
		log.Printf("[Setup] Download Fehler: %v", err)
		return fmt.Errorf("Download starten: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[Setup] HTTP Status: %d, Final URL: %s", resp.StatusCode, resp.Request.URL.String())

	if resp.StatusCode != http.StatusOK {
		// Body lesen für Fehlermeldung
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 500))
		errMsg := strings.TrimSpace(string(body))
		log.Printf("[Setup] ❌ Download fehlgeschlagen: %s", errMsg)

		// Benutzerfreundliche Fehlermeldung
		if resp.StatusCode == 404 {
			return fmt.Errorf(tf(lang, "model_not_found", modelID))
		}
		return fmt.Errorf("Download-Fehler (HTTP %d): %s", resp.StatusCode, errMsg)
	}

	log.Printf("[Setup] ✅ Download gestartet, Größe: %s", resp.Header.Get("Content-Length"))

	// Dateigröße
	totalSize, _ := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)

	// Verzeichnis erstellen
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("Verzeichnis erstellen: %w", err)
	}

	// Datei erstellen
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("Datei erstellen: %w", err)
	}
	defer out.Close()

	// Download mit Progress
	var downloaded int64
	buf := make([]byte, 1024*1024) // 1MB Buffer
	startTime := time.Now()
	lastUpdate := time.Now()

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
			downloaded += int64(n)

			// Progress Update alle 500ms
			if time.Since(lastUpdate) > 500*time.Millisecond {
				elapsed := time.Since(startTime).Seconds()
				speedMBps := float64(downloaded) / elapsed / 1024 / 1024
				percent := float64(downloaded) / float64(totalSize) * 100

				progressCh <- SetupProgress{
					Step:       "model",
					Message:    tf(lang, "downloading", modelID),
					Percent:    percent,
					BytesTotal: totalSize,
					BytesDone:  downloaded,
					SpeedMBps:  speedMBps,
				}
				lastUpdate = time.Now()
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("Download lesen: %w", err)
		}
	}

	// Bei Vision-Modellen auch mmproj-Datei herunterladen
	if isVisionModel(modelID) {
		mmprojURL, mmprojFilename := getMMProjURL(modelID)
		if mmprojURL != "" && mmprojFilename != "" {
			progressCh <- SetupProgress{
				Step:    "model",
				Message: fmt.Sprintf("Lade Vision-Encoder: %s", mmprojFilename),
				Percent: 85,
			}

			mmprojPath := filepath.Join(d.modelsDir, "library", mmprojFilename)
			if err := d.downloadFile(mmprojURL, mmprojPath); err != nil {
				log.Printf("[Setup] ⚠️ mmproj-Download fehlgeschlagen: %v", err)
				// Kein Abbruch - Hauptmodell wurde erfolgreich geladen
				progressCh <- SetupProgress{
					Step:    "model",
					Message: fmt.Sprintf("⚠️ Vision-Encoder konnte nicht geladen werden: %v", err),
					Percent: 95,
				}
			} else {
				log.Printf("[Setup] ✅ Vision-Encoder geladen: %s", mmprojFilename)
			}
		}
	}

	progressCh <- SetupProgress{
		Step:       "model",
		Message:    fmt.Sprintf("Download abgeschlossen: %s", modelID),
		Percent:    100,
		BytesTotal: totalSize,
		BytesDone:  downloaded,
		Done:       true,
	}

	return nil
}

// getVisionModelURL gibt die HuggingFace-URL für ein Vision-Modell zurück
func (d *HuggingFaceDownloader) getVisionModelURL(modelID string) string {
	lowerID := strings.ToLower(modelID)

	switch {
	case strings.Contains(lowerID, "llava-v1.6-mistral-7b") || strings.Contains(lowerID, "llava-1.6-mistral"):
		return "https://huggingface.co/cjpais/llava-1.6-mistral-7b-gguf/resolve/main/" + modelID
	case strings.Contains(lowerID, "llava-v1.6-vicuna-13b"):
		return "https://huggingface.co/cjpais/llava-v1.6-vicuna-13b-gguf/resolve/main/" + modelID
	case strings.Contains(lowerID, "llava-llama-3"):
		return "https://huggingface.co/xtuner/llava-llama-3-8b-v1_1-gguf/resolve/main/" + modelID
	case strings.Contains(lowerID, "minicpm-v"):
		return "https://huggingface.co/openbmb/MiniCPM-V-2_6-gguf/resolve/main/" + modelID
	case strings.Contains(lowerID, "moondream"):
		return "https://huggingface.co/vikhyatk/moondream2/resolve/main/" + modelID
	case strings.Contains(lowerID, "bakllava"):
		return "https://huggingface.co/mys/ggml_bakllava-1/resolve/main/" + modelID
	case strings.Contains(lowerID, "llava-phi"):
		return "https://huggingface.co/xtuner/llava-phi-3-mini-gguf/resolve/main/" + modelID
	}
	// Fallback für unbekannte LLaVA-Varianten
	return "https://huggingface.co/cjpais/llava-1.6-mistral-7b-gguf/resolve/main/" + modelID
}

// downloadFile lädt eine einzelne Datei herunter (ohne Progress-Channel)
func (d *HuggingFaceDownloader) downloadFile(url, destPath string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Fleet-Navigator/0.7.0")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("Download starten: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("Datei erstellen: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("Datei schreiben: %w", err)
	}

	return nil
}

// ---- llama-server Downloader ----

// LlamaServerDownloader lädt llama-server von GitHub herunter
type LlamaServerDownloader struct {
	binDir string
	client *http.Client
	lang   string
}

// SetLang sets the language for progress messages
func (d *LlamaServerDownloader) SetLang(lang string) {
	d.lang = lang
}

// getLangOrDefault returns the set language or "de" as default
func (d *LlamaServerDownloader) getLangOrDefault() string {
	if d.lang == "" {
		return "de"
	}
	return d.lang
}

// NewLlamaServerDownloader erstellt einen neuen llama-server Downloader
func NewLlamaServerDownloader(binDir string) *LlamaServerDownloader {
	return &LlamaServerDownloader{
		binDir: binDir,
		client: &http.Client{
			Timeout: 0, // Kein Timeout für große Downloads
		},
	}
}

// GPUType repräsentiert den erkannten GPU-Typ
type GPUType string

const (
	GPUTypeNone   GPUType = "none"
	GPUTypeCUDA   GPUType = "cuda"   // NVIDIA
	GPUTypeVulkan GPUType = "vulkan" // AMD/Intel (Fallback)
)

// DetectGPUType erkennt den GPU-Typ für den llama-server Download
func DetectGPUType() GPUType {
	hasGPU, gpuName, _ := getGPUInfo()
	if !hasGPU {
		return GPUTypeNone
	}

	gpuNameLower := strings.ToLower(gpuName)

	// NVIDIA GPUs → CUDA
	if strings.Contains(gpuNameLower, "nvidia") ||
		strings.Contains(gpuNameLower, "geforce") ||
		strings.Contains(gpuNameLower, "rtx") ||
		strings.Contains(gpuNameLower, "gtx") ||
		strings.Contains(gpuNameLower, "quadro") ||
		strings.Contains(gpuNameLower, "tesla") {
		log.Printf("[Setup] NVIDIA GPU erkannt: %s → CUDA-Version wird heruntergeladen", gpuName)
		return GPUTypeCUDA
	}

	// AMD/Intel GPUs → Vulkan als Fallback
	if strings.Contains(gpuNameLower, "radeon") ||
		strings.Contains(gpuNameLower, "amd") ||
		strings.Contains(gpuNameLower, "intel") {
		log.Printf("[Setup] AMD/Intel GPU erkannt: %s → Vulkan-Version wird heruntergeladen", gpuName)
		return GPUTypeVulkan
	}

	log.Printf("[Setup] Unbekannte GPU: %s → CPU-Version wird heruntergeladen", gpuName)
	return GPUTypeNone
}

// GetLatestReleaseURL ermittelt die Download-URL für die neueste Version
// gpuType bestimmt welche Version heruntergeladen wird (cuda, vulkan, oder cpu)
func (d *LlamaServerDownloader) GetLatestReleaseURL(gpuType GPUType) (string, string, error) {
	// GitHub API für neueste Release
	apiURL := "https://api.github.com/repos/ggerganov/llama.cpp/releases/latest"
	log.Printf("[Setup] Rufe GitHub API ab: %s", apiURL)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("Request erstellen fehlgeschlagen: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Fleet-Navigator/1.0")

	resp, err := d.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("GitHub API Fehler: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[Setup] GitHub API Antwort: Status %d", resp.StatusCode)

	// Bei Redirect: Folgen
	if resp.StatusCode == 301 || resp.StatusCode == 302 || resp.StatusCode == 307 {
		redirectURL := resp.Header.Get("Location")
		log.Printf("[Setup] Redirect zu: %s", redirectURL)
		resp.Body.Close()

		req2, _ := http.NewRequest("GET", redirectURL, nil)
		req2.Header.Set("Accept", "application/vnd.github.v3+json")
		req2.Header.Set("User-Agent", "Fleet-Navigator/1.0")
		resp, err = d.client.Do(req2)
		if err != nil {
			return "", "", fmt.Errorf("GitHub API Redirect Fehler: %w", err)
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("GitHub API Status %d: %s", resp.StatusCode, string(body))
	}

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", fmt.Errorf("JSON Parse Fehler: %w", err)
	}

	log.Printf("[Setup] Release: %s, %d Assets gefunden", release.TagName, len(release.Assets))

	// Zeige erste 5 Assets für Debugging
	for i, asset := range release.Assets {
		if i < 5 {
			log.Printf("[Setup]   Asset %d: %s", i, asset.Name)
		}
	}

	// Bestimme das gesuchte Asset basierend auf OS und GPU
	var targetAsset string

	switch runtime.GOOS {
	case "linux":
		// Linux: Asset-Namen variieren je nach GPU-Typ
		switch gpuType {
		case GPUTypeCUDA:
			// CUDA-Version: llama-bXXXX-bin-linux-cuda-cu12.4-x64.tar.gz
			targetAsset = "linux-cuda"
			log.Printf("[Setup] Linux + CUDA: Suche nach '%s'", targetAsset)
		case GPUTypeVulkan:
			targetAsset = "ubuntu-vulkan-x64"
			log.Printf("[Setup] Linux + Vulkan: Suche nach '%s'", targetAsset)
		default:
			// CPU-only: llama-bXXXX-bin-ubuntu-x64.tar.gz
			targetAsset = "ubuntu-x64"
			log.Printf("[Setup] Linux + CPU: Suche nach '%s'", targetAsset)
		}
	case "darwin":
		// macOS: Beispiel "llama-b7489-bin-macos-arm64.tar.gz"
		if runtime.GOARCH == "arm64" {
			targetAsset = "macos-arm64"
		} else {
			targetAsset = "macos-x64"
		}
		log.Printf("[Setup] macOS: Suche nach '%s'", targetAsset)
	case "windows":
		// Windows: Beispiel "llama-b7489-bin-win-cuda-12.4-x64.zip"
		switch gpuType {
		case GPUTypeCUDA:
			targetAsset = "win-cuda"
			log.Printf("[Setup] Windows + CUDA: Suche nach '%s'", targetAsset)
		case GPUTypeVulkan:
			targetAsset = "win-vulkan"
			log.Printf("[Setup] Windows + Vulkan: Suche nach '%s'", targetAsset)
		default:
			targetAsset = "win-cpu-x64"
			log.Printf("[Setup] Windows + CPU: Suche nach '%s'", targetAsset)
		}
	default:
		return "", "", fmt.Errorf("Nicht unterstütztes Betriebssystem: %s", runtime.GOOS)
	}

	// Suche das passende Asset
	var foundAsset *struct {
		Name               string
		BrowserDownloadURL string
	}

	for i := range release.Assets {
		asset := &release.Assets[i]
		name := strings.ToLower(asset.Name)

		// Überspringe cudart-* (nur Runtime-Libraries)
		if strings.HasPrefix(name, "cudart-") {
			continue
		}

		// Muss mit llama- beginnen
		if !strings.HasPrefix(name, "llama-") {
			continue
		}

		// Prüfe Dateiendung
		validExt := false
		if runtime.GOOS == "windows" {
			validExt = strings.HasSuffix(name, ".zip")
		} else {
			validExt = strings.HasSuffix(name, ".tar.gz")
		}
		if !validExt {
			continue
		}

		// Prüfe ob das Ziel-Pattern enthalten ist
		if strings.Contains(name, targetAsset) {
			// Bei Linux Standard-Build: Vulkan ausschließen
			if runtime.GOOS == "linux" && targetAsset == "ubuntu-x64" {
				if strings.Contains(name, "vulkan") {
					continue // Überspringe Vulkan-Version
				}
			}
			log.Printf("[Setup] ✅ Gefunden: %s", asset.Name)
			foundAsset = &struct {
				Name               string
				BrowserDownloadURL string
			}{asset.Name, asset.BrowserDownloadURL}
			break
		}
	}

	if foundAsset != nil {
		return foundAsset.BrowserDownloadURL, release.TagName, nil
	}

	// Fallback für Linux: Versuche beliebigen ubuntu-x64 Build
	if runtime.GOOS == "linux" {
		log.Printf("[Setup] Primäres Ziel nicht gefunden, versuche Fallback...")
		for _, asset := range release.Assets {
			name := strings.ToLower(asset.Name)
			if strings.HasPrefix(name, "llama-") &&
				strings.Contains(name, "ubuntu") &&
				strings.Contains(name, "x64") &&
				strings.HasSuffix(name, ".tar.gz") &&
				!strings.Contains(name, "s390x") {
				log.Printf("[Setup] ✅ Fallback gefunden: %s", asset.Name)
				return asset.BrowserDownloadURL, release.TagName, nil
			}
		}
	}

	// Liste alle verfügbaren Assets für Debugging
	log.Printf("[Setup] ❌ Kein passendes Asset gefunden. Verfügbare Assets:")
	for _, asset := range release.Assets {
		log.Printf("[Setup]   - %s", asset.Name)
	}

	return "", "", fmt.Errorf("Keine passende %s %s Version gefunden (gesucht: %s)", runtime.GOOS, runtime.GOARCH, targetAsset)
}

// GetCudaRuntimeURL gibt die URL für das CUDA Runtime Paket zurück (nur für Windows/Linux CUDA)
func (d *LlamaServerDownloader) GetCudaRuntimeURL() (string, error) {
	// GitHub API für neueste Release
	resp, err := d.client.Get("https://api.github.com/repos/ggerganov/llama.cpp/releases/latest")
	if err != nil {
		return "", fmt.Errorf("GitHub API Fehler: %w", err)
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("JSON Parse Fehler: %w", err)
	}

	// Plattform-spezifische Variablen
	var platformPattern, archPattern, fileExt string

	switch runtime.GOOS {
	case "windows":
		platformPattern = "win"
		archPattern = "x64"
		fileExt = ".zip"
	case "linux":
		platformPattern = "linux"
		archPattern = "x64"
		fileExt = ".tar.gz"
	default:
		return "", fmt.Errorf("CUDA Runtime nur für Windows/Linux verfügbar")
	}

	// Suche nach cudart-Paket (CUDA 12.x)
	for _, asset := range release.Assets {
		nameLower := strings.ToLower(asset.Name)
		if strings.HasPrefix(nameLower, "cudart-") &&
			strings.Contains(nameLower, platformPattern) &&
			strings.Contains(nameLower, "cuda-12") &&
			strings.Contains(nameLower, archPattern) &&
			strings.HasSuffix(nameLower, fileExt) {
			log.Printf("[Setup] ✅ CUDA Runtime gefunden: %s", asset.Name)
			return asset.BrowserDownloadURL, nil
		}
	}

	return "", fmt.Errorf("Kein CUDA Runtime Paket gefunden")
}

// IsInstalled prüft ob llama-server bereits installiert ist
func (d *LlamaServerDownloader) IsInstalled() bool {
	var binaryName string
	if runtime.GOOS == "windows" {
		binaryName = "llama-server.exe"
	} else {
		binaryName = "llama-server"
	}
	serverPath := filepath.Join(d.binDir, binaryName)
	_, err := os.Stat(serverPath)
	return err == nil
}

// Download lädt llama-server herunter und entpackt es
// Erkennt automatisch die GPU und lädt die passende Version herunter
func (d *LlamaServerDownloader) Download(progressCh chan<- SetupProgress) error {
	lang := d.getLangOrDefault()
	progressCh <- SetupProgress{
		Step:    "llama-server",
		Message: t(lang, "detecting_hardware"),
		Percent: 0,
	}

	// GPU-Typ erkennen
	gpuType := DetectGPUType()

	var gpuMessage string
	switch gpuType {
	case GPUTypeCUDA:
		gpuMessage = t(lang, "nvidia_gpu_detected")
	case GPUTypeVulkan:
		gpuMessage = t(lang, "amd_intel_gpu_detected")
	default:
		gpuMessage = t(lang, "no_gpu_detected")
	}

	progressCh <- SetupProgress{
		Step:    "llama-server",
		Message: gpuMessage,
		Percent: 2,
	}

	progressCh <- SetupProgress{
		Step:    "llama-server",
		Message: t(lang, "getting_version"),
		Percent: 3,
	}

	// URL ermitteln mit GPU-Typ
	downloadURL, version, err := d.GetLatestReleaseURL(gpuType)
	if err != nil {
		return fmt.Errorf("Release URL ermitteln: %w", err)
	}

	log.Printf("[Setup] llama-server Download: %s (Version %s, GPU: %s)", downloadURL, version, gpuType)

	progressCh <- SetupProgress{
		Step:    "llama-server",
		Message: tf(lang, "downloading_llama", version),
		Percent: 5,
	}

	// Download starten
	resp, err := d.client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("Download starten: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Download fehlgeschlagen (HTTP %d)", resp.StatusCode)
	}

	totalSize, _ := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)

	// Temporäre Datei mit korrekter Endung erstellen
	// Linux/macOS: .tar.gz, Windows: .zip
	tmpExt := ".zip"
	if runtime.GOOS != "windows" {
		tmpExt = ".tar.gz"
	}
	tmpFile, err := os.CreateTemp("", "llama-server-*"+tmpExt)
	if err != nil {
		return fmt.Errorf("Temp-Datei erstellen: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Download mit Progress
	var downloaded int64
	buf := make([]byte, 1024*1024)
	startTime := time.Now()
	lastUpdate := time.Now()

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			tmpFile.Write(buf[:n])
			downloaded += int64(n)

			if time.Since(lastUpdate) > 500*time.Millisecond {
				elapsed := time.Since(startTime).Seconds()
				speedMBps := float64(downloaded) / elapsed / 1024 / 1024
				percent := 5 + (float64(downloaded) / float64(totalSize) * 70) // 5-75%

				progressCh <- SetupProgress{
					Step:       "llama-server",
					Message:    tf(lang, "downloading_llama_speed", speedMBps),
					Percent:    percent,
					BytesTotal: totalSize,
					BytesDone:  downloaded,
					SpeedMBps:  speedMBps,
				}
				lastUpdate = time.Now()
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			tmpFile.Close()
			return fmt.Errorf("Download lesen: %w", readErr)
		}
	}
	tmpFile.Close()

	progressCh <- SetupProgress{
		Step:    "llama-server",
		Message: "Entpacke llama-server...",
		Percent: 80,
	}

	// ZIP entpacken
	if err := d.extractArchive(tmpPath, d.binDir); err != nil {
		return fmt.Errorf("Entpacken: %w", err)
	}

	// CUDA Runtime herunterladen wenn NVIDIA GPU erkannt wurde
	if gpuType == GPUTypeCUDA {
		progressCh <- SetupProgress{
			Step:    "llama-server",
			Message: "Lade CUDA Runtime...",
			Percent: 85,
		}

		cudartURL, err := d.GetCudaRuntimeURL()
		if err != nil {
			log.Printf("[Setup] ⚠️ CUDA Runtime nicht gefunden: %v", err)
		} else {
			log.Printf("[Setup] Lade CUDA Runtime: %s", cudartURL)

			// CUDA Runtime herunterladen
			cudartResp, err := d.client.Get(cudartURL)
			if err != nil {
				log.Printf("[Setup] ⚠️ CUDA Runtime Download fehlgeschlagen: %v", err)
			} else {
				defer cudartResp.Body.Close()

				if cudartResp.StatusCode == http.StatusOK {
					// Temporäre Datei für CUDA Runtime
					cudartTmp, err := os.CreateTemp("", "cudart-*.zip")
					if err == nil {
						cudartPath := cudartTmp.Name()
						defer os.Remove(cudartPath)

						_, err = io.Copy(cudartTmp, cudartResp.Body)
						cudartTmp.Close()

						if err == nil {
							progressCh <- SetupProgress{
								Step:    "llama-server",
								Message: "Entpacke CUDA Runtime...",
								Percent: 90,
							}

							if err := d.extractArchive(cudartPath, d.binDir); err != nil {
								log.Printf("[Setup] ⚠️ CUDA Runtime entpacken fehlgeschlagen: %v", err)
							} else {
								log.Printf("[Setup] ✅ CUDA Runtime installiert")
							}
						}
					}
				}
			}
		}
	}

	progressCh <- SetupProgress{
		Step:    "llama-server",
		Message: t(lang, "llama_installed"),
		Percent: 100,
		Done:    true,
	}

	log.Printf("[Setup] ✅ llama-server %s installiert in: %s", version, d.binDir)
	return nil
}

// extractArchive entpackt eine ZIP- oder tar.gz-Datei
func (d *LlamaServerDownloader) extractArchive(archivePath, destDir string) error {
	// Verzeichnis erstellen
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	// Prüfe Dateityp anhand der Endung
	if strings.HasSuffix(archivePath, ".tar.gz") || strings.HasSuffix(archivePath, ".tgz") {
		return d.extractTarGz(archivePath, destDir)
	}
	return d.extractZip(archivePath, destDir)
}

// extractZip entpackt eine ZIP-Datei (Windows)
func (d *LlamaServerDownloader) extractZip(zipPath, destDir string) error {
	// ZIP öffnen
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	extractedCount := 0
	for _, f := range r.File {
		// Nur relevante Dateien extrahieren (exe, dll)
		name := filepath.Base(f.Name)
		if !strings.HasSuffix(name, ".exe") && !strings.HasSuffix(name, ".dll") {
			continue
		}

		destPath := filepath.Join(destDir, name)
		log.Printf("[Setup] Extrahiere: %s", name)
		extractedCount++

		rc, err := f.Open()
		if err != nil {
			return err
		}

		outFile, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	log.Printf("[Setup] ✅ %d Dateien extrahiert nach: %s", extractedCount, destDir)
	return nil
}

// extractTarGz entpackt eine tar.gz-Datei (Linux/macOS)
func (d *LlamaServerDownloader) extractTarGz(tarGzPath, destDir string) error {
	// Datei öffnen
	file, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Gzip-Reader erstellen
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	// Tar-Reader erstellen
	tr := tar.NewReader(gzr)

	// Relevante Dateiendungen für Linux/macOS
	relevantExtensions := []string{
		"llama-server", // Hauptbinary
		"llama-cli",    // CLI Tool
		".so",          // Shared Libraries
		".dylib",       // macOS Libraries
	}

	isRelevant := func(name string) bool {
		baseName := filepath.Base(name)
		for _, ext := range relevantExtensions {
			if strings.HasSuffix(baseName, ext) || baseName == ext {
				return true
			}
		}
		// Auch ggml-*.so und andere Libraries
		if strings.HasPrefix(baseName, "ggml") || strings.HasPrefix(baseName, "llama") {
			return true
		}
		return false
	}

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Nur reguläre Dateien
		if header.Typeflag != tar.TypeReg {
			continue
		}

		// Nur relevante Dateien extrahieren
		name := filepath.Base(header.Name)
		if !isRelevant(name) {
			continue
		}

		destPath := filepath.Join(destDir, name)

		outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode)|0755)
		if err != nil {
			return err
		}

		_, err = io.Copy(outFile, tr)
		outFile.Close()

		if err != nil {
			return err
		}

		log.Printf("[Setup] Extrahiert: %s", name)
	}

	return nil
}

// HandleDownloadLlamaServer GET /api/setup/download-llama-server (SSE)
func (h *APIHandler) HandleDownloadLlamaServer(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	log.Printf("[Setup] download-llama-server aufgerufen - starte Download")
	// SSE Setup
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, t(lang, "streaming_not_supported"), http.StatusInternalServerError)
		return
	}

	// Progress Channel
	progressCh := make(chan SetupProgress, 10)

	// Downloader erstellen
	binDir := filepath.Join(h.service.dataDir, "bin")
	downloader := NewLlamaServerDownloader(binDir)
	downloader.SetLang(lang)

	// Prüfen ob bereits installiert
	if downloader.IsInstalled() {
		data, _ := json.Marshal(SetupProgress{
			Step:    "llama-server",
			Message: t(lang, "engine_installed"),
			Percent: 100,
			Done:    true,
		})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return
	}

	// Download in Goroutine
	go func() {
		defer close(progressCh)
		if err := downloader.Download(progressCh); err != nil {
			progressCh <- SetupProgress{
				Step:    "llama-server",
				Message: t(lang, "download_failed"),
				Error:   err.Error(),
				Done:    true,
			}
		}
	}()

	// Progress streamen
	for progress := range progressCh {
		data, _ := json.Marshal(progress)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		if progress.Done {
			break
		}
	}
}

// HandleCreateDirectories POST /api/setup/create-directories
func (h *APIHandler) HandleCreateDirectories(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Setup] create-directories aufgerufen")
	dataDir := h.service.dataDir

	dirs := []string{
		dataDir,
		filepath.Join(dataDir, "bin"),
		filepath.Join(dataDir, "config"),
		filepath.Join(dataDir, "logs"),
		filepath.Join(dataDir, "data"),
		filepath.Join(dataDir, "models"),
		filepath.Join(dataDir, "models", "library"),
		filepath.Join(dataDir, "models", "custom"),
		filepath.Join(dataDir, "voice"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			http.Error(w, fmt.Sprintf("Verzeichnis erstellen fehlgeschlagen: %v", err), http.StatusInternalServerError)
			return
		}
	}

	log.Printf("[Setup] ✅ Verzeichnisstruktur erstellt: %s", dataDir)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"dataDir": dataDir,
		"directories": dirs,
	})
}

// HandleLlamaServerStatus GET /api/setup/llama-server-status
func (h *APIHandler) HandleLlamaServerStatus(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Setup] llama-server-status aufgerufen")
	binDir := filepath.Join(h.service.dataDir, "bin")

	// Plattformspezifischer Binary-Name
	var binaryName string
	if runtime.GOOS == "windows" {
		binaryName = "llama-server.exe"
	} else {
		binaryName = "llama-server"
	}
	serverPath := filepath.Join(binDir, binaryName)

	installed := false
	if _, err := os.Stat(serverPath); err == nil {
		installed = true
	}
	log.Printf("[Setup] llama-server Status: installed=%v, path=%s", installed, serverPath)

	// GPU-Erkennung
	gpuType := DetectGPUType()
	hasGPU, gpuName, gpuMemoryGB := getGPUInfo()

	// Prüfe ob GPU-Libraries vorhanden sind (Installation ist GPU-fähig)
	var gpuLibPath string
	var gpuLibName string

	switch runtime.GOOS {
	case "windows":
		gpuLibName = "ggml-cuda.dll"
		gpuLibPath = filepath.Join(binDir, gpuLibName)
	case "linux":
		// Linux: CUDA oder Vulkan
		if gpuType == GPUTypeCUDA {
			gpuLibName = "libggml-cuda.so"
		} else {
			gpuLibName = "libggml-vulkan.so"
		}
		gpuLibPath = filepath.Join(binDir, gpuLibName)
	case "darwin":
		// macOS verwendet Metal (in der Hauptbinary integriert)
		gpuLibName = "metal-integrated"
		gpuLibPath = serverPath // Metal ist in der Binary
	}

	hasGpuLib := false
	if _, err := os.Stat(gpuLibPath); err == nil {
		hasGpuLib = true
	}

	// Für macOS: Metal ist immer verfügbar wenn llama-server installiert ist
	if runtime.GOOS == "darwin" && installed {
		hasGpuLib = true
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"installed":   installed,
		"serverPath":  serverPath,
		"binDir":      binDir,
		"platform":    runtime.GOOS,
		"arch":        runtime.GOARCH,
		"gpuType":     gpuType,
		"hasGPU":      hasGPU,
		"gpuName":     gpuName,
		"gpuMemoryGB": gpuMemoryGB,
		"gpuLibName":  gpuLibName,
		"hasGpuLib":   hasGpuLib,
		"gpuEnabled":  hasGpuLib && (gpuType == GPUTypeCUDA || gpuType == GPUTypeVulkan),
	})
}

// ---- Vision Model Handler ----

// HandleVisionOptions gibt verfügbare Vision-Modelle zurück
func (h *APIHandler) HandleVisionOptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	state := h.service.GetState()
	options := h.service.GetVisionOptions(state.SystemInfo)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}

// HandleVisionSettings speichert Vision-Einstellungen ohne Download
func (h *APIHandler) HandleVisionSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Enabled bool   `json:"enabled"`
		ModelID string `json:"modelId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Vision-Optionen speichern
	h.service.SetVisionOptions(req.Enabled, req.ModelID)

	log.Printf("[Setup] Vision-Einstellungen: enabled=%v, model=%s",
		req.Enabled, req.ModelID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Vision-Einstellungen gespeichert",
	})
}

// HandleDownloadVision startet den Download eines Vision-Modells (SSE)
func (h *APIHandler) HandleDownloadVision(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	// SSE Setup
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, t(lang, "streaming_not_supported"), http.StatusInternalServerError)
		return
	}

	// Model ID aus Query Parameter
	modelID := r.URL.Query().Get("modelId")
	if modelID == "" {
		data, _ := json.Marshal(SetupProgress{
			Step:    "vision",
			Message: t(lang, "no_model_selected"),
			Error:   "missing modelId parameter",
			Done:    true,
		})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return
	}

	log.Printf("[Setup] Vision-Download gestartet für: %s", modelID)

	// Vision-Optionen speichern
	h.service.SetVisionOptions(true, modelID)

	if h.visionDownloader == nil {
		data, _ := json.Marshal(SetupProgress{
			Step:    "vision",
			Message: t(lang, "downloader_not_config"),
			Error:   "downloader not configured",
			Done:    true,
		})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return
	}

	// Set language if downloader supports it
	if langSetter, ok := h.visionDownloader.(LangSetter); ok {
		langSetter.SetLang(lang)
	}

	// Progress Channel
	progressCh := make(chan SetupProgress, 10)

	// Download in Goroutine
	go func() {
		defer close(progressCh)
		if err := h.visionDownloader.DownloadVisionModel(modelID, progressCh); err != nil {
			log.Printf("[Setup] Vision-Download Fehler: %v", err)
			progressCh <- SetupProgress{
				Step:    "vision",
				Message: fmt.Sprintf("Download-Fehler: %v", err),
				Error:   err.Error(),
				Done:    true,
			}
		}
	}()

	// Progress an Client senden
	for progress := range progressCh {
		data, _ := json.Marshal(progress)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		if progress.Done || progress.Error != "" {
			break
		}
	}
}

// ---- Vision Model Downloader ----

// VisionModelDownloader lädt Vision-Modelle von HuggingFace herunter
type VisionModelDownloader struct {
	modelsDir string
	client    *http.Client
	lang      string
}

// SetLang sets the language for progress messages
func (d *VisionModelDownloader) SetLang(lang string) {
	d.lang = lang
}

// getLangOrDefault returns the set language or "de" as default
func (d *VisionModelDownloader) getLangOrDefault() string {
	if d.lang == "" {
		return "de"
	}
	return d.lang
}

// NewVisionModelDownloader erstellt einen neuen Vision-Model Downloader
func NewVisionModelDownloader(modelsDir string) *VisionModelDownloader {
	return &VisionModelDownloader{
		modelsDir: modelsDir,
		client: &http.Client{
			Timeout: 0, // Kein Timeout für große Downloads
		},
	}
}

// VisionModelInfo enthält Informationen zu einem Vision-Modell
type VisionModelInfo struct {
	ModelURL  string // URL zum Haupt-Modell
	MMProjURL string // URL zur mmproj-Datei (Vision Encoder)
	ModelFile string // Dateiname des Modells
	MMProjFile string // Dateiname der mmproj
}

// GetVisionModelURLs gibt die Download-URLs für ein Vision-Modell zurück
func GetVisionModelURLs(modelID string) *VisionModelInfo {
	switch modelID {
	case "llava-v1.6-mistral-7b":
		return &VisionModelInfo{
			ModelURL:   "https://huggingface.co/cjpais/llava-1.6-mistral-7b-gguf/resolve/main/llava-v1.6-mistral-7b.Q4_K_M.gguf",
			MMProjURL:  "https://huggingface.co/cjpais/llava-1.6-mistral-7b-gguf/resolve/main/mmproj-model-f16.gguf",
			ModelFile:  "llava-v1.6-mistral-7b.Q4_K_M.gguf",
			MMProjFile: "mmproj-llava-v1.6-mistral-7b-f16.gguf",
		}
	case "llava-llama-3-8b":
		return &VisionModelInfo{
			ModelURL:   "https://huggingface.co/xtuner/llava-llama-3-8b-v1_1-gguf/resolve/main/llava-llama-3-8b-v1_1-f16.gguf",
			MMProjURL:  "https://huggingface.co/xtuner/llava-llama-3-8b-v1_1-gguf/resolve/main/llava-llama-3-8b-v1_1-mmproj-f16.gguf",
			ModelFile:  "llava-llama-3-8b-v1_1-f16.gguf",
			MMProjFile: "mmproj-llava-llama-3-8b-f16.gguf",
		}
	case "minicpm-v-2.6":
		return &VisionModelInfo{
			ModelURL:   "https://huggingface.co/openbmb/MiniCPM-V-2_6-gguf/resolve/main/ggml-model-Q4_K_M.gguf",
			MMProjURL:  "https://huggingface.co/openbmb/MiniCPM-V-2_6-gguf/resolve/main/mmproj-model-f16.gguf",
			ModelFile:  "minicpm-v-2.6-Q4_K_M.gguf",
			MMProjFile: "mmproj-minicpm-v-2.6-f16.gguf",
		}
	default:
		return nil
	}
}

// DownloadVisionModel lädt ein Vision-Modell herunter (Modell + mmproj)
func (d *VisionModelDownloader) DownloadVisionModel(modelID string, progressCh chan<- SetupProgress) error {
	lang := d.getLangOrDefault()
	info := GetVisionModelURLs(modelID)
	if info == nil {
		return fmt.Errorf(tf(lang, "unknown_vision_model", modelID))
	}

	visionDir := filepath.Join(d.modelsDir, "vision")
	if err := os.MkdirAll(visionDir, 0755); err != nil {
		return fmt.Errorf("Verzeichnis erstellen: %w", err)
	}

	// 1. Haupt-Modell herunterladen
	progressCh <- SetupProgress{
		Step:    "vision",
		Message: tf(lang, "vision_model", info.ModelFile),
		Percent: 0,
	}

	modelPath := filepath.Join(visionDir, info.ModelFile)
	if err := d.downloadFile(info.ModelURL, modelPath, "vision-model", progressCh, 0, 70); err != nil {
		return fmt.Errorf("Modell-Download: %w", err)
	}

	// 2. mmproj (Vision Encoder) herunterladen
	progressCh <- SetupProgress{
		Step:    "vision",
		Message: tf(lang, "vision_encoder", info.MMProjFile),
		Percent: 70,
	}

	mmprojPath := filepath.Join(visionDir, info.MMProjFile)
	if err := d.downloadFile(info.MMProjURL, mmprojPath, "vision-encoder", progressCh, 70, 100); err != nil {
		return fmt.Errorf("mmproj-Download: %w", err)
	}

	progressCh <- SetupProgress{
		Step:    "vision",
		Message: t(lang, "vision_installed"),
		Percent: 100,
		Done:    true,
	}

	log.Printf("[Setup] Vision-Modell installiert: %s", modelID)
	return nil
}

// downloadFile lädt eine Datei mit Progress herunter
func (d *VisionModelDownloader) downloadFile(url, destPath, component string, progressCh chan<- SetupProgress, startPercent, endPercent float64) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Fleet-Navigator/0.7.0")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("Download starten: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	totalSize, _ := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("Datei erstellen: %w", err)
	}
	defer out.Close()

	var downloaded int64
	buf := make([]byte, 1024*1024) // 1MB Buffer
	startTime := time.Now()
	lastUpdate := time.Now()
	percentRange := endPercent - startPercent

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
			downloaded += int64(n)

			if time.Since(lastUpdate) > 500*time.Millisecond {
				elapsed := time.Since(startTime).Seconds()
				speedMBps := float64(downloaded) / elapsed / 1024 / 1024
				filePercent := float64(downloaded) / float64(totalSize)
				totalPercent := startPercent + (filePercent * percentRange)

				progressCh <- SetupProgress{
					Step:       "vision",
					Message:    fmt.Sprintf("Lade %s...", component),
					Percent:    totalPercent,
					BytesTotal: totalSize,
					BytesDone:  downloaded,
					SpeedMBps:  speedMBps,
				}
				lastUpdate = time.Now()
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("Download lesen: %w", err)
		}
	}

	return nil
}
