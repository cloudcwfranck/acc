#!/usr/bin/env bash
# demo/run.sh - Deterministic demo validation script
# This script validates that the acc demo workflow works correctly
# and can be used as a gate in CI

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ACC_BIN="${ACC_BIN:-$REPO_ROOT/acc}"
WORKDIR="${WORKDIR:-/tmp/acc-demo-$(date +%s)}"
FAILED=0

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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
    echo "========================================"
    echo "$*"
    echo "========================================"
}

# Cleanup on exit
cleanup() {
    local exit_code=$?
    if [ $exit_code -ne 0 ] || [ $FAILED -ne 0 ]; then
        echo ""
        log_error "Demo validation FAILED"
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

log_section "STEP 1: Initialize Project"

$ACC_BIN init demo-project

# Verify init created required structure
if [ -d ".acc" ]; then
    log ".acc directory created"
else
    log_error ".acc directory not created"
fi

if [ -d ".acc/profiles" ]; then
    log ".acc/profiles directory created"
else
    log_error ".acc/profiles directory not created"
fi

if [ -f "acc.yaml" ]; then
    log "acc.yaml created"
else
    log_error "acc.yaml not created"
fi

log_section "STEP 2: Build and Verify demo-app:ok (PASS)"

# Build passing image
cp Dockerfile.ok Dockerfile
$ACC_BIN build demo-app:ok > /dev/null 2>&1

if [ -f ".acc/sbom/demo-project.spdx.json" ]; then
    log "SBOM generated"
else
    log_error "SBOM not generated"
fi

# Verify passing image
set +e
verify_ok_output=$($ACC_BIN verify --json demo-app:ok 2>&1)
verify_ok_exit=$?
set -e

if [ $verify_ok_exit -eq 0 ]; then
    log "verify demo-app:ok: exit 0 (PASS)"
else
    log_error "verify demo-app:ok: exit $verify_ok_exit (expected 0)"
fi

# Check JSON status
status=$(echo "$verify_ok_output" | jq -r '.status' 2>/dev/null || echo "")
if [ "$status" == "pass" ]; then
    log "verify demo-app:ok: status='pass'"
else
    log_error "verify demo-app:ok: status='$status' (expected 'pass')"
fi

log_section "STEP 3: Build and Verify demo-app:root (FAIL)"

# Build failing image
cp Dockerfile.root Dockerfile
$ACC_BIN build demo-app:root > /dev/null 2>&1

# Verify failing image
set +e
verify_root_output=$($ACC_BIN verify --json demo-app:root 2>&1)
verify_root_exit=$?
set -e

if [ $verify_root_exit -eq 1 ]; then
    log "verify demo-app:root: exit 1 (FAIL)"
else
    log_error "verify demo-app:root: exit $verify_root_exit (expected 1)"
fi

# Check JSON status
status=$(echo "$verify_root_output" | jq -r '.status' 2>/dev/null || echo "")
if [ "$status" == "fail" ]; then
    log "verify demo-app:root: status='fail'"
else
    log_error "verify demo-app:root: status='$status' (expected 'fail')"
fi

# Check for no-root-user violation
if echo "$verify_root_output" | jq -e '.policyResult.violations[] | select(.rule == "no-root-user")' > /dev/null 2>&1; then
    log "verify demo-app:root: includes 'no-root-user' violation"
else
    log_error "verify demo-app:root: missing 'no-root-user' violation"
fi

log_section "STEP 4: Policy Explain"

set +e
explain_output=$($ACC_BIN policy explain --json 2>&1)
explain_exit=$?
set -e

if [ $explain_exit -eq 0 ]; then
    log "policy explain: exit 0"
else
    log "policy explain: exit $explain_exit (might be expected)"
fi

# Check for .result.input object (v0.2.7 contract)
if echo "$explain_output" | jq -e '.result.input' > /dev/null 2>&1; then
    log "policy explain: includes .result.input"
else
    log_error "policy explain: missing .result.input"
fi

log_section "STEP 5: Attestation (Mismatch Safety)"

# After verifying root, attempting to attest ok should FAIL
set +e
attest_mismatch_output=$($ACC_BIN attest demo-app:ok 2>&1)
attest_mismatch_exit=$?
set -e

if [ $attest_mismatch_exit -ne 0 ]; then
    log "attest demo-app:ok after verifying root: failed (expected)"
else
    log_error "attest demo-app:ok after verifying root: succeeded (should fail - mismatch)"
fi

# Verify it did NOT print "Creating attestation"
if echo "$attest_mismatch_output" | grep -q "Creating attestation"; then
    log_error "attest mismatch printed 'Creating attestation' (should not)"
else
    log "attest mismatch: correctly did not print 'Creating attestation'"
fi

log_section "STEP 6: Attestation (Success)"

# Re-verify ok, then attest should succeed
$ACC_BIN verify demo-app:ok > /dev/null 2>&1

set +e
attest_ok_output=$($ACC_BIN attest demo-app:ok 2>&1)
attest_ok_exit=$?
set -e

if [ $attest_ok_exit -eq 0 ]; then
    log "attest demo-app:ok: exit 0 (success)"
else
    log_error "attest demo-app:ok: exit $attest_ok_exit (expected 0)"
fi

# Verify it DID print "Creating attestation"
if echo "$attest_ok_output" | grep -q "Creating attestation"; then
    log "attest success: correctly printed 'Creating attestation'"
else
    log_error "attest success: did not print 'Creating attestation'"
fi

log_section "STEP 7: Trust Status (Unknown)"

# Build never-verified image
cp Dockerfile.ok Dockerfile
$ACC_BIN build demo-app:never-verified > /dev/null 2>&1

set +e
status_never_output=$($ACC_BIN trust status --json demo-app:never-verified 2>&1)
status_never_exit=$?
set -e

if [ $status_never_exit -eq 2 ]; then
    log "trust status demo-app:never-verified: exit 2 (unknown)"
else
    log_error "trust status demo-app:never-verified: exit $status_never_exit (expected 2)"
fi

# Check for sbomPresent field
if echo "$status_never_output" | jq -e '.sbomPresent' > /dev/null 2>&1; then
    log "trust status demo-app:never-verified: includes sbomPresent field"
else
    log_error "trust status demo-app:never-verified: missing sbomPresent field"
fi

# Verify status is unknown
status=$(echo "$status_never_output" | jq -r '.status' 2>/dev/null || echo "")
if [ "$status" == "unknown" ]; then
    log "trust status demo-app:never-verified: status='unknown'"
else
    log_error "trust status demo-app:never-verified: status='$status' (expected 'unknown')"
fi

log_section "STEP 8: Trust Verify (v0.3.0)"

# Verify attestations for ok image
set +e
verify_attest_output=$($ACC_BIN trust verify --json demo-app:ok 2>&1)
verify_attest_exit=$?
set -e

if [ $verify_attest_exit -eq 0 ]; then
    log "trust verify demo-app:ok: exit 0 (verified)"
elif [ $verify_attest_exit -eq 1 ]; then
    log "trust verify demo-app:ok: exit 1 (unverified - acceptable if no attestations)"
else
    log_error "trust verify demo-app:ok: exit $verify_attest_exit (expected 0 or 1)"
fi

# Check JSON schema
if echo "$verify_attest_output" | jq -e '.schemaVersion' > /dev/null 2>&1; then
    log "trust verify: includes schemaVersion"
else
    log_error "trust verify: missing schemaVersion"
fi

if echo "$verify_attest_output" | jq -e '.verificationStatus' > /dev/null 2>&1; then
    log "trust verify: includes verificationStatus"
else
    log_error "trust verify: missing verificationStatus"
fi

log_section "SUMMARY"

if [ $FAILED -eq 0 ]; then
    log "All checks passed! Demo is ready for recording."
    echo ""
    echo "To record the demo:"
    echo "  cd $REPO_ROOT"
    echo "  bash demo/record.sh"
    exit 0
else
    log_error "$FAILED checks failed"
    exit 1
fi
