package observer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Handlers enthält die HTTP-Handler für Observer-Endpoints
type Handlers struct {
	service *Service
}

// NewHandlers erstellt neue Observer-Handler
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// RegisterRoutes registriert die Observer-API-Routen
func (h *Handlers) RegisterRoutes(mux *http.ServeMux) {
	// Status & Konfiguration
	mux.HandleFunc("/api/observer/status", h.handleStatus)
	mux.HandleFunc("/api/observer/config", h.handleConfig)

	// Daten
	mux.HandleFunc("/api/observer/indicators", h.handleIndicators)
	mux.HandleFunc("/api/observer/sources", h.handleSources)
	mux.HandleFunc("/api/observer/values/latest", h.handleLatestValues)
	mux.HandleFunc("/api/observer/values/", h.handleIndicatorValues)
	mux.HandleFunc("/api/observer/runs", h.handleRuns)

	// Aktionen
	mux.HandleFunc("/api/observer/run", h.handleRunNow)
	mux.HandleFunc("/api/observer/backfill", h.handleBackfill)

	// Simulation
	mux.HandleFunc("/api/observer/simulate", h.handleSimulate)
	mux.HandleFunc("/api/observer/simulate/indicators", h.handleSimulatableIndicators)
	mux.HandleFunc("/api/observer/simulate/periods", h.handleSimulationPeriods)

	// Asset-Klassen
	mux.HandleFunc("/api/observer/asset-classes", h.handleAssetClasses)

	// Export/Import
	mux.HandleFunc("/api/observer/export", h.handleExport)
	mux.HandleFunc("/api/observer/import", h.handleImport)

	log.Printf("Observer: API-Routen registriert")
}

// --- Status & Konfiguration ---

// handleStatus gibt den Observer-Status zurück
func (h *Handlers) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	stats, err := h.service.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	config := h.service.GetConfig()
	collectorStatus := h.service.GetCollectorStatus(ctx)

	response := map[string]interface{}{
		"enabled":         config.Enabled,
		"strategy":        config.Strategy,
		"running":         h.service.IsRunning(),
		"stats":           stats,
		"collectorStatus": collectorStatus,
		"prompt":          config.Prompt,
	}

	if h.service.scheduler != nil {
		response["nextRun"] = h.service.scheduler.GetNextRun()
		response["schedulerRunning"] = h.service.scheduler.IsRunning()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleConfig gibt/setzt die Observer-Konfiguration
func (h *Handlers) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		config := h.service.GetConfig()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config)

	case http.MethodPost, http.MethodPut:
		var config ObserverConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		if err := h.service.SetConfig(&config); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Scheduler starten/stoppen basierend auf config.Enabled
		if config.Enabled {
			h.service.Start()
		} else {
			h.service.Stop()
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// --- Daten ---

// handleIndicators gibt alle Indikatoren zurück
func (h *Handlers) handleIndicators(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	onlyActive := r.URL.Query().Get("active") != "false"

	indicators, err := h.service.GetIndicators(onlyActive)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(indicators)
}

// handleSources gibt alle Datenquellen zurück
func (h *Handlers) handleSources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	onlyActive := r.URL.Query().Get("active") != "false"

	sources, err := h.service.GetSources(onlyActive)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sources)
}

// handleLatestValues gibt die neuesten Werte aller Indikatoren zurück
func (h *Handlers) handleLatestValues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	values, err := h.service.GetLatestValues()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Zu Response-Format konvertieren
	response := make(map[string]interface{})
	for code, val := range values {
		response[code] = map[string]interface{}{
			"value":      val.Value,
			"unit":       val.Unit,
			"observedAt": val.ObservedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleIndicatorValues gibt die Historie eines Indikators zurück
func (h *Handlers) handleIndicatorValues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Indikator-Code aus URL extrahieren
	// /api/observer/values/ECB_MAIN_RATE
	path := r.URL.Path
	code := path[len("/api/observer/values/"):]

	if code == "" {
		http.Error(w, "Indicator code required", http.StatusBadRequest)
		return
	}

	history, err := h.service.GetIndicatorHistory(code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// handleRuns gibt die letzten Sammelläufe zurück
func (h *Handlers) handleRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	days := 30 // Default
	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	runs, err := h.service.GetRuns(days)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(runs)
}

// --- Aktionen ---

// handleRunNow führt einen Sammellauf sofort aus
func (h *Handlers) handleRunNow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	run, err := h.service.RunNow(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(run)
}

// handleBackfill führt einen Backfill aus
func (h *Handlers) handleBackfill(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parameter lesen
	var req struct {
		Days int    `json:"days"`
		From string `json:"from"` // Format: 2006-01-02
		To   string `json:"to"`   // Format: 2006-01-02
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Default: 30 Tage
		req.Days = 30
	}

	var from, to time.Time
	now := time.Now()

	if req.From != "" && req.To != "" {
		var err error
		from, err = time.Parse("2006-01-02", req.From)
		if err != nil {
			http.Error(w, "Invalid from date", http.StatusBadRequest)
			return
		}
		to, err = time.Parse("2006-01-02", req.To)
		if err != nil {
			http.Error(w, "Invalid to date", http.StatusBadRequest)
			return
		}
	} else {
		if req.Days <= 0 {
			req.Days = 30
		}
		from = now.AddDate(0, 0, -req.Days)
		to = now
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()

	run, err := h.service.Backfill(ctx, from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(run)
}

// --- Export/Import ---

// handleExport exportiert die Observer-Datenbank
func (h *Handlers) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "sql"
	}

	switch format {
	case "sql":
		// SQL-Dump erstellen
		tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("observer_export_%d.sql", time.Now().Unix()))
		defer os.Remove(tmpFile)

		if err := h.service.ExportDatabase(tmpFile); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		content, err := os.ReadFile(tmpFile)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/sql")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=observer_%s.sql",
			time.Now().Format("2006-01-02")))
		w.Write(content)

	case "sqlite":
		// SQLite-Datei direkt senden
		dbPath := h.service.GetDBPath()

		w.Header().Set("Content-Type", "application/x-sqlite3")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=observer_%s.db",
			time.Now().Format("2006-01-02")))

		http.ServeFile(w, r, dbPath)

	case "json":
		// JSON-Export der aktuellen Werte
		values, err := h.service.GetLatestValues()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		stats, _ := h.service.GetStats()

		export := map[string]interface{}{
			"exportedAt": time.Now(),
			"stats":      stats,
			"values":     values,
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=observer_%s.json",
			time.Now().Format("2006-01-02")))
		json.NewEncoder(w).Encode(export)

	default:
		http.Error(w, "Unknown format. Use: sql, sqlite, json", http.StatusBadRequest)
	}
}

// handleImport importiert Observer-Daten
func (h *Handlers) handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Multipart Form lesen (max 100MB)
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		http.Error(w, "File too large or invalid form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Temporäre Datei erstellen
	tmpFile := filepath.Join(os.TempDir(), header.Filename)
	out, err := os.Create(tmpFile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpFile)

	if _, err := io.Copy(out, file); err != nil {
		out.Close()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out.Close()

	// Import basierend auf Dateityp
	ext := filepath.Ext(header.Filename)
	switch ext {
	case ".sql":
		if err := h.service.ImportDatabase(tmpFile); err != nil {
			http.Error(w, "Import failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

	default:
		http.Error(w, "Unsupported file type. Use .sql", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Import erfolgreich",
	})
}

// --- Simulation ---

// handleSimulate führt eine historische Simulation durch
func (h *Handlers) handleSimulate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validierung
	if req.IndicatorCode == "" {
		http.Error(w, "indicatorCode is required", http.StatusBadRequest)
		return
	}
	if req.Amount <= 0 {
		http.Error(w, "amount must be positive", http.StatusBadRequest)
		return
	}
	if req.Period == "" && req.StartDate == nil {
		req.Period = Period3Months // Default
	}

	result, err := h.service.Simulate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleSimulatableIndicators gibt alle simulierbaren Indikatoren zurück
func (h *Handlers) handleSimulatableIndicators(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	indicators, err := h.service.GetSimulatableIndicators()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Nach Asset-Klasse gruppieren
	grouped := make(map[string][]Indicator)
	for _, ind := range indicators {
		assetClass := GetAssetClassForIndicator(ind.Code)
		grouped[string(assetClass)] = append(grouped[string(assetClass)], ind)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"indicators": indicators,
		"grouped":    grouped,
	})
}

// handleSimulationPeriods gibt die verfügbaren Simulationsperioden zurück
func (h *Handlers) handleSimulationPeriods(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GetAvailablePeriods())
}

// handleAssetClasses gibt alle Asset-Klassen zurück
func (h *Handlers) handleAssetClasses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AllAssetClasses)
}
