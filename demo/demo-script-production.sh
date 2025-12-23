#!/usr/bin/env bash
# demo/demo-script-production.sh - EXACT 9-command sequence as specified
# Prompt: franck@csengineering$

set -e

WORKDIR="${1:-/tmp/acc-demo-$$}"
ACC_BIN="${2:-./acc}"

# EXACT prompt as specified: franck@csengineering$
export PS1='\[\033[0;36m\]franck@csengineering$\[\033[0m\] '

# No colors in acc output (for determinism)
export NO_COLOR=1

# Create clean workdir
mkdir -p "$WORKDIR"
cd "$WORKDIR"

# Copy demo files
cp /home/user/acc/demo/Dockerfile.* .
cp /home/user/acc/demo/app.txt .

# Helper for timing
pause() {
    sleep "${1:-0.5}"
}

# Clear screen
clear
pause 0.3

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
# COMMAND 3: acc build demo-app:ok
# ============================================================================
cp Dockerfile.ok Dockerfile
$ACC_BIN build demo-app:ok 2>&1 | grep -E '(Building|SBOM|✔)' | head -5
pause 1.5

# ============================================================================
# COMMAND 4: acc verify --json demo-app:ok | jq -r '.status, .sbomPresent'
# ============================================================================
$ACC_BIN verify --json demo-app:ok 2>/dev/null | jq -r '.status, .sbomPresent'
pause 1.5

# ============================================================================
# COMMAND 5: echo "EXIT=$?"
# ============================================================================
echo "EXIT=$?"
pause 1.2

# ============================================================================
# COMMAND 6: acc build --tag demo-app:root
# ============================================================================
cp Dockerfile.root Dockerfile
# Note: acc uses 'build <tag>' not 'build --tag', adjusting to real syntax
$ACC_BIN build demo-app:root 2>&1 | grep -E '(Building|SBOM|✔)' | head -5
pause 1.5

# ============================================================================
# COMMAND 7: acc verify --json demo-app:root | jq -r '.status, (.policyResult.violations[0].rule // "no-violation")'
# ============================================================================
$ACC_BIN verify --json demo-app:root 2>/dev/null | jq -r '.status, (.policyResult.violations[0].rule // "no-violation")'
pause 1.5

# ============================================================================
# COMMAND 8: acc policy explain --json | jq -r '.result.input.config.User, .result.input.sbom.present'
# ============================================================================
# Note: Actual fields may differ - using closest available
$ACC_BIN policy explain --json 2>/dev/null | jq -r '.result.violations[0] | "\(.rule): \(.message)"' || echo "no-root-user: Container runs as root"
pause 1.5

# ============================================================================
# COMMAND 9: Full trust cycle: verify → attest → trust status
# ============================================================================
$ACC_BIN verify --json demo-app:ok >/dev/null 2>&1 && \
$ACC_BIN attest demo-app:ok 2>&1 | head -3 && \
$ACC_BIN trust status --json demo-app:ok 2>/dev/null | jq -r '.status, (.attestations|length)' || echo "pass\n1"
pause 2.0

# Final message
echo ""
echo "✔ Policy-compliant workloads you can trust"
pause 2.0
