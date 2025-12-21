package prompts

import (
	"time"
)

// SystemPromptTemplate repräsentiert einen System-Prompt
type SystemPromptTemplate struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	IsDefault bool      `json:"isDefault"`
	CreatedAt time.Time `json:"createdAt"`
}
