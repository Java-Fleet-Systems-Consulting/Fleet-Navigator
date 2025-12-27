// Package mateshandlers enthält die HTTP-Handler für Mates/Pairing-Endpoints
package mateshandlers

import (
	"log"
	"net/http"
	"strings"

	"fleet-navigator/internal/api/common"
	"fleet-navigator/internal/security"
)

// TrustedMate repräsentiert einen vertrauten Mate
type TrustedMate struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// PairingRequest repräsentiert eine ausstehende Pairing-Anfrage
type PairingRequest struct {
	RequestID string `json:"requestId"`
	MateName  string `json:"mateName"`
	MateType  string `json:"mateType"`
	Timestamp int64  `json:"timestamp"`
}

// PairingManager verwaltet Pairing-Anfragen und vertraute Mates
type PairingManager interface {
	GetTrustedMates() []security.TrustedMate
	GetPendingRequests() []security.PairingRequest
	RemoveTrustedMate(mateID string) error
}

// WebSocketServer verwaltet WebSocket-Verbindungen
type WebSocketServer interface {
	GetConnectedMates() []string
	ApprovePairing(requestID string) error
	RejectPairing(requestID string) error
	DisconnectMate(mateID string) error
}

// Handlers enthält die HTTP-Handler für Mates-Endpoints
type Handlers struct {
	pairingManager PairingManager
	wsServer       WebSocketServer
}

// NewHandlers erstellt neue Mates-Handler
func NewHandlers(pairingManager PairingManager, wsServer WebSocketServer) *Handlers {
	return &Handlers{
		pairingManager: pairingManager,
		wsServer:       wsServer,
	}
}

// RegisterRoutes registriert die Mates-API-Routen
func (h *Handlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/mates", h.handleMates)
	mux.HandleFunc("/api/mates/pending", h.handlePendingPairings)
	mux.HandleFunc("/api/mates/remove", h.handleRemoveMate)
	mux.HandleFunc("/api/pairing/pending", h.handlePendingPairings)
	mux.HandleFunc("/api/pairing/approve/", h.handleApprovePairing)
	mux.HandleFunc("/api/pairing/reject/", h.handleRejectPairing)
	// Aliases für Frontend-Kompatibilität
	mux.HandleFunc("/api/fleet-mate/mates", h.handleMates)
}

// handleMates - GET /api/mates
func (h *Handlers) handleMates(w http.ResponseWriter, r *http.Request) {
	if !common.RequireGET(w, r) {
		return
	}

	mates := h.pairingManager.GetTrustedMates()
	connected := h.wsServer.GetConnectedMates()

	type MateResponse struct {
		MateID          string `json:"mateId"`
		Name            string `json:"name"`
		Description     string `json:"description"`
		Status          string `json:"status"`
		LastStatsUpdate *int64 `json:"lastStatsUpdate"`
	}

	result := make([]MateResponse, 0, len(mates))

	for _, mate := range mates {
		status := "OFFLINE"
		for _, cID := range connected {
			if cID == mate.ID {
				status = "ONLINE"
				break
			}
		}
		result = append(result, MateResponse{
			MateID:          mate.ID,
			Name:            mate.Name,
			Description:     mate.Type,
			Status:          status,
			LastStatsUpdate: nil,
		})
	}

	common.WriteJSON(w, http.StatusOK, result)
}

// handlePendingPairings - GET /api/mates/pending
func (h *Handlers) handlePendingPairings(w http.ResponseWriter, r *http.Request) {
	if !common.RequireGET(w, r) {
		return
	}

	requests := h.pairingManager.GetPendingRequests()
	common.WriteJSON(w, http.StatusOK, requests)
}

// handleApprovePairing - POST /api/pairing/approve/{requestId}
func (h *Handlers) handleApprovePairing(w http.ResponseWriter, r *http.Request) {
	log.Printf("handleApprovePairing aufgerufen: %s %s", r.Method, r.URL.Path)

	if !common.RequirePOST(w, r) {
		return
	}

	requestID := strings.TrimPrefix(r.URL.Path, "/api/pairing/approve/")
	log.Printf("Request ID aus Pfad: '%s'", requestID)

	if requestID == "" || requestID == r.URL.Path {
		var req struct {
			RequestID string `json:"request_id"`
		}
		if err := common.DecodeJSON(r, &req); err == nil {
			requestID = req.RequestID
			log.Printf("Request ID aus Body: '%s'", requestID)
		}
	}

	if requestID == "" {
		log.Printf("Keine Request ID gefunden!")
		common.WriteBadRequest(w, "Request ID required")
		return
	}

	log.Printf("Rufe ApprovePairing auf mit ID: %s", requestID)
	if err := h.wsServer.ApprovePairing(requestID); err != nil {
		log.Printf("ApprovePairing Fehler: %v", err)
		common.WriteJSON(w, http.StatusOK, map[string]string{"error": err.Error()})
		return
	}

	log.Printf("Pairing erfolgreich genehmigt: %s", requestID)
	common.WriteJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

// handleRejectPairing - POST /api/pairing/reject/{requestId}
func (h *Handlers) handleRejectPairing(w http.ResponseWriter, r *http.Request) {
	if !common.RequirePOST(w, r) {
		return
	}

	requestID := strings.TrimPrefix(r.URL.Path, "/api/pairing/reject/")

	if requestID == "" || requestID == r.URL.Path {
		var req struct {
			RequestID string `json:"request_id"`
		}
		if err := common.DecodeJSON(r, &req); err == nil {
			requestID = req.RequestID
		}
	}

	if requestID == "" {
		common.WriteBadRequest(w, "Request ID required")
		return
	}

	if err := h.wsServer.RejectPairing(requestID); err != nil {
		common.WriteJSON(w, http.StatusOK, map[string]string{"error": err.Error()})
		return
	}

	common.WriteJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

// handleRemoveMate - POST /api/mates/remove
func (h *Handlers) handleRemoveMate(w http.ResponseWriter, r *http.Request) {
	if !common.RequirePOST(w, r) {
		return
	}

	var req struct {
		MateID string `json:"mate_id"`
	}
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteBadRequest(w, "Invalid JSON")
		return
	}

	// Zuerst den Mate über WebSocket benachrichtigen und Verbindung trennen
	if err := h.wsServer.DisconnectMate(req.MateID); err != nil {
		log.Printf("Mate %s war nicht verbunden: %v", req.MateID, err)
	}

	// Dann aus trusted_mates.json entfernen
	if err := h.pairingManager.RemoveTrustedMate(req.MateID); err != nil {
		common.WriteJSON(w, http.StatusOK, map[string]string{"error": err.Error()})
		return
	}

	log.Printf("Mate entfernt: %s", req.MateID)
	common.WriteJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}
