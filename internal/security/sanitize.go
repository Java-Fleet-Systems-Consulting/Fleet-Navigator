// Package security bietet Sicherheitsfunktionen für Fleet Navigator
package security

import (
	"path/filepath"
	"regexp"
	"strings"
)

// AllowedAudioFormats enthält die erlaubten Audio-Dateiformate
var AllowedAudioFormats = map[string]bool{
	"wav":  true,
	"mp3":  true,
	"ogg":  true,
	"webm": true,
	"m4a":  true,
	"flac": true,
	"aac":  true,
	"opus": true,
}

// AllowedImageFormats enthält die erlaubten Bild-Dateiformate
var AllowedImageFormats = map[string]bool{
	"png":  true,
	"jpg":  true,
	"jpeg": true,
	"gif":  true,
	"webp": true,
	"bmp":  true,
	"tiff": true,
}

// AllowedDocumentFormats enthält die erlaubten Dokument-Dateiformate
var AllowedDocumentFormats = map[string]bool{
	"pdf":  true,
	"docx": true,
	"doc":  true,
	"odt":  true,
	"txt":  true,
	"md":   true,
	"rtf":  true,
	"xlsx": true,
	"xls":  true,
	"csv":  true,
	"html": true,
	"htm":  true,
	"xml":  true,
	"json": true,
	"eml":  true,
}

// ValidateAudioFormat prüft ob das Audio-Format erlaubt ist
// Gibt das validierte Format zurück oder "webm" als Default
func ValidateAudioFormat(format string) string {
	// Lowercase und Trim
	format = strings.ToLower(strings.TrimSpace(format))

	// Entferne führenden Punkt falls vorhanden
	format = strings.TrimPrefix(format, ".")

	// Prüfe auf erlaubtes Format
	if AllowedAudioFormats[format] {
		return format
	}

	// Default Format
	return "webm"
}

// ValidateImageFormat prüft ob das Bild-Format erlaubt ist
// Gibt das validierte Format zurück oder "png" als Default
func ValidateImageFormat(format string) string {
	format = strings.ToLower(strings.TrimSpace(format))
	format = strings.TrimPrefix(format, ".")

	if AllowedImageFormats[format] {
		return format
	}

	return "png"
}

// ValidateDocumentFormat prüft ob das Dokument-Format erlaubt ist
// Gibt das validierte Format zurück oder "txt" als Default
func ValidateDocumentFormat(format string) string {
	format = strings.ToLower(strings.TrimSpace(format))
	format = strings.TrimPrefix(format, ".")

	if AllowedDocumentFormats[format] {
		return format
	}

	return "txt"
}

// SanitizeFilename bereinigt einen Dateinamen von gefährlichen Zeichen
// Entfernt Path-Traversal-Versuche und Shell-Metazeichen
func SanitizeFilename(filename string) string {
	// Ersetze Windows-Backslashes durch Forward-Slashes
	filename = strings.ReplaceAll(filename, "\\", "/")

	// Nur den Basisnamen verwenden (entfernt Pfad-Komponenten)
	filename = filepath.Base(filename)

	// Gefährliche Zeichen entfernen
	// Erlaubt: Buchstaben, Zahlen, Punkt, Unterstrich, Bindestrich
	re := regexp.MustCompile(`[^a-zA-Z0-9._-]`)
	filename = re.ReplaceAllString(filename, "_")

	// Doppelte Punkte und Unterstriche entfernen
	for strings.Contains(filename, "..") {
		filename = strings.ReplaceAll(filename, "..", ".")
	}
	for strings.Contains(filename, "__") {
		filename = strings.ReplaceAll(filename, "__", "_")
	}

	// Führende Punkte/Unterstriche entfernen
	filename = strings.TrimLeft(filename, "._")

	// Leeren Dateinamen verhindern
	if filename == "" || filename == "." {
		filename = "unnamed"
	}

	return filename
}

// SanitizePath stellt sicher, dass ein Pfad innerhalb eines Basisverzeichnisses bleibt
// Gibt den bereinigten absoluten Pfad zurück oder einen Fehler
func SanitizePath(basePath, userPath string) (string, bool) {
	// Absolute Pfade vom User sind nicht erlaubt
	if filepath.IsAbs(userPath) {
		return "", false
	}

	// Ersetze Windows-Backslashes
	userPath = strings.ReplaceAll(userPath, "\\", "/")

	// Prüfe auf path traversal Muster
	if strings.Contains(userPath, "..") {
		return "", false
	}

	// Absoluten Basispfad ermitteln
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return "", false
	}

	// Pfad zusammensetzen und bereinigen
	fullPath := filepath.Join(absBase, userPath)
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", false
	}

	// Nochmals prüfen ob der Pfad innerhalb des Basisverzeichnisses liegt
	if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) && absPath != absBase {
		return "", false
	}

	return absPath, true
}

// IsValidLanguageCode prüft ob ein Sprachcode gültig ist (z.B. "de", "en", "tr")
func IsValidLanguageCode(code string) bool {
	validCodes := map[string]bool{
		"de":  true,
		"en":  true,
		"tr":  true,
		"fr":  true,
		"es":  true,
		"it":  true,
		"pt":  true,
		"nl":  true,
		"pl":  true,
		"ru":  true,
		"ja":  true,
		"zh":  true,
		"ko":  true,
		"ar":  true,
		"deu": true, // Tesseract Format
		"eng": true,
		"tur": true,
		"fra": true,
		"spa": true,
	}

	return validCodes[strings.ToLower(strings.TrimSpace(code))]
}
