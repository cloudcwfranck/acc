# Tier 2: Enforcement and Execution Policy

## Overview

Tier 2 Registry Integration tests are **enforceable** on trusted events and will **fail** (not skip) when credentials are missing.

## When Tier 2 Runs

### ✅ Always Runs (Enforceable)
1. **Push to `main` branch** - Required for merging
2. **Pull requests from same repository** - Same-repo branches have access to secrets
3. **Tagged releases** - `refs/tags/*`
4. **Scheduled runs** - Nightly at 2 AM UTC
5. **Manual workflow dispatch** - Triggered from Actions UI

### ⏭️ Skipped (Optional)
1. **Forked pull requests** - External contributors don't have access to secrets
   - Shows clear message: "GHCR credentials - skipping registry tests (forked PR or local run)"
   - Maintainers can trigger manually via workflow_dispatch

## Enforcement Behavior

### Trusted Events (TIER2_REQUIRED=true)
When running on trusted events, Tier 2 **MUST NOT SKIP**:

- ❌ **Missing GHCR_REPO** → Exit 1 (fail)
  ```
  Error: GHCR_REPO is required but not set
  This is a trusted event (main branch, same-repo PR, or manual trigger)
  GHCR_REPO must be set in format: 'OWNER/IMAGE'
  ```

- ❌ **Missing Docker credentials** → Exit 1 (fail)
  ```
  Error: No GHCR credentials found in ~/.docker/config.json
  This is a trusted event - authentication is required
  ```

- ❌ **Invalid GHCR_REPO format** → Exit 1 (fail)
  ```
  Error: GHCR_REPO must be in format 'OWNER/IMAGE' with exactly one slash
  Got: cloudcwfranck/acc/extra (found 2 slashes)
  ```

### Untrusted Events (TIER2_REQUIRED=false)
When running locally or on forked PRs:

- ⏭️ **Missing GHCR_REPO** → Exit 0 (skip)
- ⏭️ **Missing credentials** → Exit 0 (skip)
- ❌ **Invalid GHCR_REPO format** → Exit 1 (fail)

## Required Environment Variables

### CI (GitHub Actions)
All automatically set for trusted events:

```yaml
GHCR_REGISTRY: ghcr.io
GHCR_REPO: ${{ github.repository }}        # e.g., "cloudcwfranck/acc"
GHCR_USERNAME: ${{ github.actor }}          # GitHub username
GHCR_TOKEN: ${{ secrets.GITHUB_TOKEN }}     # Auto-injected secret
GITHUB_SHA: ${{ github.sha }}               # Commit SHA
TIER2_REQUIRED: "true"                      # Enforces failure on missing config
```

### Local Testing
For local development:

```bash
export GHCR_REPO="cloudcwfranck/acc"     # Format: OWNER/IMAGE
export GHCR_USERNAME="cloudcwfranck"     # Your username
export GHCR_TOKEN="ghp_xxxxxxxxxxxx"     # Your PAT
export TIER2_REQUIRED="false"            # Optional: skip if credentials missing
```

## Manual Workflow Dispatch

### From GitHub UI

1. Go to **Actions** tab in GitHub
2. Select **CI** workflow
3. Click **Run workflow** dropdown
4. Select branch (usually `main`)
5. Click **Run workflow**

This will:
- ✅ Run all Tier 0, Tier 1, and Tier 2 tests
- ✅ Use repository secrets (GITHUB_TOKEN)
- ✅ Enforce Tier 2 (fail if credentials missing)

### Use Cases
- **Test Tier 2 on a fork** - Maintainer can manually trigger on fork branches
- **Re-run failed Tier 2** - Retry without pushing new commits
- **Test before merge** - Validate Tier 2 passes before merging PR

## Workflow Conditions

### Job Execution Logic

```yaml
# Tier 2 runs when ANY of these is true:
if: |
  github.event_name == 'schedule' ||
  github.event_name == 'workflow_dispatch' ||
  startsWith(github.ref, 'refs/tags/') ||
  (github.event_name == 'push' && github.ref == 'refs/heads/main') ||
  (github.event_name == 'pull_request' && github.event.pull_request.head.repo.full_name == github.repository)
```

**Breakdown:**
- `schedule` → Nightly runs
- `workflow_dispatch` → Manual trigger
- `refs/tags/*` → Release tags
- `push` to `main` → Main branch updates
- `pull_request` from **same repo** → Same-repo PRs (has secrets)

**Skipped when:**
- `pull_request` from **different repo** → Forked PRs (no secrets)

## Permissions

The CI workflow has:
```yaml
permissions:
  contents: read
  packages: write  # Required for GHCR push
```

This allows:
- ✅ Reading repository code
- ✅ Publishing to GitHub Container Registry
- ✅ Publishing attestations as OCI artifacts

## Validation Sequence

```
1. Check TIER2_REQUIRED flag
   ├─ true  → Fail on missing config
   └─ false → Skip on missing config

2. Validate GHCR_REPO
   ├─ Not set → Fail/Skip based on flag
   ├─ Invalid format (≠ OWNER/IMAGE) → Always fail
   └─ Valid → Continue

3. Validate Docker credentials
   ├─ ~/.docker/config.json missing → Fail/Skip based on flag
   ├─ No ghcr.io entry → Fail/Skip based on flag
   └─ Valid → Continue

4. Run Tier 2 tests
   └─ If any test fails → Exit 1
```

## Error Messages

### Missing GHCR_REPO (Trusted)
```
❌ GHCR_REPO is required but not set
This is a trusted event (main branch, same-repo PR, or manual trigger)
GHCR_REPO must be set in format: 'OWNER/IMAGE' (e.g., 'cloudcwfranck/acc')
```

### Missing GHCR_REPO (Untrusted)
```
⏭️  GHCR_REPO not set - skipping registry integration tests (forked PR or local run)
Set GHCR_REPO to enable Tier 2 tests (format: 'OWNER/IMAGE', e.g., 'cloudcwfranck/acc')
```

### Invalid Format
```
❌ GHCR_REPO must be in format 'OWNER/IMAGE' with exactly one slash
Got: cloudcwfranck/acc/test (found 2 slashes)
Example: GHCR_REPO='cloudcwfranck/acc'
```

### Missing Credentials (Trusted)
```
❌ No GHCR credentials found in ~/.docker/config.json
This is a trusted event - authentication is required
Run: echo $GHCR_TOKEN | docker login ghcr.io -u $GHCR_USERNAME --password-stdin
```

### Missing Credentials (Untrusted)
```
⏭️  No GHCR credentials - skipping registry tests (forked PR or local run)
Run: echo $GHCR_TOKEN | docker login ghcr.io -u $GHCR_USERNAME --password-stdin
```

## CI Status Examples

### ✅ Passing (Trusted Event)
```
✓ Tier 0: CLI Help Matrix
✓ Tier 1: E2E Smoke Tests
✓ Tier 2: Registry Integration  ← Must pass
✓ Changelog Check
```

### ⏭️ Skipped (Forked PR)
```
✓ Tier 0: CLI Help Matrix
✓ Tier 1: E2E Smoke Tests
⊙ Tier 2: Registry Integration  ← Skipped (fork)
✓ Changelog Check
```

### ❌ Failed (Missing Config)
```
✓ Tier 0: CLI Help Matrix
✓ Tier 1: E2E Smoke Tests
✗ Tier 2: Registry Integration  ← Failed: GHCR_REPO not set
✓ Changelog Check
```

## Testing Strategy

### Before Merge
1. **Local**: Run with TIER2_REQUIRED=false (optional)
2. **PR**: Tier 2 runs automatically (same-repo PR)
3. **Main**: Tier 2 runs on merge to main
4. **Manual**: Can trigger via workflow_dispatch

### For Forks
1. **External PR**: Tier 2 skipped (no secrets)
2. **Maintainer Review**: Can trigger manually
3. **After Merge**: Runs on main branch

## Summary

| Event Type | Tier 2 Runs? | Enforced? | Secrets Available? |
|------------|--------------|-----------|-------------------|
| Push to main | ✅ Yes | ✅ Yes | ✅ Yes |
| Same-repo PR | ✅ Yes | ✅ Yes | ✅ Yes |
| Forked PR | ⏭️ Skipped | ❌ No | ❌ No |
| Tag/Release | ✅ Yes | ✅ Yes | ✅ Yes |
| Scheduled | ✅ Yes | ✅ Yes | ✅ Yes |
| Manual | ✅ Yes | ✅ Yes | ✅ Yes |
| Local dev | ⏭️ Optional | ❌ No | Manual setup |

**Key Points:**
- ✅ Tier 2 is **enforceable** on trusted events
- ❌ Tier 2 **fails** (not skips) when config is missing on trusted events
- ⏭️ Tier 2 **skips gracefully** on forked PRs (no secrets)
- 🔧 Maintainers can **manually trigger** Tier 2 on any branch
