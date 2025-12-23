package voice

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// WakeWordDetector erkennt Wake Words in Audio-Streams
// Verwendet Whisper für Transkription und String-Matching
type WakeWordDetector struct {
	mu sync.RWMutex

	whisper    *WhisperSTT
	wakeWords  []string
	threshold  float64
	enabled    bool
	lastDetect time.Time

	// Cooldown nach Erkennung (verhindert Mehrfach-Trigger)
	cooldownMs int

	// Callbacks
	OnWakeWordDetected func(word string, confidence float64)
}

// WakeWordConfig enthält die Konfiguration für Wake Word Detection
type WakeWordConfig struct {
	WakeWords  []string // z.B. ["hey ewa", "ewa", "hallo ewa"]
	Threshold  float64  // Mindest-Konfidenz (0.0 - 1.0)
	CooldownMs int      // Cooldown nach Erkennung in ms
}

// DefaultWakeWordConfig gibt die Standard-Konfiguration zurück
func DefaultWakeWordConfig() WakeWordConfig {
	return WakeWordConfig{
		WakeWords: []string{
			"hey ewa",
			"hallo ewa",
			"ewa",
		},
		Threshold:  0.5,
		CooldownMs: 2000, // 2 Sekunden Cooldown
	}
}

// NewWakeWordDetector erstellt einen neuen Wake Word Detector
func NewWakeWordDetector(whisper *WhisperSTT, config WakeWordConfig) *WakeWordDetector {
	return &WakeWordDetector{
		whisper:    whisper,
		wakeWords:  config.WakeWords,
		threshold:  config.Threshold,
		cooldownMs: config.CooldownMs,
		enabled:    true,
	}
}

// SetWakeWords setzt die zu erkennenden Wake Words
func (wwd *WakeWordDetector) SetWakeWords(words []string) {
	wwd.mu.Lock()
	defer wwd.mu.Unlock()
	wwd.wakeWords = words
	log.Printf("Wake Words gesetzt: %v", words)
}

// SetEnabled aktiviert/deaktiviert die Erkennung
func (wwd *WakeWordDetector) SetEnabled(enabled bool) {
	wwd.mu.Lock()
	defer wwd.mu.Unlock()
	wwd.enabled = enabled
	log.Printf("Wake Word Detection: %v", enabled)
}

// IsEnabled gibt zurück ob die Erkennung aktiv ist
func (wwd *WakeWordDetector) IsEnabled() bool {
	wwd.mu.RLock()
	defer wwd.mu.RUnlock()
	return wwd.enabled
}

// WakeWordResult enthält das Ergebnis einer Wake Word Erkennung
type WakeWordResult struct {
	Detected   bool    `json:"detected"`
	Word       string  `json:"word,omitempty"`
	Confidence float64 `json:"confidence"`
	Text       string  `json:"text,omitempty"` // Vollständige Transkription
}

// Detect prüft ob ein Wake Word in den Audio-Daten enthalten ist
func (wwd *WakeWordDetector) Detect(audioData []byte) (*WakeWordResult, error) {
	wwd.mu.RLock()
	enabled := wwd.enabled
	wakeWords := wwd.wakeWords
	threshold := wwd.threshold
	cooldownMs := wwd.cooldownMs
	lastDetect := wwd.lastDetect
	wwd.mu.RUnlock()

	if !enabled {
		return &WakeWordResult{Detected: false}, nil
	}

	// Cooldown prüfen
	if time.Since(lastDetect) < time.Duration(cooldownMs)*time.Millisecond {
		return &WakeWordResult{Detected: false}, nil
	}

	// Whisper-Check
	if wwd.whisper == nil {
		return nil, fmt.Errorf("Whisper nicht initialisiert")
	}

	// Audio transkribieren
	result, err := wwd.whisper.Transcribe(audioData, "wav")
	if err != nil {
		return nil, fmt.Errorf("Transkription: %w", err)
	}

	// Text normalisieren
	text := strings.ToLower(strings.TrimSpace(result.Text))

	// Wake Words prüfen
	for _, word := range wakeWords {
		normalizedWord := strings.ToLower(strings.TrimSpace(word))

		// Exakte Übereinstimmung oder enthält
		if strings.Contains(text, normalizedWord) {
			// Konfidenz berechnen (basierend auf Whisper-Konfidenz und Match-Qualität)
			confidence := calculateConfidence(text, normalizedWord, result.Confidence)

			if confidence >= threshold {
				wwd.mu.Lock()
				wwd.lastDetect = time.Now()
				wwd.mu.Unlock()

				wakeResult := &WakeWordResult{
					Detected:   true,
					Word:       word,
					Confidence: confidence,
					Text:       result.Text,
				}

				// Callback aufrufen
				if wwd.OnWakeWordDetected != nil {
					wwd.OnWakeWordDetected(word, confidence)
				}

				log.Printf("Wake Word erkannt: '%s' (Konfidenz: %.2f)", word, confidence)
				return wakeResult, nil
			}
		}
	}

	return &WakeWordResult{
		Detected: false,
		Text:     result.Text,
	}, nil
}

// calculateConfidence berechnet die Konfidenz einer Wake Word Erkennung
func calculateConfidence(text, wakeWord string, whisperConfidence float64) float64 {
	// Basis-Konfidenz von Whisper
	confidence := whisperConfidence

	// Bonus für exakten Match am Anfang
	if strings.HasPrefix(text, wakeWord) {
		confidence += 0.2
	}

	// Bonus für kurzen Text (weniger Rauschen)
	words := strings.Fields(text)
	if len(words) <= 3 {
		confidence += 0.1
	}

	// Malus für sehr langen Text (wahrscheinlich kein Wake Word)
	if len(words) > 10 {
		confidence -= 0.3
	}

	// Normalisieren auf 0-1
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.0 {
		confidence = 0.0
	}

	return confidence
}

// DetectFromStream ist eine kontinuierliche Wake Word Erkennung
// Gibt einen Channel zurück der bei Erkennung triggert
func (wwd *WakeWordDetector) DetectFromStream(audioCapture *AudioCapture, checkIntervalMs int) (<-chan WakeWordResult, chan<- struct{}) {
	resultChan := make(chan WakeWordResult, 10)
	stopChan := make(chan struct{})

	go func() {
		defer close(resultChan)

		ticker := time.NewTicker(time.Duration(checkIntervalMs) * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				if !wwd.IsEnabled() {
					continue
				}

				// Letzte 2 Sekunden Audio holen
				audioData, err := audioCapture.GetLastNSecondsAsWAV(2)
				if err != nil {
					log.Printf("Audio abrufen: %v", err)
					continue
				}

				// Wake Word prüfen
				result, err := wwd.Detect(audioData)
				if err != nil {
					log.Printf("Wake Word Detection: %v", err)
					continue
				}

				if result.Detected {
					resultChan <- *result
				}
			}
		}
	}()

	return resultChan, stopChan
}

// WakeWordFromSettingsKey konvertiert Settings-Key zu Wake Words Liste
func WakeWordFromSettingsKey(key string, customWord string) []string {
	switch key {
	case "hey_ewa":
		return []string{"hey ewa", "hallo ewa", "hi ewa"}
	case "ewa":
		return []string{"ewa", "eva"}
	case "custom":
		if customWord != "" {
			return []string{strings.ToLower(customWord)}
		}
		return []string{"ewa"}
	default:
		return []string{"hey ewa", "ewa"}
	}
}

// Close schließt den Wake Word Detector
func (wwd *WakeWordDetector) Close() error {
	wwd.SetEnabled(false)
	log.Printf("Wake Word Detector geschlossen")
	return nil
}
