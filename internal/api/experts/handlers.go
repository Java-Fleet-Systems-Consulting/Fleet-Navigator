// Package experts enthält die HTTP-Handler für Experten-Endpoints
package experts

import (
	"net/http"
	"strconv"
	"strings"

	"fleet-navigator/internal/api/common"
	"fleet-navigator/internal/experte"
)

// Handlers enthält die HTTP-Handler für Experten-Endpoints
type Handlers struct {
	service *experte.Service
}

// NewHandlers erstellt neue Experten-Handler
func NewHandlers(service *experte.Service) *Handlers {
	return &Handlers{service: service}
}

// RegisterRoutes registriert die Experten-API-Routen
func (h *Handlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/experts", h.handleExperts)
	mux.HandleFunc("/api/experts/", h.handleExpertByID)
	mux.HandleFunc("/api/experts/default-anti-hallucination", h.handleDefaultAntiHallucination)
}

// handleExperts behandelt GET /api/experts und POST /api/experts
func (h *Handlers) handleExperts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		onlyActive := r.URL.Query().Get("active") == "true"
		experts, err := h.service.GetAllExperts(onlyActive)
		if err != nil {
			common.WriteInternalError(w, err, "Experten konnten nicht geladen werden")
			return
		}
		common.WriteJSON(w, http.StatusOK, experts)

	case http.MethodPost:
		var req experte.CreateExpertRequest
		if err := common.DecodeJSON(r, &req); err != nil {
			common.WriteBadRequest(w, "Invalid JSON")
			return
		}

		expert, err := h.service.CreateExpert(req)
		if err != nil {
			common.WriteInternalError(w, err, "Experte konnte nicht erstellt werden")
			return
		}

		common.WriteJSON(w, http.StatusCreated, expert)

	default:
		common.WriteMethodNotAllowed(w)
	}
}

// handleDefaultAntiHallucination gibt den Standard-Anti-Halluzinations-Prompt zurück
// GET /api/experts/default-anti-hallucination
func (h *Handlers) handleDefaultAntiHallucination(w http.ResponseWriter, r *http.Request) {
	if !common.RequireGET(w, r) {
		return
	}

	common.WriteJSON(w, http.StatusOK, map[string]string{
		"defaultPrompt": experte.DefaultAntiHallucinationPrompt,
	})
}

// handleExpertByID behandelt /api/experts/{id} und /api/experts/{id}/modes
func (h *Handlers) handleExpertByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/experts/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		common.WriteBadRequest(w, "Missing expert ID")
		return
	}

	// Prüfen ob es um Modi geht
	if len(parts) >= 2 && parts[1] == "modes" {
		h.handleExpertModes(w, r, parts[0])
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		common.WriteBadRequest(w, "Invalid expert ID")
		return
	}

	switch r.Method {
	case http.MethodGet:
		expert, err := h.service.GetExpert(id)
		if err != nil {
			common.WriteInternalError(w, err, "Experte konnte nicht geladen werden")
			return
		}
		if expert == nil {
			common.WriteNotFound(w, "Expert not found")
			return
		}
		common.WriteJSON(w, http.StatusOK, expert)

	case http.MethodPut:
		var req experte.UpdateExpertRequest
		if err := common.DecodeJSON(r, &req); err != nil {
			common.WriteBadRequest(w, "Invalid JSON")
			return
		}

		expert, err := h.service.UpdateExpert(id, req)
		if err != nil {
			common.WriteInternalError(w, err, "Experte konnte nicht aktualisiert werden")
			return
		}
		common.WriteJSON(w, http.StatusOK, expert)

	case http.MethodDelete:
		if err := h.service.DeleteExpert(id); err != nil {
			common.WriteInternalError(w, err, "Experte konnte nicht gelöscht werden")
			return
		}
		common.WriteSuccessMessage(w, "Experte gelöscht")

	default:
		common.WriteMethodNotAllowed(w)
	}
}

// handleExpertModes behandelt /api/experts/{id}/modes
func (h *Handlers) handleExpertModes(w http.ResponseWriter, r *http.Request, expertIDStr string) {
	expertID, err := strconv.ParseInt(expertIDStr, 10, 64)
	if err != nil {
		common.WriteBadRequest(w, "Invalid expert ID")
		return
	}

	switch r.Method {
	case http.MethodGet:
		modes, err := h.service.GetModes(expertID)
		if err != nil {
			common.WriteInternalError(w, err, "Modi konnten nicht geladen werden")
			return
		}
		common.WriteJSON(w, http.StatusOK, modes)

	case http.MethodPost:
		var req experte.CreateModeRequest
		if err := common.DecodeJSON(r, &req); err != nil {
			common.WriteBadRequest(w, "Invalid JSON")
			return
		}

		mode, err := h.service.AddMode(expertID, req)
		if err != nil {
			common.WriteInternalError(w, err, "Modus konnte nicht erstellt werden")
			return
		}
		common.WriteJSON(w, http.StatusCreated, mode)

	default:
		common.WriteMethodNotAllowed(w)
	}
}
