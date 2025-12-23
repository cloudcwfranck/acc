#!/usr/bin/env bash
# demo/record-v2.sh - Records the production demo with asciinema
# Creates a 60-90 second, 9-command demo with colored csengineering$ prompt

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ACC_BIN="${ACC_BIN:-$REPO_ROOT/acc}"
CAST_FILE="${CAST_FILE:-$SCRIPT_DIR/demo-v2.cast}"
WORKDIR="/tmp/acc-demo-record-$(date +%s)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}========================================${NC}"
echo -e "${CYAN}acc Interactive Demo Recorder${NC}"
echo -e "${CYAN}========================================${NC}"
echo ""

# Check prerequisites
if ! command -v asciinema &> /dev/null; then
    echo -e "${RED}Error: asciinema not found${NC}"
    echo "Install it: pip install asciinema"
    exit 1
fi

if [ ! -f "$ACC_BIN" ]; then
    echo "Building acc..."
    cd "$REPO_ROOT"
    go build -o "$ACC_BIN" ./cmd/acc
    echo -e "${GREEN}✓${NC} Built acc"
fi

# Preflight validation
echo "Running preflight validation..."
if ! bash "$SCRIPT_DIR/run-v2.sh" >/dev/null 2>&1; then
    echo -e "${RED}✗ Validation failed${NC}"
    echo "Run 'bash demo/run-v2.sh' to see errors"
    exit 1
fi
echo -e "${GREEN}✓${NC} Validation passed"
echo ""

# Record
echo "Starting recording..."
echo "Output: $CAST_FILE"
echo ""
echo "Tips:"
echo "  - Terminal will auto-play the 9 commands"
echo "  - Recording will be ~60-90 seconds"
echo "  - Press Ctrl+D or wait for auto-exit"
echo ""

# Create workdir for recording
mkdir -p "$WORKDIR"

# Record with asciinema
asciinema rec "$CAST_FILE" \
    --cols 100 \
    --rows 28 \
    --title "acc - Policy Verification for CI/CD" \
    --overwrite \
    --command "bash $SCRIPT_DIR/demo-script-v2.sh $WORKDIR $ACC_BIN"

# Cleanup
rm -rf "$WORKDIR"

echo ""
echo -e "${GREEN}✓${NC} Recording complete: $CAST_FILE"
echo ""
echo "Next steps:"
echo ""
echo "1. Preview locally:"
echo "   asciinema play $CAST_FILE"
echo ""
echo "2. Upload to asciinema.org:"
echo "   asciinema upload $CAST_FILE"
echo "   # Copy the ID (e.g., 'abc123')"
echo ""
echo "3. Configure website:"
echo "   echo 'NEXT_PUBLIC_ASCIINEMA_ID=abc123' >> site/.env.local"
echo ""
echo "4. Or use local file (already copied):"
echo "   cp $CAST_FILE site/public/demo/demo.cast"
echo ""
echo "5. Preview website:"
echo "   cd site && npm install && npm run dev"
echo "   # Visit http://localhost:3000"
echo ""
