package llamaserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// WatchdogConfig konfiguriert den Watchdog
type WatchdogConfig struct {
	Enabled           bool          `json:"enabled"`
	HealthCheckURL    string        `json:"healthCheckUrl"`    // z.B. "http://127.0.0.1:2026/health"
	CheckInterval     time.Duration `json:"checkInterval"`     // Wie oft prüfen (default: 5s)
	MaxFailures       int           `json:"maxFailures"`       // Nach wie vielen Fehlern neu starten (default: 3)
	InitialBackoff    time.Duration `json:"initialBackoff"`    // Erste Wartezeit nach Crash (default: 1s)
	MaxBackoff        time.Duration `json:"maxBackoff"`        // Maximale Wartezeit (default: 60s)
	BackoffMultiplier float64       `json:"backoffMultiplier"` // Multiplikator (default: 2.0)
	MaxRestarts       int           `json:"maxRestarts"`       // Max Restarts bevor Aufgeben (0 = unbegrenzt)
}

// DefaultWatchdogConfig gibt die Standard-Konfiguration zurück
func DefaultWatchdogConfig(port int) WatchdogConfig {
	return WatchdogConfig{
		Enabled:           true,
		HealthCheckURL:    fmt.Sprintf("http://127.0.0.1:%d/health", port),
		CheckInterval:     5 * time.Second,
		MaxFailures:       3,
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        60 * time.Second,
		BackoffMultiplier: 2.0,
		MaxRestarts:       0, // Unbegrenzt
	}
}

// Watchdog überwacht den llama-server und startet ihn bei Bedarf neu
type Watchdog struct {
	server    *Server
	config    WatchdogConfig
	ctx       context.Context
	cancel    context.CancelFunc
	running   bool
	mu        sync.RWMutex
	stats     WatchdogStats
	onRestart func(reason string, attempt int) // Callback bei Restart
}

// WatchdogStats enthält Statistiken über den Watchdog
type WatchdogStats struct {
	StartTime        time.Time     `json:"startTime"`
	TotalRestarts    int           `json:"totalRestarts"`
	LastRestartTime  time.Time     `json:"lastRestartTime"`
	LastRestartReason string       `json:"lastRestartReason"`
	ConsecutiveFails int           `json:"consecutiveFails"`
	CurrentBackoff   time.Duration `json:"currentBackoff"`
	IsHealthy        bool          `json:"isHealthy"`
	LastHealthCheck  time.Time     `json:"lastHealthCheck"`
}

// NewWatchdog erstellt einen neuen Watchdog für den Server
func NewWatchdog(server *Server, config WatchdogConfig) *Watchdog {
	return &Watchdog{
		server: server,
		config: config,
		stats: WatchdogStats{
			CurrentBackoff: config.InitialBackoff,
			IsHealthy:      false,
		},
	}
}

// SetRestartCallback setzt den Callback der bei Restarts aufgerufen wird
func (w *Watchdog) SetRestartCallback(callback func(reason string, attempt int)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onRestart = callback
}

// Start startet den Watchdog
func (w *Watchdog) Start() error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return fmt.Errorf("Watchdog läuft bereits")
	}

	w.ctx, w.cancel = context.WithCancel(context.Background())
	w.running = true
	w.stats.StartTime = time.Now()
	w.mu.Unlock()

	log.Printf("🐕 Watchdog gestartet (Check-Interval: %v, Max-Failures: %d)",
		w.config.CheckInterval, w.config.MaxFailures)

	go w.monitor()
	return nil
}

// Stop stoppt den Watchdog
func (w *Watchdog) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return
	}

	if w.cancel != nil {
		w.cancel()
	}
	w.running = false
	log.Printf("🐕 Watchdog gestoppt")
}

// IsRunning prüft ob der Watchdog läuft
func (w *Watchdog) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}

// GetStats gibt die aktuellen Statistiken zurück
func (w *Watchdog) GetStats() WatchdogStats {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.stats
}

// monitor ist die Haupt-Überwachungsschleife
func (w *Watchdog) monitor() {
	client := &http.Client{Timeout: 3 * time.Second}
	ticker := time.NewTicker(w.config.CheckInterval)
	defer ticker.Stop()

	failCount := 0

	for {
		select {
		case <-w.ctx.Done():
			log.Printf("🐕 Watchdog-Monitor beendet")
			return

		case <-ticker.C:
			healthy := w.checkHealth(client)

			w.mu.Lock()
			w.stats.LastHealthCheck = time.Now()
			w.stats.IsHealthy = healthy
			w.mu.Unlock()

			if healthy {
				// Server ist gesund - Reset
				if failCount > 0 {
					log.Printf("🐕 llama-server wieder erreichbar (nach %d Fehlversuchen)", failCount)
				}
				failCount = 0
				w.resetBackoff()
			} else {
				failCount++
				w.mu.Lock()
				w.stats.ConsecutiveFails = failCount
				w.mu.Unlock()

				log.Printf("🐕 Health-Check fehlgeschlagen (%d/%d)", failCount, w.config.MaxFailures)

				if failCount >= w.config.MaxFailures {
					w.handleServerDown("Health-Check fehlgeschlagen", failCount)
					failCount = 0
				}
			}
		}
	}
}

// checkHealth prüft ob der llama-server antwortet
func (w *Watchdog) checkHealth(client *http.Client) bool {
	// Erst prüfen ob der Prozess überhaupt läuft
	if !w.server.IsRunning() {
		return false
	}

	// HTTP Health-Check
	resp, err := client.Get(w.config.HealthCheckURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// handleServerDown behandelt einen Server-Ausfall
func (w *Watchdog) handleServerDown(reason string, failCount int) {
	w.mu.Lock()
	restartCount := w.stats.TotalRestarts
	maxRestarts := w.config.MaxRestarts
	currentBackoff := w.stats.CurrentBackoff
	w.mu.Unlock()

	// Prüfen ob Max-Restarts erreicht
	if maxRestarts > 0 && restartCount >= maxRestarts {
		log.Printf("🐕 ❌ Max-Restarts erreicht (%d) - Watchdog gibt auf", maxRestarts)
		w.Stop()
		return
	}

	log.Printf("🐕 🔄 Server-Neustart erforderlich: %s", reason)

	// Callback aufrufen wenn gesetzt
	w.mu.RLock()
	callback := w.onRestart
	w.mu.RUnlock()

	if callback != nil {
		callback(reason, restartCount+1)
	}

	// Backoff-Wartezeit
	if restartCount > 0 {
		log.Printf("🐕 Warte %v vor Neustart (Backoff)...", currentBackoff)
		select {
		case <-w.ctx.Done():
			return
		case <-time.After(currentBackoff):
		}
	}

	// Server neu starten
	err := w.restartServer()

	w.mu.Lock()
	w.stats.TotalRestarts++
	w.stats.LastRestartTime = time.Now()
	w.stats.LastRestartReason = reason
	w.stats.ConsecutiveFails = 0

	if err != nil {
		log.Printf("🐕 ❌ Neustart fehlgeschlagen: %v", err)
		// Backoff erhöhen
		w.stats.CurrentBackoff = time.Duration(float64(w.stats.CurrentBackoff) * w.config.BackoffMultiplier)
		if w.stats.CurrentBackoff > w.config.MaxBackoff {
			w.stats.CurrentBackoff = w.config.MaxBackoff
		}
	} else {
		log.Printf("🐕 ✅ Server erfolgreich neu gestartet (Versuch #%d)", w.stats.TotalRestarts)
	}
	w.mu.Unlock()
}

// restartServer startet den llama-server neu
func (w *Watchdog) restartServer() error {
	// Aktuelles Modell merken
	status := w.server.GetStatus()
	modelPath := status.ModelPath

	if modelPath == "" {
		// Kein Modell geladen - versuche das letzte zu laden
		modelPath = w.server.config.ModelPath
	}

	if modelPath == "" {
		return fmt.Errorf("kein Modell zum Neustarten bekannt")
	}

	log.Printf("🐕 Starte llama-server mit Modell: %s", modelPath)

	// Server stoppen (falls noch Prozess läuft)
	w.server.Stop()

	// Kurz warten
	time.Sleep(500 * time.Millisecond)

	// Server neu starten
	err := w.server.Start(modelPath)
	if err != nil {
		return fmt.Errorf("Start fehlgeschlagen: %w", err)
	}

	// Warten bis Server bereit ist
	return w.waitForReady(30 * time.Second)
}

// waitForReady wartet bis der Server bereit ist
func (w *Watchdog) waitForReady(timeout time.Duration) error {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-w.ctx.Done():
			return fmt.Errorf("Watchdog gestoppt")
		default:
		}

		resp, err := client.Get(w.config.HealthCheckURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("Server nicht bereit nach %v", timeout)
}

// resetBackoff setzt den Backoff zurück
func (w *Watchdog) resetBackoff() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.stats.CurrentBackoff = w.config.InitialBackoff
	w.stats.ConsecutiveFails = 0
}

// ForceRestart erzwingt einen Neustart (für manuelle Intervention)
func (w *Watchdog) ForceRestart(reason string) error {
	log.Printf("🐕 Manueller Neustart angefordert: %s", reason)

	w.mu.Lock()
	w.stats.TotalRestarts++
	w.stats.LastRestartTime = time.Now()
	w.stats.LastRestartReason = "Manuell: " + reason
	w.mu.Unlock()

	return w.restartServer()
}
