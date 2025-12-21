package modeltemplates

import (
	"database/sql"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

// Repository verwaltet Model-Templates in SQLite
type Repository struct {
	db *sql.DB
}

// NewRepository erstellt ein neues Repository
func NewRepository(dbPath string) (*Repository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	repo := &Repository{db: db}
	if err := repo.createSchema(); err != nil {
		return nil, err
	}

	return repo, nil
}

// NewRepositoryWithDB erstellt ein Repository mit bestehender DB-Verbindung
func NewRepositoryWithDB(db *sql.DB) (*Repository, error) {
	repo := &Repository{db: db}
	if err := repo.createSchema(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *Repository) createSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS model_templates (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		pattern TEXT NOT NULL,
		template_format TEXT NOT NULL,
		supports_system_role INTEGER DEFAULT 1,
		system_embed_strategy TEXT DEFAULT 'native',
		system_prefix TEXT DEFAULT '',
		system_suffix TEXT DEFAULT '',
		description TEXT,
		priority INTEGER DEFAULT 100,
		is_active INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_templates_pattern ON model_templates(pattern);
	CREATE INDEX IF NOT EXISTS idx_templates_priority ON model_templates(priority DESC);
	CREATE INDEX IF NOT EXISTS idx_templates_active ON model_templates(is_active);
	`

	_, err := r.db.Exec(schema)
	return err
}

// Close schließt die Datenbankverbindung
func (r *Repository) Close() error {
	return r.db.Close()
}

// GetAll gibt alle Templates zurück (sortiert nach Priorität)
func (r *Repository) GetAll() ([]ModelTemplate, error) {
	rows, err := r.db.Query(`
		SELECT id, name, pattern, template_format, supports_system_role,
			system_embed_strategy, system_prefix, system_suffix, description,
			priority, is_active, created_at, updated_at
		FROM model_templates
		ORDER BY priority DESC, name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []ModelTemplate
	for rows.Next() {
		var t ModelTemplate
		var supportsSystem, isActive int
		err := rows.Scan(&t.ID, &t.Name, &t.Pattern, &t.TemplateFormat,
			&supportsSystem, &t.SystemEmbedStrategy, &t.SystemPrefix,
			&t.SystemSuffix, &t.Description, &t.Priority, &isActive,
			&t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, err
		}
		t.SupportsSystemRole = supportsSystem == 1
		t.IsActive = isActive == 1
		templates = append(templates, t)
	}

	return templates, nil
}

// GetActive gibt alle aktiven Templates zurück
func (r *Repository) GetActive() ([]ModelTemplate, error) {
	rows, err := r.db.Query(`
		SELECT id, name, pattern, template_format, supports_system_role,
			system_embed_strategy, system_prefix, system_suffix, description,
			priority, is_active, created_at, updated_at
		FROM model_templates
		WHERE is_active = 1
		ORDER BY priority DESC, name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []ModelTemplate
	for rows.Next() {
		var t ModelTemplate
		var supportsSystem, isActive int
		err := rows.Scan(&t.ID, &t.Name, &t.Pattern, &t.TemplateFormat,
			&supportsSystem, &t.SystemEmbedStrategy, &t.SystemPrefix,
			&t.SystemSuffix, &t.Description, &t.Priority, &isActive,
			&t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, err
		}
		t.SupportsSystemRole = supportsSystem == 1
		t.IsActive = isActive == 1
		templates = append(templates, t)
	}

	return templates, nil
}

// GetByID gibt ein Template zurück
func (r *Repository) GetByID(id int64) (*ModelTemplate, error) {
	var t ModelTemplate
	var supportsSystem, isActive int

	err := r.db.QueryRow(`
		SELECT id, name, pattern, template_format, supports_system_role,
			system_embed_strategy, system_prefix, system_suffix, description,
			priority, is_active, created_at, updated_at
		FROM model_templates WHERE id = ?
	`, id).Scan(&t.ID, &t.Name, &t.Pattern, &t.TemplateFormat,
		&supportsSystem, &t.SystemEmbedStrategy, &t.SystemPrefix,
		&t.SystemSuffix, &t.Description, &t.Priority, &isActive,
		&t.CreatedAt, &t.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	t.SupportsSystemRole = supportsSystem == 1
	t.IsActive = isActive == 1
	return &t, nil
}

// Create erstellt ein neues Template
func (r *Repository) Create(t *ModelTemplate) error {
	now := time.Now()
	supportsSystem := 0
	if t.SupportsSystemRole {
		supportsSystem = 1
	}
	isActive := 0
	if t.IsActive {
		isActive = 1
	}

	result, err := r.db.Exec(`
		INSERT INTO model_templates (
			name, pattern, template_format, supports_system_role,
			system_embed_strategy, system_prefix, system_suffix,
			description, priority, is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.Name, t.Pattern, t.TemplateFormat, supportsSystem,
		t.SystemEmbedStrategy, t.SystemPrefix, t.SystemSuffix,
		t.Description, t.Priority, isActive, now, now)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	t.ID = id
	t.CreatedAt = now
	t.UpdatedAt = now
	return nil
}

// Update aktualisiert ein Template
func (r *Repository) Update(t *ModelTemplate) error {
	supportsSystem := 0
	if t.SupportsSystemRole {
		supportsSystem = 1
	}
	isActive := 0
	if t.IsActive {
		isActive = 1
	}

	t.UpdatedAt = time.Now()

	_, err := r.db.Exec(`
		UPDATE model_templates SET
			name = ?, pattern = ?, template_format = ?, supports_system_role = ?,
			system_embed_strategy = ?, system_prefix = ?, system_suffix = ?,
			description = ?, priority = ?, is_active = ?, updated_at = ?
		WHERE id = ?
	`, t.Name, t.Pattern, t.TemplateFormat, supportsSystem,
		t.SystemEmbedStrategy, t.SystemPrefix, t.SystemSuffix,
		t.Description, t.Priority, isActive, t.UpdatedAt, t.ID)

	return err
}

// Delete löscht ein Template
func (r *Repository) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM model_templates WHERE id = ?", id)
	return err
}

// Count zählt alle Templates
func (r *Repository) Count() (int64, error) {
	var count int64
	err := r.db.QueryRow("SELECT COUNT(*) FROM model_templates").Scan(&count)
	return count, err
}

// SeedDefaults fügt die Standard-Templates ein (nur wenn Tabelle leer)
func (r *Repository) SeedDefaults() error {
	count, err := r.Count()
	if err != nil {
		return err
	}

	if count > 0 {
		log.Println("Model-Templates bereits vorhanden, überspringe Seed")
		return nil
	}

	log.Println("Erstelle Standard-Model-Templates...")

	for _, t := range DefaultTemplates() {
		if err := r.Create(&t); err != nil {
			log.Printf("Fehler beim Erstellen von Template %s: %v", t.Name, err)
			continue
		}
		log.Printf("  ✓ Template erstellt: %s (Pattern: %s)", t.Name, t.Pattern)
	}

	return nil
}
