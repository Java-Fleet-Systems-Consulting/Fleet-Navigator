package voice

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"sync"
	"time"
)

// AudioCapture ermöglicht Mikrofon-Zugriff und Audio-Aufnahme
// Aktuell: Plattformunabhängige Implementierung ohne CGO
// Für echtes Mikrofon: Frontend WebAudio API oder native Extensions
type AudioCapture struct {
	mu sync.RWMutex

	// Audio-Konfiguration
	sampleRate  uint32
	channels    uint32
	bufferSize  uint32
	initialized bool

	// Aufnahme-Status
	recording bool
	stopChan  chan struct{}

	// Audio-Buffer für kontinuierliches Listening
	ringBuffer *RingBuffer

	// Callbacks
	OnAudioData func(data []byte)
	OnError     func(err error)
}

// RingBuffer speichert Audio-Daten in einem zirkulären Buffer
type RingBuffer struct {
	mu       sync.RWMutex
	data     []byte
	size     int
	writePos int
}

// NewRingBuffer erstellt einen neuen Ring-Buffer
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		data: make([]byte, size),
		size: size,
	}
}

// Write schreibt Daten in den Buffer
func (rb *RingBuffer) Write(p []byte) (n int, err error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	for _, b := range p {
		rb.data[rb.writePos] = b
		rb.writePos = (rb.writePos + 1) % rb.size
	}
	return len(p), nil
}

// ReadLast liest die letzten n Bytes aus dem Buffer
func (rb *RingBuffer) ReadLast(n int) []byte {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if n > rb.size {
		n = rb.size
	}

	result := make([]byte, n)
	startPos := (rb.writePos - n + rb.size) % rb.size

	for i := 0; i < n; i++ {
		result[i] = rb.data[(startPos+i)%rb.size]
	}

	return result
}

// Clear leert den Buffer
func (rb *RingBuffer) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.writePos = 0
	for i := range rb.data {
		rb.data[i] = 0
	}
}

// AudioCaptureConfig enthält die Konfiguration für AudioCapture
type AudioCaptureConfig struct {
	SampleRate uint32 // Standard: 16000 (für Whisper)
	Channels   uint32 // Standard: 1 (Mono)
	BufferSize uint32 // Samples pro Buffer
	BufferSecs int    // Sekunden für Ring-Buffer
}

// DefaultAudioCaptureConfig gibt die Standard-Konfiguration zurück
func DefaultAudioCaptureConfig() AudioCaptureConfig {
	return AudioCaptureConfig{
		SampleRate: 16000, // Whisper erwartet 16kHz
		Channels:   1,     // Mono
		BufferSize: 1024,  // ~64ms bei 16kHz
		BufferSecs: 30,    // 30 Sekunden Ring-Buffer
	}
}

// NewAudioCapture erstellt eine neue AudioCapture-Instanz
func NewAudioCapture(config AudioCaptureConfig) *AudioCapture {
	// Ring-Buffer für N Sekunden Audio (16bit = 2 bytes pro Sample)
	bufferBytes := int(config.SampleRate) * int(config.Channels) * 2 * config.BufferSecs

	return &AudioCapture{
		sampleRate: config.SampleRate,
		channels:   config.Channels,
		bufferSize: config.BufferSize,
		ringBuffer: NewRingBuffer(bufferBytes),
	}
}

// Initialize initialisiert das Audio-Capture-System
func (ac *AudioCapture) Initialize() error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if ac.initialized {
		return nil
	}

	ac.initialized = true
	log.Printf("AudioCapture initialisiert: %d Hz, %d Kanal(e) (Stub-Modus)", ac.sampleRate, ac.channels)
	log.Printf("Hinweis: Für echtes Mikrofon nutze Frontend WebAudio API oder /api/voice/stt Endpoint")

	return nil
}

// ListDevices gibt verfügbare Audio-Eingabegeräte zurück
func (ac *AudioCapture) ListDevices() ([]AudioDevice, error) {
	// Im Stub-Modus: Default-Gerät zurückgeben
	// Das Frontend kann über WebAudio API echte Geräte auflisten
	return []AudioDevice{
		{
			ID:        "default",
			Name:      "Standard-Mikrofon (via Frontend)",
			IsDefault: true,
		},
		{
			ID:        "webaudio",
			Name:      "Browser WebAudio API",
			IsDefault: false,
		},
	}, nil
}

// AudioDevice repräsentiert ein Audio-Eingabegerät
type AudioDevice struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsDefault bool   `json:"isDefault"`
}

// StartCapture startet die Audio-Aufnahme (Stub-Modus)
func (ac *AudioCapture) StartCapture(deviceID string) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if !ac.initialized {
		return fmt.Errorf("AudioCapture nicht initialisiert")
	}

	if ac.recording {
		return nil // Bereits aktiv
	}

	ac.stopChan = make(chan struct{})
	ac.recording = true

	// Im Stub-Modus: Simuliere Audio-Frames
	// In Produktion: Frontend sendet Audio via WebSocket
	go ac.captureLoop()

	log.Printf("Audio-Capture gestartet (Stub-Modus) - Gerät: %s", deviceID)
	log.Printf("Tipp: Sende Audio via POST /api/voice-assistant/audio oder WebSocket")
	return nil
}

// captureLoop simuliert Audio-Aufnahme im Stub-Modus
func (ac *AudioCapture) captureLoop() {
	// Simuliere Audio-Frames alle 64ms (bei 1024 Samples @ 16kHz)
	interval := time.Duration(ac.bufferSize) * time.Second / time.Duration(ac.sampleRate)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Buffer für simulierte Stille
	silenceBuffer := make([]byte, ac.bufferSize*2) // 16-bit = 2 bytes

	for {
		select {
		case <-ac.stopChan:
			return
		case <-ticker.C:
			// Schreibe in Ring-Buffer (Stille im Stub-Modus)
			ac.ringBuffer.Write(silenceBuffer)

			// Callback aufrufen wenn registriert
			if ac.OnAudioData != nil {
				ac.OnAudioData(silenceBuffer)
			}
		}
	}
}

// StopCapture stoppt die Audio-Aufnahme
func (ac *AudioCapture) StopCapture() error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if !ac.recording {
		return nil
	}

	close(ac.stopChan)
	ac.recording = false

	log.Printf("Audio-Aufnahme gestoppt")
	return nil
}

// IsRecording gibt zurück ob gerade aufgenommen wird
func (ac *AudioCapture) IsRecording() bool {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.recording
}

// GetLastNSeconds gibt die letzten N Sekunden Audio zurück
func (ac *AudioCapture) GetLastNSeconds(seconds int) []byte {
	bytesNeeded := int(ac.sampleRate) * int(ac.channels) * 2 * seconds
	return ac.ringBuffer.ReadLast(bytesNeeded)
}

// GetLastNSecondsAsWAV gibt die letzten N Sekunden als WAV-Datei zurück
func (ac *AudioCapture) GetLastNSecondsAsWAV(seconds int) ([]byte, error) {
	pcmData := ac.GetLastNSeconds(seconds)
	return pcmToWAV(pcmData, ac.sampleRate, ac.channels)
}

// InjectAudio fügt externe Audio-Daten in den Buffer ein
// Wird vom Frontend via WebSocket oder REST API aufgerufen
func (ac *AudioCapture) InjectAudio(pcmData []byte) {
	ac.ringBuffer.Write(pcmData)
	if ac.OnAudioData != nil {
		ac.OnAudioData(pcmData)
	}
}

// InjectAudioWAV fügt WAV-Audio-Daten in den Buffer ein
func (ac *AudioCapture) InjectAudioWAV(wavData []byte) error {
	pcmData, _, _, err := ReadFromWAVBytes(wavData)
	if err != nil {
		return err
	}
	ac.InjectAudio(pcmData)
	return nil
}

// pcmToWAV konvertiert PCM-Daten zu WAV-Format
func pcmToWAV(pcmData []byte, sampleRate, channels uint32) ([]byte, error) {
	var buf bytes.Buffer

	// WAV Header
	dataSize := uint32(len(pcmData))
	fileSize := 36 + dataSize

	// RIFF Header
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, fileSize)
	buf.WriteString("WAVE")

	// fmt Chunk
	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16))       // Chunk size
	binary.Write(&buf, binary.LittleEndian, uint16(1))        // Audio format (PCM)
	binary.Write(&buf, binary.LittleEndian, uint16(channels)) // Channels
	binary.Write(&buf, binary.LittleEndian, sampleRate)       // Sample rate
	byteRate := sampleRate * channels * 2
	binary.Write(&buf, binary.LittleEndian, byteRate) // Byte rate
	blockAlign := channels * 2
	binary.Write(&buf, binary.LittleEndian, uint16(blockAlign)) // Block align
	binary.Write(&buf, binary.LittleEndian, uint16(16))         // Bits per sample

	// data Chunk
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, dataSize)
	buf.Write(pcmData)

	return buf.Bytes(), nil
}

// Close schließt das Audio-Capture-System
func (ac *AudioCapture) Close() error {
	ac.StopCapture()

	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.initialized = false

	log.Printf("AudioCapture geschlossen")
	return nil
}

// SimulateAudioInput simuliert Audio-Eingabe für Tests (Alias für InjectAudio)
func (ac *AudioCapture) SimulateAudioInput(audioData []byte) {
	ac.InjectAudio(audioData)
}

// ReadFromFile liest Audio aus einer WAV-Datei
func ReadFromFile(reader io.Reader) ([]byte, uint32, uint32, error) {
	// WAV Header lesen
	header := make([]byte, 44)
	if _, err := io.ReadFull(reader, header); err != nil {
		return nil, 0, 0, fmt.Errorf("WAV-Header lesen: %w", err)
	}

	// Parse Header
	if string(header[0:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return nil, 0, 0, fmt.Errorf("ungültiges WAV-Format")
	}

	channels := binary.LittleEndian.Uint16(header[22:24])
	sampleRate := binary.LittleEndian.Uint32(header[24:28])

	// PCM-Daten lesen
	pcmData, err := io.ReadAll(reader)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("PCM-Daten lesen: %w", err)
	}

	return pcmData, sampleRate, uint32(channels), nil
}

// ReadFromWAVBytes liest Audio aus WAV-Bytes
func ReadFromWAVBytes(wavData []byte) ([]byte, uint32, uint32, error) {
	if len(wavData) < 44 {
		return nil, 0, 0, fmt.Errorf("WAV-Daten zu kurz")
	}

	// Parse Header
	if string(wavData[0:4]) != "RIFF" || string(wavData[8:12]) != "WAVE" {
		return nil, 0, 0, fmt.Errorf("ungültiges WAV-Format")
	}

	channels := binary.LittleEndian.Uint16(wavData[22:24])
	sampleRate := binary.LittleEndian.Uint32(wavData[24:28])

	// PCM-Daten extrahieren (nach Header)
	pcmData := wavData[44:]

	return pcmData, sampleRate, uint32(channels), nil
}

// GetSupportedSampleRates gibt unterstützte Sample-Raten zurück
func GetSupportedSampleRates() []uint32 {
	return []uint32{8000, 16000, 22050, 44100, 48000}
}

// GetAudioLevel berechnet den RMS-Pegel der Audio-Daten (0.0 - 1.0)
func GetAudioLevel(pcmData []byte) float64 {
	if len(pcmData) < 2 {
		return 0
	}

	var sum float64
	samples := len(pcmData) / 2

	for i := 0; i < samples; i++ {
		sample := int16(binary.LittleEndian.Uint16(pcmData[i*2:]))
		normalized := float64(sample) / 32768.0
		sum += normalized * normalized
	}

	rms := sqrt(sum / float64(samples))
	return rms
}

// sqrt berechnet die Quadratwurzel (einfache Implementierung)
func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x / 2
	for i := 0; i < 10; i++ {
		z = z - (z*z-x)/(2*z)
	}
	return z
}

// IsSilence prüft ob die Audio-Daten Stille sind
func IsSilence(pcmData []byte, threshold float64) bool {
	level := GetAudioLevel(pcmData)
	return level < threshold
}

// DetectVoiceActivity erkennt Sprachaktivität (einfache Implementierung)
func DetectVoiceActivity(pcmData []byte) bool {
	// Einfache Schwellwert-basierte Erkennung
	// Typischer Sprach-Pegel liegt bei 0.05 - 0.5
	level := GetAudioLevel(pcmData)
	return level > 0.02 && level < 0.9 // Weder zu leise noch zu laut (Clipping)
}

// GetBufferDurationMs gibt die aktuelle Buffer-Länge in Millisekunden zurück
func (ac *AudioCapture) GetBufferDurationMs() int {
	bytesInBuffer := ac.ringBuffer.size
	samplesInBuffer := bytesInBuffer / 2 / int(ac.channels)
	durationMs := samplesInBuffer * 1000 / int(ac.sampleRate)
	return durationMs
}

// GetSampleRate gibt die Sample-Rate zurück
func (ac *AudioCapture) GetSampleRate() uint32 {
	return ac.sampleRate
}

// GetChannels gibt die Anzahl der Kanäle zurück
func (ac *AudioCapture) GetChannels() uint32 {
	return ac.channels
}
