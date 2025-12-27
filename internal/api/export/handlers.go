// Package export enthält die HTTP-Handler für Dokument-Export-Endpoints
package export

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"fleet-navigator/internal/api/common"
)

// ExportRequest ist die Standard-Anfrage für Dokument-Exports
type ExportRequest struct {
	Content  string `json:"content"`
	Title    string `json:"title"`
	Filename string `json:"filename"`
}

// CSVExportRequest ist die Anfrage für CSV-Exports
type CSVExportRequest struct {
	Data      [][]string `json:"data"`
	Content   string     `json:"content"`
	Separator string     `json:"separator"`
	Filename  string     `json:"filename"`
}

// DocumentGenerator generiert verschiedene Dokumentformate
type DocumentGenerator interface {
	GenerateDOCX(content, title string) ([]byte, error)
	GenerateODT(content, title string) ([]byte, error)
	GeneratePDF(content, title string) ([]byte, error)
	GenerateRTF(content, title string) string
}

// Handlers enthält die HTTP-Handler für Export-Endpoints
type Handlers struct {
	generator DocumentGenerator
}

// NewHandlers erstellt neue Export-Handler
func NewHandlers(generator DocumentGenerator) *Handlers {
	return &Handlers{generator: generator}
}

// RegisterRoutes registriert die Export-API-Routen
func (h *Handlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/export/docx", h.handleExportDOCX)
	mux.HandleFunc("/api/export/odt", h.handleExportODT)
	mux.HandleFunc("/api/export/csv", h.handleExportCSV)
	mux.HandleFunc("/api/export/rtf", h.handleExportRTF)
	mux.HandleFunc("/api/export/pdf", h.handleExportPDF)
}

// handleExportDOCX exportiert Inhalt als DOCX-Datei
func (h *Handlers) handleExportDOCX(w http.ResponseWriter, r *http.Request) {
	if !common.RequirePOST(w, r) {
		return
	}

	var req ExportRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteBadRequest(w, "Invalid request body")
		return
	}

	if req.Content == "" {
		common.WriteError(w, http.StatusBadRequest, "Content is required")
		return
	}

	docxBytes, err := h.generator.GenerateDOCX(req.Content, req.Title)
	if err != nil {
		log.Printf("DOCX generation failed: %v", err)
		common.WriteError(w, http.StatusInternalServerError, "DOCX-Generierung fehlgeschlagen")
		return
	}

	filename := ensureFilename(req.Filename, "Dokument", ".docx")

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(docxBytes)))
	w.Write(docxBytes)

	log.Printf("DOCX exportiert: %s (%d bytes)", filename, len(docxBytes))
}

// handleExportODT exportiert Inhalt als ODT-Datei
func (h *Handlers) handleExportODT(w http.ResponseWriter, r *http.Request) {
	if !common.RequirePOST(w, r) {
		return
	}

	var req ExportRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteBadRequest(w, "Invalid request body")
		return
	}

	if req.Content == "" {
		common.WriteError(w, http.StatusBadRequest, "Content is required")
		return
	}

	odtBytes, err := h.generator.GenerateODT(req.Content, req.Title)
	if err != nil {
		log.Printf("ODT generation failed: %v", err)
		common.WriteError(w, http.StatusInternalServerError, "ODT-Generierung fehlgeschlagen")
		return
	}

	filename := ensureFilename(req.Filename, "Dokument", ".odt")

	w.Header().Set("Content-Type", "application/vnd.oasis.opendocument.text")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(odtBytes)))
	w.Write(odtBytes)

	log.Printf("ODT exportiert: %s (%d bytes)", filename, len(odtBytes))
}

// handleExportCSV exportiert Inhalt als CSV-Datei
func (h *Handlers) handleExportCSV(w http.ResponseWriter, r *http.Request) {
	if !common.RequirePOST(w, r) {
		return
	}

	var req CSVExportRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteBadRequest(w, "Invalid request body")
		return
	}

	if len(req.Data) == 0 && req.Content == "" {
		common.WriteError(w, http.StatusBadRequest, "Data or content is required")
		return
	}

	// Separator setzen (Standard: Semikolon für deutsches Excel)
	separator := req.Separator
	if separator == "" {
		separator = ";"
	}

	// CSV generieren
	var csvContent string
	if len(req.Data) > 0 {
		var builder strings.Builder
		for _, row := range req.Data {
			builder.WriteString(strings.Join(row, separator))
			builder.WriteString("\r\n")
		}
		csvContent = builder.String()
	} else {
		csvContent = req.Content
	}

	// BOM für UTF-8 (für Excel-Kompatibilität)
	csvBytes := append([]byte{0xEF, 0xBB, 0xBF}, []byte(csvContent)...)

	filename := ensureFilename(req.Filename, "Tabelle", ".csv")

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(csvBytes)))
	w.Write(csvBytes)

	log.Printf("CSV exportiert: %s (%d bytes)", filename, len(csvBytes))
}

// handleExportRTF exportiert Inhalt als RTF-Datei
func (h *Handlers) handleExportRTF(w http.ResponseWriter, r *http.Request) {
	if !common.RequirePOST(w, r) {
		return
	}

	var req ExportRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteBadRequest(w, "Invalid request body")
		return
	}

	if req.Content == "" {
		common.WriteError(w, http.StatusBadRequest, "Content is required")
		return
	}

	rtfContent := h.generator.GenerateRTF(req.Content, req.Title)

	filename := ensureFilename(req.Filename, "Dokument", ".rtf")

	w.Header().Set("Content-Type", "application/rtf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(rtfContent)))
	w.Write([]byte(rtfContent))

	log.Printf("RTF exportiert: %s (%d bytes)", filename, len(rtfContent))
}

// handleExportPDF exportiert Inhalt als PDF-Datei
func (h *Handlers) handleExportPDF(w http.ResponseWriter, r *http.Request) {
	if !common.RequirePOST(w, r) {
		return
	}

	var req ExportRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteBadRequest(w, "Invalid request body")
		return
	}

	if req.Content == "" {
		common.WriteError(w, http.StatusBadRequest, "Content is required")
		return
	}

	pdfBytes, err := h.generator.GeneratePDF(req.Content, req.Title)
	if err != nil {
		log.Printf("PDF generation failed: %v", err)
		common.WriteError(w, http.StatusInternalServerError, "PDF-Generierung fehlgeschlagen")
		return
	}

	filename := ensureFilename(req.Filename, "Dokument", ".pdf")

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(pdfBytes)))
	w.Write(pdfBytes)

	log.Printf("PDF exportiert: %s (%d bytes)", filename, len(pdfBytes))
}

// ensureFilename stellt sicher, dass ein Dateiname vorhanden ist
func ensureFilename(filename, prefix, extension string) string {
	if filename == "" {
		filename = fmt.Sprintf("%s_%s%s", prefix, time.Now().Format("2006-01-02_15-04-05"), extension)
	}
	if !strings.HasSuffix(strings.ToLower(filename), extension) {
		filename += extension
	}
	return filename
}
