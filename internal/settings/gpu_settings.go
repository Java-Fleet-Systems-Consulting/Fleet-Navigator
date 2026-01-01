package settings

import (
	"log"
	"time"
)

// =============================================================================
// GPU SETTINGS - Multi-GPU Konfiguration
// =============================================================================

// GetGPUSettings holt die GPU-Einstellungen
func (s *Service) GetGPUSettings() GPUSettings {
	return GPUSettings{
		AutoDetect: s.GetBool(KeyGPUAutoDetect, true),
		ChatGPU: GPUAssignment{
			GPUID:     s.GetInt(KeyGPUChatID, -1),
			Backend:   s.GetString(KeyGPUChatBackend, "auto"),
			GPULayers: s.GetInt(KeyGPUChatLayers, 99),
		},
		VisionGPU: GPUAssignment{
			GPUID:     s.GetInt(KeyGPUVisionID, -1),
			Backend:   s.GetString(KeyGPUVisionBackend, "auto"),
			GPULayers: s.GetInt(KeyGPUVisionLayers, 99),
		},
		Strategy:   s.GetString(KeyGPUStrategy, "Automatische Erkennung"),
		LastDetect: s.GetString(KeyGPULastDetect, ""),
	}
}

// SaveGPUSettings speichert die GPU-Einstellungen
func (s *Service) SaveGPUSettings(settings GPUSettings) error {
	if err := s.SetBool(KeyGPUAutoDetect, settings.AutoDetect); err != nil {
		return err
	}

	// Chat GPU
	if err := s.SetInt(KeyGPUChatID, settings.ChatGPU.GPUID); err != nil {
		return err
	}
	if err := s.SetString(KeyGPUChatBackend, settings.ChatGPU.Backend); err != nil {
		return err
	}
	if err := s.SetInt(KeyGPUChatLayers, settings.ChatGPU.GPULayers); err != nil {
		return err
	}

	// Vision GPU
	if err := s.SetInt(KeyGPUVisionID, settings.VisionGPU.GPUID); err != nil {
		return err
	}
	if err := s.SetString(KeyGPUVisionBackend, settings.VisionGPU.Backend); err != nil {
		return err
	}
	if err := s.SetInt(KeyGPUVisionLayers, settings.VisionGPU.GPULayers); err != nil {
		return err
	}

	// Strategie und Zeitstempel
	if err := s.SetString(KeyGPUStrategy, settings.Strategy); err != nil {
		return err
	}
	if err := s.SetString(KeyGPULastDetect, settings.LastDetect); err != nil {
		return err
	}

	log.Printf("GPU-Einstellungen gespeichert: Chat→GPU#%d (%s), Vision→GPU#%d (%s)",
		settings.ChatGPU.GPUID, settings.ChatGPU.Backend,
		settings.VisionGPU.GPUID, settings.VisionGPU.Backend)
	return nil
}

// SaveGPUAssignmentFromDetection speichert GPU-Zuweisung basierend auf Hardware-Erkennung
// Diese Methode wird vom Hardware-Modul aufgerufen nach erfolgreicher Erkennung
func (s *Service) SaveGPUAssignmentFromDetection(
	chatGPUID int, chatBackend string, chatLayers int, chatGPUName string,
	visionGPUID int, visionBackend string, visionLayers int, visionGPUName string,
	strategy string,
) error {
	settings := GPUSettings{
		AutoDetect: true,
		ChatGPU: GPUAssignment{
			GPUID:     chatGPUID,
			GPUName:   chatGPUName,
			Backend:   chatBackend,
			GPULayers: chatLayers,
		},
		VisionGPU: GPUAssignment{
			GPUID:     visionGPUID,
			GPUName:   visionGPUName,
			Backend:   visionBackend,
			GPULayers: visionLayers,
		},
		Strategy:   strategy,
		LastDetect: time.Now().Format(time.RFC3339),
	}
	return s.SaveGPUSettings(settings)
}

// GetChatGPUBackend gibt das Backend für den Chat-Server zurück
func (s *Service) GetChatGPUBackend() string {
	return s.GetString(KeyGPUChatBackend, "auto")
}

// GetVisionGPUBackend gibt das Backend für den Vision-Server zurück
func (s *Service) GetVisionGPUBackend() string {
	return s.GetString(KeyGPUVisionBackend, "auto")
}

// GetChatGPULayers gibt die GPU-Layers für den Chat-Server zurück
func (s *Service) GetChatGPULayers() int {
	return s.GetInt(KeyGPUChatLayers, 99)
}

// GetVisionGPULayers gibt die GPU-Layers für den Vision-Server zurück
func (s *Service) GetVisionGPULayers() int {
	return s.GetInt(KeyGPUVisionLayers, 99)
}
