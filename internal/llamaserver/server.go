package llamaserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// VRAMStrategy definiert die VRAM-Management-Strategie
type VRAMStrategy string

const (
	// StrategySmartSwap - Nur VRAM löschen wenn nötig (Standard, empfohlen)
	StrategySmartSwap VRAMStrategy = "smart_swap"
	// StrategyAlwaysClear - Immer VRAM löschen vor Modell-Laden (sicherste)
	StrategyAlwaysClear VRAMStrategy = "always_clear"
	// StrategySmartOffload - Automatisches Offloading auf CPU wenn VRAM nicht reicht
	StrategySmartOffload VRAMStrategy = "smart_offload"
	// StrategyManual - Keine automatische VRAM-Verwaltung
	StrategyManual VRAMStrategy = "manual"
)

// Config für den llama-server
type Config struct {
	Port         int          `json:"port"`
	Host         string       `json:"host"`
	ModelPath    string       `json:"modelPath"`
	ModelsDir    string       `json:"modelsDir"`
	BinaryPath   string       `json:"binaryPath"`
	LibraryPath  string       `json:"libraryPath"`
	MmprojPath   string       `json:"mmprojPath"`   // --mmproj: Multimodal Projector für Vision-Modelle (LLaVA)
	GPULayers    int          `json:"gpuLayers"`    // -ngl Parameter
	ContextSize  int          `json:"contextSize"`  // -c Parameter
	Threads      int          `json:"threads"`      // -t Parameter
	VRAMStrategy VRAMStrategy `json:"vramStrategy"` // VRAM-Management-Strategie
	VRAMReserve  int          `json:"vramReserve"`  // MB VRAM für System reservieren
	UseMmap      bool         `json:"useMmap"`      // --mmap: Memory-mapped I/O für schnelleres Laden
	UseMlock     bool         `json:"useMlock"`     // --mlock: Modell im RAM pinnen (verhindert Swap)
	VisionEnabled bool        `json:"visionEnabled"` // Vision/Multimodal aktiviert
}

// DefaultConfig gibt die Standard-Konfiguration zurück
func DefaultConfig(dataDir string) Config {
	// Suche nach llama-server Binary (immer zuerst im dataDir/bin/)
	binaryPath, libraryPath, err := GetOrExtractLlamaServer(dataDir)
	if err != nil {
		log.Printf("llama-server Binary nicht gefunden: %v", err)
		binaryPath = ""
		libraryPath = ""
	}

	// ModelsDir immer im dataDir (plattformunabhängig)
	modelsDir := filepath.Join(dataDir, "models")

	return Config{
		Port:         2026, // llama-server Port (Fleet Navigator läuft auf 2025)
		Host:         "127.0.0.1", // Nur localhost - keine Firewall-Berechtigung nötig!
		ModelsDir:    modelsDir,
		BinaryPath:   binaryPath,
		LibraryPath:  libraryPath,
		GPULayers:    99,               // Alle Layer auf GPU (wird bei SmartOffload automatisch angepasst)
		ContextSize:  16384,            // 16K Context als Standard (65K braucht zu viel VRAM)
		Threads:      8,
		VRAMStrategy: StrategySmartSwap, // Standard: Smart Swap
		VRAMReserve:  512,               // 512MB für System reservieren
		UseMmap:      true,              // Standard: mmap aktiviert (schnelleres Laden)
		UseMlock:     false,             // Standard: mlock deaktiviert (braucht Berechtigung)
	}
}

// TemplateAdapter definiert das Interface für Template-Adaption
type TemplateAdapter interface {
	AdaptMessages(modelName string, messages []ChatMessage) []ChatMessage
}

// Server verwaltet den llama-server Prozess
type Server struct {
	config          Config
	cmd             *exec.Cmd
	running         bool
	modelName       string
	mu              sync.RWMutex
	cancelFunc      context.CancelFunc
	heartbeatCancel context.CancelFunc // Stoppt den Heartbeat-Monitor
	navigatorPort   int                // Port des Fleet Navigators für Heartbeat
	templateAdapter TemplateAdapter    // Für Model-Template-Adaption
}

// NewServer erstellt einen neuen Server-Manager
func NewServer(config Config) *Server {
	return &Server{
		config:        config,
		navigatorPort: 2025, // Default Fleet Navigator Port
	}
}

// SetTemplateAdapter setzt den Template-Adapter für Model-spezifische Anpassungen
func (s *Server) SetTemplateAdapter(adapter TemplateAdapter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.templateAdapter = adapter
	log.Printf("Template-Adapter gesetzt")
}

// SetNavigatorPort setzt den Port für Heartbeat-Checks
func (s *Server) SetNavigatorPort(port int) {
	s.navigatorPort = port
}

// startHeartbeatMonitor startet einen Goroutine der Fleet Navigator überwacht
// Wenn Fleet Navigator nicht mehr antwortet, wird llama-server automatisch beendet
func (s *Server) startHeartbeatMonitor() {
	ctx, cancel := context.WithCancel(context.Background())
	s.heartbeatCancel = cancel

	go func() {
		client := &http.Client{Timeout: 3 * time.Second}
		failCount := 0
		maxFailures := 3 // Nach 3 Fehlern (15 Sekunden) beenden

		log.Printf("🫀 Heartbeat-Monitor gestartet (prüft Fleet Navigator auf Port %d)", s.navigatorPort)

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Printf("🫀 Heartbeat-Monitor beendet")
				return
			case <-ticker.C:
				// Ping Fleet Navigator
				url := fmt.Sprintf("http://localhost:%d/api/system/health", s.navigatorPort)
				resp, err := client.Get(url)

				if err != nil || resp.StatusCode != http.StatusOK {
					failCount++
					log.Printf("🫀 Heartbeat fehlgeschlagen (%d/%d): Fleet Navigator antwortet nicht", failCount, maxFailures)

					if resp != nil {
						resp.Body.Close()
					}

					if failCount >= maxFailures {
						log.Printf("🫀 Fleet Navigator nicht erreichbar - beende llama-server automatisch!")
						s.Stop()
						return
					}
				} else {
					resp.Body.Close()
					if failCount > 0 {
						log.Printf("🫀 Heartbeat wiederhergestellt - Fleet Navigator ist wieder online")
					}
					failCount = 0 // Reset bei Erfolg
				}
			}
		}
	}()
}

// findLlamaServerBinary ist DEPRECATED - verwende GetOrExtractLlamaServer() stattdessen
// Diese Funktion bleibt für Kompatibilität, sucht aber nur noch an Systemstandorten
func findLlamaServerBinary() string {
	var binaryName string
	if runtime.GOOS == "windows" {
		binaryName = "llama-server.exe"
	} else {
		binaryName = "llama-server"
	}

	// Nur System-Pfade (für manuell installierte llama-server)
	var paths []string
	if runtime.GOOS == "windows" {
		// Windows: typische Installations-Pfade
		paths = []string{
			filepath.Join(os.Getenv("LOCALAPPDATA"), "llama.cpp", binaryName),
			filepath.Join(os.Getenv("PROGRAMFILES"), "llama.cpp", binaryName),
		}
	} else {
		// Linux/macOS: Standard-Pfade
		paths = []string{
			"/usr/local/bin/" + binaryName,
			"/usr/bin/" + binaryName,
		}
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			log.Printf("llama-server in Systempfad gefunden: %s", p)
			return p
		}
	}

	return ""
}

// GetAvailableModels gibt alle verfügbaren GGUF-Modelle zurück
func (s *Server) GetAvailableModels() ([]ModelInfo, error) {
	var models []ModelInfo

	// Durchsuche ModelsDir
	if s.config.ModelsDir != "" {
		err := filepath.Walk(s.config.ModelsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Fehler ignorieren
			}
			if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".gguf") {
				models = append(models, ModelInfo{
					Name:     info.Name(),
					Path:     path,
					Size:     info.Size(),
					Modified: info.ModTime(),
				})
			}
			return nil
		})
		if err != nil {
			log.Printf("Fehler beim Durchsuchen von ModelsDir: %v", err)
		}
	}

	// HINWEIS: ~/.java-fleet/ Fallback entfernt - Go-Version ist standalone
	// Nur ~/.fleet-navigator/models/ wird verwendet

	return models, nil
}

// Start startet den llama-server mit dem angegebenen Modell
func (s *Server) Start(modelPath string) error {
	// BinaryPath neu ermitteln falls noch nicht gesetzt (z.B. nach Setup-Download)
	if s.config.BinaryPath == "" {
		// Versuche Binary im ModelsDir-Parent zu finden (dataDir/bin/)
		if s.config.ModelsDir != "" {
			dataDir := filepath.Dir(s.config.ModelsDir) // models -> dataDir
			binPath, libPath, err := GetOrExtractLlamaServer(dataDir)
			if err == nil {
				s.config.BinaryPath = binPath
				s.config.LibraryPath = libPath
				log.Printf("llama-server Binary gefunden: %s", binPath)
			}
		}
	}

	if s.config.BinaryPath == "" {
		return fmt.Errorf("llama-server Binary nicht gefunden")
	}

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return fmt.Errorf("Modell nicht gefunden: %s", modelPath)
	}

	// VRAM-Bedarf schätzen
	requiredVRAM := EstimateModelVRAM(modelPath)
	gpuLayers := s.config.GPULayers

	// VRAM-Strategie anwenden
	switch s.config.VRAMStrategy {
	case StrategyAlwaysClear:
		// Immer VRAM löschen vor Modell-Laden
		log.Printf("VRAM-Strategie: AlwaysClear - lösche VRAM vor Laden")
		if s.running {
			s.Stop()
			time.Sleep(1 * time.Second)
		}
		ClearVRAM()

	case StrategySmartSwap:
		// Nur VRAM löschen wenn nötig (Standard)
		log.Printf("VRAM-Strategie: SmartSwap - verwende %d GPU-Layer, benötigt ~%dMB VRAM", gpuLayers, requiredVRAM)
		if err := s.EnsureVRAMAvailable(requiredVRAM); err != nil {
			log.Printf("WARNUNG: VRAM-Prüfung fehlgeschlagen: %v", err)
		}

	case StrategySmartOffload:
		// Automatisches Offloading berechnen
		gpuLayers = s.CalculateOptimalGPULayers(modelPath, requiredVRAM)
		log.Printf("VRAM-Strategie: SmartOffload - verwende %d GPU-Layer", gpuLayers)

	case StrategyManual:
		// Keine automatische VRAM-Verwaltung
		log.Printf("VRAM-Strategie: Manual - keine automatische VRAM-Verwaltung")

	default:
		// Fallback auf SmartSwap
		if err := s.EnsureVRAMAvailable(requiredVRAM); err != nil {
			log.Printf("WARNUNG: VRAM-Prüfung fehlgeschlagen: %v", err)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("Server läuft bereits")
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFunc = cancel

	// Kommando vorbereiten
	args := []string{
		"-m", modelPath,
		"--port", fmt.Sprintf("%d", s.config.Port),
		"--host", s.config.Host,
		"-ngl", fmt.Sprintf("%d", gpuLayers),
		"-c", fmt.Sprintf("%d", s.config.ContextSize),
		"--jinja", // Aktiviert natives Function Calling für Qwen, Llama 3.x, etc.
	}

	if s.config.Threads > 0 {
		args = append(args, "-t", fmt.Sprintf("%d", s.config.Threads))
	}

	// Memory-Optimierungen
	// mmap ist standardmäßig aktiviert in llama.cpp, daher --no-mmap wenn deaktiviert
	if !s.config.UseMmap {
		args = append(args, "--no-mmap")
		log.Printf("mmap deaktiviert: Modell wird traditionell geladen")
	}
	// mlock ist standardmäßig deaktiviert, --mlock aktiviert es
	if s.config.UseMlock {
		args = append(args, "--mlock")
		log.Printf("mlock aktiviert: Modell wird im RAM gepinnt (kein Swap)")
	}

	// Vision/Multimodal Support: --mmproj für LLaVA und ähnliche Modelle
	if s.config.MmprojPath != "" {
		if _, err := os.Stat(s.config.MmprojPath); err == nil {
			args = append(args, "--mmproj", s.config.MmprojPath)
			s.config.VisionEnabled = true
			log.Printf("🖼️ Vision aktiviert: mmproj=%s", s.config.MmprojPath)
		} else {
			log.Printf("⚠️ mmproj-Datei nicht gefunden: %s", s.config.MmprojPath)
		}
	} else {
		// Automatisch nach mmproj suchen wenn Modell "llava" oder "vision" im Namen hat
		modelNameLower := strings.ToLower(filepath.Base(modelPath))
		if strings.Contains(modelNameLower, "llava") || strings.Contains(modelNameLower, "vision") || strings.Contains(modelNameLower, "minicpm") {
			mmprojPath := s.findMmprojForModel(modelPath)
			if mmprojPath != "" {
				args = append(args, "--mmproj", mmprojPath)
				s.config.MmprojPath = mmprojPath
				s.config.VisionEnabled = true
				log.Printf("🖼️ Vision automatisch aktiviert: mmproj=%s", mmprojPath)
			} else {
				log.Printf("⚠️ Vision-Modell erkannt, aber kein mmproj gefunden. Vision deaktiviert.")
			}
		}
	}

	s.cmd = exec.CommandContext(ctx, s.config.BinaryPath, args...)

	// Library Path setzen (wichtig für libmtmd.so und andere llama.cpp Libraries)
	if s.config.LibraryPath != "" {
		// Bestehenden LD_LIBRARY_PATH erweitern, nicht ersetzen
		existingLdPath := os.Getenv("LD_LIBRARY_PATH")
		newLdPath := s.config.LibraryPath
		if existingLdPath != "" {
			newLdPath = s.config.LibraryPath + ":" + existingLdPath
		}
		// Umgebung kopieren und LD_LIBRARY_PATH setzen/überschreiben
		env := os.Environ()
		ldPathSet := false
		for i, e := range env {
			if strings.HasPrefix(e, "LD_LIBRARY_PATH=") {
				env[i] = "LD_LIBRARY_PATH=" + newLdPath
				ldPathSet = true
				break
			}
		}
		if !ldPathSet {
			env = append(env, "LD_LIBRARY_PATH="+newLdPath)
		}
		s.cmd.Env = env
		s.cmd.Dir = s.config.LibraryPath
		log.Printf("LD_LIBRARY_PATH gesetzt: %s", newLdPath)
	}

	// Output loggen
	s.cmd.Stdout = os.Stdout
	s.cmd.Stderr = os.Stderr

	log.Printf("Starte llama-server: %s %v", s.config.BinaryPath, args)

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("llama-server starten fehlgeschlagen: %w", err)
	}

	s.running = true
	s.modelName = filepath.Base(modelPath)
	s.config.ModelPath = modelPath

	// Warten bis Server bereit ist
	go func() {
		for i := 0; i < 60; i++ { // Max 60 Sekunden warten
			time.Sleep(time.Second)
			if s.IsHealthy() {
				log.Printf("llama-server ist bereit auf Port %d", s.config.Port)
				// Heartbeat-Monitor starten sobald Server bereit ist
				s.startHeartbeatMonitor()
				return
			}
		}
		log.Printf("WARNUNG: llama-server antwortet nicht nach 60 Sekunden")
	}()

	// Auf Prozess-Ende warten (im Hintergrund)
	go func() {
		err := s.cmd.Wait()
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		// Heartbeat-Monitor stoppen wenn Prozess endet
		if s.heartbeatCancel != nil {
			s.heartbeatCancel()
		}
		if err != nil && ctx.Err() == nil {
			log.Printf("llama-server beendet mit Fehler: %v", err)
		}
	}()

	return nil
}

// CalculateOptimalGPULayers berechnet die optimale Anzahl GPU-Layer basierend auf VRAM
func (s *Server) CalculateOptimalGPULayers(modelPath string, requiredVRAM int64) int {
	info := GetVRAMInfo()
	if !info.Available {
		log.Printf("nvidia-smi nicht verfügbar, verwende Standard GPU-Layer: %d", s.config.GPULayers)
		return s.config.GPULayers
	}

	// Verfügbarer VRAM minus Reserve
	availableVRAM := info.FreeMB - int64(s.config.VRAMReserve)
	if availableVRAM < 0 {
		availableVRAM = 0
	}

	// Wenn genug VRAM vorhanden, alle Layer auf GPU
	if availableVRAM >= requiredVRAM {
		log.Printf("Genug VRAM verfügbar (%dMB >= %dMB), alle Layer auf GPU", availableVRAM, requiredVRAM)
		return 99
	}

	// Anzahl Layer basierend auf Modellgröße schätzen
	// Typische Layer-Anzahl pro Modellgröße:
	// 7B = 32 Layer, 13B = 40 Layer, 70B = 80 Layer
	modelInfo, _ := os.Stat(modelPath)
	modelSizeGB := float64(modelInfo.Size()) / (1024 * 1024 * 1024)

	var totalLayers int
	switch {
	case modelSizeGB < 3:
		totalLayers = 24 // 1-3B Modelle
	case modelSizeGB < 6:
		totalLayers = 32 // 7B Modelle
	case modelSizeGB < 10:
		totalLayers = 40 // 13B Modelle
	case modelSizeGB < 25:
		totalLayers = 48 // 32B Modelle
	default:
		totalLayers = 80 // 70B+ Modelle
	}

	// Berechne Anteil der Layer die auf GPU passen
	ratio := float64(availableVRAM) / float64(requiredVRAM)
	if ratio > 1.0 {
		ratio = 1.0
	}

	optimalLayers := int(float64(totalLayers) * ratio)
	if optimalLayers < 1 {
		optimalLayers = 1 // Mindestens 1 Layer auf GPU für Beschleunigung
	}

	log.Printf("SmartOffload: %dMB VRAM verfügbar, %dMB benötigt, %.0f%% auf GPU (%d von %d Layer)",
		availableVRAM, requiredVRAM, ratio*100, optimalLayers, totalLayers)

	return optimalLayers
}

// Stop stoppt den llama-server (auch extern gestartete)
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Heartbeat-Monitor stoppen
	if s.heartbeatCancel != nil {
		s.heartbeatCancel()
		s.heartbeatCancel = nil
	}

	// Wenn intern gestartet, normales Stoppen
	if s.running {
		if s.cancelFunc != nil {
			s.cancelFunc()
		}
		if s.cmd != nil && s.cmd.Process != nil {
			s.cmd.Process.Kill()
		}
		s.running = false
		log.Printf("llama-server gestoppt (intern)")
	}

	// Auch extern gestartete llama-server auf dem konfigurierten Port beenden
	// Finde Prozess auf Port und beende ihn
	if s.IsHealthy() {
		log.Printf("Extern gestarteter llama-server auf Port %d gefunden, beende...", s.config.Port)

		// Finde PID des Prozesses auf dem Port
		cmd := exec.Command("fuser", "-k", fmt.Sprintf("%d/tcp", s.config.Port))
		if err := cmd.Run(); err != nil {
			// Fallback: pkill llama-server
			log.Printf("fuser fehlgeschlagen, versuche pkill: %v", err)
			exec.Command("pkill", "-f", "llama-server").Run()
		}
		log.Printf("Extern gestarteter llama-server beendet")
	}

	// Kurz warten damit Prozess VRAM freigeben kann
	time.Sleep(500 * time.Millisecond)

	return nil
}

// VRAMInfo enthält Informationen über den GPU-Speicher
type VRAMInfo struct {
	TotalMB     int64 `json:"totalMB"`
	UsedMB      int64 `json:"usedMB"`
	FreeMB      int64 `json:"freeMB"`
	PercentUsed int   `json:"percentUsed"`
	GPUName     string `json:"gpuName"`
	Available   bool   `json:"available"`
}

// GetVRAMInfo gibt Informationen über den verfügbaren GPU-Speicher zurück
func GetVRAMInfo() VRAMInfo {
	info := VRAMInfo{Available: false}

	// nvidia-smi aufrufen
	cmd := exec.Command("nvidia-smi", "--query-gpu=name,memory.total,memory.used,memory.free", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("nvidia-smi nicht verfügbar: %v", err)
		return info
	}

	// Output parsen: "NVIDIA GeForce RTX 3060, 12288, 5000, 7288"
	line := strings.TrimSpace(string(output))
	parts := strings.Split(line, ", ")
	if len(parts) >= 4 {
		info.GPUName = parts[0]
		info.TotalMB, _ = strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		info.UsedMB, _ = strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64)
		info.FreeMB, _ = strconv.ParseInt(strings.TrimSpace(parts[3]), 10, 64)
		if info.TotalMB > 0 {
			info.PercentUsed = int((info.UsedMB * 100) / info.TotalMB)
		}
		info.Available = true
	}

	return info
}

// GetGPUProcesses gibt laufende Prozesse auf der GPU zurück
func GetGPUProcesses() []GPUProcess {
	var processes []GPUProcess

	cmd := exec.Command("nvidia-smi", "--query-compute-apps=pid,name,used_memory", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return processes
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Split(line, ", ")
		if len(parts) >= 3 {
			pid, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
			memMB, _ := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64)
			processes = append(processes, GPUProcess{
				PID:     pid,
				Name:    strings.TrimSpace(parts[1]),
				MemoryMB: memMB,
			})
		}
	}

	return processes
}

// GPUProcess repräsentiert einen GPU-Prozess
type GPUProcess struct {
	PID      int    `json:"pid"`
	Name     string `json:"name"`
	MemoryMB int64  `json:"memoryMB"`
}

// ClearVRAM beendet alle llama-server Prozesse und gibt VRAM frei
func ClearVRAM() error {
	log.Printf("Räume VRAM auf...")

	// Alle llama-server Prozesse finden und beenden
	processes := GetGPUProcesses()
	killedCount := 0

	for _, proc := range processes {
		// llama-server oder llama.cpp Prozesse beenden
		if strings.Contains(strings.ToLower(proc.Name), "llama") ||
		   strings.Contains(strings.ToLower(proc.Name), "ggml") {
			log.Printf("Beende GPU-Prozess: PID=%d, Name=%s, Memory=%dMB", proc.PID, proc.Name, proc.MemoryMB)
			killCmd := exec.Command("kill", "-9", strconv.Itoa(proc.PID))
			killCmd.Run()
			killedCount++
		}
	}

	// Auch per pkill alle llama-server beenden
	pkillCmd := exec.Command("pkill", "-9", "-f", "llama-server")
	pkillCmd.Run()

	// Warten bis VRAM freigegeben ist
	time.Sleep(2 * time.Second)

	// Prüfen ob VRAM frei ist
	info := GetVRAMInfo()
	if info.Available {
		log.Printf("VRAM nach Bereinigung: %dMB frei von %dMB (%.1f%% belegt)",
			info.FreeMB, info.TotalMB, float64(info.PercentUsed))
	}

	if killedCount > 0 {
		log.Printf("%d GPU-Prozesse beendet", killedCount)
	}

	return nil
}

// EnsureVRAMAvailable stellt sicher, dass genug VRAM für ein Modell verfügbar ist
func (s *Server) EnsureVRAMAvailable(requiredMB int64) error {
	info := GetVRAMInfo()
	if !info.Available {
		log.Printf("VRAM-Prüfung nicht möglich (nvidia-smi nicht verfügbar)")
		return nil // Ohne nvidia-smi einfach weitermachen
	}

	log.Printf("VRAM-Status: %dMB frei von %dMB, benötigt: ~%dMB", info.FreeMB, info.TotalMB, requiredMB)

	// Wenn nicht genug VRAM frei ist, aufräumen
	if info.FreeMB < requiredMB {
		log.Printf("Nicht genug VRAM frei (%dMB < %dMB), räume auf...", info.FreeMB, requiredMB)

		// Erst den eigenen Server stoppen (falls läuft)
		if s.running {
			s.Stop()
			time.Sleep(1 * time.Second)
		}

		// Dann alle anderen llama Prozesse
		ClearVRAM()

		// Nochmal prüfen
		info = GetVRAMInfo()
		if info.FreeMB < requiredMB {
			return fmt.Errorf("VRAM konnte nicht freigegeben werden: %dMB frei, %dMB benötigt", info.FreeMB, requiredMB)
		}

		log.Printf("VRAM erfolgreich freigegeben: %dMB frei", info.FreeMB)
	}

	return nil
}

// EstimateModelVRAM schätzt den VRAM-Bedarf eines Modells basierend auf Dateigröße
func EstimateModelVRAM(modelPath string) int64 {
	info, err := os.Stat(modelPath)
	if err != nil {
		return 6000 // Standard: 6GB für mittleres Modell
	}

	fileSizeMB := info.Size() / (1024 * 1024)

	// Faustregeln für GGUF-Modelle:
	// - Q4_K_M: VRAM ≈ Dateigröße × 1.1 + 500MB Overhead
	// - Q5_K_M: VRAM ≈ Dateigröße × 1.05 + 500MB Overhead
	// - Q8_0:   VRAM ≈ Dateigröße × 1.0 + 500MB Overhead

	// Konservative Schätzung: Dateigröße × 1.2 + 1GB Overhead für Context
	estimatedMB := int64(float64(fileSizeMB)*1.2) + 1024

	log.Printf("Modell %s: %.1f GB, geschätzter VRAM-Bedarf: %dMB",
		filepath.Base(modelPath), float64(fileSizeMB)/1024, estimatedMB)

	return estimatedMB
}

// IsRunning prüft ob der Server läuft
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// IsHealthy prüft ob der Server antwortet
func (s *Server) IsHealthy() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/health", s.config.Port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// GetStatus gibt den aktuellen Status zurück
func (s *Server) GetStatus() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Prüfe ob Server gesund ist (auch wenn extern gestartet)
	healthy := s.IsHealthy()
	// Wenn Server antwortet, läuft er auch (auch wenn extern gestartet)
	running := s.running || healthy

	return Status{
		Running:     running,
		Healthy:     healthy,
		Port:        s.config.Port,
		ModelName:   s.modelName,
		ModelPath:   s.config.ModelPath,
		BinaryPath:  s.config.BinaryPath,
		BinaryFound: s.config.BinaryPath != "",
		ContextSize: s.config.ContextSize,
		GPULayers:   s.config.GPULayers,
	}
}

// Restart startet den Server mit dem gleichen Modell neu
func (s *Server) Restart() error {
	modelPath := s.config.ModelPath
	if modelPath == "" {
		return fmt.Errorf("Kein Modell geladen")
	}

	if err := s.Stop(); err != nil {
		return err
	}

	time.Sleep(2 * time.Second) // Kurz warten

	return s.Start(modelPath)
}

// SwitchModel wechselt zu einem anderen Modell
func (s *Server) SwitchModel(modelPath string) error {
	if err := s.Stop(); err != nil {
		return err
	}

	time.Sleep(2 * time.Second)

	return s.Start(modelPath)
}

// ModelInfo enthält Informationen über ein GGUF-Modell
type ModelInfo struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}

// Status enthält den Server-Status
type Status struct {
	Running     bool   `json:"running"`
	Healthy     bool   `json:"healthy"`
	Port        int    `json:"port"`
	ModelName   string `json:"modelName"`
	ModelPath   string `json:"modelPath"`
	BinaryPath  string `json:"binaryPath"`
	BinaryFound bool   `json:"binaryFound"`
	ContextSize int    `json:"contextSize"`
	GPULayers   int    `json:"gpuLayers"`
}

// DownloadModel lädt ein GGUF-Modell von Hugging Face herunter
// Mit Resume-Unterstützung: Unterbrochene Downloads werden fortgesetzt
func (s *Server) DownloadModel(url, filename string, progressChan chan<- DownloadProgress) error {
	destPath := filepath.Join(s.config.ModelsDir, filename)
	tempPath := destPath + ".downloading"

	// Verzeichnis erstellen falls nötig
	if err := os.MkdirAll(s.config.ModelsDir, 0755); err != nil {
		return fmt.Errorf("Verzeichnis erstellen: %w", err)
	}

	// HTTP Client mit Timeouts - folgt Redirects automatisch
	transport := &http.Transport{
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout:       90 * time.Second,
	}
	client := &http.Client{
		Timeout:   0, // Kein globales Timeout für große Downloads
		Transport: transport,
	}

	// HEAD-Request um Gesamtgröße zu ermitteln (für Resume-Check)
	// WICHTIG: HuggingFace gibt 302 Redirect zurück, wir müssen der finalen URL folgen
	var totalSize int64 = 0
	var finalURL = url

	// Manuell Redirects folgen für HEAD-Requests
	// (Go's http.Client folgt HEAD-Redirects nicht automatisch wie bei GET)
	maxRedirects := 10
	currentURL := url
	for i := 0; i < maxRedirects; i++ {
		headReq, err := http.NewRequest("HEAD", currentURL, nil)
		if err != nil {
			return fmt.Errorf("HEAD-Request erstellen: %w", err)
		}
		headReq.Header.Set("User-Agent", "Fleet-Navigator/0.7.0")

		// Client der nicht automatisch redirected
		noRedirectClient := &http.Client{
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // Nicht automatisch folgen
			},
		}

		headResp, err := noRedirectClient.Do(headReq)
		if err != nil {
			log.Printf("HEAD-Request fehlgeschlagen für %s: %v", currentURL, err)
			break
		}
		headResp.Body.Close()

		// Prüfe auf Redirect
		if headResp.StatusCode >= 300 && headResp.StatusCode < 400 {
			location := headResp.Header.Get("Location")
			if location == "" {
				log.Printf("Redirect ohne Location-Header bei %s", currentURL)
				break
			}
			log.Printf("[HuggingFace] Folge Redirect %d: %s", headResp.StatusCode, location)
			currentURL = location
			finalURL = location
			continue
		}

		// Erfolg - Content-Length lesen
		if headResp.StatusCode == http.StatusOK {
			totalSize = headResp.ContentLength
			finalURL = currentURL
			log.Printf("[HuggingFace] Finale URL: %s, Größe: %d bytes (%.2f MB)",
				finalURL, totalSize, float64(totalSize)/1024/1024)
		}
		break
	}

	if totalSize <= 0 {
		log.Printf("Warnung: Konnte Content-Length nicht ermitteln (möglicherweise kein Resume)")
	}

	// Für den eigentlichen Download die finale URL verwenden
	url = finalURL

	// Prüfe ob teilweiser Download existiert (Resume-Logik)
	var resumeOffset int64 = 0
	if partialInfo, err := os.Stat(tempPath); err == nil && totalSize > 0 {
		partialSize := partialInfo.Size()

		if partialSize > totalSize {
			// Korrupt: Lokale Datei größer als Server-Angabe → löschen
			log.Printf("Korrupte Temp-Datei erkannt (%d > %d), lösche...", partialSize, totalSize)
			os.Remove(tempPath)
		} else if partialSize == totalSize {
			// Vollständig heruntergeladen, nur umbenennen
			log.Printf("Download bereits vollständig, benenne um: %s", filename)
			if err := os.Rename(tempPath, destPath); err != nil {
				return fmt.Errorf("Umbenennen: %w", err)
			}
			// 100% Progress senden
			if progressChan != nil {
				progressChan <- DownloadProgress{
					Downloaded: totalSize,
					Total:      totalSize,
					Percent:    100,
					Filename:   filename,
					Resumed:    true,
				}
			}
			return nil
		} else {
			// Resume möglich
			resumeOffset = partialSize
			log.Printf("Resume Download: %s bei %.1f%% (%.2f MB von %.2f MB)",
				filename,
				float64(resumeOffset)/float64(totalSize)*100,
				float64(resumeOffset)/1024/1024,
				float64(totalSize)/1024/1024)
		}
	}

	// HTTP Request mit optionalem Range-Header
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("Request erstellen: %w", err)
	}
	req.Header.Set("User-Agent", "Fleet-Navigator/0.7.0")

	// Range-Header für Resume
	if resumeOffset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", resumeOffset))
		log.Printf("Range-Header gesetzt: bytes=%d-", resumeOffset)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Download starten: %w", err)
	}
	defer resp.Body.Close()

	// Status prüfen
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("HTTP Fehler: %s", resp.Status)
	}

	// Bei Resume: 206 Partial Content erwartet
	if resumeOffset > 0 && resp.StatusCode != http.StatusPartialContent {
		log.Printf("Server unterstützt kein Resume (Status: %d), starte neu", resp.StatusCode)
		resumeOffset = 0
		os.Remove(tempPath)
	}

	// Berechne Gesamt-Größe wenn nicht bekannt
	if totalSize <= 0 {
		totalSize = resp.ContentLength
		if resumeOffset > 0 {
			totalSize += resumeOffset // Bei Resume nur noch der Rest
		}
	}

	// Datei öffnen/erstellen
	var out *os.File
	if resumeOffset > 0 {
		out, err = os.OpenFile(tempPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			// Fallback: Neu erstellen
			log.Printf("Konnte Temp-Datei nicht zum Anhängen öffnen, starte neu: %v", err)
			resumeOffset = 0
			out, err = os.Create(tempPath)
		}
	} else {
		out, err = os.Create(tempPath)
	}
	if err != nil {
		return fmt.Errorf("Datei erstellen: %w", err)
	}
	defer out.Close()

	// Mit Progress-Tracking kopieren
	downloaded := resumeOffset
	buf := make([]byte, 64*1024) // 64KB Buffer für bessere Performance
	lastLogPercent := -1         // Für periodisches Logging

	log.Printf("[HuggingFace] Download läuft: %s (%.2f MB)", filename, float64(totalSize)/1024/1024)

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				return fmt.Errorf("Schreiben: %w", writeErr)
			}
			downloaded += int64(n)

			percent := float64(downloaded) / float64(totalSize) * 100
			percentInt := int(percent)

			// Log alle 5%
			if percentInt >= lastLogPercent+5 {
				lastLogPercent = percentInt
				log.Printf("[HuggingFace] %s: %.1f%% (%.2f / %.2f MB)",
					filename, percent, float64(downloaded)/1024/1024, float64(totalSize)/1024/1024)
			}

			if progressChan != nil && totalSize > 0 {
				progress := DownloadProgress{
					Downloaded: downloaded,
					Total:      totalSize,
					Percent:    percent,
					Filename:   filename,
					Resumed:    resumeOffset > 0,
				}
				select {
				case progressChan <- progress:
				default:
				}
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			// Bei Netzwerkfehler: Temp-Datei behalten für Resume!
			out.Close()
			log.Printf("Download unterbrochen bei %.1f%%, Temp-Datei behalten für Resume: %s",
				float64(downloaded)/float64(totalSize)*100, tempPath)
			return fmt.Errorf("Lesen unterbrochen: %w (Resume möglich)", readErr)
		}
	}

	// Erfolgreich: Umbenennen
	out.Close()
	if err := os.Rename(tempPath, destPath); err != nil {
		return fmt.Errorf("Umbenennen: %w", err)
	}

	if resumeOffset > 0 {
		log.Printf("Modell heruntergeladen (resumed): %s (%d MB)", filename, downloaded/1024/1024)
	} else {
		log.Printf("Modell heruntergeladen: %s (%d MB)", filename, downloaded/1024/1024)
	}
	return nil
}

// DownloadProgress enthält Fortschrittsinformationen
type DownloadProgress struct {
	Downloaded int64   `json:"downloaded"`
	Total      int64   `json:"total"`
	Percent    float64 `json:"percent"`
	Filename   string  `json:"filename"`
	Resumed    bool    `json:"resumed,omitempty"` // True wenn Download fortgesetzt wurde
}

// RecommendedModels gibt empfohlene Modelle zum Download zurück
func GetRecommendedModels() []RecommendedModel {
	return []RecommendedModel{
		{
			Name:        "Qwen2.5-7B-Instruct-Q5_K_M",
			Description: "Exzellentes Allround-Modell mit gutem Deutsch",
			Size:        "5.2 GB",
			URL:         "https://huggingface.co/bartowski/Qwen2.5-7B-Instruct-GGUF/resolve/main/Qwen2.5-7B-Instruct-Q5_K_M.gguf",
			Filename:    "Qwen2.5-7B-Instruct-Q5_K_M.gguf",
			Category:    "chat",
		},
		{
			Name:        "Qwen2.5-Coder-7B-Instruct-Q4_K_M",
			Description: "Spezialisiert auf Code-Generierung",
			Size:        "4.4 GB",
			URL:         "https://huggingface.co/bartowski/Qwen2.5-Coder-7B-Instruct-GGUF/resolve/main/Qwen2.5-Coder-7B-Instruct-Q4_K_M.gguf",
			Filename:    "qwen2.5-coder-7b-instruct-q4_k_m.gguf",
			Category:    "code",
		},
		{
			Name:        "Llama-3.2-3B-Instruct-Q4_K_M",
			Description: "Schnelles, kompaktes Modell von Meta",
			Size:        "2.0 GB",
			URL:         "https://huggingface.co/bartowski/Llama-3.2-3B-Instruct-GGUF/resolve/main/Llama-3.2-3B-Instruct-Q4_K_M.gguf",
			Filename:    "Llama-3.2-3B-Instruct-Q4_K_M.gguf",
			Category:    "fast",
		},
		{
			Name:        "LLaVA-v1.6-Mistral-7B-Q4_K_M",
			Description: "Vision-Modell für Bildanalyse",
			Size:        "4.4 GB",
			URL:         "https://huggingface.co/cjpais/llava-v1.6-mistral-7b-gguf/resolve/main/llava-v1.6-mistral-7b.Q4_K_M.gguf",
			Filename:    "llava-v1.6-mistral-7b.Q4_K_M.gguf",
			Category:    "vision",
		},
	}
}

// RecommendedModel enthält Informationen über ein empfohlenes Modell
type RecommendedModel struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Size        string `json:"size"`
	URL         string `json:"url"`
	Filename    string `json:"filename"`
	Category    string `json:"category"`
}

// GetConfig gibt die aktuelle Konfiguration zurück
func (s *Server) GetConfig() Config {
	return s.config
}

// SaveConfig speichert die Konfiguration
func (s *Server) SaveConfig(dataDir string) error {
	configPath := filepath.Join(dataDir, "llamaserver.json")
	data, err := json.MarshalIndent(s.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

// LoadConfig lädt die Konfiguration
func LoadConfig(dataDir string) (*Config, error) {
	configPath := filepath.Join(dataDir, "llamaserver.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// VRAMStrategyInfo enthält Informationen über eine VRAM-Strategie
type VRAMStrategyInfo struct {
	ID          VRAMStrategy `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Recommended bool         `json:"recommended"`
}

// GetAvailableVRAMStrategies gibt alle verfügbaren VRAM-Strategien zurück
func GetAvailableVRAMStrategies() []VRAMStrategyInfo {
	return []VRAMStrategyInfo{
		{
			ID:          StrategySmartSwap,
			Name:        "Smart Swap",
			Description: "Löscht VRAM nur wenn nicht genug Speicher frei ist. Beste Balance zwischen Geschwindigkeit und Zuverlässigkeit.",
			Recommended: true,
		},
		{
			ID:          StrategyAlwaysClear,
			Name:        "Always Clear",
			Description: "Löscht immer den VRAM vor dem Laden eines neuen Modells. Am sichersten, aber mit kurzer Downtime.",
			Recommended: false,
		},
		{
			ID:          StrategySmartOffload,
			Name:        "Smart Offload",
			Description: "Berechnet automatisch wie viele Layer auf GPU passen. Ermöglicht große Modelle (70B+) auf kleinen GPUs.",
			Recommended: false,
		},
		{
			ID:          StrategyManual,
			Name:        "Manual",
			Description: "Keine automatische VRAM-Verwaltung. Für erfahrene Benutzer die alles selbst kontrollieren wollen.",
			Recommended: false,
		},
	}
}

// SetVRAMStrategy setzt die VRAM-Strategie
func (s *Server) SetVRAMStrategy(strategy VRAMStrategy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config.VRAMStrategy = strategy
	log.Printf("VRAM-Strategie geändert auf: %s", strategy)
}

// GetVRAMStrategy gibt die aktuelle VRAM-Strategie zurück
func (s *Server) GetVRAMStrategy() VRAMStrategy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.VRAMStrategy
}

// SetVRAMReserve setzt die VRAM-Reserve in MB
func (s *Server) SetVRAMReserve(reserveMB int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config.VRAMReserve = reserveMB
	log.Printf("VRAM-Reserve geändert auf: %dMB", reserveMB)
}

// GetVRAMReserve gibt die aktuelle VRAM-Reserve zurück
func (s *Server) GetVRAMReserve() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.VRAMReserve
}

// SetUseMmap aktiviert/deaktiviert Memory-Mapped I/O
func (s *Server) SetUseMmap(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config.UseMmap = enabled
	log.Printf("mmap %s", map[bool]string{true: "aktiviert", false: "deaktiviert"}[enabled])
}

// GetUseMmap gibt zurück ob mmap aktiviert ist
func (s *Server) GetUseMmap() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.UseMmap
}

// SetUseMlock aktiviert/deaktiviert Memory-Locking
func (s *Server) SetUseMlock(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config.UseMlock = enabled
	log.Printf("mlock %s", map[bool]string{true: "aktiviert", false: "deaktiviert"}[enabled])
}

// GetUseMlock gibt zurück ob mlock aktiviert ist
func (s *Server) GetUseMlock() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.UseMlock
}

// SetPort setzt den llama-server Port
func (s *Server) SetPort(port int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config.Port = port
	log.Printf("llama-server Port auf %d gesetzt", port)
}

// SetContextSize setzt die Context-Größe
func (s *Server) SetContextSize(size int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config.ContextSize = size
	log.Printf("Context-Size auf %d gesetzt", size)
}

// GetContextSize gibt die aktuelle Context-Größe zurück
func (s *Server) GetContextSize() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.ContextSize
}

// RestartWithContextSize ändert die Context-Größe und startet den Server neu
// Gibt die geschätzte Restart-Dauer in Sekunden zurück
func (s *Server) RestartWithContextSize(newContextSize int) (int, error) {
	oldContext := s.GetContextSize()

	// Wenn Context gleich bleibt, kein Neustart nötig
	if oldContext == newContextSize {
		log.Printf("Context-Größe unverändert (%d), kein Neustart erforderlich", oldContext)
		return 0, nil
	}

	log.Printf("Context-Wechsel: %d → %d, starte Server neu...", oldContext, newContextSize)

	// Context-Größe setzen
	s.SetContextSize(newContextSize)

	// Server neustarten (dauert je nach Modellgröße 3-15 Sekunden)
	err := s.Restart()
	if err != nil {
		// Bei Fehler: alten Context wiederherstellen
		s.SetContextSize(oldContext)
		return 0, fmt.Errorf("Server-Neustart fehlgeschlagen: %w", err)
	}

	// Geschätzte Dauer basierend auf Context-Größe (größerer Context = längerer Start)
	estimatedSeconds := 5
	if newContextSize > 32768 {
		estimatedSeconds = 8
	}
	if newContextSize > 65536 {
		estimatedSeconds = 12
	}

	log.Printf("Server mit Context %d neugestartet (geschätzte Dauer: %ds)", newContextSize, estimatedSeconds)
	return estimatedSeconds, nil
}

// SetGPULayers setzt die Anzahl der GPU-Layers
func (s *Server) SetGPULayers(layers int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config.GPULayers = layers
	log.Printf("GPU-Layers auf %d gesetzt", layers)
}

// VRAMSettings enthält alle VRAM-bezogenen Einstellungen
type VRAMSettings struct {
	Strategy            VRAMStrategy       `json:"strategy"`
	ReserveMB           int                `json:"reserveMB"`
	CurrentVRAM         VRAMInfo           `json:"currentVram"`
	AvailableStrategies []VRAMStrategyInfo `json:"availableStrategies"`
	UseMmap             bool               `json:"useMmap"`  // Memory-Mapped I/O
	UseMlock            bool               `json:"useMlock"` // Memory-Locking (kein Swap)
}

// GetVRAMSettings gibt alle VRAM-Einstellungen zurück
func (s *Server) GetVRAMSettings() VRAMSettings {
	return VRAMSettings{
		Strategy:            s.GetVRAMStrategy(),
		ReserveMB:           s.GetVRAMReserve(),
		CurrentVRAM:         GetVRAMInfo(),
		AvailableStrategies: GetAvailableVRAMStrategies(),
		UseMmap:             s.GetUseMmap(),
		UseMlock:            s.GetUseMlock(),
	}
}

// ChatMessage repräsentiert eine Nachricht im Chat
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// SamplingParams enthält die Sampling-Parameter für LLM-Anfragen
type SamplingParams struct {
	Temperature float64 // 0.0-2.0, Default: 0.7
	TopP        float64 // 0.0-1.0, Default: 0.9
	MaxTokens   int     // Max Tokens für Antwort, Default: 4096
}

// Tool repräsentiert ein verfügbares Tool für Function Calling
type Tool struct {
	Type     string       `json:"type"` // "function"
	Function ToolFunction `json:"function"`
}

// ToolFunction definiert eine aufrufbare Funktion
type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCall repräsentiert einen Tool-Aufruf vom LLM
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // "function"
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"` // JSON-String
	} `json:"function"`
}

// ChatResponse repräsentiert eine vollständige Chat-Antwort (nicht-streaming)
type ChatResponse struct {
	Content      string     `json:"content"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	FinishReason string     `json:"finish_reason"`
}

// DefaultSamplingParams gibt die Standard-Sampling-Parameter zurück
func DefaultSamplingParams() SamplingParams {
	return SamplingParams{
		Temperature: 0.7,
		TopP:        0.9,
		MaxTokens:   4096,
	}
}

// DefaultCoderTools gibt die Standard-Tools für Coder zurück
func DefaultCoderTools() []Tool {
	return []Tool{
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "write_file",
				Description: "Erstellt oder überschreibt eine Datei mit dem angegebenen Inhalt",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path":    map[string]string{"type": "string", "description": "Dateipfad"},
						"content": map[string]string{"type": "string", "description": "Dateiinhalt"},
					},
					"required": []string{"path", "content"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "read_file",
				Description: "Liest den Inhalt einer Datei",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]string{"type": "string", "description": "Dateipfad"},
					},
					"required": []string{"path"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "list_files",
				Description: "Listet Dateien in einem Verzeichnis auf",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path":      map[string]string{"type": "string", "description": "Verzeichnispfad"},
						"pattern":   map[string]string{"type": "string", "description": "Glob-Pattern z.B. *.md"},
						"recursive": map[string]string{"type": "boolean", "description": "Rekursiv suchen"},
					},
					"required": []string{"path"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "shell",
				Description: "Führt einen Shell-Befehl aus (nicht interaktiv)",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"command": map[string]string{"type": "string", "description": "Auszuführender Befehl"},
					},
					"required": []string{"command"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "search",
				Description: "Sucht nach Text in Dateien",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"pattern": map[string]string{"type": "string", "description": "Suchtext oder Regex"},
						"path":    map[string]string{"type": "string", "description": "Suchpfad"},
					},
					"required": []string{"pattern"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "git",
				Description: "Führt Git-Operationen aus",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"action": map[string]string{"type": "string", "description": "Git-Aktion: status, diff, log, branch"},
					},
					"required": []string{"action"},
				},
			},
		},
	}
}

// adaptMessagesForModel passt die Messages für das aktuelle Modell an
// Verwendet den Template-Adapter falls vorhanden, sonst Fallback-Logik
func (s *Server) adaptMessagesForModel(messages []ChatMessage) []ChatMessage {
	s.mu.RLock()
	adapter := s.templateAdapter
	modelName := s.modelName
	s.mu.RUnlock()

	// Wenn Template-Adapter gesetzt, diesen verwenden
	if adapter != nil {
		return adapter.AdaptMessages(modelName, messages)
	}

	// Fallback: Hardcoded Logik für Gemma (falls kein Adapter)
	if !strings.Contains(strings.ToLower(modelName), "gemma") {
		return messages
	}

	// Gemma-Modell: System-Prompt in User-Nachricht einbetten (Fallback)
	if len(messages) == 0 {
		return messages
	}

	var systemPrompt string
	var otherMessages []ChatMessage

	for _, msg := range messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
		} else {
			otherMessages = append(otherMessages, msg)
		}
	}

	if systemPrompt == "" {
		return messages
	}

	result := make([]ChatMessage, 0, len(otherMessages))
	systemPrepended := false

	for _, msg := range otherMessages {
		if msg.Role == "user" && !systemPrepended {
			enhancedContent := fmt.Sprintf(`[SYSTEM-ANWEISUNGEN - DU MUSST DIESE BEFOLGEN]
%s
[ENDE DER SYSTEM-ANWEISUNGEN]

Benutzer-Nachricht: %s`, systemPrompt, msg.Content)
			result = append(result, ChatMessage{
				Role:    "user",
				Content: enhancedContent,
			})
			systemPrepended = true
		} else {
			result = append(result, msg)
		}
	}

	if !systemPrepended && systemPrompt != "" {
		result = append([]ChatMessage{{
			Role:    "user",
			Content: "[SYSTEM-ANWEISUNGEN]\n" + systemPrompt + "\n[ENDE]",
		}}, result...)
	}

	log.Printf("🔄 Gemma-Modell (Fallback): System-Prompt eingebettet (%d Zeichen)", len(systemPrompt))
	return result
}

// StreamChat sendet eine Chat-Anfrage an den llama-server und streamt die Antwort
// Verwendet die OpenAI-kompatible API des llama-servers
func (s *Server) StreamChat(messages []ChatMessage, onChunk func(content string, done bool)) error {
	return s.StreamChatWithParams(messages, DefaultSamplingParams(), onChunk)
}

// StreamChatWithParams sendet eine Chat-Anfrage mit expliziten Sampling-Parametern
func (s *Server) StreamChatWithParams(messages []ChatMessage, params SamplingParams, onChunk func(content string, done bool)) error {
	if !s.IsRunning() || !s.IsHealthy() {
		return fmt.Errorf("llama-server ist nicht aktiv")
	}

	// Defaults setzen falls nicht gesetzt
	if params.Temperature == 0 {
		params.Temperature = 0.7
	}
	if params.TopP == 0 {
		params.TopP = 0.9
	}
	if params.MaxTokens == 0 {
		params.MaxTokens = 4096
	}

	// Für Gemma-Modelle: System-Prompt in User-Nachricht einbetten
	// Gemma unterstützt die "system" Rolle nicht wie andere Modelle
	processedMessages := s.adaptMessagesForModel(messages)

	// OpenAI-kompatibles Request-Format mit Sampling-Parametern
	requestBody := map[string]interface{}{
		"messages":    processedMessages,
		"stream":      true,
		"temperature": params.Temperature,
		"top_p":       params.TopP,
		"max_tokens":  params.MaxTokens,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("JSON-Fehler: %w", err)
	}

	url := fmt.Sprintf("http://localhost:%d/v1/chat/completions", s.config.Port)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("Request-Fehler: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Minute} // Längeres Timeout für Streaming
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("llama-server nicht erreichbar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("llama-server Fehler %d: %s", resp.StatusCode, string(body))
	}

	// SSE Stream lesen
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("Stream-Lesefehler: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// SSE Format: "data: {...}"
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			onChunk("", true)
			break
		}

		// OpenAI-Format parsen
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // Ungültige JSON-Zeilen ignorieren
		}

		if len(chunk.Choices) > 0 {
			content := chunk.Choices[0].Delta.Content
			done := chunk.Choices[0].FinishReason != nil
			if content != "" || done {
				onChunk(content, done)
			}
		}
	}

	return nil
}

// StreamChatWithTools sendet eine Chat-Anfrage mit Tool-Support
// Gibt Content und eventuelle ToolCalls zurück
func (s *Server) StreamChatWithTools(messages []ChatMessage, tools []Tool, onChunk func(content string, done bool)) (*ChatResponse, error) {
	if !s.IsRunning() || !s.IsHealthy() {
		return nil, fmt.Errorf("llama-server ist nicht aktiv")
	}

	params := DefaultSamplingParams()

	// Für Gemma-Modelle: System-Prompt in User-Nachricht einbetten
	processedMessages := s.adaptMessagesForModel(messages)

	// OpenAI-kompatibles Request-Format mit Tools
	requestBody := map[string]interface{}{
		"messages":    processedMessages,
		"stream":      true,
		"temperature": params.Temperature,
		"top_p":       params.TopP,
		"max_tokens":  params.MaxTokens,
	}

	// Tools nur hinzufügen wenn vorhanden
	if len(tools) > 0 {
		requestBody["tools"] = tools
		requestBody["tool_choice"] = "auto" // LLM entscheidet selbst
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("JSON-Fehler: %w", err)
	}

	url := fmt.Sprintf("http://localhost:%d/v1/chat/completions", s.config.Port)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("Request-Fehler: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llama-server nicht erreichbar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("llama-server Fehler %d: %s", resp.StatusCode, string(body))
	}

	// Response sammeln
	response := &ChatResponse{}
	var contentBuilder strings.Builder
	var toolCalls []ToolCall

	// SSE Stream lesen
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("Stream-Lesefehler: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			if onChunk != nil {
				onChunk("", true)
			}
			break
		}

		// OpenAI-Format mit Tool-Calls parsen
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 {
			choice := chunk.Choices[0]

			// Content sammeln
			if choice.Delta.Content != "" {
				contentBuilder.WriteString(choice.Delta.Content)
				if onChunk != nil {
					onChunk(choice.Delta.Content, false)
				}
			}

			// Tool-Calls sammeln
			if len(choice.Delta.ToolCalls) > 0 {
				toolCalls = append(toolCalls, choice.Delta.ToolCalls...)
			}

			// Finish Reason
			if choice.FinishReason != nil {
				response.FinishReason = *choice.FinishReason
			}
		}
	}

	response.Content = contentBuilder.String()
	response.ToolCalls = toolCalls

	return response, nil
}

// QuickChat führt einen einfachen, nicht-streamenden Chat durch
// Ideal für kurze Anfragen wie Query-Optimierung
func (s *Server) QuickChat(systemPrompt, userMessage string) (string, error) {
	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	}

	var result strings.Builder
	err := s.StreamChat(messages, func(content string, done bool) {
		result.WriteString(content)
	})

	if err != nil {
		return "", err
	}

	return strings.TrimSpace(result.String()), nil
}

// QuickChatWithTimeout führt einen Chat mit Timeout durch
func (s *Server) QuickChatWithTimeout(systemPrompt, userMessage string, timeout time.Duration) (string, error) {
	if !s.IsRunning() || !s.IsHealthy() {
		return "", fmt.Errorf("llama-server ist nicht aktiv")
	}

	// Request mit Timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resultChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := s.QuickChat(systemPrompt, userMessage)
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- result
	}()

	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return "", err
	case <-ctx.Done():
		return "", fmt.Errorf("timeout nach %v", timeout)
	}
}

// ===== Vision/Multimodal Support =====

// findMmprojForModel sucht automatisch nach dem passenden mmproj für ein Vision-Modell
// mmproj-Dateien haben typischerweise Namen wie:
// - mmproj-model-f16.gguf
// - llava-v1.6-mistral-7b-mmproj-f16.gguf
// - mmproj.gguf
func (s *Server) findMmprojForModel(modelPath string) string {
	modelDir := filepath.Dir(modelPath)
	modelName := filepath.Base(modelPath)
	modelNameLower := strings.ToLower(modelName)

	// Entferne .gguf Endung für Basis-Name
	baseName := strings.TrimSuffix(modelNameLower, ".gguf")

	// Mögliche mmproj-Namen basierend auf dem Modellnamen
	possibleNames := []string{
		// Exakte Matches
		"mmproj.gguf",
		"mmproj-f16.gguf",
		"mmproj-f32.gguf",
		// Modellname + mmproj
		baseName + "-mmproj.gguf",
		baseName + "-mmproj-f16.gguf",
		strings.Replace(baseName, "q4_k_m", "mmproj-f16", 1) + ".gguf",
		strings.Replace(baseName, "q5_k_m", "mmproj-f16", 1) + ".gguf",
		strings.Replace(baseName, "q8_0", "mmproj-f16", 1) + ".gguf",
	}

	// Im Modell-Verzeichnis suchen
	for _, name := range possibleNames {
		path := filepath.Join(modelDir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Im ModelsDir suchen
	if s.config.ModelsDir != "" && s.config.ModelsDir != modelDir {
		for _, name := range possibleNames {
			path := filepath.Join(s.config.ModelsDir, name)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}

	// Glob-Suche nach *mmproj*.gguf im Modell-Verzeichnis
	pattern := filepath.Join(modelDir, "*mmproj*.gguf")
	matches, err := filepath.Glob(pattern)
	if err == nil && len(matches) > 0 {
		return matches[0]
	}

	// Glob-Suche im ModelsDir
	if s.config.ModelsDir != "" {
		pattern = filepath.Join(s.config.ModelsDir, "*mmproj*.gguf")
		matches, err = filepath.Glob(pattern)
		if err == nil && len(matches) > 0 {
			return matches[0]
		}
	}

	return ""
}

// IsVisionEnabled prüft ob Vision/Multimodal aktiviert ist
func (s *Server) IsVisionEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.VisionEnabled
}

// SetMmprojPath setzt den Pfad zur mmproj-Datei für Vision-Modelle
func (s *Server) SetMmprojPath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config.MmprojPath = path
	log.Printf("mmproj-Pfad gesetzt: %s", path)
}

// GetMmprojPath gibt den aktuellen mmproj-Pfad zurück
func (s *Server) GetMmprojPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.MmprojPath
}

// GetRecommendedVisionModels gibt empfohlene Vision-Modelle zurück
func GetRecommendedVisionModels() []RecommendedModel {
	return []RecommendedModel{
		{
			Name:        "LLaVA-v1.6-Mistral-7B",
			Description: "Exzellentes Vision-Modell für Bildanalyse und OCR",
			Size:        "4.4 GB + 600 MB mmproj",
			URL:         "https://huggingface.co/cjpais/llava-v1.6-mistral-7b-gguf/resolve/main/llava-v1.6-mistral-7b.Q4_K_M.gguf",
			Filename:    "llava-v1.6-mistral-7b.Q4_K_M.gguf",
			Category:    "vision",
		},
		{
			Name:        "LLaVA-v1.6-Mistral-7B-mmproj",
			Description: "Multimodal Projector für LLaVA-v1.6-Mistral-7B (ERFORDERLICH)",
			Size:        "600 MB",
			URL:         "https://huggingface.co/cjpais/llava-v1.6-mistral-7b-gguf/resolve/main/mmproj-model-f16.gguf",
			Filename:    "llava-v1.6-mistral-7b-mmproj-f16.gguf",
			Category:    "vision-mmproj",
		},
		{
			Name:        "MiniCPM-V-2.6",
			Description: "Kompaktes Vision-Modell mit gutem Deutsch",
			Size:        "4.9 GB",
			URL:         "https://huggingface.co/openbmb/MiniCPM-V-2_6-gguf/resolve/main/ggml-model-Q4_K_M.gguf",
			Filename:    "minicpm-v-2.6-Q4_K_M.gguf",
			Category:    "vision",
		},
	}
}
