package observer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// ObserverConfig enthält die Observer-Konfiguration
type ObserverConfig struct {
	// Aktiviert/Deaktiviert den Observer
	Enabled bool `json:"enabled"`

	// Strategie (CONSERVATIVE, MODERATE, AGGRESSIVE)
	Strategy Strategy `json:"strategy"`

	// Sammlung täglich um diese Uhrzeit (Format: "HH:MM")
	DailyCollectionTime string `json:"dailyCollectionTime"`

	// Automatisches Backfill aktiviert
	AutoBackfill bool `json:"autoBackfill"`

	// Maximale Backfill-Tage
	MaxBackfillDays int `json:"maxBackfillDays"`

	// Prompt für Transparenz - beschreibt was der Observer macht
	Prompt string `json:"prompt"`

	// Aktive Quellen (Source-Codes)
	ActiveSources []string `json:"activeSources"`

	// Aktive Indikatoren (Indikator-Codes)
	ActiveIndicators []string `json:"activeIndicators"`
}

// DefaultConfig gibt die Standard-Konfiguration zurück
func DefaultConfig() *ObserverConfig {
	return &ObserverConfig{
		Enabled:             false, // Muss explizit aktiviert werden
		Strategy:            StrategyConservative,
		DailyCollectionTime: "06:00",
		AutoBackfill:        true,
		MaxBackfillDays:     365,
		Prompt: `Der Observer ist eine neutrale, unsichtbare Systeminstanz des Fleet Navigators.
Seine Aufgabe besteht ausschließlich darin, relevante Finanz- und Wirtschaftsdaten
regelmäßig zu beobachten, strukturiert zu erfassen und revisionssicher abzulegen.

Der Observer hat keine Meinung, gibt keine Empfehlungen und tritt nie nach außen auf.
Er arbeitet faktenbasiert, regelgebunden und deterministisch.

Aktuelle Strategie: CONSERVATIVE
- Nur offizielle Quellen (EZB, Bundesbank)
- Täglich um 06:00 Uhr
- Kernkennzahlen: Leitzinsen, Inflation, Arbeitslosigkeit`,
		ActiveSources:    []string{"ECB", "BUNDESBANK", "ESTR"},
		ActiveIndicators: []string{}, // Leer = alle aktiven
	}
}

// Service ist der Haupt-Service für den Observer
type Service struct {
	repo      *Repository
	registry  *CollectorRegistry
	config    *ObserverConfig
	configMu  sync.RWMutex
	scheduler *Scheduler
	running   bool
	runMu     sync.Mutex
}

// NewService erstellt einen neuen Observer-Service
func NewService(dataDir string) (*Service, error) {
	repo, err := NewRepository(dataDir)
	if err != nil {
		return nil, err
	}

	// Seed-Daten einfügen
	if err := repo.SeedDefaultData(); err != nil {
		log.Printf("Observer: Seed-Daten Warnung: %v", err)
	}

	// Collector Registry aufbauen
	registry := NewCollectorRegistry()
	// Basis-Daten (offizielle Quellen)
	registry.Register(NewECBCollector())
	registry.Register(NewBundesbankCollector())
	registry.Register(NewESTRCollector())
	// Marktdaten (semi-offizielle Quellen)
	registry.Register(NewCryptoCollector())
	registry.Register(NewStockCollector())
	registry.Register(NewCommodityCollector())

	service := &Service{
		repo:     repo,
		registry: registry,
		config:   DefaultConfig(),
	}

	// Scheduler erstellen
	service.scheduler = NewScheduler(service)

	return service, nil
}

// Close beendet den Service
func (s *Service) Close() error {
	if s.scheduler != nil {
		s.scheduler.Stop()
	}
	return s.repo.Close()
}

// GetConfig gibt die aktuelle Konfiguration zurück
func (s *Service) GetConfig() *ObserverConfig {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	// Kopie zurückgeben
	config := *s.config
	return &config
}

// SetConfig setzt die Konfiguration
func (s *Service) SetConfig(config *ObserverConfig) error {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	s.config = config

	// Scheduler neu konfigurieren
	if s.scheduler != nil {
		s.scheduler.Reconfigure(config)
	}

	log.Printf("Observer: Konfiguration aktualisiert (Enabled: %v, Strategy: %s)", config.Enabled, config.Strategy)
	return nil
}

// Start startet den Observer (inkl. Scheduler)
func (s *Service) Start() error {
	s.configMu.RLock()
	enabled := s.config.Enabled
	s.configMu.RUnlock()

	if !enabled {
		log.Printf("Observer: Nicht aktiviert")
		return nil
	}

	if s.scheduler != nil {
		s.scheduler.Start()
	}

	log.Printf("Observer: Gestartet")
	return nil
}

// Stop stoppt den Observer
func (s *Service) Stop() {
	if s.scheduler != nil {
		s.scheduler.Stop()
	}
	log.Printf("Observer: Gestoppt")
}

// RunNow führt einen Sammellauf sofort aus
func (s *Service) RunNow(ctx context.Context) (*ObservationRun, error) {
	s.runMu.Lock()
	if s.running {
		s.runMu.Unlock()
		return nil, fmt.Errorf("Sammellauf läuft bereits")
	}
	s.running = true
	s.runMu.Unlock()

	defer func() {
		s.runMu.Lock()
		s.running = false
		s.runMu.Unlock()
	}()

	return s.collect(ctx, false, nil, nil)
}

// Backfill führt einen Backfill für fehlende Daten aus
func (s *Service) Backfill(ctx context.Context, from, to time.Time) (*ObservationRun, error) {
	s.runMu.Lock()
	if s.running {
		s.runMu.Unlock()
		return nil, fmt.Errorf("Sammellauf läuft bereits")
	}
	s.running = true
	s.runMu.Unlock()

	defer func() {
		s.runMu.Lock()
		s.running = false
		s.runMu.Unlock()
	}()

	return s.collect(ctx, true, &from, &to)
}

// collect ist die interne Sammelfunktion
func (s *Service) collect(ctx context.Context, isBackfill bool, from, to *time.Time) (*ObservationRun, error) {
	s.configMu.RLock()
	config := *s.config
	s.configMu.RUnlock()

	// Run erstellen
	run := &ObservationRun{
		Strategy:   config.Strategy,
		StartedAt:  time.Now(),
		Status:     RunStatusRunning,
		IsBackfill: isBackfill,
	}
	if from != nil {
		run.BackfillFrom = from
	}
	if to != nil {
		run.BackfillTo = to
	}

	if err := s.repo.CreateRun(run); err != nil {
		return nil, fmt.Errorf("Run erstellen fehlgeschlagen: %w", err)
	}

	log.Printf("Observer: Sammellauf %d gestartet (Backfill: %v)", run.ID, isBackfill)

	// Aktive Indikatoren laden
	indicators, err := s.repo.GetAllIndicators(true)
	if err != nil {
		run.Status = RunStatusFailed
		run.ErrorMessages = fmt.Sprintf("Indikatoren laden fehlgeschlagen: %v", err)
		s.repo.UpdateRun(run)
		return run, err
	}

	// Nach Quelle gruppieren
	indicatorsBySource := make(map[int64][]Indicator)
	for _, ind := range indicators {
		indicatorsBySource[ind.SourceID] = append(indicatorsBySource[ind.SourceID], ind)
	}

	// Quellen laden
	sources, err := s.repo.GetAllSources(true)
	if err != nil {
		run.Status = RunStatusFailed
		run.ErrorMessages = fmt.Sprintf("Quellen laden fehlgeschlagen: %v", err)
		s.repo.UpdateRun(run)
		return run, err
	}

	sourceMap := make(map[int64]*DataSource)
	for i := range sources {
		sourceMap[sources[i].ID] = &sources[i]
	}

	// Sammeln pro Quelle
	var errors []string
	totalValues := 0

	for sourceID, sourceIndicators := range indicatorsBySource {
		source := sourceMap[sourceID]
		if source == nil {
			continue
		}

		// Collector für diese Quelle holen
		collector := s.registry.Get(source.Code)
		if collector == nil {
			log.Printf("Observer: Kein Collector für %s", source.Code)
			continue
		}

		// Prüfen ob Quelle in aktiven Quellen
		if len(config.ActiveSources) > 0 {
			found := false
			for _, code := range config.ActiveSources {
				if code == source.Code {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Sammeln
		var result *CollectorResult
		var err error

		if isBackfill && from != nil && to != nil {
			// Backfill für jeden Indikator einzeln
			for _, ind := range sourceIndicators {
				result, err = collector.CollectHistorical(ctx, ind, *from, *to)
				if err != nil {
					errors = append(errors, fmt.Sprintf("%s/%s: %v", source.Code, ind.Code, err))
					continue
				}
				if result != nil && len(result.Values) > 0 {
					// Werte speichern (mit Duplikat-Check)
					for _, val := range result.Values {
						val.RunID = run.ID
						// Prüfen ob bereits vorhanden
						exists, _ := s.repo.HasValueForDate(val.IndicatorID, val.ObservedAt)
						if !exists {
							if err := s.repo.CreateValue(&val); err != nil {
								log.Printf("Observer: Wert speichern fehlgeschlagen: %v", err)
							} else {
								totalValues++
							}
						}
					}
				}
			}
		} else {
			// Aktuelle Daten
			result, err = collector.Collect(ctx, sourceIndicators)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", source.Code, err))
				continue
			}
			if result != nil && len(result.Values) > 0 {
				// Werte speichern
				for i := range result.Values {
					result.Values[i].RunID = run.ID
				}
				if err := s.repo.CreateValues(result.Values); err != nil {
					errors = append(errors, fmt.Sprintf("%s: Speichern fehlgeschlagen: %v", source.Code, err))
				} else {
					totalValues += len(result.Values)
				}
			}
		}
	}

	// Run abschließen
	now := time.Now()
	run.FinishedAt = &now
	run.TotalRecords = totalValues
	run.ErrorCount = len(errors)

	if len(errors) > 0 {
		errJSON, _ := json.Marshal(errors)
		run.ErrorMessages = string(errJSON)
		if totalValues > 0 {
			run.Status = RunStatusPartial
		} else {
			run.Status = RunStatusFailed
		}
	} else {
		run.Status = RunStatusCompleted
	}

	s.repo.UpdateRun(run)

	log.Printf("Observer: Sammellauf %d beendet - %d Werte, %d Fehler, Status: %s",
		run.ID, totalValues, len(errors), run.Status)

	return run, nil
}

// AutoBackfill prüft und füllt fehlende Daten automatisch
func (s *Service) AutoBackfill(ctx context.Context) error {
	s.configMu.RLock()
	config := *s.config
	s.configMu.RUnlock()

	if !config.AutoBackfill {
		return nil
	}

	log.Printf("Observer: Starte Auto-Backfill...")

	// Für jeden aktiven Indikator prüfen
	indicators, err := s.repo.GetAllIndicators(true)
	if err != nil {
		return err
	}

	now := time.Now()
	from := now.AddDate(0, 0, -config.MaxBackfillDays)

	for _, ind := range indicators {
		// Fehlende Daten finden
		missing, err := s.repo.GetMissingDates(ind.ID, from, now, ind.Frequency)
		if err != nil {
			log.Printf("Observer: Fehlende Daten für %s nicht ermittelbar: %v", ind.Code, err)
			continue
		}

		if len(missing) == 0 {
			continue
		}

		log.Printf("Observer: %d fehlende Einträge für %s gefunden", len(missing), ind.Code)

		// Collector holen
		source, _ := s.repo.GetSourceByCode("")
		// Source durch ID holen
		sources, _ := s.repo.GetAllSources(false)
		for _, src := range sources {
			if src.ID == ind.SourceID {
				source = &src
				break
			}
		}
		if source == nil {
			continue
		}

		collector := s.registry.Get(source.Code)
		if collector == nil {
			continue
		}

		// Historische Daten holen
		result, err := collector.CollectHistorical(ctx, ind, from, now)
		if err != nil {
			log.Printf("Observer: Backfill für %s fehlgeschlagen: %v", ind.Code, err)
			continue
		}

		if result != nil && len(result.Values) > 0 {
			// Werte speichern (mit Duplikat-Check)
			added := 0
			for _, val := range result.Values {
				exists, _ := s.repo.HasValueForDate(val.IndicatorID, val.ObservedAt)
				if !exists {
					val.RunID = 0 // Kein Run für Auto-Backfill
					if err := s.repo.CreateValue(&val); err == nil {
						added++
					}
				}
			}
			log.Printf("Observer: %d neue Werte für %s hinzugefügt", added, ind.Code)
		}
	}

	return nil
}

// GetStats gibt Observer-Statistiken zurück
func (s *Service) GetStats() (*ObserverStats, error) {
	return s.repo.GetStats()
}

// GetIndicators gibt alle Indikatoren zurück
func (s *Service) GetIndicators(onlyActive bool) ([]Indicator, error) {
	return s.repo.GetAllIndicators(onlyActive)
}

// GetSources gibt alle Datenquellen zurück
func (s *Service) GetSources(onlyActive bool) ([]DataSource, error) {
	return s.repo.GetAllSources(onlyActive)
}

// GetIndicatorHistory gibt die Historie eines Indikators zurück
func (s *Service) GetIndicatorHistory(code string) (*IndicatorHistory, error) {
	return s.repo.GetIndicatorHistory(code)
}

// GetLatestValues gibt die neuesten Werte aller Indikatoren zurück
func (s *Service) GetLatestValues() (map[string]*ObservationValue, error) {
	indicators, err := s.repo.GetAllIndicators(true)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*ObservationValue)
	for _, ind := range indicators {
		val, err := s.repo.GetLatestValue(ind.ID)
		if err == nil && val != nil {
			result[ind.Code] = val
		}
	}

	return result, nil
}

// GetRuns gibt die letzten Sammelläufe zurück
func (s *Service) GetRuns(days int) ([]ObservationRun, error) {
	from := time.Now().AddDate(0, 0, -days)
	to := time.Now()
	return s.repo.GetRunsByDateRange(from, to)
}

// ExportDatabase exportiert die Datenbank als SQL
func (s *Service) ExportDatabase(outputPath string) error {
	return s.repo.ExportToSQL(outputPath)
}

// ImportDatabase importiert eine SQL-Datei
func (s *Service) ImportDatabase(inputPath string) error {
	return s.repo.ImportFromSQL(inputPath)
}

// GetDBPath gibt den Pfad zur Datenbank zurück
func (s *Service) GetDBPath() string {
	return s.repo.GetDBPath()
}

// IsRunning prüft ob ein Sammellauf läuft
func (s *Service) IsRunning() bool {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	return s.running
}

// GetCollectorStatus gibt den Status aller Collectors zurück
func (s *Service) GetCollectorStatus(ctx context.Context) map[string]bool {
	result := make(map[string]bool)
	for _, collector := range s.registry.GetAll() {
		result[collector.GetSourceCode()] = collector.IsAvailable(ctx)
	}
	return result
}
