package experte

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// Repository ist das Daten-Repository für Experten
type Repository struct {
	db *sql.DB
}

// NewRepository erstellt ein neues Repository
func NewRepository(dataDir string) (*Repository, error) {
	dbPath := filepath.Join(dataDir, "experts.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("Datenbank öffnen fehlgeschlagen: %w", err)
	}

	// SQLite braucht min 2 Connections für verschachtelte Queries
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)

	repo := &Repository{db: db}

	if err := repo.createSchema(); err != nil {
		return nil, err
	}

	return repo, nil
}

// createSchema erstellt die Datenbank-Tabellen
func (r *Repository) createSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS experts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		role TEXT DEFAULT '',
		base_prompt TEXT DEFAULT '',
		base_model TEXT DEFAULT 'qwen2.5:7b',
		avatar TEXT DEFAULT '🤖',
		description TEXT DEFAULT '',
		is_active INTEGER DEFAULT 1,
		auto_mode_switch INTEGER DEFAULT 0,
		sort_order INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS expert_modes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		expert_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		prompt TEXT DEFAULT '',
		icon TEXT DEFAULT '💬',
		keywords TEXT DEFAULT '[]',
		is_default INTEGER DEFAULT 0,
		sort_order INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (expert_id) REFERENCES experts(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_expert_modes_expert_id ON expert_modes(expert_id);
	CREATE INDEX IF NOT EXISTS idx_experts_is_active ON experts(is_active);
	`

	_, err := r.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("Schema erstellen fehlgeschlagen: %w", err)
	}

	// Migration: keywords Spalte hinzufügen falls nicht vorhanden
	r.migrateKeywords()

	return nil
}

// migrateKeywords fügt die keywords Spalte hinzu falls sie fehlt
func (r *Repository) migrateKeywords() {
	// Keywords Spalte für expert_modes
	var count int
	r.db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('expert_modes') WHERE name='keywords'
	`).Scan(&count)

	if count == 0 {
		r.db.Exec(`ALTER TABLE expert_modes ADD COLUMN keywords TEXT DEFAULT '[]'`)
	}

	// auto_mode_switch Spalte für experts
	r.db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('experts') WHERE name='auto_mode_switch'
	`).Scan(&count)

	if count == 0 {
		r.db.Exec(`ALTER TABLE experts ADD COLUMN auto_mode_switch INTEGER DEFAULT 0`)
	}

	// voice Spalte für experts (TTS Stimme)
	r.db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('experts') WHERE name='voice'
	`).Scan(&count)

	if count == 0 {
		r.db.Exec(`ALTER TABLE experts ADD COLUMN voice TEXT DEFAULT ''`)
	}

	// Sampling Parameter Spalten für experts
	r.db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('experts') WHERE name='default_num_ctx'
	`).Scan(&count)

	if count == 0 {
		r.db.Exec(`ALTER TABLE experts ADD COLUMN default_num_ctx INTEGER DEFAULT 16384`)
		r.db.Exec(`ALTER TABLE experts ADD COLUMN default_max_tokens INTEGER DEFAULT 4096`)
		r.db.Exec(`ALTER TABLE experts ADD COLUMN default_temperature REAL DEFAULT 0.7`)
		r.db.Exec(`ALTER TABLE experts ADD COLUMN default_top_p REAL DEFAULT 0.9`)
	}

	// Web Search Spalten für experts
	r.db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('experts') WHERE name='auto_web_search'
	`).Scan(&count)

	if count == 0 {
		r.db.Exec(`ALTER TABLE experts ADD COLUMN auto_web_search INTEGER DEFAULT 0`)
		r.db.Exec(`ALTER TABLE experts ADD COLUMN web_search_show_links INTEGER DEFAULT 0`) // Default: RAG-Modus (keine Links)
	}

	// personality_prompt Spalte für experts (Kommunikationsstil)
	r.db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('experts') WHERE name='personality_prompt'
	`).Scan(&count)

	if count == 0 {
		r.db.Exec(`ALTER TABLE experts ADD COLUMN personality_prompt TEXT DEFAULT ''`)
	}

	// anti_hallucination_prompt Spalte für experts (Custom Anti-Halluzinations-Regeln)
	r.db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('experts') WHERE name='anti_hallucination_prompt'
	`).Scan(&count)

	if count == 0 {
		r.db.Exec(`ALTER TABLE experts ADD COLUMN anti_hallucination_prompt TEXT DEFAULT ''`)
	}
}

// Close schließt die Datenbankverbindung
func (r *Repository) Close() error {
	return r.db.Close()
}

// --- Expert CRUD ---

// CreateExpert erstellt einen neuen Experten
func (r *Repository) CreateExpert(expert *Expert) error {
	now := time.Now()

	// Defaults für Sampling Parameter setzen
	if expert.DefaultNumCtx == 0 {
		expert.DefaultNumCtx = 16384 // 16K Default (65K braucht zu viel VRAM)
	}
	if expert.DefaultMaxTokens == 0 {
		expert.DefaultMaxTokens = 4096
	}
	if expert.DefaultTemperature == 0 {
		expert.DefaultTemperature = 0.7
	}
	if expert.DefaultTopP == 0 {
		expert.DefaultTopP = 0.9
	}
	// Default für WebSearchShowLinks ist true (Links anzeigen)
	// Hier nichts setzen, da bool default false ist und wir true als DB-Default haben

	result, err := r.db.Exec(`
		INSERT INTO experts (name, role, base_prompt, personality_prompt, base_model, avatar, description, voice, is_active, auto_mode_switch, sort_order,
			default_num_ctx, default_max_tokens, default_temperature, default_top_p,
			auto_web_search, web_search_show_links, anti_hallucination_prompt, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, expert.Name, expert.Role, expert.BasePrompt, expert.PersonalityPrompt, expert.BaseModel, expert.Avatar, expert.Description, expert.Voice,
		expert.IsActive, expert.AutoModeSwitch, expert.SortOrder,
		expert.DefaultNumCtx, expert.DefaultMaxTokens, expert.DefaultTemperature, expert.DefaultTopP,
		expert.AutoWebSearch, expert.WebSearchShowLinks, expert.AntiHallucinationPrompt, now, now)

	if err != nil {
		return fmt.Errorf("Experte erstellen fehlgeschlagen: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	expert.ID = id
	expert.CreatedAt = now
	expert.UpdatedAt = now

	// Modi erstellen falls vorhanden
	for i := range expert.Modes {
		expert.Modes[i].ExpertID = id
		if err := r.CreateMode(&expert.Modes[i]); err != nil {
			return err
		}
	}

	return nil
}

// GetExpert holt einen Experten mit Modi
func (r *Repository) GetExpert(id int64) (*Expert, error) {
	expert := &Expert{}

	err := r.db.QueryRow(`
		SELECT id, name, role, base_prompt, COALESCE(personality_prompt, ''), base_model, avatar, description, voice, is_active, auto_mode_switch, sort_order,
			COALESCE(default_num_ctx, 16384), COALESCE(default_max_tokens, 4096), COALESCE(default_temperature, 0.7), COALESCE(default_top_p, 0.9),
			COALESCE(auto_web_search, 0), COALESCE(web_search_show_links, 0), COALESCE(anti_hallucination_prompt, ''),
			created_at, updated_at
		FROM experts WHERE id = ?
	`, id).Scan(&expert.ID, &expert.Name, &expert.Role, &expert.BasePrompt, &expert.PersonalityPrompt, &expert.BaseModel,
		&expert.Avatar, &expert.Description, &expert.Voice, &expert.IsActive, &expert.AutoModeSwitch, &expert.SortOrder,
		&expert.DefaultNumCtx, &expert.DefaultMaxTokens, &expert.DefaultTemperature, &expert.DefaultTopP,
		&expert.AutoWebSearch, &expert.WebSearchShowLinks, &expert.AntiHallucinationPrompt,
		&expert.CreatedAt, &expert.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Modi laden
	modes, err := r.GetModesByExpert(id)
	if err != nil {
		return nil, err
	}
	expert.Modes = modes

	return expert, nil
}

// GetAllExperts holt alle Experten
func (r *Repository) GetAllExperts(onlyActive bool) ([]Expert, error) {
	query := `SELECT id, name, role, base_prompt, COALESCE(personality_prompt, ''), base_model, avatar, description, voice, is_active, auto_mode_switch, sort_order,
		COALESCE(default_num_ctx, 16384), COALESCE(default_max_tokens, 4096), COALESCE(default_temperature, 0.7), COALESCE(default_top_p, 0.9),
		COALESCE(auto_web_search, 0), COALESCE(web_search_show_links, 0), COALESCE(anti_hallucination_prompt, ''),
		created_at, updated_at FROM experts`
	if onlyActive {
		query += " WHERE is_active = 1"
	}
	query += " ORDER BY sort_order ASC, name ASC"

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	experts := make([]Expert, 0) // Immer leeres Array, nie null
	for rows.Next() {
		var e Expert
		err := rows.Scan(&e.ID, &e.Name, &e.Role, &e.BasePrompt, &e.PersonalityPrompt, &e.BaseModel,
			&e.Avatar, &e.Description, &e.Voice, &e.IsActive, &e.AutoModeSwitch, &e.SortOrder,
			&e.DefaultNumCtx, &e.DefaultMaxTokens, &e.DefaultTemperature, &e.DefaultTopP,
			&e.AutoWebSearch, &e.WebSearchShowLinks, &e.AntiHallucinationPrompt,
			&e.CreatedAt, &e.UpdatedAt)
		if err != nil {
			return nil, err
		}
		modes, err := r.GetModesByExpert(e.ID)
		if err != nil {
			return nil, err
		}
		e.Modes = modes
		experts = append(experts, e)
	}
	return experts, nil
}

// UpdateExpert aktualisiert einen Experten
func (r *Repository) UpdateExpert(id int64, req *UpdateExpertRequest) error {
	// Dynamisches Update nur der übergebenen Felder
	updates := []string{}
	args := []interface{}{}

	if req.Name != nil {
		updates = append(updates, "name = ?")
		args = append(args, *req.Name)
	}
	if req.Role != nil {
		updates = append(updates, "role = ?")
		args = append(args, *req.Role)
	}
	if req.BasePrompt != nil {
		updates = append(updates, "base_prompt = ?")
		args = append(args, *req.BasePrompt)
	}
	if req.PersonalityPrompt != nil {
		updates = append(updates, "personality_prompt = ?")
		args = append(args, *req.PersonalityPrompt)
	}
	if req.BaseModel != nil {
		updates = append(updates, "base_model = ?")
		args = append(args, *req.BaseModel)
	}
	if req.Avatar != nil {
		updates = append(updates, "avatar = ?")
		args = append(args, *req.Avatar)
	}
	if req.Description != nil {
		updates = append(updates, "description = ?")
		args = append(args, *req.Description)
	}
	if req.IsActive != nil {
		updates = append(updates, "is_active = ?")
		args = append(args, *req.IsActive)
	}
	if req.Voice != nil {
		updates = append(updates, "voice = ?")
		args = append(args, *req.Voice)
	}
	if req.AutoModeSwitch != nil {
		updates = append(updates, "auto_mode_switch = ?")
		args = append(args, *req.AutoModeSwitch)
	}
	if req.SortOrder != nil {
		updates = append(updates, "sort_order = ?")
		args = append(args, *req.SortOrder)
	}
	// Sampling Parameter Defaults
	if req.DefaultNumCtx != nil {
		updates = append(updates, "default_num_ctx = ?")
		args = append(args, *req.DefaultNumCtx)
	}
	if req.DefaultMaxTokens != nil {
		updates = append(updates, "default_max_tokens = ?")
		args = append(args, *req.DefaultMaxTokens)
	}
	if req.DefaultTemperature != nil {
		updates = append(updates, "default_temperature = ?")
		args = append(args, *req.DefaultTemperature)
	}
	if req.DefaultTopP != nil {
		updates = append(updates, "default_top_p = ?")
		args = append(args, *req.DefaultTopP)
	}
	// Web Search Settings
	if req.AutoWebSearch != nil {
		updates = append(updates, "auto_web_search = ?")
		args = append(args, *req.AutoWebSearch)
	}
	if req.WebSearchShowLinks != nil {
		updates = append(updates, "web_search_show_links = ?")
		args = append(args, *req.WebSearchShowLinks)
	}
	// Anti-Halluzinations-Prompt
	if req.AntiHallucinationPrompt != nil {
		updates = append(updates, "anti_hallucination_prompt = ?")
		args = append(args, *req.AntiHallucinationPrompt)
	}

	if len(updates) == 0 {
		return nil // Nichts zu aktualisieren
	}

	updates = append(updates, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, id)

	query := "UPDATE experts SET "
	for i, u := range updates {
		if i > 0 {
			query += ", "
		}
		query += u
	}
	query += " WHERE id = ?"

	_, err := r.db.Exec(query, args...)
	return err
}

// DeleteExpert löscht einen Experten (und seine Modi durch CASCADE)
func (r *Repository) DeleteExpert(id int64) error {
	_, err := r.db.Exec("DELETE FROM experts WHERE id = ?", id)
	return err
}

// --- Mode CRUD ---

// CreateMode erstellt einen neuen Modus
func (r *Repository) CreateMode(mode *ExpertMode) error {
	now := time.Now()

	// Keywords als JSON serialisieren
	keywordsJSON := "[]"
	if len(mode.Keywords) > 0 {
		if data, err := json.Marshal(mode.Keywords); err == nil {
			keywordsJSON = string(data)
		}
	}

	result, err := r.db.Exec(`
		INSERT INTO expert_modes (expert_id, name, prompt, icon, keywords, is_default, sort_order, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, mode.ExpertID, mode.Name, mode.Prompt, mode.Icon, keywordsJSON, mode.IsDefault, mode.SortOrder, now)

	if err != nil {
		return fmt.Errorf("Modus erstellen fehlgeschlagen: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	mode.ID = id
	mode.CreatedAt = now

	return nil
}

// GetModesByExpert holt alle Modi eines Experten
func (r *Repository) GetModesByExpert(expertID int64) ([]ExpertMode, error) {
	rows, err := r.db.Query(`
		SELECT id, expert_id, name, prompt, icon, keywords, is_default, sort_order, created_at
		FROM expert_modes WHERE expert_id = ?
		ORDER BY sort_order ASC, name ASC
	`, expertID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	modes := make([]ExpertMode, 0) // Immer leeres Array, nie null
	for rows.Next() {
		var m ExpertMode
		var keywordsJSON string
		err := rows.Scan(&m.ID, &m.ExpertID, &m.Name, &m.Prompt, &m.Icon, &keywordsJSON, &m.IsDefault, &m.SortOrder, &m.CreatedAt)
		if err != nil {
			return nil, err
		}
		// Keywords aus JSON deserialisieren (nie null)
		m.Keywords = make([]string, 0)
		if keywordsJSON != "" && keywordsJSON != "[]" {
			json.Unmarshal([]byte(keywordsJSON), &m.Keywords)
		}
		modes = append(modes, m)
	}

	return modes, nil
}

// GetMode holt einen einzelnen Modus
func (r *Repository) GetMode(id int64) (*ExpertMode, error) {
	mode := &ExpertMode{}
	var keywordsJSON string

	err := r.db.QueryRow(`
		SELECT id, expert_id, name, prompt, icon, keywords, is_default, sort_order, created_at
		FROM expert_modes WHERE id = ?
	`, id).Scan(&mode.ID, &mode.ExpertID, &mode.Name, &mode.Prompt, &mode.Icon, &keywordsJSON, &mode.IsDefault, &mode.SortOrder, &mode.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Keywords aus JSON deserialisieren (nie null)
	mode.Keywords = make([]string, 0)
	if keywordsJSON != "" && keywordsJSON != "[]" {
		json.Unmarshal([]byte(keywordsJSON), &mode.Keywords)
	}

	return mode, nil
}

// UpdateMode aktualisiert einen Modus
func (r *Repository) UpdateMode(id int64, name, prompt, icon string, keywords []string, isDefault bool, sortOrder int) error {
	// Keywords als JSON serialisieren
	keywordsJSON := "[]"
	if len(keywords) > 0 {
		if data, err := json.Marshal(keywords); err == nil {
			keywordsJSON = string(data)
		}
	}

	_, err := r.db.Exec(`
		UPDATE expert_modes SET name = ?, prompt = ?, icon = ?, keywords = ?, is_default = ?, sort_order = ?
		WHERE id = ?
	`, name, prompt, icon, keywordsJSON, isDefault, sortOrder, id)
	return err
}

// DeleteMode löscht einen Modus
func (r *Repository) DeleteMode(id int64) error {
	_, err := r.db.Exec("DELETE FROM expert_modes WHERE id = ?", id)
	return err
}

// SetDefaultMode setzt einen Modus als Standard (und entfernt andere Defaults)
func (r *Repository) SetDefaultMode(expertID, modeID int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	// Alle anderen Defaults entfernen
	_, err = tx.Exec("UPDATE expert_modes SET is_default = 0 WHERE expert_id = ?", expertID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Neuen Default setzen
	_, err = tx.Exec("UPDATE expert_modes SET is_default = 1 WHERE id = ?", modeID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// --- Hilfsfunktionen ---

// Count gibt die Anzahl der Experten zurück
func (r *Repository) Count() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM experts").Scan(&count)
	return count, err
}

// SeedDefaultExperts fügt fehlende Standard-Experten ein
func (r *Repository) SeedDefaultExperts() error {
	// Existierende Experten-Namen laden
	existingNames := make(map[string]bool)
	rows, err := r.db.Query("SELECT name FROM experts")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		existingNames[name] = true
	}

	// Fehlende Standard-Experten hinzufügen
	defaults := DefaultExperts()
	addedCount := 0
	for i := range defaults {
		if !existingNames[defaults[i].Name] {
			if err := r.CreateExpert(&defaults[i]); err != nil {
				return fmt.Errorf("Standard-Experte '%s' erstellen fehlgeschlagen: %w", defaults[i].Name, err)
			}
			addedCount++
		}
	}

	if addedCount > 0 {
		log.Printf("Seeding: %d fehlende Standard-Experten hinzugefügt", addedCount)
	}

	// Stimmen für existierende Experten aktualisieren (falls leer)
	if err := r.UpdateMissingVoices(); err != nil {
		log.Printf("Warnung: Stimmen-Update fehlgeschlagen: %v", err)
	}

	return nil
}

// UpdateMissingVoices aktualisiert die Voice-Felder für existierende Experten ohne Stimme
func (r *Repository) UpdateMissingVoices() error {
	defaults := DefaultExperts()
	updatedCount := 0

	for _, defaultExpert := range defaults {
		if defaultExpert.Voice == "" {
			continue
		}

		// Update nur wenn Voice leer ist
		result, err := r.db.Exec(`
			UPDATE experts
			SET voice = ?
			WHERE name = ? AND (voice IS NULL OR voice = '')
		`, defaultExpert.Voice, defaultExpert.Name)

		if err != nil {
			log.Printf("Warnung: Voice-Update für '%s' fehlgeschlagen: %v", defaultExpert.Name, err)
			continue
		}

		affected, _ := result.RowsAffected()
		if affected > 0 {
			updatedCount++
			log.Printf("Voice aktualisiert: %s → %s", defaultExpert.Name, defaultExpert.Voice)
		}
	}

	if updatedCount > 0 {
		log.Printf("Migration: %d Experten-Stimmen aktualisiert", updatedCount)
	}

	return nil
}
