#!/usr/bin/env bash
# Tier 0: CLI Help Matrix Tests
# Fast command registration and help text validation
# This script verifies that all commands exist and display help correctly

set -euo pipefail

# ============================================================================
# CONFIGURATION
# ============================================================================

LOGFILE="/tmp/tier0-cli-help-$(date +%s).log"
ACC_BIN="${ACC_BIN:-./acc}"
FAILED=0

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

# ============================================================================
# TEST FUNCTIONS
# ============================================================================

# Test that a command shows help and exits with code 0
test_help_command() {
    local cmd_args="$1"
    local description="$2"

    log "Testing: $ACC_BIN $cmd_args"

    if output=$($ACC_BIN $cmd_args 2>&1); then
        exit_code=$?
        if [ $exit_code -eq 0 ]; then
            if [ -n "$output" ]; then
                log_success "$description: exit 0, non-empty output"
                return 0
            else
                log_error "$description: exit 0 but empty output"
                return 1
            fi
        else
            log_error "$description: exit code $exit_code (expected 0)"
            echo "$output" >> "$LOGFILE"
            return 1
        fi
    else
        exit_code=$?
        log_error "$description: exit code $exit_code (expected 0)"
        return 1
    fi
}

# Test that a not-implemented command returns stable error
test_not_implemented() {
    local cmd_args="$1"
    local description="$2"

    log "Testing not-implemented: $ACC_BIN $cmd_args"

    if output=$($ACC_BIN $cmd_args 2>&1); then
        exit_code=$?
        # If it returns 0, it might be implemented
        log "⚠️  $description: returned exit 0 (might be implemented, or help text)"
        return 0
    else
        exit_code=$?
        if echo "$output" | grep -qiE "(not implemented|coming soon)"; then
            log_success "$description: clear not-implemented message, exit $exit_code"
            return 0
        else
            log "⚠️  $description: exit $exit_code, checking if it's help text..."
            if echo "$output" | grep -qE "(Usage:|Flags:|Commands:)"; then
                log_success "$description: shows help text"
                return 0
            else
                log_error "$description: unclear error message"
                echo "$output" >> "$LOGFILE"
                return 1
            fi
        fi
    fi
}

# ============================================================================
# MAIN TEST EXECUTION
# ============================================================================

log_section "TIER 0: CLI HELP MATRIX TESTS"
log "ACC Binary: $ACC_BIN"
log "Log File: $LOGFILE"

# Verify acc binary exists
if [ ! -f "$ACC_BIN" ]; then
    log_error "acc binary not found at $ACC_BIN"
    exit 1
fi

log_section "Testing Root Command"
test_help_command "--help" "acc --help"

log_section "Testing Core Commands"

# Commands that should be fully implemented
CORE_COMMANDS=(
    "init --help:Initialize project"
    "build --help:Build image with SBOM"
    "verify --help:Verify SBOM and policy"
    "run --help:Run verified workload"
    "push --help:Push verified image"
    "promote --help:Promote image to environment"
    "attest --help:Create attestation"
    "inspect --help:Inspect trust summary"
    "version --help:Show version"
    "upgrade --help:Upgrade acc"
)

for cmd_spec in "${CORE_COMMANDS[@]}"; do
    cmd="${cmd_spec%%:*}"
    desc="${cmd_spec##*:}"
    test_help_command "$cmd" "$desc"
done

log_section "Testing Subcommands"

# trust subcommand
test_help_command "trust --help" "trust command help"
test_help_command "trust status --help" "trust status subcommand"

# policy subcommand
test_help_command "policy --help" "policy command help"
test_help_command "policy explain --help" "policy explain subcommand"

log_section "Testing Possibly Not-Implemented Commands"

# config and login might not be fully implemented yet
test_not_implemented "config --help" "config command"
test_not_implemented "login --help" "login command"

# ============================================================================
# RESULTS
# ============================================================================

log_section "TIER 0 RESULTS"
log "Log file: $LOGFILE"

if [ $FAILED -eq 0 ]; then
    log_success "All CLI help tests passed!"
    exit 0
else
    log_error "$FAILED test(s) failed"
    echo ""
    echo "View full log: $LOGFILE"
    exit 1
fi
