package observer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// Repository ist das Daten-Repository für Observer-Daten
type Repository struct {
	db     *sql.DB
	dbPath string
}

// NewRepository erstellt ein neues Observer-Repository mit eigener Datenbank
func NewRepository(dataDir string) (*Repository, error) {
	dbPath := filepath.Join(dataDir, "observer.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("Observer-Datenbank öffnen fehlgeschlagen: %w", err)
	}

	// SQLite Konfiguration
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)

	// WAL-Modus für bessere Concurrent-Performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		log.Printf("Observer: WAL-Modus nicht aktiviert: %v", err)
	}

	repo := &Repository{db: db, dbPath: dbPath}

	if err := repo.createSchema(); err != nil {
		return nil, err
	}

	return repo, nil
}

// createSchema erstellt die Datenbank-Tabellen (append-only Design)
func (r *Repository) createSchema() error {
	schema := `
	-- Datenquellen (Seed-Daten)
	CREATE TABLE IF NOT EXISTS data_source (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		description TEXT DEFAULT '',
		url TEXT DEFAULT '',
		source_class TEXT NOT NULL DEFAULT 'OFFICIAL',
		active INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Indikatoren/Kennzahlen (Seed-Daten)
	CREATE TABLE IF NOT EXISTS indicator (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		description TEXT DEFAULT '',
		category TEXT NOT NULL,
		unit TEXT DEFAULT '',
		frequency TEXT DEFAULT 'D',
		source_id INTEGER,
		external_code TEXT DEFAULT '',
		active INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (source_id) REFERENCES data_source(id)
	);

	-- Sammelläufe (Beobachtungsdaten)
	CREATE TABLE IF NOT EXISTS observation_run (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		strategy TEXT NOT NULL DEFAULT 'CONSERVATIVE',
		started_at DATETIME NOT NULL,
		finished_at DATETIME,
		status TEXT NOT NULL DEFAULT 'PENDING',
		total_records INTEGER DEFAULT 0,
		error_count INTEGER DEFAULT 0,
		error_messages TEXT DEFAULT '[]',
		is_backfill INTEGER DEFAULT 0,
		backfill_from DATETIME,
		backfill_to DATETIME
	);

	-- Messwerte (Beobachtungsdaten, append-only)
	CREATE TABLE IF NOT EXISTS observation_value (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		run_id INTEGER NOT NULL,
		indicator_id INTEGER NOT NULL,
		source_id INTEGER NOT NULL,
		observed_at DATETIME NOT NULL,
		collected_at DATETIME NOT NULL,
		value REAL,
		value_string TEXT DEFAULT '',
		unit TEXT DEFAULT '',
		period_start DATETIME,
		period_end DATETIME,
		raw_response TEXT DEFAULT '',
		FOREIGN KEY (run_id) REFERENCES observation_run(id),
		FOREIGN KEY (indicator_id) REFERENCES indicator(id),
		FOREIGN KEY (source_id) REFERENCES data_source(id)
	);

	-- Indizes für schnelle Abfragen
	CREATE INDEX IF NOT EXISTS idx_observation_value_indicator ON observation_value(indicator_id);
	CREATE INDEX IF NOT EXISTS idx_observation_value_observed_at ON observation_value(observed_at);
	CREATE INDEX IF NOT EXISTS idx_observation_value_indicator_date ON observation_value(indicator_id, observed_at);
	CREATE INDEX IF NOT EXISTS idx_observation_run_status ON observation_run(status);
	CREATE INDEX IF NOT EXISTS idx_observation_run_started_at ON observation_run(started_at);
	CREATE INDEX IF NOT EXISTS idx_indicator_category ON indicator(category);
	CREATE INDEX IF NOT EXISTS idx_indicator_active ON indicator(active);
	`

	_, err := r.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("Observer-Schema erstellen fehlgeschlagen: %w", err)
	}

	return nil
}

// Close schließt die Datenbankverbindung
func (r *Repository) Close() error {
	return r.db.Close()
}

// GetDBPath gibt den Pfad zur Datenbank zurück (für Export)
func (r *Repository) GetDBPath() string {
	return r.dbPath
}

// --- DataSource CRUD ---

// CreateSource erstellt eine neue Datenquelle
func (r *Repository) CreateSource(source *DataSource) error {
	result, err := r.db.Exec(`
		INSERT INTO data_source (code, name, description, url, source_class, active)
		VALUES (?, ?, ?, ?, ?, ?)
	`, source.Code, source.Name, source.Description, source.URL, source.SourceClass, source.Active)

	if err != nil {
		return fmt.Errorf("Datenquelle erstellen fehlgeschlagen: %w", err)
	}

	id, _ := result.LastInsertId()
	source.ID = id
	source.CreatedAt = time.Now()
	return nil
}

// GetSourceByCode holt eine Quelle nach Code
func (r *Repository) GetSourceByCode(code string) (*DataSource, error) {
	source := &DataSource{}
	err := r.db.QueryRow(`
		SELECT id, code, name, description, url, source_class, active, created_at
		FROM data_source WHERE code = ?
	`, code).Scan(&source.ID, &source.Code, &source.Name, &source.Description,
		&source.URL, &source.SourceClass, &source.Active, &source.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return source, nil
}

// GetAllSources holt alle Datenquellen
func (r *Repository) GetAllSources(onlyActive bool) ([]DataSource, error) {
	query := `SELECT id, code, name, description, url, source_class, active, created_at FROM data_source`
	if onlyActive {
		query += " WHERE active = 1"
	}
	query += " ORDER BY name ASC"

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sources := make([]DataSource, 0)
	for rows.Next() {
		var s DataSource
		if err := rows.Scan(&s.ID, &s.Code, &s.Name, &s.Description, &s.URL, &s.SourceClass, &s.Active, &s.CreatedAt); err != nil {
			return nil, err
		}
		sources = append(sources, s)
	}
	return sources, nil
}

// --- Indicator CRUD ---

// CreateIndicator erstellt einen neuen Indikator
func (r *Repository) CreateIndicator(ind *Indicator) error {
	result, err := r.db.Exec(`
		INSERT INTO indicator (code, name, description, category, unit, frequency, source_id, external_code, active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, ind.Code, ind.Name, ind.Description, ind.Category, ind.Unit, ind.Frequency, ind.SourceID, ind.ExternalCode, ind.Active)

	if err != nil {
		return fmt.Errorf("Indikator erstellen fehlgeschlagen: %w", err)
	}

	id, _ := result.LastInsertId()
	ind.ID = id
	ind.CreatedAt = time.Now()
	return nil
}

// GetIndicatorByCode holt einen Indikator nach Code
func (r *Repository) GetIndicatorByCode(code string) (*Indicator, error) {
	ind := &Indicator{}
	err := r.db.QueryRow(`
		SELECT id, code, name, description, category, unit, frequency, source_id, external_code, active, created_at
		FROM indicator WHERE code = ?
	`, code).Scan(&ind.ID, &ind.Code, &ind.Name, &ind.Description, &ind.Category,
		&ind.Unit, &ind.Frequency, &ind.SourceID, &ind.ExternalCode, &ind.Active, &ind.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return ind, nil
}

// GetAllIndicators holt alle Indikatoren
func (r *Repository) GetAllIndicators(onlyActive bool) ([]Indicator, error) {
	query := `SELECT id, code, name, description, category, unit, frequency, source_id, external_code, active, created_at FROM indicator`
	if onlyActive {
		query += " WHERE active = 1"
	}
	query += " ORDER BY category ASC, name ASC"

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indicators := make([]Indicator, 0)
	for rows.Next() {
		var i Indicator
		if err := rows.Scan(&i.ID, &i.Code, &i.Name, &i.Description, &i.Category,
			&i.Unit, &i.Frequency, &i.SourceID, &i.ExternalCode, &i.Active, &i.CreatedAt); err != nil {
			return nil, err
		}
		indicators = append(indicators, i)
	}
	return indicators, nil
}

// GetIndicatorsBySource holt alle Indikatoren einer Quelle
func (r *Repository) GetIndicatorsBySource(sourceID int64) ([]Indicator, error) {
	rows, err := r.db.Query(`
		SELECT id, code, name, description, category, unit, frequency, source_id, external_code, active, created_at
		FROM indicator WHERE source_id = ? AND active = 1
		ORDER BY category ASC, name ASC
	`, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indicators := make([]Indicator, 0)
	for rows.Next() {
		var i Indicator
		if err := rows.Scan(&i.ID, &i.Code, &i.Name, &i.Description, &i.Category,
			&i.Unit, &i.Frequency, &i.SourceID, &i.ExternalCode, &i.Active, &i.CreatedAt); err != nil {
			return nil, err
		}
		indicators = append(indicators, i)
	}
	return indicators, nil
}

// --- ObservationRun CRUD ---

// CreateRun erstellt einen neuen Sammellauf
func (r *Repository) CreateRun(run *ObservationRun) error {
	result, err := r.db.Exec(`
		INSERT INTO observation_run (strategy, started_at, status, is_backfill, backfill_from, backfill_to)
		VALUES (?, ?, ?, ?, ?, ?)
	`, run.Strategy, run.StartedAt, run.Status, run.IsBackfill, run.BackfillFrom, run.BackfillTo)

	if err != nil {
		return fmt.Errorf("Sammellauf erstellen fehlgeschlagen: %w", err)
	}

	id, _ := result.LastInsertId()
	run.ID = id
	return nil
}

// UpdateRun aktualisiert einen Sammellauf
func (r *Repository) UpdateRun(run *ObservationRun) error {
	_, err := r.db.Exec(`
		UPDATE observation_run
		SET finished_at = ?, status = ?, total_records = ?, error_count = ?, error_messages = ?
		WHERE id = ?
	`, run.FinishedAt, run.Status, run.TotalRecords, run.ErrorCount, run.ErrorMessages, run.ID)
	return err
}

// GetLastRun holt den letzten Sammellauf
func (r *Repository) GetLastRun() (*ObservationRun, error) {
	run := &ObservationRun{}
	err := r.db.QueryRow(`
		SELECT id, strategy, started_at, finished_at, status, total_records, error_count, error_messages,
			is_backfill, backfill_from, backfill_to
		FROM observation_run ORDER BY started_at DESC LIMIT 1
	`).Scan(&run.ID, &run.Strategy, &run.StartedAt, &run.FinishedAt, &run.Status,
		&run.TotalRecords, &run.ErrorCount, &run.ErrorMessages,
		&run.IsBackfill, &run.BackfillFrom, &run.BackfillTo)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return run, nil
}

// GetRunsByDateRange holt Sammelläufe in einem Zeitraum
func (r *Repository) GetRunsByDateRange(from, to time.Time) ([]ObservationRun, error) {
	rows, err := r.db.Query(`
		SELECT id, strategy, started_at, finished_at, status, total_records, error_count, error_messages,
			is_backfill, backfill_from, backfill_to
		FROM observation_run
		WHERE started_at >= ? AND started_at <= ?
		ORDER BY started_at DESC
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	runs := make([]ObservationRun, 0)
	for rows.Next() {
		var run ObservationRun
		if err := rows.Scan(&run.ID, &run.Strategy, &run.StartedAt, &run.FinishedAt, &run.Status,
			&run.TotalRecords, &run.ErrorCount, &run.ErrorMessages,
			&run.IsBackfill, &run.BackfillFrom, &run.BackfillTo); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, nil
}

// --- ObservationValue CRUD ---

// CreateValue erstellt einen neuen Messwert (append-only)
func (r *Repository) CreateValue(val *ObservationValue) error {
	result, err := r.db.Exec(`
		INSERT INTO observation_value (run_id, indicator_id, source_id, observed_at, collected_at,
			value, value_string, unit, period_start, period_end, raw_response)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, val.RunID, val.IndicatorID, val.SourceID, val.ObservedAt, val.CollectedAt,
		val.Value, val.ValueString, val.Unit, val.PeriodStart, val.PeriodEnd, val.RawResponse)

	if err != nil {
		return fmt.Errorf("Messwert erstellen fehlgeschlagen: %w", err)
	}

	id, _ := result.LastInsertId()
	val.ID = id
	return nil
}

// CreateValues erstellt mehrere Messwerte in einer Transaktion
func (r *Repository) CreateValues(values []ObservationValue) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		INSERT INTO observation_value (run_id, indicator_id, source_id, observed_at, collected_at,
			value, value_string, unit, period_start, period_end, raw_response)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, val := range values {
		_, err := stmt.Exec(val.RunID, val.IndicatorID, val.SourceID, val.ObservedAt, val.CollectedAt,
			val.Value, val.ValueString, val.Unit, val.PeriodStart, val.PeriodEnd, val.RawResponse)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// GetValuesByIndicator holt alle Werte eines Indikators
func (r *Repository) GetValuesByIndicator(indicatorID int64, limit int) ([]ObservationValue, error) {
	query := `
		SELECT id, run_id, indicator_id, source_id, observed_at, collected_at,
			value, value_string, unit, period_start, period_end
		FROM observation_value WHERE indicator_id = ?
		ORDER BY observed_at DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := r.db.Query(query, indicatorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := make([]ObservationValue, 0)
	for rows.Next() {
		var v ObservationValue
		if err := rows.Scan(&v.ID, &v.RunID, &v.IndicatorID, &v.SourceID, &v.ObservedAt, &v.CollectedAt,
			&v.Value, &v.ValueString, &v.Unit, &v.PeriodStart, &v.PeriodEnd); err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, nil
}

// GetValuesByIndicatorDateRange holt Werte in einem Zeitraum
func (r *Repository) GetValuesByIndicatorDateRange(indicatorID int64, from, to time.Time) ([]ObservationValue, error) {
	rows, err := r.db.Query(`
		SELECT id, run_id, indicator_id, source_id, observed_at, collected_at,
			value, value_string, unit, period_start, period_end
		FROM observation_value
		WHERE indicator_id = ? AND observed_at >= ? AND observed_at <= ?
		ORDER BY observed_at ASC
	`, indicatorID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := make([]ObservationValue, 0)
	for rows.Next() {
		var v ObservationValue
		if err := rows.Scan(&v.ID, &v.RunID, &v.IndicatorID, &v.SourceID, &v.ObservedAt, &v.CollectedAt,
			&v.Value, &v.ValueString, &v.Unit, &v.PeriodStart, &v.PeriodEnd); err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, nil
}

// GetLatestValue holt den neuesten Wert eines Indikators
func (r *Repository) GetLatestValue(indicatorID int64) (*ObservationValue, error) {
	val := &ObservationValue{}
	err := r.db.QueryRow(`
		SELECT id, run_id, indicator_id, source_id, observed_at, collected_at,
			value, value_string, unit, period_start, period_end
		FROM observation_value WHERE indicator_id = ?
		ORDER BY observed_at DESC LIMIT 1
	`, indicatorID).Scan(&val.ID, &val.RunID, &val.IndicatorID, &val.SourceID, &val.ObservedAt, &val.CollectedAt,
		&val.Value, &val.ValueString, &val.Unit, &val.PeriodStart, &val.PeriodEnd)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return val, nil
}

// GetOldestValueDate holt das älteste Beobachtungsdatum eines Indikators
func (r *Repository) GetOldestValueDate(indicatorID int64) (*time.Time, error) {
	var date time.Time
	err := r.db.QueryRow(`
		SELECT MIN(observed_at) FROM observation_value WHERE indicator_id = ?
	`, indicatorID).Scan(&date)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &date, nil
}

// HasValueForDate prüft ob bereits ein Wert für ein Datum existiert
func (r *Repository) HasValueForDate(indicatorID int64, date time.Time) (bool, error) {
	var count int
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM observation_value
		WHERE indicator_id = ? AND observed_at >= ? AND observed_at < ?
	`, indicatorID, startOfDay, endOfDay).Scan(&count)

	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetValueCount gibt die Anzahl der Werte für einen Indikator zurück
func (r *Repository) GetValueCount(indicatorID int64) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM observation_value WHERE indicator_id = ?
	`, indicatorID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// --- Statistiken ---

// GetStats holt Observer-Statistiken
func (r *Repository) GetStats() (*ObserverStats, error) {
	stats := &ObserverStats{}

	// Anzahl Läufe
	r.db.QueryRow("SELECT COUNT(*) FROM observation_run").Scan(&stats.TotalRuns)

	// Anzahl Werte
	r.db.QueryRow("SELECT COUNT(*) FROM observation_value").Scan(&stats.TotalValues)

	// Anzahl Indikatoren
	r.db.QueryRow("SELECT COUNT(*) FROM indicator WHERE active = 1").Scan(&stats.IndicatorCount)

	// Anzahl Quellen
	r.db.QueryRow("SELECT COUNT(*) FROM data_source WHERE active = 1").Scan(&stats.SourceCount)

	// Letzter Lauf
	lastRun, _ := r.GetLastRun()
	if lastRun != nil {
		stats.LastRunAt = &lastRun.StartedAt
		stats.LastRunStatus = lastRun.Status
	}

	// Älteste/Neueste Beobachtung
	var oldest, newest time.Time
	r.db.QueryRow("SELECT MIN(observed_at) FROM observation_value").Scan(&oldest)
	r.db.QueryRow("SELECT MAX(observed_at) FROM observation_value").Scan(&newest)

	if !oldest.IsZero() {
		stats.OldestObservation = &oldest
	}
	if !newest.IsZero() {
		stats.NewestObservation = &newest
	}

	return stats, nil
}

// --- Export/Import ---

// ExportToSQL exportiert die Datenbank als SQL-Dump
func (r *Repository) ExportToSQL(outputPath string) error {
	// Öffne Output-Datei
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("Export-Datei erstellen fehlgeschlagen: %w", err)
	}
	defer f.Close()

	// Header
	f.WriteString("-- Fleet Navigator Observer Database Export\n")
	f.WriteString(fmt.Sprintf("-- Exported at: %s\n", time.Now().Format(time.RFC3339)))
	f.WriteString("-- Version: 1.0 (Phase 1)\n\n")

	// Schema
	f.WriteString("-- Schema\n")
	f.WriteString("PRAGMA foreign_keys=OFF;\n")
	f.WriteString("BEGIN TRANSACTION;\n\n")

	// data_source
	f.WriteString("-- Data Sources\n")
	sources, _ := r.GetAllSources(false)
	for _, s := range sources {
		f.WriteString(fmt.Sprintf("INSERT INTO data_source (id, code, name, description, url, source_class, active, created_at) VALUES (%d, '%s', '%s', '%s', '%s', '%s', %d, '%s');\n",
			s.ID, escapeSQL(s.Code), escapeSQL(s.Name), escapeSQL(s.Description), escapeSQL(s.URL), s.SourceClass, boolToInt(s.Active), s.CreatedAt.Format(time.RFC3339)))
	}
	f.WriteString("\n")

	// indicator
	f.WriteString("-- Indicators\n")
	indicators, _ := r.GetAllIndicators(false)
	for _, i := range indicators {
		f.WriteString(fmt.Sprintf("INSERT INTO indicator (id, code, name, description, category, unit, frequency, source_id, external_code, active, created_at) VALUES (%d, '%s', '%s', '%s', '%s', '%s', '%s', %d, '%s', %d, '%s');\n",
			i.ID, escapeSQL(i.Code), escapeSQL(i.Name), escapeSQL(i.Description), i.Category, escapeSQL(i.Unit), i.Frequency, i.SourceID, escapeSQL(i.ExternalCode), boolToInt(i.Active), i.CreatedAt.Format(time.RFC3339)))
	}
	f.WriteString("\n")

	// observation_run
	f.WriteString("-- Observation Runs\n")
	rows, err := r.db.Query("SELECT id, strategy, started_at, finished_at, status, total_records, error_count, error_messages, is_backfill, backfill_from, backfill_to FROM observation_run")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var run ObservationRun
			rows.Scan(&run.ID, &run.Strategy, &run.StartedAt, &run.FinishedAt, &run.Status,
				&run.TotalRecords, &run.ErrorCount, &run.ErrorMessages, &run.IsBackfill, &run.BackfillFrom, &run.BackfillTo)

			finishedAt := "NULL"
			if run.FinishedAt != nil {
				finishedAt = fmt.Sprintf("'%s'", run.FinishedAt.Format(time.RFC3339))
			}
			backfillFrom := "NULL"
			if run.BackfillFrom != nil {
				backfillFrom = fmt.Sprintf("'%s'", run.BackfillFrom.Format(time.RFC3339))
			}
			backfillTo := "NULL"
			if run.BackfillTo != nil {
				backfillTo = fmt.Sprintf("'%s'", run.BackfillTo.Format(time.RFC3339))
			}

			f.WriteString(fmt.Sprintf("INSERT INTO observation_run (id, strategy, started_at, finished_at, status, total_records, error_count, error_messages, is_backfill, backfill_from, backfill_to) VALUES (%d, '%s', '%s', %s, '%s', %d, %d, '%s', %d, %s, %s);\n",
				run.ID, run.Strategy, run.StartedAt.Format(time.RFC3339), finishedAt, run.Status, run.TotalRecords, run.ErrorCount, escapeSQL(run.ErrorMessages), boolToInt(run.IsBackfill), backfillFrom, backfillTo))
		}
	}
	f.WriteString("\n")

	// observation_value (kann sehr groß sein - chunked)
	f.WriteString("-- Observation Values\n")
	valRows, err := r.db.Query("SELECT id, run_id, indicator_id, source_id, observed_at, collected_at, value, value_string, unit, period_start, period_end FROM observation_value")
	if err == nil {
		defer valRows.Close()
		for valRows.Next() {
			var v ObservationValue
			valRows.Scan(&v.ID, &v.RunID, &v.IndicatorID, &v.SourceID, &v.ObservedAt, &v.CollectedAt,
				&v.Value, &v.ValueString, &v.Unit, &v.PeriodStart, &v.PeriodEnd)

			f.WriteString(fmt.Sprintf("INSERT INTO observation_value (id, run_id, indicator_id, source_id, observed_at, collected_at, value, value_string, unit, period_start, period_end) VALUES (%d, %d, %d, %d, '%s', '%s', %f, '%s', '%s', '%s', '%s');\n",
				v.ID, v.RunID, v.IndicatorID, v.SourceID, v.ObservedAt.Format(time.RFC3339), v.CollectedAt.Format(time.RFC3339),
				v.Value, escapeSQL(v.ValueString), escapeSQL(v.Unit), v.PeriodStart.Format(time.RFC3339), v.PeriodEnd.Format(time.RFC3339)))
		}
	}

	f.WriteString("\nCOMMIT;\n")
	f.WriteString("PRAGMA foreign_keys=ON;\n")

	log.Printf("Observer: Datenbank exportiert nach %s", outputPath)
	return nil
}

// ImportFromSQL importiert einen SQL-Dump
func (r *Repository) ImportFromSQL(inputPath string) error {
	content, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("Import-Datei lesen fehlgeschlagen: %w", err)
	}

	_, err = r.db.Exec(string(content))
	if err != nil {
		return fmt.Errorf("SQL ausführen fehlgeschlagen: %w", err)
	}

	log.Printf("Observer: Datenbank importiert von %s", inputPath)
	return nil
}

// --- Seed-Daten ---

// SeedDefaultData fügt die Standard-Quellen und -Indikatoren ein
func (r *Repository) SeedDefaultData() error {
	log.Printf("Observer: Prüfe und ergänze Seed-Daten...")

	// === Datenquellen ===
	sources := []DataSource{
		// Offizielle Quellen
		{
			Code:        "ECB",
			Name:        "Europäische Zentralbank",
			Description: "Official ECB Data Portal API",
			URL:         "https://data-api.ecb.europa.eu",
			SourceClass: SourceClassOfficial,
			Active:      true,
		},
		{
			Code:        "BUNDESBANK",
			Name:        "Deutsche Bundesbank",
			Description: "Bundesbank SDMX Webservice",
			URL:         "https://api.statistiken.bundesbank.de",
			SourceClass: SourceClassOfficial,
			Active:      true,
		},
		{
			Code:        "ESTR",
			Name:        "Euro Short-Term Rate API",
			Description: "Free JSON API for €STR data",
			URL:         "https://estr.dev",
			SourceClass: SourceClassSemiOfficial,
			Active:      true,
		},
		// Marktdaten-Quellen
		{
			Code:        "COINGECKO",
			Name:        "CoinGecko",
			Description: "Free Cryptocurrency API",
			URL:         "https://api.coingecko.com",
			SourceClass: SourceClassSemiOfficial,
			Active:      true,
		},
		{
			Code:        "YAHOO",
			Name:        "Yahoo Finance",
			Description: "Stock and commodity market data",
			URL:         "https://finance.yahoo.com",
			SourceClass: SourceClassSemiOfficial,
			Active:      true,
		},
	}

	sourcesAdded := 0
	for i := range sources {
		existing, _ := r.GetSourceByCode(sources[i].Code)
		if existing == nil {
			if err := r.CreateSource(&sources[i]); err != nil {
				log.Printf("Observer: Warnung - Quelle %s: %v", sources[i].Code, err)
			} else {
				sourcesAdded++
			}
		}
	}

	// Quellen laden
	ecbSource, _ := r.GetSourceByCode("ECB")
	bundesbankSource, _ := r.GetSourceByCode("BUNDESBANK")
	estrSource, _ := r.GetSourceByCode("ESTR")
	coinGeckoSource, _ := r.GetSourceByCode("COINGECKO")
	yahooSource, _ := r.GetSourceByCode("YAHOO")

	// === Indikatoren ===
	indicators := []Indicator{
		// ============ BASIS-DATEN ============
		// Leitzinsen
		{
			Code:         "ECB_MAIN_RATE",
			Name:         "EZB Hauptrefinanzierungssatz",
			Description:  "Main refinancing operations rate",
			Category:     CategoryInterestRate,
			Unit:         "%",
			Frequency:    "D",
			SourceID:     ecbSource.ID,
			ExternalCode: "FM.D.U2.EUR.4F.KR.MRR_FR.LEV",
			Active:       true,
		},
		{
			Code:         "ECB_DEPOSIT_RATE",
			Name:         "EZB Einlagefazilität",
			Description:  "Deposit facility rate",
			Category:     CategoryInterestRate,
			Unit:         "%",
			Frequency:    "D",
			SourceID:     ecbSource.ID,
			ExternalCode: "FM.D.U2.EUR.4F.KR.DFR.LEV",
			Active:       true,
		},
		{
			Code:         "ECB_MARGINAL_RATE",
			Name:         "EZB Spitzenrefinanzierungsfazilität",
			Description:  "Marginal lending facility rate",
			Category:     CategoryInterestRate,
			Unit:         "%",
			Frequency:    "D",
			SourceID:     ecbSource.ID,
			ExternalCode: "FM.D.U2.EUR.4F.KR.MLFR.LEV",
			Active:       true,
		},
		{
			Code:         "ESTR",
			Name:         "Euro Short-Term Rate (€STR)",
			Description:  "Overnight unsecured rate",
			Category:     CategoryInterestRate,
			Unit:         "%",
			Frequency:    "D",
			SourceID:     estrSource.ID,
			ExternalCode: "current",
			Active:       true,
		},
		// Inflation
		{
			Code:         "HICP_EA",
			Name:         "HVPI Eurozone (Gesamtindex)",
			Description:  "Harmonised Index of Consumer Prices - Euro Area",
			Category:     CategoryInflation,
			Unit:         "% YoY",
			Frequency:    "M",
			SourceID:     ecbSource.ID,
			ExternalCode: "ICP.M.U2.N.000000.4.ANR",
			Active:       true,
		},
		{
			Code:         "HICP_DE",
			Name:         "HVPI Deutschland",
			Description:  "Harmonised Index of Consumer Prices - Germany",
			Category:     CategoryInflation,
			Unit:         "% YoY",
			Frequency:    "M",
			SourceID:     ecbSource.ID,
			ExternalCode: "ICP.M.DE.N.000000.4.ANR",
			Active:       true,
		},
		// Bundesbank-Indikatoren
		{
			Code:         "DE_10Y_YIELD",
			Name:         "Bundesanleihe 10 Jahre",
			Description:  "Umlaufrendite deutscher Staatsanleihen 10J",
			Category:     CategoryInterestRate,
			Unit:         "%",
			Frequency:    "D",
			SourceID:     bundesbankSource.ID,
			ExternalCode: "BBSIS.D.I.ZST.ZI.EUR.S1311.B.A604.R10XX.R.A.A._Z._Z.A",
			Active:       true,
		},
		// Arbeitsmarkt
		{
			Code:         "UNEMPLOYMENT_EA",
			Name:         "Arbeitslosenquote Eurozone",
			Description:  "Unemployment rate - Euro Area",
			Category:     CategoryEmployment,
			Unit:         "%",
			Frequency:    "M",
			SourceID:     ecbSource.ID,
			ExternalCode: "LFSI.M.I9.S.UNEHRT.TOTAL0.15_74.T",
			Active:       true,
		},

		// ============ AKTIENMÄRKTE ============
		{
			Code:         "DAX",
			Name:         "DAX Performance Index",
			Description:  "Deutscher Aktienindex - 40 größte Unternehmen",
			Category:     CategoryStocks,
			Unit:         "Punkte",
			Frequency:    "D",
			SourceID:     yahooSource.ID,
			ExternalCode: "^GDAXI",
			Active:       true,
		},
		{
			Code:         "MSCI_WORLD",
			Name:         "MSCI World Index",
			Description:  "Weltweiter Aktienindex (via URTH ETF)",
			Category:     CategoryStocks,
			Unit:         "USD",
			Frequency:    "D",
			SourceID:     yahooSource.ID,
			ExternalCode: "URTH",
			Active:       true,
		},
		{
			Code:         "SP500",
			Name:         "S&P 500",
			Description:  "US-Aktienindex - 500 größte Unternehmen",
			Category:     CategoryStocks,
			Unit:         "Punkte",
			Frequency:    "D",
			SourceID:     yahooSource.ID,
			ExternalCode: "^GSPC",
			Active:       true,
		},

		// ============ ANLEIHEN (US) ============
		{
			Code:         "US_10Y_YIELD",
			Name:         "US Treasury 10 Jahre",
			Description:  "US-Staatsanleihen 10J Rendite",
			Category:     CategoryInterestRate,
			Unit:         "%",
			Frequency:    "D",
			SourceID:     yahooSource.ID,
			ExternalCode: "^TNX",
			Active:       true,
		},

		// ============ ROHSTOFFE ============
		{
			Code:         "GOLD_EUR",
			Name:         "Goldpreis (EUR/oz)",
			Description:  "Gold Spot-Preis in Euro pro Unze",
			Category:     CategoryCommodities,
			Unit:         "EUR/oz",
			Frequency:    "D",
			SourceID:     yahooSource.ID,
			ExternalCode: "GC=F",
			Active:       true,
		},
		{
			Code:         "SILVER_EUR",
			Name:         "Silberpreis (EUR/oz)",
			Description:  "Silber Spot-Preis in Euro pro Unze",
			Category:     CategoryCommodities,
			Unit:         "EUR/oz",
			Frequency:    "D",
			SourceID:     yahooSource.ID,
			ExternalCode: "SI=F",
			Active:       true,
		},
		{
			Code:         "OIL_BRENT",
			Name:         "Brent Rohöl (USD/bbl)",
			Description:  "Brent Crude Oil Preis in USD pro Barrel",
			Category:     CategoryCommodities,
			Unit:         "USD/bbl",
			Frequency:    "D",
			SourceID:     yahooSource.ID,
			ExternalCode: "BZ=F",
			Active:       true,
		},

		// ============ IMMOBILIEN ============
		{
			Code:         "EPRA_EURO",
			Name:         "EPRA Eurozone Index",
			Description:  "Europäischer REIT-Index (via ETF)",
			Category:     CategoryRealEstate,
			Unit:         "EUR",
			Frequency:    "D",
			SourceID:     yahooSource.ID,
			ExternalCode: "IPRP.DE",
			Active:       true,
		},

		// ============ KRYPTOWÄHRUNGEN ============
		{
			Code:         "BTC_EUR",
			Name:         "Bitcoin (EUR)",
			Description:  "Bitcoin-Preis in Euro",
			Category:     CategoryCrypto,
			Unit:         "EUR",
			Frequency:    "D",
			SourceID:     coinGeckoSource.ID,
			ExternalCode: "bitcoin",
			Active:       true,
		},
		{
			Code:         "ETH_EUR",
			Name:         "Ethereum (EUR)",
			Description:  "Ethereum-Preis in Euro",
			Category:     CategoryCrypto,
			Unit:         "EUR",
			Frequency:    "D",
			SourceID:     coinGeckoSource.ID,
			ExternalCode: "ethereum",
			Active:       true,
		},
	}

	indicatorsAdded := 0
	for i := range indicators {
		existing, _ := r.GetIndicatorByCode(indicators[i].Code)
		if existing == nil {
			if err := r.CreateIndicator(&indicators[i]); err != nil {
				log.Printf("Observer: Warnung - Indikator %s: %v", indicators[i].Code, err)
			} else {
				indicatorsAdded++
			}
		}
	}

	if sourcesAdded > 0 || indicatorsAdded > 0 {
		log.Printf("Observer: %d neue Quellen und %d neue Indikatoren hinzugefügt", sourcesAdded, indicatorsAdded)
	} else {
		log.Printf("Observer: Alle Seed-Daten bereits vorhanden")
	}

	return nil
}

// Hilfsfunktionen

func escapeSQL(s string) string {
	// Einfaches Escaping für SQL-Strings
	result := ""
	for _, c := range s {
		if c == '\'' {
			result += "''"
		} else {
			result += string(c)
		}
	}
	return result
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// GetIndicatorHistory holt die komplette Historie eines Indikators
func (r *Repository) GetIndicatorHistory(indicatorCode string) (*IndicatorHistory, error) {
	// Indikator laden
	indicator, err := r.GetIndicatorByCode(indicatorCode)
	if err != nil {
		return nil, err
	}
	if indicator == nil {
		return nil, fmt.Errorf("Indikator nicht gefunden: %s", indicatorCode)
	}

	// Quelle laden
	source, _ := r.GetSourceByCode("")
	rows, err := r.db.Query(`
		SELECT d.id, d.code, d.name, d.description, d.url, d.source_class, d.active, d.created_at
		FROM data_source d WHERE d.id = ?
	`, indicator.SourceID)
	if err == nil {
		defer rows.Close()
		if rows.Next() {
			source = &DataSource{}
			rows.Scan(&source.ID, &source.Code, &source.Name, &source.Description,
				&source.URL, &source.SourceClass, &source.Active, &source.CreatedAt)
		}
	}

	// Werte laden
	values, err := r.GetValuesByIndicator(indicator.ID, 0)
	if err != nil {
		return nil, err
	}

	// Zu IndicatorValue konvertieren
	history := &IndicatorHistory{
		Indicator: indicator,
		Source:    source,
		Values:    make([]IndicatorValue, len(values)),
	}

	for i, v := range values {
		history.Values[i] = IndicatorValue{
			Date:  v.ObservedAt,
			Value: v.Value,
			Unit:  v.Unit,
		}
	}

	return history, nil
}

// GetMissingDates findet fehlende Daten für Backfill
func (r *Repository) GetMissingDates(indicatorID int64, from, to time.Time, frequency string) ([]time.Time, error) {
	// Existierende Daten laden
	existing := make(map[string]bool)
	rows, err := r.db.Query(`
		SELECT DATE(observed_at) FROM observation_value
		WHERE indicator_id = ? AND observed_at >= ? AND observed_at <= ?
	`, indicatorID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var dateStr string
		rows.Scan(&dateStr)
		existing[dateStr] = true
	}

	// Fehlende Daten finden basierend auf Frequenz
	missing := make([]time.Time, 0)
	current := from

	for current.Before(to) || current.Equal(to) {
		dateStr := current.Format("2006-01-02")
		if !existing[dateStr] {
			// Bei Daily: nur Wochentage (Mo-Fr)
			if frequency == "D" {
				weekday := current.Weekday()
				if weekday != time.Saturday && weekday != time.Sunday {
					missing = append(missing, current)
				}
			} else {
				missing = append(missing, current)
			}
		}

		// Nächster Tag/Woche/Monat basierend auf Frequenz
		switch frequency {
		case "D":
			current = current.AddDate(0, 0, 1)
		case "W":
			current = current.AddDate(0, 0, 7)
		case "M":
			current = current.AddDate(0, 1, 0)
		case "Q":
			current = current.AddDate(0, 3, 0)
		default:
			current = current.AddDate(0, 0, 1)
		}
	}

	return missing, nil
}

// Checkpoint erstellt einen Snapshot der aktuellen Statistiken (für JSON-Export)
func (r *Repository) Checkpoint() ([]byte, error) {
	stats, err := r.GetStats()
	if err != nil {
		return nil, err
	}
	return json.Marshal(stats)
}
