package settings

import "log"

// =============================================================================
// UI SETTINGS - Benutzeroberflächen-Einstellungen
// =============================================================================

// --- Theme ---

// GetUITheme gibt das aktuelle UI-Theme zurück
func (s *Service) GetUITheme() string {
	return s.GetString(KeyUITheme, "tech-dark")
}

// SaveUITheme speichert das UI-Theme
func (s *Service) SaveUITheme(theme string) error {
	log.Printf("Speichere UI-Theme: %s", theme)
	return s.SetString(KeyUITheme, theme)
}

// --- Dark Mode ---

// GetDarkMode gibt zurück ob Dark Mode aktiv ist
func (s *Service) GetDarkMode() bool {
	return s.GetBool(KeyDarkMode, true)
}

// SaveDarkMode speichert die Dark Mode Einstellung
func (s *Service) SaveDarkMode(darkMode bool) error {
	return s.SetBool(KeyDarkMode, darkMode)
}

// --- Welcome Tiles ---

// GetShowWelcomeTiles gibt zurück ob Welcome-Tiles angezeigt werden sollen
func (s *Service) GetShowWelcomeTiles() bool {
	return s.GetBool(KeyShowWelcomeTiles, true)
}

// SaveShowWelcomeTiles speichert die Welcome-Tiles-Einstellung
func (s *Service) SaveShowWelcomeTiles(show bool) error {
	return s.SetBool(KeyShowWelcomeTiles, show)
}

// --- TopBar ---

// GetShowTopBar gibt zurück ob die TopBar angezeigt werden soll
// Default: false (für professionelle Ansicht - nur bei Debugging aktivieren)
func (s *Service) GetShowTopBar() bool {
	return s.GetBool(KeyShowTopBar, false)
}

// SaveShowTopBar speichert die TopBar-Einstellung
func (s *Service) SaveShowTopBar(show bool) error {
	return s.SetBool(KeyShowTopBar, show)
}

// --- Font Size ---

// GetFontSize gibt die Schriftgröße zurück (50-150%)
func (s *Service) GetFontSize() int {
	return s.GetInt(KeyFontSize, 100)
}

// SaveFontSize speichert die Schriftgröße
func (s *Service) SaveFontSize(size int) error {
	// Validierung: 50-150%
	if size < 50 {
		size = 50
	}
	if size > 150 {
		size = 150
	}
	return s.SetInt(KeyFontSize, size)
}

// --- Web Search Animation ---

// WebSearchAnimationType definiert die verfügbaren Animationstypen
// Gültige Werte: "data-wave", "orbit", "radar", "constellation"
const (
	AnimationDataWave     = "data-wave"
	AnimationOrbit        = "orbit"
	AnimationRadar        = "radar"
	AnimationConstellation = "constellation"
)

// GetWebSearchAnimation gibt den aktuellen Animationstyp zurück
func (s *Service) GetWebSearchAnimation() string {
	anim := s.GetString(KeyWebSearchAnimation, AnimationDataWave)
	// Validierung
	switch anim {
	case AnimationDataWave, AnimationOrbit, AnimationRadar, AnimationConstellation:
		return anim
	default:
		return AnimationDataWave
	}
}

// SaveWebSearchAnimation speichert den Animationstyp
func (s *Service) SaveWebSearchAnimation(animation string) error {
	// Validierung
	switch animation {
	case AnimationDataWave, AnimationOrbit, AnimationRadar, AnimationConstellation:
		log.Printf("Speichere Web-Suche Animation: %s", animation)
		return s.SetString(KeyWebSearchAnimation, animation)
	default:
		log.Printf("Ungültiger Animationstyp: %s, verwende Default", animation)
		return s.SetString(KeyWebSearchAnimation, AnimationDataWave)
	}
}

// --- Complete UI Settings ---

// GetUISettings gibt alle UI-Einstellungen als Struct zurück
func (s *Service) GetUISettings() UISettings {
	return UISettings{
		Theme:              s.GetUITheme(),
		DarkMode:           s.GetDarkMode(),
		ShowWelcomeTiles:   s.GetShowWelcomeTiles(),
		ShowTopBar:         s.GetShowTopBar(),
		FontSize:           s.GetFontSize(),
		WebSearchAnimation: s.GetWebSearchAnimation(),
	}
}

// SaveUISettings speichert alle UI-Einstellungen
func (s *Service) SaveUISettings(settings UISettings) error {
	if err := s.SaveUITheme(settings.Theme); err != nil {
		return err
	}
	if err := s.SaveDarkMode(settings.DarkMode); err != nil {
		return err
	}
	if err := s.SaveShowWelcomeTiles(settings.ShowWelcomeTiles); err != nil {
		return err
	}
	if err := s.SaveShowTopBar(settings.ShowTopBar); err != nil {
		return err
	}
	if err := s.SaveFontSize(settings.FontSize); err != nil {
		return err
	}
	if err := s.SaveWebSearchAnimation(settings.WebSearchAnimation); err != nil {
		return err
	}
	return nil
}

// ResetUISettings setzt nur UI-Einstellungen zurück
func (s *Service) ResetUISettings() error {
	log.Printf("WARNUNG: Setze UI-Einstellungen zurück!")
	return s.ResetByPrefix("ui.")
}
