package voice

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// Service verwaltet Spracheingabe (STT) und Sprachausgabe (TTS)
type Service struct {
	dataDir         string
	whisper         *WhisperSTT
	piper           *PiperTTS
	wakeWordListener *WakeWordListener
	mu              sync.RWMutex
	initialized     bool

	// Wake Word Callback - wird von außen gesetzt
	OnWakeWordDetected func(word string, confidence float64)
}

// Config enthält die Konfiguration für den Voice-Service
type Config struct {
	DataDir       string `json:"dataDir"`
	WhisperModel  string `json:"whisperModel"`  // z.B. "base", "small", "medium"
	PiperVoice    string `json:"piperVoice"`    // z.B. "de_DE-thorsten-medium"
	Language      string `json:"language"`      // z.B. "de" oder "auto"
}

// DefaultConfig gibt die Standard-Konfiguration zurück
func DefaultConfig() Config {
	return Config{
		WhisperModel: "base",
		PiperVoice:   "de_DE-eva_k-x_low",
		Language:     "de",
	}
}

// NewService erstellt einen neuen Voice-Service
func NewService(dataDir string) *Service {
	voiceDir := filepath.Join(dataDir, "voice")

	return &Service{
		dataDir: voiceDir,
	}
}

// Initialize initialisiert den Voice-Service
func (s *Service) Initialize(config Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Voice-Verzeichnis erstellen
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return fmt.Errorf("Voice-Verzeichnis erstellen: %w", err)
	}

	// Unterverzeichnisse
	dirs := []string{"whisper", "piper", "temp"}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(s.dataDir, dir), 0755); err != nil {
			return fmt.Errorf("%s-Verzeichnis erstellen: %w", dir, err)
		}
	}

	// Whisper STT initialisieren
	whisperDir := filepath.Join(s.dataDir, "whisper")
	s.whisper = NewWhisperSTT(whisperDir, config.WhisperModel, config.Language)

	// Piper TTS initialisieren
	piperDir := filepath.Join(s.dataDir, "piper")
	s.piper = NewPiperTTS(piperDir, config.PiperVoice)

	s.initialized = true
	log.Printf("Voice-Service initialisiert in %s", s.dataDir)

	return nil
}

// GetStatus gibt den Status des Voice-Services zurück
func (s *Service) GetStatus() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := Status{
		Initialized: s.initialized,
		DataDir:     s.dataDir,
	}

	if s.whisper != nil {
		status.Whisper = s.whisper.GetStatus()
	}

	if s.piper != nil {
		status.Piper = s.piper.GetStatus()
	}

	return status
}

// Status enthält den Status des Voice-Services
type Status struct {
	Initialized bool          `json:"initialized"`
	DataDir     string        `json:"dataDir"`
	Whisper     WhisperStatus `json:"whisper"`
	Piper       PiperStatus   `json:"piper"`
}

// TranscribeAudio transkribiert Audio zu Text (STT)
func (s *Service) TranscribeAudio(audioData []byte, format string) (*TranscriptionResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized || s.whisper == nil {
		return nil, fmt.Errorf("Voice-Service nicht initialisiert")
	}

	return s.whisper.Transcribe(audioData, format)
}

// SynthesizeSpeech erzeugt Sprache aus Text (TTS)
// Optional kann eine spezifische Stimme angegeben werden
func (s *Service) SynthesizeSpeech(text string, voice string) (*SpeechResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized || s.piper == nil {
		return nil, fmt.Errorf("Voice-Service nicht initialisiert")
	}

	return s.piper.SynthesizeWithVoice(text, voice)
}

// EnsureModelsDownloaded stellt sicher, dass alle Modelle heruntergeladen sind
// WICHTIG: Hält den Mutex NICHT während des Downloads um Deadlocks zu vermeiden!
func (s *Service) EnsureModelsDownloaded(progressChan chan<- DownloadProgress) error {
	// Referenzen unter Lock holen
	s.mu.RLock()
	whisper := s.whisper
	piper := s.piper
	s.mu.RUnlock()

	// Downloads OHNE Lock durchführen (vermeidet Deadlock mit GetStatus)
	if whisper != nil {
		if err := whisper.EnsureDownloaded(progressChan); err != nil {
			return fmt.Errorf("Whisper-Download: %w", err)
		}
	}

	if piper != nil {
		if err := piper.EnsureDownloaded(progressChan); err != nil {
			return fmt.Errorf("Piper-Download: %w", err)
		}
	}

	return nil
}

// EnsureWhisperDownloaded stellt sicher, dass nur Whisper heruntergeladen ist
// WICHTIG: Hält den Mutex NICHT während des Downloads um Deadlocks zu vermeiden!
func (s *Service) EnsureWhisperDownloaded(progressChan chan<- DownloadProgress) error {
	// Referenz unter Lock holen
	s.mu.RLock()
	whisper := s.whisper
	s.mu.RUnlock()

	// Download OHNE Lock durchführen
	if whisper != nil {
		if err := whisper.EnsureDownloaded(progressChan); err != nil {
			return fmt.Errorf("Whisper-Download: %w", err)
		}
	}

	return nil
}

// EnsurePiperDownloaded stellt sicher, dass nur Piper heruntergeladen ist
// WICHTIG: Hält den Mutex NICHT während des Downloads um Deadlocks zu vermeiden!
func (s *Service) EnsurePiperDownloaded(progressChan chan<- DownloadProgress) error {
	// Referenz unter Lock holen
	s.mu.RLock()
	piper := s.piper
	s.mu.RUnlock()

	// Download OHNE Lock durchführen
	if piper != nil {
		if err := piper.EnsureDownloaded(progressChan); err != nil {
			return fmt.Errorf("Piper-Download: %w", err)
		}
	}

	return nil
}

// DownloadProgress enthält Fortschrittsinformationen für Downloads
type DownloadProgress struct {
	Component   string  `json:"component"`   // "whisper" oder "piper"
	File        string  `json:"file"`
	TotalBytes  int64   `json:"totalBytes"`
	Downloaded  int64   `json:"downloaded"`
	Percent     float64 `json:"percent"`
	Speed       float64 `json:"speedMBps"`
	Status      string  `json:"status"` // "downloading", "extracting", "done", "error"
	Error       string  `json:"error,omitempty"`
}

// TranscriptionResult enthält das Ergebnis einer Transkription
type TranscriptionResult struct {
	Text       string    `json:"text"`
	Language   string    `json:"language"`
	Confidence float64   `json:"confidence"`
	Duration   float64   `json:"durationSec"`
	Segments   []Segment `json:"segments,omitempty"`
}

// Segment enthält ein Transkriptions-Segment mit Zeitstempel
type Segment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

// SpeechResult enthält das Ergebnis einer TTS-Synthese
type SpeechResult struct {
	AudioData   []byte  `json:"-"`
	Format      string  `json:"format"`      // "wav" oder "mp3"
	SampleRate  int     `json:"sampleRate"`
	DurationSec float64 `json:"durationSec"`
	Voice       string  `json:"voice"`
}

// Close schließt den Voice-Service
func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Temp-Dateien aufräumen
	tempDir := filepath.Join(s.dataDir, "temp")
	os.RemoveAll(tempDir)

	s.initialized = false
	log.Printf("Voice-Service geschlossen")

	return nil
}

// SetWhisperModel wechselt das Whisper-Modell
func (s *Service) SetWhisperModel(modelID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.whisper == nil {
		return fmt.Errorf("Whisper nicht initialisiert")
	}

	s.whisper.SetModel(modelID)
	log.Printf("Whisper-Modell gewechselt auf: %s", modelID)
	return nil
}

// SetPiperVoice wechselt die Piper-Stimme
func (s *Service) SetPiperVoice(voiceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.piper == nil {
		return fmt.Errorf("Piper nicht initialisiert")
	}

	s.piper.SetVoice(voiceID)
	log.Printf("Piper-Stimme gewechselt auf: %s", voiceID)
	return nil
}

// DownloadWhisperModel lädt ein spezifisches Whisper-Modell herunter
// WICHTIG: Hält den Mutex NICHT während des Downloads um Deadlocks zu vermeiden!
func (s *Service) DownloadWhisperModel(modelID string, progressChan chan<- DownloadProgress) error {
	// Referenz holen und Modell setzen unter kurzem Lock
	s.mu.Lock()
	if s.whisper == nil {
		s.mu.Unlock()
		return fmt.Errorf("Whisper nicht initialisiert")
	}
	whisper := s.whisper
	oldModel := whisper.model
	whisper.SetModel(modelID)
	s.mu.Unlock()

	// Download OHNE Lock durchführen
	if err := whisper.downloadModel(progressChan); err != nil {
		// Bei Fehler altes Modell wiederherstellen
		s.mu.Lock()
		whisper.SetModel(oldModel)
		s.mu.Unlock()
		return err
	}

	return nil
}

// DownloadPiperVoice lädt eine spezifische Piper-Stimme herunter
// Unterstützt sowohl vordefinierte als auch beliebige Piper Voice IDs von HuggingFace
// WICHTIG: Hält den Mutex NICHT während des Downloads um Deadlocks zu vermeiden!
func (s *Service) DownloadPiperVoice(voiceID string, progressChan chan<- DownloadProgress) error {
	// Referenz unter kurzem Lock holen
	s.mu.RLock()
	piper := s.piper
	s.mu.RUnlock()

	if piper == nil {
		return fmt.Errorf("Piper nicht initialisiert")
	}

	// Download OHNE Lock durchführen
	return piper.DownloadVoiceByIDWithProgress(voiceID, progressChan)
}

// GetInstalledWhisperModels gibt installierte Whisper-Modelle zurück
func (s *Service) GetInstalledWhisperModels() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.whisper == nil {
		return nil
	}

	whisperDir := filepath.Join(s.dataDir, "whisper", "models")
	installed := make([]string, 0)

	models := []string{"tiny", "base", "small", "medium", "large"}
	for _, m := range models {
		modelFile := filepath.Join(whisperDir, fmt.Sprintf("ggml-%s.bin", m))
		if _, err := os.Stat(modelFile); err == nil {
			installed = append(installed, m)
		}
	}

	return installed
}

// GetInstalledPiperVoices gibt installierte Piper-Stimmen zurück
func (s *Service) GetInstalledPiperVoices() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.piper == nil {
		return nil
	}

	voicesDir := filepath.Join(s.dataDir, "piper", "voices")
	installed := make([]string, 0)

	for voiceID := range piperVoices {
		voiceFile := filepath.Join(voicesDir, voiceID+".onnx")
		if _, err := os.Stat(voiceFile); err == nil {
			installed = append(installed, voiceID)
		}
	}

	return installed
}

// ==================== Wake Word Listener ====================

// WakeWordListenerStatus enthält den Status des Wake Word Listeners
type WakeWordListenerStatus struct {
	Running         bool   `json:"running"`
	Available       bool   `json:"available"`
	AudioCaptureTool string `json:"audioCaptureTool,omitempty"`
	Error           string `json:"error,omitempty"`
	WhisperReady    bool   `json:"whisperReady"`
}

// GetWakeWordStatus gibt den Status des Wake Word Listeners zurück
func (s *Service) GetWakeWordStatus() WakeWordListenerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := WakeWordListenerStatus{}

	// Prüfe ob Audio-Capture verfügbar ist
	capture := NewNativeAudioCapture(DefaultNativeAudioCaptureConfig())
	available, tool := capture.IsAvailable()
	status.Available = available
	status.AudioCaptureTool = tool
	if !available {
		status.Error = tool
	}

	// Prüfe ob Whisper bereit ist
	if s.whisper != nil {
		whisperStatus := s.whisper.GetStatus()
		status.WhisperReady = whisperStatus.Available
	}

	// Prüfe ob Listener läuft
	if s.wakeWordListener != nil {
		status.Running = s.wakeWordListener.IsRunning()
	}

	return status
}

// StartWakeWordListener startet den Wake Word Listener
func (s *Service) StartWakeWordListener(wakeWords []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Bereits laufend?
	if s.wakeWordListener != nil && s.wakeWordListener.IsRunning() {
		return nil
	}

	// Whisper prüfen
	if s.whisper == nil {
		return fmt.Errorf("Whisper nicht initialisiert")
	}

	whisperStatus := s.whisper.GetStatus()
	if !whisperStatus.Available {
		return fmt.Errorf("Whisper nicht verfügbar - Binary: %v, Modell: %v", whisperStatus.BinaryFound, whisperStatus.ModelFound)
	}

	// Default Wake Words
	if len(wakeWords) == 0 {
		wakeWords = []string{"hey ewa", "hallo ewa", "ewa"}
	}

	// Listener erstellen
	config := DefaultWakeWordListenerConfig()
	s.wakeWordListener = NewWakeWordListener(s.whisper, wakeWords, config)

	// Callback setzen
	s.wakeWordListener.OnWakeWord = func(word string, confidence float64) {
		log.Printf("🎤 Wake Word erkannt: '%s' (%.0f%%)", word, confidence*100)
		if s.OnWakeWordDetected != nil {
			s.OnWakeWordDetected(word, confidence)
		}
	}

	// Starten
	if err := s.wakeWordListener.Start(); err != nil {
		s.wakeWordListener = nil
		return fmt.Errorf("Wake Word Listener starten: %w", err)
	}

	log.Printf("🎤 Wake Word Listener gestartet - lauscht auf: %v", wakeWords)
	return nil
}

// StopWakeWordListener stoppt den Wake Word Listener
func (s *Service) StopWakeWordListener() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.wakeWordListener == nil {
		return nil
	}

	if err := s.wakeWordListener.Stop(); err != nil {
		return err
	}

	s.wakeWordListener = nil
	log.Printf("🎤 Wake Word Listener gestoppt")
	return nil
}

// IsWakeWordListenerRunning gibt zurück ob der Listener läuft
func (s *Service) IsWakeWordListenerRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.wakeWordListener == nil {
		return false
	}
	return s.wakeWordListener.IsRunning()
}

// SetWakeWords ändert die Wake Words (zur Laufzeit)
func (s *Service) SetWakeWords(words []string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.wakeWordListener != nil {
		s.wakeWordListener.SetWakeWords(words)
	}
}

// GetAudioDevices gibt verfügbare Audio-Eingabegeräte zurück
func (s *Service) GetAudioDevices() ([]AudioDevice, error) {
	return ListAudioDevices()
}
