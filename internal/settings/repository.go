package settings

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Repository verwaltet App-Einstellungen in SQLite
type Repository struct {
	db *sql.DB
}

// NewRepository erstellt ein neues Repository
func NewRepository(dataDir string) (*Repository, error) {
	dbPath := filepath.Join(dataDir, "settings.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("settings DB öffnen: %w", err)
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)

	repo := &Repository{db: db}
	if err := repo.createSchema(); err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *Repository) createSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS app_settings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		setting_key TEXT UNIQUE NOT NULL,
		setting_value TEXT,
		description TEXT,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_settings_key ON app_settings(setting_key);
	`

	_, err := r.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("settings Schema erstellen: %w", err)
	}

	return nil
}

// Close schließt die Datenbankverbindung
func (r *Repository) Close() error {
	return r.db.Close()
}

// Get holt einen Einstellungswert
func (r *Repository) Get(key string) (string, error) {
	var value string
	err := r.db.QueryRow(`
		SELECT setting_value FROM app_settings WHERE setting_key = ?
	`, key).Scan(&value)

	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	return value, nil
}

// Set speichert einen Einstellungswert
func (r *Repository) Set(key, value string) error {
	now := time.Now()

	_, err := r.db.Exec(`
		INSERT INTO app_settings (setting_key, setting_value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(setting_key) DO UPDATE SET
			setting_value = excluded.setting_value,
			updated_at = excluded.updated_at
	`, key, value, now)

	return err
}

// SetWithDescription speichert einen Einstellungswert mit Beschreibung
func (r *Repository) SetWithDescription(key, value, description string) error {
	now := time.Now()

	_, err := r.db.Exec(`
		INSERT INTO app_settings (setting_key, setting_value, description, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(setting_key) DO UPDATE SET
			setting_value = excluded.setting_value,
			description = excluded.description,
			updated_at = excluded.updated_at
	`, key, value, description, now)

	return err
}

// Delete löscht eine Einstellung
func (r *Repository) Delete(key string) error {
	_, err := r.db.Exec("DELETE FROM app_settings WHERE setting_key = ?", key)
	return err
}

// GetAll holt alle Einstellungen
func (r *Repository) GetAll() ([]AppSetting, error) {
	rows, err := r.db.Query(`
		SELECT id, setting_key, setting_value, COALESCE(description, ''), updated_at
		FROM app_settings
		ORDER BY setting_key
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []AppSetting
	for rows.Next() {
		var s AppSetting
		err := rows.Scan(&s.ID, &s.Key, &s.Value, &s.Description, &s.UpdatedAt)
		if err != nil {
			return nil, err
		}
		settings = append(settings, s)
	}

	if settings == nil {
		settings = []AppSetting{}
	}

	return settings, nil
}

// GetByPrefix holt alle Einstellungen mit einem bestimmten Prefix
func (r *Repository) GetByPrefix(prefix string) (map[string]string, error) {
	rows, err := r.db.Query(`
		SELECT setting_key, setting_value FROM app_settings
		WHERE setting_key LIKE ?
	`, prefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		result[key] = value
	}

	return result, nil
}

// GetOrDefault holt einen Wert oder gibt den Default zurück
func (r *Repository) GetOrDefault(key, defaultValue string) string {
	value, err := r.Get(key)
	if err != nil || value == "" {
		return defaultValue
	}
	return value
}

// DeleteByPrefix löscht alle Einstellungen mit einem bestimmten Prefix
func (r *Repository) DeleteByPrefix(prefix string) error {
	_, err := r.db.Exec("DELETE FROM app_settings WHERE setting_key LIKE ?", prefix+"%")
	return err
}
