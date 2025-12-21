package modeltemplates

import (
	"fleet-navigator/internal/llamaserver"
	"fmt"
	"log"
	"regexp"
	"sync"
)

// Service verwaltet Model-Templates und die Message-Adaption
type Service struct {
	repo      *Repository
	templates []ModelTemplate
	mu        sync.RWMutex
}

// NewService erstellt einen neuen Template-Service
func NewService(repo *Repository) *Service {
	s := &Service{
		repo: repo,
	}
	// Templates initial laden
	s.ReloadTemplates()
	return s
}

// ReloadTemplates lädt die Templates aus der Datenbank neu
func (s *Service) ReloadTemplates() error {
	templates, err := s.repo.GetActive()
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.templates = templates
	s.mu.Unlock()

	log.Printf("Model-Templates geladen: %d aktive Templates", len(templates))
	return nil
}

// GetTemplateForModel findet das passende Template für einen Modellnamen
func (s *Service) GetTemplateForModel(modelName string) *ModelTemplate {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, t := range s.templates {
		matched, err := regexp.MatchString(t.Pattern, modelName)
		if err != nil {
			log.Printf("Regex-Fehler für Pattern %s: %v", t.Pattern, err)
			continue
		}
		if matched {
			log.Printf("Template gefunden für %s: %s (Strategy: %s)",
				modelName, t.Name, t.SystemEmbedStrategy)
			return &t
		}
	}

	// Kein Template gefunden - Fallback auf Native
	log.Printf("Kein Template für %s gefunden, verwende Native-Strategie", modelName)
	return &ModelTemplate{
		Name:                "Fallback",
		SupportsSystemRole:  true,
		SystemEmbedStrategy: StrategyNative,
	}
}

// AdaptMessages passt die Messages für das angegebene Modell an
func (s *Service) AdaptMessages(modelName string, messages []llamaserver.ChatMessage) []llamaserver.ChatMessage {
	template := s.GetTemplateForModel(modelName)

	// Native Strategie = keine Änderung nötig
	if template.SystemEmbedStrategy == StrategyNative {
		return messages
	}

	// System-Prompt extrahieren
	var systemPrompt string
	var otherMessages []llamaserver.ChatMessage

	for _, msg := range messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
		} else {
			otherMessages = append(otherMessages, msg)
		}
	}

	// Kein System-Prompt = keine Änderung nötig
	if systemPrompt == "" {
		return messages
	}

	// Je nach Strategie anpassen
	switch template.SystemEmbedStrategy {
	case StrategyEmbedInUser:
		return s.embedSystemInUser(template, systemPrompt, otherMessages)
	case StrategyPrependUser:
		return s.prependSystemAsUser(template, systemPrompt, otherMessages)
	default:
		return messages
	}
}

// embedSystemInUser bettet den System-Prompt in die erste User-Nachricht ein
func (s *Service) embedSystemInUser(template *ModelTemplate, systemPrompt string, messages []llamaserver.ChatMessage) []llamaserver.ChatMessage {
	result := make([]llamaserver.ChatMessage, 0, len(messages))
	systemEmbedded := false

	for _, msg := range messages {
		if msg.Role == "user" && !systemEmbedded {
			// System-Prompt mit Prefix/Suffix einbetten
			prefix := template.SystemPrefix
			suffix := template.SystemSuffix

			if prefix == "" {
				prefix = "[SYSTEM-ANWEISUNGEN]\n"
			}
			if suffix == "" {
				suffix = "\n[ENDE]\n\nNachricht: "
			}

			enhancedContent := fmt.Sprintf("%s%s%s%s", prefix, systemPrompt, suffix, msg.Content)
			result = append(result, llamaserver.ChatMessage{
				Role:    "user",
				Content: enhancedContent,
			})
			systemEmbedded = true
		} else {
			result = append(result, msg)
		}
	}

	// Falls keine User-Nachricht gefunden wurde
	if !systemEmbedded && systemPrompt != "" {
		result = append([]llamaserver.ChatMessage{{
			Role:    "user",
			Content: template.SystemPrefix + systemPrompt + template.SystemSuffix,
		}}, result...)
	}

	log.Printf("🔄 System-Prompt eingebettet für %s (%d Zeichen)", template.Name, len(systemPrompt))
	return result
}

// prependSystemAsUser fügt den System-Prompt als erste User-Nachricht hinzu
func (s *Service) prependSystemAsUser(template *ModelTemplate, systemPrompt string, messages []llamaserver.ChatMessage) []llamaserver.ChatMessage {
	prefix := template.SystemPrefix
	suffix := template.SystemSuffix

	if prefix == "" {
		prefix = "Kontext: "
	}
	if suffix == "" {
		suffix = ""
	}

	systemMsg := llamaserver.ChatMessage{
		Role:    "user",
		Content: prefix + systemPrompt + suffix,
	}

	// Füge eine leere Assistant-Antwort hinzu, um den Kontext zu etablieren
	ackMsg := llamaserver.ChatMessage{
		Role:    "assistant",
		Content: "Verstanden, ich werde diese Anweisungen befolgen.",
	}

	result := append([]llamaserver.ChatMessage{systemMsg, ackMsg}, messages...)

	log.Printf("🔄 System-Prompt als User-Nachricht vorangestellt für %s", template.Name)
	return result
}

// GetAll gibt alle Templates zurück
func (s *Service) GetAll() ([]ModelTemplate, error) {
	return s.repo.GetAll()
}

// GetByID gibt ein Template zurück
func (s *Service) GetByID(id int64) (*ModelTemplate, error) {
	return s.repo.GetByID(id)
}

// Create erstellt ein neues Template
func (s *Service) Create(t *ModelTemplate) error {
	if err := s.repo.Create(t); err != nil {
		return err
	}
	return s.ReloadTemplates()
}

// Update aktualisiert ein Template
func (s *Service) Update(t *ModelTemplate) error {
	if err := s.repo.Update(t); err != nil {
		return err
	}
	return s.ReloadTemplates()
}

// Delete löscht ein Template
func (s *Service) Delete(id int64) error {
	if err := s.repo.Delete(id); err != nil {
		return err
	}
	return s.ReloadTemplates()
}

// SeedDefaults fügt die Standard-Templates ein
func (s *Service) SeedDefaults() error {
	if err := s.repo.SeedDefaults(); err != nil {
		return err
	}
	return s.ReloadTemplates()
}
