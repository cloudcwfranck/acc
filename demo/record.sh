#!/usr/bin/env bash
# demo/record.sh - Record asciinema demo
# This script records a deterministic terminal demo of acc

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ACC_BIN="${ACC_BIN:-$REPO_ROOT/acc}"
CAST_FILE="$SCRIPT_DIR/demo.cast"
WORKDIR="/tmp/acc-demo-recording-$$"

# Check prerequisites
if ! command -v asciinema &> /dev/null; then
    echo "Error: asciinema not found"
    echo "Install: pip install asciinema"
    exit 1
fi

if ! command -v docker &> /dev/null; then
    echo "Error: docker not found"
    exit 1
fi

# Build acc if needed
if [ ! -f "$ACC_BIN" ]; then
    echo "Building acc..."
    cd "$REPO_ROOT"
    go build -o "$ACC_BIN" ./cmd/acc
fi

# Preflight validation
echo "Running preflight validation..."
if ! bash "$SCRIPT_DIR/run.sh"; then
    echo "Error: Demo validation failed"
    echo "Fix issues before recording"
    exit 1
fi

echo "Preflight passed!"
echo ""
echo "Starting recording in 3 seconds..."
sleep 3

# Cleanup old recording
rm -f "$CAST_FILE"

# Create clean workdir
rm -rf "$WORKDIR"
mkdir -p "$WORKDIR"

# Record the demo
asciinema rec "$CAST_FILE" \
    --cols 100 \
    --rows 30 \
    --title "acc - Policy Verification for CI/CD" \
    --command "bash $SCRIPT_DIR/demo-script.sh $WORKDIR $ACC_BIN"

echo ""
echo "Recording saved to: $CAST_FILE"
echo ""
echo "To preview:"
echo "  asciinema play $CAST_FILE"
echo ""
echo "To upload to asciinema.org:"
echo "  asciinema upload $CAST_FILE"
echo ""
echo "To integrate into website:"
echo "  1. Upload to asciinema.org and get the ID"
echo "  2. Set NEXT_PUBLIC_ASCIINEMA_ID in site/.env.local"
echo "  3. Or copy demo.cast to site/public/demo/"
