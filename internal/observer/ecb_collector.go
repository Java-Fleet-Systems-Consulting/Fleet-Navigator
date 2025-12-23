package observer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ECBCollector sammelt Daten von der EZB Data Portal API
type ECBCollector struct {
	BaseCollector
	client *http.Client
}

// NewECBCollector erstellt einen neuen ECB Collector
func NewECBCollector() *ECBCollector {
	return &ECBCollector{
		BaseCollector: BaseCollector{
			SourceCode: "ECB",
			Name:       "Europäische Zentralbank",
			BaseURL:    "https://data-api.ecb.europa.eu",
			Timeout:    30 * time.Second,
		},
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IsAvailable prüft ob die ECB API erreichbar ist
func (c *ECBCollector) IsAvailable(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/service/data/FM/D.U2.EUR.4F.KR.MRR_FR.LEV?lastNObservations=1", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		log.Printf("ECB API nicht erreichbar: %v", err)
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// Collect sammelt aktuelle Daten
func (c *ECBCollector) Collect(ctx context.Context, indicators []Indicator) (*CollectorResult, error) {
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
			log.Printf("ECB: Fehler bei %s: %v", indicator.Code, err)
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
func (c *ECBCollector) CollectHistorical(ctx context.Context, indicator Indicator, from, to time.Time) (*CollectorResult, error) {
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
func (c *ECBCollector) fetchIndicator(ctx context.Context, indicator Indicator, from, to *time.Time) ([]ObservationValue, error) {
	// SDMX-Key extrahieren (z.B. "FM.D.U2.EUR.4F.KR.MRR_FR.LEV" -> dataflow "FM", key "D.U2.EUR.4F.KR.MRR_FR.LEV")
	parts := strings.SplitN(indicator.ExternalCode, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("ungültiger SDMX-Key: %s", indicator.ExternalCode)
	}

	dataflow := parts[0]
	key := parts[1]

	// URL bauen
	url := fmt.Sprintf("%s/service/data/%s/%s", c.BaseURL, dataflow, key)

	// Parameter hinzufügen
	params := []string{}
	if from != nil {
		params = append(params, fmt.Sprintf("startPeriod=%s", from.Format("2006-01-02")))
	}
	if to != nil {
		params = append(params, fmt.Sprintf("endPeriod=%s", to.Format("2006-01-02")))
	}
	if from == nil && to == nil {
		// Nur letzte Beobachtung wenn kein Zeitraum angegeben
		params = append(params, "lastNObservations=1")
	}

	if len(params) > 0 {
		url += "?" + strings.Join(params, "&")
	}

	log.Printf("ECB: Abrufen von %s", url)

	// Request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP-Fehler: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ECB API Fehler %d: %s", resp.StatusCode, string(body))
	}

	// Response parsen
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return c.parseECBResponse(body, indicator)
}

// ECB SDMX-JSON Response Struktur
type ecbResponse struct {
	DataSets []struct {
		Series map[string]struct {
			Observations map[string][]interface{} `json:"observations"`
		} `json:"series"`
	} `json:"dataSets"`
	Structure struct {
		Dimensions struct {
			Observation []struct {
				Values []struct {
					ID    string `json:"id"`
					Name  string `json:"name"`
					Start string `json:"start,omitempty"`
					End   string `json:"end,omitempty"`
				} `json:"values"`
			} `json:"observation"`
		} `json:"dimensions"`
	} `json:"structure"`
}

// parseECBResponse parst die ECB SDMX-JSON Response
func (c *ECBCollector) parseECBResponse(data []byte, indicator Indicator) ([]ObservationValue, error) {
	var resp ecbResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("JSON-Parse-Fehler: %w", err)
	}

	values := make([]ObservationValue, 0)
	now := time.Now()

	if len(resp.DataSets) == 0 {
		return values, nil
	}

	// Zeit-Dimensionen extrahieren
	timeDimensions := make(map[int]time.Time)
	if len(resp.Structure.Dimensions.Observation) > 0 {
		for i, val := range resp.Structure.Dimensions.Observation[0].Values {
			// Parse Zeitpunkt (Format kann "2024-01" oder "2024-01-15" sein)
			t, err := parseECBDate(val.ID)
			if err != nil {
				log.Printf("ECB: Zeit-Parse-Fehler für %s: %v", val.ID, err)
				continue
			}
			timeDimensions[i] = t
		}
	}

	// Werte extrahieren
	for _, series := range resp.DataSets[0].Series {
		for obsKey, obsValues := range series.Observations {
			if len(obsValues) == 0 {
				continue
			}

			// Index parsen
			idx, err := strconv.Atoi(obsKey)
			if err != nil {
				continue
			}

			// Zeit nachschlagen
			observedAt, ok := timeDimensions[idx]
			if !ok {
				continue
			}

			// Wert extrahieren (erstes Element ist der Wert)
			var value float64
			switch v := obsValues[0].(type) {
			case float64:
				value = v
			case int:
				value = float64(v)
			case string:
				value, _ = strconv.ParseFloat(v, 64)
			default:
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
				RawResponse: string(data),
			})
		}
	}

	log.Printf("ECB: %d Werte für %s abgerufen", len(values), indicator.Code)
	return values, nil
}

// parseECBDate parst ECB Datumsformate
func parseECBDate(dateStr string) (time.Time, error) {
	// Verschiedene Formate versuchen
	formats := []string{
		"2006-01-02",
		"2006-01",
		"2006-Q1", // Quartal
		"2006",
	}

	for _, format := range formats {
		if format == "2006-Q1" && strings.Contains(dateStr, "-Q") {
			// Quartal parsen (z.B. "2024-Q1" -> 2024-01-01)
			parts := strings.Split(dateStr, "-Q")
			if len(parts) == 2 {
				year, _ := strconv.Atoi(parts[0])
				quarter, _ := strconv.Atoi(parts[1])
				month := (quarter-1)*3 + 1
				return time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC), nil
			}
		}

		t, err := time.Parse(format, dateStr)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unbekanntes Datumsformat: %s", dateStr)
}

// GetSupportedIndicators gibt die unterstützten Indikator-Codes zurück
func (c *ECBCollector) GetSupportedIndicators() []string {
	return []string{
		"ECB_MAIN_RATE",
		"ECB_DEPOSIT_RATE",
		"ECB_MARGINAL_RATE",
		"HICP_EA",
		"HICP_DE",
		"UNEMPLOYMENT_EA",
	}
}
