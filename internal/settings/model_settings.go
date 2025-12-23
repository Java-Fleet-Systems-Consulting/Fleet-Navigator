package settings

import "log"

// =============================================================================
// MODEL SETTINGS - LLM/Modell-bezogene Einstellungen
// =============================================================================

// --- Default Model ---

// GetDefaultModel gibt das Standard-Modell zurück
func (s *Service) GetDefaultModel() string {
	return s.GetString(KeyDefaultModel, "qwen2.5:7b")
}

// SaveDefaultModel speichert das Standard-Modell
func (s *Service) SaveDefaultModel(model string) error {
	log.Printf("Speichere Default-Modell: %s", model)
	return s.SetString(KeyDefaultModel, model)
}

// --- Selected Model ---

// GetSelectedModel gibt das aktuell ausgewählte Modell zurück
func (s *Service) GetSelectedModel() string {
	return s.GetString(KeySelectedModel, "")
}

// SaveSelectedModel speichert das ausgewählte Modell
func (s *Service) SaveSelectedModel(model string) error {
	log.Printf("Speichere ausgewähltes Modell: %s", model)
	return s.SetString(KeySelectedModel, model)
}

// --- LLM Provider ---

// GetActiveProvider gibt den aktiven LLM-Provider zurück
func (s *Service) GetActiveProvider() string {
	return s.GetString(KeyActiveProvider, "llama-server")
}

// SaveActiveProvider speichert den aktiven LLM-Provider
func (s *Service) SaveActiveProvider(provider string) error {
	log.Printf("Speichere aktiven Provider: %s", provider)
	return s.SetString(KeyActiveProvider, provider)
}

// --- Sampling Parameters ---

// GetSamplingParams gibt die Sampling-Parameter zurück
func (s *Service) GetSamplingParams() SamplingParams {
	defaults := DefaultSamplingParams()
	return SamplingParams{
		Temperature:   s.GetFloat64(KeySamplingTemperature, defaults.Temperature),
		TopP:          s.GetFloat64(KeySamplingTopP, defaults.TopP),
		TopK:          s.GetInt(KeySamplingTopK, defaults.TopK),
		MaxTokens:     s.GetInt(KeySamplingMaxTokens, defaults.MaxTokens),
		RepeatPenalty: s.GetFloat64(KeySamplingRepeatPenalty, defaults.RepeatPenalty),
	}
}

// SaveSamplingParams speichert die Sampling-Parameter
func (s *Service) SaveSamplingParams(params SamplingParams) error {
	if err := s.SetFloat64(KeySamplingTemperature, params.Temperature); err != nil {
		return err
	}
	if err := s.SetFloat64(KeySamplingTopP, params.TopP); err != nil {
		return err
	}
	if err := s.SetInt(KeySamplingTopK, params.TopK); err != nil {
		return err
	}
	if err := s.SetInt(KeySamplingMaxTokens, params.MaxTokens); err != nil {
		return err
	}
	if err := s.SetFloat64(KeySamplingRepeatPenalty, params.RepeatPenalty); err != nil {
		return err
	}
	log.Printf("Sampling-Parameter gespeichert: temp=%.2f, topP=%.2f, maxTokens=%d",
		params.Temperature, params.TopP, params.MaxTokens)
	return nil
}
