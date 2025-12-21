// Package config implementiert die Konfigurationsverwaltung
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

// Config ist die Hauptkonfiguration
type Config struct {
	Server   ServerConfig   `json:"server"`
	Ollama   OllamaConfig   `json:"ollama"`
	Models   ModelsConfig   `json:"models"`
	Security SecurityConfig `json:"security"`
	Logging  LoggingConfig  `json:"logging"`
	Experts  ExpertsConfig  `json:"experts"`
	DataDir  string         `json:"data_dir"`
}

// ServerConfig enthält Server-Einstellungen
type ServerConfig struct {
	Port string `json:"port"`
	Host string `json:"host"`
}

// OllamaConfig enthält Ollama-Einstellungen
type OllamaConfig struct {
	URL            string `json:"url"`
	DefaultModel   string `json:"default_model"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

// ModelsConfig enthält Smart Model Selection Einstellungen
type ModelsConfig struct {
	Enabled bool   `json:"enabled"`
	Default string `json:"default"`
	Code    string `json:"code"`
	Fast    string `json:"fast"`
	Vision  string `json:"vision"`
}

// SecurityConfig enthält Sicherheits-Einstellungen
type SecurityConfig struct {
	PairingTimeout         int `json:"pairing_timeout"`
	MaxConnectionsPerMate  int `json:"max_connections_per_mate"`
}

// LoggingConfig enthält Logging-Einstellungen
type LoggingConfig struct {
	Level   string `json:"level"`
	LogFile string `json:"log_file,omitempty"`
}

// ExpertsConfig enthält Experten-System-Einstellungen
type ExpertsConfig struct {
	SeedDefaults bool `json:"seed_defaults"`
}

// getDefaultDataDir gibt das plattformspezifische Datenverzeichnis zurück
// Windows: %LOCALAPPDATA%\FleetNavigator (z.B. C:\Users\Franz\AppData\Local\FleetNavigator)
// Linux/macOS: ~/.fleet-navigator
func getDefaultDataDir() string {
	if runtime.GOOS == "windows" {
		// Windows: LOCALAPPDATA verwenden
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return filepath.Join(localAppData, "FleetNavigator")
		}
		// Fallback auf UserConfigDir
		if configDir, err := os.UserConfigDir(); err == nil {
			return filepath.Join(configDir, "FleetNavigator")
		}
	}
	// Linux/macOS: Home-Verzeichnis mit Punkt-Präfix
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".fleet-navigator")
}

// DefaultConfig gibt die Standard-Konfiguration zurück
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: "2025",
			Host: "localhost",
		},
		Ollama: OllamaConfig{
			URL:            "http://localhost:11434",
			DefaultModel:   "qwen2.5:7b",
			TimeoutSeconds: 120,
		},
		Models: ModelsConfig{
			Enabled: true,
			Default: "qwen2.5:7b",
			Code:    "qwen2.5-coder:7b",
			Fast:    "llama3.2:3b",
			Vision:  "llava:13b",
		},
		Security: SecurityConfig{
			PairingTimeout:        300,
			MaxConnectionsPerMate: 1,
		},
		Logging: LoggingConfig{
			Level: "info",
		},
		Experts: ExpertsConfig{
			SeedDefaults: true,
		},
		DataDir: getDefaultDataDir(),
	}
}

// Load lädt die Konfiguration aus einer Datei
func Load(path string) (*Config, error) {
	config := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil // Datei existiert nicht, verwende Defaults
		}
		return nil, err
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}

// LoadFromDir sucht nach config.json im angegebenen Verzeichnis
func LoadFromDir(dir string) (*Config, error) {
	configPath := filepath.Join(dir, "config.json")
	return Load(configPath)
}

// LoadDefault lädt die Konfiguration aus Standardpfaden
// Sucht in: ./config.json, ./configs/config.json, <DataDir>/config.json
func LoadDefault() (*Config, error) {
	searchPaths := []string{
		"config.json",
		"configs/config.json",
		filepath.Join(getDefaultDataDir(), "config.json"),
	}

	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			return Load(path)
		}
	}

	// Keine Konfigurationsdatei gefunden, verwende Defaults
	return DefaultConfig(), nil
}

// Save speichert die Konfiguration
func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// Merge überschreibt Werte mit Umgebungsvariablen
func (c *Config) MergeEnv() {
	if port := os.Getenv("PORT"); port != "" {
		c.Server.Port = port
	}
	if ollamaURL := os.Getenv("OLLAMA_URL"); ollamaURL != "" {
		c.Ollama.URL = ollamaURL
	}
	if ollamaModel := os.Getenv("OLLAMA_MODEL"); ollamaModel != "" {
		c.Ollama.DefaultModel = ollamaModel
	}
	if dataDir := os.Getenv("DATA_DIR"); dataDir != "" {
		c.DataDir = dataDir
	}
}
