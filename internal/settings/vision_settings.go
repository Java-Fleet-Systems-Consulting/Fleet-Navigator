package settings

import (
	"log"
	"path/filepath"
	"strings"
)

// =============================================================================
// VISION SETTINGS - Bildanalyse und Model Chaining
// =============================================================================

// --- Vision Model Settings ---

// GetVisionSettings holt die Vision-Einstellungen
func (s *Service) GetVisionSettings() VisionSettings {
	return VisionSettings{
		Enabled:     s.GetBool(KeyVisionEnabled, true),
		Model:       s.GetString(KeyVisionModel, "MiniCPM-V-2.6"), // Standard: MiniCPM-V (beste OCR)
		ModelPath:   s.GetString(KeyVisionModelPath, ""),
		MmprojPath:  s.GetString(KeyVisionMmprojPath, ""),
		AutoRestore: s.GetBool(KeyVisionAutoRestore, true),
	}
}

// SaveVisionSettings speichert die Vision-Einstellungen
func (s *Service) SaveVisionSettings(settings VisionSettings) error {
	if err := s.SetBool(KeyVisionEnabled, settings.Enabled); err != nil {
		return err
	}
	if err := s.SetString(KeyVisionModel, settings.Model); err != nil {
		return err
	}
	if err := s.SetString(KeyVisionModelPath, settings.ModelPath); err != nil {
		return err
	}
	if err := s.SetString(KeyVisionMmprojPath, settings.MmprojPath); err != nil {
		return err
	}
	if err := s.SetBool(KeyVisionAutoRestore, settings.AutoRestore); err != nil {
		return err
	}
	log.Printf("Vision-Einstellungen gespeichert: enabled=%v, model=%s", settings.Enabled, settings.Model)
	return nil
}

// --- Individual Vision Getters/Setters ---

// GetVisionModel gibt das Vision-Modell zurück
func (s *Service) GetVisionModel() string {
	return s.GetString(KeyVisionModel, "MiniCPM-V-2.6")
}

// SaveVisionModel speichert das Vision-Modell
func (s *Service) SaveVisionModel(model string) error {
	return s.SetString(KeyVisionModel, model)
}

// SetVisionModelPath speichert nur den Vision-Modell-Pfad
func (s *Service) SetVisionModelPath(path string) error {
	return s.SetString(KeyVisionModelPath, path)
}

// SetVisionMmprojPath speichert nur den mmproj-Pfad
func (s *Service) SetVisionMmprojPath(path string) error {
	return s.SetString(KeyVisionMmprojPath, path)
}

// SaveVisionSettingsSimple speichert Vision-Settings mit einfachen Parametern
// Diese Methode wird vom Setup-Handler verwendet (SettingsUpdater Interface)
func (s *Service) SaveVisionSettingsSimple(enabled bool, modelPath, mmprojPath string) error {
	// Modellname aus Pfad ableiten (z.B. "MiniCPM-V-2_6-Q4_K_M.gguf" -> "MiniCPM-V-2.6")
	modelName := deriveVisionModelName(modelPath)

	settings := VisionSettings{
		Enabled:     enabled,
		Model:       modelName,
		ModelPath:   modelPath,
		MmprojPath:  mmprojPath,
		AutoRestore: true, // Standard: Nach Vision-Analyse Chat-Modell wiederherstellen
	}
	log.Printf("Vision-Settings: model=%s, path=%s", modelName, modelPath)
	return s.SaveVisionSettings(settings)
}

// deriveVisionModelName leitet den Modellnamen aus dem Dateipfad ab
func deriveVisionModelName(modelPath string) string {
	if modelPath == "" {
		return "MiniCPM-V-2.6" // Default
	}

	// Dateiname extrahieren und lowercase
	filename := strings.ToLower(filepath.Base(modelPath))

	// Bekannte Modelle erkennen
	switch {
	case strings.Contains(filename, "minicpm"):
		return "MiniCPM-V-2.6"
	case strings.Contains(filename, "llava-1.6") || strings.Contains(filename, "llava-v1.6"):
		return "LLaVA-1.6"
	case strings.Contains(filename, "llava"):
		return "LLaVA"
	default:
		return "MiniCPM-V-2.6" // Default für unbekannte Modelle
	}
}

// --- Chaining Settings ---

// GetChainingSettings gibt die Chaining-Einstellungen zurück
func (s *Service) GetChainingSettings() ChainingSettings {
	thinkFirst := s.GetBool(KeyChainingThinkFirst, true)
	return ChainingSettings{
		Enabled:               s.GetBool(KeyChainingEnabled, false),
		AutoSelect:            s.GetBool(KeyChainingAutoSelect, true),
		VisionModel:           s.GetString(KeyChainingVisionModel, ""),
		AnalysisModel:         s.GetString(KeyChainingAnalysisModel, ""),
		ThinkFirst:            thinkFirst,
		WebSearchThinkFirst:   thinkFirst, // Alias für Kompatibilität
		ShowIntermediateOutput: s.GetBool(KeyChainingShowIntermediateOutput, false),
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
	if err := s.SetBool(KeyChainingThinkFirst, settings.ThinkFirst); err != nil {
		return err
	}
	log.Printf("Chaining-Einstellungen gespeichert: enabled=%v, thinkFirst=%v", settings.Enabled, settings.ThinkFirst)
	return nil
}

// --- Think First (Convenience) ---

// GetThinkFirst gibt zurück ob "Think First" aktiv ist
func (s *Service) GetThinkFirst() bool {
	return s.GetBool(KeyChainingThinkFirst, true)
}

// SaveThinkFirst speichert die Think First Einstellung
func (s *Service) SaveThinkFirst(thinkFirst bool) error {
	return s.SetBool(KeyChainingThinkFirst, thinkFirst)
}

// EnableVisionChaining aktiviert Vision Chaining mit dem angegebenen Vision-Modell
// Wird vom Setup aufgerufen nachdem ein Vision-Modell heruntergeladen wurde
func (s *Service) EnableVisionChaining(visionModel string) error {
	settings := ChainingSettings{
		Enabled:               true, // Chaining aktivieren
		AutoSelect:            true, // Automatisch Vision-Modell wählen wenn Bild erkannt
		VisionModel:           visionModel,
		ThinkFirst:            true, // Standard: Erst denken, dann antworten
		ShowIntermediateOutput: false,
	}
	log.Printf("Vision-Chaining aktiviert: visionModel=%s", visionModel)
	return s.SaveChainingSettings(settings)
}
