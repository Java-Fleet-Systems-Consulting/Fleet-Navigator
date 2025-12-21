// Package mate implementiert Mate-spezifische Logik
// Jeder Mate-Typ (Writer, Mail, Web-Search, etc.) hat eigene Capabilities
package mate

import (
	"time"
)

// MateType definiert den Typ eines Mates
type MateType string

const (
	MateTypeWriter    MateType = "writer"     // LibreOffice Writer
	MateTypeMail      MateType = "mail"       // Thunderbird
	MateTypeOutlook   MateType = "outlook"    // Microsoft Outlook
	MateTypeWebSearch MateType = "web-search" // Web-Recherche
	MateTypeBrowser   MateType = "browser"    // Browser-Extension
	MateTypeCoder     MateType = "coder"      // FleetCoder CLI
	MateTypeCustom    MateType = "custom"     // Benutzerdefiniert
)

// Capability definiert eine Fähigkeit eines Mates
type Capability string

const (
	CapabilityChat           Capability = "chat"            // Kann Chat-Nachrichten empfangen
	CapabilityDocumentEdit   Capability = "document_edit"   // Kann Dokumente bearbeiten
	CapabilityEmailSend      Capability = "email_send"      // Kann E-Mails senden
	CapabilityEmailRead      Capability = "email_read"      // Kann E-Mails lesen
	CapabilityWebFetch       Capability = "web_fetch"       // Kann Webseiten abrufen
	CapabilityCalendar       Capability = "calendar"        // Kann Kalender verwalten
	CapabilityAppointment    Capability = "appointment"     // Kann Termine planen
	CapabilityNotification   Capability = "notification"    // Kann Benachrichtigungen zeigen
	CapabilityFileAccess     Capability = "file_access"     // Kann auf Dateien zugreifen
	CapabilityCodeExec       Capability = "code_exec"       // Kann Code ausführen
	CapabilityShell          Capability = "shell"           // Kann Shell-Befehle ausführen
	CapabilityGit            Capability = "git"             // Kann Git-Operationen ausführen
)

// MateInfo enthält Informationen über einen Mate-Typ
type MateInfo struct {
	Type         MateType     `json:"type"`
	DisplayName  string       `json:"display_name"`
	Description  string       `json:"description"`
	Icon         string       `json:"icon"`
	Capabilities []Capability `json:"capabilities"`
}

// ConnectedMate repräsentiert einen aktuell verbundenen Mate
type ConnectedMate struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Type         MateType     `json:"type"`
	Version      string       `json:"version"`
	Capabilities []Capability `json:"capabilities"`
	ConnectedAt  time.Time    `json:"connected_at"`
	LastActivity time.Time    `json:"last_activity"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// MateRequest repräsentiert eine Anfrage von einem Mate
type MateRequest struct {
	ID        string                 `json:"id"`
	MateID    string                 `json:"mate_id"`
	Action    string                 `json:"action"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
}

// MateResponse repräsentiert eine Antwort an einen Mate
type MateResponse struct {
	RequestID string                 `json:"request_id"`
	Success   bool                   `json:"success"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// GetMateTypeInfo gibt Informationen zu einem Mate-Typ zurück
func GetMateTypeInfo(mateType MateType) *MateInfo {
	info, ok := mateTypes[mateType]
	if !ok {
		return nil
	}
	return &info
}

// GetAllMateTypes gibt alle verfügbaren Mate-Typen zurück
func GetAllMateTypes() []MateInfo {
	types := make([]MateInfo, 0, len(mateTypes))
	for _, info := range mateTypes {
		types = append(types, info)
	}
	return types
}

// HasCapability prüft ob ein Mate eine Fähigkeit hat
func (m *ConnectedMate) HasCapability(cap Capability) bool {
	for _, c := range m.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

// Vordefinierte Mate-Typen mit ihren Fähigkeiten
var mateTypes = map[MateType]MateInfo{
	MateTypeWriter: {
		Type:        MateTypeWriter,
		DisplayName: "Fleet Mate Writer",
		Description: "LibreOffice Writer Integration für Dokument-Assistenz",
		Icon:        "📝",
		Capabilities: []Capability{
			CapabilityChat,
			CapabilityDocumentEdit,
			CapabilityNotification,
		},
	},
	MateTypeMail: {
		Type:        MateTypeMail,
		DisplayName: "Fleet Email Mate",
		Description: "Thunderbird Integration für E-Mail-Assistenz",
		Icon:        "📧",
		Capabilities: []Capability{
			CapabilityChat,
			CapabilityEmailSend,
			CapabilityEmailRead,
			CapabilityCalendar,
			CapabilityAppointment,
			CapabilityNotification,
		},
	},
	MateTypeOutlook: {
		Type:        MateTypeOutlook,
		DisplayName: "Fleet Outlook Mate",
		Description: "Microsoft Outlook Integration für E-Mail-Assistenz",
		Icon:        "📬",
		Capabilities: []Capability{
			CapabilityChat,
			CapabilityEmailSend,
			CapabilityEmailRead,
			CapabilityCalendar,
			CapabilityAppointment,
			CapabilityNotification,
		},
	},
	MateTypeWebSearch: {
		Type:        MateTypeWebSearch,
		DisplayName: "Fleet Web Search Mate",
		Description: "Web-Recherche und Informationssammlung",
		Icon:        "🔍",
		Capabilities: []Capability{
			CapabilityChat,
			CapabilityWebFetch,
			CapabilityNotification,
		},
	},
	MateTypeBrowser: {
		Type:        MateTypeBrowser,
		DisplayName: "Fleet Browser Mate",
		Description: "Browser-Extension für Web-Assistenz",
		Icon:        "🌐",
		Capabilities: []Capability{
			CapabilityChat,
			CapabilityWebFetch,
			CapabilityNotification,
		},
	},
	MateTypeCoder: {
		Type:        MateTypeCoder,
		DisplayName: "FleetCoder",
		Description: "KI-gestützter Code-Assistent mit Shell und Git",
		Icon:        "💻",
		Capabilities: []Capability{
			CapabilityChat,
			CapabilityFileAccess,
			CapabilityCodeExec,
			CapabilityShell,
			CapabilityGit,
			CapabilityNotification,
		},
	},
	MateTypeCustom: {
		Type:        MateTypeCustom,
		DisplayName: "Custom Mate",
		Description: "Benutzerdefinierter Mate",
		Icon:        "🔧",
		Capabilities: []Capability{
			CapabilityChat,
		},
	},
}
