// Package llm implementiert die LLM Provider Abstraktion
// Unterstuetzt Ollama und llama.cpp
package llm

import (
	"context"
	"time"
)

// ProviderType definiert den Provider-Typ
type ProviderType string

const (
	ProviderOllama   ProviderType = "ollama"
	ProviderLlamaCpp ProviderType = "llamacpp"
)

// ProviderFeature definiert unterstuetzte Features
type ProviderFeature string

const (
	FeatureStreaming          ProviderFeature = "streaming"
	FeatureBlocking           ProviderFeature = "blocking"
	FeatureListModels         ProviderFeature = "list_models"
	FeaturePullModels         ProviderFeature = "pull_models"
	FeatureVision             ProviderFeature = "vision"
	FeatureCustomModels       ProviderFeature = "custom_models"
	FeatureDynamicContextSize ProviderFeature = "dynamic_context_size"
	FeatureGPUAcceleration    ProviderFeature = "gpu_acceleration"
)

// ModelInfo enthaelt Informationen ueber ein Modell
type ModelInfo struct {
	Name         string    `json:"name"`
	DisplayName  string    `json:"display_name,omitempty"`
	Provider     string    `json:"provider"`
	Size         int64     `json:"size"`
	SizeHuman    string    `json:"size_human,omitempty"`
	ModifiedAt   time.Time `json:"modified_at,omitempty"`
	Architecture string    `json:"architecture,omitempty"`
	Quantization string    `json:"quantization,omitempty"`
	Description  string    `json:"description,omitempty"`
	Installed    bool      `json:"installed"`
	Custom       bool      `json:"custom,omitempty"`
}

// ChatMessage repraesentiert eine Chat-Nachricht
type ChatMessage struct {
	Role    string `json:"role"`    // "user", "assistant", "system"
	Content string `json:"content"`
}

// ChatOptions enthaelt optionale Parameter fuer Chat
type ChatOptions struct {
	MaxTokens     int     `json:"max_tokens,omitempty"`
	Temperature   float64 `json:"temperature,omitempty"`
	TopP          float64 `json:"top_p,omitempty"`
	TopK          int     `json:"top_k,omitempty"`
	RepeatPenalty float64 `json:"repeat_penalty,omitempty"`
	Seed          int     `json:"seed,omitempty"`
	StopSequences []string `json:"stop_sequences,omitempty"`
}

// Provider definiert das Interface fuer LLM-Provider
type Provider interface {
	// GetProviderName gibt den Provider-Namen zurueck
	GetProviderName() string

	// GetProviderType gibt den Provider-Typ zurueck
	GetProviderType() ProviderType

	// IsAvailable prueft ob der Provider verfuegbar ist
	IsAvailable() bool

	// SupportsFeature prueft ob ein Feature unterstuetzt wird
	SupportsFeature(feature ProviderFeature) bool

	// GetSupportedFeatures gibt alle unterstuetzten Features zurueck
	GetSupportedFeatures() []ProviderFeature

	// === CHAT ===

	// Chat fuehrt einen nicht-streamenden Chat durch
	Chat(ctx context.Context, model, prompt, systemPrompt, requestID string) (string, error)

	// ChatStream fuehrt einen streamenden Chat durch
	ChatStream(ctx context.Context, model, prompt, systemPrompt, requestID string,
		onChunk func(chunk string), options *ChatOptions) error

	// ChatWithMessages fuehrt einen Chat mit Nachrichtenverlauf durch
	ChatWithMessages(ctx context.Context, model string, messages []ChatMessage,
		requestID string, onChunk func(chunk string, done bool), options *ChatOptions) error

	// === MODELL-VERWALTUNG ===

	// GetAvailableModels gibt verfuegbare Modelle zurueck
	GetAvailableModels() ([]ModelInfo, error)

	// PullModel laedt ein Modell herunter
	PullModel(modelName string, onProgress func(progress string)) error

	// DeleteModel loescht ein Modell
	DeleteModel(modelName string) error

	// GetModelDetails gibt Details zu einem Modell zurueck
	GetModelDetails(modelName string) (map[string]interface{}, error)

	// === REQUEST-MANAGEMENT ===

	// CancelRequest bricht einen laufenden Request ab
	CancelRequest(requestID string) bool
}

// ProviderConfig enthält die Konfiguration für einen Provider
type ProviderConfig struct {
	Type       ProviderType `json:"type"`
	BaseURL    string       `json:"base_url,omitempty"`
	ModelsDir  string       `json:"models_dir,omitempty"`
	Timeout    int          `json:"timeout,omitempty"` // in Sekunden
	GPULayers  int          `json:"gpu_layers,omitempty"`
	ContextSize int         `json:"context_size,omitempty"`
	Threads    int          `json:"threads,omitempty"`
}

// ProviderManager verwaltet mehrere LLM-Provider
type ProviderManager struct {
	providers       map[ProviderType]Provider
	activeProvider  ProviderType
	defaultProvider ProviderType
}

// NewProviderManager erstellt einen neuen Provider-Manager
func NewProviderManager() *ProviderManager {
	return &ProviderManager{
		providers:       make(map[ProviderType]Provider),
		activeProvider:  ProviderOllama,
		defaultProvider: ProviderOllama,
	}
}

// RegisterProvider registriert einen Provider
func (pm *ProviderManager) RegisterProvider(provider Provider) {
	pm.providers[provider.GetProviderType()] = provider
}

// GetProvider gibt einen Provider zurueck
func (pm *ProviderManager) GetProvider(providerType ProviderType) (Provider, bool) {
	p, ok := pm.providers[providerType]
	return p, ok
}

// GetActiveProvider gibt den aktiven Provider zurueck
func (pm *ProviderManager) GetActiveProvider() (Provider, bool) {
	return pm.GetProvider(pm.activeProvider)
}

// SetActiveProvider setzt den aktiven Provider
func (pm *ProviderManager) SetActiveProvider(providerType ProviderType) bool {
	if _, ok := pm.providers[providerType]; ok {
		pm.activeProvider = providerType
		return true
	}
	return false
}

// GetAllProviders gibt alle registrierten Provider zurueck
func (pm *ProviderManager) GetAllProviders() map[ProviderType]Provider {
	return pm.providers
}

// GetAvailableProviders gibt alle verfuegbaren Provider zurueck
func (pm *ProviderManager) GetAvailableProviders() []Provider {
	result := make([]Provider, 0)
	for _, p := range pm.providers {
		if p.IsAvailable() {
			result = append(result, p)
		}
	}
	return result
}
