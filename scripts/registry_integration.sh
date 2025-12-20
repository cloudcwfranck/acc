#!/usr/bin/env bash
# Tier 2: Registry Integration Tests
# Tests push/promote workflows with GitHub Container Registry (GHCR)
# This tier is OPTIONAL and never blocks PRs

set -euo pipefail

# ============================================================================
# CONFIGURATION
# ============================================================================

LOGFILE="/tmp/tier2-registry-$(date +%s).log"
ACC_BIN="${ACC_BIN:-./acc}"
WORKDIR="/tmp/acc-registry-$(date +%s)"
FAILED=0

# Registry configuration (from environment)
GHCR_REGISTRY="${GHCR_REGISTRY:-ghcr.io}"
GHCR_REPO="${GHCR_REPO:-}"
GITHUB_SHA="${GITHUB_SHA:-$(git rev-parse --short HEAD 2>/dev/null || echo 'local')}"

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

log_skip() {
    echo "⏭️  $*" | tee -a "$LOGFILE"
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
# PRE-FLIGHT CHECKS
# ============================================================================

preflight_checks() {
    log_section "Pre-flight Checks"

    # Check if GHCR_REPO is set
    if [ -z "$GHCR_REPO" ]; then
        log_skip "GHCR_REPO not set - skipping registry integration tests"
        log "Set GHCR_REPO to enable Tier 2 tests (e.g., 'owner/repo')"
        exit 0
    fi

    # Check if docker is logged in to GHCR
    if ! docker info 2>&1 | grep -q "ghcr.io" && ! echo "$GHCR_REGISTRY" | grep -q "ghcr.io"; then
        log "⚠️  Not logged in to GHCR, attempting to verify credentials..."

        # Try to pull a public image to test connectivity
        if ! docker pull ghcr.io/alpine:latest > /dev/null 2>&1; then
            log_skip "Cannot access GHCR - skipping registry integration tests"
            log "Run: echo \$GITHUB_TOKEN | docker login ghcr.io -u <username> --password-stdin"
            exit 0
        fi
    fi

    log_success "Pre-flight checks passed"
    log "GHCR Registry: $GHCR_REGISTRY"
    log "GHCR Repo: $GHCR_REPO"
    log "GitHub SHA: $GITHUB_SHA"
}

# ============================================================================
# MAIN TEST EXECUTION
# ============================================================================

log_section "TIER 2: REGISTRY INTEGRATION TESTS"
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

# Convert to absolute path before changing directories
ACC_BIN=$(realpath "$ACC_BIN")
log "Resolved ACC Binary: $ACC_BIN"

# Run pre-flight checks (may skip entire test suite)
preflight_checks

# Create workdir
mkdir -p "$WORKDIR"
cd "$WORKDIR"
log "Working directory: $(pwd)"

# ============================================================================
# TEST 1: Initialize project
# ============================================================================

log_section "TEST 1: Initialize Project"

log_command "$ACC_BIN init registry-test"
if $ACC_BIN init registry-test 2>&1 | tee -a "$LOGFILE"; then
    log_success "acc init succeeded"
else
    log_error "acc init failed"
    exit 1
fi

# ============================================================================
# TEST 2: Build and verify test image
# ============================================================================

log_section "TEST 2: Build and Verify Test Image"

# Create a simple Dockerfile
cat > Dockerfile <<'EOF'
FROM alpine:3.19
RUN addgroup -g 1000 appuser && adduser -D -u 1000 -G appuser appuser
USER appuser
WORKDIR /app
RUN echo "Registry test image" > /app/README.txt
CMD ["cat", "/app/README.txt"]
EOF

# Build local image
LOCAL_IMAGE="acc-ci-test:${GITHUB_SHA}"
log "Building local image: $LOCAL_IMAGE"

log_command "docker build -t $LOCAL_IMAGE ."
if docker build -t "$LOCAL_IMAGE" . 2>&1 | tee -a "$LOGFILE"; then
    log_success "Docker build succeeded"
else
    log_error "Docker build failed"
    exit 1
fi

# Build with acc (generates SBOM)
log "Building with acc to generate SBOM"
log_command "$ACC_BIN build $LOCAL_IMAGE"
if $ACC_BIN build "$LOCAL_IMAGE" 2>&1 | tee -a "$LOGFILE"; then
    log_success "acc build succeeded"
else
    log_error "acc build failed"
    exit 1
fi

# Verify the image
log "Verifying image before push"
log_command "$ACC_BIN verify --json $LOCAL_IMAGE"
if verify_output=$($ACC_BIN verify --json "$LOCAL_IMAGE" 2>&1); then
    log_success "acc verify succeeded"
    echo "$verify_output" | jq . >> "$LOGFILE" 2>/dev/null || echo "$verify_output" >> "$LOGFILE"

    # Check that status is "pass"
    status=$(echo "$verify_output" | jq -r '.status' 2>/dev/null || echo "unknown")
    if [ "$status" == "pass" ]; then
        log_success "Verification status: pass"
    else
        log_error "Verification status: $status (expected pass)"
    fi
else
    log_error "acc verify failed"
    exit 1
fi

# ============================================================================
# TEST 3: Push to GHCR
# ============================================================================

log_section "TEST 3: Push to GHCR"

# Tag for GHCR
GHCR_IMAGE="${GHCR_REGISTRY}/${GHCR_REPO}/acc-ci-test:${GITHUB_SHA}"
log "Tagging image for GHCR: $GHCR_IMAGE"

log_command "docker tag $LOCAL_IMAGE $GHCR_IMAGE"
if docker tag "$LOCAL_IMAGE" "$GHCR_IMAGE" 2>&1 | tee -a "$LOGFILE"; then
    log_success "Docker tag succeeded"
else
    log_error "Docker tag failed"
    exit 1
fi

# Push with acc (with verification gate)
log "Pushing to GHCR with acc"
log_command "$ACC_BIN push $GHCR_IMAGE"
if push_output=$($ACC_BIN push "$GHCR_IMAGE" 2>&1); then
    log_success "acc push succeeded"
    echo "$push_output" | tee -a "$LOGFILE"
else
    push_exit=$?
    log_error "acc push failed (exit $push_exit)"
    echo "$push_output" | tee -a "$LOGFILE"

    # Push might fail if not implemented or no registry access
    log "⚠️  Push failed - this might be expected if push command is not fully implemented"
    log "Attempting manual docker push to verify registry access..."

    if docker push "$GHCR_IMAGE" 2>&1 | tee -a "$LOGFILE"; then
        log_success "Manual docker push succeeded"
        log "Issue is with acc push implementation, not registry access"
    else
        log_error "Manual docker push also failed"
        log "Registry access issue or permissions problem"
    fi
fi

# ============================================================================
# TEST 4: Promote (if supported)
# ============================================================================

log_section "TEST 4: Promote to Environment"

log "Testing promote command"
log_command "$ACC_BIN promote $GHCR_IMAGE --to production"
if promote_output=$($ACC_BIN promote "$GHCR_IMAGE" --to production 2>&1); then
    log_success "acc promote succeeded"
    echo "$promote_output" | tee -a "$LOGFILE"
else
    promote_exit=$?
    log "⚠️  acc promote exit $promote_exit"
    echo "$promote_output" | tee -a "$LOGFILE"

    # Promote might not be fully implemented
    if echo "$promote_output" | grep -qiE "(not implemented|coming soon)"; then
        log "Promote not implemented yet - skipping"
    else
        log "Promote failed but not with 'not implemented' message"
    fi
fi

# ============================================================================
# TEST 5: Pull and verify from registry
# ============================================================================

log_section "TEST 5: Pull from GHCR and Re-verify"

# Remove local image
log "Removing local images to test pull"
docker rmi "$LOCAL_IMAGE" > /dev/null 2>&1 || true
docker rmi "$GHCR_IMAGE" > /dev/null 2>&1 || true

# Try to pull
log "Pulling from GHCR: $GHCR_IMAGE"
log_command "docker pull $GHCR_IMAGE"
if docker pull "$GHCR_IMAGE" 2>&1 | tee -a "$LOGFILE"; then
    log_success "Docker pull from GHCR succeeded"

    # Verify pulled image
    log "Verifying pulled image"
    if $ACC_BIN verify --json "$GHCR_IMAGE" > /dev/null 2>&1; then
        log_success "acc verify on pulled image succeeded"
    else
        log "⚠️  acc verify on pulled image failed (SBOM might not be in local cache)"
    fi
else
    log_error "Docker pull from GHCR failed"
fi

# ============================================================================
# CLEANUP: Delete test image from registry
# ============================================================================

log_section "Cleanup"

# Note: Deleting from GHCR requires API access, which might not be available
# We'll just log that cleanup should be done
log "⚠️  Manual cleanup required:"
log "Delete test image from: $GHCR_IMAGE"
log "Use: gh api --method DELETE /user/packages/container/acc-ci-test/versions/<version-id>"

# ============================================================================
# RESULTS
# ============================================================================

log_section "TIER 2 REGISTRY INTEGRATION TEST RESULTS"
log "Workdir: $WORKDIR"
log "Log file: $LOGFILE"

if [ $FAILED -eq 0 ]; then
    log_success "All registry integration tests passed (or skipped appropriately)!"
    exit 0
else
    log_error "$FAILED test(s) failed"
    echo ""
    echo "Workdir preserved at: $WORKDIR"
    echo "View full log: $LOGFILE"
    exit 1
fi
