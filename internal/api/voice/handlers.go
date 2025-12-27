// Package voicehandlers enthält die HTTP-Handler für Voice-Endpoints (STT/TTS)
package voicehandlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"fleet-navigator/internal/api/common"
	"fleet-navigator/internal/security"
	"fleet-navigator/internal/voice"
)

// VoiceService definiert die Voice-Service-Methoden
type VoiceService interface {
	GetStatus() voice.Status
	TranscribeAudio(audioData []byte, format string) (*voice.TranscriptionResult, error)
	SynthesizeSpeech(text, voiceName string) (*voice.SpeechResult, error)
	GetInstalledWhisperModels() []string
	GetInstalledPiperVoices() []string
	SetWhisperModel(model string) error
	SetPiperVoice(voiceName string) error
	EnsureModelsDownloaded(progressChan chan<- voice.DownloadProgress) error
	EnsureWhisperDownloaded(progressChan chan<- voice.DownloadProgress) error
	EnsurePiperDownloaded(progressChan chan<- voice.DownloadProgress) error
}

// SettingsProvider speichert Voice-Einstellungen
type SettingsProvider interface {
	SaveWhisperModel(model string) error
	GetVoiceSettings() VoiceSettings
	SaveVoiceSettings(settings VoiceSettings) error
}

// VoiceSettings enthält Voice-Konfiguration
type VoiceSettings struct {
	PiperVoice string `json:"piperVoice"`
}

// Handlers enthält die HTTP-Handler für Voice-Endpoints
type Handlers struct {
	voiceService VoiceService
	settings     SettingsProvider
}

// NewHandlers erstellt neue Voice-Handler
func NewHandlers(voiceService VoiceService) *Handlers {
	return &Handlers{voiceService: voiceService}
}

// SetSettingsProvider setzt den Settings-Provider (optional)
func (h *Handlers) SetSettingsProvider(settings SettingsProvider) {
	h.settings = settings
}

// RegisterRoutes registriert die Voice-API-Routen
func (h *Handlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/voice/status", h.handleStatus)
	mux.HandleFunc("/api/voice/stt", h.handleSTT)
	mux.HandleFunc("/api/voice/tts", h.handleTTS)
	mux.HandleFunc("/api/voice/download", h.handleDownload)
	mux.HandleFunc("/api/voice/models", h.handleModels)
	mux.HandleFunc("/api/voice/config", h.handleConfig)
}

// handleStatus - GET /api/voice/status
func (h *Handlers) handleStatus(w http.ResponseWriter, r *http.Request) {
	if !common.RequireGET(w, r) {
		return
	}

	status := h.voiceService.GetStatus()
	common.WriteJSON(w, http.StatusOK, status)
}

// handleSTT - POST /api/voice/stt (Speech-to-Text)
func (h *Handlers) handleSTT(w http.ResponseWriter, r *http.Request) {
	if !common.RequirePOST(w, r) {
		return
	}

	if err := r.ParseMultipartForm(50 << 20); err != nil {
		common.WriteBadRequest(w, fmt.Sprintf("Form parsen: %v", err))
		return
	}

	file, header, err := r.FormFile("audio")
	if err != nil {
		common.WriteBadRequest(w, fmt.Sprintf("Audio-Datei fehlt: %v", err))
		return
	}
	defer file.Close()

	audioData, err := io.ReadAll(file)
	if err != nil {
		common.WriteInternalError(w, err, "Audio lesen fehlgeschlagen")
		return
	}

	// Format aus Dateiname extrahieren (mit Validierung)
	format := "webm"
	if ext := filepath.Ext(header.Filename); ext != "" {
		format = security.ValidateAudioFormat(strings.TrimPrefix(ext, "."))
	}

	if f := r.FormValue("format"); f != "" {
		format = security.ValidateAudioFormat(f)
	}

	log.Printf("STT-Anfrage: %d bytes, Format: %s", len(audioData), format)

	result, err := h.voiceService.TranscribeAudio(audioData, format)
	if err != nil {
		log.Printf("STT-Fehler: %v", err)
		common.WriteInternalError(w, err, "Transkription fehlgeschlagen")
		return
	}

	common.WriteJSON(w, http.StatusOK, result)
}

// handleTTS - POST /api/voice/tts (Text-to-Speech)
func (h *Handlers) handleTTS(w http.ResponseWriter, r *http.Request) {
	if !common.RequirePOST(w, r) {
		return
	}

	var req struct {
		Text  string `json:"text"`
		Voice string `json:"voice,omitempty"`
	}

	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteBadRequest(w, "JSON parsen fehlgeschlagen")
		return
	}

	if req.Text == "" {
		common.WriteBadRequest(w, "Text ist erforderlich")
		return
	}

	log.Printf("TTS-Anfrage: %d Zeichen, Stimme: %s", len(req.Text), req.Voice)

	result, err := h.voiceService.SynthesizeSpeech(req.Text, req.Voice)
	if err != nil {
		log.Printf("TTS-Fehler: %v", err)
		common.WriteInternalError(w, err, "Sprachsynthese fehlgeschlagen")
		return
	}

	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Content-Disposition", "inline; filename=\"speech.wav\"")
	w.Header().Set("Content-Length", strconv.Itoa(len(result.AudioData)))
	w.Write(result.AudioData)
}

// handleDownload - GET/POST /api/voice/download (SSE)
func (h *Handlers) handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		common.WriteMethodNotAllowed(w)
		return
	}

	component := r.URL.Query().Get("component")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		common.WriteInternalError(w, nil, "Streaming nicht unterstützt")
		return
	}

	progressChan := make(chan voice.DownloadProgress, 10)

	go func() {
		defer close(progressChan)
		var err error

		switch component {
		case "whisper":
			err = h.voiceService.EnsureWhisperDownloaded(progressChan)
		case "piper":
			err = h.voiceService.EnsurePiperDownloaded(progressChan)
		default:
			err = h.voiceService.EnsureModelsDownloaded(progressChan)
		}

		if err != nil {
			progressChan <- voice.DownloadProgress{
				Component: component,
				Status:    "error",
				Error:     err.Error(),
			}
		}
	}()

	for progress := range progressChan {
		data, _ := json.Marshal(progress)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		if progress.Status == "error" || progress.Status == "done" {
			break
		}
	}

	fmt.Fprintf(w, "data: {\"status\":\"complete\"}\n\n")
	flusher.Flush()
}

// handleModels - GET /api/voice/models
func (h *Handlers) handleModels(w http.ResponseWriter, r *http.Request) {
	if !common.RequireGET(w, r) {
		return
	}

	installedWhisper := h.voiceService.GetInstalledWhisperModels()
	installedPiper := h.voiceService.GetInstalledPiperVoices()

	whisperSet := make(map[string]bool)
	for _, m := range installedWhisper {
		whisperSet[m] = true
	}
	piperSet := make(map[string]bool)
	for _, v := range installedPiper {
		piperSet[v] = true
	}

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

	status := h.voiceService.GetStatus()

	common.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"whisper":        whisperResult,
		"piper":          piperResult,
		"currentWhisper": status.Whisper.Model,
		"currentPiper":   status.Piper.Voice,
		"whisperBinary":  status.Whisper.BinaryFound,
		"piperBinary":    status.Piper.BinaryFound,
	})
}

// handleConfig - GET/POST /api/voice/config
func (h *Handlers) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		status := h.voiceService.GetStatus()
		common.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"whisperModel":    status.Whisper.Model,
			"whisperLanguage": status.Whisper.Language,
			"piperVoice":      status.Piper.Voice,
			"whisperReady":    status.Whisper.Available,
			"piperReady":      status.Piper.Available,
		})

	case http.MethodPost:
		var req struct {
			WhisperModel string `json:"whisperModel,omitempty"`
			PiperVoice   string `json:"piperVoice,omitempty"`
			Language     string `json:"language,omitempty"`
		}
		if err := common.DecodeJSON(r, &req); err != nil {
			common.WriteBadRequest(w, "Invalid JSON")
			return
		}

		if req.WhisperModel != "" {
			if err := h.voiceService.SetWhisperModel(req.WhisperModel); err != nil {
				log.Printf("Fehler beim Setzen des Whisper-Modells: %v", err)
			} else if h.settings != nil {
				_ = h.settings.SaveWhisperModel(req.WhisperModel)
			}
		}
		if req.PiperVoice != "" {
			if err := h.voiceService.SetPiperVoice(req.PiperVoice); err != nil {
				log.Printf("Fehler beim Setzen der Piper-Stimme: %v", err)
			} else if h.settings != nil {
				settings := h.settings.GetVoiceSettings()
				settings.PiperVoice = req.PiperVoice
				_ = h.settings.SaveVoiceSettings(settings)
			}
		}

		status := h.voiceService.GetStatus()
		common.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"status":       "ok",
			"whisperModel": status.Whisper.Model,
			"piperVoice":   status.Piper.Voice,
		})

	default:
		common.WriteMethodNotAllowed(w)
	}
}
