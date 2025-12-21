// Package chat implementiert die Chat-Persistenz für Fleet Navigator.
//
// Dieses Paket verwaltet die SQLite-Datenbank für Chat-Konversationen:
//   - Chats: Konversations-Container mit Titel und Modell
//   - Messages: Einzelne Nachrichten mit Rolle (USER/ASSISTANT)
//   - Expert/Mode-Zuordnung: Fixe Verknüpfung pro Nachricht
//
// Datenbank: SQLite mit WAL-Modus für bessere Concurrent-Performance
// Erstellt: 2025-12-15
package chat

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	// SQLite-Treiber (pure Go, keine CGO-Abhängigkeit)
	_ "modernc.org/sqlite"
)

// =============================================================================
// DATENMODELLE
// =============================================================================

// Chat repräsentiert eine Chat-Konversation.
// Ein Chat enthält Metadaten und eine Liste von Nachrichten.
type Chat struct {
	// ID: Eindeutige Kennung des Chats (auto-increment)
	ID int64 `json:"id"`

	// Title: Anzeigename des Chats (wird aus erster User-Nachricht generiert)
	Title string `json:"title"`

	// Model: Das verwendete LLM-Modell (z.B. "gpt-4", "claude-3")
	Model string `json:"model"`

	// CreatedAt: Erstellungszeitpunkt des Chats
	CreatedAt time.Time `json:"createdAt"`

	// UpdatedAt: Zeitpunkt der letzten Änderung (neue Nachricht, Umbenennung, etc.)
	UpdatedAt time.Time `json:"updatedAt"`

	// Messages: Liste aller Nachrichten in diesem Chat (chronologisch sortiert)
	// Wird nur bei GetChat() geladen, nicht bei GetAllChats()
	Messages []StoredMessage `json:"messages,omitempty"`
}

// StoredMessage repräsentiert eine einzelne Chat-Nachricht.
// Jede Nachricht hat eine fixe, unveränderliche Zuordnung zu Expert und Modus.
type StoredMessage struct {
	// ID: Eindeutige Kennung der Nachricht (auto-increment)
	ID int64 `json:"id"`

	// ChatID: Fremdschlüssel zum übergeordneten Chat
	ChatID int64 `json:"chatId"`

	// Role: Absender der Nachricht
	// - "USER": Nachricht vom Benutzer
	// - "ASSISTANT": Antwort vom KI-Assistenten
	Role string `json:"role"`

	// Content: Der eigentliche Nachrichtentext (kann Markdown enthalten)
	Content string `json:"content"`

	// Tokens: Anzahl der verbrauchten Tokens (für Statistik/Abrechnung)
	Tokens int `json:"tokens,omitempty"`

	// Model: Das verwendete Modell für diese spezifische Nachricht
	// Kann vom Chat-Modell abweichen (bei Modell-Wechsel)
	Model string `json:"model,omitempty"`

	// ExpertID: Fixe Zuordnung zu einem Experten-Profil
	// NULL bedeutet: Kein spezifischer Experte (Standard-Assistent)
	// WICHTIG: Diese Zuordnung ist UNVERÄNDERLICH nach Erstellung!
	ExpertID *int64 `json:"expertId,omitempty"`

	// ModeID: Fixe Zuordnung zu einem Antwort-Modus (z.B. "Kreativ", "Präzise")
	// NULL bedeutet: Standard-Modus
	// WICHTIG: Diese Zuordnung ist UNVERÄNDERLICH nach Erstellung!
	ModeID *int64 `json:"modeId,omitempty"`

	// CreatedAt: Erstellungszeitpunkt der Nachricht
	CreatedAt time.Time `json:"createdAt"`
}

// =============================================================================
// STORE (Datenbankzugriff)
// =============================================================================

// Store ist der zentrale Datenbankzugriff für Chat-Operationen.
// Thread-safe durch SQLite's interne Synchronisation und Connection-Pooling.
type Store struct {
	db *sql.DB // Datenbank-Connection-Pool
}

// NewStore erstellt einen neuen Chat-Store und initialisiert die Datenbank.
//
// Parameter:
//   - dataDir: Verzeichnis für die Datenbankdatei (z.B. ~/.fleet-navigator)
//
// Rückgabe:
//   - *Store: Der initialisierte Store
//   - error: Fehler bei DB-Öffnung oder Schema-Erstellung
func NewStore(dataDir string) (*Store, error) {
	// Datenbankpfad konstruieren
	dbPath := filepath.Join(dataDir, "chats.db")

	// SQLite-Datenbank öffnen (wird erstellt falls nicht vorhanden)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("Chat-DB öffnen fehlgeschlagen: %w", err)
	}

	// -------------------------------------------------------------------------
	// SECURITY: Foreign Key Constraints aktivieren
	// SQLite hat Foreign Keys standardmäßig DEAKTIVIERT!
	// Ohne diese Einstellung würden ON DELETE CASCADE nicht funktionieren.
	// -------------------------------------------------------------------------
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		log.Printf("WARNUNG: Foreign Keys konnten nicht aktiviert werden: %v", err)
	}

	// -------------------------------------------------------------------------
	// PERFORMANCE: SQLite-Optimierungen
	// -------------------------------------------------------------------------

	// WAL-Modus: Write-Ahead-Logging für bessere Concurrent-Performance
	// Ermöglicht gleichzeitiges Lesen während des Schreibens
	db.Exec("PRAGMA journal_mode=WAL")

	// Busy-Timeout: Wartezeit bei gesperrter Datenbank (5 Sekunden)
	// Verhindert "database is locked" Fehler bei parallelen Zugriffen
	db.Exec("PRAGMA busy_timeout=5000")

	// Connection-Pool Einstellungen
	db.SetMaxOpenConns(5) // Max 5 gleichzeitige Verbindungen
	db.SetMaxIdleConns(2) // 2 Verbindungen im Pool halten

	// Store erstellen und Schema initialisieren
	store := &Store{db: db}

	if err := store.createSchema(); err != nil {
		return nil, err
	}

	return store, nil
}

// createSchema erstellt das Datenbankschema (Tabellen und Indizes).
// Verwendet CREATE IF NOT EXISTS - sicher für wiederholte Aufrufe.
func (s *Store) createSchema() error {
	schema := `
	-- Tabelle: chats
	-- Speichert Chat-Metadaten (Konversations-Container)
	CREATE TABLE IF NOT EXISTS chats (
		id INTEGER PRIMARY KEY AUTOINCREMENT,  -- Eindeutige Chat-ID
		title TEXT NOT NULL,                   -- Anzeigename
		model TEXT DEFAULT '',                 -- Standard-Modell
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Tabelle: messages
	-- Speichert einzelne Nachrichten mit fixer Expert/Mode-Zuordnung
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,  -- Eindeutige Nachrichten-ID
		chat_id INTEGER NOT NULL,              -- Fremdschlüssel zu chats
		role TEXT NOT NULL,                    -- USER oder ASSISTANT
		content TEXT NOT NULL,                 -- Nachrichteninhalt
		tokens INTEGER DEFAULT 0,              -- Token-Verbrauch
		model TEXT DEFAULT '',                 -- Verwendetes Modell
		expert_id INTEGER DEFAULT NULL,        -- Fixer Experte (unveränderlich!)
		mode_id INTEGER DEFAULT NULL,          -- Fixer Modus (unveränderlich!)
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE
	);

	-- Index für schnelle Nachrichten-Abfragen pro Chat
	CREATE INDEX IF NOT EXISTS idx_messages_chat_id ON messages(chat_id);
	`

	_, err := s.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("Chat-Schema erstellen fehlgeschlagen: %w", err)
	}

	// Migration für bestehende Datenbanken ausführen
	s.migrateSchema()

	return nil
}

// migrateSchema führt Migrationen für bestehende Datenbanken durch.
// Fügt neue Spalten hinzu ohne bestehende Daten zu verlieren.
//
// Aktuelle Migrationen:
//   - expert_id: Hinzugefügt 2025-12-15 für fixe Expert-Zuordnung
//   - mode_id: Hinzugefügt 2025-12-15 für fixe Modus-Zuordnung
func (s *Store) migrateSchema() {
	// -------------------------------------------------------------------------
	// Migration 1: expert_id Spalte
	// Ermöglicht fixe Zuordnung einer Nachricht zu einem Experten-Profil
	// -------------------------------------------------------------------------
	_, err := s.db.Exec(`ALTER TABLE messages ADD COLUMN expert_id INTEGER DEFAULT NULL`)
	if err != nil {
		// "duplicate column" ist OK - bedeutet Spalte existiert bereits
		if !strings.Contains(err.Error(), "duplicate column") {
			log.Printf("Migration expert_id fehlgeschlagen: %v", err)
		}
	} else {
		log.Printf("Migration: expert_id Spalte zu messages hinzugefügt")
	}

	// -------------------------------------------------------------------------
	// Migration 2: mode_id Spalte
	// Ermöglicht fixe Zuordnung einer Nachricht zu einem Antwort-Modus
	// -------------------------------------------------------------------------
	_, err = s.db.Exec(`ALTER TABLE messages ADD COLUMN mode_id INTEGER DEFAULT NULL`)
	if err != nil {
		if !strings.Contains(err.Error(), "duplicate column") {
			log.Printf("Migration mode_id fehlgeschlagen: %v", err)
		}
	} else {
		log.Printf("Migration: mode_id Spalte zu messages hinzugefügt")
	}
}

// Close schließt die Datenbankverbindung.
// Sollte beim Beenden der Anwendung aufgerufen werden.
func (s *Store) Close() error {
	return s.db.Close()
}

// =============================================================================
// CHAT CRUD-OPERATIONEN
// =============================================================================

// CreateChat erstellt einen neuen leeren Chat.
//
// Parameter:
//   - title: Anzeigename des Chats
//   - model: Standard-Modell für diesen Chat
//
// Rückgabe:
//   - *Chat: Der neu erstellte Chat mit generierter ID
//   - error: Datenbankfehler
func (s *Store) CreateChat(title, model string) (*Chat, error) {
	now := time.Now()

	result, err := s.db.Exec(`
		INSERT INTO chats (title, model, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`, title, model, now, now)

	if err != nil {
		return nil, fmt.Errorf("Chat erstellen fehlgeschlagen: %w", err)
	}

	// Auto-generierte ID abrufen
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Chat{
		ID:        id,
		Title:     title,
		Model:     model,
		CreatedAt: now,
		UpdatedAt: now,
		Messages:  []StoredMessage{}, // Leere Nachrichtenliste
	}, nil
}

// GetChat lädt einen Chat mit allen zugehörigen Nachrichten.
//
// Parameter:
//   - id: Die Chat-ID
//
// Rückgabe:
//   - *Chat: Der Chat mit Messages (nil wenn nicht gefunden)
//   - error: Datenbankfehler
func (s *Store) GetChat(id int64) (*Chat, error) {
	chat := &Chat{}

	// Chat-Metadaten laden
	err := s.db.QueryRow(`
		SELECT id, title, model, created_at, updated_at
		FROM chats WHERE id = ?
	`, id).Scan(&chat.ID, &chat.Title, &chat.Model, &chat.CreatedAt, &chat.UpdatedAt)

	// Chat nicht gefunden ist kein Fehler, gibt nil zurück
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Alle Nachrichten des Chats laden
	messages, err := s.GetMessages(id)
	if err != nil {
		return nil, err
	}
	chat.Messages = messages

	return chat, nil
}

// GetAllChats lädt alle Chats (ohne Nachrichten).
// Sortiert nach letzter Aktualisierung (neueste zuerst).
//
// Rückgabe:
//   - []Chat: Liste aller Chats (ohne Messages)
//   - error: Datenbankfehler
func (s *Store) GetAllChats() ([]Chat, error) {
	rows, err := s.db.Query(`
		SELECT id, title, model, created_at, updated_at
		FROM chats
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close() // Wichtig: Cursor schließen nach Verwendung

	// Slice mit initialer Kapazität 0 (wächst bei Bedarf)
	chats := make([]Chat, 0)
	for rows.Next() {
		var c Chat
		err := rows.Scan(&c.ID, &c.Title, &c.Model, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, err
		}
		chats = append(chats, c)
	}

	return chats, nil
}

// UpdateChat aktualisiert Chat-Metadaten (Titel und/oder Modell).
// Nur übergebene Werte werden geändert (nil = keine Änderung).
//
// Parameter:
//   - id: Die Chat-ID
//   - title: Neuer Titel (nil = unverändert)
//   - model: Neues Modell (nil = unverändert)
func (s *Store) UpdateChat(id int64, title, model *string) error {
	now := time.Now()

	// Titel aktualisieren wenn angegeben
	if title != nil {
		_, err := s.db.Exec(`UPDATE chats SET title = ?, updated_at = ? WHERE id = ?`, *title, now, id)
		if err != nil {
			return err
		}
	}

	// Modell aktualisieren wenn angegeben
	if model != nil {
		_, err := s.db.Exec(`UPDATE chats SET model = ?, updated_at = ? WHERE id = ?`, *model, now, id)
		if err != nil {
			return err
		}
	}

	return nil
}

// DeleteChat löscht einen Chat und alle zugehörigen Nachrichten.
// Nachrichten werden durch ON DELETE CASCADE automatisch mitgelöscht.
//
// Parameter:
//   - id: Die Chat-ID
func (s *Store) DeleteChat(id int64) error {
	_, err := s.db.Exec("DELETE FROM chats WHERE id = ?", id)
	return err
}

// =============================================================================
// NACHRICHTEN-OPERATIONEN
// =============================================================================

// AddMessage fügt eine neue Nachricht zu einem Chat hinzu.
// Die Expert- und Modus-Zuordnung ist FIX und kann später nicht geändert werden!
//
// Parameter:
//   - chatID: Der Chat zu dem die Nachricht gehört
//   - role: "USER" oder "ASSISTANT"
//   - content: Der Nachrichteninhalt
//   - model: Das verwendete LLM-Modell
//   - tokens: Anzahl verbrauchter Tokens
//   - expertID: Fixe Expert-Zuordnung (nil = kein Experte)
//   - modeID: Fixe Modus-Zuordnung (nil = Standard-Modus)
//
// Rückgabe:
//   - *StoredMessage: Die gespeicherte Nachricht mit generierter ID
//   - error: Datenbankfehler
func (s *Store) AddMessage(chatID int64, role, content, model string, tokens int, expertID, modeID *int64) (*StoredMessage, error) {
	now := time.Now()

	// Nachricht in Datenbank einfügen
	result, err := s.db.Exec(`
		INSERT INTO messages (chat_id, role, content, tokens, model, expert_id, mode_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, chatID, role, content, tokens, model, expertID, modeID, now)

	if err != nil {
		return nil, fmt.Errorf("Message hinzufügen fehlgeschlagen: %w", err)
	}

	// Chat-Timestamp aktualisieren (neue Nachricht = neue Aktivität)
	// Fehler nur loggen, nicht abbrechen - Nachricht wurde bereits gespeichert
	if _, err := s.db.Exec(`UPDATE chats SET updated_at = ? WHERE id = ?`, now, chatID); err != nil {
		log.Printf("WARNUNG: Chat-Timestamp konnte nicht aktualisiert werden: %v", err)
	}

	// Auto-generierte Nachrichten-ID abrufen
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &StoredMessage{
		ID:        id,
		ChatID:    chatID,
		Role:      role,
		Content:   content,
		Tokens:    tokens,
		Model:     model,
		ExpertID:  expertID,
		ModeID:    modeID,
		CreatedAt: now,
	}, nil
}

// GetMessages lädt alle Nachrichten eines Chats.
// Sortiert chronologisch (älteste zuerst).
//
// Parameter:
//   - chatID: Die Chat-ID
//
// Rückgabe:
//   - []StoredMessage: Alle Nachrichten des Chats (leeres Array wenn keine)
//   - error: Datenbankfehler
func (s *Store) GetMessages(chatID int64) ([]StoredMessage, error) {
	rows, err := s.db.Query(`
		SELECT id, chat_id, role, content, tokens, model, expert_id, mode_id, created_at
		FROM messages
		WHERE chat_id = ?
		ORDER BY created_at ASC
	`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []StoredMessage
	for rows.Next() {
		var m StoredMessage
		// Alle Felder scannen inkl. nullable expert_id und mode_id
		err := rows.Scan(&m.ID, &m.ChatID, &m.Role, &m.Content, &m.Tokens, &m.Model, &m.ExpertID, &m.ModeID, &m.CreatedAt)
		if err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}

	// Leeres Slice statt nil zurückgeben (für JSON-Serialisierung)
	if messages == nil {
		messages = []StoredMessage{}
	}

	return messages, nil
}

// UpdateChatTimestamp aktualisiert nur den updated_at Timestamp eines Chats.
// Nützlich für Operationen die den Chat "berühren" ohne Inhalte zu ändern.
func (s *Store) UpdateChatTimestamp(chatID int64) error {
	_, err := s.db.Exec(`UPDATE chats SET updated_at = ? WHERE id = ?`, time.Now(), chatID)
	return err
}

// RenameChat ändert den Titel eines Chats.
//
// Parameter:
//   - id: Die Chat-ID
//   - newTitle: Der neue Titel
//
// Rückgabe:
//   - *Chat: Der aktualisierte Chat mit allen Nachrichten
//   - error: Datenbankfehler
func (s *Store) RenameChat(id int64, newTitle string) (*Chat, error) {
	now := time.Now()
	_, err := s.db.Exec(`UPDATE chats SET title = ?, updated_at = ? WHERE id = ?`, newTitle, now, id)
	if err != nil {
		return nil, fmt.Errorf("Chat umbenennen fehlgeschlagen: %w", err)
	}
	// Vollständigen Chat mit neuen Daten zurückgeben
	return s.GetChat(id)
}

// UpdateChatExpert ist DEPRECATED und sollte nicht mehr verwendet werden!
//
// HINTERGRUND:
// Früher wurde der Experte auf Chat-Ebene gespeichert.
// Seit 2025-12-15 wird der Experte auf Message-Ebene gespeichert,
// damit jede Nachricht eine FIXE, UNVERÄNDERLICHE Zuordnung hat.
//
// Diese Funktion existiert nur noch für API-Abwärtskompatibilität
// und aktualisiert nur den Chat-Timestamp.
//
// VERWENDE STATTDESSEN: AddMessage() mit expertID und modeID Parametern
//
// Deprecated: Seit 2025-12-15
func (s *Store) UpdateChatExpert(id int64, expertID *int64) error {
	// Nur Timestamp aktualisieren - Expert-Zuordnung erfolgt jetzt pro Message
	_, err := s.db.Exec(`UPDATE chats SET updated_at = ? WHERE id = ?`, time.Now(), id)
	return err
}

// DeleteMessage löscht eine einzelne Nachricht aus einem Chat.
//
// Parameter:
//   - chatID: Die Chat-ID (für Validierung)
//   - messageID: Die zu löschende Nachrichten-ID
//
// Rückgabe:
//   - error: Fehler wenn Nachricht nicht gefunden oder DB-Fehler
func (s *Store) DeleteMessage(chatID, messageID int64) error {
	// Nachricht nur löschen wenn sie zum angegebenen Chat gehört (Sicherheit)
	result, err := s.db.Exec(`DELETE FROM messages WHERE id = ? AND chat_id = ?`, messageID, chatID)
	if err != nil {
		return fmt.Errorf("Nachricht löschen fehlgeschlagen: %w", err)
	}

	// Prüfen ob eine Zeile gelöscht wurde
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("Nachricht nicht gefunden")
	}

	// Chat-Timestamp aktualisieren
	s.UpdateChatTimestamp(chatID)
	return nil
}

// =============================================================================
// ERWEITERTE OPERATIONEN
// =============================================================================

// ForkChat erstellt eine vollständige Kopie eines Chats.
// Alle Nachrichten werden kopiert, inkl. Expert- und Modus-Zuordnungen.
// Nützlich für "Was wäre wenn"-Szenarien oder Versionierung.
//
// Parameter:
//   - originalID: Die ID des zu kopierenden Chats
//   - newTitle: Titel für den neuen Chat
//
// Rückgabe:
//   - *Chat: Der neu erstellte Chat mit kopierten Nachrichten
//   - error: Fehler wenn Original nicht gefunden oder DB-Fehler
func (s *Store) ForkChat(originalID int64, newTitle string) (*Chat, error) {
	// Original-Chat mit allen Nachrichten laden
	original, err := s.GetChat(originalID)
	if err != nil {
		return nil, fmt.Errorf("Original-Chat laden fehlgeschlagen: %w", err)
	}
	if original == nil {
		return nil, fmt.Errorf("Original-Chat nicht gefunden")
	}

	// Neuen leeren Chat mit gleichem Modell erstellen
	forkedChat, err := s.CreateChat(newTitle, original.Model)
	if err != nil {
		return nil, fmt.Errorf("Fork erstellen fehlgeschlagen: %w", err)
	}

	// Alle Nachrichten vom Original in den Fork kopieren
	// WICHTIG: ExpertID und ModeID werden mitkopiert (fixe Zuordnung bleibt erhalten)
	for _, msg := range original.Messages {
		_, err := s.AddMessage(forkedChat.ID, msg.Role, msg.Content, msg.Model, msg.Tokens, msg.ExpertID, msg.ModeID)
		if err != nil {
			// Bei Fehler: Fork-Chat wieder löschen (Transaktion simulieren)
			s.DeleteChat(forkedChat.ID)
			return nil, fmt.Errorf("Nachrichten kopieren fehlgeschlagen: %w", err)
		}
	}

	// Vollständigen Fork mit allen Nachrichten zurückgeben
	return s.GetChat(forkedChat.ID)
}

// ExportChat exportiert einen Chat in ein strukturiertes Format.
// Geeignet für JSON-Export, Backup oder Import in andere Systeme.
//
// Das Export-Format enthält:
//   - Chat-Metadaten (id, title, model, timestamps)
//   - Alle Nachrichten mit allen Feldern inkl. expertId und modeId
//
// Parameter:
//   - id: Die Chat-ID
//
// Rückgabe:
//   - map[string]interface{}: Export-Objekt (JSON-kompatibel)
//   - error: Fehler wenn Chat nicht gefunden
func (s *Store) ExportChat(id int64) (map[string]interface{}, error) {
	// Chat mit allen Nachrichten laden
	chatObj, err := s.GetChat(id)
	if err != nil {
		return nil, err
	}
	if chatObj == nil {
		return nil, fmt.Errorf("Chat nicht gefunden")
	}

	// Export-Objekt aufbauen
	export := map[string]interface{}{
		"id":        chatObj.ID,
		"title":     chatObj.Title,
		"model":     chatObj.Model,
		"createdAt": chatObj.CreatedAt.Format(time.RFC3339), // ISO 8601 Format
		"updatedAt": chatObj.UpdatedAt.Format(time.RFC3339),
		"messages":  make([]map[string]interface{}, 0, len(chatObj.Messages)),
	}

	// Nachrichten in Export-Format konvertieren
	messages := make([]map[string]interface{}, 0, len(chatObj.Messages))
	for _, msg := range chatObj.Messages {
		msgExport := map[string]interface{}{
			"id":        msg.ID,
			"role":      msg.Role,
			"content":   msg.Content,
			"tokens":    msg.Tokens,
			"model":     msg.Model,
			"createdAt": msg.CreatedAt.Format(time.RFC3339),
		}
		// Optionale Felder nur hinzufügen wenn gesetzt (nicht null)
		if msg.ExpertID != nil {
			msgExport["expertId"] = *msg.ExpertID
		}
		if msg.ModeID != nil {
			msgExport["modeId"] = *msg.ModeID
		}
		messages = append(messages, msgExport)
	}
	export["messages"] = messages

	return export, nil
}
