package settings

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

// Service verwaltet App-Einstellungen
type Service struct {
	repo *Repository
}

// NewService erstellt einen neuen Service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// --- Model Selection Settings ---

// GetModelSelectionSettings gibt die Model-Selection-Einstellungen zurück
func (s *Service) GetModelSelectionSettings() ModelSelectionSettings {
	return ModelSelectionSettings{
		Enabled:      s.GetBool(KeyModelSelectionEnabled, true),
		DefaultModel: s.GetString(KeyDefaultModel, "qwen2.5:7b"),
		CodeModel:    s.GetString(KeyCodeModel, "qwen2.5-coder:7b"),
		FastModel:    s.GetString(KeyFastModel, "qwen2.5:3b"),
		VisionModel:  s.GetString(KeyVisionModel, "llava:7b"),
	}
}

// UpdateModelSelectionSettings aktualisiert die Model-Selection-Einstellungen
func (s *Service) UpdateModelSelectionSettings(settings ModelSelectionSettings) error {
	if err := s.SetBool(KeyModelSelectionEnabled, settings.Enabled); err != nil {
		return err
	}
	if err := s.SetString(KeyDefaultModel, settings.DefaultModel); err != nil {
		return err
	}
	if err := s.SetString(KeyCodeModel, settings.CodeModel); err != nil {
		return err
	}
	if err := s.SetString(KeyFastModel, settings.FastModel); err != nil {
		return err
	}
	if err := s.SetString(KeyVisionModel, settings.VisionModel); err != nil {
		return err
	}
	log.Printf("Model Selection Settings aktualisiert")
	return nil
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

// --- UI Settings ---

// GetShowWelcomeTiles gibt zurück ob Welcome-Tiles angezeigt werden sollen
func (s *Service) GetShowWelcomeTiles() bool {
	return s.GetBool(KeyShowWelcomeTiles, true)
}

// SaveShowWelcomeTiles speichert die Welcome-Tiles-Einstellung
func (s *Service) SaveShowWelcomeTiles(show bool) error {
	return s.SetBool(KeyShowWelcomeTiles, show)
}

// GetShowTopBar gibt zurück ob die TopBar angezeigt werden soll
// Default: false (für professionelle Ansicht - nur bei Debugging aktivieren)
func (s *Service) GetShowTopBar() bool {
	return s.GetBool(KeyShowTopBar, false)
}

// SaveShowTopBar speichert die TopBar-Einstellung
func (s *Service) SaveShowTopBar(show bool) error {
	return s.SetBool(KeyShowTopBar, show)
}

// GetUITheme gibt das aktuelle UI-Theme zurück
func (s *Service) GetUITheme() string {
	return s.GetString(KeyUITheme, "tech-dark")
}

// SaveUITheme speichert das UI-Theme
func (s *Service) SaveUITheme(theme string) error {
	log.Printf("Speichere UI-Theme: %s", theme)
	return s.SetString(KeyUITheme, theme)
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

// --- Mate Model Settings ---

// GetEmailModel gibt das Email-Modell zurück
func (s *Service) GetEmailModel() string {
	return s.GetString(KeyEmailModel, "")
}

// SaveEmailModel speichert das Email-Modell
func (s *Service) SaveEmailModel(model string) error {
	return s.SetString(KeyEmailModel, model)
}

// GetDocumentModel gibt das Document-Modell zurück
func (s *Service) GetDocumentModel() string {
	return s.GetString(KeyDocumentModel, "")
}

// SaveDocumentModel speichert das Document-Modell
func (s *Service) SaveDocumentModel(model string) error {
	return s.SetString(KeyDocumentModel, model)
}

// GetLogAnalysisModel gibt das Log-Analyse-Modell zurück
func (s *Service) GetLogAnalysisModel() string {
	return s.GetString(KeyLogAnalysisModel, "")
}

// SaveLogAnalysisModel speichert das Log-Analyse-Modell
func (s *Service) SaveLogAnalysisModel(model string) error {
	return s.SetString(KeyLogAnalysisModel, model)
}

// GetCoderModel gibt das Coder-Modell zurück (für FleetCoder)
func (s *Service) GetCoderModel() string {
	return s.GetString(KeyCoderModel, "")
}

// SaveCoderModel speichert das Coder-Modell
func (s *Service) SaveCoderModel(model string) error {
	return s.SetString(KeyCoderModel, model)
}

// --- Generic Helpers ---

// GetString holt einen String-Wert
func (s *Service) GetString(key, defaultValue string) string {
	return s.repo.GetOrDefault(key, defaultValue)
}

// SetString speichert einen String-Wert
func (s *Service) SetString(key, value string) error {
	return s.repo.Set(key, value)
}

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

// GetAll gibt alle Einstellungen zurück
func (s *Service) GetAll() ([]AppSetting, error) {
	return s.repo.GetAll()
}

// Delete löscht eine Einstellung
func (s *Service) Delete(key string) error {
	return s.repo.Delete(key)
}

// --- Reset Functions ---

// ResetSettings setzt alle Einstellungen auf Defaults zurück
func (s *Service) ResetSettings() error {
	log.Printf("WARNUNG: Setze alle Einstellungen zurück!")
	return s.repo.DeleteByPrefix("")
}

// ResetUISettings setzt nur UI-Einstellungen zurück
func (s *Service) ResetUISettings() error {
	log.Printf("WARNUNG: Setze UI-Einstellungen zurück!")
	return s.repo.DeleteByPrefix("ui.")
}

// --- Sampling Parameters (Wichtig für KI-Verhalten) ---

// GetSamplingParams gibt die Sampling-Parameter zurück
func (s *Service) GetSamplingParams() SamplingParams {
	return SamplingParams{
		Temperature:   s.GetFloat64(KeySamplingTemperature, 0.7),
		TopP:          s.GetFloat64(KeySamplingTopP, 0.9),
		TopK:          int(s.GetInt64(KeySamplingTopK, 40)),
		MaxTokens:     int(s.GetInt64(KeySamplingMaxTokens, 2048)),
		RepeatPenalty: s.GetFloat64(KeySamplingRepeatPenalty, 1.1),
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
	if err := s.SetInt64(KeySamplingTopK, int64(params.TopK)); err != nil {
		return err
	}
	if err := s.SetInt64(KeySamplingMaxTokens, int64(params.MaxTokens)); err != nil {
		return err
	}
	if err := s.SetFloat64(KeySamplingRepeatPenalty, params.RepeatPenalty); err != nil {
		return err
	}
	log.Printf("Sampling-Parameter gespeichert: temp=%.2f, topP=%.2f", params.Temperature, params.TopP)
	return nil
}

// --- Chaining Settings (Wichtig für KI-Workflow) ---

// GetChainingSettings gibt die Chaining-Einstellungen zurück
func (s *Service) GetChainingSettings() ChainingSettings {
	return ChainingSettings{
		Enabled:                s.GetBool(KeyChainingEnabled, false),
		AutoSelect:             s.GetBool(KeyChainingAutoSelect, true),
		VisionModel:            s.GetString(KeyChainingVisionModel, ""),
		AnalysisModel:          s.GetString(KeyChainingAnalysisModel, ""),
		ShowIntermediateOutput: s.GetBool(KeyChainingShowIntermediateOutput, false),
		WebSearchThinkFirst:    s.GetBool(KeyChainingWebSearchThinkFirst, true), // Default: true
	}
}

// SaveChainingSettings speichert die Chaining-Einstellungen
func (s *Service) SaveChainingSettings(settings ChainingSettings) error {
	if err := s.SetBool(KeyChainingEnabled, settings.Enabled); err != nil {
		return err
	}
	if err := s.SetBool(KeyChainingAutoSelect, settings.AutoSelect); err != nil {
		return err
	}
	if err := s.SetString(KeyChainingVisionModel, settings.VisionModel); err != nil {
		return err
	}
	if err := s.SetString(KeyChainingAnalysisModel, settings.AnalysisModel); err != nil {
		return err
	}
	if err := s.SetBool(KeyChainingShowIntermediateOutput, settings.ShowIntermediateOutput); err != nil {
		return err
	}
	if err := s.SetBool(KeyChainingWebSearchThinkFirst, settings.WebSearchThinkFirst); err != nil {
		return err
	}
	log.Printf("Chaining-Einstellungen gespeichert: enabled=%v, thinkFirst=%v", settings.Enabled, settings.WebSearchThinkFirst)
	return nil
}

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

// --- Float64 Helpers ---

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

// --- Voice Settings ---

// GetVoiceSettings holt die Voice-Konfiguration aus der DB
func (s *Service) GetVoiceSettings() VoiceSettings {
	return VoiceSettings{
		WhisperModel: s.GetString(KeyVoiceWhisperModel, "base"),
		PiperVoice:   s.GetString(KeyVoicePiperVoice, "de_DE-thorsten-medium"),
		Language:     s.GetString(KeyVoiceLanguage, "de"),
	}
}

// SaveVoiceSettings speichert die Voice-Konfiguration in der DB
func (s *Service) SaveVoiceSettings(settings VoiceSettings) error {
	if err := s.repo.Set(KeyVoiceWhisperModel, settings.WhisperModel); err != nil {
		return err
	}
	if err := s.repo.Set(KeyVoicePiperVoice, settings.PiperVoice); err != nil {
		return err
	}
	if err := s.repo.Set(KeyVoiceLanguage, settings.Language); err != nil {
		return err
	}
	log.Printf("Voice-Settings gespeichert: whisper=%s, piper=%s, lang=%s",
		settings.WhisperModel, settings.PiperVoice, settings.Language)
	return nil
}

// SaveWhisperModel speichert nur das Whisper-Modell
func (s *Service) SaveWhisperModel(model string) error {
	if err := s.repo.Set(KeyVoiceWhisperModel, model); err != nil {
		return err
	}
	log.Printf("Whisper-Modell gespeichert: %s", model)
	return nil
}

// GetWhisperModel holt das aktuelle Whisper-Modell
func (s *Service) GetWhisperModel() string {
	return s.GetString(KeyVoiceWhisperModel, "base")
}

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

// --- Vision Swap Settings (VRAM Memory Management) ---

// GetVisionSwapSettings holt die Vision-Swap-Einstellungen
func (s *Service) GetVisionSwapSettings() VisionSwapSettings {
	return VisionSwapSettings{
		Enabled:     s.GetBool(KeyVisionSwapEnabled, true),       // Default: aktiviert
		ModelPath:   s.GetString(KeyVisionSwapModelPath, ""),
		MmprojPath:  s.GetString(KeyVisionSwapMmprojPath, ""),
		AutoRestore: s.GetBool(KeyVisionSwapAutoRestore, true),   // Default: automatisch zurückwechseln
	}
}

// SaveVisionSwapSettings speichert die Vision-Swap-Einstellungen
func (s *Service) SaveVisionSwapSettings(settings VisionSwapSettings) error {
	if err := s.SetBool(KeyVisionSwapEnabled, settings.Enabled); err != nil {
		return err
	}
	if err := s.SetString(KeyVisionSwapModelPath, settings.ModelPath); err != nil {
		return err
	}
	if err := s.SetString(KeyVisionSwapMmprojPath, settings.MmprojPath); err != nil {
		return err
	}
	if err := s.SetBool(KeyVisionSwapAutoRestore, settings.AutoRestore); err != nil {
		return err
	}
	log.Printf("Vision-Swap-Einstellungen gespeichert: enabled=%v, autoRestore=%v", settings.Enabled, settings.AutoRestore)
	return nil
}

// SetVisionModelPath speichert nur den Vision-Modell-Pfad
func (s *Service) SetVisionModelPath(path string) error {
	return s.SetString(KeyVisionSwapModelPath, path)
}

// SetVisionMmprojPath speichert nur den mmproj-Pfad
func (s *Service) SetVisionMmprojPath(path string) error {
	return s.SetString(KeyVisionSwapMmprojPath, path)
}
