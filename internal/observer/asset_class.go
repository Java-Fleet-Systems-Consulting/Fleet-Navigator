package observer

// AssetClass repräsentiert eine Anlageklasse
type AssetClass string

const (
	AssetClassBase        AssetClass = "BASE"        // Basis-Daten (Zinsen, Inflation)
	AssetClassStocks      AssetClass = "STOCKS"      // Aktienmärkte
	AssetClassBonds       AssetClass = "BONDS"       // Anleihen
	AssetClassCommodities AssetClass = "COMMODITIES" // Rohstoffe
	AssetClassRealEstate  AssetClass = "REAL_ESTATE" // Immobilien
	AssetClassCrypto      AssetClass = "CRYPTO"      // Kryptowährungen
)

// AssetClassInfo enthält Metadaten zu einer Anlageklasse
type AssetClassInfo struct {
	Code        AssetClass `json:"code"`
	Name        string     `json:"name"`
	NameEN      string     `json:"nameEn"`
	Description string     `json:"description"`
	Icon        string     `json:"icon"`
	Indicators  []string   `json:"indicators"` // Zugehörige Indikator-Codes
}

// AllAssetClasses enthält alle verfügbaren Anlageklassen
var AllAssetClasses = []AssetClassInfo{
	{
		Code:        AssetClassBase,
		Name:        "Basis-Daten",
		NameEN:      "Base Data",
		Description: "EZB-Zinsen, Inflation, Arbeitsmarkt",
		Icon:        "📊",
		Indicators:  []string{"ECB_MAIN_RATE", "ECB_DEPOSIT_RATE", "ESTR", "HICP_EA", "HICP_DE", "UNEMPLOYMENT_EA"},
	},
	{
		Code:        AssetClassStocks,
		Name:        "Aktienmärkte",
		NameEN:      "Stock Markets",
		Description: "DAX, MSCI World, S&P 500",
		Icon:        "📈",
		Indicators:  []string{"DAX", "MSCI_WORLD", "SP500"},
	},
	{
		Code:        AssetClassBonds,
		Name:        "Anleihen",
		NameEN:      "Bonds",
		Description: "Staatsanleihen Deutschland, USA",
		Icon:        "📃",
		Indicators:  []string{"DE_10Y_YIELD", "US_10Y_YIELD"},
	},
	{
		Code:        AssetClassCommodities,
		Name:        "Rohstoffe",
		NameEN:      "Commodities",
		Description: "Gold, Silber, Öl",
		Icon:        "🥇",
		Indicators:  []string{"GOLD_EUR", "SILVER_EUR", "OIL_BRENT"},
	},
	{
		Code:        AssetClassRealEstate,
		Name:        "Immobilien",
		NameEN:      "Real Estate",
		Description: "REIT-Indizes Europa",
		Icon:        "🏠",
		Indicators:  []string{"EPRA_EURO"},
	},
	{
		Code:        AssetClassCrypto,
		Name:        "Kryptowährungen",
		NameEN:      "Cryptocurrencies",
		Description: "Bitcoin, Ethereum",
		Icon:        "₿",
		Indicators:  []string{"BTC_EUR", "ETH_EUR"},
	},
}

// GetAssetClassInfo gibt die Info zu einer Anlageklasse zurück
func GetAssetClassInfo(code AssetClass) *AssetClassInfo {
	for _, ac := range AllAssetClasses {
		if ac.Code == code {
			return &ac
		}
	}
	return nil
}

// GetAssetClassForIndicator findet die Anlageklasse für einen Indikator
func GetAssetClassForIndicator(indicatorCode string) AssetClass {
	for _, ac := range AllAssetClasses {
		for _, ind := range ac.Indicators {
			if ind == indicatorCode {
				return ac.Code
			}
		}
	}
	return AssetClassBase
}

// GetAllIndicatorCodes gibt alle Indikator-Codes zurück
func GetAllIndicatorCodes() []string {
	var codes []string
	for _, ac := range AllAssetClasses {
		codes = append(codes, ac.Indicators...)
	}
	return codes
}
