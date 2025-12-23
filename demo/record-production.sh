#!/usr/bin/env bash
# demo/record-production.sh - Records EXACT 9-command demo with franck@csengineering$ prompt

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ACC_BIN="${ACC_BIN:-$REPO_ROOT/acc}"
CAST_FILE="${CAST_FILE:-$SCRIPT_DIR/demo-production.cast}"
WORKDIR="/tmp/acc-demo-record-$(date +%s)"

echo "========================================="
echo "acc Production Demo Recorder"
echo "9 commands with franck@csengineering\$ prompt"
echo "========================================="
echo ""

# Check prerequisites
if ! command -v asciinema &> /dev/null; then
    echo "❌ Error: asciinema not found"
    echo "Install: pip install asciinema"
    exit 1
fi

if ! command -v docker &> /dev/null; then
    echo "❌ Error: Docker not found"
    echo "Install: https://docs.docker.com/get-docker/"
    exit 1
fi

if [ ! -f "$ACC_BIN" ]; then
    echo "Building acc..."
    cd "$REPO_ROOT"
    go build -o "$ACC_BIN" ./cmd/acc
    echo "✓ Built acc"
fi

echo "Recording configuration:"
echo "  Prompt: franck@csengineering\$"
echo "  Commands: 9 (exact sequence)"
echo "  Terminal: 100x28"
echo "  Output: $CAST_FILE"
echo ""

# Create workdir
mkdir -p "$WORKDIR"

# Record
asciinema rec "$CAST_FILE" \
    --cols 100 \
    --rows 28 \
    --title "acc - Policy Verification for CI/CD" \
    --overwrite \
    --command "bash $SCRIPT_DIR/demo-script-production.sh $WORKDIR $ACC_BIN"

# Cleanup
rm -rf "$WORKDIR"

echo ""
echo "✓ Recording complete: $CAST_FILE"
echo ""
echo "Next steps:"
echo "1. Preview: asciinema play $CAST_FILE"
echo "2. Deploy to site: cp $CAST_FILE site/public/demo/demo.cast"
echo "3. Commit: git add site/public/demo/demo.cast && git commit -m 'feat(demo): Production recording with exact 9 commands'"
echo ""
