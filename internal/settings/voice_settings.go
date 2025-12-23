package settings

import "log"

// =============================================================================
// VOICE SETTINGS - Spracheingabe (STT) und Sprachausgabe (TTS)
// =============================================================================

// GetVoiceSettings holt die Voice-Konfiguration aus der DB
func (s *Service) GetVoiceSettings() VoiceSettings {
	defaults := DefaultVoiceSettings()
	return VoiceSettings{
		WhisperModel: s.GetString(KeyVoiceWhisperModel, defaults.WhisperModel),
		PiperVoice:   s.GetString(KeyVoicePiperVoice, defaults.PiperVoice),
		Language:     s.GetString(KeyVoiceLanguage, defaults.Language),
		TTSEnabled:   s.GetBool(KeyVoiceTTSEnabled, defaults.TTSEnabled),
	}
}

// SaveVoiceSettings speichert die Voice-Konfiguration in der DB
func (s *Service) SaveVoiceSettings(settings VoiceSettings) error {
	if err := s.SetString(KeyVoiceWhisperModel, settings.WhisperModel); err != nil {
		return err
	}
	if err := s.SetString(KeyVoicePiperVoice, settings.PiperVoice); err != nil {
		return err
	}
	if err := s.SetString(KeyVoiceLanguage, settings.Language); err != nil {
		return err
	}
	if err := s.SetBool(KeyVoiceTTSEnabled, settings.TTSEnabled); err != nil {
		return err
	}
	log.Printf("Voice-Settings gespeichert: whisper=%s, piper=%s, lang=%s, tts=%v",
		settings.WhisperModel, settings.PiperVoice, settings.Language, settings.TTSEnabled)
	return nil
}

// --- Individual Whisper Settings ---

// GetWhisperModel holt das aktuelle Whisper-Modell
func (s *Service) GetWhisperModel() string {
	return s.GetString(KeyVoiceWhisperModel, "base")
}

// SaveWhisperModel speichert nur das Whisper-Modell
func (s *Service) SaveWhisperModel(model string) error {
	if err := s.SetString(KeyVoiceWhisperModel, model); err != nil {
		return err
	}
	log.Printf("Whisper-Modell gespeichert: %s", model)
	return nil
}

// --- Individual Piper Settings ---

// GetPiperVoice holt die aktuelle Piper-Stimme
func (s *Service) GetPiperVoice() string {
	return s.GetString(KeyVoicePiperVoice, "de_DE-kerstin-low")
}

// SavePiperVoice speichert die Piper-Stimme
func (s *Service) SavePiperVoice(voice string) error {
	return s.SetString(KeyVoicePiperVoice, voice)
}

// --- TTS Enabled ---

// GetTTSEnabled gibt zurück ob TTS aktiviert ist
func (s *Service) GetTTSEnabled() bool {
	return s.GetBool(KeyVoiceTTSEnabled, false)
}

// SaveTTSEnabled speichert die TTS-Einstellung
func (s *Service) SaveTTSEnabled(enabled bool) error {
	return s.SetBool(KeyVoiceTTSEnabled, enabled)
}

// --- Voice Language ---

// GetVoiceLanguage holt die Voice-Sprache
func (s *Service) GetVoiceLanguage() string {
	return s.GetString(KeyVoiceLanguage, "de")
}

// SaveVoiceLanguage speichert die Voice-Sprache
func (s *Service) SaveVoiceLanguage(lang string) error {
	return s.SetString(KeyVoiceLanguage, lang)
}
