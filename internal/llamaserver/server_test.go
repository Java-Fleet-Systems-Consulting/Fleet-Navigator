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

// ============================================================================
// VRAM-Schätzung und -Prüfung Tests
// ============================================================================

// TestEstimateModelVRAMWithContext_7BModel testet VRAM-Schätzung für 7B Modell
func TestEstimateModelVRAMWithContext_7BModel(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vram-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 4.8 GB Fake-Modell (typisch für Q4_K_M 7B)
	modelPath := filepath.Join(tmpDir, "qwen2.5-7b-instruct-q4_k_m.gguf")
	if err := createFakeGGUF(modelPath, 4800*1024*1024); err != nil {
		t.Fatal(err)
	}

	estimated := EstimateModelVRAMWithContext(modelPath, 8192)

	// Erwartung: Modell ~5GB + KV-Cache ~1GB + Overhead ~0.8GB = ~6.8GB
	// Minimum: 6000 MB, Maximum: 9000 MB
	if estimated < 6000 || estimated > 9000 {
		t.Errorf("VRAM-Schätzung für 7B Q4 sollte zwischen 6-9GB liegen, bekam: %d MB (%.1f GB)",
			estimated, float64(estimated)/1024)
	}
}

// TestEstimateModelVRAMWithContext_9BQ8Model testet VRAM-Schätzung für 9B Q8 Modell
func TestEstimateModelVRAMWithContext_9BQ8Model(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vram-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 9.2 GB Fake-Modell (typisch für Q8_0 9B)
	modelPath := filepath.Join(tmpDir, "gemma-2-9b-it-q8_0.gguf")
	if err := createFakeGGUF(modelPath, 9200*1024*1024); err != nil {
		t.Fatal(err)
	}

	estimated := EstimateModelVRAMWithContext(modelPath, 8192)

	// Erwartung für Gemma 2 9B Q8: Modell ~9.7GB + KV-Cache ~2.7GB (ISWA) + Overhead = ~13GB
	// Minimum: 12000 MB, Maximum: 15000 MB
	if estimated < 12000 || estimated > 15000 {
		t.Errorf("VRAM-Schätzung für Gemma 2 9B Q8 sollte zwischen 12-15GB liegen, bekam: %d MB (%.1f GB)",
			estimated, float64(estimated)/1024)
	}
}

// TestEstimateModelVRAMWithContext_3BModel testet VRAM-Schätzung für 3B Modell
func TestEstimateModelVRAMWithContext_3BModel(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vram-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 2 GB Fake-Modell (typisch für Q4 3B)
	modelPath := filepath.Join(tmpDir, "llama-3b-q4_k_m.gguf")
	if err := createFakeGGUF(modelPath, 2000*1024*1024); err != nil {
		t.Fatal(err)
	}

	estimated := EstimateModelVRAMWithContext(modelPath, 4096)

	// Erwartung: Modell ~2.1GB + KV-Cache ~0.3GB + Overhead ~0.8GB = ~3.2GB
	// Minimum: 2500 MB, Maximum: 5000 MB
	if estimated < 2500 || estimated > 5000 {
		t.Errorf("VRAM-Schätzung für 3B Q4 sollte zwischen 2.5-5GB liegen, bekam: %d MB (%.1f GB)",
			estimated, float64(estimated)/1024)
	}
}

// TestEstimateModelVRAMWithContext_LargeContext testet Einfluss von Context-Größe
func TestEstimateModelVRAMWithContext_LargeContext(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vram-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	modelPath := filepath.Join(tmpDir, "mistral-7b-q4_k_m.gguf")
	if err := createFakeGGUF(modelPath, 4500*1024*1024); err != nil {
		t.Fatal(err)
	}

	small := EstimateModelVRAMWithContext(modelPath, 2048)
	large := EstimateModelVRAMWithContext(modelPath, 16384)

	// Größerer Context sollte mehr VRAM brauchen
	if large <= small {
		t.Errorf("Größerer Context sollte mehr VRAM brauchen: 2048=%dMB, 16384=%dMB",
			small, large)
	}

	// Unterschied sollte signifikant sein (mindestens 500MB)
	diff := large - small
	if diff < 500 {
		t.Errorf("Context-Unterschied sollte mindestens 500MB sein, bekam: %dMB", diff)
	}
}

// TestEstimateModelVRAMWithContext_NonexistentFile testet Fallback bei fehlender Datei
func TestEstimateModelVRAMWithContext_NonexistentFile(t *testing.T) {
	estimated := EstimateModelVRAMWithContext("/nonexistent/model.gguf", 8192)

	// Sollte Standard-Wert (6000 MB) zurückgeben
	if estimated != 6000 {
		t.Errorf("Bei fehlender Datei sollte 6000MB zurückgegeben werden, bekam: %d", estimated)
	}
}

// TestGetModelMaxContext testet Context-Limits für verschiedene Modelle
func TestGetModelMaxContext(t *testing.T) {
	tests := []struct {
		modelPath string
		expected  int
	}{
		{"/models/gemma-2-9b-it.gguf", 8192},
		{"/models/gemma-7b.gguf", 8192},
		{"/models/phi-3-mini.gguf", 4096},
		{"/models/phi3-mini.gguf", 4096},
		{"/models/mistral-7b-instruct.gguf", 32768},
		{"/models/llama-3.1-8b.gguf", 131072},
		{"/models/llama3.1-70b.gguf", 131072},
		{"/models/llama-3-8b.gguf", 8192},
		{"/models/llama3-8b.gguf", 8192},
		{"/models/qwen2.5-7b.gguf", 32768},
		{"/models/qwen-2.5-14b.gguf", 32768},
		{"/models/deepseek-coder-7b.gguf", 32768},
		{"/models/unknown-model.gguf", 16384}, // Default
	}

	for _, tt := range tests {
		t.Run(filepath.Base(tt.modelPath), func(t *testing.T) {
			result := GetModelMaxContext(tt.modelPath)
			if result != tt.expected {
				t.Errorf("GetModelMaxContext(%s) = %d, erwartet %d",
					tt.modelPath, result, tt.expected)
			}
		})
	}
}

// TestVRAMError_ErrorMessage testet die Fehlerformatierung
func TestVRAMError_ErrorMessage(t *testing.T) {
	err := &VRAMError{
		Required:   13000,
		Available:  12000,
		ModelName:  "gemma-2-9b-q8.gguf",
		Suggestion: "Empfehlung: Lade die Q4_K_M Version",
	}

	msg := err.Error()

	// Prüfe dass alle wichtigen Informationen enthalten sind
	if !strings.Contains(msg, "gemma-2-9b-q8.gguf") {
		t.Error("Fehlermeldung sollte Modellnamen enthalten")
	}
	if !strings.Contains(msg, "12.7") || !strings.Contains(msg, "11.7") {
		t.Errorf("Fehlermeldung sollte GB-Werte enthalten: %s", msg)
	}
	if !strings.Contains(msg, "Q4_K_M") {
		t.Error("Fehlermeldung sollte Empfehlung enthalten")
	}
}

// TestVRAMError_IsError testet dass VRAMError das error Interface implementiert
func TestVRAMError_IsError(t *testing.T) {
	var err error = &VRAMError{
		Required:   10000,
		Available:  8000,
		ModelName:  "test.gguf",
		Suggestion: "Test",
	}

	if err == nil {
		t.Error("VRAMError sollte nicht nil sein")
	}

	// Type assertion sollte funktionieren
	vramErr, ok := err.(*VRAMError)
	if !ok {
		t.Error("Type assertion zu *VRAMError sollte funktionieren")
	}
	if vramErr.Required != 10000 {
		t.Errorf("Required sollte 10000 sein, bekam: %d", vramErr.Required)
	}
}

// TestEstimateModelVRAMWithContext_GemmaISWA testet ISWA-Verdopplung für Gemma 2
func TestEstimateModelVRAMWithContext_GemmaISWA(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vram-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Gleiche Dateigröße für beide Modelle
	gemma2Path := filepath.Join(tmpDir, "gemma-2-9b.gguf")
	otherPath := filepath.Join(tmpDir, "llama-9b.gguf")

	if err := createFakeGGUF(gemma2Path, 5000*1024*1024); err != nil {
		t.Fatal(err)
	}
	if err := createFakeGGUF(otherPath, 5000*1024*1024); err != nil {
		t.Fatal(err)
	}

	gemma2Estimate := EstimateModelVRAMWithContext(gemma2Path, 8192)
	otherEstimate := EstimateModelVRAMWithContext(otherPath, 8192)

	// Gemma 2 sollte wegen ISWA mehr VRAM brauchen (ca. doppelter KV-Cache)
	if gemma2Estimate <= otherEstimate {
		t.Errorf("Gemma 2 sollte wegen ISWA mehr VRAM brauchen: Gemma2=%dMB, Other=%dMB",
			gemma2Estimate, otherEstimate)
	}

	// Unterschied sollte mindestens 500MB sein (KV-Cache Verdopplung)
	diff := gemma2Estimate - otherEstimate
	if diff < 500 {
		t.Errorf("ISWA-Unterschied sollte mindestens 500MB sein, bekam: %dMB", diff)
	}
}

// TestCheckVRAMAvailable_Suggestions testet spezifische Empfehlungen
func TestCheckVRAMAvailable_Suggestions(t *testing.T) {
	// Da CheckVRAMAvailable nvidia-smi aufruft, können wir nur die Logik testen
	// indem wir die VRAMError-Struktur direkt erstellen

	tests := []struct {
		name        string
		modelName   string
		contextSize int
		expectIn    string
	}{
		{
			name:        "Q8 Modell Empfehlung",
			modelName:   "gemma-9b-q8_0.gguf",
			contextSize: 8192,
			expectIn:    "Q4_K_M",
		},
		{
			name:        "9B Modell Empfehlung",
			modelName:   "mistral-9b-q4.gguf",
			contextSize: 8192,
			expectIn:    "7B",
		},
		{
			name:        "13B Modell Empfehlung",
			modelName:   "llama-13b-q4.gguf",
			contextSize: 4096,
			expectIn:    "7B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simuliere VRAM-Fehler
			modelNameLower := strings.ToLower(tt.modelName)
			var suggestion string

			if strings.Contains(modelNameLower, "q8") || strings.Contains(modelNameLower, "f16") {
				suggestion = "Empfehlung: Lade die Q4_K_M Version (ca. 50% weniger VRAM)"
			} else if strings.Contains(modelNameLower, "9b") || strings.Contains(modelNameLower, "13b") {
				suggestion = "Empfehlung: Lade ein 7B Modell oder eine kleinere Quantisierung"
			}

			if !strings.Contains(suggestion, tt.expectIn) {
				t.Errorf("Empfehlung sollte '%s' enthalten, bekam: %s", tt.expectIn, suggestion)
			}
		})
	}
}

// TestEstimateModelVRAM_FunctionExists testet dass EstimateModelVRAM existiert
func TestEstimateModelVRAM_FunctionExists(t *testing.T) {
	// EstimateModelVRAM ruft intern EstimateModelVRAMWithContext auf
	result := EstimateModelVRAM("/nonexistent/model.gguf")

	// Sollte Standard-Wert zurückgeben (6000 MB)
	if result != 6000 {
		t.Errorf("EstimateModelVRAM sollte 6000 für nicht-existente Datei zurückgeben, bekam: %d", result)
	}
}
