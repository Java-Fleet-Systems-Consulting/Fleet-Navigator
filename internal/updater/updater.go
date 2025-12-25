// Package updater implementiert Auto-Updates von GitHub Releases
package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	// GitHub Repository
	GitHubOwner = "java-fleet"
	GitHubRepo  = "fleet-navigator"

	// API URL
	GitHubAPIURL = "https://api.github.com/repos/%s/%s/releases/latest"

	// Check interval
	DefaultCheckInterval = 24 * time.Hour
)

// Version wird beim Build gesetzt: go build -ldflags "-X fleet-navigator/internal/updater.Version=1.0.0"
var Version = "0.8.13+20251225.1425"

// BuildTime wird beim Build gesetzt: go build -ldflags "-X fleet-navigator/internal/updater.BuildTime=2025-01-01"
var BuildTime = "development"

// FrontendBuildTime wird beim Build gesetzt: go build -ldflags "-X fleet-navigator/internal/updater.FrontendBuildTime=2025-01-01"
var FrontendBuildTime = "development"

// GitHubRelease repräsentiert ein GitHub Release
type GitHubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"` // Release Notes
	PublishedAt time.Time     `json:"published_at"`
	Assets      []GitHubAsset `json:"assets"`
	HTMLURL     string        `json:"html_url"`
}

// GitHubAsset repräsentiert ein Release-Asset (Binary)
type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	ContentType        string `json:"content_type"`
}

// UpdateInfo enthält Informationen über ein verfügbares Update
type UpdateInfo struct {
	CurrentVersion string    `json:"current_version"`
	NewVersion     string    `json:"new_version"`
	ReleaseNotes   string    `json:"release_notes"`
	DownloadURL    string    `json:"download_url"`
	DownloadSize   int64     `json:"download_size"`
	PublishedAt    time.Time `json:"published_at"`
	ReleaseURL     string    `json:"release_url"`
}

// Updater verwaltet Auto-Updates
type Updater struct {
	owner         string
	repo          string
	currentVersion string
	httpClient    *http.Client

	// Callbacks
	OnUpdateAvailable func(info *UpdateInfo)
	OnDownloadProgress func(downloaded, total int64)
}

// NewUpdater erstellt einen neuen Updater
func NewUpdater() *Updater {
	return &Updater{
		owner:         GitHubOwner,
		repo:          GitHubRepo,
		currentVersion: Version,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetCurrentVersion gibt die aktuelle Version zurück
func (u *Updater) GetCurrentVersion() string {
	return u.currentVersion
}

// CheckForUpdate prüft ob ein Update verfügbar ist
func (u *Updater) CheckForUpdate() (*UpdateInfo, error) {
	url := fmt.Sprintf(GitHubAPIURL, u.owner, u.repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// GitHub API Header
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Fleet-Navigator-Updater/"+u.currentVersion)

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub API nicht erreichbar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("Repository nicht gefunden: %s/%s", u.owner, u.repo)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API Fehler: Status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("JSON Parse Fehler: %w", err)
	}

	// Version vergleichen (v1.0.0 -> 1.0.0)
	newVersion := strings.TrimPrefix(release.TagName, "v")
	currentVersion := strings.TrimPrefix(u.currentVersion, "v")

	if !isNewerVersion(newVersion, currentVersion) {
		return nil, nil // Kein Update verfügbar
	}

	// Passendes Asset finden
	asset := u.findAssetForPlatform(release.Assets)
	if asset == nil {
		return nil, fmt.Errorf("Kein Download für %s/%s verfügbar", runtime.GOOS, runtime.GOARCH)
	}

	return &UpdateInfo{
		CurrentVersion: u.currentVersion,
		NewVersion:     release.TagName,
		ReleaseNotes:   release.Body,
		DownloadURL:    asset.BrowserDownloadURL,
		DownloadSize:   asset.Size,
		PublishedAt:    release.PublishedAt,
		ReleaseURL:     release.HTMLURL,
	}, nil
}

// findAssetForPlatform findet das passende Asset für das aktuelle OS/Arch
func (u *Updater) findAssetForPlatform(assets []GitHubAsset) *GitHubAsset {
	// Erwartete Dateinamen:
	// fleet-navigator-windows-amd64.exe
	// fleet-navigator-darwin-arm64
	// fleet-navigator-linux-amd64

	goos := runtime.GOOS
	goarch := runtime.GOARCH

	patterns := []string{
		fmt.Sprintf("fleet-navigator-%s-%s", goos, goarch),
		fmt.Sprintf("navigator-%s-%s", goos, goarch),
	}

	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		for _, pattern := range patterns {
			if strings.Contains(name, pattern) {
				return &asset
			}
		}
	}

	return nil
}

// DownloadUpdate lädt das Update herunter
func (u *Updater) DownloadUpdate(info *UpdateInfo, targetPath string) error {
	resp, err := u.httpClient.Get(info.DownloadURL)
	if err != nil {
		return fmt.Errorf("Download fehlgeschlagen: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("Download Fehler: Status %d", resp.StatusCode)
	}

	// Temporäre Datei erstellen
	tmpFile := targetPath + ".tmp"
	out, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("Temp-Datei konnte nicht erstellt werden: %w", err)
	}
	defer out.Close()

	// Mit Progress-Tracking kopieren
	var downloaded int64
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				os.Remove(tmpFile)
				return fmt.Errorf("Schreibfehler: %w", writeErr)
			}
			downloaded += int64(n)

			if u.OnDownloadProgress != nil {
				u.OnDownloadProgress(downloaded, info.DownloadSize)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			os.Remove(tmpFile)
			return fmt.Errorf("Download abgebrochen: %w", err)
		}
	}

	out.Close()

	// Ausführbar machen (Linux/macOS)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpFile, 0755); err != nil {
			os.Remove(tmpFile)
			return fmt.Errorf("Chmod fehlgeschlagen: %w", err)
		}
	}

	// Alte Datei umbenennen (Backup)
	execPath, err := os.Executable()
	if err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("Executable-Pfad nicht gefunden: %w", err)
	}

	backupPath := execPath + ".backup"
	os.Remove(backupPath) // Altes Backup löschen

	// Unter Windows: Rename während Prozess läuft funktioniert
	if err := os.Rename(execPath, backupPath); err != nil {
		// Fallback: Neue Version neben alte legen
		targetPath = execPath + ".new"
		if err := os.Rename(tmpFile, targetPath); err != nil {
			os.Remove(tmpFile)
			return fmt.Errorf("Update konnte nicht installiert werden: %w", err)
		}
		return fmt.Errorf("Bitte manuell ersetzen: %s", targetPath)
	}

	// Neue Version an Stelle der alten
	if err := os.Rename(tmpFile, execPath); err != nil {
		// Rollback
		os.Rename(backupPath, execPath)
		os.Remove(tmpFile)
		return fmt.Errorf("Update Installation fehlgeschlagen: %w", err)
	}

	return nil
}

// ApplyUpdate wendet das Update an (startet neu)
func (u *Updater) ApplyUpdate() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	// Neuen Prozess starten
	// Unter Windows: Start-Process
	// Unter Linux/macOS: exec

	fmt.Printf("Update erfolgreich! Bitte Navigator neu starten: %s\n", execPath)

	// Hinweis: Automatischer Neustart ist plattformspezifisch
	// Für jetzt: User muss manuell neu starten
	return nil
}

// GetUpdateDirectory gibt das Verzeichnis für Update-Downloads zurück
func GetUpdateDirectory() string {
	dir := filepath.Join(GetDataDirectory(), "updates")
	os.MkdirAll(dir, 0755)
	return dir
}

// GetDataDirectory gibt das plattformspezifische Datenverzeichnis zurück
// Windows: %LOCALAPPDATA%\FleetNavigator
// Linux/macOS: ~/.fleet-navigator
func GetDataDirectory() string {
	if runtime.GOOS == "windows" {
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return filepath.Join(localAppData, "FleetNavigator")
		}
		// Fallback auf UserConfigDir
		if configDir, err := os.UserConfigDir(); err == nil {
			return filepath.Join(configDir, "FleetNavigator")
		}
	}
	// Linux/macOS
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".fleet-navigator")
}

// isNewerVersion vergleicht Semver-Versionen
func isNewerVersion(new, current string) bool {
	if current == "dev" {
		return true // Dev-Version bekommt immer Updates
	}

	newParts := parseVersion(new)
	currentParts := parseVersion(current)

	for i := 0; i < 3; i++ {
		if newParts[i] > currentParts[i] {
			return true
		}
		if newParts[i] < currentParts[i] {
			return false
		}
	}
	return false
}

// parseVersion parsed "1.2.3" zu [1, 2, 3]
func parseVersion(v string) [3]int {
	var parts [3]int
	v = strings.TrimPrefix(v, "v")

	segments := strings.Split(v, ".")
	for i := 0; i < len(segments) && i < 3; i++ {
		fmt.Sscanf(segments[i], "%d", &parts[i])
	}
	return parts
}

// StartBackgroundChecker startet periodische Update-Checks
func (u *Updater) StartBackgroundChecker(interval time.Duration) {
	go func() {
		// Erster Check nach 5 Sekunden
		time.Sleep(5 * time.Second)

		for {
			info, err := u.CheckForUpdate()
			if err == nil && info != nil {
				if u.OnUpdateAvailable != nil {
					u.OnUpdateAvailable(info)
				}
			}
			time.Sleep(interval)
		}
	}()
}
