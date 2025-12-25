package settings

import "log"

// =============================================================================
// VOICE ASSISTANT SETTINGS - Sprachassistent / Lauschfunktion
// =============================================================================

// --- Hauptschalter ---

// GetVoiceAssistantEnabled gibt zurück ob der Sprachassistent (Lauschfunktion) aktiv ist
// Default: false - muss explizit aktiviert werden
func (s *Service) GetVoiceAssistantEnabled() bool {
	return s.GetBool(KeyVoiceAssistantEnabled, false)
}

// SaveVoiceAssistantEnabled speichert den Sprachassistent-Status
func (s *Service) SaveVoiceAssistantEnabled(enabled bool) error {
	log.Printf("Voice Assistant Lauschfunktion: %v", enabled)
	return s.SetBool(KeyVoiceAssistantEnabled, enabled)
}

// --- Wake Word ---

// Wake Word Optionen
const (
	WakeWordHeyEwa = "hey_ewa"  // "Hey Ewa"
	WakeWordEwa    = "ewa"      // Nur "Ewa"
	WakeWordCustom = "custom"   // Benutzerdefiniert
)

// GetVoiceAssistantWakeWord gibt das konfigurierte Wake Word zurück
func (s *Service) GetVoiceAssistantWakeWord() string {
	return s.GetString(KeyVoiceAssistantWakeWord, WakeWordHeyEwa)
}

// SaveVoiceAssistantWakeWord speichert das Wake Word
func (s *Service) SaveVoiceAssistantWakeWord(wakeWord string) error {
	log.Printf("Wake Word gesetzt: %s", wakeWord)
	return s.SetString(KeyVoiceAssistantWakeWord, wakeWord)
}

// --- Auto-Stop ---

// GetVoiceAssistantAutoStop gibt zurück ob nach Antwort automatisch gestoppt wird
func (s *Service) GetVoiceAssistantAutoStop() bool {
	return s.GetBool(KeyVoiceAssistantAutoStop, true)
}

// SaveVoiceAssistantAutoStop speichert die Auto-Stop Einstellung
func (s *Service) SaveVoiceAssistantAutoStop(autoStop bool) error {
	return s.SetBool(KeyVoiceAssistantAutoStop, autoStop)
}

// --- Ruhezeiten (Work-Life-Balance) ---

// GetVoiceAssistantQuietHoursEnabled gibt zurück ob Ruhezeiten aktiv sind
func (s *Service) GetVoiceAssistantQuietHoursEnabled() bool {
	return s.GetBool(KeyVoiceAssistantQuietHoursEnabled, false)
}

// SaveVoiceAssistantQuietHoursEnabled speichert die Ruhezeiten-Einstellung
func (s *Service) SaveVoiceAssistantQuietHoursEnabled(enabled bool) error {
	return s.SetBool(KeyVoiceAssistantQuietHoursEnabled, enabled)
}

// GetVoiceAssistantQuietHoursStart gibt die Startzeit der Ruhezeit zurück (Format: "HH:MM")
func (s *Service) GetVoiceAssistantQuietHoursStart() string {
	return s.GetString(KeyVoiceAssistantQuietHoursStart, "22:00")
}

// SaveVoiceAssistantQuietHoursStart speichert die Startzeit der Ruhezeit
func (s *Service) SaveVoiceAssistantQuietHoursStart(time string) error {
	return s.SetString(KeyVoiceAssistantQuietHoursStart, time)
}

// GetVoiceAssistantQuietHoursEnd gibt die Endzeit der Ruhezeit zurück (Format: "HH:MM")
func (s *Service) GetVoiceAssistantQuietHoursEnd() string {
	return s.GetString(KeyVoiceAssistantQuietHoursEnd, "07:00")
}

// SaveVoiceAssistantQuietHoursEnd speichert die Endzeit der Ruhezeit
func (s *Service) SaveVoiceAssistantQuietHoursEnd(time string) error {
	return s.SetString(KeyVoiceAssistantQuietHoursEnd, time)
}

// --- Komplette Settings ---

// VoiceAssistantSettings enthält alle Sprachassistent-Einstellungen
type VoiceAssistantSettings struct {
	Enabled           bool   `json:"enabled"`           // Lauschfunktion aktiv
	WakeWord          string `json:"wakeWord"`          // Wake Word (hey_ewa, ewa, custom)
	CustomWakeWord    string `json:"customWakeWord"`    // Benutzerdefiniertes Wake Word
	AutoStop          bool   `json:"autoStop"`          // Nach Antwort stoppen
	QuietHoursEnabled bool   `json:"quietHoursEnabled"` // Ruhezeiten aktiv
	QuietHoursStart   string `json:"quietHoursStart"`   // Ruhezeit Start
	QuietHoursEnd     string `json:"quietHoursEnd"`     // Ruhezeit Ende
	TTSEnabled        bool   `json:"ttsEnabled"`        // Text-to-Speech aktiviert
}

// --- Custom Wake Word ---

const KeyVoiceAssistantCustomWakeWord = "voice_assistant_custom_wake_word"

// GetVoiceAssistantCustomWakeWord gibt das benutzerdefinierte Wake Word zurück
func (s *Service) GetVoiceAssistantCustomWakeWord() string {
	return s.GetString(KeyVoiceAssistantCustomWakeWord, "")
}

// SaveVoiceAssistantCustomWakeWord speichert das benutzerdefinierte Wake Word
func (s *Service) SaveVoiceAssistantCustomWakeWord(wakeWord string) error {
	return s.SetString(KeyVoiceAssistantCustomWakeWord, wakeWord)
}

// --- TTS Enabled ---

const KeyVoiceAssistantTTSEnabled = "voice_assistant_tts_enabled"

// GetVoiceAssistantTTSEnabled gibt zurück ob TTS aktiviert ist
func (s *Service) GetVoiceAssistantTTSEnabled() bool {
	return s.GetBool(KeyVoiceAssistantTTSEnabled, true)
}

// SaveVoiceAssistantTTSEnabled speichert die TTS-Einstellung
func (s *Service) SaveVoiceAssistantTTSEnabled(enabled bool) error {
	return s.SetBool(KeyVoiceAssistantTTSEnabled, enabled)
}

// --- Komplette Settings ---

// GetVoiceAssistantSettings gibt alle Voice Assistant Settings zurück
func (s *Service) GetVoiceAssistantSettings() VoiceAssistantSettings {
	return VoiceAssistantSettings{
		Enabled:           s.GetVoiceAssistantEnabled(),
		WakeWord:          s.GetVoiceAssistantWakeWord(),
		CustomWakeWord:    s.GetVoiceAssistantCustomWakeWord(),
		AutoStop:          s.GetVoiceAssistantAutoStop(),
		QuietHoursEnabled: s.GetVoiceAssistantQuietHoursEnabled(),
		QuietHoursStart:   s.GetVoiceAssistantQuietHoursStart(),
		QuietHoursEnd:     s.GetVoiceAssistantQuietHoursEnd(),
		TTSEnabled:        s.GetVoiceAssistantTTSEnabled(),
	}
}

// SaveVoiceAssistantSettings speichert alle Voice Assistant Settings
func (s *Service) SaveVoiceAssistantSettings(settings VoiceAssistantSettings) error {
	if err := s.SaveVoiceAssistantEnabled(settings.Enabled); err != nil {
		return err
	}
	if err := s.SaveVoiceAssistantWakeWord(settings.WakeWord); err != nil {
		return err
	}
	if err := s.SaveVoiceAssistantCustomWakeWord(settings.CustomWakeWord); err != nil {
		return err
	}
	if err := s.SaveVoiceAssistantAutoStop(settings.AutoStop); err != nil {
		return err
	}
	if err := s.SaveVoiceAssistantQuietHoursEnabled(settings.QuietHoursEnabled); err != nil {
		return err
	}
	if err := s.SaveVoiceAssistantQuietHoursStart(settings.QuietHoursStart); err != nil {
		return err
	}
	if err := s.SaveVoiceAssistantQuietHoursEnd(settings.QuietHoursEnd); err != nil {
		return err
	}
	if err := s.SaveVoiceAssistantTTSEnabled(settings.TTSEnabled); err != nil {
		return err
	}
	return nil
}
