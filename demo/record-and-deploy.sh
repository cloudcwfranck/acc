#!/usr/bin/env bash
# demo/record-and-deploy.sh - Record production demo and deploy to website
# This script records the exact 9-command demo and deploys it in one step

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DEMO_CAST="$SCRIPT_DIR/demo.cast"
SITE_CAST="$REPO_ROOT/site/public/demo/demo.cast"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() { echo -e "${GREEN}✓${NC} $*"; }
log_info() { echo -e "${CYAN}ℹ${NC} $*"; }
log_warn() { echo -e "${YELLOW}⚠${NC} $*"; }
log_error() { echo -e "${RED}✗${NC} $*"; exit 1; }

echo ""
echo "========================================="
echo "acc Demo Recording + Deployment"
echo "========================================="
echo ""

# Prerequisites check
log_info "Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    log_error "Docker not found. Install Docker to record the demo."
fi

if ! command -v asciinema &> /dev/null; then
    log_error "asciinema not found. Install: pip install asciinema"
fi

if ! command -v jq &> /dev/null; then
    log_error "jq not found. Install: sudo apt-get install jq (or brew install jq)"
fi

log "Prerequisites met: docker, asciinema, jq"
echo ""

# Build acc if needed
ACC_BIN="$REPO_ROOT/acc"
if [ ! -f "$ACC_BIN" ]; then
    log_info "Building acc..."
    cd "$REPO_ROOT"
    go build -o "$ACC_BIN" ./cmd/acc || log_error "Failed to build acc"
    log "Built acc binary"
else
    log "acc binary found: $ACC_BIN"
fi
echo ""

# Step 1: Record the demo
log_info "Step 1/3: Recording demo with exact 9 commands..."
echo ""

cd "$SCRIPT_DIR"
if bash record.sh; then
    log "Recording complete: $DEMO_CAST"
else
    log_error "Recording failed. Check output above."
fi
echo ""

# Verify the recording exists
if [ ! -f "$DEMO_CAST" ]; then
    log_error "Demo recording not found: $DEMO_CAST"
fi

# Get recording stats
lines=$(wc -l < "$DEMO_CAST")
duration=$(tail -1 "$DEMO_CAST" | grep -oE '^\[[0-9.]+' | tr -d '[' || echo "unknown")

log_info "Recording stats:"
echo "  - File: $DEMO_CAST"
echo "  - Lines: $lines"
echo "  - Duration: ${duration}s"
echo ""

# Verify it's valid asciinema format
if ! head -1 "$DEMO_CAST" | grep -q '"version"'; then
    log_error "Recording doesn't appear to be valid asciinema format"
fi

log "Recording validated (asciinema v2 format)"
echo ""

# Step 2: Preview (optional)
log_info "Step 2/3: Preview recording (optional)"
echo ""
echo "You can preview the recording with:"
echo "  asciinema play $DEMO_CAST"
echo ""
read -p "Preview now? (y/N): " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    asciinema play "$DEMO_CAST"
    echo ""
fi

# Step 3: Deploy to website
log_info "Step 3/3: Deploying to website..."
echo ""

# Ensure site/public/demo directory exists
mkdir -p "$(dirname "$SITE_CAST")"

# Copy recording to website
cp "$DEMO_CAST" "$SITE_CAST"
log "Copied to: $SITE_CAST"
echo ""

# Check git status
cd "$REPO_ROOT"
if git diff --quiet site/public/demo/demo.cast 2>/dev/null; then
    log_warn "No changes detected in demo.cast - file is identical to previous version"
    echo ""
    echo "This might mean:"
    echo "  1. The recording is the same as before"
    echo "  2. The recording didn't capture new changes"
    echo ""
    read -p "Continue with commit anyway? (y/N): " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Deployment cancelled"
        exit 0
    fi
fi

# Commit the new demo
log_info "Committing new demo recording..."
git add site/public/demo/demo.cast

git commit -m "$(cat <<'EOF'
feat(demo): Deploy production v2 recording with exact 9 commands

Recorded using demo/record.sh with the following specifications:

Commands (9 total):
1. acc version - Versioned, deterministic tool
2. acc init demo-project - Policy baseline
3. acc build demo-app:ok - PASSING workload + SBOM
4. acc verify --json demo-app:ok | jq - Verify PASS with JSON
5. echo "EXIT=0" - CI gate PASS (exit code 0)
6. acc build demo-app:root - FAILING workload (runs as root)
7. acc verify --json demo-app:root | jq - Verify FAIL with violations
8. acc policy explain - Explainable violations
9. verify → attest → trust status - Full trust cycle

Recording specs:
- Terminal: 100×28
- Prompt: csengineering$ (cyan colored)
- Duration: ~60-85 seconds
- Format: asciinema v2

Contract compliance:
- Exit codes: PASS=0, FAIL=1 ✓
- JSON schema validation ✓
- Deterministic (local builds only) ✓

Source: demo/record.sh
EOF
)"

log "Committed demo recording"
echo ""

# Push to remote
log_info "Pushing to remote..."
CURRENT_BRANCH=$(git branch --show-current)

if git push -u origin "$CURRENT_BRANCH"; then
    log "Pushed to origin/$CURRENT_BRANCH"
else
    log_error "Push failed. Check network and retry."
fi

echo ""
echo "========================================="
echo "Deployment Complete!"
echo "========================================="
echo ""
log "New demo recording deployed to website"
echo ""
echo "Next steps:"
echo "  1. The demo will be available after website rebuild"
echo "  2. Preview locally: cd site && npm run dev"
echo "  3. Visit http://localhost:3000 to see the new demo"
echo ""
echo "Recording files:"
echo "  - Source: $DEMO_CAST"
echo "  - Website: $SITE_CAST"
echo "  - Branch: $CURRENT_BRANCH"
echo ""
log "Demo is now live! 🚀"
echo ""
