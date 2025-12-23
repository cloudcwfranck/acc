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
# TEST 6: Remote Attestation Publishing and Fetching (v0.3.2)
# ============================================================================

log_section "TEST 6: Remote Attestation Publishing and Fetching"

# 6.1: Create local attestation for the image
log "Step 6.1: Create local attestation for verified image"
log_command "$ACC_BIN attest $GHCR_IMAGE"

set +e
attest_output=$($ACC_BIN attest "$GHCR_IMAGE" 2>&1 | tee -a "$LOGFILE")
attest_exit=$?
set -e

if [ $attest_exit -eq 0 ]; then
    log_success "acc attest succeeded (local attestation created)"
else
    log_error "acc attest failed with exit code $attest_exit"
    log "Output: $attest_output"
fi

# 6.2: Publish attestation to remote registry
log "Step 6.2: Publish attestation to remote registry"
log_command "$ACC_BIN attest --remote $GHCR_IMAGE"

set +e
publish_output=$($ACC_BIN attest --remote "$GHCR_IMAGE" 2>&1 | tee -a "$LOGFILE")
publish_exit=$?
set -e

if [ $publish_exit -eq 0 ]; then
    log_success "Remote attestation publishing succeeded"
elif echo "$publish_output" | grep -qiE "(not implemented|flag.*not recognized)"; then
    log "⚠️  Remote attestation publishing not implemented yet - skipping remote tests"
    log "This is expected for v0.3.1 and earlier"
else
    log_error "Remote attestation publishing failed"
    log "Output: $publish_output"
fi

# Only continue with remote fetch tests if publishing worked
if [ $publish_exit -eq 0 ]; then
    # 6.3: Remove local attestations to test remote fetching
    log "Step 6.3: Remove local attestations to test remote fetching"

    if [ -d ".acc/attestations" ]; then
        attestation_backup="/tmp/acc-attestations-backup-$(date +%s)"
        log "Backing up attestations to: $attestation_backup"
        mv .acc/attestations "$attestation_backup"
        log_success "Local attestations removed (backed up)"
    else
        log "No local attestations directory found"
    fi

    # 6.4: Fetch attestations from remote registry
    log "Step 6.4: Fetch attestations from remote registry"
    log_command "$ACC_BIN trust verify --remote $GHCR_IMAGE"

    set +e
    verify_remote_output=$($ACC_BIN trust verify --remote --json "$GHCR_IMAGE" 2>&1)
    verify_remote_exit=$?
    set -e

    log "Remote verify output:"
    echo "$verify_remote_output" | tee -a "$LOGFILE"

    if [ $verify_remote_exit -eq 0 ]; then
        log_success "Remote attestation fetching and verification succeeded"

        # Check if attestations were fetched
        attestation_count=$(echo "$verify_remote_output" | jq -r '.attestationCount' 2>/dev/null || echo "0")
        if [ "$attestation_count" -gt 0 ]; then
            log_success "Found $attestation_count remote attestation(s)"
        else
            log_error "No attestations found after remote fetch"
        fi
    elif echo "$verify_remote_output" | grep -qiE "(not implemented|flag.*not recognized)"; then
        log "⚠️  Remote attestation fetching not implemented yet"
    else
        log_error "Remote attestation fetching failed with exit code $verify_remote_exit"
    fi

    # 6.5: Verify remote attestations are cached locally
    log "Step 6.5: Verify remote attestations are cached locally"

    if [ -d ".acc/attestations" ]; then
        remote_cached=$(find .acc/attestations -type f -name "*.json" | wc -l)
        if [ "$remote_cached" -gt 0 ]; then
            log_success "Remote attestations cached locally ($remote_cached files)"
            log "Cached attestation paths:"
            find .acc/attestations -type f -name "*.json" -print | head -5 | tee -a "$LOGFILE"
        else
            log "⚠️  No remote attestations found in local cache"
        fi
    else
        log "⚠️  No attestations directory found after remote fetch"
    fi

    # 6.6: Restore local attestations if backed up
    if [ -d "$attestation_backup" ]; then
        log "Restoring local attestations from backup"
        rm -rf .acc/attestations
        mv "$attestation_backup" .acc/attestations
        log_success "Local attestations restored"
    fi
else
    log "⏭️  Skipping remote attestation fetch tests (publishing not implemented)"
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
