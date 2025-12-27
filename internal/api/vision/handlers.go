// Package visionhandlers enthält die HTTP-Handler für Vision/OCR-Endpoints
package visionhandlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"fleet-navigator/internal/api/common"
	"fleet-navigator/internal/vision"
)

// ImageAnalysis ist das Ergebnis einer Bild-Analyse
type ImageAnalysis = vision.ImageAnalysis

// CombinedAnalysisResult ist das Ergebnis einer kombinierten Vision+OCR Analyse
type CombinedAnalysisResult = vision.CombinedAnalysisResult

// VisionService definiert die Vision-Service-Methoden
type VisionService interface {
	IsAvailable() bool
	GetModel() string
	StreamAnalyzeImage(ctx context.Context, base64Image, prompt string, onChunk func(content string, done bool)) error
	AnalyzeDocument(ctx context.Context, base64Image string) (*ImageAnalysis, error)
	AnalyzeDocumentWithOCR(ctx context.Context, base64Image, dataDir string) (*CombinedAnalysisResult, error)
}

// Handlers enthält die HTTP-Handler für Vision-Endpoints
type Handlers struct {
	visionService VisionService
	dataDir       string
}

// NewHandlers erstellt neue Vision-Handler
func NewHandlers(visionService VisionService, dataDir string) *Handlers {
	return &Handlers{
		visionService: visionService,
		dataDir:       dataDir,
	}
}

// RegisterRoutes registriert die Vision-API-Routen
func (h *Handlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/vision/analyze", h.handleAnalyze)
	mux.HandleFunc("/api/vision/document", h.handleDocument)
	mux.HandleFunc("/api/vision/status", h.handleStatus)
	mux.HandleFunc("/api/vision/ocr", h.handleOCR)
	mux.HandleFunc("/api/vision/pdf-stream", h.handlePDFStream)
}

// handleAnalyze - POST /api/vision/analyze (Streaming)
func (h *Handlers) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if !common.RequirePOST(w, r) {
		return
	}

	var req struct {
		Image  string `json:"image"`
		Prompt string `json:"prompt"`
	}
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteBadRequest(w, "Invalid JSON")
		return
	}

	if req.Image == "" {
		common.WriteBadRequest(w, "Image (base64) is required")
		return
	}

	// SSE Headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		common.WriteInternalError(w, nil, "Streaming not supported")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 180*time.Second)
	defer cancel()

	err := h.visionService.StreamAnalyzeImage(ctx, req.Image, req.Prompt, func(content string, done bool) {
		data := map[string]interface{}{
			"content": content,
			"done":    done,
		}
		jsonData, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	})

	if err != nil {
		errData := map[string]interface{}{
			"error": err.Error(),
			"done":  true,
		}
		jsonData, _ := json.Marshal(errData)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	}
}

// handleDocument - POST /api/vision/document
func (h *Handlers) handleDocument(w http.ResponseWriter, r *http.Request) {
	if !common.RequirePOST(w, r) {
		return
	}

	var req struct {
		Image string `json:"image"`
	}
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteBadRequest(w, "Invalid JSON")
		return
	}

	if req.Image == "" {
		common.WriteBadRequest(w, "Image (base64) is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 180*time.Second)
	defer cancel()

	combinedResult, err := h.visionService.AnalyzeDocumentWithOCR(ctx, req.Image, h.dataDir)
	if err != nil {
		log.Printf("[Vision] Kombinierte Analyse fehlgeschlagen, versuche nur Vision: %v", err)
		analysis, visionErr := h.visionService.AnalyzeDocument(ctx, req.Image)
		if visionErr != nil {
			common.WriteJSON(w, http.StatusOK, map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		common.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"success":        true,
			"analysis":       analysis,
			"tesseractUsed":  false,
			"tesseractError": err.Error(),
		})
		return
	}

	common.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success":       true,
		"analysis":      combinedResult.ImageAnalysis,
		"fullOcrText":   combinedResult.FullOCRText,
		"tesseractUsed": combinedResult.TesseractUsed,
		"pageCount":     combinedResult.PageCount,
	})
}

// handleStatus - GET /api/vision/status
func (h *Handlers) handleStatus(w http.ResponseWriter, r *http.Request) {
	if !common.RequireGET(w, r) {
		return
	}

	visionAvailable := h.visionService.IsAvailable()
	tesseractInstalled := vision.IsTesseractInstalled(h.dataDir)
	tesseractLangs := vision.GetTesseractLanguages(h.dataDir)

	common.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"available": visionAvailable,
		"model":     h.visionService.GetModel(),
		"provider":  "llama-server",
		"tesseract": map[string]interface{}{
			"installed": tesseractInstalled,
			"languages": tesseractLangs,
		},
		"combinedAnalysis": visionAvailable && tesseractInstalled,
	})
}

// handleOCR - POST /api/vision/ocr
func (h *Handlers) handleOCR(w http.ResponseWriter, r *http.Request) {
	if !common.RequirePOST(w, r) {
		return
	}

	var req struct {
		Image     string `json:"image"`
		Languages string `json:"languages"`
	}
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteBadRequest(w, "Invalid JSON")
		return
	}

	if req.Image == "" {
		common.WriteBadRequest(w, "Image (base64) is required")
		return
	}

	if !vision.IsTesseractInstalled(h.dataDir) {
		common.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   "Tesseract ist nicht installiert. Bitte über Setup installieren.",
			"hint":    "GET /api/setup/tesseract/download",
		})
		return
	}

	languages := req.Languages
	if languages == "" {
		languages = "deu+eng"
	}

	log.Printf("[OCR] Starte Tesseract OCR (Sprachen: %s)", languages)

	text, err := vision.TesseractOCRFromBase64(req.Image, h.dataDir, languages)
	if err != nil {
		log.Printf("[OCR] Fehler: %v", err)
		common.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	log.Printf("[OCR] ✅ Erfolgreich: %d Zeichen extrahiert", len(text))

	common.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"text":      text,
		"length":    len(text),
		"languages": languages,
	})
}

// handlePDFStream - POST /api/vision/pdf-stream
func (h *Handlers) handlePDFStream(w http.ResponseWriter, r *http.Request) {
	if !common.RequirePOST(w, r) {
		return
	}

	if err := r.ParseMultipartForm(100 << 20); err != nil {
		common.WriteBadRequest(w, "PDF zu groß oder ungültiges Format")
		return
	}

	file, header, err := r.FormFile("pdf")
	if err != nil {
		common.WriteBadRequest(w, "PDF-Datei fehlt")
		return
	}
	defer file.Close()

	log.Printf("[Vision/PDF] Verarbeite PDF: %s (%d Bytes)", header.Filename, header.Size)

	tmpPDF, err := os.CreateTemp("", "fleet-pdf-*.pdf")
	if err != nil {
		common.WriteInternalError(w, err, "Temp-Datei erstellen fehlgeschlagen")
		return
	}
	defer os.Remove(tmpPDF.Name())

	if _, err := io.Copy(tmpPDF, file); err != nil {
		tmpPDF.Close()
		common.WriteInternalError(w, err, "PDF speichern fehlgeschlagen")
		return
	}
	tmpPDF.Close()

	// SSE-Header setzen
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		common.WriteInternalError(w, nil, "Streaming nicht unterstützt")
		return
	}

	tempDir, err := os.MkdirTemp("", "fleet-pdf-pages-*")
	if err != nil {
		sendSSEError(w, flusher, "Temp-Verzeichnis: "+err.Error())
		return
	}
	defer os.RemoveAll(tempDir)

	sendSSEProgress(w, flusher, "converting", "Konvertiere PDF zu Bildern...", 0, 0)

	cmd := exec.Command("pdftoppm", "-png", "-r", "200", tmpPDF.Name(), filepath.Join(tempDir, "page"))
	if err := cmd.Run(); err != nil {
		sendSSEError(w, flusher, "PDF-Konvertierung fehlgeschlagen (poppler-utils installiert?)")
		return
	}

	pattern := filepath.Join(tempDir, "page-*.png")
	images, err := filepath.Glob(pattern)
	if err != nil || len(images) == 0 {
		sendSSEError(w, flusher, "Keine Seiten gefunden")
		return
	}

	totalPages := len(images)
	sendSSEProgress(w, flusher, "processing", fmt.Sprintf("Starte OCR für %d Seiten...", totalPages), 0, totalPages)

	var allText strings.Builder
	for i, imagePath := range images {
		pageNum := i + 1
		progress := int(float64(pageNum) / float64(totalPages) * 100)

		fmt.Fprintf(w, "data: {\"status\": \"ocr\", \"page\": %d, \"totalPages\": %d, \"progress\": %d}\n\n", pageNum, totalPages, progress)
		flusher.Flush()

		text, err := vision.TesseractOCR(imagePath, h.dataDir, "deu+eng+tur")
		if err != nil {
			log.Printf("[Vision/PDF] ⚠️ Seite %d OCR-Fehler: %v", pageNum, err)
			continue
		}

		if allText.Len() > 0 {
			allText.WriteString("\n\n--- Seite ")
			allText.WriteString(fmt.Sprintf("%d", pageNum))
			allText.WriteString(" ---\n\n")
		}
		allText.WriteString(text)
	}

	sendSSEProgress(w, flusher, "vision", "Analysiere Layout und visuelle Elemente...", 100, totalPages)

	var visionResult *ImageAnalysis
	if len(images) > 0 {
		base64Data, err := loadImageAsBase64(images[0])
		if err == nil {
			ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
			defer cancel()
			visionResult, _ = h.visionService.AnalyzeDocument(ctx, base64Data)
		}
	}

	result := map[string]interface{}{
		"status":        "complete",
		"totalPages":    totalPages,
		"fullOcrText":   allText.String(),
		"tesseractUsed": true,
		"charCount":     allText.Len(),
	}
	if visionResult != nil {
		result["analysis"] = visionResult
	}

	resultJSON, _ := json.Marshal(result)
	fmt.Fprintf(w, "data: %s\n\n", resultJSON)
	flusher.Flush()

	log.Printf("[Vision/PDF] ✅ PDF verarbeitet: %d Seiten, %d Zeichen", totalPages, allText.Len())
}

// --- Helper Functions ---

func sendSSEError(w http.ResponseWriter, flusher http.Flusher, errMsg string) {
	fmt.Fprintf(w, "data: {\"error\": \"%s\"}\n\n", errMsg)
	flusher.Flush()
}

func sendSSEProgress(w http.ResponseWriter, flusher http.Flusher, status, message string, progress, total int) {
	data := map[string]interface{}{
		"status":  status,
		"message": message,
	}
	if total > 0 {
		data["totalPages"] = total
		data["progress"] = progress
	}
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
	flusher.Flush()
}

func loadImageAsBase64(imagePath string) (string, error) {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}
