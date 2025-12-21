package custommodel

import (
	"time"
)

// CustomModel repräsentiert ein benutzerdefinertes Ollama-Modell
type CustomModel struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`           // Model name in Ollama (e.g. "nova:latest")
	BaseModel     string     `json:"baseModel"`      // Base model (e.g. "llama3.2:3b")
	SystemPrompt  string     `json:"systemPrompt"`   // System prompt defining character
	Description   string     `json:"description"`    // Short description
	Temperature   *float64   `json:"temperature"`    // 0.0 - 2.0
	TopP          *float64   `json:"topP"`           // 0.0 - 1.0
	TopK          *int       `json:"topK"`           // 0 - 100
	RepeatPenalty *float64   `json:"repeatPenalty"`  // Repeat penalty
	NumPredict    *int       `json:"numPredict"`     // Max tokens to generate
	NumCtx        *int       `json:"numCtx"`         // Context window size (max 131072)
	OllamaDigest  string     `json:"ollamaDigest"`   // Ollama model digest
	ParentModelID *int64     `json:"parentModelId"`  // Parent model ID for versioning
	Version       int        `json:"version"`        // Version number (starts at 1)
	Modelfile     string     `json:"modelfile"`      // Complete Modelfile content
	Size          *int64     `json:"size"`           // Model size in bytes
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// CreateRequest für das Erstellen eines Custom Models
type CreateRequest struct {
	Name          string   `json:"name"`
	BaseModel     string   `json:"baseModel"`
	SystemPrompt  string   `json:"systemPrompt"`
	Description   string   `json:"description,omitempty"`
	Temperature   *float64 `json:"temperature,omitempty"`
	TopP          *float64 `json:"topP,omitempty"`
	TopK          *int     `json:"topK,omitempty"`
	RepeatPenalty *float64 `json:"repeatPenalty,omitempty"`
	NumPredict    *int     `json:"numPredict,omitempty"`
	NumCtx        *int     `json:"numCtx,omitempty"`
}

// UpdateRequest für das Aktualisieren eines Custom Models
type UpdateRequest struct {
	SystemPrompt  *string  `json:"systemPrompt,omitempty"`
	Description   *string  `json:"description,omitempty"`
	Temperature   *float64 `json:"temperature,omitempty"`
	TopP          *float64 `json:"topP,omitempty"`
	TopK          *int     `json:"topK,omitempty"`
	RepeatPenalty *float64 `json:"repeatPenalty,omitempty"`
	NumPredict    *int     `json:"numPredict,omitempty"`
	NumCtx        *int     `json:"numCtx,omitempty"`
}

// GgufModelConfig repräsentiert eine Konfiguration für ein GGUF-Modell (llama-server)
type GgufModelConfig struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`          // Display name
	BaseModel     string    `json:"baseModel"`     // GGUF filename or path
	Description   string    `json:"description"`   // Short description
	SystemPrompt  string    `json:"systemPrompt"`  // System prompt
	Temperature   float64   `json:"temperature"`   // 0.0 - 2.0
	TopP          float64   `json:"topP"`          // 0.0 - 1.0
	TopK          int       `json:"topK"`          // 0 - 100
	RepeatPenalty float64   `json:"repeatPenalty"` // Repeat penalty
	MaxTokens     int       `json:"maxTokens"`     // Max tokens to generate
	ContextSize   int       `json:"contextSize"`   // Context window size
	GpuLayers     int       `json:"gpuLayers"`     // Number of GPU layers (-1 = all)
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// GgufConfigCreateRequest für das Erstellen einer GGUF-Konfiguration
type GgufConfigCreateRequest struct {
	Name          string  `json:"name"`
	BaseModel     string  `json:"baseModel"`
	Description   string  `json:"description,omitempty"`
	SystemPrompt  string  `json:"systemPrompt,omitempty"`
	Temperature   float64 `json:"temperature"`
	TopP          float64 `json:"topP"`
	TopK          int     `json:"topK"`
	RepeatPenalty float64 `json:"repeatPenalty"`
	MaxTokens     int     `json:"maxTokens"`
	ContextSize   int     `json:"contextSize"`
	GpuLayers     int     `json:"gpuLayers"`
}

// GgufConfigUpdateRequest für das Aktualisieren einer GGUF-Konfiguration
type GgufConfigUpdateRequest struct {
	Name          *string  `json:"name,omitempty"`
	Description   *string  `json:"description,omitempty"`
	SystemPrompt  *string  `json:"systemPrompt,omitempty"`
	Temperature   *float64 `json:"temperature,omitempty"`
	TopP          *float64 `json:"topP,omitempty"`
	TopK          *int     `json:"topK,omitempty"`
	RepeatPenalty *float64 `json:"repeatPenalty,omitempty"`
	MaxTokens     *int     `json:"maxTokens,omitempty"`
	ContextSize   *int     `json:"contextSize,omitempty"`
	GpuLayers     *int     `json:"gpuLayers,omitempty"`
}
