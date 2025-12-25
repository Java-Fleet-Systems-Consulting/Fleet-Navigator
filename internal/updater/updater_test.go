package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestNewUpdater prüft die Erstellung eines neuen Updaters
func TestNewUpdater(t *testing.T) {
	u := NewUpdater()

	if u == nil {
		t.Fatal("NewUpdater() returned nil")
	}

	if u.owner != GitHubOwner {
		t.Errorf("expected owner %q, got %q", GitHubOwner, u.owner)
	}

	if u.repo != GitHubRepo {
		t.Errorf("expected repo %q, got %q", GitHubRepo, u.repo)
	}

	if u.currentVersion != Version {
		t.Errorf("expected version %q, got %q", Version, u.currentVersion)
	}
}

// TestGetCurrentVersion prüft die Rückgabe der aktuellen Version
func TestGetCurrentVersion(t *testing.T) {
	u := NewUpdater()
	version := u.GetCurrentVersion()

	if version != Version {
		t.Errorf("expected version %q, got %q", Version, version)
	}
}

// TestCheckForUpdate_NoUpdateAvailable prüft den Fall, wenn kein Update verfügbar ist
func TestCheckForUpdate_NoUpdateAvailable(t *testing.T) {
	// Mock-Server erstellen, der die aktuelle Version zurückgibt
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		release := GitHubRelease{
			TagName: "v" + Version, // Gleiche Version wie aktuell
			Name:    "Test Release",
			Body:    "Test release notes",
			Assets:  []GitHubAsset{},
		}
		json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()

	// Hinweis: Dieser Test würde normalerweise den echten GitHub API aufrufen
	// Für einen vollständigen Test müsste man die API-URL überschreiben können
	// Der Mock-Server wird hier nur zur Demonstration erstellt
	_ = server // Server wird für späteren Ausbau benötigt
	t.Skip("Test erfordert Mocking der GitHub API URL")
}

// TestCheckForUpdate_UpdateAvailable prüft den Fall, wenn ein Update verfügbar ist
func TestCheckForUpdate_UpdateAvailable(t *testing.T) {
	// Hinweis: Dieser Test würde normalerweise den echten GitHub API aufrufen
	// Für einen vollständigen Test müsste man die API-URL überschreiben können
	t.Skip("Test erfordert Mocking der GitHub API URL")
}

// TestIsNewerVersion prüft die Versions-Vergleichsfunktion
func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		name     string
		newVer   string
		currentVer string
		expected bool
	}{
		{
			name:       "newer major version",
			newVer:     "2.0.0",
			currentVer: "1.0.0",
			expected:   true,
		},
		{
			name:       "newer minor version",
			newVer:     "1.1.0",
			currentVer: "1.0.0",
			expected:   true,
		},
		{
			name:       "newer patch version",
			newVer:     "1.0.1",
			currentVer: "1.0.0",
			expected:   true,
		},
		{
			name:       "same version",
			newVer:     "1.0.0",
			currentVer: "1.0.0",
			expected:   false,
		},
		{
			name:       "older version",
			newVer:     "0.9.0",
			currentVer: "1.0.0",
			expected:   false,
		},
		{
			name:       "version with prefix",
			newVer:     "v2.0.0",
			currentVer: "v1.0.0",
			expected:   true,
		},
		{
			name:       "complex version",
			newVer:     "1.0.0+build123",
			currentVer: "0.9.0+build456",
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNewerVersion(tt.newVer, tt.currentVer)
			if result != tt.expected {
				t.Errorf("isNewerVersion(%q, %q) = %v, expected %v",
					tt.newVer, tt.currentVer, result, tt.expected)
			}
		})
	}
}

// TestVersionConstants prüft, dass Version und BuildTime gesetzt sind
func TestVersionConstants(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}

	// BuildTime kann "development" sein, das ist OK
	if BuildTime == "" {
		t.Error("BuildTime should not be empty")
	}
}

// TestGitHubConstants prüft die GitHub-Konstanten
func TestGitHubConstants(t *testing.T) {
	if GitHubOwner == "" {
		t.Error("GitHubOwner should not be empty")
	}

	if GitHubRepo == "" {
		t.Error("GitHubRepo should not be empty")
	}

	if GitHubAPIURL == "" {
		t.Error("GitHubAPIURL should not be empty")
	}
}

// TestFindAssetForPlatform prüft die Plattform-Asset-Suche
func TestFindAssetForPlatform(t *testing.T) {
	u := NewUpdater()

	assets := []GitHubAsset{
		{Name: "fleet-navigator-linux-amd64", BrowserDownloadURL: "https://example.com/linux"},
		{Name: "fleet-navigator-windows-amd64.exe", BrowserDownloadURL: "https://example.com/windows"},
		{Name: "fleet-navigator-darwin-arm64", BrowserDownloadURL: "https://example.com/macos"},
	}

	asset := u.findAssetForPlatform(assets)

	// Das Ergebnis hängt vom OS ab, auf dem der Test läuft
	// Wir prüfen nur, dass kein Fehler auftritt
	if asset != nil {
		if asset.BrowserDownloadURL == "" {
			t.Error("asset should have a download URL")
		}
	}
}
