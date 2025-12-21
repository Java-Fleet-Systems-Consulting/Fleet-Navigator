package custommodel

import (
	"fmt"
	"log"
	"strings"
)

// Service verwaltet Custom-Model-Operationen
type Service struct {
	repo *Repository
}

// NewService erstellt einen neuen Service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetAll gibt alle Custom Models zurück
func (s *Service) GetAll() ([]CustomModel, error) {
	return s.repo.GetAll()
}

// GetByID gibt ein Custom Model nach ID zurück
func (s *Service) GetByID(id int64) (*CustomModel, error) {
	return s.repo.GetByID(id)
}

// GetByName gibt ein Custom Model nach Name zurück
func (s *Service) GetByName(name string) (*CustomModel, error) {
	return s.repo.GetByName(name)
}

// GetAncestry gibt die Versionskette zurück
func (s *Service) GetAncestry(id int64) ([]CustomModel, error) {
	return s.repo.GetAncestry(id)
}

// Create erstellt ein neues Custom Model
func (s *Service) Create(req CreateRequest) (*CustomModel, error) {
	// Name validieren
	if req.Name == "" {
		return nil, fmt.Errorf("Name ist erforderlich")
	}
	if req.BaseModel == "" {
		return nil, fmt.Errorf("BaseModel ist erforderlich")
	}

	// Prüfen ob Name bereits existiert
	existing, err := s.repo.GetByName(req.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("Model mit Name '%s' existiert bereits", req.Name)
	}

	// Model-Name mit Tag versehen, falls nicht vorhanden
	modelName := req.Name
	if !strings.Contains(modelName, ":") {
		modelName = modelName + ":latest"
	}

	// Modelfile generieren
	modelfile := s.GenerateModelfile(req.BaseModel, req.SystemPrompt,
		req.Temperature, req.TopP, req.TopK, req.RepeatPenalty, req.NumPredict, req.NumCtx)

	model := &CustomModel{
		Name:          modelName,
		BaseModel:     req.BaseModel,
		SystemPrompt:  req.SystemPrompt,
		Description:   req.Description,
		Temperature:   req.Temperature,
		TopP:          req.TopP,
		TopK:          req.TopK,
		RepeatPenalty: req.RepeatPenalty,
		NumPredict:    req.NumPredict,
		NumCtx:        req.NumCtx,
		Version:       1,
		Modelfile:     modelfile,
	}

	if err := s.repo.Create(model); err != nil {
		return nil, err
	}

	log.Printf("Custom Model erstellt: %s (ID: %d)", model.Name, model.ID)
	return model, nil
}

// Update aktualisiert ein Custom Model (erstellt neue Version)
func (s *Service) Update(id int64, req UpdateRequest) (*CustomModel, error) {
	// Original laden
	original, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if original == nil {
		return nil, fmt.Errorf("Custom Model nicht gefunden: %d", id)
	}

	// Neuen Namen generieren (Version erhöhen)
	baseName := strings.Split(original.Name, ":")[0]
	newVersion := original.Version + 1
	newName := fmt.Sprintf("%s:v%d", baseName, newVersion)

	// Werte übernehmen oder aktualisieren
	systemPrompt := original.SystemPrompt
	if req.SystemPrompt != nil {
		systemPrompt = *req.SystemPrompt
	}

	description := original.Description
	if req.Description != nil {
		description = *req.Description
	}

	temperature := original.Temperature
	if req.Temperature != nil {
		temperature = req.Temperature
	}

	topP := original.TopP
	if req.TopP != nil {
		topP = req.TopP
	}

	topK := original.TopK
	if req.TopK != nil {
		topK = req.TopK
	}

	repeatPenalty := original.RepeatPenalty
	if req.RepeatPenalty != nil {
		repeatPenalty = req.RepeatPenalty
	}

	numPredict := original.NumPredict
	if req.NumPredict != nil {
		numPredict = req.NumPredict
	}

	numCtx := original.NumCtx
	if req.NumCtx != nil {
		numCtx = req.NumCtx
	}

	// Neues Modelfile generieren
	modelfile := s.GenerateModelfile(original.BaseModel, systemPrompt,
		temperature, topP, topK, repeatPenalty, numPredict, numCtx)

	// Neue Version erstellen
	newModel := &CustomModel{
		Name:          newName,
		BaseModel:     original.BaseModel, // Base bleibt gleich
		SystemPrompt:  systemPrompt,
		Description:   description,
		Temperature:   temperature,
		TopP:          topP,
		TopK:          topK,
		RepeatPenalty: repeatPenalty,
		NumPredict:    numPredict,
		NumCtx:        numCtx,
		ParentModelID: &original.ID,
		Version:       newVersion,
		Modelfile:     modelfile,
	}

	if err := s.repo.Create(newModel); err != nil {
		return nil, err
	}

	log.Printf("Custom Model aktualisiert: %s -> %s (Version: %d)", original.Name, newName, newVersion)
	return newModel, nil
}

// Delete löscht ein Custom Model
func (s *Service) Delete(id int64) error {
	model, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if model == nil {
		return fmt.Errorf("Custom Model nicht gefunden: %d", id)
	}

	if err := s.repo.Delete(id); err != nil {
		return err
	}

	log.Printf("Custom Model gelöscht: %s (ID: %d)", model.Name, id)
	return nil
}

// UpdateOllamaInfo aktualisiert Ollama-spezifische Infos
func (s *Service) UpdateOllamaInfo(id int64, digest string, size *int64) error {
	model, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if model == nil {
		return fmt.Errorf("Custom Model nicht gefunden: %d", id)
	}

	model.OllamaDigest = digest
	model.Size = size

	return s.repo.Update(model)
}

// GenerateModelfile erzeugt ein Ollama Modelfile
func (s *Service) GenerateModelfile(baseModel, systemPrompt string,
	temperature, topP *float64, topK *int, repeatPenalty *float64,
	numPredict, numCtx *int) string {

	var sb strings.Builder

	// FROM directive
	sb.WriteString(fmt.Sprintf("FROM %s\n\n", baseModel))

	// SYSTEM directive
	if systemPrompt != "" {
		// Multi-line system prompt in triple quotes
		sb.WriteString(fmt.Sprintf("SYSTEM \"\"\"\n%s\n\"\"\"\n\n", systemPrompt))
	}

	// PARAMETER directives
	if temperature != nil {
		sb.WriteString(fmt.Sprintf("PARAMETER temperature %.2f\n", *temperature))
	}

	if topP != nil {
		sb.WriteString(fmt.Sprintf("PARAMETER top_p %.2f\n", *topP))
	}

	if topK != nil {
		sb.WriteString(fmt.Sprintf("PARAMETER top_k %d\n", *topK))
	}

	if repeatPenalty != nil {
		sb.WriteString(fmt.Sprintf("PARAMETER repeat_penalty %.2f\n", *repeatPenalty))
	}

	if numPredict != nil {
		sb.WriteString(fmt.Sprintf("PARAMETER num_predict %d\n", *numPredict))
	}

	if numCtx != nil {
		sb.WriteString(fmt.Sprintf("PARAMETER num_ctx %d\n", *numCtx))
	}

	return sb.String()
}

// Count gibt die Anzahl der Custom Models zurück
func (s *Service) Count() (int, error) {
	return s.repo.Count()
}

// ========== GGUF Model Config Methods ==========

// GetAllGgufConfigs gibt alle GGUF-Konfigurationen zurück
func (s *Service) GetAllGgufConfigs() ([]GgufModelConfig, error) {
	return s.repo.GetAllGgufConfigs()
}

// GetGgufConfigByID gibt eine GGUF-Konfiguration nach ID zurück
func (s *Service) GetGgufConfigByID(id int64) (*GgufModelConfig, error) {
	return s.repo.GetGgufConfigByID(id)
}

// CreateGgufConfig erstellt eine neue GGUF-Konfiguration
func (s *Service) CreateGgufConfig(req GgufConfigCreateRequest) (*GgufModelConfig, error) {
	// Validierung
	if req.Name == "" {
		return nil, fmt.Errorf("Name ist erforderlich")
	}
	if req.BaseModel == "" {
		return nil, fmt.Errorf("BaseModel ist erforderlich")
	}

	// Prüfen ob Name bereits existiert
	existing, err := s.repo.GetGgufConfigByName(req.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("GGUF-Konfiguration mit Name '%s' existiert bereits", req.Name)
	}

	// Defaults setzen
	if req.Temperature == 0 {
		req.Temperature = 0.8
	}
	if req.TopP == 0 {
		req.TopP = 0.9
	}
	if req.TopK == 0 {
		req.TopK = 40
	}
	if req.RepeatPenalty == 0 {
		req.RepeatPenalty = 1.1
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = 2048
	}
	if req.ContextSize == 0 {
		req.ContextSize = 8192
	}
	if req.GpuLayers == 0 {
		req.GpuLayers = -1 // Alle Layers auf GPU
	}

	config := &GgufModelConfig{
		Name:          req.Name,
		BaseModel:     req.BaseModel,
		Description:   req.Description,
		SystemPrompt:  req.SystemPrompt,
		Temperature:   req.Temperature,
		TopP:          req.TopP,
		TopK:          req.TopK,
		RepeatPenalty: req.RepeatPenalty,
		MaxTokens:     req.MaxTokens,
		ContextSize:   req.ContextSize,
		GpuLayers:     req.GpuLayers,
	}

	if err := s.repo.CreateGgufConfig(config); err != nil {
		return nil, err
	}

	log.Printf("GGUF-Konfiguration erstellt: %s (ID: %d)", config.Name, config.ID)
	return config, nil
}

// UpdateGgufConfig aktualisiert eine GGUF-Konfiguration
func (s *Service) UpdateGgufConfig(id int64, req GgufConfigUpdateRequest) (*GgufModelConfig, error) {
	config, err := s.repo.GetGgufConfigByID(id)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, fmt.Errorf("GGUF-Konfiguration nicht gefunden: %d", id)
	}

	// Felder aktualisieren wenn gesetzt
	if req.Name != nil {
		config.Name = *req.Name
	}
	if req.Description != nil {
		config.Description = *req.Description
	}
	if req.SystemPrompt != nil {
		config.SystemPrompt = *req.SystemPrompt
	}
	if req.Temperature != nil {
		config.Temperature = *req.Temperature
	}
	if req.TopP != nil {
		config.TopP = *req.TopP
	}
	if req.TopK != nil {
		config.TopK = *req.TopK
	}
	if req.RepeatPenalty != nil {
		config.RepeatPenalty = *req.RepeatPenalty
	}
	if req.MaxTokens != nil {
		config.MaxTokens = *req.MaxTokens
	}
	if req.ContextSize != nil {
		config.ContextSize = *req.ContextSize
	}
	if req.GpuLayers != nil {
		config.GpuLayers = *req.GpuLayers
	}

	if err := s.repo.UpdateGgufConfig(config); err != nil {
		return nil, err
	}

	log.Printf("GGUF-Konfiguration aktualisiert: %s (ID: %d)", config.Name, id)
	return config, nil
}

// DeleteGgufConfig löscht eine GGUF-Konfiguration
func (s *Service) DeleteGgufConfig(id int64) error {
	config, err := s.repo.GetGgufConfigByID(id)
	if err != nil {
		return err
	}
	if config == nil {
		return fmt.Errorf("GGUF-Konfiguration nicht gefunden: %d", id)
	}

	if err := s.repo.DeleteGgufConfig(id); err != nil {
		return err
	}

	log.Printf("GGUF-Konfiguration gelöscht: %s (ID: %d)", config.Name, id)
	return nil
}
