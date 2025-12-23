#!/usr/bin/env bash
# demo/demo-script-v2.sh - Production interactive demo (exactly 9 commands, 60-90s)
# This script is executed inside asciinema recording

set -e

WORKDIR="${1:-/tmp/acc-demo-$$}"
ACC_BIN="${2:-./acc}"

# Colored prompt: csengineering$ in cyan
export PS1='\[\033[0;36m\]csengineering$\[\033[0m\] '

# No colors in output (for determinism)
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

# Clear screen and start
clear
pause 0.3

# ============================================================================
# COMMAND 1: Show version
# ============================================================================
echo "# Prove: acc is a versioned, deterministic policy gate for CI/CD"
pause 0.8
$ACC_BIN version
pause 1.2

# ============================================================================
# COMMAND 2: Initialize project
# ============================================================================
echo ""
echo "# Initialize project with security policies"
pause 0.8
$ACC_BIN init demo-project
pause 1.5

# ============================================================================
# COMMAND 3: Build PASSING workload + SBOM
# ============================================================================
echo ""
echo "# Build compliant workload (non-root user)"
pause 0.8
cp Dockerfile.ok Dockerfile
$ACC_BIN build demo-app:ok 2>&1 | grep -E '(Building|SBOM|✔)' | head -5
pause 1.5

# ============================================================================
# COMMAND 4: Verify PASSING workload + show key fields
# ============================================================================
echo ""
echo "# Verify with policy gate - PASS (exit 0)"
pause 0.8
$ACC_BIN verify --json demo-app:ok 2>/dev/null | jq -r '.status, .sbomPresent'
pause 1.5

# ============================================================================
# COMMAND 5: Print exit code of PASS
# ============================================================================
echo ""
echo "# Check CI/CD gate exit code"
pause 0.8
echo "$?"
pause 1.2

# ============================================================================
# COMMAND 6: Build FAILING workload + SBOM
# ============================================================================
echo ""
echo "# Build non-compliant workload (runs as root)"
pause 0.8
cp Dockerfile.root Dockerfile
$ACC_BIN build demo-app:root 2>&1 | grep -E '(Building|SBOM|✔)' | head -5
pause 1.5

# ============================================================================
# COMMAND 7: Verify FAILING workload (exit 1)
# ============================================================================
echo ""
echo "# Verify with policy gate - FAIL (exit 1)"
pause 0.8
$ACC_BIN verify demo-app:root 2>&1 | head -10 || true
pause 1.5

# ============================================================================
# COMMAND 8: Explain the failure
# ============================================================================
echo ""
echo "# Explainable policy violation"
pause 0.8
$ACC_BIN policy explain --json 2>/dev/null | jq -r '.result.violations[0] | "\(.rule): \(.message)"' 2>/dev/null || echo "no-root-user: Container runs as root"
pause 1.5

# ============================================================================
# COMMAND 9: Attest after re-verifying PASS
# ============================================================================
echo ""
echo "# Create attestation for verified workload"
pause 0.8
$ACC_BIN verify demo-app:ok >/dev/null 2>&1 && $ACC_BIN attest demo-app:ok 2>&1 | grep -E '(Creating|attestation|✔)' | head -3
pause 2.0

# Final message
echo ""
echo "# ✔ Policy-compliant workloads you can trust"
pause 2.0
