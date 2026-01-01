// Package llm - Model Service
// Kombiniert Provider und Registry fuer Modellverwaltung
package llm

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ModelService verwaltet Modelle und Chat-Funktionalitaet
type ModelService struct {
	providerManager *ProviderManager
	registry        *ModelRegistry
	defaultModel    string
	selectedModel   string // Aktuell vom User ausgewaehltes Modell
	systemPrompt    string
	modelsDir       string // Verzeichnis fuer lokale GGUF-Modelle
	mu              sync.RWMutex
}

// ModelServiceConfig enthaelt die Service-Konfiguration
type ModelServiceConfig struct {
	OllamaURL       string
	DefaultModel    string
	SystemPrompt    string
	ModelsDir       string // Verzeichnis fuer lokale GGUF-Modelle
	SkipOllamaCheck bool   // True wenn Ollama nicht der aktive Provider ist
}

// DefaultModelServiceConfig gibt die Standard-Konfiguration zurueck
func DefaultModelServiceConfig() ModelServiceConfig {
	return ModelServiceConfig{
		OllamaURL:    "http://localhost:11434",
		DefaultModel: "qwen2.5:7b",
		SystemPrompt: `Du bist der Fleet Navigator, ein freundlicher und kompetenter KI-Assistent fuer kleine Bueros.
Du hilfst bei alltaeglichen Bueroaufgaben wie:
- Textverarbeitung und Formatierung
- E-Mail-Kommunikation
- Terminplanung und Organisation
- Rechts- und Datenschutzfragen (allgemeine Hinweise)
- Marketing und Kommunikation

SPRACHE: Du antwortest IMMER und AUSSCHLIESSLICH auf Deutsch. Niemals auf Chinesisch, Englisch oder anderen Sprachen. Dein gesamter Output bleibt konsequent auf Deutsch.

WICHTIG - KEINE HALLUZINATIONEN:
- Erfinde NIEMALS Informationen, Fakten oder Quellen
- Wenn du etwas nicht weisst, sage ehrlich: "Das weiss ich leider nicht" oder "Dazu habe ich keine Informationen"
- Zitiere KEINE Webseiten, Buecher oder Quellen, die du nicht tatsaechlich kennst
- Unterscheide klar zwischen Fakten und deinen Vermutungen
- Bei Unsicherheit: Lieber zugeben als raten

Halte deine Antworten praezise und praxisnah.`,
	}
}

// NewModelService erstellt einen neuen Model Service
// HINWEIS: Ollama-Support wurde entfernt - nur noch llama-server wird verwendet
func NewModelService(config ModelServiceConfig) *ModelService {
	// Provider Manager erstellen (ohne aktiven Provider - llama-server wird separat verwaltet)
	pm := NewProviderManager()

	// Registry erstellen
	registry := NewModelRegistry()

	service := &ModelService{
		providerManager: pm,
		registry:        registry,
		defaultModel:    config.DefaultModel,
		selectedModel:   config.DefaultModel,
		systemPrompt:    config.SystemPrompt,
		modelsDir:       config.ModelsDir,
	}

	log.Printf("ModelService initialisiert (llama-server Modus)")
	return service
}

// syncInstalledModels gleicht installierte Modelle mit der Registry ab
func (s *ModelService) syncInstalledModels() {
	provider, ok := s.providerManager.GetActiveProvider()
	if !ok {
		return
	}

	models, err := provider.GetAvailableModels()
	if err != nil {
		log.Printf("Fehler beim Laden der installierten Modelle: %v", err)
		return
	}

	log.Printf("Gefundene Ollama-Modelle: %d", len(models))
	for _, m := range models {
		// Pruefen ob Modell in Registry bekannt ist
		entry := s.registry.FindByOllamaName(m.Name)
		if entry != nil {
			log.Printf("  - %s (%s) - bekannt als: %s", m.Name, m.SizeHuman, entry.DisplayName)
		} else {
			log.Printf("  - %s (%s) - nicht in Registry", m.Name, m.SizeHuman)
		}
	}
}

// GetInstalledModels gibt alle installierten Modelle zurueck
func (s *ModelService) GetInstalledModels() ([]ModelWithMetadata, error) {
	provider, ok := s.providerManager.GetActiveProvider()
	if !ok {
		return nil, fmt.Errorf("kein aktiver Provider")
	}

	models, err := provider.GetAvailableModels()
	if err != nil {
		return nil, err
	}

	result := make([]ModelWithMetadata, len(models))
	for i, m := range models {
		// Metadata aus Registry holen
		entry := s.registry.FindByOllamaName(m.Name)

		result[i] = ModelWithMetadata{
			ModelInfo:     m,
			RegistryEntry: entry,
			IsDefault:     m.Name == s.defaultModel,
			IsSelected:    m.Name == s.selectedModel,
		}
	}

	return result, nil
}

// ModelWithMetadata kombiniert ModelInfo mit Registry-Metadaten
type ModelWithMetadata struct {
	ModelInfo
	RegistryEntry *ModelRegistryEntry `json:"registry_entry,omitempty"`
	IsDefault     bool                `json:"is_default"`
	IsSelected    bool                `json:"is_selected"`
}

// GetAvailableModelsFromRegistry gibt alle Modelle aus der Registry zurueck
func (s *ModelService) GetAvailableModelsFromRegistry() []ModelRegistryEntry {
	return s.registry.GetAllModels()
}

// GetFeaturedModels gibt Featured-Modelle zurueck
func (s *ModelService) GetFeaturedModels() []ModelRegistryEntry {
	return s.registry.GetFeaturedModels()
}

// GetModelsByCategory gibt Modelle einer Kategorie zurueck
func (s *ModelService) GetModelsByCategory(category ModelCategory) []ModelRegistryEntry {
	return s.registry.GetByCategory(category)
}

// SetSelectedModel setzt das ausgewaehlte Modell
func (s *ModelService) SetSelectedModel(model string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.selectedModel = model
	log.Printf("Ausgewaehltes Modell: %s", model)
}

// GetSelectedModel gibt das ausgewaehlte Modell zurueck
func (s *ModelService) GetSelectedModel() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.selectedModel
}

// SetDefaultModel setzt das Standard-Modell
func (s *ModelService) SetDefaultModel(model string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultModel = model
	log.Printf("Standard-Modell: %s", model)
}

// GetDefaultModel gibt das Standard-Modell zurueck
func (s *ModelService) GetDefaultModel() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.defaultModel
}

// Chat fuehrt einen Chat mit dem ausgewaehlten Modell durch
func (s *ModelService) Chat(ctx context.Context, message string, onChunk func(chunk string, done bool)) error {
	return s.ChatWithModel(ctx, s.GetSelectedModel(), message, onChunk)
}

// ChatWithModel fuehrt einen Chat mit einem bestimmten Modell durch
func (s *ModelService) ChatWithModel(ctx context.Context, model, message string, onChunk func(chunk string, done bool)) error {
	provider, ok := s.providerManager.GetActiveProvider()
	if !ok {
		return fmt.Errorf("kein aktiver Provider")
	}

	messages := []ChatMessage{
		{Role: "system", Content: s.systemPrompt},
		{Role: "user", Content: message},
	}

	requestID := fmt.Sprintf("chat-%d", ctx.Value("request_id"))
	return provider.ChatWithMessages(ctx, model, messages, requestID, onChunk, nil)
}

// ChatWithHistory fuehrt einen Chat mit Verlauf durch
func (s *ModelService) ChatWithHistory(ctx context.Context, model string, history []ChatMessage,
	onChunk func(chunk string, done bool), options *ChatOptions) error {

	provider, ok := s.providerManager.GetActiveProvider()
	if !ok {
		return fmt.Errorf("kein aktiver Provider")
	}

	// System-Prompt hinzufuegen falls nicht vorhanden
	messages := make([]ChatMessage, 0, len(history)+1)
	if len(history) == 0 || history[0].Role != "system" {
		messages = append(messages, ChatMessage{Role: "system", Content: s.systemPrompt})
	}
	messages = append(messages, history...)

	requestID := fmt.Sprintf("chat-history-%d", ctx.Value("request_id"))
	return provider.ChatWithMessages(ctx, model, messages, requestID, onChunk, options)
}

// PullModel laedt ein Modell herunter
func (s *ModelService) PullModel(modelName string, onProgress func(progress string)) error {
	provider, ok := s.providerManager.GetActiveProvider()
	if !ok {
		return fmt.Errorf("kein aktiver Provider")
	}

	log.Printf("Lade Modell: %s", modelName)
	return provider.PullModel(modelName, onProgress)
}

// DeleteModel loescht ein Modell
// Funktioniert auch ohne aktiven Provider - loescht GGUF-Dateien direkt
func (s *ModelService) DeleteModel(modelName string) error {
	log.Printf("Loesche Modell: %s", modelName)

	// Zuerst versuchen, die Datei direkt zu loeschen (fuer lokale GGUF-Modelle)
	if s.modelsDir != "" {
		// Suche in verschiedenen Verzeichnissen
		searchDirs := []string{
			s.modelsDir,
			filepath.Join(s.modelsDir, "library"),
			filepath.Join(s.modelsDir, "vision"),
			filepath.Join(s.modelsDir, "custom"),
		}

		for _, dir := range searchDirs {
			modelPath := filepath.Join(dir, modelName)
			if _, err := os.Stat(modelPath); err == nil {
				// Datei gefunden - loeschen
				if err := os.Remove(modelPath); err != nil {
					return fmt.Errorf("Modell konnte nicht geloescht werden: %w", err)
				}
				log.Printf("Modell geloescht: %s", modelPath)
				return nil
			}
		}
	}

	// Falls nicht als Datei gefunden, versuche es ueber den Provider
	provider, ok := s.providerManager.GetActiveProvider()
	if ok {
		return provider.DeleteModel(modelName)
	}

	return fmt.Errorf("Modell nicht gefunden: %s", modelName)
}

// GetModelDetails gibt Details zu einem Modell zurueck
func (s *ModelService) GetModelDetails(modelName string) (map[string]interface{}, error) {
	provider, ok := s.providerManager.GetActiveProvider()
	if !ok {
		return nil, fmt.Errorf("kein aktiver Provider")
	}

	return provider.GetModelDetails(modelName)
}

// IsProviderAvailable prueft ob ein Provider verfuegbar ist
func (s *ModelService) IsProviderAvailable() bool {
	provider, ok := s.providerManager.GetActiveProvider()
	if !ok {
		return false
	}
	return provider.IsAvailable()
}

// GetProviderName gibt den Namen des aktiven Providers zurueck
func (s *ModelService) GetProviderName() string {
	provider, ok := s.providerManager.GetActiveProvider()
	if !ok {
		return "none"
	}
	return provider.GetProviderName()
}

// GetRegistry gibt die Model Registry zurueck
func (s *ModelService) GetRegistry() *ModelRegistry {
	return s.registry
}

// GetProviderManager gibt den Provider Manager zurueck
func (s *ModelService) GetProviderManager() *ProviderManager {
	return s.providerManager
}

// SetSystemPrompt setzt den System-Prompt
func (s *ModelService) SetSystemPrompt(prompt string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.systemPrompt = prompt
}

// GetSystemPrompt gibt den System-Prompt zurueck
func (s *ModelService) GetSystemPrompt() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.systemPrompt
}

// FindModelInRegistry sucht ein Modell in der Registry
func (s *ModelService) FindModelInRegistry(name string) *ModelRegistryEntry {
	// Erst nach ID suchen
	if entry := s.registry.FindByID(name); entry != nil {
		return entry
	}
	// Dann nach Ollama-Name
	if entry := s.registry.FindByOllamaName(name); entry != nil {
		return entry
	}
	// Dann nach Dateiname (GGUF)
	if entry := s.registry.FindByFilename(name); entry != nil {
		return entry
	}
	// Fuzzy-Suche nach Display-Name (z.B. "meta llama 3.1 8b")
	nameLower := strings.ToLower(name)
	for _, m := range s.registry.GetAllModels() {
		displayLower := strings.ToLower(m.DisplayName)
		if strings.Contains(displayLower, nameLower) || strings.Contains(nameLower, strings.ToLower(m.ID)) {
			entryCopy := m // Kopie für Pointer
			return &entryCopy
		}
	}
	return nil
}

// SearchModels sucht Modelle in der Registry
func (s *ModelService) SearchModels(query string) []ModelRegistryEntry {
	return s.registry.Search(query)
}

// GetOllamaNameForModel konvertiert einen beliebigen Modellnamen in den Ollama-Namen
func (s *ModelService) GetOllamaNameForModel(name string) string {
	// Wenn es schon ein Ollama-Name ist, zurueckgeben
	if strings.Contains(name, ":") {
		return name
	}

	// In Registry suchen
	entry := s.FindModelInRegistry(name)
	if entry != nil && entry.OllamaName != "" {
		return entry.OllamaName
	}

	// Fallback: Name unveraendert zurueckgeben
	return name
}
