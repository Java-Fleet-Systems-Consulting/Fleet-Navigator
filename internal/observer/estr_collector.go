package observer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// ESTRCollector sammelt €STR Daten von estr.dev
type ESTRCollector struct {
	BaseCollector
	client *http.Client
}

// NewESTRCollector erstellt einen neuen ESTR Collector
func NewESTRCollector() *ESTRCollector {
	return &ESTRCollector{
		BaseCollector: BaseCollector{
			SourceCode: "ESTR",
			Name:       "Euro Short-Term Rate API",
			BaseURL:    "https://estr.dev",
			Timeout:    15 * time.Second,
		},
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// IsAvailable prüft ob die ESTR API erreichbar ist
func (c *ESTRCollector) IsAvailable(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/api/current", nil)
	if err != nil {
		return false
	}

	resp, err := c.client.Do(req)
	if err != nil {
		log.Printf("ESTR API nicht erreichbar: %v", err)
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// ESTR API Response Strukturen
type estrCurrentResponse struct {
	Date  string  `json:"date"`
	Rate  float64 `json:"rate"`
	Unit  string  `json:"unit"`
}

type estrHistoryResponse struct {
	Data []estrDataPoint `json:"data"`
}

type estrDataPoint struct {
	Date string  `json:"date"`
	Rate float64 `json:"rate"`
}

// Collect sammelt aktuelle Daten
func (c *ESTRCollector) Collect(ctx context.Context, indicators []Indicator) (*CollectorResult, error) {
	result := &CollectorResult{
		SourceCode:  c.SourceCode,
		CollectedAt: time.Now(),
		Values:      make([]ObservationValue, 0),
		Success:     true,
	}

	for _, indicator := range indicators {
		if indicator.Code != "ESTR" {
			continue
		}

		values, err := c.fetchCurrent(ctx, indicator)
		if err != nil {
			log.Printf("ESTR: Fehler: %v", err)
			result.ErrorMessage = err.Error()
			result.Success = false
			continue
		}

		result.Values = append(result.Values, values...)
	}

	return result, nil
}

// CollectHistorical sammelt historische Daten (Backfill)
func (c *ESTRCollector) CollectHistorical(ctx context.Context, indicator Indicator, from, to time.Time) (*CollectorResult, error) {
	result := &CollectorResult{
		SourceCode:  c.SourceCode,
		CollectedAt: time.Now(),
		Values:      make([]ObservationValue, 0),
		Success:     true,
	}

	values, err := c.fetchHistory(ctx, indicator, from, to)
	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		return result, err
	}

	result.Values = values
	return result, nil
}

// fetchCurrent holt den aktuellen €STR Wert
func (c *ESTRCollector) fetchCurrent(ctx context.Context, indicator Indicator) ([]ObservationValue, error) {
	url := c.BaseURL + "/api/current"
	log.Printf("ESTR: Abrufen von %s", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP-Fehler: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ESTR API Fehler %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data estrCurrentResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("JSON-Parse-Fehler: %w", err)
	}

	// Datum parsen
	observedAt, err := time.Parse("2006-01-02", data.Date)
	if err != nil {
		return nil, fmt.Errorf("Datum-Parse-Fehler: %w", err)
	}

	now := time.Now()
	values := []ObservationValue{
		{
			IndicatorID: indicator.ID,
			SourceID:    indicator.SourceID,
			ObservedAt:  observedAt,
			CollectedAt: now,
			Value:       data.Rate,
			Unit:        "%",
			PeriodStart: observedAt,
			PeriodEnd:   observedAt,
			RawResponse: string(body),
		},
	}

	log.Printf("ESTR: Aktueller Wert: %.3f%% (%s)", data.Rate, data.Date)
	return values, nil
}

// fetchHistory holt historische €STR Werte
func (c *ESTRCollector) fetchHistory(ctx context.Context, indicator Indicator, from, to time.Time) ([]ObservationValue, error) {
	// Die estr.dev API hat verschiedene Endpunkte für historische Daten
	// /api/history für alle Daten
	// /api/range?from=YYYY-MM-DD&to=YYYY-MM-DD für einen Bereich

	url := fmt.Sprintf("%s/api/range?from=%s&to=%s",
		c.BaseURL,
		from.Format("2006-01-02"),
		to.Format("2006-01-02"))

	log.Printf("ESTR: Abrufen von %s", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP-Fehler: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Fallback auf /api/history wenn /api/range nicht existiert
		return c.fetchFullHistory(ctx, indicator, from, to)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data estrHistoryResponse
	if err := json.Unmarshal(body, &data); err != nil {
		// Versuche alternatives Format (Array direkt)
		var points []estrDataPoint
		if err := json.Unmarshal(body, &points); err != nil {
			return nil, fmt.Errorf("JSON-Parse-Fehler: %w", err)
		}
		data.Data = points
	}

	return c.parseHistoryData(data.Data, indicator, from, to)
}

// fetchFullHistory holt die gesamte Historie und filtert
func (c *ESTRCollector) fetchFullHistory(ctx context.Context, indicator Indicator, from, to time.Time) ([]ObservationValue, error) {
	url := c.BaseURL + "/api/history"
	log.Printf("ESTR: Fallback auf %s", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP-Fehler: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ESTR API Fehler %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data estrHistoryResponse
	if err := json.Unmarshal(body, &data); err != nil {
		// Versuche alternatives Format
		var points []estrDataPoint
		if err := json.Unmarshal(body, &points); err != nil {
			return nil, fmt.Errorf("JSON-Parse-Fehler: %w", err)
		}
		data.Data = points
	}

	return c.parseHistoryData(data.Data, indicator, from, to)
}

// parseHistoryData parst und filtert historische Daten
func (c *ESTRCollector) parseHistoryData(data []estrDataPoint, indicator Indicator, from, to time.Time) ([]ObservationValue, error) {
	values := make([]ObservationValue, 0)
	now := time.Now()

	for _, point := range data {
		observedAt, err := time.Parse("2006-01-02", point.Date)
		if err != nil {
			continue
		}

		// Filtern nach Zeitraum
		if observedAt.Before(from) || observedAt.After(to) {
			continue
		}

		values = append(values, ObservationValue{
			IndicatorID: indicator.ID,
			SourceID:    indicator.SourceID,
			ObservedAt:  observedAt,
			CollectedAt: now,
			Value:       point.Rate,
			Unit:        "%",
			PeriodStart: observedAt,
			PeriodEnd:   observedAt,
		})
	}

	log.Printf("ESTR: %d historische Werte abgerufen", len(values))
	return values, nil
}

// GetSupportedIndicators gibt die unterstützten Indikator-Codes zurück
func (c *ESTRCollector) GetSupportedIndicators() []string {
	return []string{
		"ESTR",
	}
}
