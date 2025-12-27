package llamaserver

import (
	"testing"
	"time"
)

// TestDefaultWatchdogConfig prüft die Standard-Konfiguration
func TestDefaultWatchdogConfig(t *testing.T) {
	config := DefaultWatchdogConfig(2026)

	if !config.Enabled {
		t.Error("Watchdog sollte standardmäßig aktiviert sein")
	}
	if config.HealthCheckURL != "http://127.0.0.1:2026/health" {
		t.Errorf("Falscher HealthCheckURL: %s", config.HealthCheckURL)
	}
	if config.CheckInterval != 5*time.Second {
		t.Errorf("Erwartet 5s CheckInterval, bekam %v", config.CheckInterval)
	}
	if config.MaxFailures != 3 {
		t.Errorf("Erwartet 3 MaxFailures, bekam %d", config.MaxFailures)
	}
	if config.InitialBackoff != 1*time.Second {
		t.Errorf("Erwartet 1s InitialBackoff, bekam %v", config.InitialBackoff)
	}
	if config.MaxBackoff != 60*time.Second {
		t.Errorf("Erwartet 60s MaxBackoff, bekam %v", config.MaxBackoff)
	}
	if config.BackoffMultiplier != 2.0 {
		t.Errorf("Erwartet 2.0 BackoffMultiplier, bekam %f", config.BackoffMultiplier)
	}
}

// TestNewWatchdog testet die Watchdog-Erstellung
func TestNewWatchdog(t *testing.T) {
	srv := NewServer(Config{Port: 2026})
	config := DefaultWatchdogConfig(2026)

	watchdog := NewWatchdog(srv, config)

	if watchdog == nil {
		t.Fatal("Watchdog sollte nicht nil sein")
	}
	if watchdog.server != srv {
		t.Error("Watchdog sollte den Server referenzieren")
	}
	if watchdog.IsRunning() {
		t.Error("Watchdog sollte nach Erstellung nicht laufen")
	}
}

// TestWatchdogStartStop testet Start und Stop
func TestWatchdogStartStop(t *testing.T) {
	srv := NewServer(Config{Port: 2026})
	config := DefaultWatchdogConfig(2026)
	config.CheckInterval = 100 * time.Millisecond // Schneller für Test

	watchdog := NewWatchdog(srv, config)

	// Start
	err := watchdog.Start()
	if err != nil {
		t.Errorf("Start sollte nicht fehlschlagen: %v", err)
	}
	if !watchdog.IsRunning() {
		t.Error("Watchdog sollte nach Start laufen")
	}

	// Doppelter Start sollte fehlschlagen
	err = watchdog.Start()
	if err == nil {
		t.Error("Doppelter Start sollte Fehler geben")
	}

	// Stop
	watchdog.Stop()
	if watchdog.IsRunning() {
		t.Error("Watchdog sollte nach Stop nicht mehr laufen")
	}

	// Doppelter Stop sollte nicht crashen
	watchdog.Stop()
}

// TestWatchdogStats testet die Statistik-Sammlung
func TestWatchdogStats(t *testing.T) {
	srv := NewServer(Config{Port: 2026})
	config := DefaultWatchdogConfig(2026)

	watchdog := NewWatchdog(srv, config)

	stats := watchdog.GetStats()
	if stats.TotalRestarts != 0 {
		t.Error("Initialer TotalRestarts sollte 0 sein")
	}
	if stats.IsHealthy {
		t.Error("Initialer IsHealthy sollte false sein")
	}
	if stats.CurrentBackoff != config.InitialBackoff {
		t.Errorf("Initialer Backoff sollte %v sein, ist %v", config.InitialBackoff, stats.CurrentBackoff)
	}
}

// TestWatchdogCallback testet den Restart-Callback
func TestWatchdogCallback(t *testing.T) {
	srv := NewServer(Config{Port: 2026})
	config := DefaultWatchdogConfig(2026)

	watchdog := NewWatchdog(srv, config)

	callbackCalled := false
	callbackReason := ""
	callbackAttempt := 0

	watchdog.SetRestartCallback(func(reason string, attempt int) {
		callbackCalled = true
		callbackReason = reason
		callbackAttempt = attempt
	})

	// Simuliere Callback-Aufruf (intern)
	watchdog.mu.RLock()
	callback := watchdog.onRestart
	watchdog.mu.RUnlock()

	if callback != nil {
		callback("Test-Reason", 1)
	}

	if !callbackCalled {
		t.Error("Callback sollte aufgerufen worden sein")
	}
	if callbackReason != "Test-Reason" {
		t.Errorf("Callback-Reason sollte 'Test-Reason' sein, ist '%s'", callbackReason)
	}
	if callbackAttempt != 1 {
		t.Errorf("Callback-Attempt sollte 1 sein, ist %d", callbackAttempt)
	}
}

// TestWatchdogResetBackoff testet den Backoff-Reset
func TestWatchdogResetBackoff(t *testing.T) {
	srv := NewServer(Config{Port: 2026})
	config := DefaultWatchdogConfig(2026)

	watchdog := NewWatchdog(srv, config)

	// Backoff erhöhen (simuliert)
	watchdog.mu.Lock()
	watchdog.stats.CurrentBackoff = 30 * time.Second
	watchdog.stats.ConsecutiveFails = 5
	watchdog.mu.Unlock()

	// Reset
	watchdog.resetBackoff()

	stats := watchdog.GetStats()
	if stats.CurrentBackoff != config.InitialBackoff {
		t.Errorf("Backoff nach Reset sollte %v sein, ist %v", config.InitialBackoff, stats.CurrentBackoff)
	}
	if stats.ConsecutiveFails != 0 {
		t.Error("ConsecutiveFails nach Reset sollte 0 sein")
	}
}

// TestServerEnableWatchdog testet die Server-Integration
func TestServerEnableWatchdog(t *testing.T) {
	srv := NewServer(Config{Port: 2026})

	if srv.IsWatchdogEnabled() {
		t.Error("Watchdog sollte initial deaktiviert sein")
	}

	err := srv.EnableWatchdog()
	if err != nil {
		t.Errorf("EnableWatchdog sollte nicht fehlschlagen: %v", err)
	}

	if !srv.IsWatchdogEnabled() {
		t.Error("Watchdog sollte jetzt aktiviert sein")
	}

	stats := srv.GetWatchdogStats()
	if stats == nil {
		t.Error("GetWatchdogStats sollte nicht nil sein wenn aktiviert")
	}

	srv.DisableWatchdog()
	if srv.IsWatchdogEnabled() {
		t.Error("Watchdog sollte nach Disable deaktiviert sein")
	}
}

// TestServerWatchdogDoubleEnable testet doppelte Aktivierung
func TestServerWatchdogDoubleEnable(t *testing.T) {
	srv := NewServer(Config{Port: 2026})

	err1 := srv.EnableWatchdog()
	if err1 != nil {
		t.Errorf("Erste Aktivierung sollte erfolgreich sein: %v", err1)
	}

	err2 := srv.EnableWatchdog()
	if err2 == nil {
		t.Error("Zweite Aktivierung sollte Fehler geben")
	}

	srv.DisableWatchdog()
}

// TestForceWatchdogRestart_NotEnabled testet Restart ohne aktiven Watchdog
func TestForceWatchdogRestart_NotEnabled(t *testing.T) {
	srv := NewServer(Config{Port: 2026})

	err := srv.ForceWatchdogRestart("Test")
	if err == nil {
		t.Error("ForceRestart ohne aktivierten Watchdog sollte Fehler geben")
	}
}
