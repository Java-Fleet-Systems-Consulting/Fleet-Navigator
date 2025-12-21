package llamaserver

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

//go:embed bin/linux-amd64/llama-server
var embeddedLlamaServerLinuxAMD64 []byte

// Platzhalter für andere Plattformen (werden später hinzugefügt)
// //go:embed bin/windows-amd64/llama-server.exe
// var embeddedLlamaServerWindowsAMD64 []byte
// //go:embed bin/darwin-amd64/llama-server
// var embeddedLlamaServerDarwinAMD64 []byte
// //go:embed bin/darwin-arm64/llama-server
// var embeddedLlamaServerDarwinARM64 []byte

// ExtractEmbeddedLlamaServer extrahiert das eingebettete llama-server Binary für die aktuelle Plattform
func ExtractEmbeddedLlamaServer(destDir string) (string, error) {
	var binaryData []byte
	var binaryName string

	switch runtime.GOOS {
	case "linux":
		if runtime.GOARCH == "amd64" {
			binaryData = embeddedLlamaServerLinuxAMD64
			binaryName = "llama-server"
		}
	case "windows":
		if runtime.GOARCH == "amd64" {
			// binaryData = embeddedLlamaServerWindowsAMD64
			binaryName = "llama-server.exe"
		}
	case "darwin":
		if runtime.GOARCH == "amd64" {
			// binaryData = embeddedLlamaServerDarwinAMD64
			binaryName = "llama-server"
		} else if runtime.GOARCH == "arm64" {
			// binaryData = embeddedLlamaServerDarwinARM64
			binaryName = "llama-server"
		}
	}

	if len(binaryData) == 0 {
		return "", fmt.Errorf("kein eingebettetes llama-server Binary für %s/%s verfügbar", runtime.GOOS, runtime.GOARCH)
	}

	destPath := filepath.Join(destDir, binaryName)

	// Prüfen ob bereits vorhanden und gleiche Größe
	if info, err := os.Stat(destPath); err == nil {
		if info.Size() == int64(len(binaryData)) {
			log.Printf("llama-server Binary bereits vorhanden: %s", destPath)
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

	log.Printf("llama-server Binary extrahiert: %s (%d bytes)", destPath, len(binaryData))
	return destPath, nil
}

// HasEmbeddedLlamaServer prüft ob ein eingebettetes Binary für das aktuelle OS verfügbar ist
func HasEmbeddedLlamaServer() bool {
	switch runtime.GOOS {
	case "linux":
		return runtime.GOARCH == "amd64" && len(embeddedLlamaServerLinuxAMD64) > 0
	case "windows":
		// return runtime.GOARCH == "amd64" && len(embeddedLlamaServerWindowsAMD64) > 0
		return false // Noch nicht verfügbar
	case "darwin":
		// return (runtime.GOARCH == "amd64" && len(embeddedLlamaServerDarwinAMD64) > 0) ||
		//        (runtime.GOARCH == "arm64" && len(embeddedLlamaServerDarwinARM64) > 0)
		return false // Noch nicht verfügbar
	}
	return false
}

// GetEmbeddedLlamaServerInfo gibt Infos über verfügbare eingebettete Binaries zurück
func GetEmbeddedLlamaServerInfo() map[string]int {
	return map[string]int{
		"linux-amd64": len(embeddedLlamaServerLinuxAMD64),
		// "windows-amd64": len(embeddedLlamaServerWindowsAMD64),
		// "darwin-amd64":  len(embeddedLlamaServerDarwinAMD64),
		// "darwin-arm64":  len(embeddedLlamaServerDarwinARM64),
	}
}

// hasLlamaLibraries prüft ob llama.cpp Libraries (libmtmd.so, libggml.so, etc.) in einem Verzeichnis vorhanden sind
// Diese werden für Vision/Multimodal-Unterstützung benötigt
func hasLlamaLibraries(dir string) bool {
	// Prüfe auf libmtmd.so* (multimodal library - kritisch für Vision)
	pattern := filepath.Join(dir, "libmtmd.so*")
	matches, err := filepath.Glob(pattern)
	if err == nil && len(matches) > 0 {
		return true
	}
	// Alternativ: libggml.so* (Basis-Library)
	pattern = filepath.Join(dir, "libggml.so*")
	matches, err = filepath.Glob(pattern)
	return err == nil && len(matches) > 0
}

// GetOrExtractLlamaServer sucht nach llama-server Binary oder extrahiert es aus eingebetteten Daten
// Gibt den Pfad zur Binary und den Library-Pfad zurück
// WICHTIG: Sucht IMMER ZUERST im dataDir/bin/ Ordner - das ist der Ort für heruntergeladene Binaries
// WICHTIG: Der Library-Pfad wird SEPARAT gesucht - Libraries können im bundled Verzeichnis liegen,
//          auch wenn das Binary woanders liegt (z.B. in dataDir/bin/)
func GetOrExtractLlamaServer(dataDir string) (binaryPath string, libraryPath string, err error) {
	// Binär-Name je nach OS
	var binaryName string
	if runtime.GOOS == "windows" {
		binaryName = "llama-server.exe"
	} else {
		binaryName = "llama-server"
	}

	var foundBinaryPath string
	var foundLibraryPath string

	// PRIORITÄT 1: Im dataDir/bin/ Ordner suchen (Setup-Download-Ort)
	binDir := filepath.Join(dataDir, "bin")
	downloadedPath := filepath.Join(binDir, binaryName)
	if info, err := os.Stat(downloadedPath); err == nil && info.Mode().IsRegular() {
		log.Printf("✅ llama-server gefunden (Setup-Download): %s", downloadedPath)
		foundBinaryPath = downloadedPath
		// Prüfe ob Libraries hier auch vorhanden sind
		if hasLlamaLibraries(binDir) {
			foundLibraryPath = binDir
			log.Printf("📚 Libraries gefunden (dataDir): %s", binDir)
		}
	}

	// PRIORITÄT 2: Neben dem Fleet Navigator Binary (für portable Builds)
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		bundledPath := filepath.Join(execDir, "bin", binaryName)
		bundledBinDir := filepath.Join(execDir, "bin")

		// Binary suchen (falls noch nicht gefunden)
		if foundBinaryPath == "" {
			if info, statErr := os.Stat(bundledPath); statErr == nil && info.Mode().IsRegular() {
				log.Printf("✅ llama-server gefunden (bundled): %s", bundledPath)
				foundBinaryPath = bundledPath
			}
		}

		// Library-Pfad suchen (falls noch nicht gefunden)
		if foundLibraryPath == "" && hasLlamaLibraries(bundledBinDir) {
			foundLibraryPath = bundledBinDir
			log.Printf("📚 Libraries gefunden (bundled): %s", bundledBinDir)
		}
	}

	// PRIORITÄT 3: Eingebettetes Binary extrahieren (Linux)
	if foundBinaryPath == "" && HasEmbeddedLlamaServer() {
		extractedPath, err := ExtractEmbeddedLlamaServer(binDir)
		if err != nil {
			return "", "", fmt.Errorf("llama-server extrahieren: %w", err)
		}
		foundBinaryPath = extractedPath
	}

	// Kein Binary gefunden?
	if foundBinaryPath == "" {
		log.Printf("⚠️ llama-server nicht gefunden in: %s", downloadedPath)
		return "", "", fmt.Errorf("kein llama-server Binary gefunden in %s", binDir)
	}

	// Library-Pfad: Fallback auf Binary-Verzeichnis wenn keine Libraries gefunden
	if foundLibraryPath == "" {
		foundLibraryPath = filepath.Dir(foundBinaryPath)
		log.Printf("⚠️ Keine Libraries gefunden, verwende Binary-Verzeichnis: %s", foundLibraryPath)
	}

	return foundBinaryPath, foundLibraryPath, nil
}
