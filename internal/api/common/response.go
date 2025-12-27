// Package common enthält gemeinsame Hilfsfunktionen für API-Handler
package common

import (
	"encoding/json"
	"log"
	"net/http"
)

// ErrorResponse repräsentiert eine einheitliche Fehlerantwort
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// SuccessResponse repräsentiert eine einheitliche Erfolgsantwort
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// WriteJSON schreibt eine JSON-Antwort mit dem angegebenen Status-Code
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("JSON encode error: %v", err)
	}
}

// WriteSuccess schreibt eine erfolgreiche JSON-Antwort
func WriteSuccess(w http.ResponseWriter, data interface{}) {
	WriteJSON(w, http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

// WriteSuccessMessage schreibt eine erfolgreiche JSON-Antwort mit Nachricht
func WriteSuccessMessage(w http.ResponseWriter, message string) {
	WriteJSON(w, http.StatusOK, SuccessResponse{
		Success: true,
		Message: message,
	})
}

// WriteError schreibt eine Fehler-JSON-Antwort
func WriteError(w http.ResponseWriter, statusCode int, message string) {
	WriteJSON(w, statusCode, ErrorResponse{
		Success: false,
		Error:   message,
	})
}

// WriteBadRequest schreibt einen 400 Bad Request Fehler
func WriteBadRequest(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusBadRequest, message)
}

// WriteNotFound schreibt einen 404 Not Found Fehler
func WriteNotFound(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusNotFound, message)
}

// WriteInternalError schreibt einen 500 Internal Server Error
// WICHTIG: Gibt generische Nachricht an Client, loggt Details intern
func WriteInternalError(w http.ResponseWriter, err error, publicMessage string) {
	log.Printf("Internal error: %v", err)
	WriteError(w, http.StatusInternalServerError, publicMessage)
}

// WriteMethodNotAllowed schreibt einen 405 Method Not Allowed Fehler
func WriteMethodNotAllowed(w http.ResponseWriter) {
	WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
}

// RequireMethod prüft ob die Request-Methode erlaubt ist
// Gibt true zurück wenn die Methode erlaubt ist
func RequireMethod(w http.ResponseWriter, r *http.Request, methods ...string) bool {
	for _, m := range methods {
		if r.Method == m {
			return true
		}
	}
	WriteMethodNotAllowed(w)
	return false
}

// RequireGET prüft ob es ein GET-Request ist
func RequireGET(w http.ResponseWriter, r *http.Request) bool {
	return RequireMethod(w, r, http.MethodGet)
}

// RequirePOST prüft ob es ein POST-Request ist
func RequirePOST(w http.ResponseWriter, r *http.Request) bool {
	return RequireMethod(w, r, http.MethodPost)
}

// RequireGETorPOST prüft ob es ein GET oder POST-Request ist
func RequireGETorPOST(w http.ResponseWriter, r *http.Request) bool {
	return RequireMethod(w, r, http.MethodGet, http.MethodPost)
}

// DecodeJSON dekodiert JSON aus dem Request-Body in das Ziel
func DecodeJSON(r *http.Request, target interface{}) error {
	return json.NewDecoder(r.Body).Decode(target)
}
