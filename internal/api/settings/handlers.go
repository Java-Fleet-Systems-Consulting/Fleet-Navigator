// Package settings enthält die HTTP-Handler für Settings-Endpoints
package settings

import (
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"fleet-navigator/internal/api/common"
	"fleet-navigator/internal/settings"
)

// ProviderChecker ist ein Interface für Provider-Verbindungsprüfungen
type ProviderChecker interface {
	CheckProviderAvailable(provider string) bool
	CheckOllamaConnection() bool
}

// VoiceInfoProvider liefert Informationen über verfügbare Stimmen
type VoiceInfoProvider interface {
	GetAvailableVoicesForLocale(locale string) []map[string]interface{}
}

// Handlers enthält die HTTP-Handler für Settings-Endpoints
type Handlers struct {
	service         *settings.Service
	providerChecker ProviderChecker
	voiceProvider   VoiceInfoProvider
}

// NewHandlers erstellt neue Settings-Handler
func NewHandlers(service *settings.Service) *Handlers {
	return &Handlers{service: service}
}

// SetProviderChecker setzt den Provider-Checker (optional)
func (h *Handlers) SetProviderChecker(checker ProviderChecker) {
	h.providerChecker = checker
}

// SetVoiceProvider setzt den Voice-Provider (optional)
func (h *Handlers) SetVoiceProvider(provider VoiceInfoProvider) {
	h.voiceProvider = provider
}

// RegisterRoutes registriert die Settings-API-Routen
func (h *Handlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/settings", h.handleSettings)
	mux.HandleFunc("/api/settings/model-selection", h.handleModelSelection)
	mux.HandleFunc("/api/settings/selected-expert", h.handleSelectedExpert)
	mux.HandleFunc("/api/settings/ui-theme", h.handleUITheme)
	mux.HandleFunc("/api/settings/llm-provider", h.handleLLMProvider)
	mux.HandleFunc("/api/settings/document-model", h.handleDocumentModel)
	mux.HandleFunc("/api/settings/email-model", h.handleEmailModel)
	mux.HandleFunc("/api/settings/log-analysis-model", h.handleLogAnalysisModel)
	mux.HandleFunc("/api/settings/coder-model", h.handleCoderModel)
	mux.HandleFunc("/api/settings/sampling", h.handleSampling)
	mux.HandleFunc("/api/settings/chaining", h.handleChaining)
	mux.HandleFunc("/api/settings/preferences", h.handlePreferences)
	mux.HandleFunc("/api/settings/language", h.handleLanguage)
}

// handleSettings - GET /api/settings (alle Einstellungen)
func (h *Handlers) handleSettings(w http.ResponseWriter, r *http.Request) {
	if !common.RequireGET(w, r) {
		return
	}

	allSettings, err := h.service.GetAll()
	if err != nil {
		common.WriteInternalError(w, err, "Einstellungen konnten nicht geladen werden")
		return
	}

	// Als Key-Value Map zurückgeben
	settingsMap := make(map[string]string)
	for _, s := range allSettings {
		settingsMap[s.Key] = s.Value
	}

	common.WriteJSON(w, http.StatusOK, settingsMap)
}

// handleModelSelection - GET/PUT /api/settings/model-selection
func (h *Handlers) handleModelSelection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		modelSettings := h.service.GetModelSelectionSettings()
		common.WriteJSON(w, http.StatusOK, modelSettings)

	case http.MethodPut:
		var req settings.ModelSelectionSettings
		if err := common.DecodeJSON(r, &req); err != nil {
			common.WriteBadRequest(w, "Invalid JSON")
			return
		}

		if err := h.service.UpdateModelSelectionSettings(req); err != nil {
			common.WriteInternalError(w, err, "Einstellungen konnten nicht gespeichert werden")
			return
		}

		common.WriteJSON(w, http.StatusOK, req)

	default:
		common.WriteMethodNotAllowed(w)
	}
}

// handleSelectedExpert - GET/POST /api/settings/selected-expert
func (h *Handlers) handleSelectedExpert(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		expertID := h.service.GetSelectedExpertID()
		if expertID == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		common.WriteJSON(w, http.StatusOK, expertID)

	case http.MethodPost:
		body, _ := io.ReadAll(io.LimitReader(r.Body, 1024))
		expertIDStr := strings.TrimSpace(string(body))

		if expertIDStr == "" || expertIDStr == "null" {
			h.service.SaveSelectedExpertID(0)
			w.WriteHeader(http.StatusOK)
			return
		}

		expertID, err := strconv.ParseInt(expertIDStr, 10, 64)
		if err != nil {
			common.WriteBadRequest(w, "Invalid expert ID")
			return
		}

		if err := h.service.SaveSelectedExpertID(expertID); err != nil {
			common.WriteInternalError(w, err, "Experte konnte nicht gespeichert werden")
			return
		}

		w.WriteHeader(http.StatusOK)

	default:
		common.WriteMethodNotAllowed(w)
	}
}

// handleUITheme - GET/POST /api/settings/ui-theme
func (h *Handlers) handleUITheme(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		theme := h.service.GetUITheme()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(theme))

	case http.MethodPost:
		body, _ := io.ReadAll(io.LimitReader(r.Body, 1024))
		theme := strings.TrimSpace(string(body))
		// Remove quotes if sent as JSON string
		theme = strings.Trim(theme, "\"")

		if theme == "" {
			theme = "tech-dark"
		}

		if err := h.service.SaveUITheme(theme); err != nil {
			common.WriteInternalError(w, err, "Theme konnte nicht gespeichert werden")
			return
		}

		w.WriteHeader(http.StatusOK)

	default:
		common.WriteMethodNotAllowed(w)
	}
}

// handleLLMProvider - GET/POST /api/settings/llm-provider
func (h *Handlers) handleLLMProvider(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		provider := h.service.GetActiveProvider()
		available := true
		if h.providerChecker != nil {
			available = h.providerChecker.CheckProviderAvailable(provider)
		}
		common.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"provider":  provider,
			"available": available,
		})

	case http.MethodPost:
		var req struct {
			Provider string `json:"provider"`
		}
		if err := common.DecodeJSON(r, &req); err != nil {
			common.WriteBadRequest(w, "Invalid JSON")
			return
		}

		requestedProvider := normalizeProviderName(req.Provider)
		log.Printf("Provider-Wechsel angefordert: %s (original: %s)", requestedProvider, req.Provider)

		// Bei Wechsel zu Ollama: Verbindung prüfen
		if requestedProvider == "ollama" && h.providerChecker != nil {
			if !h.providerChecker.CheckOllamaConnection() {
				log.Printf("Ollama nicht erreichbar, verwende llama-cpp als Fallback")
				common.WriteJSON(w, http.StatusOK, map[string]interface{}{
					"success":      false,
					"provider":     "llama-cpp",
					"error":        "Ollama nicht erreichbar. Fallback auf llama-cpp.",
					"ollamaStatus": "offline",
				})
				return
			}
		}

		if err := h.service.SaveActiveProvider(requestedProvider); err != nil {
			common.WriteInternalError(w, err, "Provider konnte nicht gespeichert werden")
			return
		}

		common.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"success":  true,
			"provider": requestedProvider,
		})

	default:
		common.WriteMethodNotAllowed(w)
	}
}

// handleDocumentModel - GET/POST /api/settings/document-model
func (h *Handlers) handleDocumentModel(w http.ResponseWriter, r *http.Request) {
	h.handleTextModelSetting(w, r, h.service.GetDocumentModel, h.service.SaveDocumentModel)
}

// handleEmailModel - GET/POST /api/settings/email-model
func (h *Handlers) handleEmailModel(w http.ResponseWriter, r *http.Request) {
	h.handleTextModelSetting(w, r, h.service.GetEmailModel, h.service.SaveEmailModel)
}

// handleLogAnalysisModel - GET/POST /api/settings/log-analysis-model
func (h *Handlers) handleLogAnalysisModel(w http.ResponseWriter, r *http.Request) {
	h.handleTextModelSetting(w, r, h.service.GetLogAnalysisModel, h.service.SaveLogAnalysisModel)
}

// handleCoderModel - GET/POST /api/settings/coder-model
func (h *Handlers) handleCoderModel(w http.ResponseWriter, r *http.Request) {
	h.handleTextModelSetting(w, r, h.service.GetCoderModel, h.service.SaveCoderModel)
}

// handleTextModelSetting ist ein generischer Handler für text/plain Model-Settings
func (h *Handlers) handleTextModelSetting(w http.ResponseWriter, r *http.Request, getter func() string, setter func(string) error) {
	switch r.Method {
	case http.MethodGet:
		model := getter()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(model))

	case http.MethodPost:
		body, _ := io.ReadAll(io.LimitReader(r.Body, 1024))
		model := strings.TrimSpace(string(body))
		if err := setter(model); err != nil {
			common.WriteInternalError(w, err, "Modell konnte nicht gespeichert werden")
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		common.WriteMethodNotAllowed(w)
	}
}

// handleSampling - GET/POST /api/settings/sampling
func (h *Handlers) handleSampling(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		params := h.service.GetSamplingParams()
		common.WriteJSON(w, http.StatusOK, params)

	case http.MethodPost:
		var params settings.SamplingParams
		if err := common.DecodeJSON(r, &params); err != nil {
			common.WriteBadRequest(w, "Invalid JSON")
			return
		}
		if err := h.service.SaveSamplingParams(params); err != nil {
			common.WriteInternalError(w, err, "Sampling-Parameter konnten nicht gespeichert werden")
			return
		}
		common.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": true, "params": params})

	default:
		common.WriteMethodNotAllowed(w)
	}
}

// handleChaining - GET/POST /api/settings/chaining
func (h *Handlers) handleChaining(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		chainingSettings := h.service.GetChainingSettings()
		common.WriteJSON(w, http.StatusOK, chainingSettings)

	case http.MethodPost:
		var chainingSettings settings.ChainingSettings
		if err := common.DecodeJSON(r, &chainingSettings); err != nil {
			common.WriteBadRequest(w, "Invalid JSON")
			return
		}
		if err := h.service.SaveChainingSettings(chainingSettings); err != nil {
			common.WriteInternalError(w, err, "Chaining-Settings konnten nicht gespeichert werden")
			return
		}
		common.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": true, "settings": chainingSettings})

	default:
		common.WriteMethodNotAllowed(w)
	}
}

// handlePreferences - GET/POST /api/settings/preferences
func (h *Handlers) handlePreferences(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		prefs := h.service.GetUserPreferences()
		common.WriteJSON(w, http.StatusOK, prefs)

	case http.MethodPost:
		var prefs settings.UserPreferences
		if err := common.DecodeJSON(r, &prefs); err != nil {
			common.WriteBadRequest(w, "Invalid JSON")
			return
		}
		if err := h.service.SaveUserPreferences(prefs); err != nil {
			common.WriteInternalError(w, err, "Präferenzen konnten nicht gespeichert werden")
			return
		}
		common.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": true, "preferences": prefs})

	default:
		common.WriteMethodNotAllowed(w)
	}
}

// handleLanguage - GET/POST /api/settings/language
func (h *Handlers) handleLanguage(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		locale := h.service.GetLocale()
		response := map[string]interface{}{
			"locale":           locale,
			"supportedLocales": []string{"de", "en", "tr"},
		}
		if h.voiceProvider != nil {
			response["availableVoices"] = h.voiceProvider.GetAvailableVoicesForLocale(locale)
		}
		common.WriteJSON(w, http.StatusOK, response)

	case http.MethodPost:
		var req struct {
			Locale string `json:"locale"`
		}
		if err := common.DecodeJSON(r, &req); err != nil {
			common.WriteBadRequest(w, "Invalid JSON")
			return
		}

		locale := strings.ToLower(strings.TrimSpace(req.Locale))
		if locale == "" {
			locale = "de"
		}

		// Nur unterstützte Locales akzeptieren
		validLocales := map[string]bool{"de": true, "en": true, "tr": true}
		if !validLocales[locale] {
			common.WriteBadRequest(w, "Unsupported locale")
			return
		}

		if err := h.service.SaveLocale(locale); err != nil {
			common.WriteInternalError(w, err, "Sprache konnte nicht gespeichert werden")
			return
		}

		response := map[string]interface{}{
			"success":          true,
			"locale":           locale,
			"supportedLocales": []string{"de", "en", "tr"},
		}
		if h.voiceProvider != nil {
			response["availableVoices"] = h.voiceProvider.GetAvailableVoicesForLocale(locale)
		}
		common.WriteJSON(w, http.StatusOK, response)

	default:
		common.WriteMethodNotAllowed(w)
	}
}

// normalizeProviderName normalisiert Provider-Namen
func normalizeProviderName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	switch name {
	case "ollama":
		return "ollama"
	case "llama-cpp", "llamacpp", "llama_cpp", "llama-server":
		return "llama-cpp"
	default:
		return "llama-cpp"
	}
}
