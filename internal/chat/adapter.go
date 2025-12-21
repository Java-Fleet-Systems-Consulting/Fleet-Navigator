// Package chat - Adapter fuer WebSocket-Integration
package chat

import (
	"context"
	"fmt"
	"log"

	"fleet-navigator/internal/models"
)

// ProviderChecker interface zum Abfragen des aktiven Providers
type ProviderChecker interface {
	GetActiveProvider() string
}

// LlamaSamplingParams für Sampling-Parameter (identisch mit llamaserver.SamplingParams)
type LlamaSamplingParams struct {
	Temperature float64
	TopP        float64
	MaxTokens   int
}

// LlamaServerChatter interface für llama-server Chat
type LlamaServerChatter interface {
	StreamChat(messages []LlamaMessage, onChunk func(content string, done bool)) error
	StreamChatWithParams(messages []LlamaMessage, params LlamaSamplingParams, onChunk func(content string, done bool)) error
	IsRunning() bool
	IsHealthy() bool
}

// LlamaMessage für llama-server kompatibilität (identisch mit llamaserver.ChatMessage)
type LlamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Adapter wraps Service fuer WebSocket ChatHandler Interface
type Adapter struct {
	service          *Service
	selectionService *models.SelectionService
	systemPrompt     string
	providerChecker  ProviderChecker
	llamaServer      LlamaServerChatter
}

// NewAdapter erstellt einen neuen Chat-Adapter
func NewAdapter(service *Service, selectionService *models.SelectionService, systemPrompt string) *Adapter {
	return &Adapter{
		service:          service,
		selectionService: selectionService,
		systemPrompt:     systemPrompt,
	}
}

// SetProviderChecker setzt den ProviderChecker (settingsService)
func (a *Adapter) SetProviderChecker(checker ProviderChecker) {
	a.providerChecker = checker
}

// SetLlamaServer setzt den llama-server für Provider-basiertes Routing
func (a *Adapter) SetLlamaServer(server LlamaServerChatter) {
	a.llamaServer = server
}

// Chat implementiert das ChatHandler Interface
func (a *Adapter) Chat(sessionID, message string, onChunk func(chunk string)) (string, error) {
	log.Printf("Chat-Adapter: Verwende llama-server")
	return a.chatWithLlamaServer(sessionID, message, onChunk)
}

// chatWithLlamaServer verwendet den llama-server für Chat
func (a *Adapter) chatWithLlamaServer(sessionID, message string, onChunk func(chunk string)) (string, error) {
	if a.llamaServer == nil {
		return "", fmt.Errorf("llama-server ist nicht konfiguriert")
	}

	if !a.llamaServer.IsRunning() || !a.llamaServer.IsHealthy() {
		return "", fmt.Errorf("llama-server ist nicht aktiv. Bitte starte den Server in den Einstellungen.")
	}

	// Session-History holen oder erstellen
	session := a.service.GetOrCreateSession(sessionID, a.systemPrompt)

	// User-Nachricht zur Session hinzufügen
	session.mu.Lock()
	session.Messages = append(session.Messages, Message{
		Role:    "user",
		Content: message,
	})
	// Messages in llama-server Format konvertieren
	llamaMessages := make([]LlamaMessage, len(session.Messages))
	for i, msg := range session.Messages {
		llamaMessages[i] = LlamaMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	session.mu.Unlock()

	// Antwort sammeln
	var fullResponse string
	err := a.llamaServer.StreamChat(llamaMessages, func(content string, done bool) {
		fullResponse += content
		if onChunk != nil && content != "" {
			onChunk(content)
		}
	})

	if err != nil {
		return "", err
	}

	// Assistant-Antwort zur Session hinzufügen
	session.mu.Lock()
	session.Messages = append(session.Messages, Message{
		Role:    "assistant",
		Content: fullResponse,
	})
	session.mu.Unlock()

	return fullResponse, nil
}


// ChatWithModel verwendet ein explizit gewaehltes Modell
func (a *Adapter) ChatWithModel(sessionID, message, model string, onChunk func(chunk string)) (string, error) {
	log.Printf("Using explicit model: %s", model)

	// Modell im Service setzen
	originalModel := a.service.GetModel()
	a.service.SetModel(model)
	defer a.service.SetModel(originalModel)

	// Session mit System-Prompt erstellen falls nicht vorhanden
	a.service.GetOrCreateSession(sessionID, a.systemPrompt)

	ctx := context.Background()
	return a.service.StreamChat(ctx, sessionID, message, onChunk)
}

// ChatWithSystemPrompt verwendet einen custom System-Prompt (für Mates)
func (a *Adapter) ChatWithSystemPrompt(sessionID, message, customSystemPrompt string, onChunk func(chunk string)) (string, error) {
	if a.llamaServer == nil {
		return "", fmt.Errorf("llama-server ist nicht konfiguriert")
	}

	if !a.llamaServer.IsRunning() || !a.llamaServer.IsHealthy() {
		return "", fmt.Errorf("llama-server ist nicht aktiv")
	}

	// Verwende custom systemPrompt wenn vorhanden, sonst Standard
	effectivePrompt := customSystemPrompt
	if effectivePrompt == "" {
		effectivePrompt = a.systemPrompt
	}

	log.Printf("Chat mit custom System-Prompt (%d Zeichen)", len(effectivePrompt))

	// Session mit custom System-Prompt erstellen
	session := a.service.GetOrCreateSession(sessionID, effectivePrompt)

	// User-Nachricht zur Session hinzufügen
	session.mu.Lock()
	session.Messages = append(session.Messages, Message{
		Role:    "user",
		Content: message,
	})
	llamaMessages := make([]LlamaMessage, len(session.Messages))
	for i, msg := range session.Messages {
		llamaMessages[i] = LlamaMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	session.mu.Unlock()

	// Antwort sammeln
	var fullResponse string
	err := a.llamaServer.StreamChat(llamaMessages, func(content string, done bool) {
		fullResponse += content
		if onChunk != nil && content != "" {
			onChunk(content)
		}
	})

	if err != nil {
		return "", err
	}

	// Assistant-Antwort zur Session hinzufügen
	session.mu.Lock()
	session.Messages = append(session.Messages, Message{
		Role:    "assistant",
		Content: fullResponse,
	})
	session.mu.Unlock()

	return fullResponse, nil
}

// ClearHistory implementiert das ChatHandler Interface
func (a *Adapter) ClearHistory(sessionID string) error {
	return a.service.ClearHistory(sessionID)
}

// GetModelConfig gibt die aktuelle Model-Konfiguration zurueck
func (a *Adapter) GetModelConfig() models.SelectionConfig {
	return a.selectionService.GetConfig()
}

// UpdateModelConfig aktualisiert die Model-Konfiguration
func (a *Adapter) UpdateModelConfig(config models.SelectionConfig) {
	a.selectionService.UpdateConfig(config)
}
