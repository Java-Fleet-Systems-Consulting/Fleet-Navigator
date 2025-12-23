package settings

import (
	"log"
	"strconv"
)

// =============================================================================
// SERVICE BASE - Basis-Service mit generischen Hilfsfunktionen
// =============================================================================

// Service verwaltet App-Einstellungen
type Service struct {
	repo *Repository
}

// NewService erstellt einen neuen Service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// =============================================================================
// GENERIC HELPERS - Typ-sichere Getter/Setter
// =============================================================================

// --- String ---

// GetString holt einen String-Wert
func (s *Service) GetString(key, defaultValue string) string {
	return s.repo.GetOrDefault(key, defaultValue)
}

// SetString speichert einen String-Wert
func (s *Service) SetString(key, value string) error {
	return s.repo.Set(key, value)
}

// --- Bool ---

// GetBool holt einen Bool-Wert
func (s *Service) GetBool(key string, defaultValue bool) bool {
	value := s.repo.GetOrDefault(key, "")
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1"
}

// SetBool speichert einen Bool-Wert
func (s *Service) SetBool(key string, value bool) error {
	strValue := "false"
	if value {
		strValue = "true"
	}
	return s.repo.Set(key, strValue)
}

// --- Int64 ---

// GetInt64 holt einen Int64-Wert
func (s *Service) GetInt64(key string, defaultValue int64) int64 {
	value := s.repo.GetOrDefault(key, "")
	if value == "" {
		return defaultValue
	}
	i, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return defaultValue
	}
	return i
}

// SetInt64 speichert einen Int64-Wert
func (s *Service) SetInt64(key string, value int64) error {
	return s.repo.Set(key, strconv.FormatInt(value, 10))
}

// --- Int ---

// GetInt holt einen Int-Wert
func (s *Service) GetInt(key string, defaultValue int) int {
	return int(s.GetInt64(key, int64(defaultValue)))
}

// SetInt speichert einen Int-Wert
func (s *Service) SetInt(key string, value int) error {
	return s.SetInt64(key, int64(value))
}

// --- Float64 ---

// GetFloat64 holt einen Float64-Wert
func (s *Service) GetFloat64(key string, defaultValue float64) float64 {
	value := s.repo.GetOrDefault(key, "")
	if value == "" {
		return defaultValue
	}
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue
	}
	return f
}

// SetFloat64 speichert einen Float64-Wert
func (s *Service) SetFloat64(key string, value float64) error {
	return s.repo.Set(key, strconv.FormatFloat(value, 'f', -1, 64))
}

// =============================================================================
// ADMINISTRATIVE FUNCTIONS
// =============================================================================

// GetAll gibt alle Einstellungen zurück
func (s *Service) GetAll() ([]AppSetting, error) {
	return s.repo.GetAll()
}

// Delete löscht eine Einstellung
func (s *Service) Delete(key string) error {
	return s.repo.Delete(key)
}

// ResetSettings setzt alle Einstellungen auf Defaults zurück
func (s *Service) ResetSettings() error {
	log.Printf("WARNUNG: Setze alle Einstellungen zurück!")
	return s.repo.DeleteByPrefix("")
}

// ResetByPrefix setzt Einstellungen mit Prefix zurück
func (s *Service) ResetByPrefix(prefix string) error {
	log.Printf("WARNUNG: Setze Einstellungen mit Prefix '%s' zurück!", prefix)
	return s.repo.DeleteByPrefix(prefix)
}

// GetByPrefix holt alle Einstellungen mit Prefix
func (s *Service) GetByPrefix(prefix string) (map[string]string, error) {
	return s.repo.GetByPrefix(prefix)
}
