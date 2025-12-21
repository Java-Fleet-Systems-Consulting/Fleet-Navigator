#!/bin/bash
# Fleet Navigator Service Installation Script
# Muss als root ausgeführt werden: sudo ./install-service.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="/opt/fleet-navigator"
SERVICE_FILE="fleet-navigator.service"
BINARY="fleet-navigator"

echo "=== Fleet Navigator Service Installation ==="

# Prüfe root-Rechte
if [ "$EUID" -ne 0 ]; then
    echo "Fehler: Dieses Script muss als root ausgeführt werden."
    echo "Verwende: sudo $0"
    exit 1
fi

# Prüfe ob Binary existiert
if [ ! -f "$SCRIPT_DIR/$BINARY" ]; then
    echo "Fehler: $BINARY nicht gefunden in $SCRIPT_DIR"
    echo "Bitte erst builden: go build -o $BINARY ./cmd/navigator/"
    exit 1
fi

# Erstelle Installations-Verzeichnis
echo "Erstelle $INSTALL_DIR..."
mkdir -p "$INSTALL_DIR"

# Kopiere Binary
echo "Kopiere Binary..."
cp "$SCRIPT_DIR/$BINARY" "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/$BINARY"

# Kopiere Frontend-Dist falls vorhanden
if [ -d "$SCRIPT_DIR/cmd/navigator/dist" ]; then
    echo "Kopiere Frontend-Dist..."
    cp -r "$SCRIPT_DIR/cmd/navigator/dist" "$INSTALL_DIR/"
fi

# Kopiere Konfiguration falls vorhanden
if [ -d "$SCRIPT_DIR/configs" ]; then
    echo "Kopiere Konfiguration..."
    cp -r "$SCRIPT_DIR/configs" "$INSTALL_DIR/"
fi

# Erstelle Daten-Verzeichnis
echo "Erstelle Daten-Verzeichnis..."
TRAINER_HOME=$(getent passwd trainer | cut -d: -f6)
mkdir -p "$TRAINER_HOME/.fleet-navigator"
chown -R trainer:trainer "$TRAINER_HOME/.fleet-navigator"

# Installiere Service-File
echo "Installiere Systemd-Service..."
cp "$SCRIPT_DIR/$SERVICE_FILE" /etc/systemd/system/
chmod 644 /etc/systemd/system/$SERVICE_FILE

# Reload Systemd
echo "Lade Systemd neu..."
systemctl daemon-reload

# Aktiviere Service (startet automatisch beim Boot)
echo "Aktiviere Service..."
systemctl enable fleet-navigator

echo ""
echo "=== Installation abgeschlossen ==="
echo ""
echo "Verfügbare Befehle:"
echo "  sudo systemctl start fleet-navigator   # Service starten"
echo "  sudo systemctl stop fleet-navigator    # Service stoppen"
echo "  sudo systemctl restart fleet-navigator # Service neustarten"
echo "  sudo systemctl status fleet-navigator  # Status anzeigen"
echo "  sudo journalctl -u fleet-navigator -f  # Logs verfolgen"
echo ""
echo "Fleet Navigator wird unter http://localhost:2025 verfügbar sein."
