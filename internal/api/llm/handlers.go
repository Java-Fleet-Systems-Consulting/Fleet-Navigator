// Package llmhandlers enthält die HTTP-Handler für LLM-Endpoints
package llmhandlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"fleet-navigator/internal/api/common"
	"fleet-navigator/internal/llm"
)

// SettingsProvider liefert Provider-Einstellungen
type SettingsProvider interface {
	GetActiveProvider() string
	SaveActiveProvider(provider string) error
}

// GGUFModelInfo enthält Informationen über ein lokales GGUF-Modell
type GGUFModelInfo struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Modified string `json:"modified_at"`
}

// DownloadProgress enthält Fortschrittsinformationen für Downloads
type DownloadProgress struct {
	Percent    float64 `json:"percent"`
	Downloaded int64   `json:"downloaded"`
	Total      int64   `json:"total"`
	Filename   string  `json:"filename"`
}

// LlamaServerProvider stellt lokale GGUF-Modell-Funktionalität bereit
type LlamaServerProvider interface {
	GetAvailableModels() ([]GGUFModelInfo, error)
	IsRunning() bool
	GetContextSize() int
	DownloadModel(url, filename string, progress chan<- DownloadProgress) error
}

// OllamaConnectionChecker prüft die Ollama-Verbindung
type OllamaConnectionChecker interface {
	CheckOllamaConnection() bool
}

// Handlers enthält die HTTP-Handler für LLM-Endpoints
type Handlers struct {
	modelService   *llm.ModelService
	settings       SettingsProvider
	llamaServer    LlamaServerProvider
	ollamaChecker  OllamaConnectionChecker
}

// NewHandlers erstellt neue LLM-Handler
func NewHandlers(modelService *llm.ModelService, settings SettingsProvider) *Handlers {
	return &Handlers{
		modelService: modelService,
		settings:     settings,
	}
}

// SetLlamaServer setzt den LlamaServer-Provider (optional)
func (h *Handlers) SetLlamaServer(server LlamaServerProvider) {
	h.llamaServer = server
}

// SetOllamaChecker setzt den Ollama-Verbindungsprüfer (optional)
func (h *Handlers) SetOllamaChecker(checker OllamaConnectionChecker) {
	h.ollamaChecker = checker
}

// RegisterRoutes registriert die LLM-API-Routen
func (h *Handlers) RegisterRoutes(mux *http.ServeMux) {
	// Status & Info
	mux.HandleFunc("/api/llm/status", h.handleStatus)
	mux.HandleFunc("/api/llm/models", h.handleModels)
	mux.HandleFunc("/api/llm/models/installed", h.handleInstalledModels)
	mux.HandleFunc("/api/llm/models/registry", h.handleRegistry)
	mux.HandleFunc("/api/llm/models/featured", h.handleFeaturedModels)
	mux.HandleFunc("/api/llm/models/context", h.handleModelContext)

	// Model Operations
	mux.HandleFunc("/api/llm/models/pull", h.handlePullModel)
	mux.HandleFunc("/api/llm/models/delete", h.handleDeleteModel)
	mux.HandleFunc("/api/llm/models/details/", h.handleModelDetails)

	// Providers
	mux.HandleFunc("/api/llm/providers", h.handleProviders)
	mux.HandleFunc("/api/llm/providers/switch", h.handleProviderSwitch)
	mux.HandleFunc("/api/llm/providers/active", h.handleProviderActive)
}

// handleStatus - GET /api/llm/status
func (h *Handlers) handleStatus(w http.ResponseWriter, r *http.Request) {
	if !common.RequireGET(w, r) {
		return
	}

	status := map[string]interface{}{
		"provider":        h.modelService.GetProviderName(),
		"provider_online": h.modelService.IsProviderAvailable(),
		"selected_model":  h.modelService.GetSelectedModel(),
		"default_model":   h.modelService.GetDefaultModel(),
	}

	// Installierte Modelle zählen
	if models, err := h.modelService.GetInstalledModels(); err == nil {
		status["installed_models_count"] = len(models)
	}

	// Registry-Statistiken
	registry := h.modelService.GetRegistry()
	status["registry_total"] = len(registry.GetAllModels())
	status["registry_featured"] = len(registry.GetFeaturedModels())

	common.WriteJSON(w, http.StatusOK, status)
}

// handleModels - GET /api/llm/models
func (h *Handlers) handleModels(w http.ResponseWriter, r *http.Request) {
	if !common.RequireGET(w, r) {
		return
	}

	activeProvider := h.settings.GetActiveProvider()

	// Bei llama-cpp: GGUF-Dateien direkt lesen
	if isLlamaCppProvider(activeProvider) && h.llamaServer != nil {
		ggufModels, _ := h.llamaServer.GetAvailableModels()

		// In Frontend-Format konvertieren
		installed := make([]map[string]interface{}, 0, len(ggufModels))
		for _, m := range ggufModels {
			entry := h.modelService.GetRegistry().FindByFilename(m.Name)

			model := map[string]interface{}{
				"name":        m.Name,
				"path":        m.Path,
				"size":        m.Size,
				"size_human":  formatBytes(m.Size),
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

		common.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"installed":       installed,
			"registry":        h.modelService.GetAvailableModelsFromRegistry(),
			"selected_model":  h.modelService.GetSelectedModel(),
			"default_model":   h.modelService.GetDefaultModel(),
			"provider":        "llama-server",
			"provider_online": true,
		})
		return
	}

	// Ollama Provider
	installed, err := h.modelService.GetInstalledModels()
	if err != nil {
		common.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"installed": []interface{}{},
			"registry":  h.modelService.GetAvailableModelsFromRegistry(),
			"error":     err.Error(),
		})
		return
	}

	common.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"installed":       installed,
		"registry":        h.modelService.GetAvailableModelsFromRegistry(),
		"selected_model":  h.modelService.GetSelectedModel(),
		"default_model":   h.modelService.GetDefaultModel(),
		"provider":        h.modelService.GetProviderName(),
		"provider_online": h.modelService.IsProviderAvailable(),
	})
}

// handleInstalledModels - GET /api/llm/models/installed
func (h *Handlers) handleInstalledModels(w http.ResponseWriter, r *http.Request) {
	if !common.RequireGET(w, r) {
		return
	}

	activeProvider := h.settings.GetActiveProvider()

	// Bei llama-cpp: GGUF-Dateien direkt lesen
	if isLlamaCppProvider(activeProvider) && h.llamaServer != nil {
		ggufModels, _ := h.llamaServer.GetAvailableModels()

		models := make([]map[string]interface{}, 0, len(ggufModels))
		for _, m := range ggufModels {
			entry := h.modelService.GetRegistry().FindByFilename(m.Name)

			model := map[string]interface{}{
				"name":        m.Name,
				"path":        m.Path,
				"size":        m.Size,
				"size_human":  formatBytes(m.Size),
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

		common.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"models":         models,
			"selected_model": h.modelService.GetSelectedModel(),
			"provider":       "llama-server",
		})
		return
	}

	// Ollama Provider
	models, err := h.modelService.GetInstalledModels()
	if err != nil {
		common.WriteInternalError(w, err, "Modelle konnten nicht geladen werden")
		return
	}

	common.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"models":         models,
		"selected_model": h.modelService.GetSelectedModel(),
	})
}

// handleRegistry - GET /api/llm/models/registry
func (h *Handlers) handleRegistry(w http.ResponseWriter, r *http.Request) {
	if !common.RequireGET(w, r) {
		return
	}

	category := r.URL.Query().Get("category")

	var models []llm.ModelRegistryEntry
	if category != "" {
		models = h.modelService.GetModelsByCategory(llm.ModelCategory(category))
	} else {
		models = h.modelService.GetAvailableModelsFromRegistry()
	}

	common.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"models":     models,
		"categories": []string{"chat", "code", "vision", "compact"},
	})
}

// handleFeaturedModels - GET /api/llm/models/featured
func (h *Handlers) handleFeaturedModels(w http.ResponseWriter, r *http.Request) {
	if !common.RequireGET(w, r) {
		return
	}

	common.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"featured": h.modelService.GetFeaturedModels(),
		"trending": h.modelService.GetRegistry().GetTrendingModels(),
	})
}

// handleModelContext - GET /api/llm/models/context
func (h *Handlers) handleModelContext(w http.ResponseWriter, r *http.Request) {
	if !common.RequireGET(w, r) {
		return
	}

	modelName := r.URL.Query().Get("model")
	if modelName == "" && h.llamaServer != nil {
		common.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"currentContext": h.llamaServer.GetContextSize(),
			"defaultContext": llm.DefaultContextSize,
		})
		return
	}

	registry := h.modelService.GetRegistry()
	modelMaxContext := registry.GetModelContextSize(modelName)
	effectiveContext := registry.GetEffectiveContextSize(modelName, 0)

	currentContext := 0
	if h.llamaServer != nil {
		currentContext = h.llamaServer.GetContextSize()
	}
	restartNeeded := currentContext != effectiveContext

	common.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"model":            modelName,
		"modelMaxContext":  modelMaxContext,
		"effectiveContext": effectiveContext,
		"currentContext":   currentContext,
		"defaultContext":   llm.DefaultContextSize,
		"restartNeeded":    restartNeeded,
	})
}

// handlePullModel - POST /api/llm/models/pull
func (h *Handlers) handlePullModel(w http.ResponseWriter, r *http.Request) {
	if !common.RequirePOST(w, r) {
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteBadRequest(w, "Invalid JSON")
		return
	}

	if req.Name == "" {
		common.WriteBadRequest(w, "Model name required")
		return
	}

	// SSE Headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		common.WriteInternalError(w, nil, "Streaming not supported")
		return
	}

	activeProvider := h.settings.GetActiveProvider()
	log.Printf("Model Pull: %s (Provider: %s)", req.Name, activeProvider)

	// Bei llama-cpp: GGUF von HuggingFace herunterladen
	if isLlamaCppProvider(activeProvider) && h.llamaServer != nil {
		h.pullGGUFModel(w, flusher, req.Name)
		return
	}

	// Fallback: Ollama Provider
	err := h.modelService.PullModel(req.Name, func(progress string) {
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

// pullGGUFModel lädt ein GGUF-Modell von HuggingFace herunter
func (h *Handlers) pullGGUFModel(w http.ResponseWriter, flusher http.Flusher, modelName string) {
	entry := h.modelService.FindModelInRegistry(modelName)
	if entry == nil {
		entry = h.modelService.GetRegistry().FindByOllamaName(modelName)
	}

	if entry == nil || entry.HuggingFaceRepo == "" {
		errJSON, _ := json.Marshal(map[string]string{
			"error": fmt.Sprintf("Modell '%s' nicht in Registry gefunden oder keine HuggingFace URL", modelName),
		})
		fmt.Fprintf(w, "data: %s\n\n", errJSON)
		flusher.Flush()
		return
	}

	downloadURL := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", entry.HuggingFaceRepo, entry.Filename)
	log.Printf("Downloading GGUF from HuggingFace: %s", downloadURL)

	progressChan := make(chan DownloadProgress, 100)
	done := make(chan error, 1)

	go func() {
		done <- h.llamaServer.DownloadModel(downloadURL, entry.Filename, progressChan)
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
			"status":   "success",
			"message":  fmt.Sprintf("Modell %s erfolgreich heruntergeladen", entry.Filename),
			"filename": entry.Filename,
		})
		fmt.Fprintf(w, "data: %s\n\n", doneJSON)
	}
	flusher.Flush()
}

// handleDeleteModel - POST /api/llm/models/delete
func (h *Handlers) handleDeleteModel(w http.ResponseWriter, r *http.Request) {
	if !common.RequirePOST(w, r) {
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteBadRequest(w, "Invalid JSON")
		return
	}

	if req.Name == "" {
		common.WriteBadRequest(w, "Model name required")
		return
	}

	if err := h.modelService.DeleteModel(req.Name); err != nil {
		common.WriteInternalError(w, err, "Modell konnte nicht gelöscht werden")
		return
	}

	common.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Modell %s gelöscht", req.Name),
	})
}

// handleModelDetails - GET /api/llm/models/details/{name}
func (h *Handlers) handleModelDetails(w http.ResponseWriter, r *http.Request) {
	if !common.RequireGET(w, r) {
		return
	}

	// Extract model name from path
	modelName := r.URL.Path[len("/api/llm/models/details/"):]
	if modelName == "" {
		common.WriteBadRequest(w, "Model name required")
		return
	}

	details, err := h.modelService.GetModelDetails(modelName)
	if err != nil {
		common.WriteInternalError(w, err, "Details konnten nicht geladen werden")
		return
	}

	common.WriteJSON(w, http.StatusOK, details)
}

// handleProviders - GET /api/llm/providers
func (h *Handlers) handleProviders(w http.ResponseWriter, r *http.Request) {
	if !common.RequireGET(w, r) {
		return
	}

	activeProvider := h.settings.GetActiveProvider()

	ollamaAvailable := false
	if activeProvider == "ollama" {
		ollamaAvailable = h.modelService.IsProviderAvailable()
	}

	llamaServerAvailable := false
	if h.llamaServer != nil {
		llamaServerAvailable = h.llamaServer.IsRunning()
	}

	// Map internal names to frontend names
	frontendActiveProvider := activeProvider
	if activeProvider == "llama-server" {
		frontendActiveProvider = "java-llama-cpp"
	}

	common.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"activeProvider":     frontendActiveProvider,
		"availableProviders": []string{"ollama", "java-llama-cpp"},
		"providerStatus": map[string]bool{
			"ollama":         ollamaAvailable,
			"java-llama-cpp": llamaServerAvailable,
		},
	})
}

// handleProviderSwitch - POST /api/llm/providers/switch
func (h *Handlers) handleProviderSwitch(w http.ResponseWriter, r *http.Request) {
	if !common.RequirePOST(w, r) {
		return
	}

	var req struct {
		Provider string `json:"provider"`
	}
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteBadRequest(w, "Invalid JSON")
		return
	}

	requestedProvider := normalizeProviderName(req.Provider)
	currentProvider := h.settings.GetActiveProvider()

	log.Printf("Provider-Wechsel: %s -> %s (angefordert: %s)", currentProvider, requestedProvider, req.Provider)

	// Bei Wechsel zu Ollama: Verbindung prüfen
	if requestedProvider == "ollama" && h.ollamaChecker != nil {
		if !h.ollamaChecker.CheckOllamaConnection() {
			log.Printf("FEHLER: Ollama-Wechsel fehlgeschlagen - Server nicht erreichbar")

			// Fallback auf llama-server setzen
			_ = h.settings.SaveActiveProvider("llama-server")

			common.WriteJSON(w, http.StatusOK, map[string]interface{}{
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

	if err := h.settings.SaveActiveProvider(requestedProvider); err != nil {
		common.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"message": "Provider konnte nicht gespeichert werden",
			"error":   err.Error(),
		})
		return
	}

	log.Printf("Provider erfolgreich gewechselt zu: %s", requestedProvider)

	common.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success":          true,
		"message":          fmt.Sprintf("Provider erfolgreich auf %s gewechselt", getProviderDisplayName(requestedProvider)),
		"provider":         requestedProvider,
		"activeProvider":   requestedProvider,
		"showNotification": true,
		"notificationType": "success",
	})
}

// handleProviderActive - GET /api/llm/providers/active
func (h *Handlers) handleProviderActive(w http.ResponseWriter, r *http.Request) {
	if !common.RequireGET(w, r) {
		return
	}

	activeProvider := h.settings.GetActiveProvider()

	// Map to frontend name
	if activeProvider == "llama-server" {
		activeProvider = "java-llama-cpp"
	}

	common.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"provider": activeProvider,
	})
}

// --- Helper Functions ---

func isLlamaCppProvider(provider string) bool {
	return provider == "llama-cpp" || provider == "java-llama-cpp" || provider == "llama-server"
}

func normalizeProviderName(name string) string {
	switch name {
	case "ollama":
		return "ollama"
	case "llama-cpp", "llamacpp", "llama_cpp", "llama-server", "java-llama-cpp":
		return "llama-server"
	default:
		return "llama-server"
	}
}

func getProviderDisplayName(provider string) string {
	switch provider {
	case "ollama":
		return "Ollama"
	case "llama-server", "llama-cpp", "java-llama-cpp":
		return "llama-server (lokal)"
	default:
		return provider
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
