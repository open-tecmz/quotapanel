#!/usr/bin/env bash
# Package QuotaPanel Linux release: deb + AppImage
#
# Usage: scripts/package-linux.sh <goarch> <version>
#   goarch:  amd64 | arm64
#   version: e.g. 0.2.4
#
# Requires: wails build output at build/bin/QuotaPanel, build/appicon.png
# Outputs:  QuotaPanel-<version>-linux-<goarch>.deb / .AppImage (repo root)

set -euo pipefail

GOARCH="${1:?goarch required (amd64|arm64)}"
VERSION="${2:?version required}"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

case "$GOARCH" in
  amd64)
    DEB_ARCH="amd64"
    APPIMAGE_TOOL="appimagetool-x86_64.AppImage"
    ;;
  arm64)
    DEB_ARCH="arm64"
    APPIMAGE_TOOL="appimagetool-aarch64.AppImage"
    ;;
  *)
    echo "ERROR: unsupported goarch: $GOARCH" >&2
    exit 1
    ;;
esac

DESKTOP_CONTENT='[Desktop Entry]
Name=QuotaPanel
Comment=Monitor AI subscription and API quota
Exec=quotapanel
Icon=quotapanel
Terminal=false
Type=Application
Categories=Utility;Development;
'

# ── deb (via fpm) ──────────────────────────────────────────────
if ! command -v fpm >/dev/null 2>&1; then
  echo "Installing fpm..."
  sudo apt-get install -y ruby ruby-dev rubygems >/dev/null 2>&1
  sudo gem install fpm >/dev/null
fi

PKG_DIR="pkg-deb"
rm -rf "$PKG_DIR"
mkdir -p "$PKG_DIR/usr/bin" "$PKG_DIR/usr/share/applications" "$PKG_DIR/usr/share/icons/hicolor/256x256/apps"
cp build/bin/QuotaPanel "$PKG_DIR/usr/bin/quotapanel"
printf '%s' "$DESKTOP_CONTENT" > "$PKG_DIR/usr/share/applications/quotapanel.desktop"
cp build/appicon.png "$PKG_DIR/usr/share/icons/hicolor/256x256/apps/quotapanel.png"

fpm -s dir -t deb \
  -n quotapanel \
  -v "$VERSION" \
  -a "$DEB_ARCH" \
  --description "QuotaPanel - Monitor AI subscription and API quota" \
  --category "Utility" \
  --maintainer "MZ <modstart@163.com>" \
  --url "https://open.tecmz.com/quotapanel" \
  --license "Apache-2.0" \
  -C "$PKG_DIR" \
  -p "QuotaPanel-${VERSION}-linux-${GOARCH}.deb" >/dev/null
rm -rf "$PKG_DIR"
echo "✅ deb: QuotaPanel-${VERSION}-linux-${GOARCH}.deb"

# ── AppImage (via appimagetool) ────────────────────────────────
if [ ! -f appimagetool ]; then
  echo "Downloading appimagetool..."
  wget -q "https://github.com/AppImage/appimagetool/releases/download/continuous/${APPIMAGE_TOOL}" -O appimagetool
  chmod +x appimagetool
fi

APPDIR="AppDir"
rm -rf "$APPDIR"
mkdir -p "$APPDIR/usr/bin"
cp build/bin/QuotaPanel "$APPDIR/usr/bin/quotapanel"
cp build/appicon.png "$APPDIR/quotapanel.png"
printf '%s' "$DESKTOP_CONTENT" > "$APPDIR/quotapanel.desktop"
printf '#!/bin/sh\nexec "$(dirname "$0")/usr/bin/quotapanel" "$@"\n' > "$APPDIR/AppRun"
chmod +x "$APPDIR/AppRun"

APPIMAGE_EXTRACT_AND_RUN=1 ./appimagetool "$APPDIR" "QuotaPanel-${VERSION}-linux-${GOARCH}.AppImage" >/dev/null 2>&1
rm -rf "$APPDIR"
echo "✅ AppImage: QuotaPanel-${VERSION}-linux-${GOARCH}.AppImage"

echo ""
echo "Packaged files:"
ls -lh QuotaPanel-${VERSION}-linux-${GOARCH}.deb QuotaPanel-${VERSION}-linux-${GOARCH}.AppImage
