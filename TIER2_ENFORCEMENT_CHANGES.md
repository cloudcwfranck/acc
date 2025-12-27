# Tier 2 Enforcement Changes - Complete Summary

## Problem Statement

Tier 2 Registry Integration was being **skipped** on all pull requests, even same-repository PRs where secrets are available. This meant:
- ❌ Changes could merge without Tier 2 validation
- ❌ Issues only discovered post-merge on main branch
- ❌ No way to validate Tier 2 on PRs before merging

## Solution

Make Tier 2 **enforceable** on trusted events while remaining friendly to forked PRs.

---

## File Changes

### 1. `.github/workflows/ci.yml`

#### A) Add workflow_dispatch Trigger

**Before:**
```yaml
on:
  push:
    branches: [ main, claude/* ]
  pull_request:
    branches: [ main ]
  schedule:
    - cron: '0 2 * * *'
```

**After:**
```yaml
on:
  push:
    branches: [ main, claude/* ]
  pull_request:
    branches: [ main ]
  schedule:
    - cron: '0 2 * * *'
  workflow_dispatch:
    # Allow manual triggering (useful for Tier 2 on forks)
```

**Why:** Enables maintainers to manually trigger Tier 2 on fork branches via Actions UI.

---

#### B) Update Tier 2 Job Conditions

**Before:**
```yaml
tier2-registry:
  name: "Tier 2: Registry Integration"
  runs-on: ubuntu-latest
  # Only run on schedule (nightly), tags, or main branch (not PRs)
  if: |
    github.event_name == 'schedule' ||
    startsWith(github.ref, 'refs/tags/') ||
    (github.event_name == 'push' && github.ref == 'refs/heads/main')
```

**After:**
```yaml
tier2-registry:
  name: "Tier 2: Registry Integration"
  runs-on: ubuntu-latest
  # Run on all events EXCEPT forked PRs (where secrets are unavailable)
  if: |
    github.event_name == 'schedule' ||
    github.event_name == 'workflow_dispatch' ||
    startsWith(github.ref, 'refs/tags/') ||
    (github.event_name == 'push' && github.ref == 'refs/heads/main') ||
    (github.event_name == 'pull_request' && github.event.pull_request.head.repo.full_name == github.repository)
```

**Key Change:** Added condition for same-repo PRs:
```yaml
(github.event_name == 'pull_request' && github.event.pull_request.head.repo.full_name == github.repository)
```

This runs Tier 2 on PRs from branches in the same repository (where secrets are available).

---

#### C) Add Environment Variables

**Before:**
```yaml
- name: Run registry integration tests
  env:
    GHCR_REGISTRY: ghcr.io
    GHCR_REPO: ${{ github.repository }}
    GHCR_USERNAME: ${{ github.actor }}
    GITHUB_SHA: ${{ github.sha }}
  run: bash scripts/registry_integration.sh
```

**After:**
```yaml
- name: Run registry integration tests
  env:
    GHCR_REGISTRY: ghcr.io
    GHCR_REPO: ${{ github.repository }}
    GHCR_USERNAME: ${{ github.actor }}
    GHCR_TOKEN: ${{ secrets.GITHUB_TOKEN }}          # ← NEW
    GITHUB_SHA: ${{ github.sha }}
    TIER2_REQUIRED: "true"                           # ← NEW
  run: bash scripts/registry_integration.sh
```

**New Variables:**
- `GHCR_TOKEN`: Passes GITHUB_TOKEN secret (already used by docker/login-action)
- `TIER2_REQUIRED`: Flag telling script this is a trusted event (fail, don't skip)

---

### 2. `scripts/registry_integration.sh`

#### A) Add New Variables

**Added at top of script:**
```bash
GHCR_TOKEN="${GHCR_TOKEN:-}"
TIER2_REQUIRED="${TIER2_REQUIRED:-false}"  # If true, fail instead of skip on missing config
```

---

#### B) Update GHCR_REPO Validation

**Before:**
```bash
# Check if GHCR_REPO is set
if [ -z "$GHCR_REPO" ]; then
    log_skip "GHCR_REPO not set - skipping registry integration tests"
    log "Set GHCR_REPO to enable Tier 2 tests (format: 'OWNER/IMAGE', e.g., 'cloudcwfranck/acc')"
    exit 0
fi
```

**After:**
```bash
# Check if GHCR_REPO is set
if [ -z "$GHCR_REPO" ]; then
    if [ "$TIER2_REQUIRED" = "true" ]; then
        log_error "GHCR_REPO is required but not set"
        log "This is a trusted event (main branch, same-repo PR, or manual trigger)"
        log "GHCR_REPO must be set in format: 'OWNER/IMAGE' (e.g., 'cloudcwfranck/acc')"
        exit 1  # ← FAIL instead of skip
    else
        log_skip "GHCR_REPO not set - skipping registry integration tests (forked PR or local run)"
        log "Set GHCR_REPO to enable Tier 2 tests (format: 'OWNER/IMAGE', e.g., 'cloudcwfranck/acc')"
        exit 0  # ← Skip gracefully
    fi
fi
```

**Behavior:**
- **TIER2_REQUIRED=true**: Exit 1 (fail) with error message
- **TIER2_REQUIRED=false**: Exit 0 (skip) with informational message

---

#### C) Update Docker Authentication Check

**Before:**
```bash
# Check if we have credentials in docker config
if [ -f ~/.docker/config.json ]; then
    if grep -q "ghcr.io" ~/.docker/config.json; then
        log_success "Found GHCR credentials in docker config"
    else
        log_error "No GHCR credentials found in ~/.docker/config.json"
        log "Run: echo \$GHCR_TOKEN | docker login ghcr.io -u \$GHCR_USERNAME --password-stdin"
        exit 1  # ← Always failed
    fi
else
    log_error "Docker config not found at ~/.docker/config.json"
    log "Run: docker login ghcr.io"
    exit 1  # ← Always failed
fi
```

**After:**
```bash
# Check if we have credentials in docker config
if [ -f ~/.docker/config.json ]; then
    if grep -q "ghcr.io" ~/.docker/config.json; then
        log_success "Found GHCR credentials in docker config"
    else
        if [ "$TIER2_REQUIRED" = "true" ]; then
            log_error "No GHCR credentials found in ~/.docker/config.json"
            log "This is a trusted event - authentication is required"
            log "Run: echo \$GHCR_TOKEN | docker login ghcr.io -u \$GHCR_USERNAME --password-stdin"
            exit 1  # ← Fail on trusted events
        else
            log_skip "No GHCR credentials - skipping registry tests (forked PR or local run)"
            log "Run: echo \$GHCR_TOKEN | docker login ghcr.io -u \$GHCR_USERNAME --password-stdin"
            exit 0  # ← Skip on untrusted events
        fi
    fi
else
    if [ "$TIER2_REQUIRED" = "true" ]; then
        log_error "Docker config not found at ~/.docker/config.json"
        log "This is a trusted event - authentication is required"
        log "Run: docker login ghcr.io"
        exit 1  # ← Fail on trusted events
    else
        log_skip "Docker config not found - skipping registry tests (forked PR or local run)"
        log "Run: docker login ghcr.io"
        exit 0  # ← Skip on untrusted events
    fi
fi
```

**Behavior:**
- **TIER2_REQUIRED=true**: Exit 1 with "This is a trusted event - authentication is required"
- **TIER2_REQUIRED=false**: Exit 0 with "skipping registry tests (forked PR or local run)"

---

### 3. `docs/tier2-enforcement.md`

**New comprehensive documentation covering:**
- When Tier 2 runs vs skips
- Enforcement behavior (trusted vs untrusted)
- Required environment variables
- Manual workflow dispatch instructions
- Validation sequence
- Error message reference
- CI status examples
- Testing strategy

---

## Conditions Summary

### ✅ Tier 2 Runs and is ENFORCED (must pass)

| Event | Condition | TIER2_REQUIRED |
|-------|-----------|----------------|
| Push to main | `github.event_name == 'push' && github.ref == 'refs/heads/main'` | true |
| Same-repo PR | `github.event_name == 'pull_request' && github.event.pull_request.head.repo.full_name == github.repository` | true |
| Tags | `startsWith(github.ref, 'refs/tags/')` | true |
| Schedule | `github.event_name == 'schedule'` | true |
| Manual | `github.event_name == 'workflow_dispatch'` | true |

**Behavior:** If GHCR_REPO or credentials missing → **Exit 1 (fail)**

---

### ⏭️ Tier 2 Skips (job doesn't run)

| Event | Reason |
|-------|--------|
| Forked PR | `github.event.pull_request.head.repo.full_name != github.repository` |

**Behavior:** Job skipped entirely at workflow level (never starts)

**Note:** Maintainers can manually trigger via workflow_dispatch to run Tier 2 on fork branches.

---

## How to Run Tier 2 Manually

### From GitHub Actions UI

1. **Navigate to Actions tab** in GitHub repository
2. **Select "CI" workflow** from left sidebar
3. **Click "Run workflow"** dropdown (top right)
4. **Select branch** you want to test (e.g., a fork's branch)
5. **Click "Run workflow"** button

This will:
- ✅ Run all tiers (Tier 0, Tier 1, Tier 2)
- ✅ Use repository secrets (GITHUB_TOKEN)
- ✅ Set TIER2_REQUIRED=true (enforce validation)

### From Command Line (gh CLI)

```bash
# Trigger workflow on main branch
gh workflow run ci.yml --ref main

# Trigger workflow on specific branch
gh workflow run ci.yml --ref feature-branch

# View workflow runs
gh run list --workflow=ci.yml
```

---

## Validation Flow

```
┌─────────────────────────────────────┐
│ GitHub Event Triggered              │
└────────────┬────────────────────────┘
             │
             ▼
┌─────────────────────────────────────┐
│ Is it a forked PR?                  │
│ (head.repo != base.repo)            │
└────────────┬────────────────────────┘
             │
        ┌────┴────┐
        │         │
      YES        NO
        │         │
        ▼         ▼
    ⏭️ Skip   ✅ Run Tier 2
    (no          │
    secrets)     │
                 ▼
        ┌────────────────────┐
        │ Set                │
        │ TIER2_REQUIRED=true│
        └────────┬───────────┘
                 │
                 ▼
        ┌────────────────────┐
        │ Check GHCR_REPO    │
        └────────┬───────────┘
                 │
        ┌────────┴────────┐
        │                 │
     Missing           Valid
        │                 │
        ▼                 ▼
    ❌ Fail          Continue
                         │
                         ▼
                ┌─────────────────┐
                │ Check Docker    │
                │ Credentials     │
                └────────┬────────┘
                         │
                ┌────────┴────────┐
                │                 │
             Missing           Valid
                │                 │
                ▼                 ▼
            ❌ Fail          ✅ Run Tests
```

---

## Error Messages

### Missing GHCR_REPO (Trusted Event)
```
❌ GHCR_REPO is required but not set
This is a trusted event (main branch, same-repo PR, or manual trigger)
GHCR_REPO must be set in format: 'OWNER/IMAGE' (e.g., 'cloudcwfranck/acc')
```

### Missing GHCR_REPO (Local/Fork)
```
⏭️  GHCR_REPO not set - skipping registry integration tests (forked PR or local run)
Set GHCR_REPO to enable Tier 2 tests (format: 'OWNER/IMAGE', e.g., 'cloudcwfranck/acc')
```

### Missing Credentials (Trusted Event)
```
❌ No GHCR credentials found in ~/.docker/config.json
This is a trusted event - authentication is required
Run: echo $GHCR_TOKEN | docker login ghcr.io -u $GHCR_USERNAME --password-stdin
```

### Missing Credentials (Local/Fork)
```
⏭️  No GHCR credentials - skipping registry tests (forked PR or local run)
Run: echo $GHCR_TOKEN | docker login ghcr.io -u $GHCR_USERNAME --password-stdin
```

---

## CI Status Examples

### ✅ Trusted Event - All Passing
```
✓ Tier 0: CLI Help Matrix (28s)
✓ Tier 1: E2E Smoke Tests (1m 15s)
✓ Tier 2: Registry Integration (2m 30s)  ← Enforced and passing
✓ Changelog Check (5s)
```

### ❌ Trusted Event - Tier 2 Failed
```
✓ Tier 0: CLI Help Matrix (28s)
✓ Tier 1: E2E Smoke Tests (1m 15s)
✗ Tier 2: Registry Integration (15s)     ← Failed: credentials missing
✓ Changelog Check (5s)
```

### ⏭️ Forked PR - Tier 2 Skipped
```
✓ Tier 0: CLI Help Matrix (28s)
✓ Tier 1: E2E Smoke Tests (1m 15s)
⊙ Tier 2: Registry Integration           ← Skipped (fork, no secrets)
✓ Changelog Check (5s)
```

---

## Testing Scenarios

### Scenario 1: Same-Repo PR
**Setup:** Branch created in cloudcwfranck/acc
**Trigger:** Pull request opened

**Result:**
- ✅ Tier 2 **runs**
- ✅ Tier 2 **enforced** (must pass)
- ✅ GITHUB_TOKEN available
- ✅ Can push to GHCR

### Scenario 2: Forked PR
**Setup:** Branch in external-user/acc (fork)
**Trigger:** Pull request opened

**Result:**
- ⏭️ Tier 2 **skipped** (job doesn't start)
- ℹ️ Message: "Forked PR - secrets unavailable"
- ✅ Tier 0 and Tier 1 still run
- 🔧 Maintainer can trigger manually

### Scenario 3: Push to Main
**Setup:** Commit pushed directly to main
**Trigger:** Push event

**Result:**
- ✅ Tier 2 **runs**
- ✅ Tier 2 **enforced** (must pass)
- ❌ Fails if GHCR_REPO or credentials missing

### Scenario 4: Manual Trigger
**Setup:** Maintainer clicks "Run workflow"
**Trigger:** workflow_dispatch

**Result:**
- ✅ Tier 2 **runs** on selected branch
- ✅ Tier 2 **enforced**
- ✅ Useful for testing forks before merging

---

## Impact on Development Workflow

### Before Changes
1. Create same-repo PR → Tier 2 **skipped**
2. Merge to main → Tier 2 **runs** (might fail)
3. Discover issue **post-merge** ❌

### After Changes
1. Create same-repo PR → Tier 2 **runs and enforced**
2. Fix issues **before merge** ✅
3. Merge to main → Tier 2 **passes** (already validated)

---

## Commit Information

**Commit:** `dda9d34`
**Message:** "feat(tier2): Make Tier 2 enforceable on trusted events"

**Files Changed:**
- `.github/workflows/ci.yml` - Workflow conditions and env vars
- `scripts/registry_integration.sh` - Enforcement logic
- `docs/tier2-enforcement.md` - Comprehensive documentation

**Branch:** `claude/fix-attestation-publishing-GUFxd`

---

## Summary

✅ **Tier 2 is now enforceable** on trusted events (main, same-repo PRs, tags, schedule, manual)
❌ **Tier 2 fails** (not skips) when credentials are missing on trusted events
⏭️ **Tier 2 skips gracefully** on forked PRs (no secrets available)
🔧 **Maintainers can manually trigger** Tier 2 on any branch via workflow_dispatch
📚 **Comprehensive documentation** explains when/why Tier 2 runs or skips

The changes ensure Tier 2 validation happens before merging while remaining friendly to external contributors.
