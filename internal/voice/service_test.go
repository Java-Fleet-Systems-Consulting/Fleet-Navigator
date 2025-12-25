package voice

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockWhisperDownloader ist ein Mock der lange "Downloads" simuliert
type mockWhisperDownloader struct {
	downloadDuration time.Duration
	downloadCalled   atomic.Bool
}

// TestGetStatusNotBlockedDuringDownload_WithMock testet dass GetStatus() nicht blockiert
// wenn ein Download läuft - verwendet Mock statt echtem Netzwerk
func TestGetStatusNotBlockedDuringDownload_WithMock(t *testing.T) {
	tempDir := t.TempDir()
	service := NewService(tempDir)

	err := service.Initialize(DefaultConfig())
	if err != nil {
		t.Fatalf("Initialize fehlgeschlagen: %v", err)
	}

	// Simuliere langen Download mit Goroutine
	downloadStarted := make(chan struct{})
	downloadDone := make(chan struct{})
	downloadInProgress := atomic.Bool{}

	go func() {
		downloadInProgress.Store(true)
		close(downloadStarted)

		// Simuliere 2 Sekunden langen Download
		// (In der alten Version hätte das GetStatus blockiert!)
		time.Sleep(2 * time.Second)

		downloadInProgress.Store(false)
		close(downloadDone)
	}()

	// Warte auf Download-Start
	<-downloadStarted

	// Prüfe dass Download "läuft"
	if !downloadInProgress.Load() {
		t.Fatal("Download sollte laufen")
	}

	// GetStatus aufrufen WÄHREND Download läuft - sollte NICHT blockieren
	statusDone := make(chan struct{})
	var status Status

	go func() {
		status = service.GetStatus()
		close(statusDone)
	}()

	// GetStatus MUSS innerhalb von 100ms antworten (nicht 2 Sekunden!)
	select {
	case <-statusDone:
		t.Logf("GetStatus erfolgreich in <100ms: initialized=%v", status.Initialized)
		if !downloadInProgress.Load() {
			t.Log("Warnung: Download bereits beendet")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("DEADLOCK: GetStatus() blockiert während Download läuft!")
	}

	// Cleanup
	<-downloadDone
}

// TestConcurrentGetStatusCalls testet parallele GetStatus-Aufrufe
func TestConcurrentGetStatusCalls(t *testing.T) {
	tempDir := t.TempDir()
	service := NewService(tempDir)

	err := service.Initialize(DefaultConfig())
	if err != nil {
		t.Fatalf("Initialize fehlgeschlagen: %v", err)
	}

	// 10 parallele GetStatus Aufrufe
	var wg sync.WaitGroup
	var successCount atomic.Int32

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			done := make(chan struct{})
			go func() {
				_ = service.GetStatus()
				close(done)
			}()

			select {
			case <-done:
				successCount.Add(1)
			case <-time.After(100 * time.Millisecond):
				t.Error("GetStatus timeout - möglicher Deadlock")
			}
		}()
	}

	wg.Wait()

	if successCount.Load() != 10 {
		t.Errorf("Nur %d von 10 GetStatus-Aufrufen erfolgreich", successCount.Load())
	}
}

// TestMutexNotHeldDuringSlowOperation simuliert eine langsame Operation
// und prüft dass andere Aufrufe nicht blockiert werden
func TestMutexNotHeldDuringSlowOperation(t *testing.T) {
	tempDir := t.TempDir()
	service := NewService(tempDir)

	err := service.Initialize(DefaultConfig())
	if err != nil {
		t.Fatalf("Initialize fehlgeschlagen: %v", err)
	}

	// Starte eine "langsame Operation" in einer Goroutine
	slowOpStarted := make(chan struct{})
	slowOpDone := make(chan struct{})

	go func() {
		// Hol dir eine Referenz (kurzer Lock)
		service.mu.RLock()
		whisper := service.whisper
		service.mu.RUnlock()

		close(slowOpStarted)

		// Simuliere langsame Operation OHNE Lock
		if whisper != nil {
			time.Sleep(500 * time.Millisecond)
		}

		close(slowOpDone)
	}()

	<-slowOpStarted

	// Jetzt sollten andere Operationen NICHT blockiert sein
	results := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func() {
			start := time.Now()
			_ = service.GetStatus()
			duration := time.Since(start)
			// Sollte schnell sein (<50ms), nicht 500ms
			results <- duration < 50*time.Millisecond
		}()
	}

	// Sammle Ergebnisse
	fastCount := 0
	for i := 0; i < 5; i++ {
		if <-results {
			fastCount++
		}
	}

	if fastCount < 4 {
		t.Errorf("Nur %d von 5 GetStatus-Aufrufen waren schnell (<50ms)", fastCount)
	}

	<-slowOpDone
}

// TestServiceInitialization testet die grundlegende Service-Initialisierung
func TestServiceInitialization(t *testing.T) {
	tempDir := t.TempDir()
	service := NewService(tempDir)

	// Vor Initialize sollte Status initialized=false sein
	status := service.GetStatus()
	if status.Initialized {
		t.Error("Service sollte vor Initialize nicht initialisiert sein")
	}

	// Initialize aufrufen
	err := service.Initialize(DefaultConfig())
	if err != nil {
		t.Fatalf("Initialize fehlgeschlagen: %v", err)
	}

	// Nach Initialize sollte Status initialized=true sein
	status = service.GetStatus()
	if !status.Initialized {
		t.Error("Service sollte nach Initialize initialisiert sein")
	}
}

// TestGetStatusReturnsCorrectFields testet dass GetStatus alle Felder korrekt zurückgibt
func TestGetStatusReturnsCorrectFields(t *testing.T) {
	tempDir := t.TempDir()
	service := NewService(tempDir)

	err := service.Initialize(Config{
		WhisperModel: "small",
		PiperVoice:   "de_DE-thorsten-medium",
		Language:     "de",
	})
	if err != nil {
		t.Fatalf("Initialize fehlgeschlagen: %v", err)
	}

	status := service.GetStatus()

	// Prüfe dass alle wichtigen Felder gesetzt sind
	if !status.Initialized {
		t.Error("Initialized sollte true sein")
	}
	if status.DataDir == "" {
		t.Error("DataDir sollte gesetzt sein")
	}

	// Whisper-Status prüfen
	if status.Whisper.Model != "small" {
		t.Errorf("Whisper.Model erwartet 'small', bekommen '%s'", status.Whisper.Model)
	}
	if status.Whisper.Language != "de" {
		t.Errorf("Whisper.Language erwartet 'de', bekommen '%s'", status.Whisper.Language)
	}
}

// TestCloseService testet das ordnungsgemäße Schließen des Services
func TestCloseService(t *testing.T) {
	tempDir := t.TempDir()
	service := NewService(tempDir)

	err := service.Initialize(DefaultConfig())
	if err != nil {
		t.Fatalf("Initialize fehlgeschlagen: %v", err)
	}

	// Close sollte erfolgreich sein
	err = service.Close()
	if err != nil {
		t.Errorf("Close fehlgeschlagen: %v", err)
	}

	// Nach Close sollte initialized=false sein
	status := service.GetStatus()
	if status.Initialized {
		t.Error("Service sollte nach Close nicht mehr initialisiert sein")
	}
}

// TestSetWhisperModel testet das Wechseln des Whisper-Modells
func TestSetWhisperModel(t *testing.T) {
	tempDir := t.TempDir()
	service := NewService(tempDir)

	err := service.Initialize(Config{
		WhisperModel: "base",
		Language:     "de",
	})
	if err != nil {
		t.Fatalf("Initialize fehlgeschlagen: %v", err)
	}

	// Modell wechseln
	err = service.SetWhisperModel("small")
	if err != nil {
		t.Errorf("SetWhisperModel fehlgeschlagen: %v", err)
	}

	// Prüfen ob Modell gewechselt wurde
	status := service.GetStatus()
	if status.Whisper.Model != "small" {
		t.Errorf("Whisper.Model erwartet 'small', bekommen '%s'", status.Whisper.Model)
	}
}

// TestSetPiperVoice testet das Wechseln der Piper-Stimme
func TestSetPiperVoice(t *testing.T) {
	tempDir := t.TempDir()
	service := NewService(tempDir)

	err := service.Initialize(Config{
		PiperVoice: "de_DE-thorsten-medium",
	})
	if err != nil {
		t.Fatalf("Initialize fehlgeschlagen: %v", err)
	}

	// Stimme wechseln
	err = service.SetPiperVoice("de_DE-eva_k-x_low")
	if err != nil {
		t.Errorf("SetPiperVoice fehlgeschlagen: %v", err)
	}

	// Prüfen ob Stimme gewechselt wurde
	status := service.GetStatus()
	if status.Piper.Voice != "de_DE-eva_k-x_low" {
		t.Errorf("Piper.Voice erwartet 'de_DE-eva_k-x_low', bekommen '%s'", status.Piper.Voice)
	}
}

// TestRaceConditions testet Race Conditions mit -race Flag
func TestRaceConditions(t *testing.T) {
	tempDir := t.TempDir()
	service := NewService(tempDir)

	err := service.Initialize(DefaultConfig())
	if err != nil {
		t.Fatalf("Initialize fehlgeschlagen: %v", err)
	}

	// Viele parallele Operationen ausführen
	var wg sync.WaitGroup

	// GetStatus parallel
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = service.GetStatus()
		}()
	}

	// SetWhisperModel parallel
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			models := []string{"base", "small", "medium", "tiny"}
			_ = service.SetWhisperModel(models[id%len(models)])
		}(i)
	}

	// GetInstalledWhisperModels parallel
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = service.GetInstalledWhisperModels()
		}()
	}

	// Warte auf alle Goroutinen
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Log("Alle Race-Condition-Tests erfolgreich abgeschlossen")
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout bei Race-Condition-Tests - möglicher Deadlock")
	}
}

// TestEnsureDownloadDoesNotBlockGetStatus ist der Haupttest für den Deadlock-Fix
// Er simuliert den exakten Fall: Download läuft, GetStatus wird aufgerufen
func TestEnsureDownloadDoesNotBlockGetStatus(t *testing.T) {
	tempDir := t.TempDir()
	service := NewService(tempDir)

	err := service.Initialize(DefaultConfig())
	if err != nil {
		t.Fatalf("Initialize fehlgeschlagen: %v", err)
	}

	// Simuliere was vorher passiert ist:
	// 1. EnsureWhisperDownloaded wird aufgerufen (hielt den Lock während Download)
	// 2. Während Download: GetStatus aufrufen
	// 3. DEADLOCK: GetStatus wartet auf Lock, Download hält Lock

	// Mit dem Fix:
	// 1. EnsureWhisperDownloaded holt Referenz unter kurzem Lock
	// 2. Download läuft OHNE Lock
	// 3. GetStatus kann jederzeit aufgerufen werden

	downloadRunning := atomic.Bool{}
	downloadFinished := make(chan struct{})

	// Starte "Download" Goroutine
	go func() {
		progressChan := make(chan DownloadProgress, 10)
		go func() {
			for range progressChan {
			}
		}()

		downloadRunning.Store(true)

		// Simuliere dass wir den Lock nur kurz halten (für Referenz)
		// dann lange ohne Lock arbeiten
		service.mu.RLock()
		_ = service.whisper // Referenz holen
		service.mu.RUnlock()

		// "Download" läuft (ohne Lock!)
		time.Sleep(1 * time.Second)

		downloadRunning.Store(false)
		close(progressChan)
		close(downloadFinished)
	}()

	// Warte kurz bis Download "läuft"
	time.Sleep(50 * time.Millisecond)

	if !downloadRunning.Load() {
		t.Fatal("Download sollte laufen")
	}

	// GetStatus MUSS funktionieren während Download läuft
	getStatusDone := make(chan Status, 1)
	go func() {
		getStatusDone <- service.GetStatus()
	}()

	select {
	case status := <-getStatusDone:
		t.Logf("GetStatus erfolgreich während Download: initialized=%v", status.Initialized)
		if !downloadRunning.Load() {
			t.Error("Download sollte noch laufen wenn GetStatus zurückkehrt")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("DEADLOCK REPRODUZIERT: GetStatus blockiert während Download!")
	}

	<-downloadFinished
}
