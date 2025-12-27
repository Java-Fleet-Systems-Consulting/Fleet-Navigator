#!/bin/bash
# Build-Script für Fleet Navigator
# Erstellt Binaries für alle Plattformen mit eingebettetem Vue.js Frontend

set -e

VERSION="${1:-1.0.0}"
NAME="fleet-navigator"
OUTPUT_DIR="./dist"
BUILD_TIME=$(date '+%Y-%m-%d %H:%M:%S')
FRONTEND_BUILD_TIME="not built"

echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║           FLEET NAVIGATOR - Build v${VERSION}                      ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo ""

# Output-Verzeichnis erstellen
mkdir -p "$OUTPUT_DIR"

# 1. Vue.js Frontend bauen (falls vorhanden)
if [ -d "web" ] && [ -f "web/package.json" ]; then
    echo "📦 Vue.js Frontend bauen..."
    cd web
    # WICHTIG: Vite Cache löschen um alte Dateien zu vermeiden!
    rm -rf node_modules/.vite
    # --no-bin-links für NTFS/Windows-Partitionen (keine Symlinks möglich)
    npm install --silent --no-bin-links
    # vite direkt aufrufen (wegen --no-bin-links keine .bin/vite Symlinks)
    node node_modules/vite/bin/vite.js build
    # Frontend-Build-Zeit erfassen (nach erfolgreichem Build)
    FRONTEND_BUILD_TIME=$(date '+%Y-%m-%d %H:%M:%S')
    cd ..
    # Vite baut direkt nach cmd/navigator/dist/ (siehe vite.config.js)
    echo "   ✓ Frontend gebaut: ${FRONTEND_BUILD_TIME}"
else
    echo "⚠️  Kein Frontend gefunden (web/), baue nur Backend"
    mkdir -p cmd/navigator/dist
    echo "<html><body><h1>Fleet Navigator API</h1></body></html>" > cmd/navigator/dist/index.html
    FRONTEND_BUILD_TIME="embedded fallback"
fi

# LDFLAGS mit Frontend-Zeit
LDFLAGS="-s -w -X fleet-navigator/internal/updater.Version=${VERSION} -X 'fleet-navigator/internal/updater.BuildTime=${BUILD_TIME}' -X 'fleet-navigator/internal/updater.FrontendBuildTime=${FRONTEND_BUILD_TIME}'"

echo ""

# 2. Go Dependencies
echo "📥 Go Dependencies..."
go mod tidy
echo "   ✓ Dependencies aktualisiert"
echo ""

# Funktion zum Bauen
build() {
    local os=$1
    local arch=$2
    local suffix=$3

    output="${OUTPUT_DIR}/${NAME}-${os}-${arch}${suffix}"
    echo -n "🔨 Building: ${os}/${arch}..."

    GOOS=$os GOARCH=$arch go build -ldflags="${LDFLAGS}" -o "$output" ./cmd/navigator

    size=$(ls -lh "$output" | awk '{print $5}')
    echo " ✓ ${size}"
}

echo "🚀 Baue für alle Plattformen..."
echo ""

# Linux
build linux amd64 ""
build linux arm64 ""

# Windows
build windows amd64 ".exe"

# macOS
build darwin amd64 ""
build darwin arm64 ""

echo ""
echo "════════════════════════════════════════════════════════════════"
echo "✅ Build abgeschlossen!"
echo ""
echo "📁 Erstellte Dateien:"
ls -lh "$OUTPUT_DIR"/${NAME}-*

echo ""
echo "📋 Für GitHub Release:"
echo "   1. git tag v${VERSION}"
echo "   2. git push origin v${VERSION}"
echo "   3. gh release create v${VERSION} ${OUTPUT_DIR}/${NAME}-* --title \"v${VERSION}\""
echo ""
echo "🚀 Lokaler Test:"
echo "   ./${OUTPUT_DIR}/${NAME}-linux-amd64"
