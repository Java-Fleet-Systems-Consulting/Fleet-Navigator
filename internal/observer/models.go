// Package observer implementiert das Observer-System für Finanz- und Wirtschaftsdaten.
// Der Observer sammelt faktenbasiert, regelgebunden und deterministisch - ohne Interpretationen.
package observer

import (
	"time"
)

// Strategy definiert die Beobachtungsstrategie
type Strategy string

const (
	// StrategyConservative ist die Default-Strategie für Phase 1
	// Nur offizielle Quellen, niedrige Frequenz, nur Kernkennzahlen
	StrategyConservative Strategy = "CONSERVATIVE"

	// StrategyModerate erweitert die Quellen (für spätere Phasen)
	StrategyModerate Strategy = "MODERATE"

	// StrategyAggressive alle verfügbaren Quellen (für spätere Phasen)
	StrategyAggressive Strategy = "AGGRESSIVE"
)

// SourceClass definiert die Zulassungsklasse einer Quelle
type SourceClass string

const (
	// SourceClassOfficial - Offizielle Institutionen (EZB, Bundesbank, Eurostat)
	SourceClassOfficial SourceClass = "OFFICIAL"

	// SourceClassSemiOfficial - Halboffizielle Quellen (Forschungsinstitute)
	SourceClassSemiOfficial SourceClass = "SEMI_OFFICIAL"

	// SourceClassCommercial - Kommerzielle Datenanbieter
	SourceClassCommercial SourceClass = "COMMERCIAL"
)

// IndicatorCategory kategorisiert Indikatoren
type IndicatorCategory string

const (
	// CategoryInterestRate - Leitzinsen und Anleiherenditen
	CategoryInterestRate IndicatorCategory = "INTEREST_RATE"

	// CategoryInflation - Inflationsraten
	CategoryInflation IndicatorCategory = "INFLATION"

	// CategoryEmployment - Arbeitsmarktdaten
	CategoryEmployment IndicatorCategory = "EMPLOYMENT"

	// CategoryGDP - Bruttoinlandsprodukt
	CategoryGDP IndicatorCategory = "GDP"

	// CategoryExchange - Wechselkurse
	CategoryExchange IndicatorCategory = "EXCHANGE_RATE"

	// CategoryEvent - Klassifizierte Ereignisse
	CategoryEvent IndicatorCategory = "EVENT"

	// CategoryStocks - Aktienindizes
	CategoryStocks IndicatorCategory = "STOCKS"

	// CategoryCommodities - Rohstoffe (Gold, Silber, Öl)
	CategoryCommodities IndicatorCategory = "COMMODITIES"

	// CategoryRealEstate - Immobilien und REITs
	CategoryRealEstate IndicatorCategory = "REAL_ESTATE"

	// CategoryCrypto - Kryptowährungen
	CategoryCrypto IndicatorCategory = "CRYPTO"
)

// RunStatus beschreibt den Status eines Sammellaufs
type RunStatus string

const (
	RunStatusPending   RunStatus = "PENDING"
	RunStatusRunning   RunStatus = "RUNNING"
	RunStatusCompleted RunStatus = "COMPLETED"
	RunStatusFailed    RunStatus = "FAILED"
	RunStatusPartial   RunStatus = "PARTIAL" // Teilweise erfolgreich
)

// DataSource repräsentiert eine Datenquelle
type DataSource struct {
	ID          int64       `json:"id"`
	Code        string      `json:"code"`        // Eindeutiger Code (z.B. "ECB", "BUNDESBANK")
	Name        string      `json:"name"`        // Anzeigename
	Description string      `json:"description"` // Beschreibung
	URL         string      `json:"url"`         // Basis-URL der API
	SourceClass SourceClass `json:"sourceClass"` // Zulassungsklasse
	Active      bool        `json:"active"`      // Aktiv für Sammlung
	CreatedAt   time.Time   `json:"createdAt"`
}

// Indicator repräsentiert eine Kennzahl/Zeitreihe
type Indicator struct {
	ID           int64             `json:"id"`
	Code         string            `json:"code"`         // Eindeutiger Code (z.B. "ECB_MAIN_RATE")
	Name         string            `json:"name"`         // Anzeigename
	Description  string            `json:"description"`  // Beschreibung
	Category     IndicatorCategory `json:"category"`     // Kategorie
	Unit         string            `json:"unit"`         // Einheit (%, EUR, etc.)
	Frequency    string            `json:"frequency"`    // D=Daily, W=Weekly, M=Monthly, Q=Quarterly
	SourceID     int64             `json:"sourceId"`     // Primäre Datenquelle
	ExternalCode string            `json:"externalCode"` // Code bei der Quelle (z.B. SDMX-Key)
	Active       bool              `json:"active"`       // Aktiv für Sammlung
	CreatedAt    time.Time         `json:"createdAt"`
}

// ObservationRun repräsentiert einen Sammellauf
type ObservationRun struct {
	ID            int64     `json:"id"`
	Strategy      Strategy  `json:"strategy"`
	StartedAt     time.Time `json:"startedAt"`
	FinishedAt    *time.Time `json:"finishedAt,omitempty"`
	Status        RunStatus `json:"status"`
	TotalRecords  int       `json:"totalRecords"`  // Gesammelte Datensätze
	ErrorCount    int       `json:"errorCount"`    // Fehler
	ErrorMessages string    `json:"errorMessages"` // Fehlermeldungen (JSON Array)
	IsBackfill    bool      `json:"isBackfill"`    // Rückwärts-Sammlung?
	BackfillFrom  *time.Time `json:"backfillFrom,omitempty"` // Start-Datum Backfill
	BackfillTo    *time.Time `json:"backfillTo,omitempty"`   // End-Datum Backfill
}

// ObservationValue repräsentiert einen beobachteten Messwert
type ObservationValue struct {
	ID           int64     `json:"id"`
	RunID        int64     `json:"runId"`        // Referenz zum Sammellauf
	IndicatorID  int64     `json:"indicatorId"`  // Referenz zum Indikator
	SourceID     int64     `json:"sourceId"`     // Quelle des Wertes
	ObservedAt   time.Time `json:"observedAt"`   // Zeitpunkt der Beobachtung (Stichtag)
	CollectedAt  time.Time `json:"collectedAt"`  // Zeitpunkt der Sammlung
	Value        float64   `json:"value"`        // Der Messwert
	ValueString  string    `json:"valueString"`  // Optionaler String-Wert (für Events)
	Unit         string    `json:"unit"`         // Einheit
	PeriodStart  time.Time `json:"periodStart"`  // Beginn des Bezugszeitraums
	PeriodEnd    time.Time `json:"periodEnd"`    // Ende des Bezugszeitraums
	RawResponse  string    `json:"rawResponse"`  // Original-Antwort (für Revision)
}

// CollectorResult ist das Ergebnis eines Collector-Laufs
type CollectorResult struct {
	SourceCode    string             `json:"sourceCode"`
	Success       bool               `json:"success"`
	Values        []ObservationValue `json:"values"`
	ErrorMessage  string             `json:"errorMessage,omitempty"`
	CollectedAt   time.Time          `json:"collectedAt"`
}

// ObserverStats enthält Statistiken zum Observer
type ObserverStats struct {
	TotalRuns        int       `json:"totalRuns"`
	TotalValues      int       `json:"totalValues"`
	LastRunAt        *time.Time `json:"lastRunAt,omitempty"`
	LastRunStatus    RunStatus `json:"lastRunStatus,omitempty"`
	IndicatorCount   int       `json:"indicatorCount"`
	SourceCount      int       `json:"sourceCount"`
	OldestObservation *time.Time `json:"oldestObservation,omitempty"`
	NewestObservation *time.Time `json:"newestObservation,omitempty"`
}

// IndicatorValue ist ein einzelner Datenpunkt für API-Antworten
type IndicatorValue struct {
	Date  time.Time `json:"date"`
	Value float64   `json:"value"`
	Unit  string    `json:"unit"`
}

// IndicatorHistory ist die Historie eines Indikators
type IndicatorHistory struct {
	Indicator *Indicator       `json:"indicator"`
	Values    []IndicatorValue `json:"values"`
	Source    *DataSource      `json:"source"`
}
