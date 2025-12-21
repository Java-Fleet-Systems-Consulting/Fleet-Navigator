// Package models implementiert Smart Model Selection
// Automatische Modellauswahl basierend auf Prompt-Inhalt
package models

import (
	"regexp"
	"strings"
	"sync"
)

// TaskType definiert den Aufgabentyp
type TaskType string

const (
	TaskCode     TaskType = "CODE"      // Code-Generierung, Debugging
	TaskSimpleQA TaskType = "SIMPLE_QA" // Einfache Fragen
	TaskComplex  TaskType = "COMPLEX"   // Komplexe Aufgaben
	TaskVision   TaskType = "VISION"    // Bildanalyse
)

// SelectionConfig konfiguriert die Model-Auswahl
type SelectionConfig struct {
	Enabled      bool   `json:"enabled"`
	DefaultModel string `json:"default_model"`
	CodeModel    string `json:"code_model"`
	FastModel    string `json:"fast_model"`
	VisionModel  string `json:"vision_model"`
}

// DefaultSelectionConfig gibt die Standard-Konfiguration zurueck
func DefaultSelectionConfig() SelectionConfig {
	return SelectionConfig{
		Enabled:      true,
		DefaultModel: "qwen2.5:7b",
		CodeModel:    "qwen2.5-coder:7b",
		FastModel:    "llama3.2:3b",
		VisionModel:  "llava:13b",
	}
}

// SelectionService waehlt das beste Modell basierend auf Prompt
type SelectionService struct {
	config SelectionConfig
	mu     sync.RWMutex

	// Compiled Regex Patterns
	codeBlockPattern   *regexp.Regexp
	technicalPattern   *regexp.Regexp
}

// Code-relevante Keywords
var codeKeywords = []string{
	"code", "function", "class", "method", "variable", "bug", "error",
	"implement", "refactor", "debug", "algorithm", "programming",
	"java", "javascript", "python", "typescript", "vue", "react",
	"spring", "boot", "api", "database", "sql", "git", "maven",
	"npm", "package", "import", "export", "const", "let", "var",
	"golang", "go", "rust", "c++", "kotlin", "swift",
	"funktion", "klasse", "methode", "fehler", "implementieren",
}

// Deutsche einfache Frage-Patterns
var simpleQuestionPatterns = []string{
	"was ist", "was bedeutet", "erklaere", "erkläre", "definiere",
	"what is", "what does", "explain", "define",
	"wie funktioniert", "how does", "how do",
}

// NewSelectionService erstellt einen neuen Selection Service
func NewSelectionService(config SelectionConfig) *SelectionService {
	return &SelectionService{
		config:           config,
		codeBlockPattern: regexp.MustCompile("```|`[^`]+`|\\{|\\}|\\(|\\)|;|=>|::|->"),
		technicalPattern: regexp.MustCompile(`(?i)\b(HTTP|REST|API|JSON|XML|HTML|CSS|SQL|NoSQL|YAML|JWT|OAuth)\b`),
	}
}

// SelectModel waehlt das beste Modell basierend auf dem Prompt
func (s *SelectionService) SelectModel(prompt string) (string, TaskType) {
	s.mu.RLock()
	config := s.config
	s.mu.RUnlock()

	if !config.Enabled {
		return config.DefaultModel, TaskComplex
	}

	if prompt == "" {
		return config.DefaultModel, TaskComplex
	}

	lowerPrompt := strings.ToLower(prompt)

	// Code-Aufgabe?
	if s.isCodeRelated(lowerPrompt, prompt) {
		return config.CodeModel, TaskCode
	}

	// Einfache Frage?
	if s.isSimpleQuestion(lowerPrompt) {
		return config.FastModel, TaskSimpleQA
	}

	// Standard
	return config.DefaultModel, TaskComplex
}

// SelectModelForVision waehlt Vision-Modell
func (s *SelectionService) SelectModelForVision() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.VisionModel
}

// isCodeRelated prueft ob der Prompt Code-bezogen ist
func (s *SelectionService) isCodeRelated(lowerPrompt, originalPrompt string) bool {
	// Zaehle Code-Keywords
	keywordCount := 0
	for _, keyword := range codeKeywords {
		if strings.Contains(lowerPrompt, keyword) {
			keywordCount++
		}
	}

	if keywordCount >= 2 {
		return true
	}

	// Code-Bloecke oder Syntax?
	if s.codeBlockPattern.MatchString(originalPrompt) {
		return true
	}

	// Technische Begriffe?
	if s.technicalPattern.MatchString(originalPrompt) {
		return true
	}

	return false
}

// isSimpleQuestion prueft ob es eine einfache Frage ist
func (s *SelectionService) isSimpleQuestion(lowerPrompt string) bool {
	// Kurze Prompts sind wahrscheinlich einfache Fragen
	if len(lowerPrompt) > 150 {
		return false
	}

	for _, pattern := range simpleQuestionPatterns {
		if strings.Contains(lowerPrompt, pattern) {
			return true
		}
	}

	// Kurze Fragen mit Fragezeichen
	if len(lowerPrompt) < 100 && strings.Contains(lowerPrompt, "?") {
		return true
	}

	return false
}

// UpdateConfig aktualisiert die Konfiguration
func (s *SelectionService) UpdateConfig(config SelectionConfig) {
	s.mu.Lock()
	s.config = config
	s.mu.Unlock()
}

// GetConfig gibt die aktuelle Konfiguration zurueck
func (s *SelectionService) GetConfig() SelectionConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// SetEnabled aktiviert/deaktiviert Smart Selection
func (s *SelectionService) SetEnabled(enabled bool) {
	s.mu.Lock()
	s.config.Enabled = enabled
	s.mu.Unlock()
}

// SetModel setzt ein bestimmtes Modell
func (s *SelectionService) SetModel(modelType string, modelName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch modelType {
	case "default":
		s.config.DefaultModel = modelName
	case "code":
		s.config.CodeModel = modelName
	case "fast":
		s.config.FastModel = modelName
	case "vision":
		s.config.VisionModel = modelName
	}
}
