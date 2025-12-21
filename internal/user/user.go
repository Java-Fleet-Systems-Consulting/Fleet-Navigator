package user

import (
	"time"
)

// User repräsentiert einen Benutzer
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email,omitempty"`
	PasswordHash string    `json:"-"` // Nicht in JSON ausgeben
	DisplayName  string    `json:"displayName"`
	Role         string    `json:"role"` // admin, user, guest
	IsActive     bool      `json:"isActive"`
	LastLoginAt  *time.Time `json:"lastLoginAt,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// UserRole Konstanten
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
	RoleGuest = "guest"
)

// Session repräsentiert eine Benutzer-Sitzung
type Session struct {
	ID        string    `json:"id"`
	UserID    int64     `json:"userId"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
	IPAddress string    `json:"ipAddress,omitempty"`
	UserAgent string    `json:"userAgent,omitempty"`
}

// LoginRequest für Login-Anfragen
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse für Login-Antworten
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expiresAt"` // Unix timestamp
	User      *User  `json:"user"`
}

// CreateUserRequest für User-Erstellung
type CreateUserRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	Email       string `json:"email,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Role        string `json:"role,omitempty"`
}

// UpdateUserRequest für User-Updates
type UpdateUserRequest struct {
	Email       *string `json:"email,omitempty"`
	DisplayName *string `json:"displayName,omitempty"`
	IsActive    *bool   `json:"isActive,omitempty"`
	Role        *string `json:"role,omitempty"`
}

// ChangePasswordRequest für Passwort-Änderung
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}
