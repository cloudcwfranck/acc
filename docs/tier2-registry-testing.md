# Tier 2: Registry Integration Testing

This document describes how to run Tier 2 registry integration tests locally and in CI.

## Overview

Tier 2 tests validate remote attestation publishing and fetching with GitHub Container Registry (GHCR). These tests:
- Are **OPTIONAL** and never block PRs
- Run on: scheduled (nightly), tags, or main branch pushes
- Require GHCR authentication

## Required Environment Variables

### GHCR_REPO (required)
- **Format**: `OWNER/IMAGE` (exactly one slash)
- **Example**: `cloudcwfranck/acc`
- **Invalid**: `cloudcwfranck/acc/test` (too many slashes)
- **Invalid**: `acc` (missing owner)

The script validates this format and will fail-fast with a clear error if invalid.

### GHCR_USERNAME (required)
- **Format**: GitHub username or organization name
- **Example**: `cloudcwfranck`
- **CI Default**: `${{ github.actor }}`

### GHCR_TOKEN (required)
- **Format**: GitHub Personal Access Token or GITHUB_TOKEN
- **Permissions**: `packages:write`
- **CI Default**: `${{ secrets.GITHUB_TOKEN }}`

### GHCR_REGISTRY (optional)
- **Default**: `ghcr.io`
- Usually not needed unless testing against a different registry

### GITHUB_SHA (optional)
- **Default**: Current git commit SHA (short)
- Used as the image tag for uniqueness

## Running Tier 2 Tests Locally

### 1. Login to GHCR

```bash
# Set your credentials
export GHCR_USERNAME="your-github-username"
export GHCR_TOKEN="your-github-token"  # or use GITHUB_TOKEN

# Login to GHCR
echo "$GHCR_TOKEN" | docker login ghcr.io -u "$GHCR_USERNAME" --password-stdin
```

### 2. Set Environment Variables

```bash
# Required: Repository in format OWNER/IMAGE
export GHCR_REPO="your-username/acc"

# Optional: Override defaults
export GHCR_REGISTRY="ghcr.io"
export GITHUB_SHA="$(git rev-parse --short HEAD)"
```

### 3. Build acc Binary

```bash
go build -o acc ./cmd/acc
```

### 4. Run Tier 2 Tests

```bash
bash scripts/registry_integration.sh
```

## Image Naming Convention

The test creates images following this pattern:

```
${GHCR_REGISTRY}/${GHCR_REPO}:${GITHUB_SHA}
```

**Examples:**
- `ghcr.io/cloudcwfranck/acc:a02c1b8`
- `ghcr.io/myorg/myimage:1234567`

**Important:** GHCR requires exactly 2 path segments (owner/image). The old pattern of `ghcr.io/owner/repo/extra` will **fail with 403**.

## What the Tests Validate

### TEST 1: Initialize Project
- Creates test project with acc

### TEST 2: Build and Verify
- Builds a test image
- Generates SBOM with `acc build`
- Verifies with `acc verify`

### TEST 3: Push to GHCR
- Tags image for GHCR
- **Validates authentication** by pushing with `docker push`
- Tests `acc push` (if implemented)

### TEST 4: Promote (optional)
- Tests `acc promote` command (may not be implemented)

### TEST 5: Pull and Re-verify
- Removes local images
- Pulls from GHCR
- Re-verifies pulled image

### TEST 6: Remote Attestation Publishing and Fetching
- Creates local attestation with `acc attest`
- **Publishes attestation to GHCR** with `acc attest --remote`
- Removes local attestations
- **Fetches attestations from GHCR** with `acc trust verify --remote`
- Validates attestation count > 0
- Verifies remote attestations are cached locally

## Troubleshooting

### 403 Forbidden Errors

**Symptom:**
```
POST "https://ghcr.io/v2/...": response status code 403: denied
```

**Common Causes:**

1. **Wrong GHCR_REPO format**
   - ❌ `cloudcwfranck/acc/extra` (3 segments)
   - ✅ `cloudcwfranck/acc` (2 segments)

2. **Not logged in to GHCR**
   ```bash
   echo "$GHCR_TOKEN" | docker login ghcr.io -u "$GHCR_USERNAME" --password-stdin
   ```

3. **Insufficient token permissions**
   - Token needs `packages:write` scope
   - In CI, workflow needs `permissions: packages: write`

4. **Wrong repository owner**
   - Ensure GHCR_REPO owner matches your username/org
   - Cannot push to other users' namespaces

### No Credentials Found

**Symptom:**
```
No GHCR credentials found in ~/.docker/config.json
```

**Solution:**
```bash
docker login ghcr.io
```

### Tests Skipped

**Symptom:**
```
⏭️  GHCR_REPO not set - skipping registry integration tests
```

**Solution:**
Set GHCR_REPO environment variable:
```bash
export GHCR_REPO="your-username/acc"
```

## CI/CD Integration

The GitHub Actions workflow automatically:
1. Sets up Go and dependencies
2. Builds acc binary
3. Logs in to GHCR using `docker/login-action`
4. Sets environment variables:
   - `GHCR_REGISTRY=ghcr.io`
   - `GHCR_REPO=${{ github.repository }}`
   - `GHCR_USERNAME=${{ github.actor }}`
   - `GITHUB_SHA=${{ github.sha }}`
5. Runs `scripts/registry_integration.sh`

### Workflow Permissions

The Tier 2 job requires:
```yaml
permissions:
  contents: read
  packages: write  # Required for GHCR push
```

## Files Modified

- `.github/workflows/ci.yml` - Added GHCR_USERNAME env var
- `scripts/registry_integration.sh` - Fixed GHCR naming and auth validation
- `internal/attest/attest.go` - Fixed Docker credential decoding
- `internal/trust/status.go` - Fixed Docker credential decoding

## Authentication Implementation

The acc tool now properly:
1. Reads `~/.docker/config.json`
2. Decodes base64-encoded auth credentials
3. Extracts username and password
4. Passes credentials to oras-go for OCI operations
5. Tries multiple registry URL formats (ghcr.io, https://ghcr.io, https://ghcr.io/v2/)

This ensures that `acc attest --remote` and `acc trust verify --remote` work correctly with GHCR authentication.
