#!/usr/bin/env bash
# blog-update/run.sh — Deploy the Gas Town static blog to nginx.
#
# Syncs blog/site/ to the nginx document root and verifies the deployment.
#
# Usage: ./run.sh [--dry-run] [--check-only]

set -euo pipefail

# --- Configuration -----------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BLOG_SOURCE="$REPO_ROOT/blog/site"
DEPLOY_DIR="${BLOG_DEPLOY_DIR:-/var/www/gastown-blog}"
HEALTH_URL="${BLOG_HEALTH_URL:-http://localhost:80/health}"

# --- Argument parsing ---------------------------------------------------------

DRY_RUN=false
CHECK_ONLY=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run)    DRY_RUN=true; shift ;;
    --check-only) CHECK_ONLY=true; shift ;;
    --help|-h)
      echo "Usage: $0 [--dry-run] [--check-only]"
      exit 0
      ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

# --- Helpers ------------------------------------------------------------------

log() {
  echo "[blog-update] $*"
}

health_check() {
  if command -v curl &>/dev/null; then
    curl -sf --max-time 5 "$HEALTH_URL" >/dev/null 2>&1
  else
    return 1
  fi
}

# --- Check-only mode ----------------------------------------------------------

if $CHECK_ONLY; then
  if [[ ! -d "$DEPLOY_DIR" ]]; then
    log "FAIL: deploy dir $DEPLOY_DIR does not exist"
    exit 1
  fi
  if [[ ! -f "$DEPLOY_DIR/index.html" ]]; then
    log "FAIL: no index.html in $DEPLOY_DIR"
    exit 1
  fi
  if health_check; then
    log "OK: blog is deployed and nginx is serving"
    exit 0
  else
    log "WARN: blog files present but nginx health check failed"
    exit 1
  fi
fi

# --- Step 1: Validate source --------------------------------------------------

if [[ ! -f "$BLOG_SOURCE/index.html" ]]; then
  log "FAIL: blog source not found at $BLOG_SOURCE/index.html"
  exit 1
fi

FILE_COUNT=$(find "$BLOG_SOURCE" -type f | wc -l)
log "Source: $BLOG_SOURCE ($FILE_COUNT files)"

# --- Step 2: Deploy -----------------------------------------------------------

if $DRY_RUN; then
  log "DRY RUN: would sync $FILE_COUNT files to $DEPLOY_DIR"
  exit 0
fi

if [[ ! -d "$DEPLOY_DIR" ]]; then
  log "Creating deploy directory: $DEPLOY_DIR"
  mkdir -p "$DEPLOY_DIR"
fi

rsync -a --delete "$BLOG_SOURCE/" "$DEPLOY_DIR/"
log "Deployed $FILE_COUNT files to $DEPLOY_DIR"

# --- Step 3: Verify -----------------------------------------------------------

if health_check; then
  log "OK: nginx health check passed"
  RESULT="success"
  SUMMARY="blog-update: deployed $FILE_COUNT files, nginx healthy"
else
  log "WARN: deployed but nginx health check failed (nginx may not be running)"
  RESULT="success"
  SUMMARY="blog-update: deployed $FILE_COUNT files, nginx not reachable"
fi

# --- Step 4: Record result ----------------------------------------------------

bd create --title "blog-update: $SUMMARY" -t chore --ephemeral \
  -l "type:plugin-run,plugin:blog-update,result:$RESULT" \
  -d "$SUMMARY" --silent 2>/dev/null || true

log "Done: $SUMMARY"
