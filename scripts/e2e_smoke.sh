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
assert_success() {
    local description="$1"
    shift
    log_command "$@"

    if output=$("$@" 2>&1); then
        exit_code=$?
        if [ $exit_code -eq 0 ]; then
            log_success "$description"
            echo "$output" >> "$LOGFILE"
            return 0
        fi
    fi

    exit_code=$?
    log_error "$description (exit code: $exit_code)"
    echo "$output" >> "$LOGFILE"
    return 1
}

# Assert that a command fails with specific exit code
assert_failure() {
    local expected_exit=$1
    local description="$2"
    shift 2
    log_command "$@"

    if output=$("$@" 2>&1); then
        exit_code=0
    else
        exit_code=$?
    fi

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

    if echo "$json" | jq -e "$jq_expr" > /dev/null 2>&1; then
        log_success "$description: field $jq_expr exists"
        return 0
    else
        log_error "$description: field $jq_expr missing"
        return 1
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
    log "✓ $tool: $(command -v $tool)"
done

# Verify acc binary exists
if [ ! -f "$ACC_BIN" ]; then
    log_error "acc binary not found at $ACC_BIN"
    exit 1
fi

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
# TEST 2: Build test images with docker
# ============================================================================

log_section "TEST 2: Build Docker Test Images"

# Create Dockerfile for non-root user (should PASS policy)
log "Creating Dockerfile for demo-app:ok (non-root user)"
cat > Dockerfile.ok <<'EOF'
FROM alpine:3.19
RUN addgroup -g 1000 appuser && adduser -D -u 1000 -G appuser appuser
USER appuser
WORKDIR /app
CMD ["sh", "-c", "echo 'Hello from non-root user'; sleep 1"]
EOF

assert_success "Build demo-app:ok" \
    docker build -f Dockerfile.ok -t demo-app:ok .

# Create Dockerfile for root user (should FAIL policy)
log "Creating Dockerfile for demo-app:root (root user)"
cat > Dockerfile.root <<'EOF'
FROM alpine:3.19
# Intentionally no USER directive - runs as root
WORKDIR /app
CMD ["sh", "-c", "echo 'Hello from root user'; sleep 1"]
EOF

assert_success "Build demo-app:root" \
    docker build -f Dockerfile.root -t demo-app:root .

# ============================================================================
# TEST 3: acc build (if supported)
# ============================================================================

log_section "TEST 3: acc build"

# Test acc build with positional argument (v0.2.3 fix)
log "Testing acc build with positional argument"
if assert_success "acc build demo-app:ok (positional arg)" \
    $ACC_BIN build demo-app:ok; then

    # Verify SBOM was created
    if [ -f ".acc/sbom/test-project.spdx.json" ]; then
        log_success "SBOM created by acc build"
    else
        log_error "SBOM not created by acc build (v0.2.3 bug)"
    fi
fi

# Also test with --tag flag
log "Testing acc build with --tag flag"
if assert_success "acc build demo-app:root (--tag flag)" \
    $ACC_BIN build --tag demo-app:root; then

    if [ -f ".acc/sbom/test-project.spdx.json" ]; then
        log_success "SBOM created by acc build --tag"
    fi
fi

# ============================================================================
# TEST 4: acc verify demo-app:ok (should PASS)
# ============================================================================

log_section "TEST 4: acc verify demo-app:ok (expect PASS)"

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
# TEST 5: acc verify demo-app:root (should FAIL)
# ============================================================================

log_section "TEST 5: acc verify demo-app:root (expect FAIL)"

verify_root_output=$($ACC_BIN verify --json demo-app:root 2>&1) || true
verify_root_exit=$?

log "Verify output:"
echo "$verify_root_output" | tee -a "$LOGFILE"

if [ $verify_root_exit -eq 1 ]; then
    log_success "acc verify demo-app:root: exit 1"
else
    log_error "acc verify demo-app:root: exit $verify_root_exit (expected 1)"
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
# TEST 6: acc policy explain
# ============================================================================

log_section "TEST 6: acc policy explain"

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
# TEST 7: Attest UX checks
# ============================================================================

log_section "TEST 7: Attest UX Checks"

# After verifying root last, acc attest demo-app:ok should FAIL (mismatch)
log "Attempting to attest demo-app:ok after verifying demo-app:root (should fail)"
attest_mismatch_output=$($ACC_BIN attest demo-app:ok 2>&1) || true
attest_mismatch_exit=$?

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
# TEST 8: Inspect per-image (no cross leakage)
# ============================================================================

log_section "TEST 8: Inspect Per-Image State"

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
# TEST 9: Trust status
# ============================================================================

log_section "TEST 9: Trust Status"

# Trust status for demo-app:ok
status_ok_output=$($ACC_BIN trust status --json demo-app:ok 2>&1) || true
status_ok_exit=$?

log "acc trust status demo-app:ok: exit $status_ok_exit"
if echo "$status_ok_output" | jq empty 2>/dev/null; then
    log_success "Valid JSON output from trust status demo-app:ok"

    assert_json_field "$status_ok_output" ".status" "pass" \
        "trust status demo-app:ok"
fi

# Trust status for demo-app:root
status_root_output=$($ACC_BIN trust status --json demo-app:root 2>&1) || true
status_root_exit=$?

log "acc trust status demo-app:root: exit $status_root_exit"
if echo "$status_root_output" | jq empty 2>/dev/null; then
    log_success "Valid JSON output from trust status demo-app:root"

    assert_json_field "$status_root_output" ".status" "fail" \
        "trust status demo-app:root"
fi

# Build a third image that has never been verified
log "Building demo-app:never-verified"
cat > Dockerfile.never <<'EOF'
FROM alpine:3.19
CMD ["echo", "never verified"]
EOF

docker build -f Dockerfile.never -t demo-app:never-verified . > /dev/null 2>&1

# Trust status for never-verified image (should return exit code 2 or specific "unknown" status)
status_never_output=$($ACC_BIN trust status --json demo-app:never-verified 2>&1) || true
status_never_exit=$?

log "acc trust status demo-app:never-verified: exit $status_never_exit"
if [ $status_never_exit -eq 2 ]; then
    log_success "trust status for never-verified image: exit 2 as expected"
elif [ $status_never_exit -eq 0 ]; then
    # Might return exit 0 with status:"unknown"
    if echo "$status_never_output" | jq -e '.status == "unknown"' > /dev/null 2>&1; then
        log_success "trust status for never-verified image: exit 0 with status:unknown"
    else
        log_error "trust status for never-verified image: exit 0 but status not unknown"
    fi
else
    log "⚠️  trust status for never-verified image: exit $status_never_exit (documenting behavior)"
fi

# ============================================================================
# TEST 10: Run command (if supported)
# ============================================================================

log_section "TEST 10: acc run"

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
