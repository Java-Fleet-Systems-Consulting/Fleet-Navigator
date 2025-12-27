package experte

import (
	"strings"
	"testing"
)

// TestDefaultAntiHallucinationPromptExists prüft dass die Anti-Halluzinations-Konstante existiert
func TestDefaultAntiHallucinationPromptExists(t *testing.T) {
	if DefaultAntiHallucinationPrompt == "" {
		t.Error("DefaultAntiHallucinationPrompt sollte nicht leer sein")
	}

	// Prüfe wichtige Bestandteile
	requiredPhrases := []string{
		"KEINE HALLUZINATIONEN",
		"Erfinde NIEMALS",
		"nicht weisst",
		"Unterscheide KLAR",
		"Lieber zugeben",
		"SELBSTWAHRNEHMUNG", // Selbstwahrnehmungs-Schutz (was LLM über sich weiß)
		"QUELLEN",          // Quellen-Relevanz
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(DefaultAntiHallucinationPrompt, phrase) {
			t.Errorf("DefaultAntiHallucinationPrompt fehlt: '%s'", phrase)
		}
	}
}

// TestGetFullPromptIncludesAntiHallucination prüft dass GetFullPrompt immer Anti-Halluzinations-Regeln enthält
func TestGetFullPromptIncludesAntiHallucination(t *testing.T) {
	expert := Expert{
		Name:       "Test Expert",
		BasePrompt: "Du bist ein Test-Experte.",
	}

	prompt := expert.GetFullPrompt(nil)

	if !strings.Contains(prompt, "KEINE HALLUZINATIONEN") {
		t.Error("GetFullPrompt sollte Anti-Halluzinations-Regeln enthalten")
	}

	if !strings.Contains(prompt, "Erfinde NIEMALS") {
		t.Error("GetFullPrompt sollte 'Erfinde NIEMALS' enthalten")
	}
}

// TestGetFullPromptWithWebSearch prüft Quellen-Regeln bei aktivierter Websuche
func TestGetFullPromptWithWebSearch(t *testing.T) {
	expert := Expert{
		Name:               "Test Expert",
		BasePrompt:         "Du bist ein Test-Experte.",
		AutoWebSearch:      true,
		WebSearchShowLinks: true,
	}

	prompt := expert.GetFullPrompt(nil)

	// Muss Anti-Halluzinations-Regeln haben
	if !strings.Contains(prompt, "KEINE HALLUZINATIONEN") {
		t.Error("Sollte Anti-Halluzinations-Regeln enthalten")
	}

	// Muss Web-Suche Quellen-Regeln haben
	if !strings.Contains(prompt, "Quellen-Regel") {
		t.Error("Sollte Quellen-Regeln enthalten")
	}

	if !strings.Contains(prompt, "Web-Suche") {
		t.Error("Sollte Web-Suche Hinweis enthalten")
	}
}

// TestGetFullPromptWithoutWebSearch prüft Quellen-Regeln ohne Websuche
func TestGetFullPromptWithoutWebSearch(t *testing.T) {
	expert := Expert{
		Name:          "Test Expert",
		BasePrompt:    "Du bist ein Test-Experte.",
		AutoWebSearch: false,
	}

	prompt := expert.GetFullPrompt(nil)

	// Muss Anti-Halluzinations-Regeln haben
	if !strings.Contains(prompt, "KEINE HALLUZINATIONEN") {
		t.Error("Sollte Anti-Halluzinations-Regeln enthalten")
	}

	// Muss Hinweis auf fehlende Websuche haben
	if !strings.Contains(prompt, "Keine Web-Suche") {
		t.Error("Sollte Hinweis auf fehlende Web-Suche enthalten")
	}
}

// TestGetFullPromptWithMode prüft dass Modi korrekt eingefügt werden
func TestGetFullPromptWithMode(t *testing.T) {
	expert := Expert{
		Name:       "Test Expert",
		BasePrompt: "Du bist ein Test-Experte.",
	}

	mode := &ExpertMode{
		Name:   "Testmodus",
		Prompt: "Fokussiere auf Tests.",
	}

	prompt := expert.GetFullPrompt(mode)

	// Base-Prompt muss enthalten sein
	if !strings.Contains(prompt, "Du bist ein Test-Experte") {
		t.Error("Sollte Base-Prompt enthalten")
	}

	// Mode-Name muss enthalten sein
	if !strings.Contains(prompt, "Testmodus") {
		t.Error("Sollte Mode-Namen enthalten")
	}

	// Mode-Prompt muss enthalten sein
	if !strings.Contains(prompt, "Fokussiere auf Tests") {
		t.Error("Sollte Mode-Prompt enthalten")
	}

	// Anti-Halluzinations-Regeln müssen NACH dem Mode kommen
	if !strings.Contains(prompt, "KEINE HALLUZINATIONEN") {
		t.Error("Sollte Anti-Halluzinations-Regeln enthalten")
	}
}

// TestDefaultExpertsHaveAntiHallucination prüft alle Default-Experten
func TestDefaultExpertsHaveAntiHallucination(t *testing.T) {
	experts := DefaultExperts()

	if len(experts) == 0 {
		t.Fatal("DefaultExperts sollte mindestens einen Experten zurückgeben")
	}

	for _, expert := range experts {
		t.Run(expert.Name, func(t *testing.T) {
			prompt := expert.GetFullPrompt(nil)

			if !strings.Contains(prompt, "KEINE HALLUZINATIONEN") {
				t.Errorf("Experte '%s' sollte Anti-Halluzinations-Regeln haben", expert.Name)
			}

			if !strings.Contains(prompt, "Erfinde NIEMALS") {
				t.Errorf("Experte '%s' sollte 'Erfinde NIEMALS' enthalten", expert.Name)
			}
		})
	}
}

// TestDefaultExpertsCount prüft die Anzahl der Default-Experten
func TestDefaultExpertsCount(t *testing.T) {
	experts := DefaultExperts()

	// Erwartete Experten: Ewa, Roland, Ayşe, Luca, Franziska, Dr. Sol
	expected := 6
	if len(experts) != expected {
		t.Errorf("DefaultExperts sollte %d Experten haben, hat aber %d", expected, len(experts))
	}
}

// TestDefaultExpertsHaveNames prüft dass alle Experten Namen haben
func TestDefaultExpertsHaveNames(t *testing.T) {
	experts := DefaultExperts()

	expectedNames := []string{
		"Ewa Marek",
		"Roland Navarro",
		"Ayşe Yılmaz",
		"Luca Santoro",
		"Franziska Berger",
		"Dr. Sol Bashari",
	}

	for i, name := range expectedNames {
		if i >= len(experts) {
			t.Errorf("Experte %d (%s) nicht gefunden", i, name)
			continue
		}
		if experts[i].Name != name {
			t.Errorf("Experte %d sollte '%s' heißen, heißt aber '%s'", i, name, experts[i].Name)
		}
	}
}

// TestDefaultExpertsHaveBasePrompts prüft dass alle Experten Base-Prompts haben
func TestDefaultExpertsHaveBasePrompts(t *testing.T) {
	experts := DefaultExperts()

	for _, expert := range experts {
		if expert.BasePrompt == "" {
			t.Errorf("Experte '%s' hat keinen BasePrompt", expert.Name)
		}
		if len(expert.BasePrompt) < 100 {
			t.Errorf("Experte '%s' hat einen zu kurzen BasePrompt (%d Zeichen)", expert.Name, len(expert.BasePrompt))
		}
	}
}

// TestDefaultExpertsHaveCharacterProtection prüft dass alle Experten Charakterschutz haben
func TestDefaultExpertsHaveCharacterProtection(t *testing.T) {
	experts := DefaultExperts()

	for _, expert := range experts {
		t.Run(expert.Name, func(t *testing.T) {
			if !strings.Contains(expert.BasePrompt, "CHARAKTERSCHUTZ") {
				t.Errorf("Experte '%s' sollte CHARAKTERSCHUTZ haben", expert.Name)
			}
		})
	}
}

// TestGetDefaultMode prüft die Default-Mode Funktion
func TestGetDefaultMode(t *testing.T) {
	expert := Expert{
		Name: "Test",
		Modes: []ExpertMode{
			{Name: "Mode1", IsDefault: false},
			{Name: "Mode2", IsDefault: true},
			{Name: "Mode3", IsDefault: false},
		},
	}

	mode := expert.GetDefaultMode()
	if mode == nil {
		t.Fatal("GetDefaultMode sollte einen Mode zurückgeben")
	}

	if mode.Name != "Mode2" {
		t.Errorf("GetDefaultMode sollte 'Mode2' zurückgeben, gab '%s'", mode.Name)
	}
}

// TestDetectModeByKeywords prüft die Keyword-Erkennung
func TestDetectModeByKeywords(t *testing.T) {
	expert := Expert{
		Name: "Test",
		Modes: []ExpertMode{
			{Name: "Strafrecht", Keywords: []string{"strafrecht", "anzeige", "polizei"}},
			{Name: "Mietrecht", Keywords: []string{"miete", "vermieter", "wohnung"}},
		},
	}

	testCases := []struct {
		text     string
		expected string
	}{
		{"Ich wurde bei der Polizei angezeigt", "Strafrecht"},
		{"Mein Vermieter will die Miete erhöhen", "Mietrecht"},
		{"Allgemeine Frage", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.text, func(t *testing.T) {
			mode := expert.DetectModeByKeywords(tc.text)
			if tc.expected == "" {
				if mode != nil {
					t.Errorf("Sollte nil sein für '%s', war '%s'", tc.text, mode.Name)
				}
			} else {
				if mode == nil {
					t.Errorf("Sollte '%s' erkennen für '%s'", tc.expected, tc.text)
				} else if mode.Name != tc.expected {
					t.Errorf("Sollte '%s' erkennen für '%s', war '%s'", tc.expected, tc.text, mode.Name)
				}
			}
		})
	}
}

// TestCustomAntiHallucinationPrompt prüft dass ein Custom-Prompt den Default überschreibt
func TestCustomAntiHallucinationPrompt(t *testing.T) {
	customPrompt := "## MEIN CUSTOM PROMPT\n- Regel 1\n- Regel 2"

	expert := Expert{
		Name:                    "Test Expert",
		BasePrompt:              "Du bist ein Test-Experte.",
		AntiHallucinationPrompt: customPrompt,
	}

	prompt := expert.GetFullPrompt(nil)

	// Custom-Prompt muss enthalten sein
	if !strings.Contains(prompt, "MEIN CUSTOM PROMPT") {
		t.Error("GetFullPrompt sollte Custom-AntiHallucinationPrompt enthalten")
	}

	// Default-Prompt darf NICHT enthalten sein (prüfe auf spezifischen Text)
	if strings.Contains(prompt, "KRITISCH - KEINE HALLUZINATIONEN") {
		t.Error("GetFullPrompt sollte NICHT den Default-Prompt enthalten wenn Custom gesetzt")
	}
}

// TestDefaultAntiHallucinationPromptUsedWhenEmpty prüft dass Default verwendet wird wenn Custom leer
func TestDefaultAntiHallucinationPromptUsedWhenEmpty(t *testing.T) {
	expert := Expert{
		Name:                    "Test Expert",
		BasePrompt:              "Du bist ein Test-Experte.",
		AntiHallucinationPrompt: "", // Leer = Default verwenden
	}

	prompt := expert.GetFullPrompt(nil)

	// Default-Prompt muss enthalten sein
	if !strings.Contains(prompt, "KRITISCH - KEINE HALLUZINATIONEN") {
		t.Error("GetFullPrompt sollte Default-AntiHallucinationPrompt enthalten wenn Custom leer")
	}
}

// TestDefaultExpertsWebSearchShowLinksIsFalse prüft dass alle Default-Experten WebSearchShowLinks=false haben
func TestDefaultExpertsWebSearchShowLinksIsFalse(t *testing.T) {
	experts := DefaultExperts()

	for _, expert := range experts {
		t.Run(expert.Name, func(t *testing.T) {
			if expert.WebSearchShowLinks {
				t.Errorf("Experte '%s' sollte WebSearchShowLinks=false haben (RAG-Modus Default)", expert.Name)
			}
		})
	}
}

// TestDefaultExpertsHaveAutoWebSearch prüft dass alle Default-Experten AutoWebSearch aktiviert haben
func TestDefaultExpertsHaveAutoWebSearch(t *testing.T) {
	experts := DefaultExperts()

	for _, expert := range experts {
		t.Run(expert.Name, func(t *testing.T) {
			if !expert.AutoWebSearch {
				t.Errorf("Experte '%s' sollte AutoWebSearch=true haben", expert.Name)
			}
		})
	}
}

// TestGetFullPromptRAGMode prüft die Quellen-Regeln im RAG-Modus (WebSearchShowLinks=false)
func TestGetFullPromptRAGMode(t *testing.T) {
	expert := Expert{
		Name:               "Test Expert",
		BasePrompt:         "Du bist ein Test-Experte.",
		AutoWebSearch:      true,
		WebSearchShowLinks: false, // RAG-Modus
	}

	prompt := expert.GetFullPrompt(nil)

	// RAG-Modus Regeln
	if !strings.Contains(prompt, "RAG-Modus") {
		t.Error("Sollte RAG-Modus Hinweis enthalten")
	}

	if !strings.Contains(prompt, "KEINE Quellenverweise") {
		t.Error("Sollte 'KEINE Quellenverweise' im RAG-Modus enthalten")
	}
}
