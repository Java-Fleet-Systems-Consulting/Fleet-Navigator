package observer

import (
	"context"
	"time"
)

// Collector ist das Interface für Datensammler
type Collector interface {
	// GetSourceCode gibt den Quell-Code zurück (z.B. "ECB", "BUNDESBANK")
	GetSourceCode() string

	// GetName gibt den Anzeigenamen zurück
	GetName() string

	// IsAvailable prüft ob der Collector verfügbar ist
	IsAvailable(ctx context.Context) bool

	// Collect sammelt aktuelle Daten für alle Indikatoren
	Collect(ctx context.Context, indicators []Indicator) (*CollectorResult, error)

	// CollectHistorical sammelt historische Daten (Backfill)
	CollectHistorical(ctx context.Context, indicator Indicator, from, to time.Time) (*CollectorResult, error)

	// GetSupportedIndicators gibt die unterstützten Indikator-Codes zurück
	GetSupportedIndicators() []string
}

// BaseCollector ist die Basis-Implementierung für alle Collectors
type BaseCollector struct {
	SourceCode string
	Name       string
	BaseURL    string
	Timeout    time.Duration
}

// GetSourceCode implementiert Collector
func (c *BaseCollector) GetSourceCode() string {
	return c.SourceCode
}

// GetName implementiert Collector
func (c *BaseCollector) GetName() string {
	return c.Name
}

// CollectorRegistry verwaltet alle verfügbaren Collectors
type CollectorRegistry struct {
	collectors map[string]Collector
}

// NewCollectorRegistry erstellt eine neue Registry
func NewCollectorRegistry() *CollectorRegistry {
	return &CollectorRegistry{
		collectors: make(map[string]Collector),
	}
}

// Register registriert einen Collector
func (r *CollectorRegistry) Register(collector Collector) {
	r.collectors[collector.GetSourceCode()] = collector
}

// Get holt einen Collector nach Code
func (r *CollectorRegistry) Get(sourceCode string) Collector {
	return r.collectors[sourceCode]
}

// GetAll gibt alle registrierten Collectors zurück
func (r *CollectorRegistry) GetAll() []Collector {
	result := make([]Collector, 0, len(r.collectors))
	for _, c := range r.collectors {
		result = append(result, c)
	}
	return result
}

// GetAvailable gibt alle verfügbaren Collectors zurück
func (r *CollectorRegistry) GetAvailable(ctx context.Context) []Collector {
	result := make([]Collector, 0)
	for _, c := range r.collectors {
		if c.IsAvailable(ctx) {
			result = append(result, c)
		}
	}
	return result
}
