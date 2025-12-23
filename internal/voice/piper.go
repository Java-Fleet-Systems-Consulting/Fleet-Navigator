package voice

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// PiperTTS verwaltet die Piper Text-to-Speech Funktionalität
type PiperTTS struct {
	dataDir    string
	voice      string // z.B. "de_DE-thorsten-medium"
	binaryPath string
	voicePath  string // .onnx Datei
	configPath string // .onnx.json Datei
	mu         sync.RWMutex
}

// PiperStatus enthält den Status von Piper
type PiperStatus struct {
	Available   bool   `json:"available"`
	BinaryPath  string `json:"binaryPath"`
	VoicePath   string `json:"voicePath"`
	Voice       string `json:"voice"`
	BinaryFound bool   `json:"binaryFound"`
	VoiceFound  bool   `json:"voiceFound"`
}

// Piper Release URLs
const (
	piperReleaseBase = "https://github.com/rhasspy/piper/releases/download/2023.11.14-2/"
	piperVoicesBase  = "https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/"
)

// Verfügbare Stimmen (Deutsch + Englisch)
var piperVoices = map[string]VoiceInfo{
	// Deutsche Stimmen - Thorsten
	"de_DE-thorsten-medium": {
		ID:          "de_DE-thorsten-medium",
		Name:        "Thorsten",
		Language:    "de_DE",
		Quality:     "medium",
		SizeMB:      63,
		Description: "Männlich, neutral - Standard",
		ModelURL:    piperVoicesBase + "de/de_DE/thorsten/medium/de_DE-thorsten-medium.onnx",
		ConfigURL:   piperVoicesBase + "de/de_DE/thorsten/medium/de_DE-thorsten-medium.onnx.json",
	},
	"de_DE-thorsten-high": {
		ID:          "de_DE-thorsten-high",
		Name:        "Thorsten HD",
		Language:    "de_DE",
		Quality:     "high",
		SizeMB:      90,
		Description: "Männlich, beste Qualität ⭐",
		ModelURL:    piperVoicesBase + "de/de_DE/thorsten/high/de_DE-thorsten-high.onnx",
		ConfigURL:   piperVoicesBase + "de/de_DE/thorsten/high/de_DE-thorsten-high.onnx.json",
	},
	"de_DE-thorsten_emotional-medium": {
		ID:          "de_DE-thorsten_emotional-medium",
		Name:        "Thorsten Emotional",
		Language:    "de_DE",
		Quality:     "medium",
		SizeMB:      63,
		Description: "Männlich, expressiv/emotional",
		ModelURL:    piperVoicesBase + "de/de_DE/thorsten_emotional/medium/de_DE-thorsten_emotional-medium.onnx",
		ConfigURL:   piperVoicesBase + "de/de_DE/thorsten_emotional/medium/de_DE-thorsten_emotional-medium.onnx.json",
	},
	// Deutsche Stimmen - Weiblich
	"de_DE-eva_k-x_low": {
		ID:          "de_DE-eva_k-x_low",
		Name:        "Eva K",
		Language:    "de_DE",
		Quality:     "x_low",
		SizeMB:      18,
		Description: "Weiblich, kompakt",
		ModelURL:    piperVoicesBase + "de/de_DE/eva_k/x_low/de_DE-eva_k-x_low.onnx",
		ConfigURL:   piperVoicesBase + "de/de_DE/eva_k/x_low/de_DE-eva_k-x_low.onnx.json",
	},
	"de_DE-kerstin-low": {
		ID:          "de_DE-kerstin-low",
		Name:        "Kerstin",
		Language:    "de_DE",
		Quality:     "low",
		SizeMB:      30,
		Description: "Weiblich, klar ⭐ Empfohlen",
		ModelURL:    piperVoicesBase + "de/de_DE/kerstin/low/de_DE-kerstin-low.onnx",
		ConfigURL:   piperVoicesBase + "de/de_DE/kerstin/low/de_DE-kerstin-low.onnx.json",
	},
	"de_DE-ramona-low": {
		ID:          "de_DE-ramona-low",
		Name:        "Ramona",
		Language:    "de_DE",
		Quality:     "low",
		SizeMB:      30,
		Description: "Weiblich, warm",
		ModelURL:    piperVoicesBase + "de/de_DE/ramona/low/de_DE-ramona-low.onnx",
		ConfigURL:   piperVoicesBase + "de/de_DE/ramona/low/de_DE-ramona-low.onnx.json",
	},
	// Deutsche Stimmen - Männlich
	"de_DE-karlsson-low": {
		ID:          "de_DE-karlsson-low",
		Name:        "Karlsson",
		Language:    "de_DE",
		Quality:     "low",
		SizeMB:      30,
		Description: "Männlich, tief",
		ModelURL:    piperVoicesBase + "de/de_DE/karlsson/low/de_DE-karlsson-low.onnx",
		ConfigURL:   piperVoicesBase + "de/de_DE/karlsson/low/de_DE-karlsson-low.onnx.json",
	},
	"de_DE-pavoque-low": {
		ID:          "de_DE-pavoque-low",
		Name:        "Pavoque",
		Language:    "de_DE",
		Quality:     "low",
		SizeMB:      30,
		Description: "Männlich, professionell",
		ModelURL:    piperVoicesBase + "de/de_DE/pavoque/low/de_DE-pavoque-low.onnx",
		ConfigURL:   piperVoicesBase + "de/de_DE/pavoque/low/de_DE-pavoque-low.onnx.json",
	},
	// Multi-Speaker
	"de_DE-mls-medium": {
		ID:          "de_DE-mls-medium",
		Name:        "MLS Multi",
		Language:    "de_DE",
		Quality:     "medium",
		SizeMB:      100,
		Description: "Multi-Speaker Dataset",
		ModelURL:    piperVoicesBase + "de/de_DE/mls/medium/de_DE-mls-medium.onnx",
		ConfigURL:   piperVoicesBase + "de/de_DE/mls/medium/de_DE-mls-medium.onnx.json",
	},
	// Englische Stimme
	"en_US-amy-medium": {
		ID:          "en_US-amy-medium",
		Name:        "Amy (English)",
		Language:    "en_US",
		Quality:     "medium",
		SizeMB:      63,
		Description: "English, female",
		ModelURL:    piperVoicesBase + "en/en_US/amy/medium/en_US-amy-medium.onnx",
		ConfigURL:   piperVoicesBase + "en/en_US/amy/medium/en_US-amy-medium.onnx.json",
	},
}

// VoiceInfo enthält Informationen über eine Stimme
type VoiceInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Language    string `json:"language"`
	Quality     string `json:"quality"`
	SizeMB      int64  `json:"sizeMB"`
	Description string `json:"description"`
	ModelURL    string `json:"-"`
	ConfigURL   string `json:"-"`
}

// NewPiperTTS erstellt eine neue Piper-Instanz
func NewPiperTTS(dataDir, voice string) *PiperTTS {
	if voice == "" {
		voice = "de_DE-kerstin-low"
	}

	p := &PiperTTS{
		dataDir: dataDir,
		voice:   voice,
	}

	p.findPaths()

	return p
}

// findPaths sucht nach Binary und Voice-Modell
func (p *PiperTTS) findPaths() {
	// Binary suchen
	binaryName := "piper"
	if runtime.GOOS == "windows" {
		binaryName = "piper.exe"
	}

	// Im dataDir suchen
	localBinary := filepath.Join(p.dataDir, binaryName)
	if _, err := os.Stat(localBinary); err == nil {
		p.binaryPath = localBinary
	}

	// Voice-Modell suchen
	voicesDir := filepath.Join(p.dataDir, "voices")
	voiceFile := p.voice + ".onnx"
	voicePath := filepath.Join(voicesDir, voiceFile)
	configPath := voicePath + ".json"

	if _, err := os.Stat(voicePath); err == nil {
		p.voicePath = voicePath
	}
	if _, err := os.Stat(configPath); err == nil {
		p.configPath = configPath
	}
}

// GetStatus gibt den Status zurück
func (p *PiperTTS) GetStatus() PiperStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return PiperStatus{
		Available:   p.binaryPath != "" && p.voicePath != "",
		BinaryPath:  p.binaryPath,
		VoicePath:   p.voicePath,
		Voice:       p.voice,
		BinaryFound: p.binaryPath != "",
		VoiceFound:  p.voicePath != "",
	}
}

// EnsureDownloaded stellt sicher, dass Binary und Voice vorhanden sind
func (p *PiperTTS) EnsureDownloaded(progressChan chan<- DownloadProgress) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Voices-Verzeichnis erstellen
	voicesDir := filepath.Join(p.dataDir, "voices")
	if err := os.MkdirAll(voicesDir, 0755); err != nil {
		return err
	}

	// Binary herunterladen falls nicht vorhanden
	if p.binaryPath == "" {
		if err := p.downloadBinary(progressChan); err != nil {
			return fmt.Errorf("Binary-Download: %w", err)
		}
	}

	// Voice herunterladen falls nicht vorhanden
	if p.voicePath == "" {
		if err := p.downloadVoice(progressChan); err != nil {
			return fmt.Errorf("Voice-Download: %w", err)
		}
	}

	return nil
}

// downloadBinary lädt das Piper-Binary herunter
func (p *PiperTTS) downloadBinary(progressChan chan<- DownloadProgress) error {
	var downloadURL string
	var filename string

	switch runtime.GOOS {
	case "linux":
		if runtime.GOARCH == "amd64" {
			downloadURL = piperReleaseBase + "piper_linux_x86_64.tar.gz"
			filename = "piper_linux.tar.gz"
		} else if runtime.GOARCH == "arm64" {
			downloadURL = piperReleaseBase + "piper_linux_aarch64.tar.gz"
			filename = "piper_linux.tar.gz"
		} else {
			return fmt.Errorf("Architektur %s nicht unterstützt", runtime.GOARCH)
		}
	case "windows":
		downloadURL = piperReleaseBase + "piper_windows_amd64.zip"
		filename = "piper_windows.zip"
	case "darwin":
		if runtime.GOARCH == "arm64" {
			downloadURL = piperReleaseBase + "piper_macos_aarch64.tar.gz"
		} else {
			downloadURL = piperReleaseBase + "piper_macos_x64.tar.gz"
		}
		filename = "piper_macos.tar.gz"
	default:
		return fmt.Errorf("OS %s nicht unterstützt", runtime.GOOS)
	}

	archivePath := filepath.Join(p.dataDir, filename)

	// Download
	log.Printf("Lade Piper herunter: %s", downloadURL)
	if err := downloadFile(downloadURL, archivePath, "piper", "binary", progressChan); err != nil {
		return err
	}

	// Entpacken
	if progressChan != nil {
		progressChan <- DownloadProgress{
			Component: "piper",
			File:      "binary",
			Status:    "extracting",
		}
	}

	if strings.HasSuffix(filename, ".tar.gz") {
		if err := p.extractTarGz(archivePath); err != nil {
			return err
		}
	} else {
		if err := p.extractZip(archivePath); err != nil {
			return err
		}
	}

	// Archiv löschen
	os.Remove(archivePath)

	// Pfade aktualisieren
	p.findPaths()

	if progressChan != nil {
		progressChan <- DownloadProgress{
			Component: "piper",
			File:      "binary",
			Status:    "done",
		}
	}

	return nil
}

// extractTarGz entpackt ein tar.gz Archiv
func (p *PiperTTS) extractTarGz(archivePath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	// Erste Runde: Dateien extrahieren, Symlinks merken
	var symlinks []struct {
		name   string
		target string
	}

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Nur piper binary und libs extrahieren
		name := filepath.Base(header.Name)
		if name == "piper" || strings.HasSuffix(name, ".so") || strings.Contains(name, ".so.") {
			destPath := filepath.Join(p.dataDir, name)

			switch header.Typeflag {
			case tar.TypeDir:
				continue

			case tar.TypeSymlink:
				// Symlinks später erstellen
				symlinks = append(symlinks, struct {
					name   string
					target string
				}{name: name, target: header.Linkname})
				log.Printf("Piper-Symlink gemerkt: %s -> %s", name, header.Linkname)

			case tar.TypeReg:
				// Reguläre Datei extrahieren
				outFile, err := os.Create(destPath)
				if err != nil {
					return err
				}

				if _, err := io.Copy(outFile, tr); err != nil {
					outFile.Close()
					return err
				}
				outFile.Close()

				// Ausführbar machen
				if name == "piper" {
					os.Chmod(destPath, 0755)
				} else {
					os.Chmod(destPath, 0644)
				}

				log.Printf("Piper-Datei entpackt: %s (%d bytes)", destPath, header.Size)
			}
		}
	}

	// Zweite Runde: Symlinks erstellen
	for _, link := range symlinks {
		destPath := filepath.Join(p.dataDir, link.name)
		targetPath := filepath.Join(p.dataDir, filepath.Base(link.target))

		// Alte Datei/Link löschen falls vorhanden
		os.Remove(destPath)

		// Symlink erstellen
		if err := os.Symlink(filepath.Base(link.target), destPath); err != nil {
			// Falls Symlink fehlschlägt, Datei kopieren
			log.Printf("Symlink fehlgeschlagen, kopiere stattdessen: %s -> %s", link.name, link.target)
			if srcData, err := os.ReadFile(targetPath); err == nil {
				os.WriteFile(destPath, srcData, 0644)
			}
		} else {
			log.Printf("Piper-Symlink erstellt: %s -> %s", link.name, filepath.Base(link.target))
		}
	}

	return nil
}

// extractZip entpackt ein ZIP-Archiv (für Windows)
func (p *PiperTTS) extractZip(archivePath string) error {
	// Hier könnte ZIP-Extraktion implementiert werden
	// Für Linux/Mac verwenden wir tar.gz
	return fmt.Errorf("ZIP-Extraktion nicht implementiert")
}

// downloadVoice lädt das Voice-Modell herunter
func (p *PiperTTS) downloadVoice(progressChan chan<- DownloadProgress) error {
	voiceInfo, ok := piperVoices[p.voice]
	if !ok {
		return fmt.Errorf("Unbekannte Stimme: %s", p.voice)
	}

	voicesDir := filepath.Join(p.dataDir, "voices")

	// Voices-Verzeichnis erstellen falls nicht vorhanden
	if err := os.MkdirAll(voicesDir, 0755); err != nil {
		return fmt.Errorf("Voices-Verzeichnis erstellen: %w", err)
	}

	voiceFile := p.voice + ".onnx"
	configFile := voiceFile + ".json"
	voicePath := filepath.Join(voicesDir, voiceFile)
	configPath := filepath.Join(voicesDir, configFile)

	// Modell herunterladen
	log.Printf("Lade Piper-Stimme '%s' herunter: %s", p.voice, voiceInfo.ModelURL)
	if err := downloadFile(voiceInfo.ModelURL, voicePath, "piper", voiceFile, progressChan); err != nil {
		return err
	}

	// Config herunterladen
	log.Printf("Lade Piper-Config herunter: %s", voiceInfo.ConfigURL)
	if err := downloadFile(voiceInfo.ConfigURL, configPath, "piper", configFile, nil); err != nil {
		return err
	}

	p.voicePath = voicePath
	p.configPath = configPath

	if progressChan != nil {
		progressChan <- DownloadProgress{
			Component: "piper",
			File:      voiceFile,
			Status:    "done",
		}
	}

	return nil
}

// Synthesize erzeugt Sprache aus Text mit der Default-Stimme
func (p *PiperTTS) Synthesize(text string) (*SpeechResult, error) {
	return p.SynthesizeWithVoice(text, "")
}

// SynthesizeWithVoice erzeugt Sprache aus Text mit optionaler spezifischer Stimme
func (p *PiperTTS) SynthesizeWithVoice(text string, voiceID string) (*SpeechResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.binaryPath == "" {
		return nil, fmt.Errorf("Piper Binary nicht gefunden")
	}

	// Stimme bestimmen: angeforderte oder default
	useVoice := p.voice
	useVoicePath := p.voicePath

	if voiceID != "" && voiceID != p.voice {
		// Andere Stimme angefordert - prüfen ob vorhanden
		voicesDir := filepath.Join(p.dataDir, "voices")
		requestedVoicePath := filepath.Join(voicesDir, voiceID+".onnx")
		requestedConfigPath := requestedVoicePath + ".json"

		voiceExists := true
		if _, err := os.Stat(requestedVoicePath); err != nil {
			voiceExists = false
		}
		if _, err := os.Stat(requestedConfigPath); err != nil {
			voiceExists = false
		}

		if voiceExists {
			useVoice = voiceID
			useVoicePath = requestedVoicePath
			log.Printf("TTS: Verwende angeforderte Stimme '%s'", voiceID)
		} else {
			// Versuche die Stimme automatisch herunterzuladen
			log.Printf("TTS: Stimme '%s' nicht gefunden, versuche Download...", voiceID)
			if err := p.downloadVoiceByID(voiceID); err != nil {
				log.Printf("TTS: Download von '%s' fehlgeschlagen: %v, verwende Default '%s'", voiceID, err, p.voice)
			} else {
				// Nach erfolgreichem Download verwenden
				useVoice = voiceID
				useVoicePath = requestedVoicePath
				log.Printf("TTS: Stimme '%s' erfolgreich heruntergeladen und aktiviert", voiceID)
			}
		}
	}

	if useVoicePath == "" {
		return nil, fmt.Errorf("Keine Piper-Stimme verfügbar")
	}

	// Temp-Datei für Output
	tempDir := filepath.Join(p.dataDir, "..", "temp")
	os.MkdirAll(tempDir, 0755)

	outputPath := filepath.Join(tempDir, fmt.Sprintf("speech_%d.wav", time.Now().UnixNano()))
	defer os.Remove(outputPath)

	// Piper ausführen
	start := time.Now()

	// Text über stdin an piper übergeben
	cmd := exec.Command(p.binaryPath,
		"--model", useVoicePath,
		"--output_file", outputPath,
	)

	// LD_LIBRARY_PATH setzen für shared libs
	cmd.Env = append(os.Environ(), fmt.Sprintf("LD_LIBRARY_PATH=%s", p.dataDir))

	// Text über stdin
	cmd.Stdin = strings.NewReader(text)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Piper ausführen: %w - %s", err, string(output))
	}

	duration := time.Since(start).Seconds()

	// Audio-Datei lesen
	audioData, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("Audio lesen: %w", err)
	}

	// Dauer aus WAV-Header extrahieren (vereinfacht)
	audioDuration := float64(len(audioData)-44) / (22050 * 2) // 22050 Hz, 16-bit

	result := &SpeechResult{
		AudioData:   audioData,
		Format:      "wav",
		SampleRate:  22050,
		DurationSec: audioDuration,
		Voice:       useVoice,
	}

	log.Printf("TTS abgeschlossen in %.2fs: %d bytes Audio für '%s' (Stimme: %s)",
		duration, len(audioData), truncateText(text, 30), useVoice)

	return result, nil
}

// SetVoice ändert die Stimme
func (p *PiperTTS) SetVoice(voice string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.voice = voice
	p.findPaths()
}

// GetAvailableVoices gibt alle verfügbaren Stimmen zurück
func GetAvailablePiperVoices() []VoiceInfo {
	voices := make([]VoiceInfo, 0, len(piperVoices))
	for _, v := range piperVoices {
		voices = append(voices, v)
	}
	return voices
}

// GetVoiceInfo gibt Informationen über eine Stimme zurück
func GetVoiceInfo(voiceID string) (*VoiceInfo, bool) {
	info, ok := piperVoices[voiceID]
	return &info, ok
}

// SpeechConfig enthält die TTS-Konfiguration aus der .onnx.json
type SpeechConfig struct {
	Audio struct {
		SampleRate int `json:"sample_rate"`
	} `json:"audio"`
	ESpeakVoice string `json:"espeak_voice"`
	Language    struct {
		Code string `json:"code"`
	} `json:"language"`
}

// LoadSpeechConfig lädt die Sprachkonfiguration
func (p *PiperTTS) LoadSpeechConfig() (*SpeechConfig, error) {
	if p.configPath == "" {
		return nil, fmt.Errorf("Config-Pfad nicht gesetzt")
	}

	data, err := os.ReadFile(p.configPath)
	if err != nil {
		return nil, err
	}

	var config SpeechConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// TestVoice testet die Sprachausgabe mit einem kurzen Text
func (p *PiperTTS) TestVoice() (*SpeechResult, error) {
	testTexts := map[string]string{
		"de_DE": "Hallo! Dies ist ein Test der Sprachausgabe.",
		"en_US": "Hello! This is a speech output test.",
	}

	voiceInfo, ok := piperVoices[p.voice]
	if !ok {
		return nil, fmt.Errorf("Stimme nicht gefunden: %s", p.voice)
	}

	lang := voiceInfo.Language[:5] // "de_DE" oder "en_US"
	text, ok := testTexts[lang]
	if !ok {
		text = testTexts["en_US"]
	}

	return p.Synthesize(text)
}

// parseVoiceID zerlegt eine Voice ID in ihre Bestandteile
// Format: locale-name-quality (z.B. "de_DE-mls-medium" oder "en_US-lessac-high")
// Returns: lang (de), locale (de_DE), name (mls), quality (medium)
func parseVoiceID(voiceID string) (lang, locale, name, quality string, err error) {
	// Voice ID Format: locale-name-quality
	// Beispiele: de_DE-mls-medium, en_US-lessac-high, de_DE-thorsten_emotional-medium
	parts := strings.Split(voiceID, "-")
	if len(parts) < 3 {
		return "", "", "", "", fmt.Errorf("ungültiges Voice ID Format: %s (erwartet: locale-name-quality)", voiceID)
	}

	locale = parts[0]
	quality = parts[len(parts)-1]
	// Name kann Unterstriche enthalten (z.B. thorsten_emotional)
	name = strings.Join(parts[1:len(parts)-1], "-")

	// Locale validieren (z.B. de_DE, en_US)
	if len(locale) != 5 || locale[2] != '_' {
		return "", "", "", "", fmt.Errorf("ungültiges Locale Format: %s (erwartet: xx_XX)", locale)
	}

	lang = locale[:2]

	return lang, locale, name, quality, nil
}

// buildVoiceURLs erstellt die HuggingFace URLs für eine Voice ID
func buildVoiceURLs(voiceID string) (modelURL, configURL string, err error) {
	lang, locale, name, quality, err := parseVoiceID(voiceID)
	if err != nil {
		return "", "", err
	}

	// URL-Pfad: lang/locale/name/quality/voiceID.onnx
	// Beispiel: de/de_DE/mls/medium/de_DE-mls-medium.onnx
	basePath := fmt.Sprintf("%s/%s/%s/%s/%s", lang, locale, name, quality, voiceID)

	modelURL = piperVoicesBase + basePath + ".onnx"
	configURL = piperVoicesBase + basePath + ".onnx.json"

	return modelURL, configURL, nil
}

// downloadVoiceByID lädt eine Stimme anhand ihrer ID herunter (ohne Progress-Channel)
// Unterstützt sowohl vordefinierte Stimmen als auch beliebige Piper Voice IDs von HuggingFace
func (p *PiperTTS) downloadVoiceByID(voiceID string) error {
	var modelURL, configURL string

	// Erst prüfen ob in vordefinierter Map
	if voiceInfo, ok := piperVoices[voiceID]; ok {
		modelURL = voiceInfo.ModelURL
		configURL = voiceInfo.ConfigURL
		log.Printf("Verwende vordefinierte Voice-URLs für '%s'", voiceID)
	} else {
		// URL dynamisch aus Voice ID konstruieren
		var err error
		modelURL, configURL, err = buildVoiceURLs(voiceID)
		if err != nil {
			return fmt.Errorf("Voice ID parsen: %w", err)
		}
		log.Printf("Konstruiere dynamische Voice-URLs für '%s'", voiceID)
	}

	voicesDir := filepath.Join(p.dataDir, "voices")

	// Voices-Verzeichnis erstellen falls nicht vorhanden
	if err := os.MkdirAll(voicesDir, 0755); err != nil {
		return fmt.Errorf("Voices-Verzeichnis erstellen: %w", err)
	}

	voiceFile := voiceID + ".onnx"
	configFile := voiceFile + ".json"
	voicePath := filepath.Join(voicesDir, voiceFile)
	configPath := filepath.Join(voicesDir, configFile)

	// Modell herunterladen
	log.Printf("Lade Piper-Stimme '%s' herunter: %s", voiceID, modelURL)
	if err := downloadFile(modelURL, voicePath, "piper", voiceFile, nil); err != nil {
		return fmt.Errorf("Modell-Download: %w", err)
	}

	// Config herunterladen
	log.Printf("Lade Piper-Config herunter: %s", configURL)
	if err := downloadFile(configURL, configPath, "piper", configFile, nil); err != nil {
		return fmt.Errorf("Config-Download: %w", err)
	}

	log.Printf("Stimme '%s' erfolgreich heruntergeladen", voiceID)
	return nil
}

// DownloadVoiceByIDWithProgress lädt eine Stimme herunter mit Progress-Tracking
// Exportiert für API-Handler
func (p *PiperTTS) DownloadVoiceByIDWithProgress(voiceID string, progressChan chan<- DownloadProgress) error {
	var modelURL, configURL string

	// Erst prüfen ob in vordefinierter Map
	if voiceInfo, ok := piperVoices[voiceID]; ok {
		modelURL = voiceInfo.ModelURL
		configURL = voiceInfo.ConfigURL
	} else {
		// URL dynamisch aus Voice ID konstruieren
		var err error
		modelURL, configURL, err = buildVoiceURLs(voiceID)
		if err != nil {
			return fmt.Errorf("Voice ID parsen: %w", err)
		}
	}

	voicesDir := filepath.Join(p.dataDir, "voices")

	// Voices-Verzeichnis erstellen falls nicht vorhanden
	if err := os.MkdirAll(voicesDir, 0755); err != nil {
		return fmt.Errorf("Voices-Verzeichnis erstellen: %w", err)
	}

	voiceFile := voiceID + ".onnx"
	configFile := voiceFile + ".json"
	voicePath := filepath.Join(voicesDir, voiceFile)
	configPath := filepath.Join(voicesDir, configFile)

	// Progress: Start
	if progressChan != nil {
		progressChan <- DownloadProgress{
			Component: "piper",
			File:      voiceFile,
			Status:    "downloading",
		}
	}

	// Modell herunterladen
	log.Printf("Lade Piper-Stimme '%s' herunter: %s", voiceID, modelURL)
	if err := downloadFile(modelURL, voicePath, "piper", voiceFile, progressChan); err != nil {
		return fmt.Errorf("Modell-Download: %w", err)
	}

	// Config herunterladen
	log.Printf("Lade Piper-Config herunter: %s", configURL)
	if err := downloadFile(configURL, configPath, "piper", configFile, nil); err != nil {
		return fmt.Errorf("Config-Download: %w", err)
	}

	// Progress: Done
	if progressChan != nil {
		progressChan <- DownloadProgress{
			Component: "piper",
			File:      voiceFile,
			Status:    "done",
		}
	}

	log.Printf("Stimme '%s' erfolgreich heruntergeladen", voiceID)
	return nil
}
