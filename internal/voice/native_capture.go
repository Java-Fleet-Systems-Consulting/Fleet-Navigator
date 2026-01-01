package voice

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

// NativeAudioCapture nutzt OS-Tools für Mikrofon-Zugriff
// Linux: arecord/parecord, macOS/Windows: sox
type NativeAudioCapture struct {
	mu sync.RWMutex

	cmd        *exec.Cmd
	stdout     io.ReadCloser
	running    bool
	stopChan   chan struct{}
	sampleRate int
	channels   int

	// RingBuffer für Audio-Daten
	ringBuffer *RingBuffer

	// Callbacks
	OnAudioData func(data []byte)
	OnError     func(err error)
}

// NativeAudioCaptureConfig enthält die Konfiguration
type NativeAudioCaptureConfig struct {
	SampleRate int // 16000 für Whisper
	Channels   int // 1 = Mono
	BufferSecs int // Sekunden im RingBuffer
	Device     string // Audio-Gerät (optional)
}

// DefaultNativeAudioCaptureConfig gibt Standard-Konfiguration zurück
func DefaultNativeAudioCaptureConfig() NativeAudioCaptureConfig {
	return NativeAudioCaptureConfig{
		SampleRate: 16000,
		Channels:   1,
		BufferSecs: 30,
		Device:     "", // Default device
	}
}

// NewNativeAudioCapture erstellt eine neue NativeAudioCapture Instanz
func NewNativeAudioCapture(config NativeAudioCaptureConfig) *NativeAudioCapture {
	bufferSize := config.SampleRate * config.Channels * 2 * config.BufferSecs // 16-bit = 2 bytes

	return &NativeAudioCapture{
		sampleRate: config.SampleRate,
		channels:   config.Channels,
		ringBuffer: NewRingBuffer(bufferSize),
		stopChan:   make(chan struct{}),
	}
}

// GetCaptureCommand gibt das plattformspezifische Capture-Kommando zurück
func GetCaptureCommand(sampleRate, channels int, device string) (string, []string, error) {
	switch runtime.GOOS {
	case "linux":
		// Versuche zuerst parecord (PulseAudio), dann arecord (ALSA)
		if _, err := exec.LookPath("parecord"); err == nil {
			args := []string{
				"--rate", fmt.Sprintf("%d", sampleRate),
				"--channels", fmt.Sprintf("%d", channels),
				"--format", "s16le",
				"--raw",
			}
			if device != "" {
				args = append(args, "--device", device)
			}
			return "parecord", args, nil
		}
		if _, err := exec.LookPath("arecord"); err == nil {
			args := []string{
				"-f", "S16_LE",
				"-r", fmt.Sprintf("%d", sampleRate),
				"-c", fmt.Sprintf("%d", channels),
				"-t", "raw",
				"-q", // Quiet mode
			}
			if device != "" {
				args = append(args, "-D", device)
			}
			return "arecord", args, nil
		}
		return "", nil, fmt.Errorf("weder parecord noch arecord gefunden - installiere: sudo apt install pulseaudio-utils oder alsa-utils")

	case "darwin":
		// macOS: sox mit coreaudio
		if _, err := exec.LookPath("sox"); err == nil {
			args := []string{
				"-d", // Default input device
				"-t", "raw",
				"-r", fmt.Sprintf("%d", sampleRate),
				"-c", fmt.Sprintf("%d", channels),
				"-b", "16",
				"-e", "signed-integer",
				"-", // Output to stdout
			}
			return "sox", args, nil
		}
		return "", nil, fmt.Errorf("sox nicht gefunden - installiere: brew install sox")

	case "windows":
		// Windows: sox mit DirectSound
		if _, err := exec.LookPath("sox"); err == nil {
			args := []string{
				"-t", "waveaudio", "default",
				"-t", "raw",
				"-r", fmt.Sprintf("%d", sampleRate),
				"-c", fmt.Sprintf("%d", channels),
				"-b", "16",
				"-e", "signed-integer",
				"-",
			}
			return "sox", args, nil
		}
		return "", nil, fmt.Errorf("sox nicht gefunden - installiere SoX von https://sox.sourceforge.net/")

	default:
		return "", nil, fmt.Errorf("OS %s nicht unterstützt", runtime.GOOS)
	}
}

// IsAvailable prüft ob Audio-Capture möglich ist
func (nac *NativeAudioCapture) IsAvailable() (bool, string) {
	cmd, _, err := GetCaptureCommand(nac.sampleRate, nac.channels, "")
	if err != nil {
		return false, err.Error()
	}
	return true, cmd
}

// Start startet die Audio-Aufnahme
func (nac *NativeAudioCapture) Start(device string) error {
	nac.mu.Lock()
	defer nac.mu.Unlock()

	if nac.running {
		return nil // Bereits gestartet
	}

	cmdName, args, err := GetCaptureCommand(nac.sampleRate, nac.channels, device)
	if err != nil {
		return err
	}

	log.Printf("🎤 Starte Audio-Capture: %s %v", cmdName, args)

	nac.cmd = exec.Command(cmdName, args...)

	// Stdout für Audio-Daten
	stdout, err := nac.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	nac.stdout = stdout

	// Starte den Prozess
	if err := nac.cmd.Start(); err != nil {
		return fmt.Errorf("prozess starten: %w", err)
	}

	nac.running = true
	nac.stopChan = make(chan struct{})

	// Audio-Reader Goroutine
	go nac.readAudioLoop()

	log.Printf("🎤 Audio-Capture gestartet mit %s (PID: %d)", cmdName, nac.cmd.Process.Pid)
	return nil
}

// readAudioLoop liest kontinuierlich Audio-Daten
func (nac *NativeAudioCapture) readAudioLoop() {
	reader := bufio.NewReader(nac.stdout)
	buf := make([]byte, 4096) // ~128ms bei 16kHz mono

	for {
		select {
		case <-nac.stopChan:
			return
		default:
			n, err := reader.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("🎤 Audio-Lese-Fehler: %v", err)
					if nac.OnError != nil {
						nac.OnError(err)
					}
				}
				return
			}

			if n > 0 {
				// In RingBuffer schreiben
				nac.ringBuffer.Write(buf[:n])

				// Callback aufrufen
				if nac.OnAudioData != nil {
					nac.OnAudioData(buf[:n])
				}
			}
		}
	}
}

// Stop stoppt die Audio-Aufnahme
func (nac *NativeAudioCapture) Stop() error {
	nac.mu.Lock()
	defer nac.mu.Unlock()

	if !nac.running {
		return nil
	}

	log.Printf("🎤 Stoppe Audio-Capture...")

	// Signal zum Stoppen
	close(nac.stopChan)

	// Prozess beenden
	if nac.cmd != nil && nac.cmd.Process != nil {
		nac.cmd.Process.Kill()
		nac.cmd.Wait()
	}

	nac.running = false
	log.Printf("🎤 Audio-Capture gestoppt")

	return nil
}

// IsRunning gibt zurück ob Capture aktiv ist
func (nac *NativeAudioCapture) IsRunning() bool {
	nac.mu.RLock()
	defer nac.mu.RUnlock()
	return nac.running
}

// GetLastNSeconds gibt die letzten N Sekunden Audio zurück
func (nac *NativeAudioCapture) GetLastNSeconds(seconds int) []byte {
	bytesNeeded := nac.sampleRate * nac.channels * 2 * seconds
	return nac.ringBuffer.ReadLast(bytesNeeded)
}

// GetLastNSecondsAsWAV gibt die letzten N Sekunden als WAV zurück
func (nac *NativeAudioCapture) GetLastNSecondsAsWAV(seconds int) ([]byte, error) {
	pcmData := nac.GetLastNSeconds(seconds)
	return pcmToWAV(pcmData, uint32(nac.sampleRate), uint32(nac.channels))
}

// ClearBuffer leert den Audio-Buffer
func (nac *NativeAudioCapture) ClearBuffer() {
	nac.ringBuffer.Clear()
}

// GetSampleRate gibt die Sample-Rate zurück
func (nac *NativeAudioCapture) GetSampleRate() int {
	return nac.sampleRate
}

// GetBufferDurationMs gibt die Buffer-Länge in Millisekunden zurück
func (nac *NativeAudioCapture) GetBufferDurationMs() int {
	bytesInBuffer := nac.ringBuffer.size
	samplesInBuffer := bytesInBuffer / 2 / nac.channels
	durationMs := samplesInBuffer * 1000 / nac.sampleRate
	return durationMs
}

// ListDevices listet verfügbare Audio-Eingabegeräte
func ListAudioDevices() ([]AudioDevice, error) {
	switch runtime.GOOS {
	case "linux":
		return listLinuxDevices()
	case "darwin":
		return listMacDevices()
	case "windows":
		return listWindowsDevices()
	default:
		return []AudioDevice{{ID: "default", Name: "Standard-Mikrofon", IsDefault: true}}, nil
	}
}

func listLinuxDevices() ([]AudioDevice, error) {
	devices := []AudioDevice{{ID: "default", Name: "Standard-Mikrofon", IsDefault: true}}

	// PulseAudio Geräte
	if cmd, err := exec.Command("pactl", "list", "sources", "short").Output(); err == nil {
		for _, line := range splitLines(string(cmd)) {
			if line != "" {
				parts := splitFields(line)
				if len(parts) >= 2 {
					devices = append(devices, AudioDevice{
						ID:   parts[1],
						Name: parts[1],
					})
				}
			}
		}
	}

	// ALSA Geräte (Fallback)
	if cmd, err := exec.Command("arecord", "-l").Output(); err == nil {
		// Parse arecord output
		_ = cmd // TODO: Parse ALSA devices
	}

	return devices, nil
}

func listMacDevices() ([]AudioDevice, error) {
	return []AudioDevice{
		{ID: "default", Name: "Standard-Mikrofon", IsDefault: true},
	}, nil
}

func listWindowsDevices() ([]AudioDevice, error) {
	return []AudioDevice{
		{ID: "default", Name: "Standard-Mikrofon", IsDefault: true},
	}, nil
}

// Helper functions
func splitLines(s string) []string {
	var lines []string
	for _, line := range []byte(s) {
		if line == '\n' {
			lines = append(lines, "")
		}
	}
	return lines
}

func splitFields(s string) []string {
	var fields []string
	field := ""
	for _, c := range s {
		if c == ' ' || c == '\t' {
			if field != "" {
				fields = append(fields, field)
				field = ""
			}
		} else {
			field += string(c)
		}
	}
	if field != "" {
		fields = append(fields, field)
	}
	return fields
}

// WakeWordListener kombiniert NativeAudioCapture mit WakeWordDetector
type WakeWordListener struct {
	capture  *NativeAudioCapture
	detector *WakeWordDetector
	whisper  *WhisperSTT
	running  bool
	stopChan chan struct{}
	mu       sync.RWMutex

	// Callbacks
	OnWakeWord func(word string, confidence float64)
	OnError    func(err error)

	// Konfiguration
	checkIntervalMs int
	audioLengthSecs int
}

// WakeWordListenerConfig enthält die Konfiguration
type WakeWordListenerConfig struct {
	SampleRate      int
	CheckIntervalMs int // Wie oft auf Wake Word prüfen (default: 500)
	AudioLengthSecs int // Wie viele Sekunden Audio analysieren (default: 2)
	Device          string
}

// DefaultWakeWordListenerConfig gibt Standard-Konfiguration zurück
func DefaultWakeWordListenerConfig() WakeWordListenerConfig {
	return WakeWordListenerConfig{
		SampleRate:      16000,
		CheckIntervalMs: 500,
		AudioLengthSecs: 2,
		Device:          "",
	}
}

// NewWakeWordListener erstellt einen neuen WakeWordListener
func NewWakeWordListener(whisper *WhisperSTT, wakeWords []string, config WakeWordListenerConfig) *WakeWordListener {
	captureConfig := NativeAudioCaptureConfig{
		SampleRate: config.SampleRate,
		Channels:   1,
		BufferSecs: 30,
		Device:     config.Device,
	}

	capture := NewNativeAudioCapture(captureConfig)

	wakeConfig := WakeWordConfig{
		WakeWords:  wakeWords,
		Threshold:  0.5,
		CooldownMs: 2000,
	}
	detector := NewWakeWordDetector(whisper, wakeConfig)

	return &WakeWordListener{
		capture:         capture,
		detector:        detector,
		whisper:         whisper,
		checkIntervalMs: config.CheckIntervalMs,
		audioLengthSecs: config.AudioLengthSecs,
	}
}

// Start startet den WakeWordListener
func (wwl *WakeWordListener) Start() error {
	wwl.mu.Lock()
	defer wwl.mu.Unlock()

	if wwl.running {
		return nil
	}

	// Prüfe ob Audio-Capture verfügbar ist
	available, tool := wwl.capture.IsAvailable()
	if !available {
		return fmt.Errorf("Audio-Capture nicht verfügbar: %s", tool)
	}

	// Prüfe ob Whisper verfügbar ist
	status := wwl.whisper.GetStatus()
	if !status.Available {
		return fmt.Errorf("Whisper nicht verfügbar - Binary: %v, Modell: %v", status.BinaryFound, status.ModelFound)
	}

	// Starte Audio-Capture
	if err := wwl.capture.Start(""); err != nil {
		return fmt.Errorf("Audio-Capture starten: %w", err)
	}

	wwl.running = true
	wwl.stopChan = make(chan struct{})

	// Wake Word Detection Loop
	go wwl.detectionLoop()

	log.Printf("👂 Wake Word Listener gestartet (prüft alle %dms)", wwl.checkIntervalMs)
	return nil
}

// detectionLoop prüft periodisch auf Wake Words
func (wwl *WakeWordListener) detectionLoop() {
	ticker := time.NewTicker(time.Duration(wwl.checkIntervalMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-wwl.stopChan:
			return
		case <-ticker.C:
			wwl.checkForWakeWord()
		}
	}
}

// checkForWakeWord prüft die letzten N Sekunden auf Wake Word
func (wwl *WakeWordListener) checkForWakeWord() {
	// Audio aus Buffer holen
	wavData, err := wwl.capture.GetLastNSecondsAsWAV(wwl.audioLengthSecs)
	if err != nil {
		log.Printf("👂 Audio abrufen: %v", err)
		return
	}

	// Prüfe ob genug Audio da ist (mindestens 50% der erwarteten Länge)
	expectedBytes := wwl.capture.sampleRate * wwl.capture.channels * 2 * wwl.audioLengthSecs
	if len(wavData) < expectedBytes/2+44 { // +44 für WAV Header
		return // Noch nicht genug Audio
	}

	// Wake Word Detection
	result, err := wwl.detector.Detect(wavData)
	if err != nil {
		// Nicht loggen wenn einfach keine Sprache erkannt wurde
		if err.Error() != "Whisper nicht initialisiert" {
			log.Printf("👂 Wake Word Detection: %v", err)
		}
		return
	}

	if result.Detected {
		log.Printf("👂 Wake Word erkannt: '%s' (Konfidenz: %.2f)", result.Word, result.Confidence)

		// Buffer leeren um doppelte Erkennung zu vermeiden
		wwl.capture.ClearBuffer()

		// Callback
		if wwl.OnWakeWord != nil {
			wwl.OnWakeWord(result.Word, result.Confidence)
		}
	}
}

// Stop stoppt den WakeWordListener
func (wwl *WakeWordListener) Stop() error {
	wwl.mu.Lock()
	defer wwl.mu.Unlock()

	if !wwl.running {
		return nil
	}

	log.Printf("👂 Stoppe Wake Word Listener...")

	close(wwl.stopChan)
	wwl.capture.Stop()
	wwl.running = false

	log.Printf("👂 Wake Word Listener gestoppt")
	return nil
}

// IsRunning gibt zurück ob der Listener aktiv ist
func (wwl *WakeWordListener) IsRunning() bool {
	wwl.mu.RLock()
	defer wwl.mu.RUnlock()
	return wwl.running
}

// SetWakeWords ändert die Wake Words
func (wwl *WakeWordListener) SetWakeWords(words []string) {
	wwl.detector.SetWakeWords(words)
}

// SetEnabled aktiviert/deaktiviert die Erkennung (ohne Capture zu stoppen)
func (wwl *WakeWordListener) SetEnabled(enabled bool) {
	wwl.detector.SetEnabled(enabled)
}
