// Package chat implementiert die Chat-Funktionalität (Legacy)
package chat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// LegacyConfig enthält die Chat-Konfiguration (Legacy)
type LegacyConfig struct {
	BaseURL string
	Model   string
	Timeout time.Duration
}

// DefaultConfig gibt die Standard-Konfiguration zurück
func DefaultConfig() LegacyConfig {
	return LegacyConfig{
		BaseURL: "http://localhost:8080", // llama-server default
		Model:   "default",
		Timeout: 120 * time.Second,
	}
}

// Message repräsentiert eine Chat-Nachricht
type Message struct {
	Role    string `json:"role"`    // "user", "assistant", "system"
	Content string `json:"content"`
}

// ChatRequest ist die Anfrage an den Chat-Server
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
	Options  *Options  `json:"options,omitempty"`
}

// Options für den Chat-Server
type Options struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
	TopK        int     `json:"top_k,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
}

// ChatResponse ist die Antwort vom Chat-Server (streaming)
type ChatResponse struct {
	Model     string  `json:"model"`
	CreatedAt string  `json:"created_at"`
	Message   Message `json:"message"`
	Done      bool    `json:"done"`
}

// Session repräsentiert eine Chat-Session mit Verlauf
type Session struct {
	ID        string
	Messages  []Message
	CreatedAt time.Time
	UpdatedAt time.Time
	mu        sync.RWMutex
}

// Service ist der Chat-Service (Legacy)
type Service struct {
	config   LegacyConfig
	client   *http.Client
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewService erstellt einen neuen Chat-Service
func NewService(config LegacyConfig) *Service {
	return &Service{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		sessions: make(map[string]*Session),
	}
}

// CreateSession erstellt eine neue Chat-Session
func (s *Service) CreateSession(sessionID string, systemPrompt string) *Session {
	session := &Session{
		ID:        sessionID,
		Messages:  make([]Message, 0),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if systemPrompt != "" {
		session.Messages = append(session.Messages, Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	s.mu.Lock()
	s.sessions[sessionID] = session
	s.mu.Unlock()

	return session
}

// GetSession gibt eine bestehende Session zurück
func (s *Service) GetSession(sessionID string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[sessionID]
	return session, ok
}

// GetOrCreateSession gibt eine Session zurück oder erstellt eine neue
func (s *Service) GetOrCreateSession(sessionID string, systemPrompt string) *Session {
	if session, ok := s.GetSession(sessionID); ok {
		return session
	}
	return s.CreateSession(sessionID, systemPrompt)
}

// DeleteSession löscht eine Session
func (s *Service) DeleteSession(sessionID string) {
	s.mu.Lock()
	delete(s.sessions, sessionID)
	s.mu.Unlock()
}

// Chat sendet eine Nachricht und gibt die Antwort zurück (nicht-streaming)
func (s *Service) Chat(ctx context.Context, sessionID, userMessage string) (string, error) {
	session := s.GetOrCreateSession(sessionID, "")

	// User-Nachricht hinzufügen
	session.mu.Lock()
	session.Messages = append(session.Messages, Message{
		Role:    "user",
		Content: userMessage,
	})
	messages := make([]Message, len(session.Messages))
	copy(messages, session.Messages)
	session.mu.Unlock()

	// Request erstellen
	reqBody := ChatRequest{
		Model:    s.config.Model,
		Messages: messages,
		Stream:   false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("JSON Marshal Fehler: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.config.BaseURL+"/api/chat", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("Request Erstellen Fehler: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Chat-Server nicht erreichbar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Server Fehler (Status %d): %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("Response Decode Fehler: %w", err)
	}

	// Assistant-Antwort zur Session hinzufügen
	session.mu.Lock()
	session.Messages = append(session.Messages, chatResp.Message)
	session.UpdatedAt = time.Now()
	session.mu.Unlock()

	return chatResp.Message.Content, nil
}

// StreamChat sendet eine Nachricht und streamt die Antwort
func (s *Service) StreamChat(ctx context.Context, sessionID, userMessage string, onChunk func(chunk string)) (string, error) {
	session := s.GetOrCreateSession(sessionID, "")

	// User-Nachricht hinzufügen
	session.mu.Lock()
	session.Messages = append(session.Messages, Message{
		Role:    "user",
		Content: userMessage,
	})
	messages := make([]Message, len(session.Messages))
	copy(messages, session.Messages)
	session.mu.Unlock()

	// Request erstellen
	reqBody := ChatRequest{
		Model:    s.config.Model,
		Messages: messages,
		Stream:   true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("JSON Marshal Fehler: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.config.BaseURL+"/api/chat", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("Request Erstellen Fehler: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Chat-Server nicht erreichbar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Server Fehler (Status %d): %s", resp.StatusCode, string(body))
	}

	// Streaming Response lesen
	var fullResponse strings.Builder
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var chatResp ChatResponse
		if err := json.Unmarshal([]byte(line), &chatResp); err != nil {
			continue
		}

		chunk := chatResp.Message.Content
		fullResponse.WriteString(chunk)

		if onChunk != nil {
			onChunk(chunk)
		}

		if chatResp.Done {
			break
		}
	}

	// Assistant-Antwort zur Session hinzufügen
	response := fullResponse.String()
	session.mu.Lock()
	session.Messages = append(session.Messages, Message{
		Role:    "assistant",
		Content: response,
	})
	session.UpdatedAt = time.Now()
	session.mu.Unlock()

	return response, nil
}

// GetHistory gibt den Chat-Verlauf einer Session zurück
func (s *Service) GetHistory(sessionID string) ([]Message, error) {
	session, ok := s.GetSession(sessionID)
	if !ok {
		return nil, fmt.Errorf("Session %s nicht gefunden", sessionID)
	}

	session.mu.RLock()
	defer session.mu.RUnlock()

	messages := make([]Message, len(session.Messages))
	copy(messages, session.Messages)
	return messages, nil
}

// ClearHistory löscht den Chat-Verlauf einer Session
func (s *Service) ClearHistory(sessionID string) error {
	session, ok := s.GetSession(sessionID)
	if !ok {
		return fmt.Errorf("Session %s nicht gefunden", sessionID)
	}

	session.mu.Lock()
	// Behalte nur den System-Prompt falls vorhanden
	if len(session.Messages) > 0 && session.Messages[0].Role == "system" {
		session.Messages = session.Messages[:1]
	} else {
		session.Messages = make([]Message, 0)
	}
	session.UpdatedAt = time.Now()
	session.mu.Unlock()

	return nil
}

// IsAvailable prüft ob der Chat-Server erreichbar ist
func (s *Service) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", s.config.BaseURL+"/api/tags", nil)
	if err != nil {
		return false
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// ListModels gibt alle verfügbaren Modelle zurück
func (s *Service) ListModels() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", s.config.BaseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Chat-Server nicht erreichbar: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	models := make([]string, len(result.Models))
	for i, m := range result.Models {
		models[i] = m.Name
	}

	return models, nil
}

// SetModel setzt das zu verwendende Modell
func (s *Service) SetModel(model string) {
	s.config.Model = model
}

// GetModel gibt das aktuelle Modell zurück
func (s *Service) GetModel() string {
	return s.config.Model
}

// StreamChatWithMessages sendet Messages direkt und streamt die Antwort
// Diese Methode arbeitet ohne Session, ideal für REST API
func (s *Service) StreamChatWithMessages(model string, messages []Message, onChunk func(content string, done bool)) error {
	if model == "" {
		model = s.config.Model
	}

	// Request erstellen
	reqBody := ChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("JSON Marshal Fehler: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", s.config.BaseURL+"/api/chat", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("Request Erstellen Fehler: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("Chat-Server nicht erreichbar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Server Fehler (Status %d): %s", resp.StatusCode, string(body))
	}

	// Streaming Response lesen
	scanner := bufio.NewScanner(resp.Body)
	// Größere Buffer für lange Zeilen
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var chatResp ChatResponse
		if err := json.Unmarshal([]byte(line), &chatResp); err != nil {
			continue
		}

		if onChunk != nil {
			onChunk(chatResp.Message.Content, chatResp.Done)
		}

		if chatResp.Done {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("Stream lesen Fehler: %w", err)
	}

	return nil
}
