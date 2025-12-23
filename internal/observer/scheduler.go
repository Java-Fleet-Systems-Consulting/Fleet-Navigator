package observer

import (
	"context"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Scheduler verwaltet die zeitgesteuerte Datensammlung
type Scheduler struct {
	service     *Service
	stopCh      chan struct{}
	running     bool
	mu          sync.Mutex
	nextRun     time.Time
	collectTime string // Format: "HH:MM"
}

// NewScheduler erstellt einen neuen Scheduler
func NewScheduler(service *Service) *Scheduler {
	return &Scheduler{
		service:     service,
		stopCh:      make(chan struct{}),
		collectTime: "06:00",
	}
}

// Start startet den Scheduler
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	go s.run()
	log.Printf("Observer Scheduler: Gestartet (Sammelzeit: %s)", s.collectTime)
}

// Stop stoppt den Scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stopCh)
	s.mu.Unlock()

	log.Printf("Observer Scheduler: Gestoppt")
}

// Reconfigure konfiguriert den Scheduler neu
func (s *Scheduler) Reconfigure(config *ObserverConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if config.DailyCollectionTime != "" {
		s.collectTime = config.DailyCollectionTime
	}

	// NextRun neu berechnen
	s.nextRun = s.calculateNextRun()

	log.Printf("Observer Scheduler: Neu konfiguriert (Sammelzeit: %s, nächster Lauf: %s)",
		s.collectTime, s.nextRun.Format("2006-01-02 15:04"))
}

// GetNextRun gibt den nächsten geplanten Lauf zurück
func (s *Scheduler) GetNextRun() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.nextRun
}

// IsRunning prüft ob der Scheduler läuft
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// run ist die Haupt-Schleife des Schedulers
func (s *Scheduler) run() {
	// Initial nextRun berechnen
	s.mu.Lock()
	s.nextRun = s.calculateNextRun()
	s.mu.Unlock()

	log.Printf("Observer Scheduler: Nächster Lauf geplant für %s", s.nextRun.Format("2006-01-02 15:04"))

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return

		case now := <-ticker.C:
			s.mu.Lock()
			nextRun := s.nextRun
			s.mu.Unlock()

			// Prüfen ob es Zeit für den Lauf ist
			if now.After(nextRun) || now.Equal(nextRun) {
				log.Printf("Observer Scheduler: Starte geplanten Sammellauf")

				// Sammellauf ausführen
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				run, err := s.service.RunNow(ctx)
				cancel()

				if err != nil {
					log.Printf("Observer Scheduler: Fehler beim Sammellauf: %v", err)
				} else {
					log.Printf("Observer Scheduler: Sammellauf %d abgeschlossen - %d Werte",
						run.ID, run.TotalRecords)
				}

				// Auto-Backfill ausführen (wenn konfiguriert)
				config := s.service.GetConfig()
				if config.AutoBackfill {
					ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Minute)
					s.service.AutoBackfill(ctx2)
					cancel2()
				}

				// Nächsten Lauf berechnen
				s.mu.Lock()
				s.nextRun = s.calculateNextRun()
				s.mu.Unlock()

				log.Printf("Observer Scheduler: Nächster Lauf geplant für %s", s.nextRun.Format("2006-01-02 15:04"))
			}
		}
	}
}

// calculateNextRun berechnet den nächsten Laufzeitpunkt
func (s *Scheduler) calculateNextRun() time.Time {
	now := time.Now()

	// collectTime parsen (Format: "HH:MM")
	parts := strings.Split(s.collectTime, ":")
	hour := 6  // Default
	minute := 0

	if len(parts) >= 1 {
		if h, err := strconv.Atoi(parts[0]); err == nil {
			hour = h
		}
	}
	if len(parts) >= 2 {
		if m, err := strconv.Atoi(parts[1]); err == nil {
			minute = m
		}
	}

	// Heutigen Laufzeitpunkt berechnen
	nextRun := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())

	// Wenn schon vorbei, dann morgen
	if nextRun.Before(now) || nextRun.Equal(now) {
		nextRun = nextRun.Add(24 * time.Hour)
	}

	// Wochenenden überspringen (optional - nur Werktage)
	// Für Finanzdaten macht das Sinn, da an Wochenenden keine Updates kommen
	for nextRun.Weekday() == time.Saturday || nextRun.Weekday() == time.Sunday {
		nextRun = nextRun.Add(24 * time.Hour)
	}

	return nextRun
}

// RunManualBackfill führt einen manuellen Backfill aus
func (s *Scheduler) RunManualBackfill(ctx context.Context, days int) (*ObservationRun, error) {
	now := time.Now()
	from := now.AddDate(0, 0, -days)

	log.Printf("Observer Scheduler: Starte manuellen Backfill für %d Tage", days)

	return s.service.Backfill(ctx, from, now)
}
