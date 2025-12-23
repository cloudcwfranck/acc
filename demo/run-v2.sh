#!/usr/bin/env bash
# demo/run-v2.sh - Validates the 9-command demo workflow
# This script ensures deterministic, reproducible demo behavior

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ACC_BIN="${ACC_BIN:-$REPO_ROOT/acc}"
WORKDIR="${WORKDIR:-/tmp/acc-demo-validate-$(date +%s)}"
FAILED=0

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}✓${NC} $*"
}

log_error() {
    echo -e "${RED}✗${NC} $*"
    FAILED=$((FAILED + 1))
}

log_section() {
    echo ""
    echo -e "${CYAN}========================================${NC}"
    echo -e "${CYAN}$*${NC}"
    echo -e "${CYAN}========================================${NC}"
}

# Cleanup on exit
cleanup() {
    local exit_code=$?
    if [ $exit_code -ne 0 ] || [ $FAILED -ne 0 ]; then
        echo ""
        log_error "Demo validation FAILED ($FAILED checks failed)"
        echo "Workdir preserved at: $WORKDIR"
        exit 1
    else
        log "Demo validation PASSED"
        echo "Cleaning up workdir: $WORKDIR"
        rm -rf "$WORKDIR"
    fi
}

trap cleanup EXIT

# Build acc if needed
if [ ! -f "$ACC_BIN" ]; then
    log_section "Building acc"
    cd "$REPO_ROOT"
    go build -o "$ACC_BIN" ./cmd/acc
    log "Built acc at $ACC_BIN"
fi

# Create workdir
mkdir -p "$WORKDIR"
cd "$WORKDIR"
log "Working directory: $WORKDIR"

# Copy demo files
cp "$SCRIPT_DIR"/Dockerfile.* .
cp "$SCRIPT_DIR/app.txt" .

log_section "COMMAND 1: acc version"
set +e
version_output=$($ACC_BIN version 2>&1)
version_exit=$?
set -e

if [ $version_exit -eq 0 ]; then
    log "version: exit 0"
else
    log_error "version: exit $version_exit (expected 0)"
fi

if echo "$version_output" | grep -qi "acc version"; then
    log "version: includes 'acc version'"
else
    log_error "version: missing version info"
fi

log_section "COMMAND 2: acc init demo-project"
$ACC_BIN init demo-project

if [ -d ".acc" ]; then
    log "init: .acc directory created"
else
    log_error "init: .acc directory not created"
fi

if [ -f "acc.yaml" ]; then
    log "init: acc.yaml created"
else
    log_error "init: acc.yaml not created"
fi

log_section "COMMAND 3: acc build demo-app:ok"
cp Dockerfile.ok Dockerfile
$ACC_BIN build demo-app:ok > /dev/null 2>&1

if [ -f ".acc/sbom/demo-project.spdx.json" ]; then
    log "build ok: SBOM generated"
else
    log_error "build ok: SBOM not generated"
fi

log_section "COMMAND 4: acc verify --json demo-app:ok + jq"
set +e
verify_ok_output=$($ACC_BIN verify --json demo-app:ok 2>&1)
verify_ok_exit=$?
set -e

if [ $verify_ok_exit -eq 0 ]; then
    log "verify ok: exit 0 (PASS)"
else
    log_error "verify ok: exit $verify_ok_exit (expected 0)"
fi

status=$(echo "$verify_ok_output" | jq -r '.status' 2>/dev/null || echo "")
if [ "$status" == "pass" ]; then
    log "verify ok: status='pass'"
else
    log_error "verify ok: status='$status' (expected 'pass')"
fi

sbom=$(echo "$verify_ok_output" | jq -r '.sbomPresent' 2>/dev/null || echo "")
if [ "$sbom" == "true" ]; then
    log "verify ok: sbomPresent=true"
else
    log_error "verify ok: sbomPresent='$sbom' (expected 'true')"
fi

log_section "COMMAND 5: echo \$? (exit code)"
# The exit code from previous command should be 0
if [ $verify_ok_exit -eq 0 ]; then
    log "exit code: 0 (PASS gate)"
else
    log_error "exit code: $verify_ok_exit (expected 0)"
fi

log_section "COMMAND 6: acc build demo-app:root"
cp Dockerfile.root Dockerfile
$ACC_BIN build demo-app:root > /dev/null 2>&1

if [ -f ".acc/sbom/demo-project.spdx.json" ]; then
    log "build root: SBOM generated"
else
    log_error "build root: SBOM not generated"
fi

log_section "COMMAND 7: acc verify demo-app:root (FAIL)"
set +e
verify_root_output=$($ACC_BIN verify demo-app:root 2>&1)
verify_root_exit=$?
set -e

if [ $verify_root_exit -eq 1 ]; then
    log "verify root: exit 1 (FAIL)"
else
    log_error "verify root: exit $verify_root_exit (expected 1)"
fi

if echo "$verify_root_output" | grep -qi "fail\|violation"; then
    log "verify root: includes failure indication"
else
    log_error "verify root: missing failure indication"
fi

log_section "COMMAND 8: acc policy explain (violation details)"
set +e
explain_output=$($ACC_BIN policy explain --json 2>&1)
explain_exit=$?
set -e

if [ $explain_exit -eq 0 ]; then
    log "policy explain: exit 0"
else
    log "policy explain: exit $explain_exit (acceptable)"
fi

if echo "$explain_output" | grep -qi "no-root-user\|root"; then
    log "policy explain: includes root user violation"
else
    log_error "policy explain: missing violation details"
fi

log_section "COMMAND 9: acc attest demo-app:ok"
# First re-verify
$ACC_BIN verify demo-app:ok > /dev/null 2>&1

set +e
attest_output=$($ACC_BIN attest demo-app:ok 2>&1)
attest_exit=$?
set -e

if [ $attest_exit -eq 0 ]; then
    log "attest: exit 0 (success)"
else
    log_error "attest: exit $attest_exit (expected 0)"
fi

if echo "$attest_output" | grep -qi "Creating attestation\|attestation"; then
    log "attest: includes attestation creation message"
else
    log_error "attest: missing attestation message"
fi

log_section "SUMMARY"

if [ $FAILED -eq 0 ]; then
    log "All 9 commands validated successfully!"
    echo ""
    echo "Demo is ready for recording:"
    echo "  cd $REPO_ROOT"
    echo "  bash demo/record-v2.sh"
    exit 0
else
    log_error "$FAILED checks failed"
    exit 1
fi
