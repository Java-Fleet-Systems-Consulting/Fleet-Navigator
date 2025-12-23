package observer

import (
	"fmt"
	"time"
)

// SimulationPeriod definiert vordefinierte Zeiträume
type SimulationPeriod string

const (
	Period1Month  SimulationPeriod = "1M"
	Period3Months SimulationPeriod = "3M"
	Period6Months SimulationPeriod = "6M"
	Period1Year   SimulationPeriod = "1Y"
	Period2Years  SimulationPeriod = "2Y"
	Period5Years  SimulationPeriod = "5Y"
)

// SimulationRequest ist die Anfrage für eine historische Simulation
type SimulationRequest struct {
	IndicatorCode string           `json:"indicatorCode"` // z.B. "BTC_EUR", "DAX", "GOLD_EUR"
	Amount        float64          `json:"amount"`        // Startbetrag in EUR
	Period        SimulationPeriod `json:"period"`        // oder StartDate/EndDate
	StartDate     *time.Time       `json:"startDate"`     // Optional: explizites Startdatum
	EndDate       *time.Time       `json:"endDate"`       // Optional: explizites Enddatum
}

// SimulationResult ist das Ergebnis einer historischen Simulation
type SimulationResult struct {
	// Eingabe
	IndicatorCode string    `json:"indicatorCode"`
	IndicatorName string    `json:"indicatorName"`
	StartAmount   float64   `json:"startAmount"`
	StartDate     time.Time `json:"startDate"`
	EndDate       time.Time `json:"endDate"`

	// Berechnete Werte
	StartValue float64 `json:"startValue"` // Index/Kurs am Start
	EndValue   float64 `json:"endValue"`   // Index/Kurs am Ende
	EndAmount  float64 `json:"endAmount"`  // Ergebnis in EUR
	ReturnPct  float64 `json:"returnPct"`  // Rendite in Prozent
	ReturnAbs  float64 `json:"returnAbs"`  // Absoluter Gewinn/Verlust

	// Metadaten
	Unit       string `json:"unit"`       // Einheit (EUR, Punkte, etc.)
	DataPoints int    `json:"dataPoints"` // Anzahl Datenpunkte im Zeitraum

	// Disclaimer (IMMER dabei!)
	Disclaimer string `json:"disclaimer"`
}

// DisclaimerText ist der Standard-Disclaimer für alle Simulationen
const DisclaimerText = `WICHTIGER HINWEIS: Diese Simulation zeigt ausschließlich die HISTORISCHE Entwicklung.
Vergangene Wertentwicklungen sind KEIN zuverlässiger Indikator für zukünftige Ergebnisse.
Diese Berechnung stellt KEINE Anlageberatung dar und sollte nicht als Grundlage für Investitionsentscheidungen verwendet werden.`

// DisclaimerShort ist die Kurzversion für Chat-Antworten
const DisclaimerShort = "⚠️ Vergangene Entwicklungen sind kein Indikator für zukünftige Ergebnisse."

// Simulate führt eine historische Simulation durch
func (s *Service) Simulate(req SimulationRequest) (*SimulationResult, error) {
	// Zeitraum bestimmen
	endDate := time.Now()
	var startDate time.Time

	if req.StartDate != nil && req.EndDate != nil {
		startDate = *req.StartDate
		endDate = *req.EndDate
	} else {
		startDate = periodToStartDate(req.Period, endDate)
	}

	// Indikator laden
	indicator, err := s.repo.GetIndicatorByCode(req.IndicatorCode)
	if err != nil {
		return nil, fmt.Errorf("Indikator nicht gefunden: %s", req.IndicatorCode)
	}
	if indicator == nil {
		return nil, fmt.Errorf("Indikator nicht gefunden: %s", req.IndicatorCode)
	}

	// Historische Werte holen
	values, err := s.repo.GetValuesByIndicatorDateRange(indicator.ID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("Fehler beim Laden der historischen Daten: %w", err)
	}

	if len(values) < 2 {
		return nil, fmt.Errorf("Nicht genügend historische Daten für %s im Zeitraum %s bis %s",
			req.IndicatorCode, startDate.Format("02.01.2006"), endDate.Format("02.01.2006"))
	}

	// Ältesten und neuesten Wert finden
	var oldestValue, newestValue *ObservationValue
	for i := range values {
		v := &values[i]
		if oldestValue == nil || v.ObservedAt.Before(oldestValue.ObservedAt) {
			oldestValue = v
		}
		if newestValue == nil || v.ObservedAt.After(newestValue.ObservedAt) {
			newestValue = v
		}
	}

	if oldestValue.Value == 0 {
		return nil, fmt.Errorf("Ungültiger Startwert (0) für %s", req.IndicatorCode)
	}

	// Berechnung: EndAmount = StartAmount * (EndValue / StartValue)
	endAmount := req.Amount * (newestValue.Value / oldestValue.Value)
	returnPct := ((newestValue.Value / oldestValue.Value) - 1) * 100
	returnAbs := endAmount - req.Amount

	result := &SimulationResult{
		IndicatorCode: req.IndicatorCode,
		IndicatorName: indicator.Name,
		StartAmount:   req.Amount,
		StartDate:     oldestValue.ObservedAt,
		EndDate:       newestValue.ObservedAt,
		StartValue:    oldestValue.Value,
		EndValue:      newestValue.Value,
		EndAmount:     endAmount,
		ReturnPct:     returnPct,
		ReturnAbs:     returnAbs,
		Unit:          indicator.Unit,
		DataPoints:    len(values),
		Disclaimer:    DisclaimerText,
	}

	return result, nil
}

// SimulateMultiple führt Simulationen für mehrere Indikatoren durch (zum Vergleichen)
func (s *Service) SimulateMultiple(indicators []string, amount float64, period SimulationPeriod) ([]*SimulationResult, error) {
	var results []*SimulationResult

	for _, code := range indicators {
		req := SimulationRequest{
			IndicatorCode: code,
			Amount:        amount,
			Period:        period,
		}

		result, err := s.Simulate(req)
		if err != nil {
			// Fehler loggen, aber weitermachen
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

// GetSimulatableIndicators gibt alle Indikatoren zurück, für die Simulation möglich ist
func (s *Service) GetSimulatableIndicators() ([]Indicator, error) {
	indicators, err := s.repo.GetAllIndicators(true)
	if err != nil {
		return nil, err
	}

	// Nur Indikatoren mit genügend historischen Daten
	var simulatable []Indicator
	for _, ind := range indicators {
		// Prüfen ob mindestens 30 Tage Daten vorhanden
		count, _ := s.repo.GetValueCount(ind.ID)
		if count >= 5 {
			simulatable = append(simulatable, ind)
		}
	}

	return simulatable, nil
}

// FormatSimulationForChat formatiert ein Simulationsergebnis für die Chat-Ausgabe
func FormatSimulationForChat(result *SimulationResult) string {
	// Emoji basierend auf Rendite
	emoji := "📊"
	if result.ReturnPct > 0 {
		emoji = "📈"
	} else if result.ReturnPct < 0 {
		emoji = "📉"
	}

	// Vorzeichen für Rendite
	sign := ""
	if result.ReturnPct > 0 {
		sign = "+"
	}

	return fmt.Sprintf(`%s **Historische Simulation: %s**

📅 Zeitraum: %s → %s

💰 Startbetrag: %.2f €
💵 Endbetrag: **%.2f €**
%s Rendite: **%s%.2f%%** (%s%.2f €)

📊 Referenz: %s
   Start: %.2f %s
   Ende: %.2f %s

%s`,
		emoji,
		result.IndicatorName,
		result.StartDate.Format("02.01.2006"),
		result.EndDate.Format("02.01.2006"),
		result.StartAmount,
		result.EndAmount,
		emoji,
		sign,
		result.ReturnPct,
		sign,
		result.ReturnAbs,
		result.IndicatorName,
		result.StartValue,
		result.Unit,
		result.EndValue,
		result.Unit,
		DisclaimerShort,
	)
}

// periodToStartDate berechnet das Startdatum basierend auf der Periode
func periodToStartDate(period SimulationPeriod, endDate time.Time) time.Time {
	switch period {
	case Period1Month:
		return endDate.AddDate(0, -1, 0)
	case Period3Months:
		return endDate.AddDate(0, -3, 0)
	case Period6Months:
		return endDate.AddDate(0, -6, 0)
	case Period1Year:
		return endDate.AddDate(-1, 0, 0)
	case Period2Years:
		return endDate.AddDate(-2, 0, 0)
	case Period5Years:
		return endDate.AddDate(-5, 0, 0)
	default:
		return endDate.AddDate(0, -3, 0) // Default: 3 Monate
	}
}

// GetAvailablePeriods gibt die verfügbaren Simulationsperioden zurück
func GetAvailablePeriods() []struct {
	Code SimulationPeriod `json:"code"`
	Name string           `json:"name"`
} {
	return []struct {
		Code SimulationPeriod `json:"code"`
		Name string           `json:"name"`
	}{
		{Period1Month, "1 Monat"},
		{Period3Months, "3 Monate"},
		{Period6Months, "6 Monate"},
		{Period1Year, "1 Jahr"},
		{Period2Years, "2 Jahre"},
		{Period5Years, "5 Jahre"},
	}
}
