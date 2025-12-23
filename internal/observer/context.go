package observer

import (
	"fmt"
	"strings"
	"time"
)

// ContextProvider generiert Kontext-Informationen für Experten
type ContextProvider struct {
	service *Service
}

// NewContextProvider erstellt einen neuen Context Provider
func NewContextProvider(service *Service) *ContextProvider {
	return &ContextProvider{service: service}
}

// GetFinanceContext generiert einen Kontext-String mit aktuellen Finanzdaten
// Für den Finanzberater-Experten (Franziska)
func (p *ContextProvider) GetFinanceContext() string {
	if p.service == nil {
		return ""
	}

	latestValues, err := p.service.GetLatestValues()
	if err != nil || len(latestValues) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n--- AKTUELLE MARKTDATEN (Observer) ---\n")
	sb.WriteString(fmt.Sprintf("Stand: %s\n\n", time.Now().Format("02.01.2006")))

	// === LEITZINSEN & ANLEIHEN ===
	sb.WriteString("📊 ZINSEN & ANLEIHEN:\n")
	if val, ok := latestValues["ECB_MAIN_RATE"]; ok {
		sb.WriteString(fmt.Sprintf("  EZB Hauptrefinanzierungssatz: %.2f%%\n", val.Value))
	}
	if val, ok := latestValues["ECB_DEPOSIT_RATE"]; ok {
		sb.WriteString(fmt.Sprintf("  EZB Einlagefazilität: %.2f%%\n", val.Value))
	}
	if val, ok := latestValues["ESTR"]; ok {
		sb.WriteString(fmt.Sprintf("  Euro Short-Term Rate (€STR): %.3f%%\n", val.Value))
	}
	if val, ok := latestValues["DE_10Y_YIELD"]; ok {
		sb.WriteString(fmt.Sprintf("  Bundesanleihe 10 Jahre: %.2f%%\n", val.Value))
	}
	if val, ok := latestValues["US_10Y_YIELD"]; ok {
		sb.WriteString(fmt.Sprintf("  US Treasury 10 Jahre: %.2f%%\n", val.Value))
	}

	// === INFLATION & ARBEITSMARKT ===
	sb.WriteString("\n📈 INFLATION & ARBEITSMARKT:\n")
	if val, ok := latestValues["HICP_EA"]; ok {
		sb.WriteString(fmt.Sprintf("  Inflation Eurozone (HVPI): %.1f%% p.a.\n", val.Value))
	}
	if val, ok := latestValues["HICP_DE"]; ok {
		sb.WriteString(fmt.Sprintf("  Inflation Deutschland: %.1f%% p.a.\n", val.Value))
	}
	if val, ok := latestValues["UNEMPLOYMENT_EA"]; ok {
		sb.WriteString(fmt.Sprintf("  Arbeitslosenquote Eurozone: %.1f%%\n", val.Value))
	}

	// === AKTIENMÄRKTE ===
	hasStocks := false
	if _, ok := latestValues["DAX"]; ok {
		hasStocks = true
	}
	if _, ok := latestValues["SP500"]; ok {
		hasStocks = true
	}
	if hasStocks {
		sb.WriteString("\n📈 AKTIENMÄRKTE:\n")
		if val, ok := latestValues["DAX"]; ok {
			sb.WriteString(fmt.Sprintf("  DAX: %.0f Punkte\n", val.Value))
		}
		if val, ok := latestValues["SP500"]; ok {
			sb.WriteString(fmt.Sprintf("  S&P 500: %.0f Punkte\n", val.Value))
		}
		if val, ok := latestValues["MSCI_WORLD"]; ok {
			sb.WriteString(fmt.Sprintf("  MSCI World (ETF): %.2f USD\n", val.Value))
		}
	}

	// === ROHSTOFFE ===
	hasCommodities := false
	if _, ok := latestValues["GOLD_EUR"]; ok {
		hasCommodities = true
	}
	if hasCommodities {
		sb.WriteString("\n🥇 ROHSTOFFE:\n")
		if val, ok := latestValues["GOLD_EUR"]; ok {
			sb.WriteString(fmt.Sprintf("  Gold: %.2f EUR/oz\n", val.Value))
		}
		if val, ok := latestValues["SILVER_EUR"]; ok {
			sb.WriteString(fmt.Sprintf("  Silber: %.2f EUR/oz\n", val.Value))
		}
		if val, ok := latestValues["OIL_BRENT"]; ok {
			sb.WriteString(fmt.Sprintf("  Brent Öl: %.2f USD/bbl\n", val.Value))
		}
	}

	// === KRYPTOWÄHRUNGEN ===
	hasCrypto := false
	if _, ok := latestValues["BTC_EUR"]; ok {
		hasCrypto = true
	}
	if hasCrypto {
		sb.WriteString("\n₿ KRYPTOWÄHRUNGEN:\n")
		if val, ok := latestValues["BTC_EUR"]; ok {
			sb.WriteString(fmt.Sprintf("  Bitcoin: %.0f EUR\n", val.Value))
		}
		if val, ok := latestValues["ETH_EUR"]; ok {
			sb.WriteString(fmt.Sprintf("  Ethereum: %.0f EUR\n", val.Value))
		}
	}

	// === IMMOBILIEN ===
	if val, ok := latestValues["EPRA_EURO"]; ok {
		sb.WriteString("\n🏠 IMMOBILIEN:\n")
		sb.WriteString(fmt.Sprintf("  EPRA Eurozone (REIT): %.2f EUR\n", val.Value))
	}

	sb.WriteString("\n--- ENDE MARKTDATEN ---\n")
	sb.WriteString("(Historische Simulationen auf Anfrage verfügbar)\n")

	return sb.String()
}

// GetEconomicSummary generiert eine kurze Zusammenfassung der Wirtschaftslage
func (p *ContextProvider) GetEconomicSummary() string {
	if p.service == nil {
		return ""
	}

	latestValues, err := p.service.GetLatestValues()
	if err != nil || len(latestValues) == 0 {
		return ""
	}

	var sb strings.Builder

	// Leitzins
	if val, ok := latestValues["ECB_MAIN_RATE"]; ok {
		sb.WriteString(fmt.Sprintf("Aktueller EZB-Leitzins: %.2f%%. ", val.Value))
	}

	// Inflation
	if val, ok := latestValues["HICP_DE"]; ok {
		if val.Value > 4.0 {
			sb.WriteString(fmt.Sprintf("Die Inflation in Deutschland liegt mit %.1f%% noch erhöht. ", val.Value))
		} else if val.Value > 2.0 {
			sb.WriteString(fmt.Sprintf("Die Inflation in Deutschland liegt bei %.1f%% (leicht über dem EZB-Ziel von 2%%). ", val.Value))
		} else {
			sb.WriteString(fmt.Sprintf("Die Inflation in Deutschland ist mit %.1f%% nahe am EZB-Ziel. ", val.Value))
		}
	}

	return sb.String()
}

// ShouldInjectContext prüft ob Kontext für einen Experten injiziert werden soll
func (p *ContextProvider) ShouldInjectContext(expertName string, message string) bool {
	// Franziska (Finanzberaterin) bekommt immer Kontext
	if strings.Contains(strings.ToLower(expertName), "franziska") ||
		strings.Contains(strings.ToLower(expertName), "finanz") {
		return true
	}

	// Bei Finanz-Keywords auch anderen Experten Kontext geben
	financeKeywords := []string{
		// Basis
		"zinsen", "zins", "inflation", "ezb", "leitzins",
		"anlage", "investieren", "rendite", "sparen", "geld",
		"kredit", "finanzierung", "vermögen",
		// Aktien
		"aktien", "aktie", "etf", "fonds", "fond", "dax", "börse",
		"msci", "s&p", "dividende", "depot",
		// Anleihen
		"anleihe", "anleihen", "bundesanleihe", "treasury", "bond",
		// Rohstoffe
		"gold", "silber", "öl", "rohstoff", "rohstoffe", "edelmetall",
		// Immobilien
		"immobilie", "immobilien", "reit", "grundstück", "wohnung",
		// Krypto
		"bitcoin", "btc", "ethereum", "eth", "krypto", "crypto",
		// Simulation
		"simulation", "simulieren", "wäre wenn", "hätte", "entwickelt",
	}

	msgLower := strings.ToLower(message)
	for _, keyword := range financeKeywords {
		if strings.Contains(msgLower, keyword) {
			return true
		}
	}

	return false
}

// GetContextForExpert generiert den passenden Kontext für einen Experten
func (p *ContextProvider) GetContextForExpert(expertName string, message string) string {
	if !p.ShouldInjectContext(expertName, message) {
		return ""
	}

	return p.GetFinanceContext()
}

// GetIndicatorTrend analysiert den Trend eines Indikators (letzte 30 Tage)
func (p *ContextProvider) GetIndicatorTrend(indicatorCode string) string {
	if p.service == nil {
		return ""
	}

	history, err := p.service.GetIndicatorHistory(indicatorCode)
	if err != nil || history == nil || len(history.Values) < 2 {
		return ""
	}

	// Letzten und ersten Wert vergleichen
	latest := history.Values[0]
	oldest := history.Values[len(history.Values)-1]

	diff := latest.Value - oldest.Value
	percentChange := (diff / oldest.Value) * 100

	if percentChange > 0.5 {
		return "steigend"
	} else if percentChange < -0.5 {
		return "fallend"
	}
	return "stabil"
}

// FormatValueForDisplay formatiert einen Wert für die Anzeige
func FormatValueForDisplay(value float64, unit string) string {
	switch unit {
	case "%":
		return fmt.Sprintf("%.2f%%", value)
	case "% YoY":
		return fmt.Sprintf("%.1f%% (Jahresrate)", value)
	default:
		return fmt.Sprintf("%.2f %s", value, unit)
	}
}
