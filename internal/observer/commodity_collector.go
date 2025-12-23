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

// CommodityCollector sammelt Rohstoff-Daten (Gold, Silber, Öl) von Yahoo Finance
type CommodityCollector struct {
	BaseCollector
	client *http.Client
}

// NewCommodityCollector erstellt einen neuen Commodity-Collector
func NewCommodityCollector() *CommodityCollector {
	return &CommodityCollector{
		BaseCollector: BaseCollector{
			SourceCode: "YAHOO",
			Name:       "Yahoo Finance Commodities",
			BaseURL:    "https://query1.finance.yahoo.com",
			Timeout:    30 * time.Second,
		},
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IsAvailable prüft ob die Yahoo Finance API erreichbar ist
func (c *CommodityCollector) IsAvailable(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/v7/finance/quote?symbols=GC=F", nil)
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
func (c *CommodityCollector) GetSupportedIndicators() []string {
	return []string{"GOLD_EUR", "SILVER_EUR", "OIL_BRENT", "EPRA_EURO"}
}

// commoditySymbols mappt unsere Indikatoren auf Yahoo-Symbole
var commoditySymbols = map[string]string{
	"GOLD_EUR":   "GC=F",   // Gold Futures USD
	"SILVER_EUR": "SI=F",   // Silver Futures USD
	"OIL_BRENT":  "BZ=F",   // Brent Crude Oil USD
	"EPRA_EURO":  "IPRP.DE", // iShares European Property Yield ETF
}

// Collect sammelt die aktuellen Rohstoff-Preise
func (c *CommodityCollector) Collect(ctx context.Context, indicators []Indicator) (*CollectorResult, error) {
	result := &CollectorResult{
		SourceCode:  c.SourceCode,
		CollectedAt: time.Now(),
		Values:      make([]ObservationValue, 0),
		Success:     true,
	}

	log.Printf("CommodityCollector: Starte Datensammlung für %d Indikatoren...", len(indicators))

	// Erst EUR/USD Kurs holen für Umrechnung
	eurUsdRate := c.getEURUSDRate(ctx)
	if eurUsdRate == 0 {
		eurUsdRate = 1.08 // Fallback
		log.Printf("CommodityCollector: Verwende Fallback EUR/USD = %.2f", eurUsdRate)
	}

	// Symbole für Abfrage sammeln
	var symbols string
	symbolToIndicator := make(map[string]Indicator)
	for _, ind := range indicators {
		if symbol, ok := commoditySymbols[ind.Code]; ok {
			if symbols != "" {
				symbols += ","
			}
			symbols += symbol
			symbolToIndicator[symbol] = ind
		}
	}

	if symbols == "" {
		log.Printf("CommodityCollector: Keine passenden Indikatoren gefunden")
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
			value := quote.RegularMarketPrice

			// Gold und Silber in EUR umrechnen
			if ind.Code == "GOLD_EUR" || ind.Code == "SILVER_EUR" {
				value = quote.RegularMarketPrice / eurUsdRate
			}

			observedAt := now
			if quote.RegularMarketTime > 0 {
				observedAt = time.Unix(quote.RegularMarketTime, 0)
			}

			result.Values = append(result.Values, ObservationValue{
				IndicatorID: ind.ID,
				SourceID:    ind.SourceID,
				ObservedAt:  observedAt,
				CollectedAt: now,
				Value:       value,
				Unit:        ind.Unit,
			})
			log.Printf("CommodityCollector: %s = %.2f", ind.Code, value)
		}
	}

	log.Printf("CommodityCollector: %d Werte gesammelt", len(result.Values))
	return result, nil
}

// getEURUSDRate holt den aktuellen EUR/USD Wechselkurs
func (c *CommodityCollector) getEURUSDRate(ctx context.Context) float64 {
	url := c.BaseURL + "/v7/finance/quote?symbols=EURUSD=X"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := c.client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0
	}

	body, _ := io.ReadAll(resp.Body)

	var data yahooResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return 0
	}

	if len(data.QuoteResponse.Result) > 0 {
		rate := data.QuoteResponse.Result[0].RegularMarketPrice
		log.Printf("CommodityCollector: EUR/USD = %.4f", rate)
		return rate
	}

	return 0
}

// CollectHistorical sammelt historische Rohstoff-Daten
func (c *CommodityCollector) CollectHistorical(ctx context.Context, indicator Indicator, from, to time.Time) (*CollectorResult, error) {
	result := &CollectorResult{
		SourceCode:  c.SourceCode,
		CollectedAt: time.Now(),
		Values:      make([]ObservationValue, 0),
		Success:     true,
	}

	log.Printf("CommodityCollector: Sammle historische Daten für %s von %s bis %s",
		indicator.Code, from.Format("2006-01-02"), to.Format("2006-01-02"))

	symbol, ok := commoditySymbols[indicator.Code]
	if !ok {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("Unbekannter Indikator: %s", indicator.Code)
		return result, fmt.Errorf(result.ErrorMessage)
	}

	// EUR/USD historisch für Umrechnung
	eurUsdHistory := c.getHistoricalEURUSD(ctx, from, to)

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

		value := closes[i]

		// Gold und Silber in EUR umrechnen
		if indicator.Code == "GOLD_EUR" || indicator.Code == "SILVER_EUR" {
			dateStr := time.Unix(ts, 0).Format("2006-01-02")
			if rate, ok := eurUsdHistory[dateStr]; ok && rate > 0 {
				value = closes[i] / rate
			} else {
				value = closes[i] / 1.08 // Fallback
			}
		}

		result.Values = append(result.Values, ObservationValue{
			IndicatorID: indicator.ID,
			SourceID:    indicator.SourceID,
			ObservedAt:  time.Unix(ts, 0),
			CollectedAt: now,
			Value:       value,
			Unit:        indicator.Unit,
		})
	}

	log.Printf("CommodityCollector: %d historische Werte für %s gesammelt", len(result.Values), indicator.Code)
	return result, nil
}

// getHistoricalEURUSD holt historische EUR/USD Kurse
func (c *CommodityCollector) getHistoricalEURUSD(ctx context.Context, from, to time.Time) map[string]float64 {
	result := make(map[string]float64)

	url := fmt.Sprintf("%s/v8/finance/chart/EURUSD=X?period1=%d&period2=%d&interval=1d",
		c.BaseURL, from.Unix(), to.Unix())

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return result
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := c.client.Do(req)
	if err != nil {
		return result
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var data yahooChartResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return result
	}

	if len(data.Chart.Result) == 0 {
		return result
	}

	chartResult := data.Chart.Result[0]
	if len(chartResult.Indicators.Quote) == 0 {
		return result
	}

	for i, ts := range chartResult.Timestamp {
		if i >= len(chartResult.Indicators.Quote[0].Close) {
			continue
		}
		dateStr := time.Unix(ts, 0).Format("2006-01-02")
		result[dateStr] = chartResult.Indicators.Quote[0].Close[i]
	}

	return result
}
