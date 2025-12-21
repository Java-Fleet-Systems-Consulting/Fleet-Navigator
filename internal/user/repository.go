package user

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// Repository verwaltet User-Daten in SQLite
type Repository struct {
	db *sql.DB
}

// NewRepository erstellt ein neues Repository
func NewRepository(dataDir string) (*Repository, error) {
	dbPath := filepath.Join(dataDir, "users.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("User-DB öffnen: %w", err)
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
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		email TEXT,
		password_hash TEXT NOT NULL,
		display_name TEXT,
		role TEXT DEFAULT 'user',
		is_active INTEGER DEFAULT 1,
		last_login_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		token TEXT UNIQUE NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		ip_address TEXT,
		user_agent TEXT,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
	CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
	`

	_, err := r.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("User-Schema erstellen: %w", err)
	}

	return nil
}

// Close schließt die Datenbankverbindung
func (r *Repository) Close() error {
	return r.db.Close()
}

// --- User CRUD ---

// CreateUser erstellt einen neuen Benutzer
func (r *Repository) CreateUser(username, password, email, displayName, role string) (*User, error) {
	// Passwort hashen
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("Passwort hashen: %w", err)
	}

	if role == "" {
		role = RoleUser
	}
	if displayName == "" {
		displayName = username
	}

	now := time.Now()
	result, err := r.db.Exec(`
		INSERT INTO users (username, email, password_hash, display_name, role, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 1, ?, ?)
	`, username, email, string(hash), displayName, role, now, now)

	if err != nil {
		return nil, fmt.Errorf("User erstellen: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &User{
		ID:          id,
		Username:    username,
		Email:       email,
		DisplayName: displayName,
		Role:        role,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// GetUserByID holt einen User nach ID
func (r *Repository) GetUserByID(id int64) (*User, error) {
	user := &User{}
	var lastLogin sql.NullTime

	err := r.db.QueryRow(`
		SELECT id, username, email, password_hash, display_name, role, is_active, last_login_at, created_at, updated_at
		FROM users WHERE id = ?
	`, id).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.DisplayName,
		&user.Role, &user.IsActive, &lastLogin, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if lastLogin.Valid {
		user.LastLoginAt = &lastLogin.Time
	}

	return user, nil
}

// GetUserByUsername holt einen User nach Username
func (r *Repository) GetUserByUsername(username string) (*User, error) {
	user := &User{}
	var lastLogin sql.NullTime

	err := r.db.QueryRow(`
		SELECT id, username, email, password_hash, display_name, role, is_active, last_login_at, created_at, updated_at
		FROM users WHERE username = ?
	`, username).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.DisplayName,
		&user.Role, &user.IsActive, &lastLogin, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if lastLogin.Valid {
		user.LastLoginAt = &lastLogin.Time
	}

	return user, nil
}

// GetAllUsers holt alle Benutzer
func (r *Repository) GetAllUsers() ([]User, error) {
	rows, err := r.db.Query(`
		SELECT id, username, email, display_name, role, is_active, last_login_at, created_at, updated_at
		FROM users
		ORDER BY username
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var lastLogin sql.NullTime
		err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.DisplayName, &u.Role,
			&u.IsActive, &lastLogin, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if lastLogin.Valid {
			u.LastLoginAt = &lastLogin.Time
		}
		users = append(users, u)
	}

	if users == nil {
		users = []User{}
	}

	return users, nil
}

// UpdateUser aktualisiert einen Benutzer
func (r *Repository) UpdateUser(id int64, email, displayName *string, isActive *bool, role *string) error {
	now := time.Now()

	if email != nil {
		_, err := r.db.Exec(`UPDATE users SET email = ?, updated_at = ? WHERE id = ?`, *email, now, id)
		if err != nil {
			return err
		}
	}

	if displayName != nil {
		_, err := r.db.Exec(`UPDATE users SET display_name = ?, updated_at = ? WHERE id = ?`, *displayName, now, id)
		if err != nil {
			return err
		}
	}

	if isActive != nil {
		active := 0
		if *isActive {
			active = 1
		}
		_, err := r.db.Exec(`UPDATE users SET is_active = ?, updated_at = ? WHERE id = ?`, active, now, id)
		if err != nil {
			return err
		}
	}

	if role != nil {
		_, err := r.db.Exec(`UPDATE users SET role = ?, updated_at = ? WHERE id = ?`, *role, now, id)
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdatePassword ändert das Passwort eines Benutzers
func (r *Repository) UpdatePassword(id int64, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("Passwort hashen: %w", err)
	}

	_, err = r.db.Exec(`UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`,
		string(hash), time.Now(), id)
	return err
}

// UpdateLastLogin aktualisiert den letzten Login-Zeitpunkt
func (r *Repository) UpdateLastLogin(id int64) error {
	_, err := r.db.Exec(`UPDATE users SET last_login_at = ? WHERE id = ?`, time.Now(), id)
	return err
}

// DeleteUser löscht einen Benutzer
func (r *Repository) DeleteUser(id int64) error {
	_, err := r.db.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

// Count gibt die Anzahl der Benutzer zurück
func (r *Repository) Count() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

// --- Session Management ---

// CreateSession erstellt eine neue Sitzung
func (r *Repository) CreateSession(userID int64, duration time.Duration, ipAddress, userAgent string) (*Session, error) {
	// Zufälliges Token generieren
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("Token generieren: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	// Session-ID generieren
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, fmt.Errorf("Session-ID generieren: %w", err)
	}
	sessionID := hex.EncodeToString(idBytes)

	now := time.Now()
	expiresAt := now.Add(duration)

	_, err := r.db.Exec(`
		INSERT INTO sessions (id, user_id, token, expires_at, created_at, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, sessionID, userID, token, expiresAt, now, ipAddress, userAgent)

	if err != nil {
		return nil, fmt.Errorf("Session erstellen: %w", err)
	}

	return &Session{
		ID:        sessionID,
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
		CreatedAt: now,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}, nil
}

// GetSessionByToken holt eine Session nach Token
func (r *Repository) GetSessionByToken(token string) (*Session, error) {
	session := &Session{}

	err := r.db.QueryRow(`
		SELECT id, user_id, token, expires_at, created_at, ip_address, user_agent
		FROM sessions WHERE token = ?
	`, token).Scan(&session.ID, &session.UserID, &session.Token, &session.ExpiresAt,
		&session.CreatedAt, &session.IPAddress, &session.UserAgent)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return session, nil
}

// DeleteSession löscht eine Session
func (r *Repository) DeleteSession(token string) error {
	_, err := r.db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

// DeleteUserSessions löscht alle Sessions eines Users
func (r *Repository) DeleteUserSessions(userID int64) error {
	_, err := r.db.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	return err
}

// DeleteExpiredSessions löscht abgelaufene Sessions
func (r *Repository) DeleteExpiredSessions() error {
	_, err := r.db.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now())
	return err
}

// ValidatePassword prüft ein Passwort
func ValidatePassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}
