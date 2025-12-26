# Tier 2 Registry Integration Fix - Summary

## Problem Statement

Tier 2 Registry Integration tests were failing with:
```
❌ failed to publish attestation to remote registry: failed to push attestation:
POST "https://ghcr.io/v2/cloudcwfranck/acc/acc-ci-test/blobs/uploads/":
GET "https://ghcr.io/token?...": response status code 403: denied:
requested access to the resource is denied
```

**Root Causes:**
1. **Authentication**: Docker credentials were not being properly decoded from `~/.docker/config.json`
2. **Repository Naming**: Image refs were constructed as `ghcr.io/OWNER/REPO/IMAGE:TAG` (3 segments) when GHCR requires `ghcr.io/OWNER/IMAGE:TAG` (2 segments)

## Changes Made

### Commit 1: Fix Docker Credential Decoding
**Commit**: `a02c1b8` - "fix: Properly decode Docker credentials for remote attestation publishing"

**Files Changed:**
- `internal/attest/attest.go`
- `internal/trust/status.go`

**What Changed:**
1. Added `encoding/base64` import
2. Updated `loadDockerCredentials()` function to:
   - Decode base64-encoded auth from Docker config
   - Extract username and password from "username:password" format
   - Support both base64 auth and direct username/password fields
   - Try multiple registry URL formats (ghcr.io, https://ghcr.io, https://ghcr.io/v2/)
3. Improved credential loading to check all common Docker config formats

**Before:**
```go
// Look for registry auth
if authEntry, ok := config.Auths[registry]; ok && authEntry.Auth != "" {
    // For now, return empty - oras-go will use docker credential helpers
    return auth.Credential{}, nil
}
```

**After:**
```go
// Try base64-encoded auth
if authEntry.Auth != "" {
    decoded, err := base64.StdEncoding.DecodeString(authEntry.Auth)
    if err != nil {
        return auth.Credential{}, fmt.Errorf("failed to decode auth: %w", err)
    }

    parts := strings.SplitN(string(decoded), ":", 2)
    if len(parts) != 2 {
        return auth.Credential{}, fmt.Errorf("invalid auth format")
    }

    return auth.Credential{
        Username: parts[0],
        Password: parts[1],
    }, nil
}
```

### Commit 2: Fix GHCR Repository Naming and Validation
**Commit**: `cad1809` - "fix(tier2): Fix GHCR repository naming and authentication validation"

**Files Changed:**
- `.github/workflows/ci.yml`
- `scripts/registry_integration.sh`
- `docs/tier2-registry-testing.md` (new)

#### A) Workflow Changes (`.github/workflows/ci.yml`)

**Added Environment Variable:**
```yaml
- name: Run registry integration tests
  env:
    GHCR_REGISTRY: ghcr.io
    GHCR_REPO: ${{ github.repository }}
    GHCR_USERNAME: ${{ github.actor }}  # ← NEW
    GITHUB_SHA: ${{ github.sha }}
  run: bash scripts/registry_integration.sh
```

#### B) Script Changes (`scripts/registry_integration.sh`)

**1. Added GHCR_USERNAME variable:**
```bash
GHCR_USERNAME="${GHCR_USERNAME:-}"
```

**2. Enhanced Pre-flight Checks:**
```bash
# Validate GHCR_REPO format: must be exactly "OWNER/IMAGE" (one slash, two segments)
slash_count=$(echo "$GHCR_REPO" | tr -cd '/' | wc -c)
if [ "$slash_count" -ne 1 ]; then
    log_error "GHCR_REPO must be in format 'OWNER/IMAGE' with exactly one slash"
    log_error "Got: $GHCR_REPO (found $slash_count slashes)"
    exit 1
fi

# Check docker authentication
if grep -q "ghcr.io" ~/.docker/config.json; then
    log_success "Found GHCR credentials in docker config"
else
    log_error "No GHCR credentials found in ~/.docker/config.json"
    exit 1
fi
```

**3. Fixed Image Naming (TEST 3):**

**Before:**
```bash
GHCR_IMAGE="${GHCR_REGISTRY}/${GHCR_REPO}/acc-ci-test:${GITHUB_SHA}"
# Results in: ghcr.io/cloudcwfranck/acc/acc-ci-test:sha (3 segments - WRONG!)
```

**After:**
```bash
GHCR_IMAGE="${GHCR_REGISTRY}/${GHCR_REPO}:${GITHUB_SHA}"
# Results in: ghcr.io/cloudcwfranck/acc:sha (2 segments - CORRECT!)
```

**4. Added Docker Push Validation:**
```bash
# Validate docker push auth before using acc push
log "Validating docker push authentication to GHCR..."
if docker push "$GHCR_IMAGE" 2>&1 | tee -a "$LOGFILE"; then
    log_success "Docker push succeeded - GHCR authentication confirmed"
else
    log_error "Docker push failed - check GHCR authentication"
    exit 1
fi
```

#### C) Documentation (`docs/tier2-registry-testing.md`)

Created comprehensive documentation covering:
- Required environment variables and their formats
- Local testing instructions
- Troubleshooting guide for common errors
- CI/CD integration details
- Image naming conventions

## Required Environment Variables

### For CI (GitHub Actions)
All automatically set by workflow:
- `GHCR_REGISTRY=ghcr.io`
- `GHCR_REPO=${{ github.repository }}` (e.g., "cloudcwfranck/acc")
- `GHCR_USERNAME=${{ github.actor }}`
- `GITHUB_SHA=${{ github.sha }}`

### For Local Testing

```bash
# Required: Login to GHCR first
export GHCR_USERNAME="your-github-username"
export GHCR_TOKEN="your-github-token"
echo "$GHCR_TOKEN" | docker login ghcr.io -u "$GHCR_USERNAME" --password-stdin

# Required: Repository in format OWNER/IMAGE
export GHCR_REPO="your-username/acc"

# Optional: Defaults are usually fine
export GHCR_REGISTRY="ghcr.io"
export GITHUB_SHA="$(git rev-parse --short HEAD)"

# Run tests
bash scripts/registry_integration.sh
```

## Exact Commands to Reproduce Locally

```bash
# 1. Build acc binary
go build -o acc ./cmd/acc

# 2. Set up environment
export GHCR_REPO="cloudcwfranck/acc"  # Replace with your repo
export GHCR_USERNAME="cloudcwfranck"   # Replace with your username
export GHCR_TOKEN="ghp_xxxxxxxxxxxx"    # Your GitHub token

# 3. Login to GHCR
echo "$GHCR_TOKEN" | docker login ghcr.io -u "$GHCR_USERNAME" --password-stdin

# 4. Run Tier 2 tests
bash scripts/registry_integration.sh
```

## What Gets Tested

### TEST 6: Remote Attestation Publishing and Fetching
This is the critical test that was failing. It now:

1. **Creates local attestation** for verified image:
   ```bash
   acc attest ghcr.io/cloudcwfranck/acc:sha
   ```

2. **Publishes attestation to GHCR** (uses fixed credentials):
   ```bash
   acc attest --remote ghcr.io/cloudcwfranck/acc:sha
   ```
   - Now properly decodes Docker credentials
   - Uses correct registry URL format
   - Pushes to: `ghcr.io/cloudcwfranck/acc` (not `ghcr.io/cloudcwfranck/acc/extra`)

3. **Fetches attestations from GHCR**:
   ```bash
   acc trust verify --remote --json ghcr.io/cloudcwfranck/acc:sha
   ```

4. **Validates results**:
   - Attestation count > 0
   - Attestations cached locally in `.acc/attestations/`

## Image Naming Examples

### Correct (2 segments)
✅ `ghcr.io/cloudcwfranck/acc:a02c1b8`
✅ `ghcr.io/myorg/myimage:v1.0.0`
✅ `ghcr.io/user123/test:latest`

### Incorrect (3+ segments)
❌ `ghcr.io/cloudcwfranck/acc/test:sha` (too many segments)
❌ `ghcr.io/org/team/project/app:tag` (way too many)

### Repository Format
- **GHCR_REPO**: `OWNER/IMAGE` (exactly one slash)
- **Final Image**: `ghcr.io/${GHCR_REPO}:${TAG}`

## Validation Flow

```
1. Pre-flight Checks
   ├─ GHCR_REPO set? → Exit 0 if not (skip tests)
   ├─ GHCR_REPO format valid? → Exit 1 if invalid
   └─ Docker logged in? → Exit 1 if not

2. Build Test Image
   ├─ docker build → local image
   ├─ acc build → generate SBOM
   └─ acc verify → ensure it passes

3. Push to GHCR (FIXED!)
   ├─ Tag: ghcr.io/OWNER/IMAGE:TAG (2 segments ✅)
   ├─ docker push → validates auth works
   └─ acc push → tests acc command

4. Remote Attestation (FIXED!)
   ├─ acc attest → create local attestation
   ├─ acc attest --remote → publish to GHCR
   │  └─ Uses decoded Docker credentials ✅
   ├─ Remove local attestations
   ├─ acc trust verify --remote → fetch from GHCR
   │  └─ Uses decoded Docker credentials ✅
   └─ Validate: count > 0, cached locally
```

## Troubleshooting

### 403 Forbidden
**Cause**: Wrong GHCR_REPO format or bad auth
**Fix**:
```bash
# Ensure format is OWNER/IMAGE
export GHCR_REPO="username/image"  # Not "username/repo/image"

# Re-login to GHCR
echo "$GHCR_TOKEN" | docker login ghcr.io -u "$GHCR_USERNAME" --password-stdin
```

### No Credentials Found
**Cause**: Not logged in to Docker
**Fix**:
```bash
docker login ghcr.io
```

### Invalid GHCR_REPO Format
**Cause**: Multiple slashes in GHCR_REPO
**Fix**:
```bash
# Wrong
export GHCR_REPO="cloudcwfranck/acc/test"  # ❌ 2 slashes

# Right
export GHCR_REPO="cloudcwfranck/acc"       # ✅ 1 slash
```

## Summary

✅ **Fixed Docker credential decoding** - Properly extracts username/password from base64 auth
✅ **Fixed GHCR repository naming** - Uses correct 2-segment format
✅ **Added authentication validation** - Fails fast with clear errors
✅ **Added comprehensive documentation** - Easy to test locally
✅ **All tests passing** - Both Go tests and bash syntax validation

The Tier 2 tests should now reliably pass in CI when GHCR credentials are available, and can be easily reproduced locally with the provided commands.
