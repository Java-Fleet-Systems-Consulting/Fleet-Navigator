package voice

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// WhisperSTT verwaltet die Whisper Speech-to-Text Funktionalität
type WhisperSTT struct {
	dataDir    string
	model      string // base, small, medium, large
	language   string // de, en, auto
	binaryPath string
	modelPath  string
	mu         sync.RWMutex
}

// WhisperStatus enthält den Status von Whisper
type WhisperStatus struct {
	Available   bool   `json:"available"`
	BinaryPath  string `json:"binaryPath"`
	ModelPath   string `json:"modelPath"`
	Model       string `json:"model"`
	Language    string `json:"language"`
	BinaryFound bool   `json:"binaryFound"`
	ModelFound  bool   `json:"modelFound"`
}

// Whisper-Modell-URLs (ggml format für whisper.cpp)
var whisperModelURLs = map[string]string{
	"tiny":            "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-tiny.bin",
	"base":            "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.bin",
	"small":           "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.bin",
	"medium":          "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-medium.bin",
	"large":           "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3.bin",
	"large-v3-turbo":  "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo.bin",
	"turbo-q5":        "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo-q5_0.bin",
}

// Whisper-Modell-Größen (ungefähr in MB)
var whisperModelSizes = map[string]int64{
	"tiny":           75,
	"base":           148,
	"small":          488,
	"medium":         1533,
	"large":          3095,
	"large-v3-turbo": 1620,
	"turbo-q5":       574,
}

// whisper.cpp Release URL (latest zeigt automatisch auf neueste Version)
const whisperCppReleaseURL = "https://github.com/ggerganov/whisper.cpp/releases/latest/download/"

// NewWhisperSTT erstellt eine neue Whisper-Instanz
func NewWhisperSTT(dataDir, model, language string) *WhisperSTT {
	if model == "" {
		model = "base"
	}
	if language == "" {
		language = "de"
	}

	w := &WhisperSTT{
		dataDir:  dataDir,
		model:    model,
		language: language,
	}

	// Pfade setzen
	w.findPaths()

	return w
}

// findPaths sucht nach Binary und Modell
func (w *WhisperSTT) findPaths() {
	// whisper.cpp Binary suchen
	binaryNames := []string{"whisper-cli", "whisper-cpp", "main"}
	if runtime.GOOS == "windows" {
		binaryNames = []string{"whisper-cli.exe", "whisper-cpp.exe", "main.exe"}
	}

	// Suchpfade für Binary (mehrere Orte)
	var binaryPaths []string
	for _, name := range binaryNames {
		binaryPaths = append(binaryPaths,
			filepath.Join(w.dataDir, name),                          // ~/.fleet-navigator/voice/whisper/whisper-cli
			filepath.Join(w.dataDir, "whisper.cpp", "build", "bin", name), // kompiliert (cmake)
			filepath.Join(w.dataDir, "whisper.cpp", "build", name),  // kompiliert (altes cmake)
			filepath.Join(w.dataDir, "..", name),                    // ~/.fleet-navigator/voice/whisper-cli
		)
	}

	for _, path := range binaryPaths {
		if _, err := os.Stat(path); err == nil {
			w.binaryPath = path
			log.Printf("Whisper-Binary gefunden: %s", path)
			break
		}
	}
	if w.binaryPath == "" {
		log.Printf("Whisper-Binary nicht gefunden in: %v", binaryPaths[:4]) // Nur erste 4 Pfade loggen
	}

	// Modell suchen - mehrere Orte prüfen
	modelFile := fmt.Sprintf("ggml-%s.bin", w.model)
	modelFileAlt := fmt.Sprintf("ggml-whisper-%s.bin", w.model) // Alternative Benennung

	// Suchpfade (Priorität: 1. voice/models, 2. models/library)
	searchPaths := []string{
		filepath.Join(w.dataDir, "models", modelFile),
		filepath.Join(w.dataDir, "models", modelFileAlt),
		filepath.Join(w.dataDir, "..", "models", "library", modelFile),
		filepath.Join(w.dataDir, "..", "models", "library", modelFileAlt),
	}

	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			w.modelPath = path
			log.Printf("Whisper-Modell gefunden: %s", path)
			break
		}
	}
	if w.modelPath == "" {
		log.Printf("Whisper-Modell '%s' nicht gefunden", w.model)
	}
}

// GetStatus gibt den Status zurück
func (w *WhisperSTT) GetStatus() WhisperStatus {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return WhisperStatus{
		Available:   w.binaryPath != "" && w.modelPath != "",
		BinaryPath:  w.binaryPath,
		ModelPath:   w.modelPath,
		Model:       w.model,
		Language:    w.language,
		BinaryFound: w.binaryPath != "",
		ModelFound:  w.modelPath != "",
	}
}

// EnsureDownloaded stellt sicher, dass Binary und Modell vorhanden sind
func (w *WhisperSTT) EnsureDownloaded(progressChan chan<- DownloadProgress) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Models-Verzeichnis erstellen
	modelsDir := filepath.Join(w.dataDir, "models")
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		return err
	}

	// Binary herunterladen/kompilieren falls nicht vorhanden
	if w.binaryPath == "" {
		if err := w.downloadBinary(progressChan); err != nil {
			return fmt.Errorf("Binary-Download: %w", err)
		}
	}

	// Modell herunterladen falls nicht vorhanden
	if w.modelPath == "" {
		if err := w.downloadModel(progressChan); err != nil {
			return fmt.Errorf("Modell-Download: %w", err)
		}
	}

	return nil
}

// downloadBinary lädt das whisper.cpp Binary herunter oder extrahiert das eingebettete
func (w *WhisperSTT) downloadBinary(progressChan chan<- DownloadProgress) error {
	// Auf Linux: eingebettetes Binary extrahieren
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" && HasEmbeddedWhisper() {
		if progressChan != nil {
			progressChan <- DownloadProgress{
				Component: "whisper",
				File:      "binary",
				Status:    "extracting",
				Percent:   50,
			}
		}

		binaryPath, err := ExtractEmbeddedWhisper(w.dataDir)
		if err != nil {
			return fmt.Errorf("eingebettetes Binary extrahieren: %w", err)
		}

		w.binaryPath = binaryPath
		log.Printf("Whisper-Binary extrahiert: %s", binaryPath)

		if progressChan != nil {
			progressChan <- DownloadProgress{
				Component: "whisper",
				File:      "binary",
				Status:    "done",
				Percent:   100,
			}
		}
		return nil
	}

	// Fallback für Linux: kompilieren falls kein eingebettetes Binary
	if runtime.GOOS == "linux" {
		return w.compileWhisperCpp(progressChan)
	}

	// macOS: auch kompilieren (keine vorkompilierten CLI-Binaries verfügbar)
	if runtime.GOOS == "darwin" {
		return w.compileWhisperCpp(progressChan)
	}

	// Windows: vorkompilierte Binaries herunterladen
	var downloadURL string
	var filename string

	switch runtime.GOOS {
	case "windows":
		downloadURL = whisperCppReleaseURL + "whisper-cublas-12.4.0-bin-x64.zip"
		filename = "whisper-cublas.zip"
	default:
		return fmt.Errorf("OS %s nicht unterstützt", runtime.GOOS)
	}

	zipPath := filepath.Join(w.dataDir, filename)

	// Download
	log.Printf("Lade Whisper.cpp herunter: %s", downloadURL)
	if err := downloadFile(downloadURL, zipPath, "whisper", "binary", progressChan); err != nil {
		return err
	}

	// Entpacken
	if progressChan != nil {
		progressChan <- DownloadProgress{
			Component: "whisper",
			File:      "binary",
			Status:    "extracting",
		}
	}

	if err := w.extractBinary(zipPath); err != nil {
		return err
	}

	// ZIP löschen
	os.Remove(zipPath)

	// Pfade aktualisieren
	w.findPaths()

	if progressChan != nil {
		progressChan <- DownloadProgress{
			Component: "whisper",
			File:      "binary",
			Status:    "done",
		}
	}

	return nil
}

// compileWhisperCpp kompiliert whisper.cpp aus den Quellen (für Linux)
func (w *WhisperSTT) compileWhisperCpp(progressChan chan<- DownloadProgress) error {
	whisperDir := filepath.Join(w.dataDir, "whisper.cpp")

	// Prüfen ob cmake und make vorhanden sind
	if _, err := exec.LookPath("cmake"); err != nil {
		return fmt.Errorf("cmake nicht gefunden - bitte installieren: sudo apt install cmake")
	}
	if _, err := exec.LookPath("make"); err != nil {
		return fmt.Errorf("make nicht gefunden - bitte installieren: sudo apt install build-essential")
	}

	// Schritt 1: Repository klonen falls nicht vorhanden
	if _, err := os.Stat(filepath.Join(whisperDir, "CMakeLists.txt")); os.IsNotExist(err) {
		if progressChan != nil {
			progressChan <- DownloadProgress{
				Component: "whisper",
				File:      "whisper.cpp repository",
				Status:    "downloading",
				Percent:   10,
			}
		}
		log.Printf("Klone whisper.cpp Repository...")

		cmd := exec.Command("git", "clone", "--depth", "1", "https://github.com/ggerganov/whisper.cpp.git", whisperDir)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git clone fehlgeschlagen: %w - %s", err, string(output))
		}
	}

	// Schritt 2: Build-Verzeichnis erstellen
	buildDir := filepath.Join(whisperDir, "build")
	os.MkdirAll(buildDir, 0755)

	// Schritt 3: CMake konfigurieren
	if progressChan != nil {
		progressChan <- DownloadProgress{
			Component: "whisper",
			File:      "cmake configure",
			Status:    "downloading",
			Percent:   30,
		}
	}
	log.Printf("Konfiguriere whisper.cpp mit CMake...")

	cmakeCmd := exec.Command("cmake", "..", "-DCMAKE_BUILD_TYPE=Release")
	cmakeCmd.Dir = buildDir
	if output, err := cmakeCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cmake fehlgeschlagen: %w - %s", err, string(output))
	}

	// Schritt 4: Kompilieren
	if progressChan != nil {
		progressChan <- DownloadProgress{
			Component: "whisper",
			File:      "compiling",
			Status:    "downloading",
			Percent:   50,
		}
	}
	log.Printf("Kompiliere whisper.cpp (dies kann einige Minuten dauern)...")

	// Anzahl der CPUs für parallele Kompilierung
	numCPU := runtime.NumCPU()
	makeCmd := exec.Command("make", "-j", fmt.Sprintf("%d", numCPU), "main")
	makeCmd.Dir = buildDir
	if output, err := makeCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("make fehlgeschlagen: %w - %s", err, string(output))
	}

	// Schritt 5: Binary-Pfad aktualisieren
	binaryPath := filepath.Join(buildDir, "bin", "main")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		// Fallback: Binary direkt im build-Ordner
		binaryPath = filepath.Join(buildDir, "main")
	}

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("kompiliertes Binary nicht gefunden in %s", buildDir)
	}

	w.binaryPath = binaryPath
	log.Printf("whisper.cpp erfolgreich kompiliert: %s", binaryPath)

	if progressChan != nil {
		progressChan <- DownloadProgress{
			Component: "whisper",
			File:      "binary",
			Status:    "done",
			Percent:   100,
		}
	}

	return nil
}

// extractBinary entpackt das Binary aus dem ZIP
func (w *WhisperSTT) extractBinary(zipPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// Suche nach main oder whisper binary
		name := filepath.Base(f.Name)
		if name == "main" || name == "main.exe" || name == "whisper" || name == "whisper.exe" {
			destPath := filepath.Join(w.dataDir, name)

			rc, err := f.Open()
			if err != nil {
				return err
			}

			outFile, err := os.Create(destPath)
			if err != nil {
				rc.Close()
				return err
			}

			_, err = io.Copy(outFile, rc)
			rc.Close()
			outFile.Close()

			if err != nil {
				return err
			}

			// Ausführbar machen
			os.Chmod(destPath, 0755)
			log.Printf("Whisper-Binary entpackt: %s", destPath)
		}
	}

	return nil
}

// downloadModel lädt das Whisper-Modell herunter
func (w *WhisperSTT) downloadModel(progressChan chan<- DownloadProgress) error {
	url, ok := whisperModelURLs[w.model]
	if !ok {
		return fmt.Errorf("Unbekanntes Modell: %s", w.model)
	}

	modelsDir := filepath.Join(w.dataDir, "models")
	modelFile := fmt.Sprintf("ggml-%s.bin", w.model)
	modelPath := filepath.Join(modelsDir, modelFile)

	log.Printf("Lade Whisper-Modell '%s' herunter: %s", w.model, url)
	if err := downloadFile(url, modelPath, "whisper", modelFile, progressChan); err != nil {
		return err
	}

	w.modelPath = modelPath

	if progressChan != nil {
		progressChan <- DownloadProgress{
			Component: "whisper",
			File:      modelFile,
			Status:    "done",
		}
	}

	return nil
}

// Transcribe transkribiert Audio zu Text
func (w *WhisperSTT) Transcribe(audioData []byte, format string) (*TranscriptionResult, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.binaryPath == "" || w.modelPath == "" {
		return nil, fmt.Errorf("Whisper nicht vollständig initialisiert")
	}

	// Temp-Datei für Audio erstellen
	tempDir := filepath.Join(w.dataDir, "..", "temp")
	os.MkdirAll(tempDir, 0755)

	// Audio in WAV konvertieren falls nötig
	wavPath := filepath.Join(tempDir, fmt.Sprintf("audio_%d.wav", time.Now().UnixNano()))
	defer os.Remove(wavPath)

	if format == "wav" {
		// Direkt speichern
		if err := os.WriteFile(wavPath, audioData, 0644); err != nil {
			return nil, fmt.Errorf("Audio speichern: %w", err)
		}
	} else {
		// Mit ffmpeg konvertieren
		inputPath := filepath.Join(tempDir, fmt.Sprintf("input_%d.%s", time.Now().UnixNano(), format))
		if err := os.WriteFile(inputPath, audioData, 0644); err != nil {
			return nil, fmt.Errorf("Input speichern: %w", err)
		}
		defer os.Remove(inputPath)

		// ffmpeg Konvertierung: 16kHz, mono, 16-bit PCM
		cmd := exec.Command("ffmpeg", "-y", "-i", inputPath,
			"-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", wavPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("FFmpeg Konvertierung: %w - %s", err, string(output))
		}
	}

	start := time.Now()

	// whisper.cpp ausführen
	args := []string{
		"-m", w.modelPath,
		"-f", wavPath,
		"-l", w.language,
		"--output-txt",
		"--no-timestamps",
	}

	cmd := exec.Command(w.binaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("Whisper ausführen: %w - %s", err, stderr.String())
	}

	duration := time.Since(start).Seconds()

	// Output parsen
	text := strings.TrimSpace(stdout.String())

	// Falls kein Output, versuche .txt Datei zu lesen
	if text == "" {
		txtPath := wavPath + ".txt"
		if data, err := os.ReadFile(txtPath); err == nil {
			text = strings.TrimSpace(string(data))
			os.Remove(txtPath)
		}
	}

	result := &TranscriptionResult{
		Text:       text,
		Language:   w.language,
		Confidence: 0.95, // Whisper gibt keine Confidence zurück
		Duration:   duration,
	}

	log.Printf("Transkription abgeschlossen in %.2fs: %s", duration, truncateText(text, 50))

	return result, nil
}

// SetModel ändert das Whisper-Modell
func (w *WhisperSTT) SetModel(model string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.model = model
	w.findPaths()
}

// SetLanguage ändert die Sprache
func (w *WhisperSTT) SetLanguage(language string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.language = language
}

// downloadFile lädt eine Datei herunter mit Progress-Callback
func downloadFile(url, destPath, component, file string, progressChan chan<- DownloadProgress) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	totalBytes := resp.ContentLength

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	var downloaded int64
	buf := make([]byte, 32*1024)
	startTime := time.Now()
	lastUpdate := startTime

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
			downloaded += int64(n)

			// Progress Update (max alle 100ms)
			if progressChan != nil && time.Since(lastUpdate) > 100*time.Millisecond {
				elapsed := time.Since(startTime).Seconds()
				speed := float64(downloaded) / elapsed / (1024 * 1024) // MB/s

				var percent float64
				if totalBytes > 0 {
					percent = float64(downloaded) / float64(totalBytes) * 100
				}

				progressChan <- DownloadProgress{
					Component:  component,
					File:       file,
					TotalBytes: totalBytes,
					Downloaded: downloaded,
					Percent:    percent,
					Speed:      speed,
					Status:     "downloading",
				}
				lastUpdate = time.Now()
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// truncateText kürzt Text auf maxLen Zeichen
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// GetAvailableModels gibt alle verfügbaren Whisper-Modelle zurück
func GetAvailableWhisperModels() []ModelInfo {
	return []ModelInfo{
		{ID: "tiny", Name: "Tiny", SizeMB: 75, Description: "Schnellstes Modell, geringste Qualität"},
		{ID: "base", Name: "Base", SizeMB: 148, Description: "Gute Balance aus Geschwindigkeit und Qualität"},
		{ID: "small", Name: "Small", SizeMB: 488, Description: "Bessere Qualität, langsamer"},
		{ID: "turbo-q5", Name: "Turbo Q5", SizeMB: 574, Description: "Large-Qualität, 6x schneller, kompakt - EMPFOHLEN"},
		{ID: "medium", Name: "Medium", SizeMB: 1533, Description: "Hohe Qualität, benötigt mehr RAM"},
		{ID: "large-v3-turbo", Name: "Large V3 Turbo", SizeMB: 1620, Description: "Beste Turbo-Qualität, 6x schneller als Large"},
		{ID: "large", Name: "Large V3", SizeMB: 3095, Description: "Maximale Qualität, langsam"},
	}
}

// ModelInfo enthält Informationen über ein Modell
type ModelInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	SizeMB      int64  `json:"sizeMB"`
	Description string `json:"description"`
}

// ExtractDuration extrahiert die Dauer aus einer Audio-Datei (benötigt ffprobe)
func ExtractDuration(audioPath string) (float64, error) {
	cmd := exec.Command("ffprobe", "-v", "quiet", "-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1", audioPath)

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	duration, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, err
	}

	return duration, nil
}

// ValidateAudioFormat prüft ob das Audio-Format unterstützt wird
func ValidateAudioFormat(format string) bool {
	supported := []string{"wav", "mp3", "ogg", "webm", "m4a", "flac"}
	format = strings.ToLower(format)
	for _, f := range supported {
		if f == format {
			return true
		}
	}
	return false
}

// ParseWhisperOutput parst die Whisper-Ausgabe mit Zeitstempeln
func ParseWhisperOutput(output string) []Segment {
	var segments []Segment

	// Regex für Zeitstempel: [00:00:00.000 --> 00:00:02.000] Text
	re := regexp.MustCompile(`\[(\d{2}:\d{2}:\d{2}\.\d{3})\s*-->\s*(\d{2}:\d{2}:\d{2}\.\d{3})\]\s*(.+)`)

	for _, line := range strings.Split(output, "\n") {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 4 {
			start := parseTimestamp(matches[1])
			end := parseTimestamp(matches[2])
			text := strings.TrimSpace(matches[3])

			if text != "" {
				segments = append(segments, Segment{
					Start: start,
					End:   end,
					Text:  text,
				})
			}
		}
	}

	return segments
}

// parseTimestamp parst einen Zeitstempel im Format HH:MM:SS.mmm
func parseTimestamp(ts string) float64 {
	parts := strings.Split(ts, ":")
	if len(parts) != 3 {
		return 0
	}

	hours, _ := strconv.ParseFloat(parts[0], 64)
	minutes, _ := strconv.ParseFloat(parts[1], 64)
	seconds, _ := strconv.ParseFloat(parts[2], 64)

	return hours*3600 + minutes*60 + seconds
}
