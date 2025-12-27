package llamaserver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// createFakeGGUF erstellt eine Fake-GGUF-Datei mit der GGUF Magic Number
func createFakeGGUF(path string, size int64) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// GGUF Magic Number: "GGUF" (0x46554747)
	magic := []byte{0x47, 0x47, 0x55, 0x46}
	if _, err := f.Write(magic); err != nil {
		return err
	}

	// Rest mit Nullen auffüllen
	if size > 4 {
		if err := f.Truncate(size); err != nil {
			return err
		}
	}
	return nil
}

// TestCleanupIncompleteDownloads_EmptyDir testet Cleanup mit leerem Verzeichnis
func TestCleanupIncompleteDownloads_EmptyDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "llama-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	srv := NewServer(Config{ModelsDir: tmpDir})
	cleaned := srv.CleanupIncompleteDownloads()

	if cleaned != 0 {
		t.Errorf("Erwartet 0 gelöschte Dateien, bekam %d", cleaned)
	}
}

// TestCleanupIncompleteDownloads_NoModelsDir testet Cleanup ohne ModelsDir
func TestCleanupIncompleteDownloads_NoModelsDir(t *testing.T) {
	srv := NewServer(Config{ModelsDir: ""})
	cleaned := srv.CleanupIncompleteDownloads()

	if cleaned != 0 {
		t.Errorf("Erwartet 0 bei leerem ModelsDir, bekam %d", cleaned)
	}
}

// TestCleanupIncompleteDownloads_IncompleteDownload testet Cleanup von unvollständigen Downloads
func TestCleanupIncompleteDownloads_IncompleteDownload(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "llama-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Erstelle unvollständige Download-Simulation:
	// - .gguf Datei mit 50MB (unvollständig)
	// - .meta Datei mit erwarteter Größe 100MB
	ggufPath := filepath.Join(tmpDir, "test-model.gguf")
	metaPath := ggufPath + ".meta"

	// 50MB GGUF (unvollständig)
	if err := createFakeGGUF(ggufPath, 50*1024*1024); err != nil {
		t.Fatal(err)
	}

	// Meta-Datei mit erwarteter Größe 100MB
	if err := os.WriteFile(metaPath, []byte("104857600"), 0644); err != nil {
		t.Fatal(err)
	}

	srv := NewServer(Config{ModelsDir: tmpDir})
	cleaned := srv.CleanupIncompleteDownloads()

	if cleaned != 1 {
		t.Errorf("Erwartet 1 gelöschte Datei, bekam %d", cleaned)
	}

	// GGUF sollte gelöscht sein
	if _, err := os.Stat(ggufPath); !os.IsNotExist(err) {
		t.Error("Unvollständige GGUF-Datei sollte gelöscht sein")
	}

	// Meta-Datei sollte auch gelöscht sein
	if _, err := os.Stat(metaPath); !os.IsNotExist(err) {
		t.Error("Meta-Datei sollte gelöscht sein")
	}
}

// TestCleanupIncompleteDownloads_CompleteDownload testet dass vollständige Downloads nicht gelöscht werden
func TestCleanupIncompleteDownloads_CompleteDownload(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "llama-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Erstelle vollständigen Download:
	// - .gguf Datei mit 100MB
	// - .meta Datei mit erwarteter Größe 100MB
	ggufPath := filepath.Join(tmpDir, "complete-model.gguf")
	metaPath := ggufPath + ".meta"

	// 100MB GGUF (vollständig)
	if err := createFakeGGUF(ggufPath, 100*1024*1024); err != nil {
		t.Fatal(err)
	}

	// Meta-Datei mit erwarteter Größe 100MB
	if err := os.WriteFile(metaPath, []byte("104857600"), 0644); err != nil {
		t.Fatal(err)
	}

	srv := NewServer(Config{ModelsDir: tmpDir})
	cleaned := srv.CleanupIncompleteDownloads()

	if cleaned != 0 {
		t.Errorf("Vollständige Downloads sollten nicht gelöscht werden, gelöscht: %d", cleaned)
	}

	// GGUF sollte noch existieren
	if _, err := os.Stat(ggufPath); os.IsNotExist(err) {
		t.Error("Vollständige GGUF-Datei sollte noch existieren")
	}

	// Meta-Datei sollte gelöscht sein (Cleanup entfernt sie nach Prüfung)
	if _, err := os.Stat(metaPath); !os.IsNotExist(err) {
		t.Error("Meta-Datei sollte nach erfolgreicher Prüfung gelöscht sein")
	}
}

// TestCleanupIncompleteDownloads_PartFiles testet Cleanup von .part Dateien
func TestCleanupIncompleteDownloads_PartFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "llama-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Erstelle .part Dateien (von Multi-Connection Downloads)
	partFiles := []string{
		filepath.Join(tmpDir, "model.gguf.part0"),
		filepath.Join(tmpDir, "model.gguf.part1"),
		filepath.Join(tmpDir, "model.gguf.part2"),
	}

	for _, pf := range partFiles {
		if err := os.WriteFile(pf, []byte("partial data"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	srv := NewServer(Config{ModelsDir: tmpDir})
	cleaned := srv.CleanupIncompleteDownloads()

	if cleaned != 3 {
		t.Errorf("Erwartet 3 gelöschte .part Dateien, bekam %d", cleaned)
	}

	// Alle .part Dateien sollten gelöscht sein
	for _, pf := range partFiles {
		if _, err := os.Stat(pf); !os.IsNotExist(err) {
			t.Errorf(".part Datei sollte gelöscht sein: %s", pf)
		}
	}
}

// TestGetLargestModel_EmptyDir testet GetLargestModel mit leerem Verzeichnis
func TestGetLargestModel_EmptyDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "llama-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	srv := NewServer(Config{ModelsDir: tmpDir})
	_, _, err = srv.GetLargestModel()

	if err == nil {
		t.Error("Erwartet Fehler bei leerem Verzeichnis")
	}
}

// TestGetLargestModel_SingleModel testet GetLargestModel mit einem Modell
func TestGetLargestModel_SingleModel(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "llama-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Erstelle ein Modell
	modelPath := filepath.Join(tmpDir, "test-model.gguf")
	if err := createFakeGGUF(modelPath, 1024*1024); err != nil { // 1MB
		t.Fatal(err)
	}

	srv := NewServer(Config{ModelsDir: tmpDir})
	path, name, err := srv.GetLargestModel()

	if err != nil {
		t.Errorf("Unerwarteter Fehler: %v", err)
	}
	if path != modelPath {
		t.Errorf("Erwartet Pfad %s, bekam %s", modelPath, path)
	}
	if name != "test-model.gguf" {
		t.Errorf("Erwartet Name 'test-model.gguf', bekam '%s'", name)
	}
}

// TestGetLargestModel_MultipleModels testet GetLargestModel mit mehreren Modellen
func TestGetLargestModel_MultipleModels(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "llama-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Erstelle mehrere Modelle unterschiedlicher Größe
	models := []struct {
		name string
		size int64
	}{
		{"small-model.gguf", 1 * 1024 * 1024},    // 1MB
		{"medium-model.gguf", 5 * 1024 * 1024},   // 5MB
		{"large-model.gguf", 10 * 1024 * 1024},   // 10MB (größtes)
		{"another-model.gguf", 3 * 1024 * 1024},  // 3MB
	}

	for _, m := range models {
		path := filepath.Join(tmpDir, m.name)
		if err := createFakeGGUF(path, m.size); err != nil {
			t.Fatal(err)
		}
	}

	srv := NewServer(Config{ModelsDir: tmpDir})
	_, name, err := srv.GetLargestModel()

	if err != nil {
		t.Errorf("Unerwarteter Fehler: %v", err)
	}
	if name != "large-model.gguf" {
		t.Errorf("Erwartet größtes Modell 'large-model.gguf', bekam '%s'", name)
	}
}

// TestGetLargestModel_SubdirectoryModels testet ob Modelle in Unterverzeichnissen gefunden werden
func TestGetLargestModel_SubdirectoryModels(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "llama-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Erstelle Unterverzeichnis (wie library/)
	libDir := filepath.Join(tmpDir, "library")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Kleines Modell im Root
	smallPath := filepath.Join(tmpDir, "small.gguf")
	if err := createFakeGGUF(smallPath, 1*1024*1024); err != nil {
		t.Fatal(err)
	}

	// Großes Modell im Unterverzeichnis
	largePath := filepath.Join(libDir, "large.gguf")
	if err := createFakeGGUF(largePath, 10*1024*1024); err != nil {
		t.Fatal(err)
	}

	srv := NewServer(Config{ModelsDir: tmpDir})
	path, name, err := srv.GetLargestModel()

	if err != nil {
		t.Errorf("Unerwarteter Fehler: %v", err)
	}
	if name != "large.gguf" {
		t.Errorf("Erwartet größtes Modell 'large.gguf' aus Unterverzeichnis, bekam '%s'", name)
	}
	if path != largePath {
		t.Errorf("Erwartet Pfad %s, bekam %s", largePath, path)
	}
}

// TestFindModelByName_ExactMatch testet exakten Modellnamen-Match
func TestFindModelByName_ExactMatch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "llama-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	modelPath := filepath.Join(tmpDir, "Meta-Llama-3.1-8B-Instruct-Q4_K_M.gguf")
	if err := createFakeGGUF(modelPath, 1024*1024); err != nil {
		t.Fatal(err)
	}

	srv := NewServer(Config{ModelsDir: tmpDir})
	path, err := srv.FindModelByName("Meta-Llama-3.1-8B-Instruct-Q4_K_M.gguf")

	if err != nil {
		t.Errorf("Unerwarteter Fehler: %v", err)
	}
	if path != modelPath {
		t.Errorf("Erwartet Pfad %s, bekam %s", modelPath, path)
	}
}

// TestFindModelByName_CaseInsensitive testet case-insensitiven Match
func TestFindModelByName_CaseInsensitive(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "llama-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	modelPath := filepath.Join(tmpDir, "Meta-Llama-3.1-8B-Instruct-Q4_K_M.gguf")
	if err := createFakeGGUF(modelPath, 1024*1024); err != nil {
		t.Fatal(err)
	}

	srv := NewServer(Config{ModelsDir: tmpDir})

	// Teste verschiedene Schreibweisen
	testCases := []string{
		"meta-llama-3.1-8b-instruct-q4_k_m.gguf", // lowercase
		"META-LLAMA-3.1-8B-INSTRUCT-Q4_K_M.GGUF", // uppercase
		"Meta-Llama-3.1-8B-Instruct-Q4_K_M.gguf", // original
	}

	for _, tc := range testCases {
		path, err := srv.FindModelByName(tc)
		if err != nil {
			t.Errorf("Fehler bei '%s': %v", tc, err)
		}
		if path != modelPath {
			t.Errorf("Erwartet Pfad %s für '%s', bekam %s", modelPath, tc, path)
		}
	}
}

// TestFindModelByName_PartialMatch testet partiellen Match
func TestFindModelByName_PartialMatch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "llama-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	modelPath := filepath.Join(tmpDir, "Meta-Llama-3.1-8B-Instruct-Q4_K_M.gguf")
	if err := createFakeGGUF(modelPath, 1024*1024); err != nil {
		t.Fatal(err)
	}

	srv := NewServer(Config{ModelsDir: tmpDir})

	// Partielle Suche
	testCases := []string{
		"llama-3.1",
		"Meta-Llama",
		"8B-Instruct",
	}

	for _, tc := range testCases {
		path, err := srv.FindModelByName(tc)
		if err != nil {
			t.Errorf("Fehler bei partiellem Match '%s': %v", tc, err)
		}
		if path != modelPath {
			t.Errorf("Erwartet Pfad %s für '%s', bekam %s", modelPath, tc, path)
		}
	}
}

// TestFindModelByName_FuzzyMatch testet Fuzzy-Matching mit Keywords
func TestFindModelByName_FuzzyMatch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "llama-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	modelPath := filepath.Join(tmpDir, "Meta-Llama-3.1-8B-Instruct-Q4_K_M.gguf")
	if err := createFakeGGUF(modelPath, 1024*1024); err != nil {
		t.Fatal(err)
	}

	srv := NewServer(Config{ModelsDir: tmpDir})

	// Fuzzy-Suche (wie Benutzer eingeben würde)
	testCases := []string{
		"meta llama 3.1 8b",     // Mit Leerzeichen
		"llama 3.1 8b instruct", // Andere Reihenfolge
	}

	for _, tc := range testCases {
		path, err := srv.FindModelByName(tc)
		if err != nil {
			t.Errorf("Fehler bei Fuzzy-Match '%s': %v", tc, err)
		}
		if path != modelPath {
			t.Errorf("Erwartet Pfad %s für Fuzzy '%s', bekam %s", modelPath, tc, path)
		}
	}
}

// TestFindModelByName_NotFound testet Fehlschlag bei nicht existierendem Modell
func TestFindModelByName_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "llama-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Erstelle ein anderes Modell
	modelPath := filepath.Join(tmpDir, "gemma-9b.gguf")
	if err := createFakeGGUF(modelPath, 1024*1024); err != nil {
		t.Fatal(err)
	}

	srv := NewServer(Config{ModelsDir: tmpDir})

	// Suche nach nicht existierendem Modell
	_, err = srv.FindModelByName("gpt-4") // Existiert definitiv nicht

	if err == nil {
		t.Error("Erwartet Fehler bei nicht existierendem Modell")
	}
}

// TestFindModelByName_InSubdirectory testet Suche in Unterverzeichnissen
func TestFindModelByName_InSubdirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "llama-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Modell im library/ Unterverzeichnis
	libDir := filepath.Join(tmpDir, "library")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		t.Fatal(err)
	}

	modelPath := filepath.Join(libDir, "Meta-Llama-3.1-8B-Instruct-Q4_K_M.gguf")
	if err := createFakeGGUF(modelPath, 1024*1024); err != nil {
		t.Fatal(err)
	}

	srv := NewServer(Config{ModelsDir: tmpDir})
	path, err := srv.FindModelByName("meta llama 3.1 8b")

	if err != nil {
		t.Errorf("Modell im Unterverzeichnis nicht gefunden: %v", err)
	}
	if path != modelPath {
		t.Errorf("Erwartet Pfad %s, bekam %s", modelPath, path)
	}
}

// TestSwitchToModelWithFallback_EmptyName testet mit leerem Modellnamen
func TestSwitchToModelWithFallback_EmptyName(t *testing.T) {
	srv := NewServer(Config{ModelsDir: ""})
	switched, usedFallback, actualName, err := srv.SwitchToModelWithFallback("")

	if err != nil {
		t.Errorf("Unerwarteter Fehler: %v", err)
	}
	if switched {
		t.Error("Sollte nicht gewechselt haben bei leerem Namen")
	}
	if usedFallback {
		t.Error("Sollte keinen Fallback verwendet haben")
	}
	if actualName != "" {
		t.Errorf("Erwartet leeren Namen, bekam '%s'", actualName)
	}
}

// Benchmark für FindModelByName
func BenchmarkFindModelByName(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "llama-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Erstelle mehrere Modelle
	models := []string{
		"gemma-2-9b-it-IQ4_XS.gguf",
		"Meta-Llama-3.1-8B-Instruct-Q4_K_M.gguf",
		"Qwen2.5-7B-Instruct-Q4_K_M.gguf",
		"mistral-7b-instruct-v0.2.Q4_K_M.gguf",
		"phi-3-mini-4k-instruct.Q4_K_M.gguf",
	}

	for _, m := range models {
		path := filepath.Join(tmpDir, m)
		if err := createFakeGGUF(path, 1024); err != nil {
			b.Fatal(err)
		}
	}

	srv := NewServer(Config{ModelsDir: tmpDir})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		srv.FindModelByName("meta llama 3.1 8b")
	}
}

// TestWaitForHealthy_Timeout testet dass WaitForHealthy nach Timeout Fehler gibt
func TestWaitForHealthy_Timeout(t *testing.T) {
	srv := NewServer(Config{Port: 29999}) // Port ohne Server

	// Sehr kurzer Timeout für schnellen Test
	err := srv.WaitForHealthy(100 * time.Millisecond)

	if err == nil {
		t.Error("WaitForHealthy sollte Fehler geben wenn Server nicht läuft")
	}
	if !strings.Contains(err.Error(), "nicht bereit") {
		t.Errorf("Fehler sollte 'nicht bereit' enthalten, bekam: %v", err)
	}
}

// TestWaitForHealthy_Exists testet dass die Funktion existiert und korrekt signiert ist
func TestWaitForHealthy_Signature(t *testing.T) {
	srv := NewServer(Config{Port: 29998})

	// Typ-Check: WaitForHealthy(timeout time.Duration) error
	var fn func(time.Duration) error = srv.WaitForHealthy
	if fn == nil {
		t.Error("WaitForHealthy sollte existieren")
	}
}
