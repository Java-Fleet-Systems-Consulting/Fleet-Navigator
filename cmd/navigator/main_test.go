package main

import "testing"

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
