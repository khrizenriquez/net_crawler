#!/bin/sh
set -eu

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PACKAGE="$ROOT/macos/DukuNetLabMenu"
APP="$ROOT/dist/Duku Net Lab.app"
BIN="$PACKAGE/.build/release/DukuNetLabMenu"

cd "$PACKAGE"
swift build -c release
rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS"
cp "$BIN" "$APP/Contents/MacOS/DukuNetLabMenu"
cat > "$APP/Contents/Info.plist" <<'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>CFBundleExecutable</key><string>DukuNetLabMenu</string>
  <key>CFBundleIdentifier</key><string>local.duku.netlab.menu</string>
  <key>CFBundleName</key><string>Duku Net Lab</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>LSMinimumSystemVersion</key><string>14.0</string>
  <key>LSUIElement</key><true/>
</dict></plist>
EOF
echo "Packaged $APP"

