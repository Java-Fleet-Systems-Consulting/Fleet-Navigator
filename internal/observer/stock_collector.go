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

// StockCollector sammelt Aktienindex-Daten von Yahoo Finance
type StockCollector struct {
	BaseCollector
	client *http.Client
}

// NewStockCollector erstellt einen neuen Stock-Collector
func NewStockCollector() *StockCollector {
	return &StockCollector{
		BaseCollector: BaseCollector{
			SourceCode: "YAHOO",
			Name:       "Yahoo Finance",
			BaseURL:    "https://query1.finance.yahoo.com",
			Timeout:    30 * time.Second,
		},
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IsAvailable prüft ob die Yahoo Finance API erreichbar ist
func (c *StockCollector) IsAvailable(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/v7/finance/quote?symbols=^GDAXI", nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := c.client.Do(req)
	if err != nil {
		log.Printf("Yahoo Finance API nicht erreichbar: %v", err)
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GetSupportedIndicators gibt die unterstützten Indikator-Codes zurück
func (c *StockCollector) GetSupportedIndicators() []string {
	return []string{"DAX", "MSCI_WORLD", "SP500", "US_10Y_YIELD"}
}

// yahooQuote repräsentiert einen Yahoo Finance Quote
type yahooQuote struct {
	Symbol             string  `json:"symbol"`
	RegularMarketPrice float64 `json:"regularMarketPrice"`
	RegularMarketTime  int64   `json:"regularMarketTime"`
}

type yahooResponse struct {
	QuoteResponse struct {
		Result []yahooQuote `json:"result"`
		Error  interface{}  `json:"error"`
	} `json:"quoteResponse"`
}

// yahooChartResponse für historische Daten
type yahooChartResponse struct {
	Chart struct {
		Result []struct {
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Close []float64 `json:"close"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"chart"`
}

// indicatorToSymbol mappt unsere Indikatoren auf Yahoo-Symbole
var indicatorToSymbol = map[string]string{
	"DAX":          "^GDAXI",
	"SP500":        "^GSPC",
	"MSCI_WORLD":   "URTH",
	"US_10Y_YIELD": "^TNX",
}

// Collect sammelt die aktuellen Aktienindex-Stände
func (c *StockCollector) Collect(ctx context.Context, indicators []Indicator) (*CollectorResult, error) {
	result := &CollectorResult{
		SourceCode:  c.SourceCode,
		CollectedAt: time.Now(),
		Values:      make([]ObservationValue, 0),
		Success:     true,
	}

	log.Printf("StockCollector: Starte Datensammlung für %d Indikatoren...", len(indicators))

	// Symbole für Abfrage sammeln
	var symbols string
	symbolToIndicator := make(map[string]Indicator)
	for _, ind := range indicators {
		if symbol, ok := indicatorToSymbol[ind.Code]; ok {
			if symbols != "" {
				symbols += ","
			}
			symbols += symbol
			symbolToIndicator[symbol] = ind
		}
	}

	if symbols == "" {
		log.Printf("StockCollector: Keine passenden Indikatoren gefunden")
		return result, nil
	}

	url := fmt.Sprintf("%s/v7/finance/quote?symbols=%s", c.BaseURL, symbols)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		return result, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		return result, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("API-Fehler %d: %s", resp.StatusCode, string(body))
		return result, fmt.Errorf(result.ErrorMessage)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		return result, err
	}

	var data yahooResponse
	if err := json.Unmarshal(body, &data); err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		return result, err
	}

	now := time.Now()

	for _, quote := range data.QuoteResponse.Result {
		ind, ok := symbolToIndicator[quote.Symbol]
		if !ok {
			continue
		}

		if quote.RegularMarketPrice > 0 {
			observedAt := now
			if quote.RegularMarketTime > 0 {
				observedAt = time.Unix(quote.RegularMarketTime, 0)
			}

			result.Values = append(result.Values, ObservationValue{
				IndicatorID: ind.ID,
				SourceID:    ind.SourceID,
				ObservedAt:  observedAt,
				CollectedAt: now,
				Value:       quote.RegularMarketPrice,
				Unit:        ind.Unit,
			})
			log.Printf("StockCollector: %s = %.2f", ind.Code, quote.RegularMarketPrice)
		}
	}

	log.Printf("StockCollector: %d Werte gesammelt", len(result.Values))
	return result, nil
}

// CollectHistorical sammelt historische Aktiendaten
func (c *StockCollector) CollectHistorical(ctx context.Context, indicator Indicator, from, to time.Time) (*CollectorResult, error) {
	result := &CollectorResult{
		SourceCode:  c.SourceCode,
		CollectedAt: time.Now(),
		Values:      make([]ObservationValue, 0),
		Success:     true,
	}

	log.Printf("StockCollector: Sammle historische Daten für %s von %s bis %s",
		indicator.Code, from.Format("2006-01-02"), to.Format("2006-01-02"))

	symbol, ok := indicatorToSymbol[indicator.Code]
	if !ok {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("Unbekannter Indikator: %s", indicator.Code)
		return result, fmt.Errorf(result.ErrorMessage)
	}

	url := fmt.Sprintf("%s/v8/finance/chart/%s?period1=%d&period2=%d&interval=1d",
		c.BaseURL, symbol, from.Unix(), to.Unix())

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		return result, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := c.client.Do(req)
	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		return result, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("API-Fehler %d: %s", resp.StatusCode, string(body))
		return result, fmt.Errorf(result.ErrorMessage)
	}

	body, _ := io.ReadAll(resp.Body)

	var data yahooChartResponse
	if err := json.Unmarshal(body, &data); err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		return result, err
	}

	if len(data.Chart.Result) == 0 {
		return result, nil
	}

	chartResult := data.Chart.Result[0]
	if len(chartResult.Indicators.Quote) == 0 {
		return result, nil
	}

	timestamps := chartResult.Timestamp
	closes := chartResult.Indicators.Quote[0].Close
	now := time.Now()

	for i, ts := range timestamps {
		if i >= len(closes) || closes[i] == 0 {
			continue
		}

		result.Values = append(result.Values, ObservationValue{
			IndicatorID: indicator.ID,
			SourceID:    indicator.SourceID,
			ObservedAt:  time.Unix(ts, 0),
			CollectedAt: now,
			Value:       closes[i],
			Unit:        indicator.Unit,
		})
	}

	log.Printf("StockCollector: %d historische Werte für %s gesammelt", len(result.Values), indicator.Code)
	return result, nil
}
