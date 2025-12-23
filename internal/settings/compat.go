package settings

// =============================================================================
// COMPATIBILITY LAYER - Alte Methodennamen für Abwärtskompatibilität
// =============================================================================
// Diese Datei kann entfernt werden, sobald alle Aufrufe aktualisiert sind

// --- Vision Swap Settings (alte Benennung) ---

// VisionSwapSettings ist der alte Name für VisionSettings
type VisionSwapSettings = VisionSettings

// GetVisionSwapSettings ist der alte Name für GetVisionSettings
func (s *Service) GetVisionSwapSettings() VisionSettings {
	return s.GetVisionSettings()
}

// SaveVisionSwapSettings ist der alte Name für SaveVisionSettings
func (s *Service) SaveVisionSwapSettings(settings VisionSettings) error {
	return s.SaveVisionSettings(settings)
}

// --- Model Selection Settings (Legacy) ---

// ModelSelectionSettings alte Struktur für Abwärtskompatibilität
type ModelSelectionSettings struct {
	Enabled      bool   `json:"enabled"`
	DefaultModel string `json:"defaultModel"`
	CodeModel    string `json:"codeModel"`
	FastModel    string `json:"fastModel"`
	VisionModel  string `json:"visionModel"`
	// Neue Felder für Frontend-Kompatibilität
	VisionChainingEnabled bool `json:"visionChainingEnabled"`
}

// GetModelSelectionSettings gibt Model-Selection-Settings zurück (Legacy)
func (s *Service) GetModelSelectionSettings() ModelSelectionSettings {
	chaining := s.GetChainingSettings()
	return ModelSelectionSettings{
		Enabled:               true, // Immer aktiv mit llama-server
		DefaultModel:          s.GetDefaultModel(),
		CodeModel:             s.GetCoderModel(),
		FastModel:             "", // Nicht mehr verwendet
		VisionModel:           s.GetVisionModel(),
		VisionChainingEnabled: chaining.Enabled,
	}
}

// UpdateModelSelectionSettings speichert Model-Selection-Settings (Legacy)
func (s *Service) UpdateModelSelectionSettings(settings ModelSelectionSettings) error {
	if err := s.SaveDefaultModel(settings.DefaultModel); err != nil {
		return err
	}
	if settings.CodeModel != "" {
		if err := s.SaveCoderModel(settings.CodeModel); err != nil {
			return err
		}
	}
	if settings.VisionModel != "" {
		if err := s.SaveVisionModel(settings.VisionModel); err != nil {
			return err
		}
	}
	return nil
}

// --- Alte Key-Konstanten (für Code der diese noch referenziert) ---
const (
	// Legacy Keys - alte Namen beibehalten für Kompatibilität
	KeyModelSelectionEnabled = "model.selection.enabled" // Deprecated
	KeyCodeModel             = "model.selection.code"    // -> KeyMateCoderModel
	KeyFastModel             = "model.selection.fast"    // Deprecated (nicht mehr verwendet)

	// Vision Legacy
	KeyVisionChainingEnabled        = "vision.chaining.enabled"         // -> KeyChainingEnabled
	KeyVisionChainingSmartSelection = "vision.chaining.smart.selection" // Deprecated
	KeyAutoSwitchVisionOnImage      = "model.auto_switch.vision_on_image" // Deprecated

	// Vision Swap Legacy (alte Pfade)
	KeyVisionSwapEnabled     = "vision.swap.enabled"      // -> KeyVisionEnabled
	KeyVisionSwapModelPath   = "vision.swap.model_path"   // -> KeyVisionModelPath
	KeyVisionSwapMmprojPath  = "vision.swap.mmproj_path"  // -> KeyVisionMmprojPath
	KeyVisionSwapAutoRestore = "vision.swap.auto_restore" // -> KeyVisionAutoRestore

	// Chaining Legacy
	KeyChainingWebSearchThinkFirst = "chaining.web_search_think_first" // -> KeyChainingThinkFirst

	// Mate Legacy Keys (alte Pfade ohne "mate." Prefix)
	KeyEmailModel       = "mate.model.email"       // Gleich wie KeyMateEmailModel
	KeyDocumentModel    = "mate.model.document"    // Gleich wie KeyMateDocumentModel
	KeyLogAnalysisModel = "mate.model.loganalysis" // Gleich wie KeyMateLogAnalysisModel
	KeyCoderModel       = "mate.model.coder"       // Gleich wie KeyMateCoderModel

	// File Search Legacy
	KeyFileSearchFolders = "filesearch.folders" // Nicht implementiert
	KeyFileSearchEnabled = "filesearch.enabled" // Nicht implementiert
)

// --- ChainingSettings Compat ---
// WebSearchThinkFirst ist jetzt ein Feld in ChainingSettings (types.go)
