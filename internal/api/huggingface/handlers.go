// Package huggingface enthält die HTTP-Handler für HuggingFace-Endpoints
package huggingface

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"fleet-navigator/internal/api/common"
	"fleet-navigator/internal/llm"
)

// ModelDownloader lädt GGUF-Modelle herunter
type ModelDownloader interface {
	DownloadModelFromURL(w http.ResponseWriter, downloadURL, filename string)
}

// Handlers enthält die HTTP-Handler für HuggingFace-Endpoints
type Handlers struct {
	registry   *llm.ModelRegistry
	downloader ModelDownloader
}

// NewHandlers erstellt neue HuggingFace-Handler
func NewHandlers(registry *llm.ModelRegistry) *Handlers {
	return &Handlers{registry: registry}
}

// SetDownloader setzt den Model-Downloader (optional)
func (h *Handlers) SetDownloader(downloader ModelDownloader) {
	h.downloader = downloader
}

// RegisterRoutes registriert die HuggingFace-API-Routen
func (h *Handlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/huggingface/search", h.handleSearch)
	mux.HandleFunc("/api/huggingface/popular", h.handlePopular)
	mux.HandleFunc("/api/huggingface/german", h.handleGerman)
	mux.HandleFunc("/api/huggingface/instruct", h.handleInstruct)
	mux.HandleFunc("/api/huggingface/code", h.handleCode)
	mux.HandleFunc("/api/huggingface/vision", h.handleVision)
	mux.HandleFunc("/api/huggingface/details", h.handleDetails)
	mux.HandleFunc("/api/huggingface/download", h.handleDownload)
}

// handleSearch - GET /api/huggingface/search?query=...
func (h *Handlers) handleSearch(w http.ResponseWriter, r *http.Request) {
	if !common.RequireGET(w, r) {
		return
	}

	query := r.URL.Query().Get("query")
	if query == "" {
		query = r.URL.Query().Get("q")
	}
	if query == "" {
		common.WriteJSON(w, http.StatusOK, []interface{}{})
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 30
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// HuggingFace API aufrufen
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
		common.WriteJSON(w, http.StatusOK, []interface{}{})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("HuggingFace API Status: %d", resp.StatusCode)
		common.WriteJSON(w, http.StatusOK, []interface{}{})
		return
	}

	var hfModels []hfModel
	if err := json.NewDecoder(resp.Body).Decode(&hfModels); err != nil {
		log.Printf("HuggingFace JSON Parse Fehler: %v", err)
		common.WriteJSON(w, http.StatusOK, []interface{}{})
		return
	}

	results := make([]map[string]interface{}, 0, len(hfModels))
	for _, model := range hfModels {
		modelNameLower := strings.ToLower(model.ID)
		if !strings.Contains(modelNameLower, "gguf") {
			continue
		}

		paramSize := extractParamSize(model.ID, model.Tags)
		sizeBytes, sizeHuman := estimateModelSize(model.ID, paramSize)
		category := detectCategory(model.Tags)

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
			"sizeBytes":     sizeBytes,
			"sizeHuman":     sizeHuman,
			"modelSize":     sizeBytes,
			"siblings":      []string{},
			"createdAt":     model.CreatedAt,
			"lastModified":  model.LastModified,
			"source":        "huggingface",
		})
	}

	log.Printf("HuggingFace Suche '%s': %d Ergebnisse", query, len(results))
	common.WriteJSON(w, http.StatusOK, results)
}

// handlePopular - GET /api/huggingface/popular
func (h *Handlers) handlePopular(w http.ResponseWriter, r *http.Request) {
	if !common.RequireGET(w, r) {
		return
	}

	results := make([]map[string]interface{}, 0)
	for _, entry := range h.registry.GetAllModels() {
		if entry.Featured || entry.Category == "chat" || entry.Category == "coder" {
			results = append(results, h.registryEntryToMap(entry))
		}
	}

	common.WriteJSON(w, http.StatusOK, results)
}

// handleGerman - GET /api/huggingface/german
func (h *Handlers) handleGerman(w http.ResponseWriter, r *http.Request) {
	if !common.RequireGET(w, r) {
		return
	}

	results := make([]map[string]interface{}, 0)
	for _, entry := range h.registry.GetAllModels() {
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
			strings.Contains(strings.ToLower(entry.DisplayName), "qwen") {
			result := h.registryEntryToMap(entry)
			result["languages"] = entry.Languages
			results = append(results, result)
		}
	}

	common.WriteJSON(w, http.StatusOK, results)
}

// handleInstruct - GET /api/huggingface/instruct
func (h *Handlers) handleInstruct(w http.ResponseWriter, r *http.Request) {
	if !common.RequireGET(w, r) {
		return
	}

	results := make([]map[string]interface{}, 0)
	for _, entry := range h.registry.GetAllModels() {
		if entry.Category == "chat" ||
			strings.Contains(strings.ToLower(entry.DisplayName), "instruct") ||
			strings.Contains(strings.ToLower(entry.DisplayName), "chat") {
			results = append(results, h.registryEntryToMap(entry))
		}
	}

	common.WriteJSON(w, http.StatusOK, results)
}

// handleCode - GET /api/huggingface/code
func (h *Handlers) handleCode(w http.ResponseWriter, r *http.Request) {
	if !common.RequireGET(w, r) {
		return
	}

	results := make([]map[string]interface{}, 0)
	for _, entry := range h.registry.GetAllModels() {
		if entry.Category == "code" || entry.Category == "coder" ||
			strings.Contains(strings.ToLower(entry.DisplayName), "code") ||
			strings.Contains(strings.ToLower(entry.DisplayName), "coder") {
			results = append(results, h.registryEntryToMap(entry))
		}
	}

	common.WriteJSON(w, http.StatusOK, results)
}

// handleVision - GET /api/huggingface/vision
func (h *Handlers) handleVision(w http.ResponseWriter, r *http.Request) {
	if !common.RequireGET(w, r) {
		return
	}

	results := make([]map[string]interface{}, 0)
	for _, entry := range h.registry.GetAllModels() {
		if entry.Category == "vision" ||
			strings.Contains(strings.ToLower(entry.DisplayName), "vision") ||
			strings.Contains(strings.ToLower(entry.DisplayName), "llava") ||
			strings.Contains(strings.ToLower(entry.DisplayName), "bakllava") ||
			strings.Contains(strings.ToLower(entry.DisplayName), "moondream") {
			results = append(results, h.registryEntryToMap(entry))
		}
	}

	common.WriteJSON(w, http.StatusOK, results)
}

// handleDetails - GET /api/huggingface/details?modelId=...
func (h *Handlers) handleDetails(w http.ResponseWriter, r *http.Request) {
	if !common.RequireGET(w, r) {
		return
	}

	modelId := r.URL.Query().Get("modelId")
	if modelId == "" {
		common.WriteBadRequest(w, "modelId required")
		return
	}

	for _, entry := range h.registry.GetAllModels() {
		if entry.HuggingFaceRepo == modelId || entry.DisplayName == modelId || entry.ID == modelId {
			common.WriteJSON(w, http.StatusOK, map[string]interface{}{
				"name":             entry.DisplayName,
				"description":      entry.Description,
				"huggingFaceId":    entry.HuggingFaceRepo,
				"size":             entry.SizeHuman,
				"sizeBytes":        entry.SizeBytes,
				"parameters":       entry.ParameterSize,
				"quantization":     entry.Quantization,
				"category":         entry.Category,
				"downloadUrl":      fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", entry.HuggingFaceRepo, entry.Filename),
				"ggufFile":         entry.Filename,
				"license":          entry.License,
				"languages":        entry.Languages,
				"useCases":         entry.UseCases,
				"minRamGB":         entry.MinRamGB,
				"recommendedRamGB": entry.RecommendedRamGB,
			})
			return
		}
	}

	common.WriteNotFound(w, "Model not found")
}

// handleDownload - POST /api/huggingface/download
func (h *Handlers) handleDownload(w http.ResponseWriter, r *http.Request) {
	if !common.RequirePOST(w, r) {
		return
	}

	var req struct {
		URL      string `json:"url"`
		Filename string `json:"filename"`
	}

	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteBadRequest(w, "Invalid request body")
		return
	}

	if h.downloader == nil {
		common.WriteError(w, http.StatusServiceUnavailable, "Download-Service nicht verfügbar")
		return
	}

	h.downloader.DownloadModelFromURL(w, req.URL, req.Filename)
}

// --- Helper Functions ---

func (h *Handlers) registryEntryToMap(entry llm.ModelRegistryEntry) map[string]interface{} {
	return map[string]interface{}{
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
	}
}

type hfModel struct {
	ID           string   `json:"id"`
	ModelID      string   `json:"modelId"`
	Author       string   `json:"author"`
	Downloads    int      `json:"downloads"`
	Likes        int      `json:"likes"`
	Tags         []string `json:"tags"`
	CreatedAt    string   `json:"createdAt"`
	LastModified string   `json:"lastModified"`
	Private      bool     `json:"private"`
}

func extractParamSize(modelName string, tags []string) string {
	patterns := []string{"70B", "72B", "65B", "34B", "33B", "32B", "30B", "14B", "13B", "8B", "7B", "3B", "1.5B", "1B"}

	modelUpper := strings.ToUpper(modelName)
	for _, p := range patterns {
		if strings.Contains(modelUpper, p) || strings.Contains(modelUpper, "-"+p) {
			return p
		}
	}

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

func estimateModelSize(modelName string, paramSize string) (int64, string) {
	paramUpper := strings.ToUpper(paramSize)
	paramUpper = strings.TrimSuffix(paramUpper, "B")
	params, _ := strconv.ParseFloat(paramUpper, 64)

	if params == 0 {
		return 0, ""
	}

	modelLower := strings.ToLower(modelName)
	var bytesPerParam float64

	switch {
	case strings.Contains(modelLower, "q2_k"):
		bytesPerParam = 0.35
	case strings.Contains(modelLower, "q3_k"):
		bytesPerParam = 0.45
	case strings.Contains(modelLower, "q4_k"), strings.Contains(modelLower, "q4_0"), strings.Contains(modelLower, "iq4"):
		bytesPerParam = 0.55
	case strings.Contains(modelLower, "q5_k"), strings.Contains(modelLower, "q5_0"):
		bytesPerParam = 0.65
	case strings.Contains(modelLower, "q6_k"):
		bytesPerParam = 0.75
	case strings.Contains(modelLower, "q8_0"):
		bytesPerParam = 1.0
	case strings.Contains(modelLower, "f16"):
		bytesPerParam = 2.0
	default:
		bytesPerParam = 0.55 // Default: Q4
	}

	sizeBytes := int64(params * 1e9 * bytesPerParam)
	sizeGB := float64(sizeBytes) / (1024 * 1024 * 1024)
	sizeHuman := fmt.Sprintf("~%.1f GB", sizeGB)

	return sizeBytes, sizeHuman
}

func detectCategory(tags []string) string {
	for _, tag := range tags {
		tagLower := strings.ToLower(tag)
		if strings.Contains(tagLower, "code") || strings.Contains(tagLower, "coder") {
			return "code"
		}
		if strings.Contains(tagLower, "vision") || strings.Contains(tagLower, "llava") || strings.Contains(tagLower, "multimodal") {
			return "vision"
		}
	}
	return "chat"
}
