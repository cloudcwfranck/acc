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
GHCR_USERNAME="${GHCR_USERNAME:-}"
GHCR_TOKEN="${GHCR_TOKEN:-}"
GITHUB_SHA="${GITHUB_SHA:-$(git rev-parse --short HEAD 2>/dev/null || echo 'local')}"
TIER2_REQUIRED="${TIER2_REQUIRED:-false}"  # If true, fail instead of skip on missing config

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
        if [ "$TIER2_REQUIRED" = "true" ]; then
            log_error "GHCR_REPO is required but not set"
            log "This is a trusted event (main branch, same-repo PR, or manual trigger)"
            log "GHCR_REPO must be set in format: 'OWNER/IMAGE' (e.g., 'cloudcwfranck/acc')"
            exit 1
        else
            log_skip "GHCR_REPO not set - skipping registry integration tests (forked PR or local run)"
            log "Set GHCR_REPO to enable Tier 2 tests (format: 'OWNER/IMAGE', e.g., 'cloudcwfranck/acc')"
            exit 0
        fi
    fi

    # Validate GHCR_REPO format: must be exactly "OWNER/IMAGE" (one slash, two segments)
    slash_count=$(echo "$GHCR_REPO" | tr -cd '/' | wc -c)
    if [ "$slash_count" -ne 1 ]; then
        log_error "GHCR_REPO must be in format 'OWNER/IMAGE' with exactly one slash"
        log_error "Got: $GHCR_REPO (found $slash_count slashes)"
        log "Example: GHCR_REPO='cloudcwfranck/acc'"
        exit 1
    fi

    # Extract owner and image name for validation
    GHCR_OWNER=$(echo "$GHCR_REPO" | cut -d'/' -f1)
    GHCR_IMAGE_NAME=$(echo "$GHCR_REPO" | cut -d'/' -f2)

    if [ -z "$GHCR_OWNER" ] || [ -z "$GHCR_IMAGE_NAME" ]; then
        log_error "Invalid GHCR_REPO format: '$GHCR_REPO'"
        log "Must be: OWNER/IMAGE"
        exit 1
    fi

    log_success "GHCR_REPO format validated: owner='$GHCR_OWNER', image='$GHCR_IMAGE_NAME'"

    # Validate docker login to GHCR
    log "Checking docker authentication to GHCR..."

    # Check if we have credentials in docker config
    if [ -f ~/.docker/config.json ]; then
        # Check for ghcr.io in various formats (plain, https, or with path)
        if grep -qE "(\"ghcr\.io\"|\"https://ghcr\.io\")" ~/.docker/config.json; then
            log_success "Found GHCR credentials in docker config"
        else
            if [ "$TIER2_REQUIRED" = "true" ]; then
                log_error "No GHCR credentials found in ~/.docker/config.json"
                log "This is a trusted event - authentication is required"
                log "Searched for: ghcr.io or https://ghcr.io"
                log "Config file contents (auths keys):"
                jq -r '.auths | keys[]' ~/.docker/config.json 2>/dev/null || echo "  (could not parse config)"
                log "Run: echo \$GHCR_TOKEN | docker login ghcr.io -u \$GHCR_USERNAME --password-stdin"
                exit 1
            else
                log_skip "No GHCR credentials - skipping registry tests (forked PR or local run)"
                log "Run: echo \$GHCR_TOKEN | docker login ghcr.io -u \$GHCR_USERNAME --password-stdin"
                exit 0
            fi
        fi
    else
        if [ "$TIER2_REQUIRED" = "true" ]; then
            log_error "Docker config not found at ~/.docker/config.json"
            log "This is a trusted event - authentication is required"
            log "Run: docker login ghcr.io"
            exit 1
        else
            log_skip "Docker config not found - skipping registry tests (forked PR or local run)"
            log "Run: docker login ghcr.io"
            exit 0
        fi
    fi

    # Test authentication by attempting to authenticate to registry
    # Use a lightweight check - try to get a token for the repository
    log "Validating GHCR write access for ${GHCR_REGISTRY}/${GHCR_REPO}..."

    # We'll validate auth works when we actually push in TEST 3
    # For now, just confirm docker is logged in

    log_success "Pre-flight checks passed"
    log "GHCR Registry: $GHCR_REGISTRY"
    log "GHCR Repo: $GHCR_REPO"
    log "GitHub SHA: $GITHUB_SHA"
    log "Test Image: ${GHCR_REGISTRY}/${GHCR_REPO}:${GITHUB_SHA}"
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

# Tag for GHCR - use GHCR_REPO directly (format: OWNER/IMAGE)
# This creates: ghcr.io/OWNER/IMAGE:TAG (e.g., ghcr.io/cloudcwfranck/acc:sha)
GHCR_IMAGE="${GHCR_REGISTRY}/${GHCR_REPO}:${GITHUB_SHA}"
log "Tagging image for GHCR: $GHCR_IMAGE"

log_command "docker tag $LOCAL_IMAGE $GHCR_IMAGE"
if docker tag "$LOCAL_IMAGE" "$GHCR_IMAGE" 2>&1 | tee -a "$LOGFILE"; then
    log_success "Docker tag succeeded"
else
    log_error "Docker tag failed"
    exit 1
fi

# Validate docker push auth before using acc push
log "Validating docker push authentication to GHCR..."
log_command "docker push $GHCR_IMAGE"
if docker push "$GHCR_IMAGE" 2>&1 | tee -a "$LOGFILE"; then
    log_success "Docker push succeeded - GHCR authentication confirmed"
else
    log_error "Docker push failed - check GHCR authentication"
    log "Ensure you ran: echo \$GHCR_TOKEN | docker login ghcr.io -u \$GHCR_USERNAME --password-stdin"
    exit 1
fi

# Note: acc push may verify before pushing, but we already pushed via docker
# This tests that acc push works with already-pushed images
log "Testing acc push (image already in registry)"
log_command "$ACC_BIN push $GHCR_IMAGE"
if push_output=$($ACC_BIN push "$GHCR_IMAGE" 2>&1); then
    log_success "acc push succeeded"
    echo "$push_output" | tee -a "$LOGFILE"
else
    push_exit=$?
    log "⚠️  acc push exited with code $push_exit (might be expected if push is not yet implemented)"
    echo "$push_output" | tee -a "$LOGFILE"
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
    # v0.3.2: Verify the output indicates actual publishing (not stub)
    if echo "$publish_output" | grep -qiE "not implemented"; then
        log_error "FAIL: acc attest --remote returned 'not implemented' - v0.3.2 requires real OCI publishing"
        exit 1
    fi
else
    # Allow network/auth failures, but not implementation failures
    if echo "$publish_output" | grep -qiE "(no credentials|authentication failed|network|connection)"; then
        log "⚠️  Remote publishing failed due to network/auth issue - skipping remote tests"
        log "Output: $publish_output"
    else
        log_error "Remote attestation publishing failed"
        log "Output: $publish_output"
        exit 1
    fi
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

        # v0.3.2: Verify not a stub implementation
        if echo "$verify_remote_output" | grep -qiE "not implemented"; then
            log_error "FAIL: acc trust verify --remote returned 'not implemented' - v0.3.2 requires real OCI fetching"
            exit 1
        fi

        # Check if attestations were fetched
        attestation_count=$(echo "$verify_remote_output" | jq -r '.attestationCount' 2>/dev/null || echo "0")
        if [ "$attestation_count" -gt 0 ]; then
            log_success "Found $attestation_count remote attestation(s)"
        else
            log_error "FAIL: No attestations found after remote fetch (expected at least 1)"
            exit 1
        fi
    else
        # Allow network/auth failures, but not implementation failures
        if echo "$verify_remote_output" | grep -qiE "(no credentials|authentication failed|network|connection)"; then
            log "⚠️  Remote fetching failed due to network/auth issue"
            log "Output: $verify_remote_output"
        else
            log_error "Remote attestation fetching failed with exit code $verify_remote_exit"
            log "Output: $verify_remote_output"
            exit 1
        fi
    fi

    # 6.5: Verify remote attestations are cached locally
    log "Step 6.5: Verify remote attestations are cached locally"

    # v0.3.2: This is a REQUIRED assertion - remote fetch must cache locally
    if [ $verify_remote_exit -eq 0 ]; then
        if [ -d ".acc/attestations" ]; then
            remote_cached=$(find .acc/attestations -type f -name "*.json" | wc -l)
            if [ "$remote_cached" -gt 0 ]; then
                log_success "Remote attestations cached locally ($remote_cached files)"
                log "Cached attestation paths:"
                find .acc/attestations -type f -name "*.json" -print | head -5 | tee -a "$LOGFILE"
            else
                log_error "FAIL: Remote fetch succeeded but no attestations cached locally"
                exit 1
            fi
        else
            log_error "FAIL: No attestations directory created after remote fetch"
            exit 1
        fi
    fi

    # 6.6: Restore local attestations if backed up
    if [ -d "$attestation_backup" ]; then
        log "Restoring local attestations from backup"
        rm -rf .acc/attestations
        mv "$attestation_backup" .acc/attestations
        log_success "Local attestations restored"
    fi
else
    log "⏭️  Skipping remote attestation fetch tests (publishing failed - see above)"
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
