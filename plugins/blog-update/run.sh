#!/usr/bin/env bash
# blog-update/run.sh — Rebuild Gas Town blog when content changes.
#
# Compares a content hash against the last build. If changed (or --force),
# runs blog/build.sh and optionally reloads nginx.
#
# Usage: ./run.sh [--force]

set -euo pipefail

# --- Configuration -----------------------------------------------------------

PLUGIN_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$PLUGIN_DIR/../.." && pwd)"
BLOG_DIR="$REPO_ROOT/blog"
CONTENT_DIR="$BLOG_DIR/content"
BUILD_SCRIPT="$BLOG_DIR/build.sh"
HASH_FILE="$BLOG_DIR/.last-build-hash"

# --- Argument parsing ---------------------------------------------------------

FORCE=false
while [[ $# -gt 0 ]]; do
  case "$1" in
    --force) FORCE=true; shift ;;
    --help|-h)
      echo "Usage: $0 [--force]"
      exit 0
      ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

# --- Helpers ------------------------------------------------------------------

log() {
  echo "[blog-update] $*"
}

# --- Step 1: Check for content changes ---------------------------------------

# Hash all content files to detect changes
if command -v sha256sum &>/dev/null; then
  HASH_CMD="sha256sum"
elif command -v shasum &>/dev/null; then
  HASH_CMD="shasum -a 256"
else
  log "No hash command available, forcing rebuild"
  FORCE=true
  HASH_CMD="cat"
fi

CURRENT_HASH=""
if [[ -d "$CONTENT_DIR" ]]; then
  CURRENT_HASH=$(find "$CONTENT_DIR" -type f | sort | xargs $HASH_CMD 2>/dev/null | $HASH_CMD | cut -d' ' -f1)
fi

LAST_HASH=""
if [[ -f "$HASH_FILE" ]]; then
  LAST_HASH=$(cat "$HASH_FILE")
fi

if [[ "$CURRENT_HASH" == "$LAST_HASH" ]] && ! $FORCE; then
  log "Content unchanged, skipping build"
  bd create --title "blog-update: no changes" -t chore --ephemeral \
    --labels type:plugin-run,plugin:blog-update,result:skipped \
    -d "Content hash unchanged, build skipped" --silent 2>/dev/null || true
  exit 0
fi

# --- Step 2: Build the blog ---------------------------------------------------

log "Content changed, rebuilding blog..."

if [[ ! -x "$BUILD_SCRIPT" ]]; then
  chmod +x "$BUILD_SCRIPT"
fi

BUILD_OUTPUT=$(bash "$BUILD_SCRIPT" --clean 2>&1) || {
  log "Build FAILED: $BUILD_OUTPUT"
  bd create --title "blog-update: FAILED" -t chore --ephemeral \
    --labels type:plugin-run,plugin:blog-update,result:failure \
    -d "Build failed: $BUILD_OUTPUT" --silent 2>/dev/null || true
  gt escalate "blog-update FAILED: $BUILD_OUTPUT" \
    --severity low \
    --reason "Blog build script failed" 2>/dev/null || true
  exit 1
}

log "$BUILD_OUTPUT"

# --- Step 3: Save content hash ------------------------------------------------

echo "$CURRENT_HASH" > "$HASH_FILE"

# --- Step 4: Reload nginx if running ------------------------------------------

if command -v nginx &>/dev/null && pgrep -x nginx &>/dev/null; then
  if sudo nginx -t 2>/dev/null; then
    sudo systemctl reload nginx 2>/dev/null || true
    log "nginx reloaded"
  else
    log "nginx config test failed, skipping reload"
  fi
else
  log "nginx not running, skipping reload"
fi

# --- Step 5: Record success ---------------------------------------------------

bd create --title "blog-update: rebuilt" -t chore --ephemeral \
  --labels type:plugin-run,plugin:blog-update,result:success \
  -d "$BUILD_OUTPUT" --silent 2>/dev/null || true

log "Done"
