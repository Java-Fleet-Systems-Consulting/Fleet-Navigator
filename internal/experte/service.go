package experte

import (
	"fmt"
	"log"
	"sync"
)

// Service ist der Experten-Service
type Service struct {
	repo *Repository
	mu   sync.RWMutex

	// Cache für schnellen Zugriff
	activeExperts map[int64]*Expert

	// Default-Modell für neue Experten (wird beim Start gesetzt)
	defaultModel string
}

// NewService erstellt einen neuen Experten-Service
func NewService(dataDir string) (*Service, error) {
	repo, err := NewRepository(dataDir)
	if err != nil {
		return nil, err
	}

	s := &Service{
		repo:          repo,
		activeExperts: make(map[int64]*Expert),
		defaultModel:  "qwen2.5:7b", // Wird später mit SetDefaultModel überschrieben
	}

	if err := repo.SeedDefaultExperts(); err != nil {
		log.Printf("WARNUNG: Standard-Experten konnten nicht erstellt werden: %v", err)
	}

	if err := s.refreshCache(); err != nil {
		log.Printf("WARNUNG: Experten-Cache konnte nicht geladen werden: %v", err)
	}

	return s, nil
}

// Close schließt den Service
func (s *Service) Close() error {
	return s.repo.Close()
}

// SetDefaultModel setzt das Default-Modell für neue Experten
func (s *Service) SetDefaultModel(model string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultModel = model
	log.Printf("Experten Default-Modell gesetzt: %s", model)
}

// UpdateAllExpertsModel aktualisiert das BaseModel aller Experten
// Wird nach dem Setup aufgerufen, um alle Experten auf das neue Modell zu setzen
func (s *Service) UpdateAllExpertsModel(newModel string) error {
	experts, err := s.repo.GetAllExperts(false) // Alle, nicht nur aktive
	if err != nil {
		return err
	}

	updatedCount := 0
	for _, expert := range experts {
		// Nur aktualisieren wenn es das alte Default-Modell ist
		if expert.BaseModel == "qwen2.5:7b" || expert.BaseModel == "" {
			req := UpdateExpertRequest{
				BaseModel: &newModel,
			}
			if _, err := s.UpdateExpert(expert.ID, req); err != nil {
				log.Printf("WARNUNG: Experte %s konnte nicht aktualisiert werden: %v", expert.Name, err)
			} else {
				updatedCount++
				log.Printf("Experte %s: Modell aktualisiert auf %s", expert.Name, newModel)
			}
		}
	}

	log.Printf("✅ %d Experten auf Modell %s aktualisiert", updatedCount, newModel)
	return s.refreshCache()
}

// GetDefaultModel gibt das aktuelle Default-Modell zurück
func (s *Service) GetDefaultModel() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.defaultModel == "" {
		return "qwen2.5:7b" // Fallback
	}
	return s.defaultModel
}

// refreshCache lädt alle aktiven Experten in den Cache
func (s *Service) refreshCache() error {
	experts, err := s.repo.GetAllExperts(true)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.activeExperts = make(map[int64]*Expert)
	for i := range experts {
		s.activeExperts[experts[i].ID] = &experts[i]
	}
	s.mu.Unlock()

	log.Printf("Experten-Cache geladen: %d aktive Experten", len(experts))
	return nil
}

// --- Expert CRUD ---

// CreateExpert erstellt einen neuen Experten
func (s *Service) CreateExpert(req CreateExpertRequest) (*Expert, error) {
	expert := &Expert{
		Name:           req.Name,
		Role:           req.Role,
		BasePrompt:     req.BasePrompt,
		BaseModel:      req.BaseModel,
		Avatar:         req.Avatar,
		Description:    req.Description,
		IsActive:       true,
		AutoModeSwitch: req.AutoModeSwitch,
	}

	// Setze Defaults
	if expert.BaseModel == "" {
		expert.BaseModel = s.GetDefaultModel()
	}
	if expert.Avatar == "" {
		expert.Avatar = "🤖"
	}

	if err := s.repo.CreateExpert(expert); err != nil {
		return nil, err
	}

	// Cache aktualisieren
	if expert.IsActive {
		s.mu.Lock()
		s.activeExperts[expert.ID] = expert
		s.mu.Unlock()
	}

	return expert, nil
}

// GetExpert holt einen Experten
func (s *Service) GetExpert(id int64) (*Expert, error) {
	// Zuerst im Cache suchen
	s.mu.RLock()
	if expert, ok := s.activeExperts[id]; ok {
		s.mu.RUnlock()
		return expert, nil
	}
	s.mu.RUnlock()

	// Aus DB laden
	return s.repo.GetExpert(id)
}

// GetAllExperts holt alle Experten
func (s *Service) GetAllExperts(onlyActive bool) ([]Expert, error) {
	return s.repo.GetAllExperts(onlyActive)
}

// GetActiveExperts gibt alle aktiven Experten zurück (aus Cache)
func (s *Service) GetActiveExperts() []*Expert {
	s.mu.RLock()
	defer s.mu.RUnlock()

	experts := make([]*Expert, 0, len(s.activeExperts))
	for _, e := range s.activeExperts {
		experts = append(experts, e)
	}
	return experts
}

// UpdateExpert aktualisiert einen Experten
func (s *Service) UpdateExpert(id int64, req UpdateExpertRequest) (*Expert, error) {
	if err := s.repo.UpdateExpert(id, &req); err != nil {
		return nil, err
	}

	// Neu laden und Cache aktualisieren
	expert, err := s.repo.GetExpert(id)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	if expert.IsActive {
		s.activeExperts[id] = expert
	} else {
		delete(s.activeExperts, id)
	}
	s.mu.Unlock()

	return expert, nil
}

// DeleteExpert löscht einen Experten
func (s *Service) DeleteExpert(id int64) error {
	if err := s.repo.DeleteExpert(id); err != nil {
		return err
	}

	s.mu.Lock()
	delete(s.activeExperts, id)
	s.mu.Unlock()

	return nil
}

// --- Mode CRUD ---

// AddMode fügt einem Experten einen Modus hinzu
func (s *Service) AddMode(expertID int64, req CreateModeRequest) (*ExpertMode, error) {
	mode := &ExpertMode{
		ExpertID:  expertID,
		Name:      req.Name,
		Prompt:    req.Prompt,
		Icon:      req.Icon,
		Keywords:  req.Keywords,
		IsDefault: req.IsDefault,
	}

	if mode.Icon == "" {
		mode.Icon = "💬"
	}

	if err := s.repo.CreateMode(mode); err != nil {
		return nil, err
	}

	// Cache aktualisieren
	s.mu.Lock()
	if expert, ok := s.activeExperts[expertID]; ok {
		expert.Modes = append(expert.Modes, *mode)
	}
	s.mu.Unlock()

	return mode, nil
}

// GetModes holt alle Modi eines Experten
func (s *Service) GetModes(expertID int64) ([]ExpertMode, error) {
	return s.repo.GetModesByExpert(expertID)
}

// GetMode holt einen einzelnen Modus
func (s *Service) GetMode(modeID int64) (*ExpertMode, error) {
	return s.repo.GetMode(modeID)
}

// UpdateMode aktualisiert einen Modus
func (s *Service) UpdateMode(id int64, name, prompt, icon string, keywords []string, isDefault bool, sortOrder int) error {
	if err := s.repo.UpdateMode(id, name, prompt, icon, keywords, isDefault, sortOrder); err != nil {
		return err
	}

	// Cache neu laden (einfachste Lösung)
	return s.refreshCache()
}

// DeleteMode löscht einen Modus
func (s *Service) DeleteMode(modeID int64) error {
	if err := s.repo.DeleteMode(modeID); err != nil {
		return err
	}

	return s.refreshCache()
}

// SetDefaultMode setzt den Standard-Modus
func (s *Service) SetDefaultMode(expertID, modeID int64) error {
	if err := s.repo.SetDefaultMode(expertID, modeID); err != nil {
		return err
	}

	return s.refreshCache()
}

// --- Chat Integration ---

// ChatContext enthält Informationen für einen Experten-Chat
type ChatContext struct {
	SystemPrompt   string      `json:"system_prompt"`
	Model          string      `json:"model"`
	Expert         *Expert     `json:"expert"`
	ActiveMode     *ExpertMode `json:"active_mode"`
	ModeSwitched   bool        `json:"mode_switched"`   // True wenn Modus automatisch gewechselt wurde
	SwitchMessage  string      `json:"switch_message"`  // z.B. "🚗 Roland wechselt zu Verkehrsrecht"
	PreviousMode   *ExpertMode `json:"previous_mode"`   // Vorheriger Modus (für Tracking)
	// Sampling-Parameter vom Experten
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"top_p"`
	MaxTokens   int     `json:"max_tokens"`
	NumCtx      int     `json:"num_ctx"` // Context-Größe
}

// GetPromptForChat generiert den System-Prompt für einen Chat
func (s *Service) GetPromptForChat(expertID int64, modeID *int64) (string, string, error) {
	expert, err := s.GetExpert(expertID)
	if err != nil {
		return "", "", err
	}
	if expert == nil {
		return "", "", fmt.Errorf("Experte mit ID %d nicht gefunden", expertID)
	}

	var mode *ExpertMode
	if modeID != nil {
		// Spezifischen Modus suchen
		for i := range expert.Modes {
			if expert.Modes[i].ID == *modeID {
				mode = &expert.Modes[i]
				break
			}
		}
	} else {
		// Default-Modus verwenden
		mode = expert.GetDefaultMode()
	}

	return expert.GetFullPrompt(mode), expert.BaseModel, nil
}

// GetChatContext generiert den vollständigen Chat-Kontext mit automatischer Modus-Erkennung
// currentModeID ist der aktuell aktive Modus (nil = kein Modus aktiv)
// message ist die Benutzernachricht für Keyword-Erkennung
func (s *Service) GetChatContext(expertID int64, currentModeID *int64, message string) (*ChatContext, error) {
	expert, err := s.GetExpert(expertID)
	if err != nil {
		return nil, err
	}
	if expert == nil {
		return nil, fmt.Errorf("Experte mit ID %d nicht gefunden", expertID)
	}

	// Sampling-Parameter mit Defaults falls nicht gesetzt
	temperature := expert.DefaultTemperature
	if temperature == 0 {
		temperature = 0.7
	}
	topP := expert.DefaultTopP
	if topP == 0 {
		topP = 0.9
	}
	maxTokens := expert.DefaultMaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}
	numCtx := expert.DefaultNumCtx
	if numCtx == 0 {
		numCtx = 16384 // 16K Default (65K braucht zu viel VRAM)
	}

	ctx := &ChatContext{
		Expert:       expert,
		Model:        expert.BaseModel,
		ModeSwitched: false,
		// Sampling-Parameter vom Experten
		Temperature: temperature,
		TopP:        topP,
		MaxTokens:   maxTokens,
		NumCtx:      numCtx,
	}

	// Aktuellen Modus ermitteln
	var currentMode *ExpertMode
	if currentModeID != nil {
		for i := range expert.Modes {
			if expert.Modes[i].ID == *currentModeID {
				currentMode = &expert.Modes[i]
				break
			}
		}
	}
	ctx.PreviousMode = currentMode

	// Nur Keyword-Erkennung wenn AutoModeSwitch aktiviert ist
	if expert.AutoModeSwitch {
		// Modus durch Keywords erkennen
		detectedMode := expert.DetectModeByKeywords(message)

		if detectedMode != nil {
			// Keyword-Match gefunden
			if currentMode == nil || currentMode.ID != detectedMode.ID {
				// Modus wechselt!
				ctx.ModeSwitched = true
				ctx.ActiveMode = detectedMode
				ctx.SwitchMessage = fmt.Sprintf("%s *%s wechselt zu %s*",
					detectedMode.Icon, expert.Name, detectedMode.Name)
			} else {
				// Gleicher Modus, kein Wechsel
				ctx.ActiveMode = currentMode
			}
		} else {
			// Kein Keyword gefunden - behalte aktuellen oder Default
			if currentMode != nil {
				ctx.ActiveMode = currentMode
			} else {
				ctx.ActiveMode = expert.GetDefaultMode()
			}
		}
	} else {
		// AutoModeSwitch deaktiviert - behalte aktuellen oder Default
		if currentMode != nil {
			ctx.ActiveMode = currentMode
		} else {
			ctx.ActiveMode = expert.GetDefaultMode()
		}
	}

	// System-Prompt generieren
	ctx.SystemPrompt = expert.GetFullPrompt(ctx.ActiveMode)

	return ctx, nil
}

// GetExpertSummary gibt eine Zusammenfassung für die UI
func (s *Service) GetExpertSummary() []map[string]interface{} {
	experts := s.GetActiveExperts()
	summary := make([]map[string]interface{}, 0, len(experts))

	for _, e := range experts {
		summary = append(summary, map[string]interface{}{
			"id":          e.ID,
			"name":        e.Name,
			"role":        e.Role,
			"avatar":      e.Avatar,
			"description": e.Description,
			"base_model":  e.BaseModel,
			"modes":       len(e.Modes),
		})
	}

	return summary
}
