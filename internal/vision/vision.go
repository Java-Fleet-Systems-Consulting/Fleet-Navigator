// Package vision implements vision AI functionality using LLaVA via Ollama
package vision

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Service provides vision AI capabilities using LLaVA
type Service struct {
	ollamaURL   string
	visionModel string
	client      *http.Client
}

// Config holds vision service configuration
type Config struct {
	OllamaURL   string
	VisionModel string
	Timeout     time.Duration
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		OllamaURL:   "http://localhost:11434",
		VisionModel: "llava:13b",
		Timeout:     180 * time.Second, // Vision takes longer
	}
}

// VisionMessage represents a message that can contain images
type VisionMessage struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Images  []string `json:"images,omitempty"` // Base64 encoded images
}

// VisionRequest is the request format for Ollama vision models
type VisionRequest struct {
	Model    string          `json:"model"`
	Messages []VisionMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  *VisionOptions  `json:"options,omitempty"`
}

// VisionOptions for vision model
type VisionOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

// VisionResponse is the streaming response from Ollama
type VisionResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Message   struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}

// ImageAnalysis represents the result of image analysis
type ImageAnalysis struct {
	Description string   `json:"description"`
	Text        string   `json:"text,omitempty"`        // OCR extracted text
	Objects     []string `json:"objects,omitempty"`     // Detected objects
	DocumentType string  `json:"documentType,omitempty"` // invoice, contract, letter, etc.
}

// NewService creates a new vision service
func NewService(config Config) *Service {
	if config.Timeout == 0 {
		config.Timeout = 180 * time.Second
	}
	return &Service{
		ollamaURL:   config.OllamaURL,
		visionModel: config.VisionModel,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// AnalyzeImage analyzes an image and returns a description
func (s *Service) AnalyzeImage(ctx context.Context, base64Image string, prompt string) (string, error) {
	if prompt == "" {
		prompt = "Beschreibe dieses Bild detailliert auf Deutsch. Wenn Text sichtbar ist, extrahiere ihn."
	}

	messages := []VisionMessage{
		{
			Role:    "user",
			Content: prompt,
			Images:  []string{base64Image},
		},
	}

	return s.chat(ctx, messages, false, nil)
}

// AnalyzeDocument analyzes a document image and extracts text/structure
func (s *Service) AnalyzeDocument(ctx context.Context, base64Image string) (*ImageAnalysis, error) {
	prompt := `Analysiere dieses Dokument auf Deutsch:
1. Was für ein Dokumenttyp ist es? (Rechnung, Vertrag, Brief, Formular, etc.)
2. Extrahiere den wichtigsten Text
3. Fasse den Inhalt zusammen

Antworte strukturiert mit:
DOKUMENTTYP: [typ]
TEXT: [extrahierter text]
ZUSAMMENFASSUNG: [zusammenfassung]`

	result, err := s.AnalyzeImage(ctx, base64Image, prompt)
	if err != nil {
		return nil, err
	}

	// Parse the structured response
	analysis := &ImageAnalysis{
		Description: result,
	}

	// Try to extract document type
	if idx := strings.Index(result, "DOKUMENTTYP:"); idx >= 0 {
		end := strings.Index(result[idx:], "\n")
		if end > 0 {
			analysis.DocumentType = strings.TrimSpace(result[idx+len("DOKUMENTTYP:") : idx+end])
		}
	}

	// Try to extract text
	if idx := strings.Index(result, "TEXT:"); idx >= 0 {
		nextSection := strings.Index(result[idx:], "ZUSAMMENFASSUNG:")
		if nextSection > 0 {
			analysis.Text = strings.TrimSpace(result[idx+len("TEXT:") : idx+nextSection])
		}
	}

	return analysis, nil
}

// StreamAnalyzeImage analyzes an image with streaming response
func (s *Service) StreamAnalyzeImage(ctx context.Context, base64Image string, prompt string, onChunk func(content string, done bool)) error {
	if prompt == "" {
		prompt = "Beschreibe dieses Bild detailliert auf Deutsch."
	}

	messages := []VisionMessage{
		{
			Role:    "user",
			Content: prompt,
			Images:  []string{base64Image},
		},
	}

	_, err := s.chat(ctx, messages, true, onChunk)
	return err
}

// ChatWithImage sends a chat message with an image and streams the response
func (s *Service) ChatWithImage(ctx context.Context, message string, base64Images []string, onChunk func(content string, done bool)) error {
	messages := []VisionMessage{
		{
			Role:    "user",
			Content: message,
			Images:  base64Images,
		},
	}

	_, err := s.chat(ctx, messages, true, onChunk)
	return err
}

// ChatWithImageAndHistory sends a chat message with image and conversation history
func (s *Service) ChatWithImageAndHistory(ctx context.Context, history []VisionMessage, message string, base64Images []string, onChunk func(content string, done bool)) error {
	// Add new message with images
	messages := append(history, VisionMessage{
		Role:    "user",
		Content: message,
		Images:  base64Images,
	})

	_, err := s.chat(ctx, messages, true, onChunk)
	return err
}

// chat is the internal method that handles the actual API call
func (s *Service) chat(ctx context.Context, messages []VisionMessage, stream bool, onChunk func(content string, done bool)) (string, error) {
	reqBody := VisionRequest{
		Model:    s.visionModel,
		Messages: messages,
		Stream:   stream,
		Options: &VisionOptions{
			Temperature: 0.7,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("JSON Marshal Fehler: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.ollamaURL+"/api/chat", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("Request Erstellen Fehler: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Ollama nicht erreichbar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama Fehler (Status %d): %s", resp.StatusCode, string(body))
	}

	if stream && onChunk != nil {
		// Streaming mode
		var fullResponse strings.Builder
		scanner := bufio.NewScanner(resp.Body)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var visionResp VisionResponse
			if err := json.Unmarshal([]byte(line), &visionResp); err != nil {
				continue
			}

			fullResponse.WriteString(visionResp.Message.Content)
			onChunk(visionResp.Message.Content, visionResp.Done)

			if visionResp.Done {
				break
			}
		}

		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("Stream lesen Fehler: %w", err)
		}

		return fullResponse.String(), nil
	}

	// Non-streaming mode
	var visionResp VisionResponse
	if err := json.NewDecoder(resp.Body).Decode(&visionResp); err != nil {
		return "", fmt.Errorf("Response Decode Fehler: %w", err)
	}

	return visionResp.Message.Content, nil
}

// IsAvailable checks if the vision model is available
func (s *Service) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", s.ollamaURL+"/api/tags", nil)
	if err != nil {
		return false
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false
	}

	// Check if vision model is in the list
	for _, m := range result.Models {
		if m.Name == s.visionModel || strings.HasPrefix(m.Name, "llava") {
			return true
		}
	}

	return false
}

// GetModel returns the current vision model name
func (s *Service) GetModel() string {
	return s.visionModel
}

// SetModel sets the vision model to use
func (s *Service) SetModel(model string) {
	s.visionModel = model
}

// IsVisionModel checks if a model name is a known vision model
func IsVisionModel(modelName string) bool {
	visionModels := []string{
		"llava",
		"llava:7b",
		"llava:13b",
		"llava:34b",
		"bakllava",
		"llava-llama3",
		"llava-phi3",
		"moondream",
	}

	modelLower := strings.ToLower(modelName)
	for _, vm := range visionModels {
		if strings.HasPrefix(modelLower, vm) {
			return true
		}
	}
	return false
}
