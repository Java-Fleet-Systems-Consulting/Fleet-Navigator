package prompts

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Repository verwaltet System-Prompts in SQLite
type Repository struct {
	db *sql.DB
}

// NewRepository erstellt ein neues Repository
func NewRepository(dataDir string) (*Repository, error) {
	dbPath := filepath.Join(dataDir, "prompts.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("prompts DB öffnen: %w", err)
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
	CREATE TABLE IF NOT EXISTS system_prompts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		content TEXT NOT NULL,
		is_default INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_prompts_default ON system_prompts(is_default);
	`

	_, err := r.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("prompts Schema erstellen: %w", err)
	}

	return nil
}

// Close schließt die Datenbankverbindung
func (r *Repository) Close() error {
	return r.db.Close()
}

// GetAll lädt alle System-Prompts
func (r *Repository) GetAll() ([]SystemPromptTemplate, error) {
	rows, err := r.db.Query(`
		SELECT id, name, content, is_default, created_at
		FROM system_prompts
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prompts []SystemPromptTemplate
	for rows.Next() {
		var p SystemPromptTemplate
		var isDefault int
		err := rows.Scan(&p.ID, &p.Name, &p.Content, &isDefault, &p.CreatedAt)
		if err != nil {
			return nil, err
		}
		p.IsDefault = isDefault == 1
		prompts = append(prompts, p)
	}

	if prompts == nil {
		prompts = []SystemPromptTemplate{}
	}

	return prompts, nil
}

// GetByID lädt einen System-Prompt
func (r *Repository) GetByID(id int64) (*SystemPromptTemplate, error) {
	var p SystemPromptTemplate
	var isDefault int

	err := r.db.QueryRow(`
		SELECT id, name, content, is_default, created_at
		FROM system_prompts
		WHERE id = ?
	`, id).Scan(&p.ID, &p.Name, &p.Content, &isDefault, &p.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	p.IsDefault = isDefault == 1
	return &p, nil
}

// GetDefault lädt den Standard-Prompt
func (r *Repository) GetDefault() (*SystemPromptTemplate, error) {
	var p SystemPromptTemplate
	var isDefault int

	err := r.db.QueryRow(`
		SELECT id, name, content, is_default, created_at
		FROM system_prompts
		WHERE is_default = 1
		LIMIT 1
	`).Scan(&p.ID, &p.Name, &p.Content, &isDefault, &p.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	p.IsDefault = isDefault == 1
	return &p, nil
}

// Create erstellt einen neuen System-Prompt
func (r *Repository) Create(prompt *SystemPromptTemplate) error {
	now := time.Now()
	isDefault := 0
	if prompt.IsDefault {
		isDefault = 1
	}

	result, err := r.db.Exec(`
		INSERT INTO system_prompts (name, content, is_default, created_at)
		VALUES (?, ?, ?, ?)
	`, prompt.Name, prompt.Content, isDefault, now)

	if err != nil {
		return fmt.Errorf("prompt erstellen: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	prompt.ID = id
	prompt.CreatedAt = now
	return nil
}

// Update aktualisiert einen System-Prompt
func (r *Repository) Update(prompt *SystemPromptTemplate) error {
	isDefault := 0
	if prompt.IsDefault {
		isDefault = 1
	}

	_, err := r.db.Exec(`
		UPDATE system_prompts
		SET name = ?, content = ?, is_default = ?
		WHERE id = ?
	`, prompt.Name, prompt.Content, isDefault, prompt.ID)

	return err
}

// Delete löscht einen System-Prompt
func (r *Repository) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM system_prompts WHERE id = ?", id)
	return err
}

// SetDefault setzt einen Prompt als Standard (alle anderen werden zurückgesetzt)
func (r *Repository) SetDefault(id int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Alle auf nicht-default setzen
	_, err = tx.Exec("UPDATE system_prompts SET is_default = 0")
	if err != nil {
		return err
	}

	// Diesen als default setzen
	_, err = tx.Exec("UPDATE system_prompts SET is_default = 1 WHERE id = ?", id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Count zählt alle Prompts
func (r *Repository) Count() (int64, error) {
	var count int64
	err := r.db.QueryRow("SELECT COUNT(*) FROM system_prompts").Scan(&count)
	return count, err
}
