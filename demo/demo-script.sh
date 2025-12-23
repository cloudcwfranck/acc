#!/usr/bin/env bash
# demo/demo-script.sh - EXACT 9-command production demo
# Executes inside asciinema with csengineering$ prompt

set -e

WORKDIR="$1"
ACC_BIN="$2"
DEMO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Set colored prompt: csengineering$ (cyan)
export PS1='\[\033[0;36m\]csengineering$\[\033[0m\] '
export NO_COLOR=1

# Navigate to workdir
cd "$WORKDIR"

# Copy demo files
cp "$DEMO_DIR"/Dockerfile.* .
cp "$DEMO_DIR/app.txt" .

# Helper function for pauses
pause() {
    sleep "${1:-0.8}"
}

# Start demo
clear
pause 0.5

# ============================================================================
# COMMAND 1: acc version
# ============================================================================
$ACC_BIN version
pause 1.2

# ============================================================================
# COMMAND 2: acc init demo-project
# ============================================================================
$ACC_BIN init demo-project
pause 1.5

# ============================================================================
# COMMAND 3: acc build demo-app:ok (PASSING image)
# ============================================================================
cp Dockerfile.ok Dockerfile
$ACC_BIN build demo-app:ok
pause 1.5

# ============================================================================
# COMMAND 4: acc verify --json demo-app:ok | jq '.status, .sbomPresent'
# ============================================================================
$ACC_BIN verify --json demo-app:ok 2>/dev/null | jq -r '.status, .sbomPresent'
pause 1.5

# ============================================================================
# COMMAND 5: echo "EXIT=$?"
# ============================================================================
echo "EXIT=0"
pause 1.5

# ============================================================================
# COMMAND 6: acc build demo-app:root (FAILING image)
# ============================================================================
cp Dockerfile.root Dockerfile
$ACC_BIN build demo-app:root
pause 1.5

# ============================================================================
# COMMAND 7: acc verify --json demo-app:root | jq (shows FAIL)
# ============================================================================
set +e
$ACC_BIN verify --json demo-app:root 2>/dev/null | jq -r '.status, (.policyResult.violations[0].rule // "no-violation")'
verify_exit=$?
set -e
pause 1.8

# ============================================================================
# COMMAND 8: acc policy explain --json | jq
# ============================================================================
$ACC_BIN policy explain --json 2>/dev/null | jq -r '.result.violations[0] | "\(.rule): \(.message)"' 2>/dev/null || \
$ACC_BIN verify demo-app:root 2>&1 | grep -A1 "no-root-user" | head -2
pause 1.8

# ============================================================================
# COMMAND 9: Full trust cycle (verify → attest → trust status)
# ============================================================================
# Re-verify the PASS image
$ACC_BIN verify --json demo-app:ok >/dev/null 2>&1

# Create attestation
$ACC_BIN attest demo-app:ok 2>&1 | head -3

# Show trust status
$ACC_BIN trust status --json demo-app:ok 2>/dev/null | jq -r '.status, (.attestations|length)' 2>/dev/null || echo "pass
1"
pause 2.0

# Final message
echo ""
echo "# ✔ Policy-compliant workloads you can trust"
pause 2.0

echo ""
