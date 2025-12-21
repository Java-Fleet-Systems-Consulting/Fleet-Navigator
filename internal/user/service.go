package user

import (
	"errors"
	"log"
	"time"
)

// Service verwaltet User-Operationen
type Service struct {
	repo           *Repository
	sessionTimeout time.Duration
}

// NewService erstellt einen neuen Service
func NewService(repo *Repository) *Service {
	return &Service{
		repo:           repo,
		sessionTimeout: 24 * time.Hour, // Standard: 24 Stunden
	}
}

// SetSessionTimeout setzt das Session-Timeout
func (s *Service) SetSessionTimeout(duration time.Duration) {
	s.sessionTimeout = duration
}

// --- Authentication ---

// Login authentifiziert einen Benutzer und erstellt eine Session
func (s *Service) Login(username, password, ipAddress, userAgent string) (*LoginResponse, error) {
	// User suchen
	user, err := s.repo.GetUserByUsername(username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("ungültige Anmeldedaten")
	}

	// Prüfen ob aktiv
	if !user.IsActive {
		return nil, errors.New("Benutzer ist deaktiviert")
	}

	// Passwort prüfen
	if !ValidatePassword(user.PasswordHash, password) {
		return nil, errors.New("ungültige Anmeldedaten")
	}

	// Session erstellen
	session, err := s.repo.CreateSession(user.ID, s.sessionTimeout, ipAddress, userAgent)
	if err != nil {
		return nil, err
	}

	// LastLogin aktualisieren
	s.repo.UpdateLastLogin(user.ID)

	log.Printf("User '%s' eingeloggt (IP: %s)", username, ipAddress)

	return &LoginResponse{
		Token:     session.Token,
		ExpiresAt: session.ExpiresAt.Unix(),
		User:      user,
	}, nil
}

// Logout beendet eine Session
func (s *Service) Logout(token string) error {
	err := s.repo.DeleteSession(token)
	if err != nil {
		return err
	}
	log.Printf("Session beendet")
	return nil
}

// ValidateToken prüft ein Token und gibt den User zurück
func (s *Service) ValidateToken(token string) (*User, error) {
	session, err := s.repo.GetSessionByToken(token)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, errors.New("ungültiges Token")
	}

	// Prüfen ob abgelaufen
	if time.Now().After(session.ExpiresAt) {
		s.repo.DeleteSession(token)
		return nil, errors.New("Token abgelaufen")
	}

	// User laden
	user, err := s.repo.GetUserByID(session.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("Benutzer nicht gefunden")
	}
	if !user.IsActive {
		return nil, errors.New("Benutzer ist deaktiviert")
	}

	return user, nil
}

// --- User Management ---

// CreateUser erstellt einen neuen Benutzer
func (s *Service) CreateUser(req CreateUserRequest) (*User, error) {
	// Validierung
	if req.Username == "" {
		return nil, errors.New("Username ist erforderlich")
	}
	if req.Password == "" {
		return nil, errors.New("Passwort ist erforderlich")
	}
	if len(req.Password) < 6 {
		return nil, errors.New("Passwort muss mindestens 6 Zeichen lang sein")
	}

	// Prüfen ob Username existiert
	existing, err := s.repo.GetUserByUsername(req.Username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("Username bereits vergeben")
	}

	// Role setzen
	role := req.Role
	if role == "" {
		role = RoleUser
	}

	user, err := s.repo.CreateUser(req.Username, req.Password, req.Email, req.DisplayName, role)
	if err != nil {
		return nil, err
	}

	log.Printf("Neuer User erstellt: %s (Role: %s)", req.Username, role)
	return user, nil
}

// GetAllUsers gibt alle Benutzer zurück
func (s *Service) GetAllUsers() ([]User, error) {
	return s.repo.GetAllUsers()
}

// GetUserByID gibt einen Benutzer zurück
func (s *Service) GetUserByID(id int64) (*User, error) {
	return s.repo.GetUserByID(id)
}

// UpdateUser aktualisiert einen Benutzer
func (s *Service) UpdateUser(id int64, req UpdateUserRequest) (*User, error) {
	// Prüfen ob User existiert
	user, err := s.repo.GetUserByID(id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("Benutzer nicht gefunden")
	}

	// Update durchführen
	err = s.repo.UpdateUser(id, req.Email, req.DisplayName, req.IsActive, req.Role)
	if err != nil {
		return nil, err
	}

	// Aktualisieren User zurückgeben
	return s.repo.GetUserByID(id)
}

// ChangePassword ändert das Passwort eines Benutzers
func (s *Service) ChangePassword(userID int64, currentPassword, newPassword string) error {
	// User laden
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("Benutzer nicht gefunden")
	}

	// Aktuelles Passwort prüfen
	if !ValidatePassword(user.PasswordHash, currentPassword) {
		return errors.New("aktuelles Passwort ist falsch")
	}

	// Neues Passwort validieren
	if len(newPassword) < 6 {
		return errors.New("neues Passwort muss mindestens 6 Zeichen lang sein")
	}

	// Passwort ändern
	err = s.repo.UpdatePassword(userID, newPassword)
	if err != nil {
		return err
	}

	log.Printf("Passwort für User %s geändert", user.Username)
	return nil
}

// DeleteUser löscht einen Benutzer
func (s *Service) DeleteUser(id int64) error {
	// Sessions löschen
	s.repo.DeleteUserSessions(id)

	// User löschen
	err := s.repo.DeleteUser(id)
	if err != nil {
		return err
	}

	log.Printf("User %d gelöscht", id)
	return nil
}

// --- Initialization ---

// InitializeDefaults erstellt den Admin-User beim ersten Start
func (s *Service) InitializeDefaults() error {
	count, err := s.repo.Count()
	if err != nil {
		return err
	}

	if count > 0 {
		log.Println("Benutzer bereits vorhanden, überspringe Initialisierung")
		return nil
	}

	log.Println("Erstelle Standard-Admin-Benutzer...")

	// Admin-User erstellen (Standard: admin/admin wie im Frontend erwartet)
	admin, err := s.repo.CreateUser("admin", "admin", "", "Administrator", RoleAdmin)
	if err != nil {
		return err
	}

	log.Printf("Admin-User erstellt: %s (Passwort: admin - BITTE ÄNDERN!)", admin.Username)
	return nil
}

// CleanupExpiredSessions löscht abgelaufene Sessions
func (s *Service) CleanupExpiredSessions() error {
	return s.repo.DeleteExpiredSessions()
}

// LogoutAllUserSessions beendet alle Sessions eines Users
func (s *Service) LogoutAllUserSessions(userID int64) error {
	return s.repo.DeleteUserSessions(userID)
}
