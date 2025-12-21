package voice

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

// Eingebettete Binaries für alle Plattformen
// Diese Dateien werden von GitHub Actions gebaut und hier eingebettet

//go:embed bin/linux-amd64/whisper-cli
var embeddedWhisperLinuxAMD64 []byte

// Platzhalter für andere Plattformen (werden später hinzugefügt)
// //go:embed bin/windows-amd64/whisper-cli.exe
// var embeddedWhisperWindowsAMD64 []byte
// //go:embed bin/darwin-amd64/whisper-cli
// var embeddedWhisperDarwinAMD64 []byte
// //go:embed bin/darwin-arm64/whisper-cli
// var embeddedWhisperDarwinARM64 []byte

// ExtractEmbeddedWhisper extrahiert das eingebettete Whisper-Binary für die aktuelle Plattform
func ExtractEmbeddedWhisper(destDir string) (string, error) {
	var binaryData []byte
	var binaryName string

	switch runtime.GOOS {
	case "linux":
		if runtime.GOARCH == "amd64" {
			binaryData = embeddedWhisperLinuxAMD64
			binaryName = "whisper-cli"
		}
	case "windows":
		if runtime.GOARCH == "amd64" {
			// binaryData = embeddedWhisperWindowsAMD64
			binaryName = "whisper-cli.exe"
		}
	case "darwin":
		if runtime.GOARCH == "amd64" {
			// binaryData = embeddedWhisperDarwinAMD64
			binaryName = "whisper-cli"
		} else if runtime.GOARCH == "arm64" {
			// binaryData = embeddedWhisperDarwinARM64
			binaryName = "whisper-cli"
		}
	}

	if len(binaryData) == 0 {
		return "", fmt.Errorf("kein eingebettetes Binary für %s/%s verfügbar", runtime.GOOS, runtime.GOARCH)
	}

	destPath := filepath.Join(destDir, binaryName)

	// Prüfen ob bereits vorhanden und gleiche Größe
	if info, err := os.Stat(destPath); err == nil {
		if info.Size() == int64(len(binaryData)) {
			log.Printf("Whisper-Binary bereits vorhanden: %s", destPath)
			return destPath, nil
		}
	}

	// Verzeichnis erstellen
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("Verzeichnis erstellen: %w", err)
	}

	// Binary schreiben
	if err := os.WriteFile(destPath, binaryData, 0755); err != nil {
		return "", fmt.Errorf("Binary schreiben: %w", err)
	}

	log.Printf("Whisper-Binary extrahiert: %s (%d bytes)", destPath, len(binaryData))
	return destPath, nil
}

// HasEmbeddedWhisper prüft ob ein eingebettetes Binary für das aktuelle OS verfügbar ist
func HasEmbeddedWhisper() bool {
	switch runtime.GOOS {
	case "linux":
		return runtime.GOARCH == "amd64" && len(embeddedWhisperLinuxAMD64) > 0
	case "windows":
		// return runtime.GOARCH == "amd64" && len(embeddedWhisperWindowsAMD64) > 0
		return false // Noch nicht verfügbar
	case "darwin":
		// return (runtime.GOARCH == "amd64" && len(embeddedWhisperDarwinAMD64) > 0) ||
		//        (runtime.GOARCH == "arm64" && len(embeddedWhisperDarwinARM64) > 0)
		return false // Noch nicht verfügbar
	}
	return false
}

// GetEmbeddedWhisperInfo gibt Infos über verfügbare eingebettete Binaries zurück
func GetEmbeddedWhisperInfo() map[string]int {
	return map[string]int{
		"linux-amd64":  len(embeddedWhisperLinuxAMD64),
		// "windows-amd64": len(embeddedWhisperWindowsAMD64),
		// "darwin-amd64":  len(embeddedWhisperDarwinAMD64),
		// "darwin-arm64":  len(embeddedWhisperDarwinARM64),
	}
}
