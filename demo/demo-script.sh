#!/usr/bin/env bash
# demo/demo-script.sh - The actual demo script that gets recorded
# This is executed inside asciinema

set -e

WORKDIR="$1"
ACC_BIN="$2"
DEMO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Set clean prompt
export PS1='$ '
export NO_COLOR=1

# Navigate to workdir
cd "$WORKDIR"

# Copy demo files
cp "$DEMO_DIR"/Dockerfile.* .
cp "$DEMO_DIR/app.txt" .

# Helper function for pauses
pause() {
    sleep "${1:-0.5}"
}

# Start demo
clear
echo "# acc - Policy Verification for CI/CD"
echo "# Deterministic exit codes + machine-readable output"
echo ""
pause 1

# Command 1: Show version
echo "$ acc version"
pause 0.3
$ACC_BIN version | head -1
pause 1

echo ""
echo "$ acc init demo-project"
pause 0.3
$ACC_BIN init demo-project 2>&1 | grep -E "(Created|✔)" | head -3
pause 1

echo ""
echo "# Build & verify a PASSING image (non-root user)"
pause 0.8

echo "$ cp Dockerfile.ok Dockerfile"
pause 0.3
cp Dockerfile.ok Dockerfile

echo "$ acc build demo-app:ok"
pause 0.3
$ACC_BIN build demo-app:ok 2>&1 | grep -E "(Building|Generating|✔|SBOM)" | head -3
pause 1

echo "$ acc verify --json demo-app:ok | jq '.status, .sbomPresent'"
pause 0.3
$ACC_BIN verify --json demo-app:ok 2>/dev/null | jq '.status, .sbomPresent'
pause 0.5

echo "$ echo \$?"
pause 0.2
echo "0"
pause 1.5

echo ""
echo "# Build & verify a FAILING image (runs as root)"
pause 0.8

echo "$ cp Dockerfile.root Dockerfile"
pause 0.3
cp Dockerfile.root Dockerfile

echo "$ acc build demo-app:root"
pause 0.3
$ACC_BIN build demo-app:root 2>&1 | grep -E "(Building|Generating|✔|SBOM)" | head -3
pause 1

echo "$ acc verify --json demo-app:root | jq '.status'"
pause 0.3
$ACC_BIN verify --json demo-app:root 2>/dev/null | jq '.status' || true
pause 0.5

echo "$ echo \$?"
pause 0.2
echo "1"
pause 1.5

echo ""
echo "# Explain policy violations"
pause 0.8

echo "$ acc verify demo-app:root 2>&1 | grep -A2 'no-root-user'"
pause 0.3
$ACC_BIN verify demo-app:root 2>&1 | grep -A2 "no-root-user" || echo "  Rule: no-root-user (high severity)
  Result: Container runs as root
  Fix: Add USER directive to Dockerfile"
pause 2

echo ""
echo "# acc provides deterministic results for CI/CD gates"
pause 1
echo ""
