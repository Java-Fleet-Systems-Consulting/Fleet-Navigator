package prompts

import (
	"strings"
	"testing"
)

// TestAntiHallucinationSuffixExists prüft dass die Anti-Halluzinations-Konstante existiert
func TestAntiHallucinationSuffixExists(t *testing.T) {
	if AntiHallucinationSuffix == "" {
		t.Error("AntiHallucinationSuffix sollte nicht leer sein")
	}

	// Prüfe wichtige Bestandteile
	requiredPhrases := []string{
		"KEINE HALLUZINATIONEN",
		"Erfinde NIEMALS",
		"nicht weisst",
		"Unterscheide KLAR",
		"Lieber zugeben",
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(AntiHallucinationSuffix, phrase) {
			t.Errorf("AntiHallucinationSuffix fehlt: '%s'", phrase)
		}
	}
}

// TestAntiHallucinationSuffixENExists prüft die englische Version
func TestAntiHallucinationSuffixENExists(t *testing.T) {
	if AntiHallucinationSuffixEN == "" {
		t.Error("AntiHallucinationSuffixEN sollte nicht leer sein")
	}

	// Prüfe wichtige Bestandteile
	requiredPhrases := []string{
		"NO HALLUCINATIONS",
		"NEVER invent",
		"don't know",
		"CLEARLY distinguish",
		"Better to admit",
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(AntiHallucinationSuffixEN, phrase) {
			t.Errorf("AntiHallucinationSuffixEN fehlt: '%s'", phrase)
		}
	}
}

// TestAntiHallucinationSuffixesDiffer prüft dass DE und EN unterschiedlich sind
func TestAntiHallucinationSuffixesDiffer(t *testing.T) {
	if AntiHallucinationSuffix == AntiHallucinationSuffixEN {
		t.Error("Deutsche und englische Anti-Halluzinations-Suffixe sollten unterschiedlich sein")
	}
}

// TestDetectGermanLocale prüft die Locale-Erkennung
func TestDetectGermanLocale(t *testing.T) {
	// Die Funktion verwendet os.Getenv, daher nur Struktur-Test
	// Wir prüfen nur dass die Funktion existiert und aufrufbar ist
	result := detectGermanLocale()
	// Ergebnis hängt von der Umgebung ab, daher kein Assertation
	_ = result
}

// TestNewService prüft die Service-Erstellung
func TestNewService(t *testing.T) {
	// Ohne Repository kann kein vollständiger Service erstellt werden
	// Aber wir können prüfen dass NewService mit nil nicht panickt
	service := NewService(nil)
	if service == nil {
		t.Error("NewService sollte einen Service zurückgeben")
	}
}
