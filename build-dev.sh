#!/bin/bash
# Schnelles Development-Build für Fleet Navigator
# Baut nur Frontend + Linux Binary (kein Multi-Platform)

set -e

VERSION=$(grep '"version"' web/package.json | head -1 | cut -d'"' -f4)
NAME="fleet-navigator"
OUTPUT_DIR="./dist"
BUILD_TIME=$(date '+%Y-%m-%d %H:%M:%S')
LDFLAGS="-X fleet-navigator/internal/updater.Version=${VERSION} -X 'fleet-navigator/internal/updater.BuildTime=${BUILD_TIME}'"

echo "══════════════════════════════════════════════════════════════"
echo "  FLEET NAVIGATOR - Dev Build v${VERSION}"
echo "  ${BUILD_TIME}"
echo "══════════════════════════════════════════════════════════════"
echo ""

# Output-Verzeichnis erstellen
mkdir -p "$OUTPUT_DIR"

# 1. Vue.js Frontend bauen
echo "1/2 Frontend bauen..."
cd web
# WICHTIG: Vite Cache löschen um alte Dateien zu vermeiden!
rm -rf node_modules/.vite
npm run build --silent 2>/dev/null || npm run build
cd ..
echo "    Frontend gebaut"
echo ""

# 2. Go Binary bauen (nur Linux)
echo "2/2 Backend bauen (Linux)..."
go build -ldflags="${LDFLAGS}" -o "${OUTPUT_DIR}/${NAME}-linux-amd64" ./cmd/navigator
echo "    Backend gebaut"
echo ""

# Größe anzeigen
SIZE=$(ls -lh "${OUTPUT_DIR}/${NAME}-linux-amd64" | awk '{print $5}')
echo "══════════════════════════════════════════════════════════════"
echo "  Build fertig: ${OUTPUT_DIR}/${NAME}-linux-amd64 (${SIZE})"
echo "  Version: v${VERSION} | Build: ${BUILD_TIME}"
echo "══════════════════════════════════════════════════════════════"
echo ""
echo "  Starten mit: ${OUTPUT_DIR}/${NAME}-linux-amd64"
echo ""
