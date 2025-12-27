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

// CryptoCollector sammelt Kryptowährungs-Daten von CoinGecko
type CryptoCollector struct {
	BaseCollector
	client *http.Client
}

// NewCryptoCollector erstellt einen neuen Crypto-Collector
func NewCryptoCollector() *CryptoCollector {
	return &CryptoCollector{
		BaseCollector: BaseCollector{
			SourceCode: "COINGECKO",
			Name:       "CoinGecko",
			BaseURL:    "https://api.coingecko.com/api/v3",
			Timeout:    30 * time.Second,
		},
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IsAvailable prüft ob die CoinGecko API erreichbar ist
func (c *CryptoCollector) IsAvailable(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/ping", nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "FleetNavigator/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		log.Printf("CoinGecko API nicht erreichbar: %v", err)
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GetSupportedIndicators gibt die unterstützten Indikator-Codes zurück
func (c *CryptoCollector) GetSupportedIndicators() []string {
	return []string{"BTC_EUR", "ETH_EUR"}
}

// coinGeckoPrice repräsentiert die API-Antwort
type coinGeckoPrice struct {
	Bitcoin struct {
		EUR float64 `json:"eur"`
	} `json:"bitcoin"`
	Ethereum struct {
		EUR float64 `json:"eur"`
	} `json:"ethereum"`
}

// Collect sammelt die aktuellen Krypto-Preise
func (c *CryptoCollector) Collect(ctx context.Context, indicators []Indicator) (*CollectorResult, error) {
	result := &CollectorResult{
		SourceCode:  c.SourceCode,
		CollectedAt: time.Now(),
		Values:      make([]ObservationValue, 0),
		Success:     true,
	}

	log.Printf("CryptoCollector: Starte Datensammlung für %d Indikatoren...", len(indicators))

	// CoinGecko API: Einfache Preis-Abfrage
	url := fmt.Sprintf("%s/simple/price?ids=bitcoin,ethereum&vs_currencies=eur", c.BaseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		return result, err
	}

	req.Header.Set("User-Agent", "FleetNavigator/1.0")
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
		return result, fmt.Errorf("%s", result.ErrorMessage)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		return result, err
	}

	var prices coinGeckoPrice
	if err := json.Unmarshal(body, &prices); err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		return result, err
	}

	now := time.Now()

	// Mapping: Indikator-Code -> CoinGecko-Wert
	for _, ind := range indicators {
		var value float64
		switch ind.Code {
		case "BTC_EUR":
			value = prices.Bitcoin.EUR
		case "ETH_EUR":
			value = prices.Ethereum.EUR
		default:
			continue
		}

		if value > 0 {
			result.Values = append(result.Values, ObservationValue{
				IndicatorID: ind.ID,
				SourceID:    ind.SourceID,
				ObservedAt:  now,
				CollectedAt: now,
				Value:       value,
				Unit:        ind.Unit,
			})
			log.Printf("CryptoCollector: %s = %.2f", ind.Code, value)
		}
	}

	log.Printf("CryptoCollector: %d Werte gesammelt", len(result.Values))
	return result, nil
}

// CollectHistorical sammelt historische Krypto-Preise für Backfill
func (c *CryptoCollector) CollectHistorical(ctx context.Context, indicator Indicator, from, to time.Time) (*CollectorResult, error) {
	result := &CollectorResult{
		SourceCode:  c.SourceCode,
		CollectedAt: time.Now(),
		Values:      make([]ObservationValue, 0),
		Success:     true,
	}

	log.Printf("CryptoCollector: Sammle historische Daten für %s von %s bis %s",
		indicator.Code, from.Format("2006-01-02"), to.Format("2006-01-02"))

	// CoinGecko Coin-ID ermitteln
	var coinID string
	switch indicator.Code {
	case "BTC_EUR":
		coinID = "bitcoin"
	case "ETH_EUR":
		coinID = "ethereum"
	default:
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("Unbekannter Indikator: %s", indicator.Code)
		return result, fmt.Errorf("%s", result.ErrorMessage)
	}

	// CoinGecko: /coins/{id}/market_chart/range
	url := fmt.Sprintf("%s/coins/%s/market_chart/range?vs_currency=eur&from=%d&to=%d",
		c.BaseURL, coinID, from.Unix(), to.Unix())

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		return result, err
	}

	req.Header.Set("User-Agent", "FleetNavigator/1.0")
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
		return result, fmt.Errorf("%s", result.ErrorMessage)
	}

	body, _ := io.ReadAll(resp.Body)

	// Response: {"prices": [[timestamp, price], ...]}
	var data struct {
		Prices [][]float64 `json:"prices"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		return result, err
	}

	// Täglich einen Wert nehmen (CoinGecko gibt viele Datenpunkte)
	lastDate := ""
	now := time.Now()
	for _, p := range data.Prices {
		if len(p) < 2 {
			continue
		}
		ts := time.Unix(int64(p[0]/1000), 0)
		dateStr := ts.Format("2006-01-02")

		// Nur einen Wert pro Tag
		if dateStr == lastDate {
			continue
		}
		lastDate = dateStr

		result.Values = append(result.Values, ObservationValue{
			IndicatorID: indicator.ID,
			SourceID:    indicator.SourceID,
			ObservedAt:  ts,
			CollectedAt: now,
			Value:       p[1],
			Unit:        indicator.Unit,
		})
	}

	log.Printf("CryptoCollector: %d historische Werte für %s gesammelt", len(result.Values), indicator.Code)
	return result, nil
}
