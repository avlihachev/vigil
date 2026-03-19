#!/bin/bash
set -e
echo "Building universal binary..."
wails build -platform darwin/universal -clean

APP="build/bin/Vigil.app"
BIN="$APP/Contents/MacOS/vigil"
BUNDLE_ID="com.vigil.app"

echo "Signing with identifier $BUNDLE_ID..."
codesign --force --sign - --identifier "$BUNDLE_ID" "$BIN"
codesign --force --sign - --identifier "$BUNDLE_ID" "$APP"
DMG="dist/Vigil.dmg"
STAGE="/tmp/vigil-dmg-stage"

mkdir -p dist
rm -rf "$STAGE"
mkdir -p "$STAGE"
cp -r "$APP" "$STAGE/"
ln -sf /Applications "$STAGE/Applications"

echo "Creating DMG..."
hdiutil create -volname "Vigil" \
  -srcfolder "$STAGE" \
  -ov -format UDZO \
  "$DMG"

rm -rf "$STAGE"
echo "Done: $DMG"
