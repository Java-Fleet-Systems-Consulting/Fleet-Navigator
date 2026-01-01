package main

import "testing"

// TestDetermineFileType testet die Dateityp-Erkennung
func TestDetermineFileType(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		contentType string
		expected    string
	}{
		// Bilder nach Dateiendung
		{"PNG by extension", "test.png", "", "image"},
		{"JPG by extension", "photo.jpg", "", "image"},
		{"JPEG by extension", "photo.jpeg", "", "image"},
		{"GIF by extension", "animation.gif", "", "image"},
		{"WebP by extension", "image.webp", "", "image"},
		{"BMP by extension", "bitmap.bmp", "", "image"},
		{"TIFF by extension", "scan.tiff", "", "image"},
		{"TIF by extension", "scan.tif", "", "image"},

		// Bilder mit Großbuchstaben
		{"PNG uppercase", "TEST.PNG", "", "image"},
		{"JPG uppercase", "PHOTO.JPG", "", "image"},
		{"Mixed case", "Photo.JpG", "", "image"},

		// Bilder nach Content-Type (Fallback)
		{"Image by content-type", "unknown", "image/png", "image"},
		{"Image by content-type jpeg", "file", "image/jpeg", "image"},
		{"Image by content-type webp", "file", "image/webp", "image"},

		// PDF
		{"PDF by extension", "document.pdf", "", "pdf"},
		{"PDF by content-type", "file", "application/pdf", "pdf"},

		// Text
		{"TXT by extension", "readme.txt", "", "text"},
		{"MD by extension", "README.md", "", "text"},
		{"Markdown by extension", "doc.markdown", "", "text"},
		{"Text by content-type", "file", "text/plain", "text"},
		{"Text HTML content-type", "file", "text/html", "text"},

		// Office Dokumente
		{"DOCX by extension", "document.docx", "", "docx"},
		{"DOCX by content-type", "file", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "docx"},
		{"ODT by extension", "document.odt", "", "odt"},
		{"ODT by content-type", "file", "application/vnd.oasis.opendocument.text", "odt"},
		{"XLSX by extension", "spreadsheet.xlsx", "", "xlsx"},
		{"XLSX by content-type", "file", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "xlsx"},
		{"XLS by extension", "spreadsheet.xls", "", "xls"},
		{"CSV by extension", "data.csv", "", "csv"},

		// Andere Formate
		{"HTML by extension", "page.html", "", "html"},
		{"HTM by extension", "page.htm", "", "html"},
		{"JSON by extension", "config.json", "", "json"},
		{"XML by extension", "data.xml", "", "xml"},
		{"EML by extension", "email.eml", "", "eml"},

		// Unbekannte Typen
		{"Unknown extension", "file.xyz", "", "unknown"},
		{"No extension", "file", "", "unknown"},
		{"Empty filename", "", "", "unknown"},
		{"Unknown with unknown content-type", "file", "application/octet-stream", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetermineFileType(tt.filename, tt.contentType)
			if result != tt.expected {
				t.Errorf("DetermineFileType(%q, %q) = %q, expected %q",
					tt.filename, tt.contentType, result, tt.expected)
			}
		})
	}
}

// TestDetermineFileType_ExtensionPriority testet dass Dateiendung Vorrang vor Content-Type hat
func TestDetermineFileType_ExtensionPriority(t *testing.T) {
	// Wenn Dateiendung bekannt ist, sollte sie Vorrang haben
	result := DetermineFileType("image.png", "application/pdf")
	if result != "image" {
		t.Errorf("Extension should take priority: got %q, expected 'image'", result)
	}

	result = DetermineFileType("document.pdf", "image/png")
	if result != "pdf" {
		t.Errorf("Extension should take priority: got %q, expected 'pdf'", result)
	}
}

// TestDetermineFileType_AllImageFormats testet alle unterstützten Bildformate
func TestDetermineFileType_AllImageFormats(t *testing.T) {
	imageExtensions := []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".tiff", ".tif"}

	for _, ext := range imageExtensions {
		result := DetermineFileType("test"+ext, "")
		if result != "image" {
			t.Errorf("Extension %s should be detected as image, got %q", ext, result)
		}
	}
}

func TestIsIdentityOrPersonalQuestion(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected bool
	}{
		// Identitätsfragen - sollten true sein
		{"Wer bist du - deutsch", "Wer bist du?", true},
		{"Wer bist du - lowercase", "wer bist du", true},
		{"Wer sind Sie - formal", "Wer sind Sie?", true},
		{"Was bist du", "Was bist du eigentlich?", true},
		{"Stell dich vor", "Stell dich vor!", true},
		{"Stelle dich vor", "Stelle dich vor!", true},
		{"Erzähl von dir", "Erzähl mir von dir", true},
		{"Beschreib dich", "Beschreib dich mal", true},
		{"Who are you - englisch", "Who are you?", true},
		{"Sen kimsin - türkisch", "Sen kimsin?", true},
		{"Siz kimsiniz - türkisch formal", "Siz kimsiniz?", true},

		// Persönliche Fragen - sollten true sein
		{"Wie geht es dir", "Wie geht es dir?", true},
		{"Wie gehts dir - umgangssprache", "wie gehts dir so?", true},
		{"How are you", "How are you doing?", true},
		{"Nasılsın - türkisch", "Nasılsın?", true},

		// Technische Selbstfragen - sollten true sein
		{"Welches Modell", "Welches Modell bist du?", true},
		{"Auf welchem Modell basierst du", "Auf welchem Modell basierst du?", true},
		{"Du läufst auf CPU", "du läufst nur auf der CPU welches Modell", true},
		{"What model", "What model are you based on?", true},
		{"Which model", "Which model powers you?", true},
		{"Worauf basierst du", "Worauf basierst du eigentlich?", true},
		{"Welches LLM", "Welches LLM verwendest du?", true},
		{"Bist du aufgebaut", "Auf was bist du aufgebaut?", true},

		// Normale Fragen - sollten false sein
		{"Rechtsfrage", "Was muss ich bei einem Mietvertrag beachten?", false},
		{"Technikfrage", "Wie installiere ich Python?", false},
		{"Allgemeine Frage", "Was ist die Hauptstadt von Deutschland?", false},
		{"Wetter", "Wie wird das Wetter morgen?", false},
		{"Aktuelle Nachrichten", "Was sind die Nachrichten heute?", false},
		{"Preisfrage", "Was kostet ein iPhone?", false},
		{"GPU Frage allgemein", "Welche GPU ist die beste für Gaming?", false},
		{"Modell Frage allgemein", "Welches Auto-Modell ist empfehlenswert?", false},
		{"Leere Nachricht", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isIdentityOrPersonalQuestion(tt.message)
			if result != tt.expected {
				t.Errorf("isIdentityOrPersonalQuestion(%q) = %v, expected %v", tt.message, result, tt.expected)
			}
		})
	}
}
