package settings

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

// =============================================================================
// USER SETTINGS - Benutzer-Präferenzen, Experten, Disclaimer
// =============================================================================

// --- User Preferences ---

// GetUserPreferences gibt die Benutzer-Präferenzen zurück
func (s *Service) GetUserPreferences() UserPreferences {
	return UserPreferences{
		Locale:   s.GetString(KeyLocale, "de"),
		DarkMode: s.GetBool(KeyDarkMode, true),
	}
}

// SaveUserPreferences speichert die Benutzer-Präferenzen
func (s *Service) SaveUserPreferences(prefs UserPreferences) error {
	if err := s.SetString(KeyLocale, prefs.Locale); err != nil {
		return err
	}
	if err := s.SetBool(KeyDarkMode, prefs.DarkMode); err != nil {
		return err
	}
	log.Printf("Benutzer-Präferenzen gespeichert: locale=%s, darkMode=%v", prefs.Locale, prefs.DarkMode)
	return nil
}

// GetLocale gibt die Sprache zurück
func (s *Service) GetLocale() string {
	return s.GetString(KeyLocale, "de")
}

// SaveLocale speichert die Sprache
func (s *Service) SaveLocale(locale string) error {
	return s.SetString(KeyLocale, locale)
}

// --- Selected Expert ---

// GetSelectedExpertID gibt die ID des ausgewählten Experten zurück
func (s *Service) GetSelectedExpertID() int64 {
	return s.GetInt64(KeySelectedExpert, 0)
}

// SaveSelectedExpertID speichert die ID des ausgewählten Experten
func (s *Service) SaveSelectedExpertID(expertID int64) error {
	log.Printf("Speichere ausgewählten Experten: %d", expertID)
	return s.SetInt64(KeySelectedExpert, expertID)
}

// --- Disclaimer / Legal ---

// GetDisclaimerAccepted prüft ob der rechtliche Hinweis akzeptiert wurde
func (s *Service) GetDisclaimerAccepted() bool {
	return s.GetBool(KeyDisclaimerAccepted, false)
}

// SaveDisclaimerAccepted speichert die Akzeptanz des rechtlichen Hinweises
func (s *Service) SaveDisclaimerAccepted(accepted bool) error {
	if err := s.SetBool(KeyDisclaimerAccepted, accepted); err != nil {
		return err
	}
	// Zeitstempel speichern
	if accepted {
		if err := s.SetString(KeyDisclaimerAcceptedAt, time.Now().Format(time.RFC3339)); err != nil {
			return err
		}
	}
	log.Printf("Disclaimer-Akzeptanz gespeichert: %v", accepted)
	return nil
}

// GetDisclaimerAcceptedAt gibt den Zeitpunkt der Akzeptanz zurück
func (s *Service) GetDisclaimerAcceptedAt() *time.Time {
	str := s.GetString(KeyDisclaimerAcceptedAt, "")
	if str == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return nil
	}
	return &t
}

// --- Web Search Counter ---

// IncrementWebSearchCount erhöht den Websuche-Zähler für den aktuellen Monat
func (s *Service) IncrementWebSearchCount() (int, error) {
	currentMonth := time.Now().Format("2006-01")

	// Aktuellen Wert laden (Format: "YYYY-MM:count")
	stored := s.GetString(KeyWebSearchCount, "")

	var count int
	if stored != "" {
		// Parse "YYYY-MM:count"
		parts := strings.Split(stored, ":")
		if len(parts) == 2 && parts[0] == currentMonth {
			count, _ = strconv.Atoi(parts[1])
		}
		// Wenn anderer Monat, beginnt count bei 0
	}

	count++

	// Speichern
	newValue := fmt.Sprintf("%s:%d", currentMonth, count)
	if err := s.repo.Set(KeyWebSearchCount, newValue); err != nil {
		return count, err
	}

	return count, nil
}

// GetWebSearchCount holt den aktuellen Monatszähler
func (s *Service) GetWebSearchCount() (int, string) {
	currentMonth := time.Now().Format("2006-01")
	stored := s.GetString(KeyWebSearchCount, "")

	if stored == "" {
		return 0, currentMonth
	}

	parts := strings.Split(stored, ":")
	if len(parts) == 2 && parts[0] == currentMonth {
		count, _ := strconv.Atoi(parts[1])
		return count, currentMonth
	}

	// Anderer Monat = 0
	return 0, currentMonth
}
