package settings

import "log"

// =============================================================================
// MATE SETTINGS - Modell-Zuordnungen für Fleet Mates
// =============================================================================

// GetMateModelSettings gibt alle Mate-Modell-Zuordnungen zurück
func (s *Service) GetMateModelSettings() MateModelSettings {
	return MateModelSettings{
		EmailModel:       s.GetString(KeyMateEmailModel, ""),
		DocumentModel:    s.GetString(KeyMateDocumentModel, ""),
		LogAnalysisModel: s.GetString(KeyMateLogAnalysisModel, ""),
		CoderModel:       s.GetString(KeyMateCoderModel, ""),
	}
}

// SaveMateModelSettings speichert alle Mate-Modell-Zuordnungen
func (s *Service) SaveMateModelSettings(settings MateModelSettings) error {
	if err := s.SetString(KeyMateEmailModel, settings.EmailModel); err != nil {
		return err
	}
	if err := s.SetString(KeyMateDocumentModel, settings.DocumentModel); err != nil {
		return err
	}
	if err := s.SetString(KeyMateLogAnalysisModel, settings.LogAnalysisModel); err != nil {
		return err
	}
	if err := s.SetString(KeyMateCoderModel, settings.CoderModel); err != nil {
		return err
	}
	log.Printf("Mate-Modell-Einstellungen gespeichert")
	return nil
}

// --- Email Model (Thunderbird Mate) ---

// GetEmailModel gibt das Email-Modell zurück
func (s *Service) GetEmailModel() string {
	return s.GetString(KeyMateEmailModel, "")
}

// SaveEmailModel speichert das Email-Modell
func (s *Service) SaveEmailModel(model string) error {
	log.Printf("Speichere Email-Modell: %s", model)
	return s.SetString(KeyMateEmailModel, model)
}

// --- Document Model (Writer Mate) ---

// GetDocumentModel gibt das Document-Modell zurück
func (s *Service) GetDocumentModel() string {
	return s.GetString(KeyMateDocumentModel, "")
}

// SaveDocumentModel speichert das Document-Modell
func (s *Service) SaveDocumentModel(model string) error {
	log.Printf("Speichere Document-Modell: %s", model)
	return s.SetString(KeyMateDocumentModel, model)
}

// --- Log Analysis Model (OS Mate) ---

// GetLogAnalysisModel gibt das Log-Analyse-Modell zurück
func (s *Service) GetLogAnalysisModel() string {
	return s.GetString(KeyMateLogAnalysisModel, "")
}

// SaveLogAnalysisModel speichert das Log-Analyse-Modell
func (s *Service) SaveLogAnalysisModel(model string) error {
	log.Printf("Speichere Log-Analyse-Modell: %s", model)
	return s.SetString(KeyMateLogAnalysisModel, model)
}

// --- Coder Model (FleetCoder) ---

// GetCoderModel gibt das Coder-Modell zurück (für FleetCoder)
func (s *Service) GetCoderModel() string {
	return s.GetString(KeyMateCoderModel, "")
}

// SaveCoderModel speichert das Coder-Modell
func (s *Service) SaveCoderModel(model string) error {
	log.Printf("Speichere Coder-Modell: %s", model)
	return s.SetString(KeyMateCoderModel, model)
}
