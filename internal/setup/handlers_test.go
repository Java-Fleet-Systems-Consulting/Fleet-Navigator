package setup

import (
	"runtime"
	"testing"
)

// TestMirrorAssetMappingHasCUDA prüft dass CUDA-Mappings für alle Plattformen existieren
func TestMirrorAssetMappingHasCUDA(t *testing.T) {
	// Windows CUDA
	if _, ok := MirrorAssetMapping["win-cuda-12.4-x64"]; !ok {
		t.Error("MirrorAssetMapping fehlt Windows CUDA 12.4")
	}

	// Linux CUDA (war der Bug!)
	if _, ok := MirrorAssetMapping["ubuntu-cuda-x64"]; !ok {
		t.Error("MirrorAssetMapping fehlt Linux CUDA")
	}
}

// TestMirrorAssetMappingHasAllPlatforms prüft alle Plattform-Mappings
func TestMirrorAssetMappingHasAllPlatforms(t *testing.T) {
	requiredMappings := []struct {
		key  string
		desc string
	}{
		// Windows
		{"win-cuda-12.4-x64", "Windows CUDA 12.4"},
		{"win-vulkan-x64", "Windows Vulkan"},
		{"win-cpu-x64", "Windows CPU"},
		// Linux
		{"ubuntu-cuda-x64", "Linux CUDA"},
		{"ubuntu-vulkan-x64", "Linux Vulkan"},
		{"ubuntu-x64", "Linux CPU"},
		// macOS
		{"macos-arm64", "macOS ARM64"},
		{"macos-x64", "macOS x64"},
	}

	for _, m := range requiredMappings {
		t.Run(m.desc, func(t *testing.T) {
			if _, ok := MirrorAssetMapping[m.key]; !ok {
				t.Errorf("MirrorAssetMapping fehlt: %s (%s)", m.key, m.desc)
			}
		})
	}
}

// TestGetMirrorAssetKeyLinuxCUDA prüft den Linux CUDA Fall
func TestGetMirrorAssetKeyLinuxCUDA(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Test nur auf Linux relevant")
	}

	downloader := &LlamaServerDownloader{}
	key := downloader.GetMirrorAssetKey(GPUTypeCUDA)

	if key != "ubuntu-cuda-x64" {
		t.Errorf("GetMirrorAssetKey(CUDA) auf Linux erwartet 'ubuntu-cuda-x64', bekommen '%s'", key)
	}
}

// TestGetMirrorAssetKeyLinuxVulkan prüft den Linux Vulkan Fall
func TestGetMirrorAssetKeyLinuxVulkan(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Test nur auf Linux relevant")
	}

	downloader := &LlamaServerDownloader{}
	key := downloader.GetMirrorAssetKey(GPUTypeVulkan)

	if key != "ubuntu-vulkan-x64" {
		t.Errorf("GetMirrorAssetKey(Vulkan) auf Linux erwartet 'ubuntu-vulkan-x64', bekommen '%s'", key)
	}
}

// TestGetMirrorAssetKeyLinuxCPU prüft den Linux CPU Fall
func TestGetMirrorAssetKeyLinuxCPU(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Test nur auf Linux relevant")
	}

	downloader := &LlamaServerDownloader{}
	key := downloader.GetMirrorAssetKey(GPUTypeNone)

	if key != "ubuntu-x64" {
		t.Errorf("GetMirrorAssetKey(None) auf Linux erwartet 'ubuntu-x64', bekommen '%s'", key)
	}
}

// TestGPUTypeConstants prüft dass alle GPU-Typen definiert sind
func TestGPUTypeConstants(t *testing.T) {
	// Stelle sicher dass alle GPU-Typen unterschiedlich sind
	types := map[GPUType]string{
		GPUTypeNone:   "none",
		GPUTypeCUDA:   "cuda",
		GPUTypeVulkan: "vulkan",
	}

	seen := make(map[GPUType]bool)
	for gpuType, name := range types {
		if seen[gpuType] {
			t.Errorf("GPU-Typ %s ist doppelt definiert", name)
		}
		seen[gpuType] = true
	}
}

// TestMirrorConfigDefaults prüft die Mirror-Konfiguration
func TestMirrorConfigDefaults(t *testing.T) {
	if MirrorConfig.BaseURL == "" {
		t.Error("MirrorConfig.BaseURL sollte nicht leer sein")
	}

	if !MirrorConfig.Enabled {
		t.Error("MirrorConfig.Enabled sollte true sein")
	}

	if MirrorConfig.LlamaServerPath == "" {
		t.Error("MirrorConfig.LlamaServerPath sollte nicht leer sein")
	}

	if MirrorConfig.MinSpeedMBps <= 0 {
		t.Error("MirrorConfig.MinSpeedMBps sollte > 0 sein")
	}
}

// TestMirrorAssetMappingFilenames prüft die Dateinamen
func TestMirrorAssetMappingFilenames(t *testing.T) {
	testCases := []struct {
		key      string
		contains string
	}{
		{"ubuntu-cuda-x64", "cuda"},
		{"ubuntu-vulkan-x64", "vulkan"},
		{"win-cuda-12.4-x64", "cuda"},
	}

	for _, tc := range testCases {
		t.Run(tc.key, func(t *testing.T) {
			filename, ok := MirrorAssetMapping[tc.key]
			if !ok {
				t.Fatalf("Mapping nicht gefunden: %s", tc.key)
			}
			if filename == "" {
				t.Error("Dateiname sollte nicht leer sein")
			}
		})
	}
}

// TestCampaignModelsExist prüft dass Campaign-Modelle definiert sind
func TestCampaignModelsExist(t *testing.T) {
	if len(CampaignModels) == 0 {
		t.Error("CampaignModels sollte mindestens ein Modell enthalten")
	}

	// Prüfe dass jedes Modell eine ID hat
	for i, model := range CampaignModels {
		if model.ID == "" {
			t.Errorf("CampaignModels[%d] hat keine ID", i)
		}
		if model.Name == "" {
			t.Errorf("CampaignModels[%d] hat keinen Namen", i)
		}
		if model.SizeGB <= 0 {
			t.Errorf("CampaignModels[%d] hat keine gültige Größe", i)
		}
	}
}

// TestCampaignModelsHasRecommended prüft dass ein empfohlenes Modell existiert
func TestCampaignModelsHasRecommended(t *testing.T) {
	hasRecommended := false
	for _, model := range CampaignModels {
		if model.Recommended {
			hasRecommended = true
			break
		}
	}

	if !hasRecommended {
		t.Error("CampaignModels sollte mindestens ein empfohlenes Modell haben")
	}
}

// ============================================================================
// Mirror-First Download-Strategie Tests
// ============================================================================

// TestMirrorConfigIsPrimarySource prüft dass Mirror als primäre Quelle konfiguriert ist
func TestMirrorConfigIsPrimarySource(t *testing.T) {
	// Mirror muss aktiviert sein
	if !MirrorConfig.Enabled {
		t.Error("MirrorConfig.Enabled sollte true sein (Mirror = primäre Quelle)")
	}

	// Fallback zu HuggingFace muss aktiviert sein
	if !MirrorConfig.FallbackToHuggingFace {
		t.Error("MirrorConfig.FallbackToHuggingFace sollte true sein")
	}

	// Fallback zu GitHub muss aktiviert sein
	if !MirrorConfig.FallbackToGitHub {
		t.Error("MirrorConfig.FallbackToGitHub sollte true sein")
	}
}

// TestMirrorConfigBaseURL prüft die Mirror-URL
func TestMirrorConfigBaseURL(t *testing.T) {
	expectedURL := "https://mirror.java-fleet.com"
	if MirrorConfig.BaseURL != expectedURL {
		t.Errorf("MirrorConfig.BaseURL erwartet '%s', bekommen '%s'", expectedURL, MirrorConfig.BaseURL)
	}
}

// TestMirrorConfigPaths prüft alle Pfad-Konfigurationen
func TestMirrorConfigPaths(t *testing.T) {
	paths := []struct {
		name     string
		path     string
		expected string
	}{
		{"LlamaServerPath", MirrorConfig.LlamaServerPath, "/llama-server/latest/"},
		{"ModelsPath", MirrorConfig.ModelsPath, "/models/"},
		{"VisionPath", MirrorConfig.VisionPath, "/vision/"},
		{"WhisperPath", MirrorConfig.WhisperPath, "/whisper/"},
		{"PiperPath", MirrorConfig.PiperPath, "/piper/"},
		{"TesseractPath", MirrorConfig.TesseractPath, "/tesseract/"},
	}

	for _, p := range paths {
		t.Run(p.name, func(t *testing.T) {
			if p.path != p.expected {
				t.Errorf("%s erwartet '%s', bekommen '%s'", p.name, p.expected, p.path)
			}
		})
	}
}

// TestMirrorURLConstruction prüft die URL-Konstruktion für verschiedene Komponenten
func TestMirrorURLConstruction(t *testing.T) {
	testCases := []struct {
		name     string
		url      string
		contains string
	}{
		{
			"LLM Model URL",
			MirrorConfig.BaseURL + MirrorConfig.ModelsPath + "test-model.gguf",
			"mirror.java-fleet.com/models/test-model.gguf",
		},
		{
			"Vision Model URL",
			MirrorConfig.BaseURL + MirrorConfig.VisionPath + "llava-model.gguf",
			"mirror.java-fleet.com/vision/llava-model.gguf",
		},
		{
			"Whisper Model URL",
			MirrorConfig.BaseURL + MirrorConfig.WhisperPath + "ggml-base.bin",
			"mirror.java-fleet.com/whisper/ggml-base.bin",
		},
		{
			"Piper Voice URL",
			MirrorConfig.BaseURL + MirrorConfig.PiperPath + "voices/de_DE-eva_k-medium.onnx",
			"mirror.java-fleet.com/piper/voices/de_DE-eva_k-medium.onnx",
		},
		{
			"LlamaServer URL",
			MirrorConfig.BaseURL + MirrorConfig.LlamaServerPath + "ubuntu-cuda-12.4-x64.tar.gz",
			"mirror.java-fleet.com/llama-server/latest/ubuntu-cuda-12.4-x64.tar.gz",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.url != "https://"+tc.contains {
				t.Errorf("%s URL falsch konstruiert: %s", tc.name, tc.url)
			}
		})
	}
}

// TestMirrorFirstStrategyConfig prüft dass die Strategie korrekt konfiguriert ist
func TestMirrorFirstStrategyConfig(t *testing.T) {
	// Timeout muss gesetzt sein
	if MirrorConfig.TimeoutSeconds <= 0 {
		t.Error("MirrorConfig.TimeoutSeconds sollte > 0 sein")
	}

	// Speed-Check Konfiguration
	if MirrorConfig.SpeedCheckAfterSec <= 0 {
		t.Error("MirrorConfig.SpeedCheckAfterSec sollte > 0 sein")
	}

	if MirrorConfig.MinSpeedMBps <= 0 {
		t.Error("MirrorConfig.MinSpeedMBps sollte > 0 sein")
	}
}

// TestAllMirrorPathsStartWithSlash prüft dass alle Pfade mit / beginnen
func TestAllMirrorPathsStartWithSlash(t *testing.T) {
	paths := map[string]string{
		"LlamaServerPath": MirrorConfig.LlamaServerPath,
		"ModelsPath":      MirrorConfig.ModelsPath,
		"VisionPath":      MirrorConfig.VisionPath,
		"WhisperPath":     MirrorConfig.WhisperPath,
		"PiperPath":       MirrorConfig.PiperPath,
		"TesseractPath":   MirrorConfig.TesseractPath,
	}

	for name, path := range paths {
		if len(path) == 0 || path[0] != '/' {
			t.Errorf("%s sollte mit '/' beginnen: %s", name, path)
		}
	}
}

// TestAllMirrorPathsEndWithSlash prüft dass alle Pfade mit / enden
func TestAllMirrorPathsEndWithSlash(t *testing.T) {
	paths := map[string]string{
		"LlamaServerPath": MirrorConfig.LlamaServerPath,
		"ModelsPath":      MirrorConfig.ModelsPath,
		"VisionPath":      MirrorConfig.VisionPath,
		"WhisperPath":     MirrorConfig.WhisperPath,
		"PiperPath":       MirrorConfig.PiperPath,
		"TesseractPath":   MirrorConfig.TesseractPath,
	}

	for name, path := range paths {
		if len(path) == 0 || path[len(path)-1] != '/' {
			t.Errorf("%s sollte mit '/' enden: %s", name, path)
		}
	}
}
