package huggingface

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"fleet-navigator/internal/llm"
)

func TestExtractParamSize(t *testing.T) {
	tests := []struct {
		modelName string
		tags      []string
		expected  string
	}{
		{"TheBloke/Llama-2-7B-GGUF", []string{}, "7B"},
		{"TheBloke/Llama-2-13B-GGUF", []string{}, "13B"},
		{"TheBloke/Llama-2-70B-GGUF", []string{}, "70B"},
		{"qwen2.5-1.5b-instruct-gguf", []string{}, "1.5B"},
		{"SomeModel-GGUF", []string{"7B", "text-generation"}, "7B"},
		{"UnknownModel", []string{}, ""},
	}

	for _, tt := range tests {
		result := extractParamSize(tt.modelName, tt.tags)
		if result != tt.expected {
			t.Errorf("extractParamSize(%q, %v) = %q, expected %q", tt.modelName, tt.tags, result, tt.expected)
		}
	}
}

func TestEstimateModelSize(t *testing.T) {
	tests := []struct {
		modelName  string
		paramSize  string
		expectSize bool
		expectGB   float64
	}{
		{"model-q4_k_m", "7B", true, 3.5},
		{"model-q8_0", "7B", true, 7.0},
		{"model-f16", "7B", true, 14.0},
		{"model", "0", false, 0},
		{"model", "", false, 0},
	}

	for _, tt := range tests {
		sizeBytes, sizeHuman := estimateModelSize(tt.modelName, tt.paramSize)
		if tt.expectSize {
			if sizeBytes == 0 {
				t.Errorf("estimateModelSize(%q, %q) returned 0 bytes", tt.modelName, tt.paramSize)
			}
			if sizeHuman == "" {
				t.Errorf("estimateModelSize(%q, %q) returned empty size string", tt.modelName, tt.paramSize)
			}
		} else {
			if sizeBytes != 0 {
				t.Errorf("estimateModelSize(%q, %q) = %d, expected 0", tt.modelName, tt.paramSize, sizeBytes)
			}
		}
	}
}

func TestDetectCategory(t *testing.T) {
	tests := []struct {
		tags     []string
		expected string
	}{
		{[]string{"text-generation", "code"}, "code"},
		{[]string{"coder", "python"}, "code"},
		{[]string{"vision", "image"}, "vision"},
		{[]string{"llava", "multimodal"}, "vision"},
		{[]string{"text-generation"}, "chat"},
		{[]string{}, "chat"},
	}

	for _, tt := range tests {
		result := detectCategory(tt.tags)
		if result != tt.expected {
			t.Errorf("detectCategory(%v) = %q, expected %q", tt.tags, result, tt.expected)
		}
	}
}

// --- Handler Tests ---

// MockModelDownloader implementiert ModelDownloader für Tests
type MockModelDownloader struct {
	CalledWithURL      string
	CalledWithFilename string
	ShouldError        bool
}

func (m *MockModelDownloader) DownloadModelFromURL(w http.ResponseWriter, downloadURL, filename string) {
	m.CalledWithURL = downloadURL
	m.CalledWithFilename = filename
	if m.ShouldError {
		http.Error(w, "Download failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Write([]byte("data: {\"done\":true}\n\n"))
}

// createTestRegistry erstellt eine Test-Registry mit Beispiel-Modellen
func createTestRegistry() *llm.ModelRegistry {
	registry := llm.NewModelRegistry()
	// Die Registry hat bereits Standard-Modelle, wir brauchen nichts hinzuzufügen
	return registry
}

func TestHandleDetails_MethodNotAllowed(t *testing.T) {
	h := NewHandlers(createTestRegistry())

	req := httptest.NewRequest(http.MethodPost, "/api/huggingface/details?modelId=test", nil)
	w := httptest.NewRecorder()

	h.handleDetails(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestHandleDetails_MissingModelId(t *testing.T) {
	h := NewHandlers(createTestRegistry())

	req := httptest.NewRequest(http.MethodGet, "/api/huggingface/details", nil)
	w := httptest.NewRecorder()

	h.handleDetails(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleDetails_RegistryHit(t *testing.T) {
	registry := createTestRegistry()
	h := NewHandlers(registry)

	// Suche nach einem bekannten Modell in der Registry
	// Die Registry enthält Standard-Modelle wie "qwen2.5-7b"
	allModels := registry.GetAllModels()
	if len(allModels) == 0 {
		t.Skip("No models in registry")
	}

	testModel := allModels[0]
	req := httptest.NewRequest(http.MethodGet, "/api/huggingface/details?modelId="+testModel.HuggingFaceRepo, nil)
	w := httptest.NewRecorder()

	h.handleDetails(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if response["name"] == nil {
		t.Error("Response should contain 'name' field")
	}
}

func TestHandleDetails_NotFound(t *testing.T) {
	h := NewHandlers(createTestRegistry())

	req := httptest.NewRequest(http.MethodGet, "/api/huggingface/details?modelId=non-existent-model-xyz", nil)
	w := httptest.NewRecorder()

	h.handleDetails(w, req)

	// Aktuelle Implementierung gibt 404 zurück wenn nicht in Registry
	// TODO: HuggingFace API Fallback hinzufügen
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// TestHandleDetails_ResponseFields prüft dass alle erwarteten Felder zurückgegeben werden
func TestHandleDetails_ResponseFields(t *testing.T) {
	registry := createTestRegistry()
	h := NewHandlers(registry)

	allModels := registry.GetAllModels()
	if len(allModels) == 0 {
		t.Skip("No models in registry")
	}

	testModel := allModels[0]
	req := httptest.NewRequest(http.MethodGet, "/api/huggingface/details?modelId="+testModel.HuggingFaceRepo, nil)
	w := httptest.NewRecorder()

	h.handleDetails(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Prüfe erforderliche Felder
	requiredFields := []string{"name", "huggingFaceId", "downloadUrl", "ggufFile"}
	for _, field := range requiredFields {
		if response[field] == nil {
			t.Errorf("Response missing required field: %s", field)
		}
	}
}

func TestHandleDownload_MethodNotAllowed(t *testing.T) {
	h := NewHandlers(createTestRegistry())

	// PUT sollte nicht erlaubt sein
	req := httptest.NewRequest(http.MethodPut, "/api/huggingface/download", nil)
	w := httptest.NewRecorder()

	h.handleDownload(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestHandleDownload_GET_Success(t *testing.T) {
	h := NewHandlers(createTestRegistry())
	mock := &MockModelDownloader{}
	h.SetDownloader(mock)

	// GET mit Query-Parametern (wie EventSource)
	req := httptest.NewRequest(http.MethodGet, "/api/huggingface/download?modelId=bartowski/test-model-GGUF&filename=test-model-Q4_K_M.gguf", nil)
	w := httptest.NewRecorder()

	h.handleDownload(w, req)

	// Prüfen dass URL korrekt konstruiert wurde
	expectedURL := "https://huggingface.co/bartowski/test-model-GGUF/resolve/main/test-model-Q4_K_M.gguf"
	if mock.CalledWithURL != expectedURL {
		t.Errorf("Expected URL '%s', got '%s'", expectedURL, mock.CalledWithURL)
	}

	if mock.CalledWithFilename != "test-model-Q4_K_M.gguf" {
		t.Errorf("Expected filename 'test-model-Q4_K_M.gguf', got '%s'", mock.CalledWithFilename)
	}
}

func TestHandleDownload_GET_MissingModelId(t *testing.T) {
	h := NewHandlers(createTestRegistry())

	// GET ohne modelId
	req := httptest.NewRequest(http.MethodGet, "/api/huggingface/download?filename=test.gguf", nil)
	w := httptest.NewRecorder()

	h.handleDownload(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleDownload_GET_MissingFilename(t *testing.T) {
	h := NewHandlers(createTestRegistry())

	// GET ohne filename
	req := httptest.NewRequest(http.MethodGet, "/api/huggingface/download?modelId=bartowski/test", nil)
	w := httptest.NewRecorder()

	h.handleDownload(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleDownload_InvalidBody(t *testing.T) {
	h := NewHandlers(createTestRegistry())

	req := httptest.NewRequest(http.MethodPost, "/api/huggingface/download", bytes.NewBufferString("not json"))
	w := httptest.NewRecorder()

	h.handleDownload(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleDownload_POST_MissingFields(t *testing.T) {
	h := NewHandlers(createTestRegistry())

	// POST ohne url
	body := `{"filename":"model.gguf"}`
	req := httptest.NewRequest(http.MethodPost, "/api/huggingface/download", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.handleDownload(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleDownload_NoDownloader(t *testing.T) {
	h := NewHandlers(createTestRegistry())
	// Kein Downloader gesetzt

	body := `{"url":"https://example.com/model.gguf","filename":"model.gguf"}`
	req := httptest.NewRequest(http.MethodPost, "/api/huggingface/download", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.handleDownload(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestHandleDownload_GET_NoDownloader(t *testing.T) {
	h := NewHandlers(createTestRegistry())
	// Kein Downloader gesetzt

	req := httptest.NewRequest(http.MethodGet, "/api/huggingface/download?modelId=test/model&filename=model.gguf", nil)
	w := httptest.NewRecorder()

	h.handleDownload(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestHandleDownload_Success(t *testing.T) {
	h := NewHandlers(createTestRegistry())
	mock := &MockModelDownloader{}
	h.SetDownloader(mock)

	body := `{"url":"https://huggingface.co/test/model.gguf","filename":"test-model.gguf"}`
	req := httptest.NewRequest(http.MethodPost, "/api/huggingface/download", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.handleDownload(w, req)

	if mock.CalledWithURL != "https://huggingface.co/test/model.gguf" {
		t.Errorf("Expected URL 'https://huggingface.co/test/model.gguf', got '%s'", mock.CalledWithURL)
	}

	if mock.CalledWithFilename != "test-model.gguf" {
		t.Errorf("Expected filename 'test-model.gguf', got '%s'", mock.CalledWithFilename)
	}
}

func TestHandleDownload_BodyReadOnce(t *testing.T) {
	// Dieser Test stellt sicher, dass der Request-Body nur einmal gelesen wird
	// und korrekt an den Downloader weitergegeben wird
	h := NewHandlers(createTestRegistry())
	mock := &MockModelDownloader{}
	h.SetDownloader(mock)

	body := `{"url":"https://example.com/big-model.gguf","filename":"big-model.gguf"}`
	req := httptest.NewRequest(http.MethodPost, "/api/huggingface/download", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.handleDownload(w, req)

	// Prüfen dass die Parameter korrekt geparst wurden
	if mock.CalledWithURL == "" {
		t.Error("URL was not passed to downloader - body may have been read twice")
	}

	if mock.CalledWithFilename == "" {
		t.Error("Filename was not passed to downloader - body may have been read twice")
	}
}

func TestHandleSearch_EmptyQuery(t *testing.T) {
	h := NewHandlers(createTestRegistry())

	req := httptest.NewRequest(http.MethodGet, "/api/huggingface/search", nil)
	w := httptest.NewRecorder()

	h.handleSearch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response []interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if len(response) != 0 {
		t.Errorf("Expected empty array for empty query, got %d items", len(response))
	}
}

func TestHandlePopular(t *testing.T) {
	h := NewHandlers(createTestRegistry())

	req := httptest.NewRequest(http.MethodGet, "/api/huggingface/popular", nil)
	w := httptest.NewRecorder()

	h.handlePopular(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	// Sollte Featured-Modelle enthalten
	if len(response) == 0 {
		t.Log("Warning: No popular models returned")
	}
}

func TestHandleCode(t *testing.T) {
	h := NewHandlers(createTestRegistry())

	req := httptest.NewRequest(http.MethodGet, "/api/huggingface/code", nil)
	w := httptest.NewRecorder()

	h.handleCode(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	// Alle zurückgegebenen Modelle sollten Code-Kategorie haben
	for _, model := range response {
		category, ok := model["category"].(string)
		if !ok {
			continue
		}
		if category != "code" && category != "coder" {
			// Check if name contains "code"
			name, _ := model["displayName"].(string)
			if name != "" && !containsCode(name) {
				t.Errorf("Model %v should be code-related", model["displayName"])
			}
		}
	}
}

func containsCode(s string) bool {
	lower := stringToLower(s)
	return stringContains(lower, "code") || stringContains(lower, "coder")
}

func stringToLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) != -1
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func TestHandleVision(t *testing.T) {
	h := NewHandlers(createTestRegistry())

	req := httptest.NewRequest(http.MethodGet, "/api/huggingface/vision", nil)
	w := httptest.NewRecorder()

	h.handleVision(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	// Alle zurückgegebenen Modelle sollten Vision-Kategorie haben
	for _, model := range response {
		category, ok := model["category"].(string)
		if !ok {
			continue
		}
		if category != "vision" {
			name, _ := model["displayName"].(string)
			if name != "" && !containsVision(name) {
				t.Errorf("Model %v should be vision-related", model["displayName"])
			}
		}
	}
}

func containsVision(s string) bool {
	lower := stringToLower(s)
	return stringContains(lower, "vision") ||
		stringContains(lower, "llava") ||
		stringContains(lower, "bakllava") ||
		stringContains(lower, "moondream")
}
