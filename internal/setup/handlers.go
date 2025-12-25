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
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
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
		"download_error_server_overload": "Download-Server sind derzeit überlastet. Bitte versuchen Sie es in einigen Minuten erneut.",
		"download_error_timeout":  "Verbindung zum Download-Server ist abgebrochen. Bitte prüfen Sie Ihre Internetverbindung und versuchen Sie es erneut.",
		"download_error_no_internet": "Keine Internetverbindung. Bitte prüfen Sie Ihre Netzwerkeinstellungen.",
		"download_error_generic":  "Download fehlgeschlagen. Bitte prüfen Sie Ihre Internetverbindung und versuchen Sie es später erneut.",
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
		"download_error_server_overload": "Download servers are currently overloaded. Please try again in a few minutes.",
		"download_error_timeout":  "Connection to download server timed out. Please check your internet connection and try again.",
		"download_error_no_internet": "No internet connection. Please check your network settings.",
		"download_error_generic":  "Download failed. Please check your internet connection and try again later.",
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
		"download_error_server_overload": "İndirme sunucuları şu anda aşırı yüklü. Lütfen birkaç dakika sonra tekrar deneyin.",
		"download_error_timeout":  "İndirme sunucusuna bağlantı zaman aşımına uğradı. Lütfen internet bağlantınızı kontrol edin ve tekrar deneyin.",
		"download_error_no_internet": "İnternet bağlantısı yok. Lütfen ağ ayarlarınızı kontrol edin.",
		"download_error_generic":  "İndirme başarısız. Lütfen internet bağlantınızı kontrol edin ve daha sonra tekrar deneyin.",
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
		"download_error_server_overload": "Les serveurs de téléchargement sont actuellement surchargés. Veuillez réessayer dans quelques minutes.",
		"download_error_timeout":  "La connexion au serveur de téléchargement a expiré. Veuillez vérifier votre connexion internet et réessayer.",
		"download_error_no_internet": "Pas de connexion internet. Veuillez vérifier vos paramètres réseau.",
		"download_error_generic":  "Échec du téléchargement. Veuillez vérifier votre connexion internet et réessayer plus tard.",
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

// VisionDownloadResult enthält die Pfade des installierten Vision-Modells
type VisionDownloadResult struct {
	ModelPath  string // Pfad zum Vision-Modell (.gguf)
	MmprojPath string // Pfad zur mmproj-Datei
}

// VisionDownloader Interface für Vision-Model-Downloads
type VisionDownloader interface {
	DownloadVisionModel(modelID string, progressCh chan<- SetupProgress) (*VisionDownloadResult, error)
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
	SaveDisclaimerAccepted(accepted bool) error
	GetDisclaimerAccepted() bool
	// Vision Settings für automatische Modell-Wechsel
	SaveVisionSettingsSimple(enabled bool, modelPath, mmprojPath string) error
	// Vision Chaining aktivieren (Vision → Analyse)
	EnableVisionChaining(visionModel string) error
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

// HandleCampaignModels GET /api/setup/campaign-models
// Gibt die vereinfachte Modellauswahl für das Setup zurück (3 Optionen)
func (h *APIHandler) HandleCampaignModels(w http.ResponseWriter, r *http.Request) {
	type CampaignModelResponse struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		Size        string  `json:"size"`
		SizeGB      float64 `json:"sizeGB"`
		Description string  `json:"description"`
		Category    string  `json:"category"`
		Recommended bool    `json:"recommended"`
	}

	models := make([]CampaignModelResponse, len(CampaignModels))
	for i, m := range CampaignModels {
		models[i] = CampaignModelResponse{
			ID:          m.ID,
			Name:        m.Name,
			Size:        m.Size,
			SizeGB:      m.SizeGB,
			Description: m.Description,
			Category:    m.Category,
			Recommended: m.Recommended,
		}
	}

	response := map[string]interface{}{
		"models": models,
		"vision": map[string]interface{}{
			"modelId":     CampaignVisionModel.ModelID,
			"modelFile":   CampaignVisionModel.ModelFile,
			"mmprojFile":  CampaignVisionModel.MMProjFile,
			"totalSizeGB": CampaignVisionModel.TotalSizeGB,
		},
		"mirrorEnabled": MirrorConfig.Enabled,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
		Enabled      bool     `json:"enabled"`
		WhisperModel string   `json:"whisperModel"`
		PiperVoice   string   `json:"piperVoice"`   // Default-Stimme
		PiperVoices  []string `json:"piperVoices"`  // Alle ausgewählten Stimmen
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Ungültiger Request", http.StatusBadRequest)
		return
	}

	// Default-Stimme aus Array nehmen falls nicht explizit gesetzt
	defaultVoice := req.PiperVoice
	if defaultVoice == "" && len(req.PiperVoices) > 0 {
		defaultVoice = req.PiperVoices[0]
	}

	h.service.SetVoiceOptions(req.Enabled, req.WhisperModel, defaultVoice)

	// Log alle ausgewählten Stimmen
	if len(req.PiperVoices) > 0 {
		log.Printf("[Setup] Ausgewählte Piper-Stimmen: %v (Default: %s)", req.PiperVoices, defaultVoice)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"defaultVoice": defaultVoice,
		"voices":       req.PiperVoices,
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
		} else {
			// Erfolg melden
			progressCh <- SetupProgress{
				Step:    "model",
				Message: t(lang, "download_complete"),
				Percent: 100,
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
			// Primäre Stimme herunterladen
			err = h.voiceDownloader.DownloadPiper(modelID, progressCh)

			// Zusätzliche Stimme für Geschlechtervielfalt herunterladen
			if err == nil {
				var secondaryVoice string
				if modelID == "de_DE-eva_k-x_low" {
					// Wenn Eva gewählt, auch Thorsten laden (männlich)
					secondaryVoice = "de_DE-thorsten-medium"
				} else {
					// Wenn andere Stimme gewählt, auch Eva laden (weiblich)
					secondaryVoice = "de_DE-eva_k-x_low"
				}

				progressCh <- SetupProgress{
					Step:    "piper-secondary",
					Message: "Lade zusätzliche Stimme...",
					Percent: 0,
				}

				// Zweite Stimme herunterladen (Fehler ignorieren)
				if err2 := h.voiceDownloader.DownloadPiper(secondaryVoice, progressCh); err2 != nil {
					log.Printf("[Setup] Zusätzliche Stimme konnte nicht geladen werden: %v", err2)
				}
			}
		}

		if err != nil {
			progressCh <- SetupProgress{
				Step:    component,
				Message: "Download fehlgeschlagen",
				Error:   err.Error(),
				Done:    true,
			}
		} else {
			// Erfolg melden
			progressCh <- SetupProgress{
				Step:    component,
				Message: "Download abgeschlossen!",
				Percent: 100,
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
	mux.HandleFunc("/api/setup/campaign-models", h.HandleCampaignModels) // Vereinfachte 3-Modell-Auswahl für Kampagne
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
	// Abschluss-Dialog
	mux.HandleFunc("/api/setup/summary", h.HandleSetupSummary)
	mux.HandleFunc("/api/setup/accept-disclaimer", h.HandleAcceptDisclaimer)
}

// ---- Model Downloader Adapter ----

// MirrorConfig enthält die Konfiguration für den Download-Mirror
// Strategie: Mirror ZUERST (schneller, stabiler) → HuggingFace/GitHub als Fallback
// Bei Mirror-Fehlern wird automatisch auf Original-Quellen gewechselt
var MirrorConfig = struct {
	// BaseURL ist die Basis-URL des Mirrors
	BaseURL string
	// Enabled aktiviert den Mirror als primäre Quelle
	Enabled bool
	// Komponenten-spezifische Pfade
	LlamaServerPath string // z.B. /llama-server/latest/
	ModelsPath      string // z.B. /models/
	VisionPath      string // z.B. /vision/
	WhisperPath     string // z.B. /whisper/
	PiperPath       string // z.B. /piper/
	TesseractPath   string // z.B. /tesseract/
	// Speed-basiertes Fallback
	MinSpeedMBps       float64 // Minimale Geschwindigkeit bevor Fallback (MB/s)
	SpeedCheckAfterSec int     // Nach wie vielen Sekunden Geschwindigkeit prüfen
	// Fallback-Optionen (Mirror → Original wenn Mirror fehlschlägt)
	FallbackToGitHub      bool // Bei llama-server Mirror-Fehler
	FallbackToHuggingFace bool // Bei Model Mirror-Fehler
	// Timeout für Mirror-Requests
	TimeoutSeconds int
}{
	BaseURL:               "https://mirror.java-fleet.com",
	Enabled:               true, // Mirror als Fallback aktiviert
	LlamaServerPath:       "/llama-server/latest/",
	ModelsPath:            "/models/",
	VisionPath:            "/vision/",
	WhisperPath:           "/whisper/",
	PiperPath:             "/piper/",
	TesseractPath:         "/tesseract/",
	MinSpeedMBps:          12.0, // Fallback zu Mirror wenn < 12 MB/s (normal: 80 MB/s)
	SpeedCheckAfterSec:    5,    // Geschwindigkeit nach 5 Sekunden prüfen
	FallbackToGitHub:      true,
	FallbackToHuggingFace: true,
	TimeoutSeconds:        10,
}

// MirrorAssetMapping mappt llama.cpp Asset-Namen zu stabilen Mirror-Namen
// So sind wir unabhängig von Upstream-Namensänderungen
var MirrorAssetMapping = map[string]string{
	// Windows
	"win-cuda-12.4-x64": "win-cuda-12.4-x64.zip",
	"win-cuda-13.1-x64": "win-cuda-13.1-x64.zip",
	"win-vulkan-x64":    "win-vulkan-x64.zip",
	"win-cpu-x64":       "win-cpu-x64.zip",
	// Linux
	"ubuntu-cuda-x64":   "ubuntu-cuda-12.4-x64.tar.gz", // NVIDIA CUDA Version
	"ubuntu-vulkan-x64": "ubuntu-vulkan-x64.tar.gz",
	"ubuntu-x64":        "ubuntu-x64.tar.gz",
	// macOS
	"macos-arm64": "macos-arm64.tar.gz",
	"macos-x64":   "macos-x64.tar.gz",
}

// CampaignModels definiert die vereinfachten Modell-Optionen für das Setup
var CampaignModels = []struct {
	ID          string  // Dateiname
	Name        string  // Anzeigename
	Size        string  // Größe als String
	SizeGB      float64 // Größe in GB
	Description string  // Kurzbeschreibung
	Category    string  // "small", "standard", "large"
	Recommended bool    // Empfohlen für diese Kategorie
}{
	{
		ID:          "Llama-3.2-1B-Instruct-Q4_K_M.gguf",
		Name:        "Llama 3.2 1B",
		Size:        "0.8 GB",
		SizeGB:      0.8,
		Description: "Ultra-kompakt - für sehr schwache Hardware",
		Category:    "small",
		Recommended: false,
	},
	{
		ID:          "Llama-3.2-3B-Instruct-Q4_K_M.gguf",
		Name:        "Llama 3.2 3B",
		Size:        "2.0 GB",
		SizeGB:      2.0,
		Description: "Schnell & kompakt - ideal für schwächere Hardware",
		Category:    "small",
		Recommended: false,
	},
	{
		ID:          "Meta-Llama-3.1-8B-Instruct-Q4_K_M.gguf",
		Name:        "Llama 3.1 8B",
		Size:        "4.6 GB",
		SizeGB:      4.6,
		Description: "Beste Balance aus Geschwindigkeit und Qualität",
		Category:    "standard",
		Recommended: true,
	},
	{
		ID:          "gemma-2-9b-it-Q4_K_M.gguf",
		Name:        "Gemma 2 9B",
		Size:        "5.4 GB",
		SizeGB:      5.4,
		Description: "Google DeepMind - exzellent für Multilingual",
		Category:    "standard",
		Recommended: false,
	},
	{
		ID:          "Qwen2.5-14B-Instruct-Q4_K_M.gguf",
		Name:        "Qwen 2.5 14B",
		Size:        "8.4 GB",
		SizeGB:      8.9,
		Description: "Höchste Qualität - für starke Hardware",
		Category:    "large",
		Recommended: false,
	},
}

// CampaignVisionModel ist das einzige Vision-Modell im vereinfachten Setup
var CampaignVisionModel = struct {
	ModelID     string
	ModelFile   string
	MMProjFile  string
	TotalSizeGB float64
}{
	ModelID:     "minicpm-v-2.6",
	ModelFile:   "MiniCPM-V-2_6-Q4_K_M.gguf",
	MMProjFile:  "MiniCPM-V-2_6-mmproj-f16.gguf",
	TotalSizeGB: 5.4,
}

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
			"MiniCPM-V-2_6-mmproj-f16.gguf"
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

// getHuggingFaceURL gibt die HuggingFace-Download-URL für ein Modell zurück
func (d *HuggingFaceDownloader) getHuggingFaceURL(modelID string) string {
	// Vision-Modelle haben eigene Repos
	if isVisionModel(modelID) {
		return d.getVisionModelURL(modelID)
	}

	modelLower := strings.ToLower(modelID)
	switch {
	// Meta Llama 3.2 Modelle (neue kleine Modelle)
	case strings.Contains(modelLower, "llama-3.2-1b"):
		return "https://huggingface.co/bartowski/Llama-3.2-1B-Instruct-GGUF/resolve/main/" + modelID
	case strings.Contains(modelLower, "llama-3.2-3b"):
		return "https://huggingface.co/bartowski/Llama-3.2-3B-Instruct-GGUF/resolve/main/" + modelID
	// Meta Llama 3.1 Modelle
	case strings.Contains(modelLower, "llama-3.1-8b"):
		return "https://huggingface.co/bartowski/Meta-Llama-3.1-8B-Instruct-GGUF/resolve/main/" + modelID
	case strings.Contains(modelLower, "llama-3.1-70b"):
		return "https://huggingface.co/bartowski/Meta-Llama-3.1-70B-Instruct-GGUF/resolve/main/" + modelID
	case strings.Contains(modelLower, "llama"):
		return "https://huggingface.co/bartowski/Meta-Llama-3.1-8B-Instruct-GGUF/resolve/main/" + modelID
	// Google Gemma
	case strings.Contains(modelLower, "gemma"):
		return "https://huggingface.co/bartowski/gemma-2-9b-it-GGUF/resolve/main/" + modelID
	// Microsoft Phi-4
	case strings.Contains(modelLower, "phi-4"):
		return "https://huggingface.co/bartowski/phi-4-GGUF/resolve/main/" + modelID
	// Qwen Modelle nach Größe
	case strings.Contains(modelLower, "1.5b"):
		return "https://huggingface.co/Qwen/Qwen2.5-1.5B-Instruct-GGUF/resolve/main/" + modelID
	case strings.Contains(modelLower, "qwen") && strings.Contains(modelLower, "3b"):
		return "https://huggingface.co/Qwen/Qwen2.5-3B-Instruct-GGUF/resolve/main/" + modelID
	case strings.Contains(modelLower, "qwen") && strings.Contains(modelLower, "7b"):
		return "https://huggingface.co/bartowski/Qwen2.5-7B-Instruct-GGUF/resolve/main/" + modelID
	case strings.Contains(modelLower, "qwen") && strings.Contains(modelLower, "14b"):
		return "https://huggingface.co/bartowski/Qwen2.5-14B-Instruct-GGUF/resolve/main/" + modelID
	case strings.Contains(modelLower, "qwen") && strings.Contains(modelLower, "32b"):
		return "https://huggingface.co/bartowski/Qwen2.5-32B-Instruct-GGUF/resolve/main/" + modelID
	default:
		return "https://huggingface.co/bartowski/Qwen2.5-7B-Instruct-GGUF/resolve/main/" + modelID
	}
}

// DownloadModel lädt ein Modell herunter
// Strategie: Mirror ZUERST (schneller, stabiler) → HuggingFace-Fallback bei Fehler
// Bei Vision-Modellen wird automatisch auch die mmproj-Datei mitgeladen
// Unterstützt Multi-Connection-Downloads für große Dateien (>100MB)
func (d *HuggingFaceDownloader) DownloadModel(modelID string, progressCh chan<- SetupProgress) error {
	lang := d.getLangOrDefault()
	destPath := filepath.Join(d.modelsDir, "library", modelID)

	progressCh <- SetupProgress{
		Step:    "model",
		Message: tf(lang, "start_download", modelID),
		Percent: 0,
	}

	// Verzeichnis erstellen
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("Verzeichnis erstellen: %w", err)
	}

	// Mirror URL (primär)
	var mirrorURL string
	if MirrorConfig.Enabled {
		mirrorURL = MirrorConfig.BaseURL + MirrorConfig.ModelsPath + modelID
	}

	// HuggingFace URL (Fallback)
	huggingfaceURL := d.getHuggingFaceURL(modelID)

	if MirrorConfig.Enabled {
		log.Printf("[Setup] 🚀 Primäre Download URL (Mirror): %s", mirrorURL)
		log.Printf("[Setup] Fallback URL (HuggingFace): %s", huggingfaceURL)
	} else {
		log.Printf("[Setup] Primäre Download URL: %s", huggingfaceURL)
	}

	var err error

	// Strategie: Mirror ZUERST (schneller, stabiler), HuggingFace als Fallback
	if MirrorConfig.Enabled && mirrorURL != "" {
		progressCh <- SetupProgress{
			Step:    "model",
			Message: "Lade von Mirror...",
			Percent: 0,
		}
		_, err = d.tryDownloadWithSpeedCheck(mirrorURL, destPath, modelID, progressCh, lang)

		if err == nil {
			log.Printf("[Setup] ✅ Download von Mirror erfolgreich!")
			return nil
		}

		// Mirror fehlgeschlagen - Fallback zu HuggingFace
		if MirrorConfig.FallbackToHuggingFace {
			log.Printf("[Setup] ⚠️ Mirror fehlgeschlagen (%v), wechsle zu HuggingFace...", err)
			os.Remove(destPath) // Teildatei löschen
			progressCh <- SetupProgress{
				Step:    "model",
				Message: "Mirror nicht erreichbar, wechsle zu HuggingFace...",
				Percent: 0,
			}
			_, err = d.tryDownloadWithSpeedCheck(huggingfaceURL, destPath, modelID, progressCh, lang)
		}
	} else {
		// Mirror deaktiviert - direkt von HuggingFace
		progressCh <- SetupProgress{
			Step:    "model",
			Message: "Lade von HuggingFace...",
			Percent: 0,
		}
		_, err = d.tryDownloadWithSpeedCheck(huggingfaceURL, destPath, modelID, progressCh, lang)
	}

	return err
}

// tryDownloadWithSpeedCheck versucht Download und gibt die Geschwindigkeit zurück
// Wenn Geschwindigkeit unter MinSpeedMBps fällt, wird der Download abgebrochen
// Rückgabe: (durchschnittliche Geschwindigkeit in MB/s, Fehler)
func (d *HuggingFaceDownloader) tryDownloadWithSpeedCheck(url, destPath, modelID string, progressCh chan<- SetupProgress, lang string) (float64, error) {
	err := d.tryDownload(url, destPath, modelID, progressCh, lang)
	if err != nil {
		// Prüfe ob es ein SlowDownloadError ist
		if slowErr, ok := err.(*SlowDownloadError); ok {
			return slowErr.Speed, err
		}
		// Anderer Fehler (Server nicht erreichbar, 404, etc.)
		return 0, err
	}
	return 10.0, nil // Erfolgreicher Download = schnell genug
}

// tryDownload versucht einen Download von der angegebenen URL
func (d *HuggingFaceDownloader) tryDownload(url, destPath, modelID string, progressCh chan<- SetupProgress, lang string) error {
	// HEAD Request um Größe und Range-Support zu prüfen
	// WICHTIG: HuggingFace gibt 302 Redirect zurück - wir müssen Redirects manuell folgen
	var totalSize int64
	var finalURL = url
	var acceptRanges string
	useMultiConnection := false

	// Client der nicht automatisch redirected (für HEAD)
	noRedirectClient := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Manuell Redirects folgen (max 10)
	currentURL := url
	for i := 0; i < 10; i++ {
		headReq, _ := http.NewRequest("HEAD", currentURL, nil)
		headReq.Header.Set("User-Agent", "Fleet-Navigator/0.7.0")
		headResp, err := noRedirectClient.Do(headReq)
		if err != nil {
			return fmt.Errorf("HEAD request fehlgeschlagen: %w", err)
		}
		headResp.Body.Close()

		// Prüfe auf Redirect
		if headResp.StatusCode >= 300 && headResp.StatusCode < 400 {
			location := headResp.Header.Get("Location")
			if location == "" {
				break
			}
			log.Printf("[Setup] Folge Redirect %d: %s", headResp.StatusCode, location)
			currentURL = location
			finalURL = location
			continue
		}

		// Fehler-Status
		if headResp.StatusCode == 404 {
			return fmt.Errorf("Modell nicht gefunden (404)")
		}
		if headResp.StatusCode >= 400 {
			return fmt.Errorf("Server-Fehler: %d", headResp.StatusCode)
		}

		// Erfolg - Header lesen
		if headResp.StatusCode == http.StatusOK {
			totalSize, _ = strconv.ParseInt(headResp.Header.Get("Content-Length"), 10, 64)
			acceptRanges = headResp.Header.Get("Accept-Ranges")
			finalURL = currentURL
			log.Printf("[Setup] Finale URL: %s, Größe: %.1f MB, Accept-Ranges: %s",
				finalURL, float64(totalSize)/(1024*1024), acceptRanges)
		}
		break
	}

	// Aktualisiere URL auf finale URL (nach Redirects)
	url = finalURL

	// Multi-Connection für Dateien > 100MB wenn Server Range Requests unterstützt
	if totalSize > 100*1024*1024 && acceptRanges == "bytes" {
		useMultiConnection = true
		log.Printf("[Setup] ⚡ Multi-Connection: size=%d MB > 100MB, ranges='%s' → JA",
			totalSize/(1024*1024), acceptRanges)
	} else {
		log.Printf("[Setup] ⚠️ Single-Connection: size=%d MB, ranges='%s'",
			totalSize/(1024*1024), acceptRanges)
	}

	if useMultiConnection {
		// Multi-Connection Download (8 parallele Verbindungen)
		log.Printf("[Setup] 🚀 Starte Multi-Connection Download mit 8 Verbindungen...")
		return d.downloadModelMulti(url, destPath, modelID, totalSize, 8, progressCh, lang)
	}

	// Single-Connection Download (Fallback)
	log.Printf("[Setup] Starte Single-Connection Download...")
	return d.downloadModelSingle(url, destPath, modelID, progressCh, lang)
}

// downloadModelSingle - Standard Single-Connection Download
func (d *HuggingFaceDownloader) downloadModelSingle(url, destPath, modelID string, progressCh chan<- SetupProgress, lang string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("Request erstellen: %w", err)
	}
	req.Header.Set("User-Agent", "Fleet-Navigator/0.7.0")

	resp, err := d.client.Do(req)
	if err != nil {
		log.Printf("[Setup] Download Fehler: %v", err)
		return fmt.Errorf("Download starten: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[Setup] HTTP Status: %d, Final URL: %s", resp.StatusCode, resp.Request.URL.String())

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 500))
		errMsg := strings.TrimSpace(string(body))
		log.Printf("[Setup] ❌ Download fehlgeschlagen: %s", errMsg)

		if resp.StatusCode == 404 {
			return fmt.Errorf("%s", tf(lang, "model_not_found", modelID))
		}
		return fmt.Errorf("Download-Fehler (HTTP %d): %s", resp.StatusCode, errMsg)
	}

	log.Printf("[Setup] ✅ Download gestartet, Größe: %s", resp.Header.Get("Content-Length"))

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
	speedChecked := false

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
			downloaded += int64(n)

			elapsed := time.Since(startTime).Seconds()
			speedMBps := float64(downloaded) / elapsed / 1024 / 1024

			// Speed-Check nach konfigurierten Sekunden (nur einmal)
			if !speedChecked && MirrorConfig.Enabled && elapsed >= float64(MirrorConfig.SpeedCheckAfterSec) {
				speedChecked = true
				if speedMBps < MirrorConfig.MinSpeedMBps {
					log.Printf("[Setup] ⚠️ Download zu langsam: %.2f MB/s (min: %.2f MB/s) nach %.0fs",
						speedMBps, MirrorConfig.MinSpeedMBps, elapsed)
					out.Close()
					return &SlowDownloadError{Speed: speedMBps}
				}
				log.Printf("[Setup] ✓ Geschwindigkeit OK: %.2f MB/s nach %.0fs", speedMBps, elapsed)
			}

			if time.Since(lastUpdate) > 500*time.Millisecond {
				percent := float64(downloaded) / float64(totalSize) * 100

				progressCh <- SetupProgress{
					Step:       "model",
					Message:    tf(lang, "downloading_speed", modelID, speedMBps),
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

	return nil
}

// SlowDownloadError wird zurückgegeben wenn die Download-Geschwindigkeit zu langsam ist
type SlowDownloadError struct {
	Speed float64 // Gemessene Geschwindigkeit in MB/s
}

func (e *SlowDownloadError) Error() string {
	return fmt.Sprintf("Download zu langsam: %.2f MB/s", e.Speed)
}

// downloadModelMulti - Multi-Connection Download für große Dateien
func (d *HuggingFaceDownloader) downloadModelMulti(url, destPath, modelID string, totalSize int64, numConnections int, progressCh chan<- SetupProgress, lang string) error {
	chunkSize := totalSize / int64(numConnections)

	// Temporäre Dateien für jeden Chunk
	tempFiles := make([]string, numConnections)
	for i := 0; i < numConnections; i++ {
		tempFiles[i] = fmt.Sprintf("%s.part%d", destPath, i)
	}

	// Progress-Tracking
	var progressMu sync.Mutex
	chunkProgress := make([]int64, numConnections)
	startTime := time.Now()

	// Error-Channel für Goroutines
	errChan := make(chan error, numConnections)
	var wg sync.WaitGroup

	// Parallele Downloads starten
	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(connID int) {
			defer wg.Done()

			start := int64(connID) * chunkSize
			end := start + chunkSize - 1
			if connID == numConnections-1 {
				end = totalSize - 1
			}

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				errChan <- fmt.Errorf("Conn %d: Request erstellen: %w", connID, err)
				return
			}
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
			req.Header.Set("User-Agent", "Fleet-Navigator/0.7.0")

			client := &http.Client{Timeout: 30 * time.Minute}
			resp, err := client.Do(req)
			if err != nil {
				errChan <- fmt.Errorf("Conn %d: Download starten: %w", connID, err)
				return
			}
			defer resp.Body.Close()

			log.Printf("[Setup] Conn %d: HTTP %d, Range: bytes=%d-%d, Content-Length: %s",
				connID, resp.StatusCode, start, end, resp.Header.Get("Content-Length"))

			if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
				errChan <- fmt.Errorf("Conn %d: HTTP %d", connID, resp.StatusCode)
				return
			}

			// Warnung wenn Server Range ignoriert (200 statt 206)
			if resp.StatusCode == http.StatusOK {
				log.Printf("[Setup] ⚠️ Conn %d: Server ignoriert Range-Header! Kompletter Download statt Chunk.", connID)
			}

			out, err := os.Create(tempFiles[connID])
			if err != nil {
				errChan <- fmt.Errorf("Conn %d: Temp-Datei erstellen: %w", connID, err)
				return
			}
			defer out.Close()

			buf := make([]byte, 64*1024)
			for {
				n, err := resp.Body.Read(buf)
				if n > 0 {
					out.Write(buf[:n])
					progressMu.Lock()
					chunkProgress[connID] += int64(n)
					progressMu.Unlock()
				}
				if err == io.EOF {
					break
				}
				if err != nil {
					errChan <- fmt.Errorf("Conn %d: Lesen: %w", connID, err)
					return
				}
			}
		}(i)
	}

	// Progress-Reporter
	done := make(chan bool)
	go func() {
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				progressMu.Lock()
				var totalDownloaded int64
				for _, p := range chunkProgress {
					totalDownloaded += p
				}
				progressMu.Unlock()

				elapsed := time.Since(startTime).Seconds()
				speed := float64(totalDownloaded) / elapsed / (1024 * 1024)
				percent := float64(totalDownloaded) / float64(totalSize) * 100

				progressCh <- SetupProgress{
					Step:       "model",
					Message:    tf(lang, "downloading", modelID),
					Percent:    percent,
					BytesTotal: totalSize,
					BytesDone:  totalDownloaded,
					SpeedMBps:  speed,
				}
			}
		}
	}()

	wg.Wait()
	close(done)
	close(errChan)

	// Fehler prüfen
	for err := range errChan {
		for _, tf := range tempFiles {
			os.Remove(tf)
		}
		return err
	}

	// Chunks zusammenfügen
	log.Printf("[Setup] Füge %d Chunks zusammen...", numConnections)
	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("Zieldatei erstellen: %w", err)
	}
	defer outFile.Close()

	for i, tf := range tempFiles {
		chunk, err := os.Open(tf)
		if err != nil {
			return fmt.Errorf("Chunk %d öffnen: %w", i, err)
		}

		_, err = io.Copy(outFile, chunk)
		chunk.Close()
		if err != nil {
			return fmt.Errorf("Chunk %d kopieren: %w", i, err)
		}

		os.Remove(tf)
	}

	elapsed := time.Since(startTime).Seconds()
	avgSpeed := float64(totalSize) / elapsed / (1024 * 1024)
	log.Printf("[Setup] Multi-Connection Download abgeschlossen: %.1f MB in %.1fs (%.1f MB/s)",
		float64(totalSize)/(1024*1024), elapsed, avgSpeed)

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

// GetMirrorAssetKey gibt den Schlüssel für MirrorAssetMapping zurück
func (d *LlamaServerDownloader) GetMirrorAssetKey(gpuType GPUType) string {
	switch runtime.GOOS {
	case "windows":
		switch gpuType {
		case GPUTypeCUDA:
			return "win-cuda-12.4-x64"
		case GPUTypeVulkan:
			return "win-vulkan-x64"
		default:
			return "win-cpu-x64"
		}
	case "linux":
		switch gpuType {
		case GPUTypeCUDA:
			return "ubuntu-cuda-x64"
		case GPUTypeVulkan:
			return "ubuntu-vulkan-x64"
		default:
			return "ubuntu-x64"
		}
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "macos-arm64"
		}
		return "macos-x64"
	}
	return ""
}

// TryMirrorDownload versucht llama-server vom eigenen Mirror zu laden
// Gibt URL und nil zurück bei Erfolg, oder "", error bei Fehler
func (d *LlamaServerDownloader) TryMirrorDownload(gpuType GPUType) (string, error) {
	if !MirrorConfig.Enabled {
		return "", fmt.Errorf("Mirror ist deaktiviert")
	}

	assetKey := d.GetMirrorAssetKey(gpuType)
	if assetKey == "" {
		return "", fmt.Errorf("Kein Asset-Mapping für %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	assetName, ok := MirrorAssetMapping[assetKey]
	if !ok {
		return "", fmt.Errorf("Asset %s nicht im Mirror-Mapping", assetKey)
	}

	mirrorURL := MirrorConfig.BaseURL + MirrorConfig.LlamaServerPath + assetName
	log.Printf("[Setup] Versuche Mirror-Download: %s", mirrorURL)

	// Schneller HEAD-Request um Verfügbarkeit zu prüfen
	client := &http.Client{
		Timeout: time.Duration(MirrorConfig.TimeoutSeconds) * time.Second,
	}

	req, err := http.NewRequest("HEAD", mirrorURL, nil)
	if err != nil {
		return "", fmt.Errorf("Mirror Request erstellen: %w", err)
	}
	req.Header.Set("User-Agent", "Fleet-Navigator/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Mirror nicht erreichbar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Mirror Status %d", resp.StatusCode)
	}

	log.Printf("[Setup] ✅ Mirror verfügbar: %s", mirrorURL)
	return mirrorURL, nil
}

// GetLatestReleaseURL ermittelt die Download-URL für die neueste Version
// gpuType bestimmt welche Version heruntergeladen wird (cuda, vulkan, oder cpu)
func (d *LlamaServerDownloader) GetLatestReleaseURL(gpuType GPUType) (string, string, error) {
	// GitHub API für neueste Release
	apiURL := "https://api.github.com/repos/ggerganov/llama.cpp/releases/latest"
	log.Printf("[Setup] Rufe GitHub API ab: %s", apiURL)

	// Retry-Konfiguration für transiente Fehler (502, 503, 504)
	maxRetries := 3
	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 2s, 4s, 8s
			backoffDuration := time.Duration(1<<attempt) * time.Second
			log.Printf("[Setup] GitHub API Retry %d/%d nach %v...", attempt+1, maxRetries, backoffDuration)
			time.Sleep(backoffDuration)
		}

		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return "", "", fmt.Errorf("Request erstellen fehlgeschlagen: %w", err)
		}
		req.Header.Set("Accept", "application/vnd.github.v3+json")
		req.Header.Set("User-Agent", "Fleet-Navigator/1.0")

		resp, err = d.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("GitHub API Fehler: %w", err)
			continue
		}

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
				lastErr = fmt.Errorf("GitHub API Redirect Fehler: %w", err)
				continue
			}
		}

		// Transiente Fehler (502, 503, 504) - Retry
		if resp.StatusCode == 502 || resp.StatusCode == 503 || resp.StatusCode == 504 {
			log.Printf("[Setup] GitHub API temporärer Fehler (Status %d), versuche erneut...", resp.StatusCode)
			resp.Body.Close()
			lastErr = fmt.Errorf("GitHub API temporärer Fehler: Status %d", resp.StatusCode)
			continue
		}

		// Erfolg oder nicht-transienter Fehler
		if resp.StatusCode == 200 {
			lastErr = nil
			break
		}

		// Andere Fehler - nicht wiederholen
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return "", "", fmt.Errorf("GitHub API Status %d: %s", resp.StatusCode, string(body))
	}

	if lastErr != nil {
		return "", "", fmt.Errorf("GitHub API nach %d Versuchen fehlgeschlagen: %w", maxRetries, lastErr)
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
			// CUDA-Version: llama-bXXXX-bin-ubuntu-cuda-12.4-x64.tar.gz (neues Format ohne "cu")
			targetAsset = "ubuntu-cuda"
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

// GetLatestReleaseURLFromSourceForge ermittelt die Download-URL von SourceForge (Fallback)
// SourceForge Mirror: https://sourceforge.net/projects/llama-cpp.mirror/files/
// Hinweis: Linux-CUDA ist auf SourceForge NICHT verfügbar!
func (d *LlamaServerDownloader) GetLatestReleaseURLFromSourceForge(gpuType GPUType) (string, string, error) {
	log.Printf("[Setup] SourceForge Fallback wird verwendet...")

	// SourceForge Dateien-Seite abrufen (RSS Feed für Versionsliste)
	rssURL := "https://sourceforge.net/projects/llama-cpp.mirror/rss?path=/"
	log.Printf("[Setup] Rufe SourceForge RSS ab: %s", rssURL)

	req, err := http.NewRequest("GET", rssURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("SourceForge Request erstellen fehlgeschlagen: %w", err)
	}
	req.Header.Set("User-Agent", "Fleet-Navigator/1.0")

	resp, err := d.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("SourceForge Fehler: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("SourceForge Status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("SourceForge Response lesen: %w", err)
	}

	// Suche nach der neuesten Version (bXXXX Format) im RSS Feed
	// RSS enthält <link>...files/bXXXX/...</link> Elemente
	rssContent := string(body)
	versionPattern := regexp.MustCompile(`/files/(b\d+)/`)
	matches := versionPattern.FindAllStringSubmatch(rssContent, -1)

	if len(matches) == 0 {
		return "", "", fmt.Errorf("Keine Versionen auf SourceForge gefunden")
	}

	// Finde die höchste Version (höchste Build-Nummer)
	var latestVersion string
	var latestBuildNum int
	for _, match := range matches {
		if len(match) > 1 {
			versionStr := match[1]
			// Extrahiere Build-Nummer (b7503 → 7503)
			if len(versionStr) > 1 {
				buildNum := 0
				fmt.Sscanf(versionStr[1:], "%d", &buildNum)
				if buildNum > latestBuildNum {
					latestBuildNum = buildNum
					latestVersion = versionStr
				}
			}
		}
	}

	if latestVersion == "" {
		return "", "", fmt.Errorf("Keine gültige Version auf SourceForge gefunden")
	}

	log.Printf("[Setup] SourceForge neueste Version: %s", latestVersion)

	// Bestimme Asset-Namen basierend auf OS und GPU
	var assetName string
	switch runtime.GOOS {
	case "linux":
		switch gpuType {
		case GPUTypeCUDA:
			// Linux-CUDA ist auf SourceForge NICHT verfügbar!
			return "", "", fmt.Errorf("Linux-CUDA ist auf SourceForge nicht verfügbar. Bitte GitHub verwenden oder Vulkan/CPU wählen")
		case GPUTypeVulkan:
			assetName = fmt.Sprintf("llama-%s-bin-ubuntu-vulkan-x64.tar.gz", latestVersion)
		default:
			assetName = fmt.Sprintf("llama-%s-bin-ubuntu-x64.tar.gz", latestVersion)
		}
	case "darwin":
		if runtime.GOARCH == "arm64" {
			assetName = fmt.Sprintf("llama-%s-bin-macos-arm64.tar.gz", latestVersion)
		} else {
			assetName = fmt.Sprintf("llama-%s-bin-macos-x64.tar.gz", latestVersion)
		}
	case "windows":
		switch gpuType {
		case GPUTypeCUDA:
			// Windows CUDA 12.4 verfügbar auf SourceForge (neues Format ohne "cu" Prefix)
			assetName = fmt.Sprintf("llama-%s-bin-win-cuda-12.4-x64.zip", latestVersion)
		case GPUTypeVulkan:
			assetName = fmt.Sprintf("llama-%s-bin-win-vulkan-x64.zip", latestVersion)
		default:
			assetName = fmt.Sprintf("llama-%s-bin-win-cpu-x64.zip", latestVersion)
		}
	default:
		return "", "", fmt.Errorf("Nicht unterstütztes Betriebssystem: %s", runtime.GOOS)
	}

	// Konstruiere SourceForge Download-URL
	downloadURL := fmt.Sprintf("https://sourceforge.net/projects/llama-cpp.mirror/files/%s/%s/download",
		latestVersion, assetName)

	log.Printf("[Setup] SourceForge Download-URL: %s", downloadURL)

	return downloadURL, latestVersion, nil
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
	// 3-Tier Strategie: Eigener Mirror → SourceForge → GitHub
	// Der eigene Mirror hat stabile Dateinamen und ist unabhängig von Upstream-Änderungen
	var downloadURL, version string
	var mirrorErr, primaryErr, fallbackErr error
	var primarySource, fallbackSource string

	// Tier 1: Eigener Mirror (wenn aktiviert)
	if MirrorConfig.Enabled {
		log.Printf("[Setup] 🚀 Versuche Fleet-Navigator Mirror...")
		downloadURL, mirrorErr = d.TryMirrorDownload(gpuType)
		if mirrorErr == nil {
			version = "mirror" // Version nicht bekannt, aber stabil
			log.Printf("[Setup] ✅ Mirror wird verwendet: %s", downloadURL)
		} else {
			log.Printf("[Setup] ⚠️ Mirror nicht verfügbar: %v", mirrorErr)
			if !MirrorConfig.FallbackToGitHub {
				return fmt.Errorf("Mirror fehlgeschlagen und Fallback deaktiviert: %w", mirrorErr)
			}
		}
	}

	// Tier 2 & 3: SourceForge / GitHub Fallback
	if downloadURL == "" {
		useSourceForgeFirst := true
		if runtime.GOOS == "linux" && gpuType == GPUTypeCUDA {
			// Linux-CUDA nur auf GitHub verfügbar
			useSourceForgeFirst = false
			log.Printf("[Setup] Linux-CUDA: Verwende GitHub (SourceForge hat keine Linux-CUDA Binaries)")
		}

		if useSourceForgeFirst {
			primarySource = "SourceForge"
			fallbackSource = "GitHub"

			// Versuche zuerst SourceForge (schnellerer Download)
			log.Printf("[Setup] Versuche SourceForge Mirror (schneller)...")
			downloadURL, version, primaryErr = d.GetLatestReleaseURLFromSourceForge(gpuType)
			if primaryErr != nil {
				log.Printf("[Setup] SourceForge fehlgeschlagen: %v - Fallback auf GitHub", primaryErr)
				downloadURL, version, fallbackErr = d.GetLatestReleaseURL(gpuType)
			}
		} else {
			primarySource = "GitHub"
			fallbackSource = "SourceForge"

			// GitHub als primäre Quelle
			downloadURL, version, primaryErr = d.GetLatestReleaseURL(gpuType)
			if primaryErr != nil {
				// Für nicht-CUDA Fälle: SourceForge als Fallback
				if gpuType != GPUTypeCUDA {
					log.Printf("[Setup] GitHub fehlgeschlagen: %v - Fallback auf SourceForge", primaryErr)
					downloadURL, version, fallbackErr = d.GetLatestReleaseURLFromSourceForge(gpuType)
				} else {
					// Linux-CUDA hat keinen Fallback
					fallbackErr = fmt.Errorf("Linux-CUDA ist nur auf GitHub verfügbar")
				}
			}
		}

		// Benutzerfreundliche Fehlermeldung wenn beide Quellen fehlschlagen
		if primaryErr != nil && fallbackErr != nil {
			log.Printf("[Setup] FEHLER: Beide Download-Quellen fehlgeschlagen!")
			log.Printf("[Setup]   %s: %v", primarySource, primaryErr)
			log.Printf("[Setup]   %s: %v", fallbackSource, fallbackErr)

			// Prüfe auf typische Fehlerursachen
			errMsg := primaryErr.Error() + " " + fallbackErr.Error()
			var userHint string

			if strings.Contains(errMsg, "504") || strings.Contains(errMsg, "502") || strings.Contains(errMsg, "503") {
				userHint = t(lang, "download_error_server_overload")
			} else if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "Timeout") {
				userHint = t(lang, "download_error_timeout")
			} else if strings.Contains(errMsg, "no such host") || strings.Contains(errMsg, "lookup") {
				userHint = t(lang, "download_error_no_internet")
			} else {
				userHint = t(lang, "download_error_generic")
			}

			// Sende Fehler-Progress für Frontend
			progressCh <- SetupProgress{
				Step:    "llama-server",
				Message: userHint,
				Percent: 0,
				Error:   userHint,
			}

			return fmt.Errorf("%s\n\nDetails:\n- %s: %v\n- %s: %v",
				userHint, primarySource, primaryErr, fallbackSource, fallbackErr)
		}
	} // Ende: if downloadURL == ""

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

	isRelevant := func(name string) bool {
		baseName := filepath.Base(name)

		// Hauptbinaries
		if baseName == "llama-server" || baseName == "llama-cli" {
			return true
		}

		// Shared Libraries: .so, .so.0, .so.0.0.7508 etc.
		if strings.Contains(baseName, ".so") {
			return true
		}

		// macOS Libraries
		if strings.HasSuffix(baseName, ".dylib") {
			return true
		}

		// Explizite Prefix-Matches für Binaries
		if strings.HasPrefix(baseName, "llama-") {
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

		name := filepath.Base(header.Name)

		// Nur relevante Dateien extrahieren
		if !isRelevant(name) {
			continue
		}

		destPath := filepath.Join(destDir, name)

		switch header.Typeflag {
		case tar.TypeReg:
			// Reguläre Datei
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

		case tar.TypeSymlink:
			// Symlink - wichtig für .so Versionen (libmtmd.so -> libmtmd.so.0)
			// Erst alte Datei/Symlink entfernen falls vorhanden
			os.Remove(destPath)
			if err := os.Symlink(header.Linkname, destPath); err != nil {
				log.Printf("[Setup] Warnung: Symlink %s -> %s fehlgeschlagen: %v", name, header.Linkname, err)
			} else {
				log.Printf("[Setup] Symlink: %s -> %s", name, header.Linkname)
			}
		}
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
	// Result Channel für Pfade nach erfolgreichem Download
	resultCh := make(chan *VisionDownloadResult, 1)

	// Download in Goroutine
	go func() {
		defer close(progressCh)
		defer close(resultCh)

		result, err := h.visionDownloader.DownloadVisionModel(modelID, progressCh)
		if err != nil {
			log.Printf("[Setup] Vision-Download Fehler: %v", err)
			progressCh <- SetupProgress{
				Step:    "vision",
				Message: fmt.Sprintf("Download-Fehler: %v", err),
				Error:   err.Error(),
				Done:    true,
			}
			return
		}

		// Erfolgreich - Pfade zurückgeben
		if result != nil {
			resultCh <- result
		}
	}()

	// Progress an Client senden
	var downloadSuccess bool
	for progress := range progressCh {
		data, _ := json.Marshal(progress)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		if progress.Done && progress.Error == "" {
			downloadSuccess = true
		}
		if progress.Done || progress.Error != "" {
			break
		}
	}

	// Nach erfolgreichem Download: Vision-Settings und Chaining automatisch konfigurieren
	if downloadSuccess {
		if result := <-resultCh; result != nil {
			if h.settingsUpdater != nil {
				// Vision-Settings speichern (Pfade)
				if err := h.settingsUpdater.SaveVisionSettingsSimple(true, result.ModelPath, result.MmprojPath); err != nil {
					log.Printf("[Setup] ⚠️ Vision-Settings speichern fehlgeschlagen: %v", err)
				} else {
					log.Printf("[Setup] ✅ Vision-Settings automatisch konfiguriert: %s", filepath.Base(result.ModelPath))
				}

				// Vision-Chaining aktivieren (Vision → Analyse-Modell)
				visionModelName := "MiniCPM-V-2.6" // Default
				if strings.Contains(strings.ToLower(filepath.Base(result.ModelPath)), "llava") {
					visionModelName = "LLaVA"
				}
				if err := h.settingsUpdater.EnableVisionChaining(visionModelName); err != nil {
					log.Printf("[Setup] ⚠️ Vision-Chaining aktivieren fehlgeschlagen: %v", err)
				} else {
					log.Printf("[Setup] ✅ Vision-Chaining aktiviert mit: %s", visionModelName)
				}
			}
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
			ModelFile:  "MiniCPM-V-2_6-Q4_K_M.gguf",
			MMProjFile: "MiniCPM-V-2_6-mmproj-f16.gguf",
		}
	default:
		return nil
	}
}

// DownloadVisionModel lädt ein Vision-Modell herunter (Modell + mmproj)
// Strategie: Mirror ZUERST (schneller, stabiler) → HuggingFace-Fallback bei Fehler
// Gibt die installierten Pfade zurück für Settings-Konfiguration
func (d *VisionModelDownloader) DownloadVisionModel(modelID string, progressCh chan<- SetupProgress) (*VisionDownloadResult, error) {
	lang := d.getLangOrDefault()
	info := GetVisionModelURLs(modelID)
	if info == nil {
		return nil, fmt.Errorf("%s", tf(lang, "unknown_vision_model", modelID))
	}

	visionDir := filepath.Join(d.modelsDir, "vision")
	if err := os.MkdirAll(visionDir, 0755); err != nil {
		return nil, fmt.Errorf("Verzeichnis erstellen: %w", err)
	}

	// Mirror-URLs vorbereiten (falls aktiviert) - PRIMÄRE Quelle weil schneller!
	var mirrorModelURL, mirrorMMProjURL string
	if MirrorConfig.Enabled {
		mirrorModelURL = MirrorConfig.BaseURL + MirrorConfig.VisionPath + info.ModelFile
		mirrorMMProjURL = MirrorConfig.BaseURL + MirrorConfig.VisionPath + info.MMProjFile
		log.Printf("[Vision] 🚀 Primäre Download URLs (Mirror): %s", mirrorModelURL)
		log.Printf("[Vision] Fallback URLs (HuggingFace): %s", info.ModelURL)
	}

	// 1. Haupt-Modell herunterladen
	progressCh <- SetupProgress{
		Step:    "vision",
		Message: tf(lang, "vision_model", info.ModelFile),
		Percent: 0,
	}

	modelPath := filepath.Join(visionDir, info.ModelFile)
	err := d.downloadWithMirrorFirst(mirrorModelURL, info.ModelURL, modelPath, "vision-model", progressCh, 0, 70)
	if err != nil {
		return nil, fmt.Errorf("Modell-Download: %w", err)
	}

	// 2. mmproj (Vision Encoder) herunterladen
	progressCh <- SetupProgress{
		Step:    "vision",
		Message: tf(lang, "vision_encoder", info.MMProjFile),
		Percent: 70,
	}

	mmprojPath := filepath.Join(visionDir, info.MMProjFile)
	err = d.downloadWithMirrorFirst(mirrorMMProjURL, info.MMProjURL, mmprojPath, "vision-encoder", progressCh, 70, 100)
	if err != nil {
		return nil, fmt.Errorf("mmproj-Download: %w", err)
	}

	progressCh <- SetupProgress{
		Step:    "vision",
		Message: t(lang, "vision_installed"),
		Percent: 100,
		Done:    true,
	}

	log.Printf("[Setup] Vision-Modell installiert: %s", modelID)

	// Pfade für Settings-Konfiguration zurückgeben
	return &VisionDownloadResult{
		ModelPath:  modelPath,
		MmprojPath: mmprojPath,
	}, nil
}

// downloadWithMirrorFirst lädt eine Datei - Mirror ZUERST (schneller, stabiler), HuggingFace als Fallback
func (d *VisionModelDownloader) downloadWithMirrorFirst(mirrorURL, huggingfaceURL, destPath, component string, progressCh chan<- SetupProgress, startPercent, endPercent float64) error {
	// Strategie: Mirror ZUERST (schneller, stabiler), HuggingFace als Fallback
	if mirrorURL != "" && MirrorConfig.Enabled {
		progressCh <- SetupProgress{
			Step:    "vision",
			Message: "Lade von Mirror...",
			Percent: startPercent,
		}
		err := d.downloadFile(mirrorURL, destPath, component, progressCh, startPercent, endPercent)
		if err == nil {
			log.Printf("[Vision] ✅ Download von Mirror erfolgreich: %s", component)
			return nil
		}

		// Mirror fehlgeschlagen - Fallback zu HuggingFace
		log.Printf("[Vision] ⚠️ Mirror fehlgeschlagen (%v), wechsle zu HuggingFace...", err)
		os.Remove(destPath) // Teildatei löschen
		progressCh <- SetupProgress{
			Step:    "vision",
			Message: "Mirror nicht erreichbar, wechsle zu HuggingFace...",
			Percent: startPercent,
		}
		return d.downloadFile(huggingfaceURL, destPath, component, progressCh, startPercent, endPercent)
	}

	// Kein Mirror verfügbar - direkt von HuggingFace
	progressCh <- SetupProgress{
		Step:    "vision",
		Message: "Lade von HuggingFace...",
		Percent: startPercent,
	}
	return d.downloadFile(huggingfaceURL, destPath, component, progressCh, startPercent, endPercent)
}

// downloadFile lädt eine Datei mit Progress herunter
// Unterstützt Multi-Connection für große Dateien (>100MB)
func (d *VisionModelDownloader) downloadFile(url, destPath, component string, progressCh chan<- SetupProgress, startPercent, endPercent float64) error {
	// HEAD Request mit Redirect-Following um Accept-Ranges zu prüfen
	var totalSize int64
	var finalURL = url
	var acceptRanges string

	noRedirectClient := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	currentURL := url
	log.Printf("[Vision] HEAD Request Start: %s", currentURL)
	for i := 0; i < 10; i++ {
		headReq, _ := http.NewRequest("HEAD", currentURL, nil)
		headReq.Header.Set("User-Agent", "Fleet-Navigator/0.7.0")
		headResp, err := noRedirectClient.Do(headReq)
		if err != nil {
			log.Printf("[Vision] HEAD Request fehlgeschlagen: %v", err)
			break
		}
		headResp.Body.Close()

		log.Printf("[Vision] HEAD Response: Status=%d, Content-Length=%s, Accept-Ranges=%s",
			headResp.StatusCode,
			headResp.Header.Get("Content-Length"),
			headResp.Header.Get("Accept-Ranges"))

		if headResp.StatusCode >= 300 && headResp.StatusCode < 400 {
			location := headResp.Header.Get("Location")
			if location == "" {
				log.Printf("[Vision] ⚠️ Redirect ohne Location Header!")
				break
			}
			log.Printf("[Vision] Folge Redirect %d → %s", headResp.StatusCode, location)
			currentURL = location
			finalURL = location
			continue
		}

		if headResp.StatusCode == http.StatusOK {
			totalSize, _ = strconv.ParseInt(headResp.Header.Get("Content-Length"), 10, 64)
			acceptRanges = headResp.Header.Get("Accept-Ranges")
			finalURL = currentURL
			log.Printf("[Vision] ✅ Finale URL: %s", finalURL)
			log.Printf("[Vision] ✅ Größe: %.1f MB, Accept-Ranges: '%s'", float64(totalSize)/(1024*1024), acceptRanges)
		} else {
			log.Printf("[Vision] ⚠️ Unerwarteter Status: %d", headResp.StatusCode)
		}
		break
	}

	log.Printf("[Vision] Download-Entscheidung: totalSize=%d (>100MB=%v), acceptRanges='%s'",
		totalSize, totalSize > 100*1024*1024, acceptRanges)

	// Multi-Connection für Dateien > 100MB
	if totalSize > 100*1024*1024 && acceptRanges == "bytes" {
		log.Printf("[Vision] ⚡ Multi-Connection Download aktiviert für %s (%.1f MB, 8 Verbindungen)",
			component, float64(totalSize)/(1024*1024))
		return d.downloadFileMulti(finalURL, destPath, component, totalSize, 8, progressCh, startPercent, endPercent)
	}

	// Single-Connection Download (Fallback)
	return d.downloadFileSingle(finalURL, destPath, component, progressCh, startPercent, endPercent)
}

// downloadFileSingle - Standard Single-Connection Download
func (d *VisionModelDownloader) downloadFileSingle(url, destPath, component string, progressCh chan<- SetupProgress, startPercent, endPercent float64) error {
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
	speedChecked := false

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
			downloaded += int64(n)

			elapsed := time.Since(startTime).Seconds()
			speedMBps := float64(downloaded) / elapsed / 1024 / 1024

			// Speed-Check nach konfigurierten Sekunden (nur einmal)
			if !speedChecked && MirrorConfig.Enabled && elapsed >= float64(MirrorConfig.SpeedCheckAfterSec) {
				speedChecked = true
				if speedMBps < MirrorConfig.MinSpeedMBps {
					log.Printf("[Vision] ⚠️ Download zu langsam: %.2f MB/s (min: %.2f MB/s) nach %.0fs",
						speedMBps, MirrorConfig.MinSpeedMBps, elapsed)
					out.Close()
					return &SlowDownloadError{Speed: speedMBps}
				}
				log.Printf("[Vision] ✓ Geschwindigkeit OK: %.2f MB/s nach %.0fs", speedMBps, elapsed)
			}

			if time.Since(lastUpdate) > 500*time.Millisecond {
				filePercent := float64(downloaded) / float64(totalSize)
				totalPercent := startPercent + (filePercent * percentRange)

				progressCh <- SetupProgress{
					Step:       "vision",
					Message:    fmt.Sprintf("Lade %s... (%.1f MB/s)", component, speedMBps),
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

// downloadFileMulti - Multi-Connection Download für große Dateien (8x schneller)
func (d *VisionModelDownloader) downloadFileMulti(url, destPath, component string, totalSize int64, numConnections int, progressCh chan<- SetupProgress, startPercent, endPercent float64) error {
	tempDir := destPath + ".parts"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("Temp-Verzeichnis erstellen: %w", err)
	}

	chunkSize := totalSize / int64(numConnections)
	var wg sync.WaitGroup
	var downloadErr error
	var errMu sync.Mutex

	downloaded := make([]int64, numConnections)
	var progressMu sync.Mutex
	startTime := time.Now()
	percentRange := endPercent - startPercent

	// Progress-Reporter
	progressDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				progressMu.Lock()
				var total int64
				for _, d := range downloaded {
					total += d
				}
				progressMu.Unlock()

				elapsed := time.Since(startTime).Seconds()
				speedMBps := float64(total) / elapsed / 1024 / 1024
				filePercent := float64(total) / float64(totalSize)
				totalPercent := startPercent + (filePercent * percentRange)

				select {
				case progressCh <- SetupProgress{
					Step:       "vision",
					Message:    fmt.Sprintf("Lade %s (8 Verbindungen)...", component),
					Percent:    totalPercent,
					BytesTotal: totalSize,
					BytesDone:  total,
					SpeedMBps:  speedMBps,
				}:
				default:
				}
			case <-progressDone:
				return
			}
		}
	}()

	// Parallel Downloads
	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(connID int) {
			defer wg.Done()

			start := int64(connID) * chunkSize
			end := start + chunkSize - 1
			if connID == numConnections-1 {
				end = totalSize - 1
			}

			partPath := filepath.Join(tempDir, fmt.Sprintf("part_%d", connID))

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				errMu.Lock()
				if downloadErr == nil {
					downloadErr = fmt.Errorf("Request (Conn %d): %w", connID, err)
				}
				errMu.Unlock()
				return
			}
			req.Header.Set("User-Agent", "Fleet-Navigator/0.7.0")
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

			client := &http.Client{Timeout: 0}
			resp, err := client.Do(req)
			if err != nil {
				errMu.Lock()
				if downloadErr == nil {
					downloadErr = fmt.Errorf("Download (Conn %d): %w", connID, err)
				}
				errMu.Unlock()
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
				errMu.Lock()
				if downloadErr == nil {
					downloadErr = fmt.Errorf("HTTP %s (Conn %d)", resp.Status, connID)
				}
				errMu.Unlock()
				return
			}

			partFile, err := os.Create(partPath)
			if err != nil {
				errMu.Lock()
				if downloadErr == nil {
					downloadErr = fmt.Errorf("Part-Datei (Conn %d): %w", connID, err)
				}
				errMu.Unlock()
				return
			}
			defer partFile.Close()

			buf := make([]byte, 64*1024)
			for {
				n, readErr := resp.Body.Read(buf)
				if n > 0 {
					_, writeErr := partFile.Write(buf[:n])
					if writeErr != nil {
						errMu.Lock()
						if downloadErr == nil {
							downloadErr = fmt.Errorf("Schreiben (Conn %d): %w", connID, writeErr)
						}
						errMu.Unlock()
						return
					}
					progressMu.Lock()
					downloaded[connID] += int64(n)
					progressMu.Unlock()
				}
				if readErr == io.EOF {
					break
				}
				if readErr != nil {
					errMu.Lock()
					if downloadErr == nil {
						downloadErr = fmt.Errorf("Lesen (Conn %d): %w", connID, readErr)
					}
					errMu.Unlock()
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(progressDone)

	if downloadErr != nil {
		os.RemoveAll(tempDir)
		return downloadErr
	}

	// Parts zusammenfügen
	log.Printf("[Vision] Füge %d Parts zusammen: %s", numConnections, component)
	outFile, err := os.Create(destPath)
	if err != nil {
		os.RemoveAll(tempDir)
		return fmt.Errorf("Zieldatei erstellen: %w", err)
	}

	for i := 0; i < numConnections; i++ {
		partPath := filepath.Join(tempDir, fmt.Sprintf("part_%d", i))
		partFile, err := os.Open(partPath)
		if err != nil {
			outFile.Close()
			os.Remove(destPath)
			os.RemoveAll(tempDir)
			return fmt.Errorf("Part öffnen: %w", err)
		}
		_, err = io.Copy(outFile, partFile)
		partFile.Close()
		if err != nil {
			outFile.Close()
			os.Remove(destPath)
			os.RemoveAll(tempDir)
			return fmt.Errorf("Part kopieren: %w", err)
		}
	}
	outFile.Close()
	os.RemoveAll(tempDir)

	elapsed := time.Since(startTime).Seconds()
	speed := float64(totalSize) / 1024 / 1024 / elapsed
	log.Printf("[Vision] ✅ Multi-Download abgeschlossen: %s (%.1f MB in %.1fs = %.1f MB/s)",
		component, float64(totalSize)/(1024*1024), elapsed, speed)

	return nil
}

// HandleSetupSummary GET /api/setup/summary
// Gibt eine Zusammenfassung aller installierten Komponenten zurück
func (h *APIHandler) HandleSetupSummary(w http.ResponseWriter, r *http.Request) {
	state := h.service.GetState()

	summary := SetupSummary{
		DisclaimerText: "Die Experten im Fleet Navigator sind virtuelle Assistenzrollen. Sie unterstützen bei Analyse und Vorbereitung, ersetzen jedoch keine individuelle Fach- oder Rechtsberatung.",
	}

	// LLM-Modell Status
	if state.SelectedModel != "" {
		modelPath := filepath.Join(h.modelsDir, "library", state.SelectedModel)
		installed := false
		if _, err := os.Stat(modelPath); err == nil {
			installed = true
		}
		summary.LLMModel = &ComponentStatus{
			Name:        deriveDisplayName(state.SelectedModel),
			Installed:   installed,
			Description: "KI-Sprachmodell für Textverarbeitung",
		}
	}

	// llama-server Status
	binDir := filepath.Join(h.service.dataDir, "bin")
	var binaryName string
	if runtime.GOOS == "windows" {
		binaryName = "llama-server.exe"
	} else {
		binaryName = "llama-server"
	}
	serverPath := filepath.Join(binDir, binaryName)
	llamaInstalled := false
	if _, err := os.Stat(serverPath); err == nil {
		llamaInstalled = true
	}
	summary.LlamaServer = &ComponentStatus{
		Name:        "llama-server",
		Installed:   llamaInstalled,
		Description: "KI-Engine für lokale Modellausführung",
	}

	// Vision-Modell Status
	if state.VisionEnabled && state.VisionModel != "" {
		visionDir := filepath.Join(h.service.dataDir, "models", "vision")
		visionInstalled := false
		// Prüfe ob das Vision-Modell existiert
		if entries, err := os.ReadDir(visionDir); err == nil {
			for _, entry := range entries {
				if strings.HasSuffix(entry.Name(), ".gguf") {
					visionInstalled = true
					break
				}
			}
		}
		summary.VisionModel = &ComponentStatus{
			Name:        state.VisionModel,
			Installed:   visionInstalled,
			Description: "Bildanalyse und Dokumentenerkennung",
		}
	}

	// Whisper STT Status
	if state.VoiceEnabled && state.WhisperModel != "" {
		whisperDir := filepath.Join(h.service.dataDir, "voice", "whisper")
		whisperInstalled := false
		// Prüfe ob Whisper-Binary und Modell existieren
		whisperBin := filepath.Join(whisperDir, "whisper-cli")
		if runtime.GOOS == "windows" {
			whisperBin = filepath.Join(whisperDir, "whisper-cli.exe")
		}
		if _, err := os.Stat(whisperBin); err == nil {
			whisperInstalled = true
		}
		summary.WhisperSTT = &ComponentStatus{
			Name:        "Whisper " + strings.ToUpper(state.WhisperModel),
			Installed:   whisperInstalled,
			Description: "Spracherkennung (Speech-to-Text)",
		}
	}

	// Piper TTS Status
	if state.VoiceEnabled && state.PiperVoice != "" {
		piperDir := filepath.Join(h.service.dataDir, "voice", "piper")
		piperInstalled := false
		// Prüfe ob Piper-Binary existiert
		piperBin := filepath.Join(piperDir, "piper")
		if runtime.GOOS == "windows" {
			piperBin = filepath.Join(piperDir, "piper.exe")
		}
		if _, err := os.Stat(piperBin); err == nil {
			piperInstalled = true
		}

		// Voice-Name ableiten
		voiceName := state.PiperVoice
		if strings.Contains(voiceName, "eva") {
			voiceName = "Eva (weiblich)"
		} else if strings.Contains(voiceName, "thorsten") {
			voiceName = "Thorsten (männlich)"
		}

		summary.PiperTTS = &ComponentStatus{
			Name:        "Piper TTS - " + voiceName,
			Installed:   piperInstalled,
			Description: "Sprachausgabe (Text-to-Speech)",
		}
	}

	// Experten Status (immer vorhanden)
	summary.Experts = &ComponentStatus{
		Name:        "6 KI-Experten",
		Installed:   true,
		Description: "Ewa, Roland, Ayşe, Luca, Franziska, Dr. Sol",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// HandleAcceptDisclaimer POST /api/setup/accept-disclaimer
// Speichert die Bestätigung des Disclaimers
func (h *APIHandler) HandleAcceptDisclaimer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Accepted bool `json:"accepted"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !req.Accepted {
		http.Error(w, "Disclaimer muss akzeptiert werden", http.StatusBadRequest)
		return
	}

	// In Settings-DB speichern
	if h.settingsUpdater != nil {
		if err := h.settingsUpdater.SaveDisclaimerAccepted(true); err != nil {
			log.Printf("[Setup] Warnung: Disclaimer-Akzeptanz konnte nicht gespeichert werden: %v", err)
		} else {
			log.Printf("[Setup] ✅ Disclaimer akzeptiert und gespeichert")
		}
	}

	// State aktualisieren
	h.service.mu.Lock()
	h.service.state.DisclaimerAccepted = true
	h.service.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Hinweis bestätigt. Sie können Fleet Navigator jetzt nutzen.",
	})
}

// ============================================================================
// VOICE DOWNLOADER - Whisper & Piper
// ============================================================================

// VoiceModelDownloader implementiert VoiceDownloader Interface
type VoiceModelDownloader struct {
	dataDir string
	client  *http.Client
}

// NewVoiceModelDownloader erstellt einen neuen Voice-Downloader
func NewVoiceModelDownloader(dataDir string) *VoiceModelDownloader {
	return &VoiceModelDownloader{
		dataDir: dataDir,
		client: &http.Client{
			Timeout: 30 * time.Minute,
		},
	}
}

// DownloadWhisper lädt Whisper (Binary + Modell) herunter
// Strategie: Mirror → GitHub Fallback mit Retry
func (d *VoiceModelDownloader) DownloadWhisper(model string, progressCh chan<- SetupProgress) error {
	whisperDir := filepath.Join(d.dataDir, "voice", "whisper")
	if err := os.MkdirAll(whisperDir, 0755); err != nil {
		return fmt.Errorf("Verzeichnis erstellen: %w", err)
	}

	// 1. Whisper Binary herunterladen
	progressCh <- SetupProgress{
		Step:    "whisper",
		Message: "Lade Whisper Binary...",
		Percent: 0,
	}

	// GitHub Release URL für whisper.cpp (Repo ist jetzt ggml-org/whisper.cpp)
	const whisperRelease = "https://github.com/ggml-org/whisper.cpp/releases/download/v1.8.2"

	// Binary-Name der im ZIP liegt
	var binaryName string
	var zipBinaryName string // Name der Binary im ZIP
	var zipURL string
	var mirrorURL string

	switch runtime.GOOS {
	case "windows":
		binaryName = "whisper-cli.exe"
		zipBinaryName = "whisper-cli.exe" // oder main.exe je nach Build
		zipURL = whisperRelease + "/whisper-bin-x64.zip"
		mirrorURL = MirrorConfig.BaseURL + MirrorConfig.WhisperPath + "whisper-cli-windows-amd64.exe"
	case "darwin":
		binaryName = "whisper-cli"
		zipBinaryName = "whisper-cli"
		if runtime.GOARCH == "arm64" {
			// macOS ARM hat kein offizielles Release - nutze Mirror oder baue selbst
			zipURL = "" // Kein offizielles Release
			mirrorURL = MirrorConfig.BaseURL + MirrorConfig.WhisperPath + "whisper-cli-darwin-arm64"
		} else {
			zipURL = whisperRelease + "/whisper-bin-x64.zip"
			mirrorURL = MirrorConfig.BaseURL + MirrorConfig.WhisperPath + "whisper-cli-darwin-amd64"
		}
	default: // linux
		binaryName = "whisper-cli"
		zipBinaryName = "whisper-cli"
		// Linux hat kein offizielles Release - nutze Mirror
		zipURL = ""
		mirrorURL = MirrorConfig.BaseURL + MirrorConfig.WhisperPath + "whisper-cli-linux-amd64"
	}

	binaryPath := filepath.Join(whisperDir, binaryName)
	var lastErr error

	// Strategie: GitHub ZIP zuerst (für Windows), dann Mirror
	downloadSources := []struct {
		url   string
		isZip bool
	}{}

	if zipURL != "" {
		downloadSources = append(downloadSources, struct {
			url   string
			isZip bool
		}{zipURL, true})
	}
	if mirrorURL != "" {
		downloadSources = append(downloadSources, struct {
			url   string
			isZip bool
		}{mirrorURL, false})
	}

	for i, source := range downloadSources {
		log.Printf("[Whisper] Versuche Source %d/%d: %s (ZIP=%v)", i+1, len(downloadSources), source.url, source.isZip)

		for retry := 0; retry < 3; retry++ {
			if retry > 0 {
				log.Printf("[Whisper] Retry %d/3...", retry+1)
				time.Sleep(time.Duration(retry) * time.Second)
			}

			var err error
			if source.isZip {
				// ZIP herunterladen und entpacken
				zipPath := filepath.Join(whisperDir, "whisper-temp.zip")
				err = d.downloadFile(source.url, zipPath, progressCh, 0, 25)
				if err == nil {
					// ZIP entpacken
					progressCh <- SetupProgress{
						Step:    "whisper",
						Message: "Entpacke Whisper Binary...",
						Percent: 25,
					}
					err = d.extractBinaryFromZip(zipPath, whisperDir, zipBinaryName, binaryName)
					os.Remove(zipPath) // ZIP löschen
				}
			} else {
				// Direkte Binary herunterladen
				err = d.downloadFile(source.url, binaryPath, progressCh, 0, 30)
			}

			if err == nil {
				log.Printf("[Whisper] ✅ Binary installiert von: %s", source.url)
				goto binaryDone
			}
			lastErr = err
			log.Printf("[Whisper] ⚠️ Fehlgeschlagen: %v", err)
		}
	}
	return fmt.Errorf("Binary-Download fehlgeschlagen: %w", lastErr)

binaryDone:
	// Ausführbar machen (Unix)
	if runtime.GOOS != "windows" {
		os.Chmod(binaryPath, 0755)
	}

	// 2. Whisper Modell herunterladen
	progressCh <- SetupProgress{
		Step:    "whisper",
		Message: fmt.Sprintf("Lade Whisper Modell (%s)...", model),
		Percent: 30,
	}

	// Modell-Dateiname (z.B. ggml-base.bin, ggml-small.bin)
	modelFile := fmt.Sprintf("ggml-%s.bin", model)
	modelURLs := []string{
		MirrorConfig.BaseURL + MirrorConfig.WhisperPath + modelFile,                            // Mirror zuerst (keine /models/ Subdirectory)
		fmt.Sprintf("https://huggingface.co/ggerganov/whisper.cpp/resolve/main/%s", modelFile), // HuggingFace Fallback
	}
	modelPath := filepath.Join(whisperDir, modelFile)

	// Versuche alle URLs mit Retry
	for i, url := range modelURLs {
		log.Printf("[Whisper] Versuche Modell URL %d/%d: %s", i+1, len(modelURLs), url)

		for retry := 0; retry < 3; retry++ {
			if retry > 0 {
				log.Printf("[Whisper] Retry %d/3...", retry+1)
				time.Sleep(time.Duration(retry) * time.Second)
			}

			err := d.downloadFile(url, modelPath, progressCh, 30, 100)
			if err == nil {
				log.Printf("[Whisper] ✅ Modell heruntergeladen von: %s", url)
				goto modelDone
			}
			lastErr = err
			log.Printf("[Whisper] ⚠️ Fehlgeschlagen: %v", err)
		}
	}
	return fmt.Errorf("Modell-Download fehlgeschlagen (alle Quellen versucht): %w", lastErr)

modelDone:
	progressCh <- SetupProgress{
		Step:    "whisper",
		Message: "Whisper erfolgreich installiert!",
		Percent: 100,
		Done:    true,
	}

	log.Printf("[Whisper] ✅ Installiert: %s", whisperDir)
	return nil
}

// DownloadPiper lädt Piper (Binary + Stimme) herunter
// Strategie: GitHub → Mirror Fallback mit Retry
func (d *VoiceModelDownloader) DownloadPiper(voice string, progressCh chan<- SetupProgress) error {
	piperDir := filepath.Join(d.dataDir, "voice", "piper")
	if err := os.MkdirAll(piperDir, 0755); err != nil {
		return fmt.Errorf("Verzeichnis erstellen: %w", err)
	}

	// 1. Piper Binary herunterladen
	progressCh <- SetupProgress{
		Step:    "piper",
		Message: "Lade Piper Binary...",
		Percent: 0,
	}

	// GitHub Release URL für Piper
	const piperRelease = "https://github.com/rhasspy/piper/releases/download/2023.11.14-2"

	var binaryName string
	var archiveURL string
	var archiveType string // "zip" oder "tar.gz"
	var mirrorURL string

	switch runtime.GOOS {
	case "windows":
		binaryName = "piper.exe"
		archiveURL = piperRelease + "/piper_windows_amd64.zip"
		archiveType = "zip"
		mirrorURL = MirrorConfig.BaseURL + MirrorConfig.PiperPath + "piper-windows-amd64.exe"
	case "darwin":
		binaryName = "piper"
		if runtime.GOARCH == "arm64" {
			archiveURL = piperRelease + "/piper_macos_aarch64.tar.gz"
		} else {
			archiveURL = piperRelease + "/piper_macos_x64.tar.gz"
		}
		archiveType = "tar.gz"
		mirrorURL = MirrorConfig.BaseURL + MirrorConfig.PiperPath + "piper-darwin-arm64"
	default: // linux
		binaryName = "piper"
		archiveURL = piperRelease + "/piper_linux_x86_64.tar.gz"
		archiveType = "tar.gz"
		mirrorURL = MirrorConfig.BaseURL + MirrorConfig.PiperPath + "piper-linux-amd64"
	}

	binaryPath := filepath.Join(piperDir, binaryName)
	var lastErr error

	// Strategie: GitHub Archive zuerst, dann Mirror
	downloadSources := []struct {
		url         string
		archiveType string // "zip", "tar.gz", oder "" für direkt
	}{
		{archiveURL, archiveType},
		{mirrorURL, ""},
	}

	for i, source := range downloadSources {
		if source.url == "" {
			continue
		}
		log.Printf("[Piper] Versuche Source %d/%d: %s (Type=%s)", i+1, len(downloadSources), source.url, source.archiveType)

		for retry := 0; retry < 3; retry++ {
			if retry > 0 {
				log.Printf("[Piper] Retry %d/3...", retry+1)
				time.Sleep(time.Duration(retry) * time.Second)
			}

			var err error
			if source.archiveType == "zip" {
				// ZIP herunterladen und entpacken
				archivePath := filepath.Join(piperDir, "piper-temp.zip")
				err = d.downloadFile(source.url, archivePath, progressCh, 0, 25)
				if err == nil {
					progressCh <- SetupProgress{
						Step:    "piper",
						Message: "Entpacke Piper...",
						Percent: 25,
					}
					err = d.extractBinaryFromZip(archivePath, piperDir, "piper.exe", binaryName)
					os.Remove(archivePath)
				}
			} else if source.archiveType == "tar.gz" {
				// TAR.GZ herunterladen und entpacken
				archivePath := filepath.Join(piperDir, "piper-temp.tar.gz")
				err = d.downloadFile(source.url, archivePath, progressCh, 0, 25)
				if err == nil {
					progressCh <- SetupProgress{
						Step:    "piper",
						Message: "Entpacke Piper...",
						Percent: 25,
					}
					err = d.extractBinaryFromTarGz(archivePath, piperDir, "piper", binaryName)
					os.Remove(archivePath)
				}
			} else {
				// Direkte Binary
				err = d.downloadFile(source.url, binaryPath, progressCh, 0, 30)
			}

			if err == nil {
				log.Printf("[Piper] ✅ Binary installiert von: %s", source.url)
				goto binaryDone
			}
			lastErr = err
			log.Printf("[Piper] ⚠️ Fehlgeschlagen: %v", err)
		}
	}
	return fmt.Errorf("Binary-Download fehlgeschlagen: %w", lastErr)

binaryDone:
	// Ausführbar machen (Unix)
	if runtime.GOOS != "windows" {
		os.Chmod(binaryPath, 0755)
	}

	// 2. Piper Stimme herunterladen (.onnx + .json)
	progressCh <- SetupProgress{
		Step:    "piper",
		Message: fmt.Sprintf("Lade Stimme (%s)...", voice),
		Percent: 30,
	}

	// Stimmen-Dateien (z.B. de_DE-eva_k-medium.onnx + .json)
	voiceOnnx := voice + ".onnx"
	voiceJson := voice + ".onnx.json"
	voicesDir := filepath.Join(piperDir, "voices")
	os.MkdirAll(voicesDir, 0755)

	// Voice URLs - Mirror + HuggingFace Fallback
	// HuggingFace Format: https://huggingface.co/rhasspy/piper-voices/resolve/main/de/de_DE/eva_k/medium/de_DE-eva_k-medium.onnx
	voiceParts := strings.Split(voice, "-") // z.B. "de_DE-eva_k-medium" -> ["de_DE", "eva_k", "medium"]
	var hfVoicePath string
	if len(voiceParts) >= 3 {
		lang := strings.Split(voiceParts[0], "_")[0] // "de_DE" -> "de"
		locale := voiceParts[0]                       // "de_DE"
		speaker := voiceParts[1]                      // "eva_k"
		quality := voiceParts[2]                      // "medium"
		hfVoicePath = fmt.Sprintf("%s/%s/%s/%s/%s", lang, locale, speaker, quality, voice)
	}

	// Mirror zuerst, dann HuggingFace Fallback
	var onnxURLs []string
	onnxURLs = append(onnxURLs,
		MirrorConfig.BaseURL+MirrorConfig.PiperPath+"voices/"+voiceOnnx) // Mirror zuerst
	if hfVoicePath != "" {
		onnxURLs = append(onnxURLs,
			fmt.Sprintf("https://huggingface.co/rhasspy/piper-voices/resolve/main/%s.onnx", hfVoicePath)) // HuggingFace Fallback
	}

	onnxPath := filepath.Join(voicesDir, voiceOnnx)

	for i, url := range onnxURLs {
		log.Printf("[Piper] Versuche Voice URL %d/%d: %s", i+1, len(onnxURLs), url)

		for retry := 0; retry < 3; retry++ {
			if retry > 0 {
				log.Printf("[Piper] Retry %d/3...", retry+1)
				time.Sleep(time.Duration(retry) * time.Second)
			}

			err := d.downloadFile(url, onnxPath, progressCh, 30, 90)
			if err == nil {
				log.Printf("[Piper] ✅ Voice heruntergeladen von: %s", url)
				goto voiceDone
			}
			lastErr = err
			log.Printf("[Piper] ⚠️ Fehlgeschlagen: %v", err)
		}
	}
	return fmt.Errorf("Voice-Download fehlgeschlagen (alle Quellen versucht): %w", lastErr)

voiceDone:
	// JSON-Config (optional) - Mirror zuerst, HuggingFace Fallback
	var jsonURLs []string
	jsonURLs = append(jsonURLs,
		MirrorConfig.BaseURL+MirrorConfig.PiperPath+"voices/"+voiceJson) // Mirror zuerst
	if hfVoicePath != "" {
		jsonURLs = append(jsonURLs,
			fmt.Sprintf("https://huggingface.co/rhasspy/piper-voices/resolve/main/%s.onnx.json", hfVoicePath)) // HuggingFace Fallback
	}

	jsonPath := filepath.Join(voicesDir, voiceJson)
	for _, url := range jsonURLs {
		if err := d.downloadFile(url, jsonPath, progressCh, 90, 100); err == nil {
			log.Printf("[Piper] ✅ Voice-Config heruntergeladen")
			break
		}
	}

	progressCh <- SetupProgress{
		Step:    "piper",
		Message: "Piper erfolgreich installiert!",
		Percent: 100,
		Done:    true,
	}

	log.Printf("[Piper] ✅ Installiert: %s", piperDir)
	return nil
}

// DownloadTesseract lädt Tesseract OCR (Binary + Sprachpakete) herunter
// Tesseract wird als ZIP vom Mirror geladen und entpackt
func (d *VoiceModelDownloader) DownloadTesseract(progressCh chan<- SetupProgress) error {
	tesseractDir := filepath.Join(d.dataDir, "tesseract")
	if err := os.MkdirAll(tesseractDir, 0755); err != nil {
		return fmt.Errorf("Verzeichnis erstellen: %w", err)
	}

	// Prüfen ob bereits installiert
	var binaryName string
	switch runtime.GOOS {
	case "windows":
		binaryName = "tesseract.exe"
	default:
		binaryName = "tesseract"
	}

	binaryPath := filepath.Join(tesseractDir, binaryName)
	if _, err := os.Stat(binaryPath); err == nil {
		log.Printf("[Tesseract] Bereits installiert: %s", binaryPath)
		progressCh <- SetupProgress{
			Step:    "tesseract",
			Message: "Tesseract OCR bereits installiert!",
			Percent: 100,
			Done:    true,
		}
		return nil
	}

	progressCh <- SetupProgress{
		Step:    "tesseract",
		Message: "Lade Tesseract OCR...",
		Percent: 0,
	}

	// Download-URL basierend auf Betriebssystem
	var zipURL string
	var zipName string

	switch runtime.GOOS {
	case "windows":
		// Portable Tesseract für Windows vom Mirror
		zipURL = MirrorConfig.BaseURL + MirrorConfig.TesseractPath + "tesseract-ocr-windows-x64.zip"
		zipName = "tesseract-ocr-windows-x64.zip"
	case "darwin":
		// macOS - falls später unterstützt
		zipURL = MirrorConfig.BaseURL + MirrorConfig.TesseractPath + "tesseract-ocr-macos-arm64.tar.gz"
		zipName = "tesseract-ocr-macos.tar.gz"
	default:
		// Linux - apt/dnf empfohlen
		zipURL = MirrorConfig.BaseURL + MirrorConfig.TesseractPath + "tesseract-ocr-linux-x64.tar.gz"
		zipName = "tesseract-ocr-linux.tar.gz"
	}

	zipPath := filepath.Join(tesseractDir, zipName)
	var lastErr error

	// Download mit Retry
	for retry := 0; retry < 3; retry++ {
		if retry > 0 {
			log.Printf("[Tesseract] Retry %d/3...", retry+1)
			time.Sleep(time.Duration(retry) * time.Second)
		}

		progressCh <- SetupProgress{
			Step:    "tesseract",
			Message: fmt.Sprintf("Lade Tesseract OCR... (Versuch %d/3)", retry+1),
			Percent: 5,
		}

		err := d.downloadFile(zipURL, zipPath, progressCh, 5, 70)
		if err == nil {
			log.Printf("[Tesseract] ✅ Download erfolgreich: %s", zipURL)
			goto downloadDone
		}
		lastErr = err
		log.Printf("[Tesseract] ⚠️ Download fehlgeschlagen: %v", err)
	}
	return fmt.Errorf("Tesseract-Download fehlgeschlagen: %w", lastErr)

downloadDone:
	// Entpacken
	progressCh <- SetupProgress{
		Step:    "tesseract",
		Message: "Entpacke Tesseract OCR...",
		Percent: 75,
	}

	if runtime.GOOS == "windows" {
		// ZIP entpacken
		if err := d.extractZipToDir(zipPath, tesseractDir); err != nil {
			os.Remove(zipPath)
			return fmt.Errorf("ZIP entpacken: %w", err)
		}
	} else {
		// TAR.GZ entpacken
		if err := d.extractTarGzToDir(zipPath, tesseractDir); err != nil {
			os.Remove(zipPath)
			return fmt.Errorf("TAR.GZ entpacken: %w", err)
		}
	}

	// ZIP/TAR löschen
	os.Remove(zipPath)

	// Prüfen ob Binary existiert
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		// Vielleicht ist es in einem Unterordner?
		// Suche nach tesseract.exe
		var foundPath string
		filepath.Walk(tesseractDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.Name() == binaryName {
				foundPath = path
				return filepath.SkipAll
			}
			return nil
		})

		if foundPath != "" {
			log.Printf("[Tesseract] Binary gefunden in Unterordner: %s", foundPath)
			// Binary ist in einem Unterordner - das ist okay
		} else {
			return fmt.Errorf("tesseract binary nicht gefunden nach Entpacken")
		}
	}

	// Unter Unix ausführbar machen
	if runtime.GOOS != "windows" {
		os.Chmod(binaryPath, 0755)
	}

	progressCh <- SetupProgress{
		Step:    "tesseract",
		Message: "Tesseract OCR erfolgreich installiert!",
		Percent: 100,
		Done:    true,
	}

	log.Printf("[Tesseract] ✅ Installiert: %s", tesseractDir)
	return nil
}

// extractZipToDir entpackt ein ZIP-Archiv in ein Verzeichnis
func (d *VoiceModelDownloader) extractZipToDir(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)

		// Sicherheitscheck: Pfad darf nicht außerhalb von destDir sein
		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("ungültiger Pfad in ZIP: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Elternverzeichnis erstellen
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// extractTarGzToDir entpackt ein TAR.GZ-Archiv in ein Verzeichnis
func (d *VoiceModelDownloader) extractTarGzToDir(tarGzPath, destDir string) error {
	file, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		fpath := filepath.Join(destDir, header.Name)

		// Sicherheitscheck
		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("ungültiger Pfad in TAR: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(fpath, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
				return err
			}

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	return nil
}

// downloadFile lädt eine Datei herunter mit Progress
// Behandelt HTTP-Fehler sauber und gibt aussagekräftige Fehlermeldungen
func (d *VoiceModelDownloader) downloadFile(url, destPath string, progressCh chan<- SetupProgress, startPercent, endPercent float64) error {
	resp, err := d.client.Get(url)
	if err != nil {
		return fmt.Errorf("Verbindungsfehler: %w", err)
	}
	defer resp.Body.Close()

	// HTTP-Fehler sauber abfangen
	switch resp.StatusCode {
	case http.StatusOK:
		// Alles gut, weiter
	case http.StatusNotFound:
		return fmt.Errorf("404 - Datei nicht gefunden auf Server")
	case http.StatusForbidden:
		return fmt.Errorf("403 - Zugriff verweigert")
	case http.StatusServiceUnavailable:
		return fmt.Errorf("503 - Server nicht verfügbar")
	case http.StatusTooManyRequests:
		return fmt.Errorf("429 - Rate Limit erreicht")
	default:
		if resp.StatusCode >= 400 {
			return fmt.Errorf("HTTP %d - %s", resp.StatusCode, resp.Status)
		}
	}

	// Temporäre Datei erstellen (nicht direkt Ziel, falls Download abbricht)
	tempPath := destPath + ".tmp"
	out, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("Temp-Datei erstellen: %w", err)
	}
	defer func() {
		out.Close()
		// Bei Fehler temp Datei löschen
		if err != nil {
			os.Remove(tempPath)
		}
	}()

	totalSize := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 32*1024)

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				err = fmt.Errorf("Schreibfehler: %w", writeErr)
				return err
			}
			downloaded += int64(n)

			if totalSize > 0 {
				progress := startPercent + (endPercent-startPercent)*float64(downloaded)/float64(totalSize)
				progressCh <- SetupProgress{
					Step:    "voice",
					Percent: progress,
				}
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			err = fmt.Errorf("Lesefehler: %w", readErr)
			return err
		}
	}

	out.Close()

	// Erfolgreich - temp zu Ziel umbenennen
	if renameErr := os.Rename(tempPath, destPath); renameErr != nil {
		return fmt.Errorf("Umbenennen fehlgeschlagen: %w", renameErr)
	}

	return nil
}

// extractBinaryFromZip extrahiert eine bestimmte Binary aus einem ZIP-Archiv
// zipBinaryName ist der Name der Datei im ZIP, binaryName ist der Zielname
func (d *VoiceModelDownloader) extractBinaryFromZip(zipPath, destDir, zipBinaryName, binaryName string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("ZIP öffnen: %w", err)
	}
	defer reader.Close()

	// Suche nach der Binary im ZIP (kann in Unterordner sein)
	var targetFile *zip.File
	for _, f := range reader.File {
		// Prüfe ob Dateiname endet mit gesuchtem Namen
		baseName := filepath.Base(f.Name)
		if baseName == zipBinaryName || baseName == "main.exe" || baseName == "main" {
			targetFile = f
			log.Printf("[ZIP] Gefunden: %s", f.Name)
			break
		}
	}

	if targetFile == nil {
		// Liste alle Dateien im ZIP für Debug
		log.Printf("[ZIP] Binary '%s' nicht gefunden. Inhalt:", zipBinaryName)
		for _, f := range reader.File {
			log.Printf("[ZIP]   - %s", f.Name)
		}
		return fmt.Errorf("Binary '%s' nicht im ZIP gefunden", zipBinaryName)
	}

	// Datei extrahieren
	srcFile, err := targetFile.Open()
	if err != nil {
		return fmt.Errorf("Datei im ZIP öffnen: %w", err)
	}
	defer srcFile.Close()

	destPath := filepath.Join(destDir, binaryName)
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("Zieldatei erstellen: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("Datei extrahieren: %w", err)
	}

	log.Printf("[ZIP] ✅ Extrahiert: %s -> %s", targetFile.Name, destPath)
	return nil
}

// extractBinaryFromTarGz extrahiert eine bestimmte Binary aus einem TAR.GZ-Archiv
func (d *VoiceModelDownloader) extractBinaryFromTarGz(archivePath, destDir, tarBinaryName, binaryName string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("Archiv öffnen: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("GZIP lesen: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("TAR lesen: %w", err)
		}

		// Prüfe ob Dateiname passt
		baseName := filepath.Base(header.Name)
		if baseName == tarBinaryName {
			destPath := filepath.Join(destDir, binaryName)
			destFile, err := os.Create(destPath)
			if err != nil {
				return fmt.Errorf("Zieldatei erstellen: %w", err)
			}

			_, err = io.Copy(destFile, tarReader)
			destFile.Close()
			if err != nil {
				return fmt.Errorf("Datei extrahieren: %w", err)
			}

			log.Printf("[TAR.GZ] ✅ Extrahiert: %s -> %s", header.Name, destPath)
			return nil
		}
	}

	return fmt.Errorf("Binary '%s' nicht im TAR.GZ gefunden", tarBinaryName)
}
