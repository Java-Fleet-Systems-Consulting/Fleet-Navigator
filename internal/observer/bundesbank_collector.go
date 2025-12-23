package observer

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// BundesbankCollector sammelt Daten von der Bundesbank SDMX API
type BundesbankCollector struct {
	BaseCollector
	client *http.Client
}

// NewBundesbankCollector erstellt einen neuen Bundesbank Collector
func NewBundesbankCollector() *BundesbankCollector {
	return &BundesbankCollector{
		BaseCollector: BaseCollector{
			SourceCode: "BUNDESBANK",
			Name:       "Deutsche Bundesbank",
			BaseURL:    "https://api.statistiken.bundesbank.de",
			Timeout:    30 * time.Second,
		},
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IsAvailable prüft ob die Bundesbank API erreichbar ist
func (c *BundesbankCollector) IsAvailable(ctx context.Context) bool {
	// Test mit einem einfachen Metadaten-Request
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/rest/metadata/dataflow/BBK", nil)
	if err != nil {
		return false
	}

	resp, err := c.client.Do(req)
	if err != nil {
		log.Printf("Bundesbank API nicht erreichbar: %v", err)
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// Collect sammelt aktuelle Daten
func (c *BundesbankCollector) Collect(ctx context.Context, indicators []Indicator) (*CollectorResult, error) {
	result := &CollectorResult{
		SourceCode:  c.SourceCode,
		CollectedAt: time.Now(),
		Values:      make([]ObservationValue, 0),
		Success:     true,
	}

	for _, indicator := range indicators {
		if indicator.ExternalCode == "" {
			continue
		}

		values, err := c.fetchIndicator(ctx, indicator, nil, nil)
		if err != nil {
			log.Printf("Bundesbank: Fehler bei %s: %v", indicator.Code, err)
			result.ErrorMessage += fmt.Sprintf("%s: %v; ", indicator.Code, err)
			continue
		}

		result.Values = append(result.Values, values...)
	}

	if result.ErrorMessage != "" {
		result.Success = false
	}

	return result, nil
}

// CollectHistorical sammelt historische Daten (Backfill)
func (c *BundesbankCollector) CollectHistorical(ctx context.Context, indicator Indicator, from, to time.Time) (*CollectorResult, error) {
	result := &CollectorResult{
		SourceCode:  c.SourceCode,
		CollectedAt: time.Now(),
		Values:      make([]ObservationValue, 0),
		Success:     true,
	}

	values, err := c.fetchIndicator(ctx, indicator, &from, &to)
	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		return result, err
	}

	result.Values = values
	return result, nil
}

// fetchIndicator holt Daten für einen einzelnen Indikator
func (c *BundesbankCollector) fetchIndicator(ctx context.Context, indicator Indicator, from, to *time.Time) ([]ObservationValue, error) {
	// Bundesbank verwendet SDMX-Key direkt
	// Format: BBSIS.D.I.ZST.ZI.EUR.S1311.B.A604.R10XX.R.A.A._Z._Z.A
	// -> datastructure/key-family.key

	parts := strings.SplitN(indicator.ExternalCode, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("ungültiger SDMX-Key: %s", indicator.ExternalCode)
	}

	dataflow := parts[0]
	key := parts[1]

	// URL bauen - CSV Format für einfacheres Parsen
	url := fmt.Sprintf("%s/rest/data/%s/%s", c.BaseURL, dataflow, key)

	// Parameter hinzufügen
	params := []string{"format=csv"}
	if from != nil {
		params = append(params, fmt.Sprintf("startPeriod=%s", from.Format("2006-01-02")))
	}
	if to != nil {
		params = append(params, fmt.Sprintf("endPeriod=%s", to.Format("2006-01-02")))
	}
	if from == nil && to == nil {
		// Nur letzte Beobachtung
		params = append(params, "lastNObservations=1")
	}

	url += "?" + strings.Join(params, "&")

	log.Printf("Bundesbank: Abrufen von %s", url)

	// Request
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
		return nil, fmt.Errorf("Bundesbank API Fehler %d: %s", resp.StatusCode, string(body))
	}

	// CSV parsen
	return c.parseCSVResponse(resp.Body, indicator)
}

// parseCSVResponse parst die Bundesbank CSV Response
func (c *BundesbankCollector) parseCSVResponse(reader io.Reader, indicator Indicator) ([]ObservationValue, error) {
	csvReader := csv.NewReader(reader)
	csvReader.Comma = ','
	csvReader.LazyQuotes = true

	values := make([]ObservationValue, 0)
	now := time.Now()

	// Header lesen
	header, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("CSV-Header-Fehler: %w", err)
	}

	// Spalten-Indizes finden
	timeIdx := -1
	valueIdx := -1
	for i, col := range header {
		colLower := strings.ToLower(col)
		if strings.Contains(colLower, "time") || strings.Contains(colLower, "period") {
			timeIdx = i
		}
		if strings.Contains(colLower, "obs_value") || strings.Contains(colLower, "value") {
			valueIdx = i
		}
	}

	if timeIdx == -1 || valueIdx == -1 {
		return nil, fmt.Errorf("Zeit- oder Wert-Spalte nicht gefunden in: %v", header)
	}

	// Daten lesen
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		if len(record) <= timeIdx || len(record) <= valueIdx {
			continue
		}

		// Zeit parsen
		observedAt, err := parseBundesbankDate(record[timeIdx])
		if err != nil {
			continue
		}

		// Wert parsen
		valueStr := strings.TrimSpace(record[valueIdx])
		if valueStr == "" || valueStr == "." || valueStr == "-" {
			continue // Keine Daten
		}

		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			continue
		}

		values = append(values, ObservationValue{
			IndicatorID: indicator.ID,
			SourceID:    indicator.SourceID,
			ObservedAt:  observedAt,
			CollectedAt: now,
			Value:       value,
			Unit:        indicator.Unit,
			PeriodStart: observedAt,
			PeriodEnd:   observedAt,
		})
	}

	log.Printf("Bundesbank: %d Werte für %s abgerufen", len(values), indicator.Code)
	return values, nil
}

// parseBundesbankDate parst Bundesbank Datumsformate
func parseBundesbankDate(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)

	// Verschiedene Formate versuchen
	formats := []string{
		"2006-01-02",
		"2006-01",
		"2006",
	}

	for _, format := range formats {
		t, err := time.Parse(format, dateStr)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unbekanntes Datumsformat: %s", dateStr)
}

// GetSupportedIndicators gibt die unterstützten Indikator-Codes zurück
func (c *BundesbankCollector) GetSupportedIndicators() []string {
	return []string{
		"DE_10Y_YIELD",
	}
}
