package custommodel

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Repository verwaltet Custom-Model-Daten in SQLite
type Repository struct {
	db *sql.DB
}

// NewRepository erstellt ein neues Repository
func NewRepository(dataDir string) (*Repository, error) {
	dbPath := filepath.Join(dataDir, "custom_models.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("Custom-Models-DB öffnen: %w", err)
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
	CREATE TABLE IF NOT EXISTS custom_models (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		base_model TEXT NOT NULL,
		system_prompt TEXT,
		description TEXT,
		temperature REAL,
		top_p REAL,
		top_k INTEGER,
		repeat_penalty REAL,
		num_predict INTEGER,
		num_ctx INTEGER,
		ollama_digest TEXT,
		parent_model_id INTEGER,
		version INTEGER DEFAULT 1,
		modelfile TEXT NOT NULL,
		size INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (parent_model_id) REFERENCES custom_models(id) ON DELETE SET NULL
	);

	CREATE INDEX IF NOT EXISTS idx_custom_models_name ON custom_models(name);
	CREATE INDEX IF NOT EXISTS idx_custom_models_base_model ON custom_models(base_model);
	CREATE INDEX IF NOT EXISTS idx_custom_models_parent ON custom_models(parent_model_id);

	-- GGUF Model Configs (für llama-server)
	CREATE TABLE IF NOT EXISTS gguf_model_configs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		base_model TEXT NOT NULL,
		description TEXT,
		system_prompt TEXT,
		temperature REAL DEFAULT 0.8,
		top_p REAL DEFAULT 0.9,
		top_k INTEGER DEFAULT 40,
		repeat_penalty REAL DEFAULT 1.1,
		max_tokens INTEGER DEFAULT 2048,
		context_size INTEGER DEFAULT 8192,
		gpu_layers INTEGER DEFAULT -1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_gguf_configs_name ON gguf_model_configs(name);
	`

	_, err := r.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("Custom-Models-Schema erstellen: %w", err)
	}

	return nil
}

// Close schließt die Datenbankverbindung
func (r *Repository) Close() error {
	return r.db.Close()
}

// Create erstellt ein neues Custom Model
func (r *Repository) Create(model *CustomModel) error {
	now := time.Now()
	model.CreatedAt = now
	model.UpdatedAt = now

	if model.Version == 0 {
		model.Version = 1
	}

	result, err := r.db.Exec(`
		INSERT INTO custom_models (
			name, base_model, system_prompt, description, temperature, top_p, top_k,
			repeat_penalty, num_predict, num_ctx, ollama_digest, parent_model_id,
			version, modelfile, size, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, model.Name, model.BaseModel, model.SystemPrompt, model.Description,
		model.Temperature, model.TopP, model.TopK, model.RepeatPenalty,
		model.NumPredict, model.NumCtx, model.OllamaDigest, model.ParentModelID,
		model.Version, model.Modelfile, model.Size, now, now)

	if err != nil {
		return fmt.Errorf("Custom Model erstellen: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	model.ID = id
	return nil
}

// GetByID holt ein Custom Model nach ID
func (r *Repository) GetByID(id int64) (*CustomModel, error) {
	model := &CustomModel{}

	err := r.db.QueryRow(`
		SELECT id, name, base_model, system_prompt, description, temperature, top_p, top_k,
			repeat_penalty, num_predict, num_ctx, ollama_digest, parent_model_id,
			version, modelfile, size, created_at, updated_at
		FROM custom_models WHERE id = ?
	`, id).Scan(&model.ID, &model.Name, &model.BaseModel, &model.SystemPrompt,
		&model.Description, &model.Temperature, &model.TopP, &model.TopK,
		&model.RepeatPenalty, &model.NumPredict, &model.NumCtx, &model.OllamaDigest,
		&model.ParentModelID, &model.Version, &model.Modelfile, &model.Size,
		&model.CreatedAt, &model.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return model, nil
}

// GetByName holt ein Custom Model nach Name
func (r *Repository) GetByName(name string) (*CustomModel, error) {
	model := &CustomModel{}

	err := r.db.QueryRow(`
		SELECT id, name, base_model, system_prompt, description, temperature, top_p, top_k,
			repeat_penalty, num_predict, num_ctx, ollama_digest, parent_model_id,
			version, modelfile, size, created_at, updated_at
		FROM custom_models WHERE name = ?
	`, name).Scan(&model.ID, &model.Name, &model.BaseModel, &model.SystemPrompt,
		&model.Description, &model.Temperature, &model.TopP, &model.TopK,
		&model.RepeatPenalty, &model.NumPredict, &model.NumCtx, &model.OllamaDigest,
		&model.ParentModelID, &model.Version, &model.Modelfile, &model.Size,
		&model.CreatedAt, &model.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return model, nil
}

// GetAll holt alle Custom Models
func (r *Repository) GetAll() ([]CustomModel, error) {
	rows, err := r.db.Query(`
		SELECT id, name, base_model, system_prompt, description, temperature, top_p, top_k,
			repeat_penalty, num_predict, num_ctx, ollama_digest, parent_model_id,
			version, modelfile, size, created_at, updated_at
		FROM custom_models
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []CustomModel
	for rows.Next() {
		var m CustomModel
		err := rows.Scan(&m.ID, &m.Name, &m.BaseModel, &m.SystemPrompt,
			&m.Description, &m.Temperature, &m.TopP, &m.TopK,
			&m.RepeatPenalty, &m.NumPredict, &m.NumCtx, &m.OllamaDigest,
			&m.ParentModelID, &m.Version, &m.Modelfile, &m.Size,
			&m.CreatedAt, &m.UpdatedAt)
		if err != nil {
			return nil, err
		}
		models = append(models, m)
	}

	if models == nil {
		models = []CustomModel{}
	}

	return models, nil
}

// GetAncestry holt die Versionskette eines Models
func (r *Repository) GetAncestry(id int64) ([]CustomModel, error) {
	var ancestry []CustomModel

	// Rekursiv nach oben traversieren
	currentID := &id
	for currentID != nil {
		model, err := r.GetByID(*currentID)
		if err != nil {
			return nil, err
		}
		if model == nil {
			break
		}

		ancestry = append(ancestry, *model)
		currentID = model.ParentModelID
	}

	return ancestry, nil
}

// Update aktualisiert ein Custom Model
func (r *Repository) Update(model *CustomModel) error {
	model.UpdatedAt = time.Now()

	_, err := r.db.Exec(`
		UPDATE custom_models SET
			name = ?, base_model = ?, system_prompt = ?, description = ?,
			temperature = ?, top_p = ?, top_k = ?, repeat_penalty = ?,
			num_predict = ?, num_ctx = ?, ollama_digest = ?, parent_model_id = ?,
			version = ?, modelfile = ?, size = ?, updated_at = ?
		WHERE id = ?
	`, model.Name, model.BaseModel, model.SystemPrompt, model.Description,
		model.Temperature, model.TopP, model.TopK, model.RepeatPenalty,
		model.NumPredict, model.NumCtx, model.OllamaDigest, model.ParentModelID,
		model.Version, model.Modelfile, model.Size, model.UpdatedAt, model.ID)

	return err
}

// Delete löscht ein Custom Model
func (r *Repository) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM custom_models WHERE id = ?", id)
	return err
}

// Count gibt die Anzahl der Custom Models zurück
func (r *Repository) Count() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM custom_models").Scan(&count)
	return count, err
}

// ========== GGUF Model Config Methods ==========

// CreateGgufConfig erstellt eine neue GGUF-Konfiguration
func (r *Repository) CreateGgufConfig(config *GgufModelConfig) error {
	now := time.Now()
	config.CreatedAt = now
	config.UpdatedAt = now

	result, err := r.db.Exec(`
		INSERT INTO gguf_model_configs (
			name, base_model, description, system_prompt, temperature, top_p, top_k,
			repeat_penalty, max_tokens, context_size, gpu_layers, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, config.Name, config.BaseModel, config.Description, config.SystemPrompt,
		config.Temperature, config.TopP, config.TopK, config.RepeatPenalty,
		config.MaxTokens, config.ContextSize, config.GpuLayers, now, now)

	if err != nil {
		return fmt.Errorf("GGUF-Config erstellen: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	config.ID = id
	return nil
}

// GetGgufConfigByID holt eine GGUF-Konfiguration nach ID
func (r *Repository) GetGgufConfigByID(id int64) (*GgufModelConfig, error) {
	config := &GgufModelConfig{}

	err := r.db.QueryRow(`
		SELECT id, name, base_model, description, system_prompt, temperature, top_p, top_k,
			repeat_penalty, max_tokens, context_size, gpu_layers, created_at, updated_at
		FROM gguf_model_configs WHERE id = ?
	`, id).Scan(&config.ID, &config.Name, &config.BaseModel, &config.Description,
		&config.SystemPrompt, &config.Temperature, &config.TopP, &config.TopK,
		&config.RepeatPenalty, &config.MaxTokens, &config.ContextSize, &config.GpuLayers,
		&config.CreatedAt, &config.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return config, nil
}

// GetGgufConfigByName holt eine GGUF-Konfiguration nach Name
func (r *Repository) GetGgufConfigByName(name string) (*GgufModelConfig, error) {
	config := &GgufModelConfig{}

	err := r.db.QueryRow(`
		SELECT id, name, base_model, description, system_prompt, temperature, top_p, top_k,
			repeat_penalty, max_tokens, context_size, gpu_layers, created_at, updated_at
		FROM gguf_model_configs WHERE name = ?
	`, name).Scan(&config.ID, &config.Name, &config.BaseModel, &config.Description,
		&config.SystemPrompt, &config.Temperature, &config.TopP, &config.TopK,
		&config.RepeatPenalty, &config.MaxTokens, &config.ContextSize, &config.GpuLayers,
		&config.CreatedAt, &config.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return config, nil
}

// GetAllGgufConfigs holt alle GGUF-Konfigurationen
func (r *Repository) GetAllGgufConfigs() ([]GgufModelConfig, error) {
	rows, err := r.db.Query(`
		SELECT id, name, base_model, description, system_prompt, temperature, top_p, top_k,
			repeat_penalty, max_tokens, context_size, gpu_layers, created_at, updated_at
		FROM gguf_model_configs
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []GgufModelConfig
	for rows.Next() {
		var c GgufModelConfig
		err := rows.Scan(&c.ID, &c.Name, &c.BaseModel, &c.Description,
			&c.SystemPrompt, &c.Temperature, &c.TopP, &c.TopK,
			&c.RepeatPenalty, &c.MaxTokens, &c.ContextSize, &c.GpuLayers,
			&c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}

	if configs == nil {
		configs = []GgufModelConfig{}
	}

	return configs, nil
}

// UpdateGgufConfig aktualisiert eine GGUF-Konfiguration
func (r *Repository) UpdateGgufConfig(config *GgufModelConfig) error {
	config.UpdatedAt = time.Now()

	_, err := r.db.Exec(`
		UPDATE gguf_model_configs SET
			name = ?, base_model = ?, description = ?, system_prompt = ?,
			temperature = ?, top_p = ?, top_k = ?, repeat_penalty = ?,
			max_tokens = ?, context_size = ?, gpu_layers = ?, updated_at = ?
		WHERE id = ?
	`, config.Name, config.BaseModel, config.Description, config.SystemPrompt,
		config.Temperature, config.TopP, config.TopK, config.RepeatPenalty,
		config.MaxTokens, config.ContextSize, config.GpuLayers, config.UpdatedAt, config.ID)

	return err
}

// DeleteGgufConfig löscht eine GGUF-Konfiguration
func (r *Repository) DeleteGgufConfig(id int64) error {
	_, err := r.db.Exec("DELETE FROM gguf_model_configs WHERE id = ?", id)
	return err
}
