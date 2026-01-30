#!/bin/bash

set -e
set -u

# Configuration
BIN_DIR="./bin"
APP_NAME="octa"
BUILDER_NAME="gocraft"
BUILDER_SRC="./scripts/gocraft.go"
APP_SRC="./cmd/octa"

# Detect if running on Windows (Git Bash / WSL / Cygwin) to handle .exe extensions
EXT=""
case "$OSTYPE" in
  msys*|cygwin*|win32*)
    EXT=".exe"
    ;;
esac

TARGET_APP="$BIN_DIR/$APP_NAME$EXT"
TARGET_CRAFT="$BIN_DIR/$BUILDER_NAME$EXT"

echo "[INFO] Starting build workflow for $APP_NAME..."

# Check if the bin directory exists
if [ ! -d "$BIN_DIR" ]; then
    mkdir -p "$BIN_DIR"
fi

if [ -f "$TARGET_CRAFT" ]; then
    echo "[INFO] Builder binary found. Using $TARGET_CRAFT..."
    "$TARGET_CRAFT" -n "$APP_NAME" -e "$APP_SRC"
else
    echo "[INFO] Builder binary not found. Running from source..."
    go run "$BUILDER_SRC" -n "$APP_NAME" -e "$APP_SRC"
fi

if [ -f "$TARGET_APP" ]; then
    echo "[INFO] Build successful. Starting application..."
    echo "--------------------------------------------------"
    "$TARGET_APP"
else
    echo "[ERROR] Build failed. Target binary not found."
    exit 1
fi