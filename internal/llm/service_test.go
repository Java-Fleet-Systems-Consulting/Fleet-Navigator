package llm

import (
	"strings"
	"testing"
)

// TestDefaultSystemPromptHasAntiHallucination prüft dass der System-Prompt
// Anti-Halluzinations-Anweisungen enthält
func TestDefaultSystemPromptHasAntiHallucination(t *testing.T) {
	config := DefaultModelServiceConfig()

	// Muss Anti-Halluzinations-Anweisungen enthalten
	requiredPhrases := []string{
		"KEINE HALLUZINATIONEN",
		"Erfinde NIEMALS",
		"nicht weisst",
		"Zitiere KEINE",
		"Unsicherheit",
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(config.SystemPrompt, phrase) {
			t.Errorf("System-Prompt fehlt Anti-Halluzinations-Phrase: '%s'", phrase)
		}
	}
}

// TestDefaultSystemPromptIsGerman prüft dass der System-Prompt auf Deutsch ist
func TestDefaultSystemPromptIsGerman(t *testing.T) {
	config := DefaultModelServiceConfig()

	// Deutsche Sprache-Anweisung muss vorhanden sein
	if !strings.Contains(config.SystemPrompt, "AUSSCHLIESSLICH auf Deutsch") {
		t.Error("System-Prompt fehlt Deutsche Sprach-Anweisung")
	}

	// Muss "Niemals auf Chinesisch" enthalten
	if !strings.Contains(config.SystemPrompt, "Niemals auf Chinesisch") {
		t.Error("System-Prompt fehlt Anweisung gegen Chinesisch")
	}
}

// TestDefaultSystemPromptContainsRole prüft die Rollenbeschreibung
func TestDefaultSystemPromptContainsRole(t *testing.T) {
	config := DefaultModelServiceConfig()

	// Fleet Navigator Rolle
	if !strings.Contains(config.SystemPrompt, "Fleet Navigator") {
		t.Error("System-Prompt fehlt Fleet Navigator Rolle")
	}

	// Büro-Assistent Beschreibung
	if !strings.Contains(config.SystemPrompt, "Bueroaufgaben") {
		t.Error("System-Prompt fehlt Büro-Aufgaben Beschreibung")
	}
}

// TestDefaultSystemPromptTasks prüft die Aufgabenbeschreibung
func TestDefaultSystemPromptTasks(t *testing.T) {
	config := DefaultModelServiceConfig()

	tasks := []string{
		"Textverarbeitung",
		"E-Mail",
		"Terminplanung",
		"Datenschutz",
		"Marketing",
	}

	for _, task := range tasks {
		if !strings.Contains(config.SystemPrompt, task) {
			t.Errorf("System-Prompt fehlt Aufgabe: '%s'", task)
		}
	}
}

// TestModelServiceCreation prüft die Service-Erstellung
func TestModelServiceCreation(t *testing.T) {
	config := DefaultModelServiceConfig()
	service := NewModelService(config)

	if service == nil {
		t.Fatal("NewModelService sollte nicht nil zurückgeben")
	}

	if service.systemPrompt == "" {
		t.Error("Service sollte System-Prompt haben")
	}

	if service.defaultModel == "" {
		t.Error("Service sollte Default-Modell haben")
	}
}

// TestDefaultModelConfig prüft die Standard-Konfiguration
func TestDefaultModelConfig(t *testing.T) {
	config := DefaultModelServiceConfig()

	// Ollama URL prüfen
	if config.OllamaURL == "" {
		t.Error("OllamaURL sollte nicht leer sein")
	}

	// Default Model prüfen
	if config.DefaultModel == "" {
		t.Error("DefaultModel sollte nicht leer sein")
	}

	// System Prompt prüfen
	if len(config.SystemPrompt) < 100 {
		t.Error("SystemPrompt sollte mindestens 100 Zeichen haben")
	}
}

// TestAntiHallucinationInstructions prüft die spezifischen Anti-Halluzinations-Anweisungen
func TestAntiHallucinationInstructions(t *testing.T) {
	config := DefaultModelServiceConfig()

	testCases := []struct {
		name     string
		contains string
	}{
		{"Nicht erfinden", "Erfinde NIEMALS"},
		{"Ehrlich zugeben", "Das weiss ich leider nicht"},
		{"Keine falschen Quellen", "Zitiere KEINE Webseiten"},
		{"Fakten vs Vermutungen", "Unterscheide klar zwischen Fakten"},
		{"Lieber zugeben", "Lieber zugeben als raten"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(config.SystemPrompt, tc.contains) {
				t.Errorf("System-Prompt fehlt: '%s'", tc.contains)
			}
		})
	}
}
