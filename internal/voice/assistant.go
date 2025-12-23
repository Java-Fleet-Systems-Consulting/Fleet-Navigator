package voice

import (
	"bytes"
	_ "embed"
	"fmt"
	"log"
	"sync"
	"time"
)

// VoiceAssistant ist der Hauptservice für "Hey Ewa" Voice Assistant
// Kombiniert Audio-Capture, Wake Word Detection, STT und TTS
type VoiceAssistant struct {
	mu sync.RWMutex

	// Kern-Komponenten
	voiceService   *Service
	audioCapture   *AudioCapture
	wakeWordDetect *WakeWordDetector

	// Status
	state           AssistantState
	listeningActive bool
	currentDevice   string

	// Konfiguration
	config AssistantConfig

	// Audio Feedback Sounds
	activationSound []byte
	deactivationSound []byte
	errorSound       []byte

	// Channels für Kommunikation
	stopChan       chan struct{}
	wakeResultChan <-chan WakeWordResult
	wakeStopChan   chan<- struct{}

	// Callbacks
	OnStateChange     func(state AssistantState)
	OnWakeWordHeard   func(word string)
	OnTranscription   func(text string)
	OnResponse        func(text string)
	OnError           func(err error)
}

// AssistantState repräsentiert den aktuellen Zustand des Assistenten
type AssistantState string

const (
	StateIdle        AssistantState = "idle"        // Nicht aktiv
	StateListening   AssistantState = "listening"   // Lauscht auf Wake Word
	StateProcessing  AssistantState = "processing"  // Verarbeitet Sprache
	StateSpeaking    AssistantState = "speaking"    // Spricht Antwort
	StateQuietHours  AssistantState = "quiet_hours" // Ruhezeit aktiv
	StateError       AssistantState = "error"       // Fehler aufgetreten
)

// AssistantConfig enthält die Konfiguration für den Voice Assistant
type AssistantConfig struct {
	// Wake Word
	WakeWords     []string `json:"wakeWords"`
	WakeThreshold float64  `json:"wakeThreshold"`

	// Audio
	AudioDevice     string `json:"audioDevice"`
	SampleRate      uint32 `json:"sampleRate"`
	ListenDurationS int    `json:"listenDurationS"` // Wie lange nach Wake Word lauschen

	// Ruhezeiten
	QuietHoursEnabled bool   `json:"quietHoursEnabled"`
	QuietHoursStart   string `json:"quietHoursStart"` // "22:00"
	QuietHoursEnd     string `json:"quietHoursEnd"`   // "07:00"

	// Verhalten
	AutoStop           bool `json:"autoStop"`           // Nach Antwort stoppen
	ContinuousMode     bool `json:"continuousMode"`     // Mehrere Fragen ohne Wake Word
	PlayFeedbackSounds bool `json:"playFeedbackSounds"` // Pling-Sounds
}

// DefaultAssistantConfig gibt die Standard-Konfiguration zurück
func DefaultAssistantConfig() AssistantConfig {
	return AssistantConfig{
		WakeWords:          []string{"hey ewa", "ewa"},
		WakeThreshold:      0.5,
		AudioDevice:        "default",
		SampleRate:         16000,
		ListenDurationS:    10,
		QuietHoursEnabled:  false,
		QuietHoursStart:    "22:00",
		QuietHoursEnd:      "07:00",
		AutoStop:           true,
		ContinuousMode:     false,
		PlayFeedbackSounds: true,
	}
}

// NewVoiceAssistant erstellt einen neuen Voice Assistant
func NewVoiceAssistant(voiceService *Service, config AssistantConfig) *VoiceAssistant {
	// Audio Capture erstellen
	audioConfig := DefaultAudioCaptureConfig()
	audioConfig.SampleRate = config.SampleRate
	audioCapture := NewAudioCapture(audioConfig)

	// Wake Word Detector erstellen
	wakeConfig := DefaultWakeWordConfig()
	wakeConfig.WakeWords = config.WakeWords
	wakeConfig.Threshold = config.WakeThreshold

	var whisper *WhisperSTT
	if voiceService != nil {
		whisper = voiceService.whisper
	}
	wakeWordDetect := NewWakeWordDetector(whisper, wakeConfig)

	return &VoiceAssistant{
		voiceService:   voiceService,
		audioCapture:   audioCapture,
		wakeWordDetect: wakeWordDetect,
		config:         config,
		state:          StateIdle,
		stopChan:       make(chan struct{}),
	}
}

// Initialize initialisiert den Voice Assistant
func (va *VoiceAssistant) Initialize() error {
	va.mu.Lock()
	defer va.mu.Unlock()

	// Audio Capture initialisieren
	if err := va.audioCapture.Initialize(); err != nil {
		return fmt.Errorf("AudioCapture: %w", err)
	}

	// Feedback Sounds laden
	va.loadFeedbackSounds()

	log.Printf("Voice Assistant initialisiert")
	return nil
}

// loadFeedbackSounds lädt die Audio-Feedback Sounds
func (va *VoiceAssistant) loadFeedbackSounds() {
	// TODO: Echte Sound-Dateien laden
	// Für jetzt: Generiere einfache Beep-Sounds
	va.activationSound = generateBeep(800, 200, 16000)   // 800 Hz, 200ms
	va.deactivationSound = generateBeep(400, 200, 16000) // 400 Hz, 200ms
	va.errorSound = generateBeep(200, 500, 16000)        // 200 Hz, 500ms
}

// generateBeep generiert einen einfachen Beep-Ton als PCM-Daten
func generateBeep(frequencyHz, durationMs int, sampleRate uint32) []byte {
	numSamples := int(sampleRate) * durationMs / 1000
	samples := make([]byte, numSamples*2) // 16-bit = 2 bytes

	for i := 0; i < numSamples; i++ {
		// Sinuswelle generieren
		t := float64(i) / float64(sampleRate)
		amplitude := 0.3 // 30% Lautstärke
		value := int16(amplitude * 32767 * sin2pi(float64(frequencyHz)*t))

		// Little-Endian
		samples[i*2] = byte(value)
		samples[i*2+1] = byte(value >> 8)
	}

	return samples
}

// sin2pi berechnet sin(2 * pi * x)
func sin2pi(x float64) float64 {
	// Einfache Sinus-Approximation
	x = x - float64(int(x)) // Modulo 1
	x = x * 2 - 1           // -1 bis 1
	return x * (1.27323954 - 0.405284735*x*x*sign(x))
}

func sign(x float64) float64 {
	if x < 0 {
		return -1
	}
	return 1
}

// Start startet den Voice Assistant (Always-On Modus)
func (va *VoiceAssistant) Start() error {
	va.mu.Lock()
	defer va.mu.Unlock()

	if va.listeningActive {
		return nil // Bereits aktiv
	}

	// Ruhezeiten prüfen
	if va.config.QuietHoursEnabled && va.isQuietHours() {
		va.setState(StateQuietHours)
		return fmt.Errorf("Ruhezeit aktiv - Assistent pausiert")
	}

	// Audio Capture starten
	if err := va.audioCapture.StartCapture(va.config.AudioDevice); err != nil {
		va.setState(StateError)
		return fmt.Errorf("Audio-Aufnahme starten: %w", err)
	}

	// Wake Word Detection starten
	va.wakeWordDetect.SetEnabled(true)
	va.wakeResultChan, va.wakeStopChan = va.wakeWordDetect.DetectFromStream(va.audioCapture, 500)

	// Haupt-Loop starten
	va.stopChan = make(chan struct{})
	go va.mainLoop()

	va.listeningActive = true
	va.setState(StateListening)

	log.Printf("Voice Assistant gestartet - lauscht auf Wake Word")
	return nil
}

// mainLoop ist die Hauptschleife des Voice Assistants
func (va *VoiceAssistant) mainLoop() {
	// Ruhezeiten-Check alle Minute
	quietHoursTicker := time.NewTicker(1 * time.Minute)
	defer quietHoursTicker.Stop()

	for {
		select {
		case <-va.stopChan:
			return

		case result := <-va.wakeResultChan:
			if result.Detected {
				va.handleWakeWord(result)
			}

		case <-quietHoursTicker.C:
			va.checkQuietHours()
		}
	}
}

// handleWakeWord verarbeitet ein erkanntes Wake Word
func (va *VoiceAssistant) handleWakeWord(result WakeWordResult) {
	log.Printf("Wake Word erkannt: %s", result.Word)

	// Callback
	if va.OnWakeWordHeard != nil {
		va.OnWakeWordHeard(result.Word)
	}

	// Aktivierungs-Sound abspielen
	if va.config.PlayFeedbackSounds {
		va.playSound(va.activationSound)
	}

	va.setState(StateProcessing)

	// Auf Spracheingabe warten und verarbeiten
	go va.processVoiceInput()
}

// processVoiceInput verarbeitet die Spracheingabe nach Wake Word
func (va *VoiceAssistant) processVoiceInput() {
	// Warte kurz nach Wake Word
	time.Sleep(500 * time.Millisecond)

	// Audio für N Sekunden aufnehmen
	listenDuration := va.config.ListenDurationS
	log.Printf("Höre zu für %d Sekunden...", listenDuration)

	time.Sleep(time.Duration(listenDuration) * time.Second)

	// Audio abrufen und transkribieren
	audioData, err := va.audioCapture.GetLastNSecondsAsWAV(listenDuration)
	if err != nil {
		va.handleError(fmt.Errorf("Audio abrufen: %w", err))
		return
	}

	result, err := va.voiceService.TranscribeAudio(audioData, "wav")
	if err != nil {
		va.handleError(fmt.Errorf("Transkription: %w", err))
		return
	}

	text := result.Text
	log.Printf("Transkribiert: %s", text)

	// Callback
	if va.OnTranscription != nil {
		va.OnTranscription(text)
	}

	// TODO: An Chat-Engine senden und Antwort bekommen
	// response := chatEngine.Chat(text)
	response := fmt.Sprintf("Ich habe verstanden: %s", text)

	// Antwort vorlesen
	va.speakResponse(response)
}

// speakResponse spricht die Antwort vor
func (va *VoiceAssistant) speakResponse(text string) {
	va.setState(StateSpeaking)

	if va.OnResponse != nil {
		va.OnResponse(text)
	}

	// TTS
	if va.voiceService != nil && va.voiceService.piper != nil {
		speechResult, err := va.voiceService.SynthesizeSpeech(text, "")
		if err != nil {
			log.Printf("TTS-Fehler: %v", err)
		} else {
			va.playSound(speechResult.AudioData)
		}
	}

	// Deaktivierungs-Sound
	if va.config.PlayFeedbackSounds {
		va.playSound(va.deactivationSound)
	}

	// Zurück zum Lauschen oder Stoppen
	if va.config.AutoStop && !va.config.ContinuousMode {
		va.Stop()
	} else {
		va.setState(StateListening)
	}
}

// playSound spielt einen Sound ab
func (va *VoiceAssistant) playSound(pcmData []byte) {
	// TODO: Echte Audio-Ausgabe implementieren
	// Für jetzt: Nur loggen
	if len(pcmData) > 0 {
		log.Printf("Sound abspielen (%d bytes)", len(pcmData))
	}
}

// handleError behandelt Fehler
func (va *VoiceAssistant) handleError(err error) {
	log.Printf("Voice Assistant Fehler: %v", err)

	if va.config.PlayFeedbackSounds {
		va.playSound(va.errorSound)
	}

	if va.OnError != nil {
		va.OnError(err)
	}

	va.setState(StateListening) // Zurück zum Lauschen
}

// checkQuietHours prüft ob Ruhezeit aktiv ist
func (va *VoiceAssistant) checkQuietHours() {
	if !va.config.QuietHoursEnabled {
		return
	}

	if va.isQuietHours() {
		if va.state != StateQuietHours {
			log.Printf("Ruhezeit begonnen - Assistent pausiert")
			va.wakeWordDetect.SetEnabled(false)
			va.setState(StateQuietHours)
		}
	} else {
		if va.state == StateQuietHours {
			log.Printf("Ruhezeit beendet - Assistent wieder aktiv")
			va.wakeWordDetect.SetEnabled(true)
			va.setState(StateListening)
		}
	}
}

// isQuietHours prüft ob gerade Ruhezeit ist
func (va *VoiceAssistant) isQuietHours() bool {
	now := time.Now()
	currentTime := now.Hour()*60 + now.Minute()

	start := parseTimeToMinutes(va.config.QuietHoursStart)
	end := parseTimeToMinutes(va.config.QuietHoursEnd)

	// Über Mitternacht (z.B. 22:00 - 07:00)
	if start > end {
		return currentTime >= start || currentTime < end
	}

	// Innerhalb eines Tages (z.B. 13:00 - 14:00)
	return currentTime >= start && currentTime < end
}

// parseTimeToMinutes parst "HH:MM" zu Minuten seit Mitternacht
func parseTimeToMinutes(timeStr string) int {
	var hour, minute int
	fmt.Sscanf(timeStr, "%d:%d", &hour, &minute)
	return hour*60 + minute
}

// setState setzt den Zustand und ruft Callback auf
func (va *VoiceAssistant) setState(state AssistantState) {
	va.state = state
	if va.OnStateChange != nil {
		va.OnStateChange(state)
	}
}

// Stop stoppt den Voice Assistant
func (va *VoiceAssistant) Stop() error {
	va.mu.Lock()
	defer va.mu.Unlock()

	if !va.listeningActive {
		return nil
	}

	// Wake Word Detection stoppen
	if va.wakeStopChan != nil {
		close(va.wakeStopChan)
	}

	// Haupt-Loop stoppen
	close(va.stopChan)

	// Audio Capture stoppen
	va.audioCapture.StopCapture()

	va.listeningActive = false
	va.setState(StateIdle)

	log.Printf("Voice Assistant gestoppt")
	return nil
}

// GetState gibt den aktuellen Zustand zurück
func (va *VoiceAssistant) GetState() AssistantState {
	va.mu.RLock()
	defer va.mu.RUnlock()
	return va.state
}

// IsActive gibt zurück ob der Assistent aktiv ist
func (va *VoiceAssistant) IsActive() bool {
	va.mu.RLock()
	defer va.mu.RUnlock()
	return va.listeningActive
}

// GetStatus gibt den vollständigen Status zurück
func (va *VoiceAssistant) GetStatus() AssistantStatus {
	va.mu.RLock()
	defer va.mu.RUnlock()

	return AssistantStatus{
		State:           string(va.state),
		Active:          va.listeningActive,
		WakeWordsActive: va.wakeWordDetect.IsEnabled(),
		CurrentDevice:   va.currentDevice,
		QuietHours:      va.config.QuietHoursEnabled && va.isQuietHours(),
		Config:          va.config,
	}
}

// AssistantStatus enthält den vollständigen Status
type AssistantStatus struct {
	State           string          `json:"state"`
	Active          bool            `json:"active"`
	WakeWordsActive bool            `json:"wakeWordsActive"`
	CurrentDevice   string          `json:"currentDevice"`
	QuietHours      bool            `json:"quietHours"`
	Config          AssistantConfig `json:"config"`
}

// UpdateConfig aktualisiert die Konfiguration
func (va *VoiceAssistant) UpdateConfig(config AssistantConfig) {
	va.mu.Lock()
	defer va.mu.Unlock()

	va.config = config
	va.wakeWordDetect.SetWakeWords(config.WakeWords)

	log.Printf("Voice Assistant Konfiguration aktualisiert")
}

// Close schließt den Voice Assistant
func (va *VoiceAssistant) Close() error {
	va.Stop()

	if va.audioCapture != nil {
		va.audioCapture.Close()
	}

	if va.wakeWordDetect != nil {
		va.wakeWordDetect.Close()
	}

	log.Printf("Voice Assistant geschlossen")
	return nil
}

// PlayActivationSound spielt den Aktivierungs-Sound
func (va *VoiceAssistant) PlayActivationSound() {
	va.playSound(va.activationSound)
}

// SetChatCallback setzt den Callback für Chat-Verarbeitung
type ChatCallback func(input string) (response string, err error)

var chatCallback ChatCallback

// SetChatCallback setzt die Chat-Callback-Funktion
func (va *VoiceAssistant) SetChatCallback(callback ChatCallback) {
	chatCallback = callback
}

// GenerateActivationWAV generiert den Aktivierungs-Sound als WAV
func (va *VoiceAssistant) GenerateActivationWAV() ([]byte, error) {
	return pcmToWAV(va.activationSound, 16000, 1)
}

// GenerateDeactivationWAV generiert den Deaktivierungs-Sound als WAV
func (va *VoiceAssistant) GenerateDeactivationWAV() ([]byte, error) {
	return pcmToWAV(va.deactivationSound, 16000, 1)
}

// EmbeddedSounds enthält eingebettete Sound-Dateien
// TODO: Echte Sound-Dateien einbetten
type EmbeddedSounds struct {
	Activation   []byte
	Deactivation []byte
	Error        []byte
	Listening    []byte
}

// LoadEmbeddedSounds lädt eingebettete Sounds
func LoadEmbeddedSounds() *EmbeddedSounds {
	return &EmbeddedSounds{
		Activation:   generateBeep(880, 150, 16000),  // A5
		Deactivation: generateBeep(440, 150, 16000),  // A4
		Error:        generateBeep(220, 300, 16000),  // A3
		Listening:    generateBeep(660, 100, 16000),  // E5
	}
}

// Hilfsfunktion: bytes.Buffer Import vermeiden
var _ = bytes.Buffer{}
