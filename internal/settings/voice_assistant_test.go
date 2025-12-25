package settings

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestService erstellt einen Test-Service mit temporärer Datenbank
func setupTestService(t *testing.T) (*Service, func()) {
	t.Helper()

	// Temporäres Verzeichnis erstellen
	tempDir, err := os.MkdirTemp("", "voice_assistant_test")
	if err != nil {
		t.Fatalf("Fehler beim Erstellen des Temp-Verzeichnisses: %v", err)
	}

	// Repository und Service erstellen
	repo, err := NewRepository(tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Fehler beim Erstellen des Repositories: %v", err)
	}

	service := NewService(repo)

	// Cleanup-Funktion
	cleanup := func() {
		repo.Close()
		os.RemoveAll(tempDir)
	}

	return service, cleanup
}

// =============================================================================
// TESTS: Default Values
// =============================================================================

func TestVoiceAssistantEnabled_DefaultFalse(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	// Default sollte false sein
	enabled := service.GetVoiceAssistantEnabled()
	if enabled != false {
		t.Errorf("GetVoiceAssistantEnabled() = %v, want false", enabled)
	}
}

func TestVoiceAssistantWakeWord_DefaultHeyEwa(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	// Default sollte "hey_ewa" sein
	wakeWord := service.GetVoiceAssistantWakeWord()
	if wakeWord != WakeWordHeyEwa {
		t.Errorf("GetVoiceAssistantWakeWord() = %q, want %q", wakeWord, WakeWordHeyEwa)
	}
}

func TestVoiceAssistantAutoStop_DefaultTrue(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	// Default sollte true sein
	autoStop := service.GetVoiceAssistantAutoStop()
	if autoStop != true {
		t.Errorf("GetVoiceAssistantAutoStop() = %v, want true", autoStop)
	}
}

func TestVoiceAssistantQuietHoursEnabled_DefaultFalse(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	enabled := service.GetVoiceAssistantQuietHoursEnabled()
	if enabled != false {
		t.Errorf("GetVoiceAssistantQuietHoursEnabled() = %v, want false", enabled)
	}
}

func TestVoiceAssistantQuietHoursStart_Default2200(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	start := service.GetVoiceAssistantQuietHoursStart()
	if start != "22:00" {
		t.Errorf("GetVoiceAssistantQuietHoursStart() = %q, want %q", start, "22:00")
	}
}

func TestVoiceAssistantQuietHoursEnd_Default0700(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	end := service.GetVoiceAssistantQuietHoursEnd()
	if end != "07:00" {
		t.Errorf("GetVoiceAssistantQuietHoursEnd() = %q, want %q", end, "07:00")
	}
}

func TestVoiceAssistantCustomWakeWord_DefaultEmpty(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	customWakeWord := service.GetVoiceAssistantCustomWakeWord()
	if customWakeWord != "" {
		t.Errorf("GetVoiceAssistantCustomWakeWord() = %q, want empty string", customWakeWord)
	}
}

func TestVoiceAssistantTTSEnabled_DefaultTrue(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ttsEnabled := service.GetVoiceAssistantTTSEnabled()
	if ttsEnabled != true {
		t.Errorf("GetVoiceAssistantTTSEnabled() = %v, want true", ttsEnabled)
	}
}

// =============================================================================
// TESTS: Save and Get
// =============================================================================

func TestSaveVoiceAssistantEnabled(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	// Aktivieren
	err := service.SaveVoiceAssistantEnabled(true)
	if err != nil {
		t.Fatalf("SaveVoiceAssistantEnabled(true) error: %v", err)
	}

	enabled := service.GetVoiceAssistantEnabled()
	if enabled != true {
		t.Errorf("GetVoiceAssistantEnabled() after save = %v, want true", enabled)
	}

	// Deaktivieren
	err = service.SaveVoiceAssistantEnabled(false)
	if err != nil {
		t.Fatalf("SaveVoiceAssistantEnabled(false) error: %v", err)
	}

	enabled = service.GetVoiceAssistantEnabled()
	if enabled != false {
		t.Errorf("GetVoiceAssistantEnabled() after disable = %v, want false", enabled)
	}
}

func TestSaveVoiceAssistantWakeWord(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	tests := []struct {
		name     string
		wakeWord string
	}{
		{"hey_ewa", WakeWordHeyEwa},
		{"ewa", WakeWordEwa},
		{"custom", WakeWordCustom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.SaveVoiceAssistantWakeWord(tt.wakeWord)
			if err != nil {
				t.Fatalf("SaveVoiceAssistantWakeWord(%q) error: %v", tt.wakeWord, err)
			}

			got := service.GetVoiceAssistantWakeWord()
			if got != tt.wakeWord {
				t.Errorf("GetVoiceAssistantWakeWord() = %q, want %q", got, tt.wakeWord)
			}
		})
	}
}

func TestSaveVoiceAssistantCustomWakeWord(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	customWord := "Hallo Computer"

	err := service.SaveVoiceAssistantCustomWakeWord(customWord)
	if err != nil {
		t.Fatalf("SaveVoiceAssistantCustomWakeWord(%q) error: %v", customWord, err)
	}

	got := service.GetVoiceAssistantCustomWakeWord()
	if got != customWord {
		t.Errorf("GetVoiceAssistantCustomWakeWord() = %q, want %q", got, customWord)
	}
}

func TestSaveVoiceAssistantAutoStop(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	// Deaktivieren (Default ist true)
	err := service.SaveVoiceAssistantAutoStop(false)
	if err != nil {
		t.Fatalf("SaveVoiceAssistantAutoStop(false) error: %v", err)
	}

	autoStop := service.GetVoiceAssistantAutoStop()
	if autoStop != false {
		t.Errorf("GetVoiceAssistantAutoStop() after save = %v, want false", autoStop)
	}
}

func TestSaveVoiceAssistantQuietHours(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	// Aktivieren
	err := service.SaveVoiceAssistantQuietHoursEnabled(true)
	if err != nil {
		t.Fatalf("SaveVoiceAssistantQuietHoursEnabled(true) error: %v", err)
	}

	enabled := service.GetVoiceAssistantQuietHoursEnabled()
	if enabled != true {
		t.Errorf("GetVoiceAssistantQuietHoursEnabled() = %v, want true", enabled)
	}

	// Zeiten setzen
	err = service.SaveVoiceAssistantQuietHoursStart("20:00")
	if err != nil {
		t.Fatalf("SaveVoiceAssistantQuietHoursStart error: %v", err)
	}

	err = service.SaveVoiceAssistantQuietHoursEnd("06:30")
	if err != nil {
		t.Fatalf("SaveVoiceAssistantQuietHoursEnd error: %v", err)
	}

	start := service.GetVoiceAssistantQuietHoursStart()
	if start != "20:00" {
		t.Errorf("GetVoiceAssistantQuietHoursStart() = %q, want %q", start, "20:00")
	}

	end := service.GetVoiceAssistantQuietHoursEnd()
	if end != "06:30" {
		t.Errorf("GetVoiceAssistantQuietHoursEnd() = %q, want %q", end, "06:30")
	}
}

func TestSaveVoiceAssistantTTSEnabled(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	// Deaktivieren (Default ist true)
	err := service.SaveVoiceAssistantTTSEnabled(false)
	if err != nil {
		t.Fatalf("SaveVoiceAssistantTTSEnabled(false) error: %v", err)
	}

	ttsEnabled := service.GetVoiceAssistantTTSEnabled()
	if ttsEnabled != false {
		t.Errorf("GetVoiceAssistantTTSEnabled() = %v, want false", ttsEnabled)
	}
}

// =============================================================================
// TESTS: Complete Settings Struct
// =============================================================================

func TestGetVoiceAssistantSettings_DefaultValues(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	settings := service.GetVoiceAssistantSettings()

	// Alle Defaults prüfen
	if settings.Enabled != false {
		t.Errorf("settings.Enabled = %v, want false", settings.Enabled)
	}
	if settings.WakeWord != WakeWordHeyEwa {
		t.Errorf("settings.WakeWord = %q, want %q", settings.WakeWord, WakeWordHeyEwa)
	}
	if settings.CustomWakeWord != "" {
		t.Errorf("settings.CustomWakeWord = %q, want empty", settings.CustomWakeWord)
	}
	if settings.AutoStop != true {
		t.Errorf("settings.AutoStop = %v, want true", settings.AutoStop)
	}
	if settings.QuietHoursEnabled != false {
		t.Errorf("settings.QuietHoursEnabled = %v, want false", settings.QuietHoursEnabled)
	}
	if settings.QuietHoursStart != "22:00" {
		t.Errorf("settings.QuietHoursStart = %q, want %q", settings.QuietHoursStart, "22:00")
	}
	if settings.QuietHoursEnd != "07:00" {
		t.Errorf("settings.QuietHoursEnd = %q, want %q", settings.QuietHoursEnd, "07:00")
	}
	if settings.TTSEnabled != true {
		t.Errorf("settings.TTSEnabled = %v, want true", settings.TTSEnabled)
	}
}

func TestSaveVoiceAssistantSettings_AllFields(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	// Alle Felder setzen
	input := VoiceAssistantSettings{
		Enabled:           true,
		WakeWord:          WakeWordCustom,
		CustomWakeWord:    "Hallo Fleet",
		AutoStop:          false,
		QuietHoursEnabled: true,
		QuietHoursStart:   "21:00",
		QuietHoursEnd:     "08:00",
		TTSEnabled:        false,
	}

	err := service.SaveVoiceAssistantSettings(input)
	if err != nil {
		t.Fatalf("SaveVoiceAssistantSettings() error: %v", err)
	}

	// Alle Felder prüfen
	settings := service.GetVoiceAssistantSettings()

	if settings.Enabled != true {
		t.Errorf("settings.Enabled = %v, want true", settings.Enabled)
	}
	if settings.WakeWord != WakeWordCustom {
		t.Errorf("settings.WakeWord = %q, want %q", settings.WakeWord, WakeWordCustom)
	}
	if settings.CustomWakeWord != "Hallo Fleet" {
		t.Errorf("settings.CustomWakeWord = %q, want %q", settings.CustomWakeWord, "Hallo Fleet")
	}
	if settings.AutoStop != false {
		t.Errorf("settings.AutoStop = %v, want false", settings.AutoStop)
	}
	if settings.QuietHoursEnabled != true {
		t.Errorf("settings.QuietHoursEnabled = %v, want true", settings.QuietHoursEnabled)
	}
	if settings.QuietHoursStart != "21:00" {
		t.Errorf("settings.QuietHoursStart = %q, want %q", settings.QuietHoursStart, "21:00")
	}
	if settings.QuietHoursEnd != "08:00" {
		t.Errorf("settings.QuietHoursEnd = %q, want %q", settings.QuietHoursEnd, "08:00")
	}
	if settings.TTSEnabled != false {
		t.Errorf("settings.TTSEnabled = %v, want false", settings.TTSEnabled)
	}
}

func TestSaveVoiceAssistantSettings_PartialUpdate(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	// Erst alles setzen
	initial := VoiceAssistantSettings{
		Enabled:           true,
		WakeWord:          WakeWordEwa,
		CustomWakeWord:    "Test",
		AutoStop:          true,
		QuietHoursEnabled: true,
		QuietHoursStart:   "23:00",
		QuietHoursEnd:     "05:00",
		TTSEnabled:        true,
	}

	err := service.SaveVoiceAssistantSettings(initial)
	if err != nil {
		t.Fatalf("SaveVoiceAssistantSettings() initial error: %v", err)
	}

	// Dann nur einige Felder ändern
	updated := VoiceAssistantSettings{
		Enabled:           false, // geändert
		WakeWord:          WakeWordEwa,
		CustomWakeWord:    "Neuer Test", // geändert
		AutoStop:          true,
		QuietHoursEnabled: false, // geändert
		QuietHoursStart:   "23:00",
		QuietHoursEnd:     "05:00",
		TTSEnabled:        true,
	}

	err = service.SaveVoiceAssistantSettings(updated)
	if err != nil {
		t.Fatalf("SaveVoiceAssistantSettings() update error: %v", err)
	}

	settings := service.GetVoiceAssistantSettings()

	if settings.Enabled != false {
		t.Errorf("settings.Enabled = %v, want false", settings.Enabled)
	}
	if settings.CustomWakeWord != "Neuer Test" {
		t.Errorf("settings.CustomWakeWord = %q, want %q", settings.CustomWakeWord, "Neuer Test")
	}
	if settings.QuietHoursEnabled != false {
		t.Errorf("settings.QuietHoursEnabled = %v, want false", settings.QuietHoursEnabled)
	}
}

// =============================================================================
// TESTS: Wake Word Constants
// =============================================================================

func TestWakeWordConstants(t *testing.T) {
	if WakeWordHeyEwa != "hey_ewa" {
		t.Errorf("WakeWordHeyEwa = %q, want %q", WakeWordHeyEwa, "hey_ewa")
	}
	if WakeWordEwa != "ewa" {
		t.Errorf("WakeWordEwa = %q, want %q", WakeWordEwa, "ewa")
	}
	if WakeWordCustom != "custom" {
		t.Errorf("WakeWordCustom = %q, want %q", WakeWordCustom, "custom")
	}
}

// =============================================================================
// TESTS: Database Persistence
// =============================================================================

func TestVoiceAssistantSettings_Persistence(t *testing.T) {
	// Temporäres Verzeichnis erstellen
	tempDir, err := os.MkdirTemp("", "voice_assistant_persist_test")
	if err != nil {
		t.Fatalf("Fehler beim Erstellen des Temp-Verzeichnisses: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "settings.db")

	// Erste Session: Daten schreiben
	{
		repo, err := NewRepository(tempDir)
		if err != nil {
			t.Fatalf("Fehler beim Erstellen des Repositories: %v", err)
		}

		service := NewService(repo)

		settings := VoiceAssistantSettings{
			Enabled:           true,
			WakeWord:          WakeWordCustom,
			CustomWakeWord:    "Persistenz Test",
			AutoStop:          false,
			QuietHoursEnabled: true,
			QuietHoursStart:   "19:00",
			QuietHoursEnd:     "09:00",
			TTSEnabled:        false,
		}

		err = service.SaveVoiceAssistantSettings(settings)
		if err != nil {
			t.Fatalf("SaveVoiceAssistantSettings() error: %v", err)
		}

		repo.Close()
	}

	// Prüfen dass DB-Datei existiert
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("settings.db wurde nicht erstellt")
	}

	// Zweite Session: Daten lesen
	{
		repo, err := NewRepository(tempDir)
		if err != nil {
			t.Fatalf("Fehler beim erneuten Öffnen des Repositories: %v", err)
		}
		defer repo.Close()

		service := NewService(repo)

		settings := service.GetVoiceAssistantSettings()

		if settings.Enabled != true {
			t.Errorf("Nach Neustart: settings.Enabled = %v, want true", settings.Enabled)
		}
		if settings.WakeWord != WakeWordCustom {
			t.Errorf("Nach Neustart: settings.WakeWord = %q, want %q", settings.WakeWord, WakeWordCustom)
		}
		if settings.CustomWakeWord != "Persistenz Test" {
			t.Errorf("Nach Neustart: settings.CustomWakeWord = %q, want %q", settings.CustomWakeWord, "Persistenz Test")
		}
		if settings.AutoStop != false {
			t.Errorf("Nach Neustart: settings.AutoStop = %v, want false", settings.AutoStop)
		}
		if settings.QuietHoursEnabled != true {
			t.Errorf("Nach Neustart: settings.QuietHoursEnabled = %v, want true", settings.QuietHoursEnabled)
		}
		if settings.QuietHoursStart != "19:00" {
			t.Errorf("Nach Neustart: settings.QuietHoursStart = %q, want %q", settings.QuietHoursStart, "19:00")
		}
		if settings.QuietHoursEnd != "09:00" {
			t.Errorf("Nach Neustart: settings.QuietHoursEnd = %q, want %q", settings.QuietHoursEnd, "09:00")
		}
		if settings.TTSEnabled != false {
			t.Errorf("Nach Neustart: settings.TTSEnabled = %v, want false", settings.TTSEnabled)
		}
	}
}
