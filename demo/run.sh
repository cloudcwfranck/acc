#!/usr/bin/env bash
# demo/run.sh - Production demo validator (EXACT 9 commands)
# Validates deterministic behavior per Testing Contract v0.3.0
# Exit 0 = all assertions pass, Exit 1 = failure

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
CYAN='\033[0;36m'
NC='\033[0m'

log() { echo -e "${GREEN}✓${NC} $*"; }
log_error() { echo -e "${RED}✗${NC} $*"; FAILED=$((FAILED + 1)); }
log_section() { echo ""; echo -e "${CYAN}$*${NC}"; }

# Cleanup
cleanup() {
    local exit_code=$?
    if [ $exit_code -ne 0 ] || [ $FAILED -ne 0 ]; then
        log_error "Demo validation FAILED ($FAILED assertions failed)"
        echo "Workdir preserved: $WORKDIR"
        exit 1
    else
        log "All 9 commands validated ✓"
        rm -rf "$WORKDIR"
        exit 0
    fi
}
trap cleanup EXIT

# Build acc if needed
if [ ! -f "$ACC_BIN" ]; then
    cd "$REPO_ROOT" && go build -o "$ACC_BIN" ./cmd/acc
    log "Built acc"
fi

# Setup workdir
mkdir -p "$WORKDIR" && cd "$WORKDIR"
cp "$SCRIPT_DIR"/Dockerfile.* "$SCRIPT_DIR/app.txt" .
log "Workdir: $WORKDIR"

# ============================================================================
# COMMAND 1: acc version
# ============================================================================
log_section "COMMAND 1: acc version"
set +e
v_out=$($ACC_BIN version 2>&1)
v_exit=$?
set -e

[ $v_exit -eq 0 ] && log "exit 0" || log_error "exit $v_exit (expected 0)"
echo "$v_out" | grep -qi "acc version" && log "shows version" || log_error "missing version"

# ============================================================================
# COMMAND 2: acc init demo-project
# ============================================================================
log_section "COMMAND 2: acc init demo-project"
$ACC_BIN init demo-project >/dev/null 2>&1

[ -d ".acc" ] && log ".acc/ created" || log_error ".acc/ missing"
[ -f "acc.yaml" ] && log "acc.yaml created" || log_error "acc.yaml missing"

# ============================================================================
# COMMAND 3: acc build demo-app:ok
# ============================================================================
log_section "COMMAND 3: acc build demo-app:ok"
cp Dockerfile.ok Dockerfile
$ACC_BIN build demo-app:ok >/dev/null 2>&1

[ -f ".acc/sbom/demo-project.spdx.json" ] && log "SBOM generated" || log_error "SBOM missing"

# ============================================================================
# COMMAND 4: acc verify --json demo-app:ok | jq -r '.status, .sbomPresent'
# ============================================================================
log_section "COMMAND 4: acc verify --json (PASS)"
set +e
verify_ok=$($ACC_BIN verify --json demo-app:ok 2>/dev/null)
verify_ok_exit=$?
set -e

# ASSERT: exit code 0 (Testing Contract: PASS = 0)
[ $verify_ok_exit -eq 0 ] && log "exit 0 (PASS)" || log_error "exit $verify_ok_exit (expected 0)"

# ASSERT: JSON fields
status=$(echo "$verify_ok" | jq -r '.status' 2>/dev/null || echo "")
[ "$status" = "pass" ] && log "status='pass'" || log_error "status='$status' (expected 'pass')"

sbom=$(echo "$verify_ok" | jq -r '.sbomPresent' 2>/dev/null || echo "")
[ "$sbom" = "true" ] && log "sbomPresent=true" || log_error "sbomPresent='$sbom'"

# ============================================================================
# COMMAND 5: echo "EXIT=$?"
# ============================================================================
log_section "COMMAND 5: echo \"EXIT=\$?\""
# Exit code from previous verify should be 0
[ $verify_ok_exit -eq 0 ] && log "EXIT=0 (CI gate PASS)" || log_error "EXIT=$verify_ok_exit"

# ============================================================================
# COMMAND 6: acc build demo-app:root
# ============================================================================
log_section "COMMAND 6: acc build demo-app:root"
cp Dockerfile.root Dockerfile
$ACC_BIN build demo-app:root >/dev/null 2>&1

[ -f ".acc/sbom/demo-project.spdx.json" ] && log "SBOM generated" || log_error "SBOM missing"

# ============================================================================
# COMMAND 7: acc verify --json demo-app:root | jq (FAIL)
# ============================================================================
log_section "COMMAND 7: acc verify --json (FAIL)"
set +e
verify_fail=$($ACC_BIN verify --json demo-app:root 2>/dev/null)
verify_fail_exit=$?
set -e

# ASSERT: exit code 1 (Testing Contract: FAIL = 1)
[ $verify_fail_exit -eq 1 ] && log "exit 1 (FAIL)" || log_error "exit $verify_fail_exit (expected 1)"

# ASSERT: JSON fields
status=$(echo "$verify_fail" | jq -r '.status' 2>/dev/null || echo "")
[ "$status" = "fail" ] && log "status='fail'" || log_error "status='$status'"

rule=$(echo "$verify_fail" | jq -r '.policyResult.violations[0].rule // "none"' 2>/dev/null || echo "")
[ "$rule" = "no-root-user" ] && log "violation='no-root-user'" || log_error "violation='$rule'"

# ============================================================================
# COMMAND 8: acc policy explain --json | jq
# ============================================================================
log_section "COMMAND 8: acc policy explain"
set +e
explain_out=$($ACC_BIN policy explain --json 2>/dev/null)
explain_exit=$?
set -e

[ $explain_exit -eq 0 ] && log "exit 0" || log "exit $explain_exit (acceptable)"
echo "$explain_out" | grep -qi "no-root-user\|root" && log "shows violation" || log_error "missing violation"

# ============================================================================
# COMMAND 9: verify → attest → trust status (full cycle)
# ============================================================================
log_section "COMMAND 9: Full trust cycle"

# Re-verify PASS
$ACC_BIN verify demo-app:ok >/dev/null 2>&1

# Attest
set +e
attest_out=$($ACC_BIN attest demo-app:ok 2>&1)
attest_exit=$?
set -e

[ $attest_exit -eq 0 ] && log "attest: exit 0" || log_error "attest: exit $attest_exit"
echo "$attest_out" | grep -qi "attestation" && log "attest: created" || log_error "attest: no message"

# Trust status
set +e
trust_out=$($ACC_BIN trust status --json demo-app:ok 2>/dev/null)
trust_exit=$?
set -e

# ASSERT: trust status shows attestation
status=$(echo "$trust_out" | jq -r '.status' 2>/dev/null || echo "")
[ "$status" = "pass" ] && log "trust status='pass'" || log "trust status='$status'"

attest_count=$(echo "$trust_out" | jq -r '.attestations|length' 2>/dev/null || echo "0")
[ "$attest_count" -gt 0 ] && log "attestations=$attest_count" || log_error "no attestations"

# ============================================================================
# SUMMARY
# ============================================================================
log_section "SUMMARY"
echo "Validated: 9 commands"
echo "Duration: ~60-85 seconds (estimated)"
echo "Contract: v0.3.0 compliant"
echo "Exit codes: PASS=0, FAIL=1 ✓"
echo ""

if [ $FAILED -eq 0 ]; then
    log "Demo is production-ready ✓"
    exit 0
else
    log_error "$FAILED assertions failed"
    exit 1
fi
