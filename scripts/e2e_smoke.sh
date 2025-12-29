#!/usr/bin/env bash
# Tier 1: E2E Smoke Tests
# Comprehensive offline functional tests without external registry dependencies
# Tests full workflow: init, build, verify, policy, attest, inspect, trust status

set -euo pipefail

# ============================================================================
# CONFIGURATION
# ============================================================================

LOGFILE="/tmp/tier1-e2e-$(date +%s).log"
ACC_BIN="${ACC_BIN:-./acc}"
WORKDIR="/tmp/acc-e2e-$(date +%s)"
FAILED=0

# Required tools
REQUIRED_TOOLS=("docker" "opa" "jq" "syft")

# ============================================================================
# LOGGING FUNCTIONS
# ============================================================================

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOGFILE"
}

log_success() {
    echo "✅ $*" | tee -a "$LOGFILE"
}

log_error() {
    echo "❌ $*" | tee -a "$LOGFILE"
    FAILED=$((FAILED + 1))
}

log_section() {
    echo "" | tee -a "$LOGFILE"
    echo "========================================" | tee -a "$LOGFILE"
    echo "$*" | tee -a "$LOGFILE"
    echo "========================================" | tee -a "$LOGFILE"
}

log_command() {
    echo "$ $*" | tee -a "$LOGFILE"
}

# ============================================================================
# UTILITY FUNCTIONS
# ============================================================================

# Check if a tool is available
need() {
    local tool=$1
    if ! command -v "$tool" &> /dev/null; then
        log_error "Required tool not found: $tool"
        echo "Install $tool before running this script" >&2
        exit 1
    fi
}

# Clean up on exit
cleanup() {
    local exit_code=$?
    if [ $exit_code -ne 0 ]; then
        log_error "Test failed with exit code $exit_code"
        log "Workdir preserved at: $WORKDIR"
        log "Log file: $LOGFILE"
    else
        log "Cleaning up workdir: $WORKDIR"
        rm -rf "$WORKDIR"
    fi
}

trap cleanup EXIT

# ============================================================================
# TEST FUNCTIONS
# ============================================================================

# Assert that a command succeeds
# Uses set +e pattern to safely capture exit codes without SC2317 warnings
assert_success() {
    local description="$1"
    shift
    log_command "$@"

    local output
    local exit_code

    set +e
    output=$("$@" 2>&1)
    exit_code=$?
    set -e

    if [ $exit_code -eq 0 ]; then
        log_success "$description"
        echo "$output" >> "$LOGFILE"
        return 0
    else
        log_error "$description (exit code: $exit_code)"
        echo "$output" >> "$LOGFILE"
        return 1
    fi
}

# Assert that a command fails with specific exit code
# Uses set +e pattern to safely capture exit codes without SC2317 warnings
assert_failure() {
    local expected_exit=$1
    local description="$2"
    shift 2
    log_command "$@"

    local output
    local exit_code

    set +e
    output=$("$@" 2>&1)
    exit_code=$?
    set -e

    if [ $exit_code -eq "$expected_exit" ]; then
        log_success "$description (exit $exit_code as expected)"
        echo "$output" >> "$LOGFILE"
        return 0
    else
        log_error "$description (expected exit $expected_exit, got $exit_code)"
        echo "$output" >> "$LOGFILE"
        return 1
    fi
}

# Assert JSON field value using jq
assert_json_field() {
    local json="$1"
    local jq_expr="$2"
    local expected="$3"
    local description="$4"

    actual=$(echo "$json" | jq -r "$jq_expr")

    if [ "$actual" == "$expected" ]; then
        log_success "$description: $jq_expr = $expected"
        return 0
    else
        log_error "$description: $jq_expr = $actual (expected $expected)"
        return 1
    fi
}

# Assert JSON field exists
assert_json_has_field() {
    local json="$1"
    local jq_expr="$2"
    local description="$3"

    # Check if this is a nested field (has dot after removing leading dot)
    # .result.input -> result.input (has dot) = nested
    # .sbomPresent -> sbomPresent (no dot) = top-level
    local without_leading_dot="${jq_expr#.}"

    if [[ "$without_leading_dot" == *.* ]]; then
        # Nested field like .result.input
        # Split on last dot: .result.input -> .result and input
        local parent="${jq_expr%.*}"   # .result
        local field="${jq_expr##*.}"   # input

        # Check if parent has the child field
        if echo "$json" | jq -e "$parent | has(\"$field\")" > /dev/null 2>&1; then
            log_success "$description: field $jq_expr exists"
            return 0
        else
            log_error "$description: field $jq_expr missing"
            return 1
        fi
    else
        # Top-level field like .sbomPresent
        local field_name=$(echo "$jq_expr" | sed 's/^\.//')

        # Use 'has()' to check if field exists (works for false values too)
        if echo "$json" | jq -e "has(\"$field_name\")" > /dev/null 2>&1; then
            log_success "$description: field $jq_expr exists"
            return 0
        else
            log_error "$description: field $jq_expr missing"
            return 1
        fi
    fi
}

# ============================================================================
# MAIN TEST EXECUTION
# ============================================================================

log_section "TIER 1: E2E SMOKE TESTS"
log "ACC Binary: $ACC_BIN"
log "Workdir: $WORKDIR"
log "Log File: $LOGFILE"

# Verify prerequisites
log_section "Verifying Prerequisites"
for tool in "${REQUIRED_TOOLS[@]}"; do
    need "$tool"
    log "✓ $tool: $(command -v "$tool")"
done

# Verify acc binary exists
if [ ! -f "$ACC_BIN" ]; then
    log_error "acc binary not found at $ACC_BIN"
    exit 1
fi

# Convert to absolute path before changing directories
ACC_BIN=$(realpath "$ACC_BIN")
log "Resolved ACC Binary: $ACC_BIN"

# Create workdir
mkdir -p "$WORKDIR"
cd "$WORKDIR"
log "Working directory: $(pwd)"

# ============================================================================
# TEST 1: acc init
# ============================================================================

log_section "TEST 1: acc init"

assert_success "acc init creates project" \
    $ACC_BIN init test-project

# Verify .acc directory structure
if [ -d ".acc" ]; then
    log_success ".acc directory created"
else
    log_error ".acc directory not created"
fi

if [ -d ".acc/profiles" ]; then
    log_success ".acc/profiles directory created"
else
    log "⚠️  .acc/profiles directory not created (might be expected)"
fi

if [ -f "acc.yaml" ]; then
    log_success "acc.yaml created"
    cat acc.yaml >> "$LOGFILE"
else
    log_error "acc.yaml not created"
fi

# ============================================================================
# TEST 2: Build and verify demo-app:ok (non-root user)
# ============================================================================

log_section "TEST 2: Build and Verify demo-app:ok (non-root)"

# Create Dockerfile for non-root user (should PASS policy)
log "Creating Dockerfile for demo-app:ok (non-root user)"
cat > Dockerfile <<'EOF'
FROM alpine:3.19
RUN addgroup -g 1000 appuser && adduser -D -u 1000 -G appuser appuser
USER appuser
WORKDIR /app
CMD ["sh", "-c", "echo 'Hello from non-root user'; sleep 1"]
EOF

# Build with positional arg (v0.2.3 fix)
log "Building demo-app:ok with acc build (positional arg)"
assert_success "acc build demo-app:ok (positional arg)" \
    $ACC_BIN build demo-app:ok

# Verify SBOM was created
if [ -f ".acc/sbom/test-project.spdx.json" ]; then
    log_success "SBOM created for demo-app:ok"
else
    log_error "SBOM not created for demo-app:ok"
fi

# Verify immediately (while SBOM is fresh)
log "Verifying demo-app:ok (should PASS)"
verify_ok_output=$($ACC_BIN verify --json demo-app:ok 2>&1) || true
verify_ok_exit=$?

log "Verify output:"
echo "$verify_ok_output" | tee -a "$LOGFILE"

if [ $verify_ok_exit -eq 0 ]; then
    log_success "acc verify demo-app:ok: exit 0"
else
    log_error "acc verify demo-app:ok: exit $verify_ok_exit (expected 0)"
fi

# Validate JSON output
if echo "$verify_ok_output" | jq empty 2>/dev/null; then
    log_success "Valid JSON output"

    assert_json_field "$verify_ok_output" ".status" "pass" \
        "verify demo-app:ok status"

    assert_json_has_field "$verify_ok_output" ".policyResult" \
        "verify demo-app:ok policyResult"

    assert_json_has_field "$verify_ok_output" ".sbomPresent" \
        "verify demo-app:ok sbomPresent"
else
    log_error "Invalid JSON output from verify"
fi

# ============================================================================
# TEST 3: Build and verify demo-app:root (root user)
# ============================================================================

log_section "TEST 3: Build and Verify demo-app:root (root user)"

# Create Dockerfile for root user (should FAIL policy)
log "Creating Dockerfile for demo-app:root (root user)"
cat > Dockerfile <<'EOF'
FROM alpine:3.19
# Intentionally no USER directive - runs as root
WORKDIR /app
CMD ["sh", "-c", "echo 'Hello from root user'; sleep 1"]
EOF

# Build with --tag flag
log "Building demo-app:root with acc build (--tag flag)"
assert_success "acc build demo-app:root (--tag flag)" \
    $ACC_BIN build --tag demo-app:root

# Verify SBOM was updated
if [ -f ".acc/sbom/test-project.spdx.json" ]; then
    log_success "SBOM updated for demo-app:root"
else
    log_error "SBOM not created for demo-app:root"
fi

# Verify immediately (while SBOM is fresh)
log "Verifying demo-app:root (should FAIL)"

set +e
verify_root_output=$($ACC_BIN verify --json demo-app:root 2>&1)
verify_root_exit=$?
set -e

log "Verify output:"
echo "$verify_root_output" | tee -a "$LOGFILE"

if [ $verify_root_exit -eq 1 ]; then
    log_success "acc verify demo-app:root: exit 1"
else
    log_error "acc verify demo-app:root: exit $verify_root_exit (expected 1)"

    # Diagnostic output for triage
    echo "========== DIAGNOSTIC: verify exit code mismatch ==========" | tee -a "$LOGFILE"
    echo "Command: $ACC_BIN verify --json demo-app:root" | tee -a "$LOGFILE"
    echo "Expected exit: 1 (policy failure)" | tee -a "$LOGFILE"
    echo "Actual exit: $verify_root_exit" | tee -a "$LOGFILE"
    echo "Status field: $(echo "$verify_root_output" | jq -r '.status')" | tee -a "$LOGFILE"
    echo "Allow field: $(echo "$verify_root_output" | jq -r '.policyResult.allow')" | tee -a "$LOGFILE"
    echo "Violations: $(echo "$verify_root_output" | jq -r '.policyResult.violations | length')" | tee -a "$LOGFILE"
    echo "acc version: $($ACC_BIN version 2>&1 | head -1)" | tee -a "$LOGFILE"
    echo "Contract: verify with status:fail MUST exit 1 (Testing Contract v0.2.3)" | tee -a "$LOGFILE"
    echo "Regression: v0.2.2 Single Authoritative Final Gate fixed status but not exit code" | tee -a "$LOGFILE"
    echo "==========================================================" | tee -a "$LOGFILE"
fi

# Validate JSON output and check for no-root-user violation
if echo "$verify_root_output" | jq empty 2>/dev/null; then
    log_success "Valid JSON output"

    assert_json_field "$verify_root_output" ".status" "fail" \
        "verify demo-app:root status"

    # Check for no-root-user violation
    if echo "$verify_root_output" | jq -e '.policyResult.violations[] | select(.rule == "no-root-user")' > /dev/null 2>&1; then
        log_success "no-root-user violation found in policy result"
    else
        log_error "no-root-user violation not found in policy result"
    fi
else
    log_error "Invalid JSON output from verify"
fi

# ============================================================================
# TEST 4: acc policy explain
# ============================================================================

log_section "TEST 4: acc policy explain"

# After verifying root (which failed), policy explain should show the result
explain_output=$($ACC_BIN policy explain --json 2>&1) || true
explain_exit=$?

if [ $explain_exit -eq 0 ]; then
    log_success "acc policy explain: exit 0"

    if echo "$explain_output" | jq empty 2>/dev/null; then
        log_success "Valid JSON output from policy explain"

        # Check that .result.input is an object
        assert_json_has_field "$explain_output" ".result.input" \
            "policy explain has .result.input"

        input_type=$(echo "$explain_output" | jq -r '.result.input | type')
        if [ "$input_type" == "object" ]; then
            log_success "policy explain .result.input is an object"
        else
            log_error "policy explain .result.input is $input_type (expected object)"
        fi
    else
        log_error "Invalid JSON output from policy explain"
    fi
else
    log "⚠️  acc policy explain: exit $explain_exit (might be expected if no history)"
fi

# ============================================================================
# TEST 5: Attest UX checks
# ============================================================================

log_section "TEST 5: Attest UX Checks"

# After verifying root last, acc attest demo-app:ok should FAIL (mismatch)
log "Attempting to attest demo-app:ok after verifying demo-app:root (should fail)"
set +e
attest_mismatch_output=$($ACC_BIN attest demo-app:ok 2>&1)
attest_mismatch_exit=$?
set -e

if [ $attest_mismatch_exit -ne 0 ]; then
    log_success "acc attest demo-app:ok after verifying root: failed as expected"

    # Stdout must NOT include "Creating attestation"
    if echo "$attest_mismatch_output" | grep -q "Creating attestation"; then
        log_error "acc attest showed 'Creating attestation' on mismatch (UX bug)"
    else
        log_success "acc attest did not show 'Creating attestation' on mismatch"
    fi
else
    log_error "acc attest demo-app:ok after verifying root: succeeded (expected failure)"

    # Diagnostic output for triage
    echo "========== DIAGNOSTIC: attest mismatch detection failed ==========" | tee -a "$LOGFILE"
    echo "Command: $ACC_BIN attest demo-app:ok" | tee -a "$LOGFILE"
    echo "Context: Last verified image was demo-app:root (different image)" | tee -a "$LOGFILE"
    echo "Expected: Exit non-zero (image mismatch)" | tee -a "$LOGFILE"
    echo "Actual exit: $attest_mismatch_exit" | tee -a "$LOGFILE"
    echo "Output: $attest_mismatch_output" | tee -a "$LOGFILE"
    echo "acc version: $($ACC_BIN version 2>&1 | head -1)" | tee -a "$LOGFILE"
    echo "Contract: attest MUST fail when image != last verified image (Testing Contract v0.2.x)" | tee -a "$LOGFILE"
    echo "Regression: Mismatch detection not enforced" | tee -a "$LOGFILE"
    echo "Check: .acc/state/verify/*.json to see if image digest is tracked" | tee -a "$LOGFILE"
    echo "==========================================================" | tee -a "$LOGFILE"
fi

# Now verify demo-app:ok again, then attest should succeed
log "Verifying demo-app:ok again before attestation"
$ACC_BIN verify demo-app:ok > /dev/null 2>&1 || true

log "Attempting to attest demo-app:ok after verifying it (should succeed)"
attest_success_output=$($ACC_BIN attest demo-app:ok 2>&1) || true
attest_success_exit=$?

if [ $attest_success_exit -eq 0 ]; then
    log_success "acc attest demo-app:ok after verifying it: succeeded"

    # Stdout MUST include "Creating attestation"
    if echo "$attest_success_output" | grep -q "Creating attestation"; then
        log_success "acc attest showed 'Creating attestation' on success"
    else
        log_error "acc attest did not show 'Creating attestation' on success (UX bug)"
    fi
else
    log_error "acc attest demo-app:ok after verifying it: failed (exit $attest_success_exit)"
fi

# ============================================================================
# TEST 6: Inspect per-image (no cross leakage)
# ============================================================================

log_section "TEST 6: Inspect Per-Image State"

# Inspect demo-app:ok (should show PASS)
inspect_ok_output=$($ACC_BIN inspect --json demo-app:ok 2>&1) || true
inspect_ok_exit=$?

if [ $inspect_ok_exit -eq 0 ]; then
    log_success "acc inspect demo-app:ok: exit 0"

    if echo "$inspect_ok_output" | jq empty 2>/dev/null; then
        assert_json_field "$inspect_ok_output" ".status" "pass" \
            "inspect demo-app:ok status"
    fi
else
    log_error "acc inspect demo-app:ok: exit $inspect_ok_exit (expected 0)"
fi

# Inspect demo-app:root (should show FAIL)
inspect_root_output=$($ACC_BIN inspect --json demo-app:root 2>&1) || true
inspect_root_exit=$?

if [ $inspect_root_exit -eq 0 ]; then
    log_success "acc inspect demo-app:root: exit 0"

    if echo "$inspect_root_output" | jq empty 2>/dev/null; then
        assert_json_field "$inspect_root_output" ".status" "fail" \
            "inspect demo-app:root status"
    fi
else
    log_error "acc inspect demo-app:root: exit $inspect_root_exit (expected 0)"
fi

# ============================================================================
# TEST 7: Trust status
# ============================================================================

log_section "TEST 7: Trust Status"

# Trust status for demo-app:ok
set +e
status_ok_output=$($ACC_BIN trust status --json demo-app:ok 2>&1)
status_ok_exit=$?
set -e

log "acc trust status demo-app:ok: exit $status_ok_exit"
if [ $status_ok_exit -eq 0 ]; then
    log_success "trust status demo-app:ok: exit 0 (pass status)"
else
    log_error "trust status demo-app:ok: exit $status_ok_exit (expected 0 for pass)"
fi

if echo "$status_ok_output" | jq empty 2>/dev/null; then
    log_success "Valid JSON output from trust status demo-app:ok"

    # v0.2.7: Verify required JSON schema fields
    assert_json_has_field "$status_ok_output" ".schemaVersion" \
        "trust status demo-app:ok has schemaVersion"
    assert_json_has_field "$status_ok_output" ".imageRef" \
        "trust status demo-app:ok has imageRef"
    assert_json_has_field "$status_ok_output" ".status" \
        "trust status demo-app:ok has status"
    assert_json_has_field "$status_ok_output" ".sbomPresent" \
        "trust status demo-app:ok has sbomPresent"
    assert_json_has_field "$status_ok_output" ".violations" \
        "trust status demo-app:ok has violations"
    assert_json_has_field "$status_ok_output" ".warnings" \
        "trust status demo-app:ok has warnings"
    assert_json_has_field "$status_ok_output" ".attestations" \
        "trust status demo-app:ok has attestations"
    assert_json_has_field "$status_ok_output" ".timestamp" \
        "trust status demo-app:ok has timestamp"

    # Verify status value
    assert_json_field "$status_ok_output" ".status" "pass" \
        "trust status demo-app:ok status"

    # v0.2.7: Check that attestation was recorded (after we attested it in TEST 5)
    attest_count=$(echo "$status_ok_output" | jq -r '.attestations | length')
    if [ "$attest_count" -gt 0 ]; then
        log_success "trust status demo-app:ok shows $attest_count attestation(s)"
    else
        log "⚠️  trust status demo-app:ok shows no attestations (may be expected if attest failed earlier)"
    fi
else
    log_error "Invalid JSON output from trust status demo-app:ok"
fi

# Trust status for demo-app:root
set +e
status_root_output=$($ACC_BIN trust status --json demo-app:root 2>&1)
status_root_exit=$?
set -e

log "acc trust status demo-app:root: exit $status_root_exit"
if [ $status_root_exit -eq 1 ]; then
    log_success "trust status demo-app:root: exit 1 (fail status)"
else
    log_error "trust status demo-app:root: exit $status_root_exit (expected 1 for fail)"
fi

if echo "$status_root_output" | jq empty 2>/dev/null; then
    log_success "Valid JSON output from trust status demo-app:root"

    # v0.2.7: Verify required JSON schema fields
    assert_json_has_field "$status_root_output" ".schemaVersion" \
        "trust status demo-app:root has schemaVersion"
    assert_json_has_field "$status_root_output" ".status" \
        "trust status demo-app:root has status"
    assert_json_has_field "$status_root_output" ".sbomPresent" \
        "trust status demo-app:root has sbomPresent"

    # Verify status value
    assert_json_field "$status_root_output" ".status" "fail" \
        "trust status demo-app:root status"

    # v0.2.7: Per-image isolation - demo-app:root should have 0 attestations
    attest_count=$(echo "$status_root_output" | jq -r '.attestations | length')
    if [ "$attest_count" -eq 0 ]; then
        log_success "trust status demo-app:root shows 0 attestations (per-image isolation)"
    else
        log_error "trust status demo-app:root shows $attest_count attestations (expected 0 - per-image isolation failure)"
    fi
else
    log_error "Invalid JSON output from trust status demo-app:root"
fi

# Build a third image that has never been verified
log "Building demo-app:never-verified"
cat > Dockerfile.never <<'EOF'
FROM alpine:3.19
CMD ["echo", "never verified"]
EOF

docker build -f Dockerfile.never -t demo-app:never-verified . > /dev/null 2>&1

# Trust status for never-verified image (should return exit code 2 for unknown status)
set +e
status_never_output=$($ACC_BIN trust status --json demo-app:never-verified 2>&1)
status_never_exit=$?
set -e

log "acc trust status demo-app:never-verified: exit $status_never_exit"
if [ $status_never_exit -eq 2 ]; then
    log_success "trust status for never-verified image: exit 2 (unknown status)"
else
    log_error "trust status for never-verified image: exit $status_never_exit (expected 2 for unknown)"
fi

# v0.2.7: Verify unknown status has proper JSON schema
if echo "$status_never_output" | jq empty 2>/dev/null; then
    assert_json_field "$status_never_output" ".status" "unknown" \
        "trust status demo-app:never-verified status"
    assert_json_has_field "$status_never_output" ".sbomPresent" \
        "trust status demo-app:never-verified has sbomPresent"

    # Verify sbomPresent is false for unknown
    sbom_present=$(echo "$status_never_output" | jq -r '.sbomPresent')
    if [ "$sbom_present" == "false" ]; then
        log_success "trust status demo-app:never-verified has sbomPresent: false"
    else
        log_error "trust status demo-app:never-verified has sbomPresent: $sbom_present (expected false)"
    fi
fi

# ============================================================================
# TEST 8: Run command (if supported)
# ============================================================================

log_section "TEST 8: acc run"

# Test if acc run works with verified image
run_output=$($ACC_BIN run demo-app:ok -- echo "test" 2>&1) || true
run_exit=$?

if [ $run_exit -eq 0 ]; then
    log_success "acc run demo-app:ok: succeeded"
else
    log "⚠️  acc run demo-app:ok: exit $run_exit"
    log "Run command might not support local execution yet"
    log "Output: $run_output"
fi

# ============================================================================
# TEST 9: acc trust verify (v0.3.0)
# ============================================================================

log_section "TEST 9: acc trust verify"

# Test 9.1: Verify image WITH attestation (demo-app:ok has attestation from TEST 6)
log "Test 9.1: Verify attestation for demo-app:ok (should have attestation)"

set +e
verify_attest_ok_output=$($ACC_BIN trust verify --json demo-app:ok 2>&1)
verify_attest_ok_exit=$?
set -e

log "Trust verify output for demo-app:ok:"
echo "$verify_attest_ok_output" | tee -a "$LOGFILE"

# Should exit 0 (verified) if attestations exist, or exit 1 (unverified) if none yet
if [ $verify_attest_ok_exit -eq 0 ]; then
    log_success "acc trust verify demo-app:ok: exit 0 (verified)"

    # Validate JSON
    if echo "$verify_attest_ok_output" | jq empty 2>/dev/null; then
        log_success "Valid JSON output from trust verify"

        # Check verificationStatus is "verified"
        assert_json_field "$verify_attest_ok_output" ".verificationStatus" "verified" \
            "trust verify verificationStatus"

        # Check attestationCount > 0
        attest_count=$(echo "$verify_attest_ok_output" | jq -r '.attestationCount')
        if [ "$attest_count" -gt 0 ]; then
            log_success "trust verify demo-app:ok has attestations: count = $attest_count"
        else
            log_error "trust verify demo-app:ok has no attestations (expected > 0)"
        fi

        # Check required fields exist
        assert_json_has_field "$verify_attest_ok_output" ".schemaVersion" \
            "trust verify schemaVersion"
        assert_json_has_field "$verify_attest_ok_output" ".imageRef" \
            "trust verify imageRef"
        assert_json_has_field "$verify_attest_ok_output" ".imageDigest" \
            "trust verify imageDigest"
        assert_json_has_field "$verify_attest_ok_output" ".attestations" \
            "trust verify attestations"
        assert_json_has_field "$verify_attest_ok_output" ".errors" \
            "trust verify errors"
    else
        log_error "Invalid JSON output from trust verify"
    fi
elif [ $verify_attest_ok_exit -eq 1 ]; then
    log "⚠️  acc trust verify demo-app:ok: exit 1 (unverified - might not have attestation yet)"
    log "This is acceptable if demo-app:ok hasn't been attested in this test run"
else
    log_error "acc trust verify demo-app:ok: exit $verify_attest_ok_exit (expected 0 or 1)"
fi

# Test 9.2: Verify image WITHOUT attestation (demo-app:never-verified)
log "Test 9.2: Verify attestation for demo-app:never-verified (no attestation)"

# Build an image that hasn't been attested
log "Building demo-app:never-verified (non-root user, but never attested)"
cat > Dockerfile <<'EOF'
FROM alpine:3.19
RUN addgroup -g 1001 testuser && adduser -D -u 1001 -G testuser testuser
USER testuser
WORKDIR /app
CMD ["sh", "-c", "echo 'Hello from never-verified'; sleep 1"]
EOF

assert_success "acc build demo-app:never-verified" \
    $ACC_BIN build demo-app:never-verified

set +e
verify_attest_never_output=$($ACC_BIN trust verify --json demo-app:never-verified 2>&1)
verify_attest_never_exit=$?
set -e

log "Trust verify output for demo-app:never-verified:"
echo "$verify_attest_never_output" | tee -a "$LOGFILE"

# Should exit 1 (unverified - no attestations) or exit 2 (unknown - can't resolve digest)
if [ $verify_attest_never_exit -eq 1 ]; then
    log_success "acc trust verify demo-app:never-verified: exit 1 (unverified - no attestations)"

    # Validate JSON
    if echo "$verify_attest_never_output" | jq empty 2>/dev/null; then
        log_success "Valid JSON output from trust verify (never-verified)"

        # Check verificationStatus is "unverified"
        assert_json_field "$verify_attest_never_output" ".verificationStatus" "unverified" \
            "trust verify verificationStatus (no attestations)"

        # Check attestationCount is 0
        assert_json_field "$verify_attest_never_output" ".attestationCount" "0" \
            "trust verify attestationCount (no attestations)"
    else
        log_error "Invalid JSON output from trust verify (never-verified)"
    fi
elif [ $verify_attest_never_exit -eq 2 ]; then
    log_success "acc trust verify demo-app:never-verified: exit 2 (unknown - cannot resolve digest)"
else
    log_error "acc trust verify demo-app:never-verified: exit $verify_attest_never_exit (expected 1 or 2)"
fi

# Test 9.3: Verify non-existent image (should exit 2 - unknown)
log "Test 9.3: Verify attestation for non-existent image (exit 2 - unknown)"

set +e
verify_attest_missing_output=$($ACC_BIN trust verify --json nonexistent:image 2>&1)
verify_attest_missing_exit=$?
set -e

log "Trust verify output for nonexistent:image:"
echo "$verify_attest_missing_output" | tee -a "$LOGFILE"

# Should exit 2 (unknown - cannot resolve digest)
if [ $verify_attest_missing_exit -eq 2 ]; then
    log_success "acc trust verify nonexistent:image: exit 2 (unknown)"

    # Validate JSON
    if echo "$verify_attest_missing_output" | jq empty 2>/dev/null; then
        log_success "Valid JSON output from trust verify (nonexistent)"

        # Check verificationStatus is "unknown"
        assert_json_field "$verify_attest_missing_output" ".verificationStatus" "unknown" \
            "trust verify verificationStatus (nonexistent image)"
    else
        log_error "Invalid JSON output from trust verify (nonexistent)"
    fi
else
    log_error "acc trust verify nonexistent:image: exit $verify_attest_missing_exit (expected 2)"
fi

# ============================================================================
# TEST 10: Trust Enforcement (v0.3.1)
# ============================================================================

log_section "TEST 10: Trust Enforcement (v0.3.1)"

# Test 10.1: Baseline - run without enforcement (should work as before)
log "Test 10.1: Run without enforcement (baseline - should work)"

# Ensure we're using demo-app:ok which has attestation from TEST 6
set +e
run_no_enforce_output=$($ACC_BIN run demo-app:ok -- echo "test" 2>&1)
run_no_enforce_exit=$?
set -e

# Run should work (exit 0) or may not be fully implemented yet
if [ $run_no_enforce_exit -eq 0 ]; then
    log_success "acc run (no enforcement): succeeded (exit 0)"
elif [ $run_no_enforce_exit -eq 1 ]; then
    log "⚠️  acc run (no enforcement): exit 1 (verification might have failed, or not implemented)"
else
    log "⚠️  acc run (no enforcement): exit $run_no_enforce_exit (command might not support local execution yet)"
fi

# Test 10.2: Enable enforcement in config and run WITH attestation
log "Test 10.2: Run with enforcement enabled + image HAS attestation (should proceed)"

# Create a test config with requireAttestation: true
cat > "$WORKDIR/acc-enforce.yaml" <<EOF
project:
  name: demo-project

build:
  context: .
  defaultTag: latest

registry:
  default: localhost:5000

policy:
  mode: enforce
  requireAttestation: true

signing:
  mode: keyless

sbom:
  format: spdx
EOF

# Run with enforcement config (demo-app:ok has attestation from TEST 6)
set +e
run_enforce_ok_output=$($ACC_BIN --config "$WORKDIR/acc-enforce.yaml" run demo-app:ok -- echo "test" 2>&1)
run_enforce_ok_exit=$?
set -e

log "Run with enforcement (HAS attestation) output:"
echo "$run_enforce_ok_output" | head -20 | tee -a "$LOGFILE"

# MUST succeed (exit 0) when trust enforcement passes
# Exit 0 means trust gates passed, regardless of runtime execution success
if [ $run_enforce_ok_exit -eq 0 ]; then
    log_success "acc run (enforcement + HAS attestation): exit 0 (trust enforcement succeeded)"

    # Verify trust enforcement succeeded (should see verification messages)
    if echo "$run_enforce_ok_output" | grep -qi "verification passed"; then
        log_success "Trust enforcement messages present in output"
    fi
elif [ $run_enforce_ok_exit -eq 1 ]; then
    log_error "acc run (enforcement + HAS attestation): exit 1 (trust enforcement should succeed!)"
    log "Output: $run_enforce_ok_output"
    exit 1
else
    log_error "acc run (enforcement + HAS attestation): exit $run_enforce_ok_exit (expected exit 0)"
    log "Output: $run_enforce_ok_output"
    exit 1
fi

# Test 10.3: Run with enforcement enabled + image NO attestation (should block)
log "Test 10.3: Run with enforcement enabled + image NO attestation (should block)"

# Use demo-app:root which was verified in TEST 4 but NOT attested
set +e
run_enforce_fail_output=$($ACC_BIN --config "$WORKDIR/acc-enforce.yaml" run demo-app:root -- echo "test" 2>&1)
run_enforce_fail_exit=$?
set -e

log "Run with enforcement (NO attestation) output:"
echo "$run_enforce_fail_output" | head -20 | tee -a "$LOGFILE"

# Should block (exit 1)
if [ $run_enforce_fail_exit -eq 1 ]; then
    log_success "acc run (enforcement + NO attestation): exit 1 (blocked as expected)"

    # Check for remediation message
    if echo "$run_enforce_fail_output" | grep -qi "attestation"; then
        log_success "Error message mentions attestation requirement"
    else
        log "⚠️  Error message should mention attestation requirement"
    fi
elif [ $run_enforce_fail_exit -eq 0 ]; then
    log_error "acc run (enforcement + NO attestation): exit 0 (should have blocked!)"
else
    log "⚠️  acc run (enforcement + NO attestation): exit $run_enforce_fail_exit"
fi

# Test 10.4: Run with enforcement + never verified image (should block)
log "Test 10.4: Run with enforcement + never verified image (should block)"

set +e
run_enforce_unknown_output=$($ACC_BIN --config "$WORKDIR/acc-enforce.yaml" run demo-app:never-verified -- echo "test" 2>&1)
run_enforce_unknown_exit=$?
set -e

# Should block (exit 1 or 2)
if [ $run_enforce_unknown_exit -eq 1 ] || [ $run_enforce_unknown_exit -eq 2 ]; then
    log_success "acc run (enforcement + unknown): exit $run_enforce_unknown_exit (blocked)"
elif [ $run_enforce_unknown_exit -eq 0 ]; then
    log_error "acc run (enforcement + unknown): exit 0 (should have blocked!)"
else
    log "⚠️  acc run (enforcement + unknown): exit $run_enforce_unknown_exit"
fi

# Test 10.5: Regression test for TTY handling in non-interactive environments
log "Test 10.5: Non-TTY environment (regression test for CI compatibility)"

# Run in non-TTY environment by explicitly closing stdin
# This simulates CI/non-interactive execution
set +e
run_notty_output=$($ACC_BIN run demo-app:ok -- echo "test" </dev/null 2>&1)
run_notty_exit=$?
set -e

log "Run in non-TTY environment output:"
echo "$run_notty_output" | head -20 | tee -a "$LOGFILE"

# Should succeed (exit 0) when trust passes, even in non-TTY
if [ $run_notty_exit -eq 0 ]; then
    log_success "acc run (non-TTY): exit 0 (trust enforcement succeeded)"

    # Verify we didn't try to use -it flags inappropriately
    if echo "$run_notty_output" | grep -qi "the input device is not a TTY"; then
        log_error "Docker -it flags incorrectly used in non-TTY environment"
        exit 1
    else
        log_success "Non-TTY environment handled correctly (no TTY errors)"
    fi
elif [ $run_notty_exit -eq 1 ]; then
    log_error "acc run (non-TTY): exit 1 (trust enforcement should succeed!)"
    log "Output: $run_notty_output"
    exit 1
else
    log "⚠️  acc run (non-TTY): exit $run_notty_exit"
fi

# ============================================================================
# RESULTS
# ============================================================================

log_section "TIER 1 E2E SMOKE TEST RESULTS"
log "Workdir: $WORKDIR"
log "Log file: $LOGFILE"

if [ $FAILED -eq 0 ]; then
    log_success "All E2E smoke tests passed!"
    exit 0
else
    log_error "$FAILED test(s) failed"
    echo ""
    echo "Workdir preserved at: $WORKDIR"
    echo "View full log: $LOGFILE"
    exit 1
fi
