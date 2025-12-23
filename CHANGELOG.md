# Changelog

All notable changes to acc will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed - Demo Workflow Upload Job Running on PRs

**Summary**: Fixed demo validation workflow configuration bug where the `upload-to-release` job was incorrectly running on pull requests and failing due to missing artifacts.

**What's Fixed:**

- ✅ **Workflow Condition** - Added explicit `github.event_name == 'push'` check to upload-to-release job
- ✅ **PR Stability** - Job now only runs on version tags, not on pull requests
- ✅ **Clear Intent** - Updated condition from `if: startsWith(github.ref, 'refs/tags/v')` to `if: github.event_name == 'push' && startsWith(github.ref, 'refs/tags/v')`

**Why This Matters:**

The `upload-to-release` job depends on the `demo-recording` artifact which is only created on version tag pushes. When the job ran on PRs (due to incomplete condition), it failed trying to download a non-existent artifact. This fix ensures the job is completely skipped on PRs, improving CI stability.

**Files Modified:**

- `.github/workflows/demo.yml` - Updated upload-to-release job condition

**Impact:** Workflow infrastructure fix, no user-facing changes

---

### Added - v0.3.1: Trust Enforcement for Run and Push

**Summary**: Optional attestation enforcement extends `acc run` and `acc push` to require verified attestations before execution or registry push. This opt-in policy gate ensures only attested workloads proceed, strengthening supply chain security.

**What's New:**

- ✅ **Optional Attestation Enforcement** - New config field `policy.requireAttestation` (default: false)
- ✅ **Run Command Enforcement** - `acc run` blocks execution if attestations unverified when enforcement enabled
- ✅ **Push Command Enforcement** - `acc push` blocks registry push if attestations unverified when enforcement enabled
- ✅ **Backward Compatible** - Default behavior unchanged (preserves v0.3.0 behavior)
- ✅ **Clear Remediation** - User-friendly error messages with step-by-step fix instructions
- ✅ **Testing Contract Updated** - Documented in docs/testing-contract.md with version history

**Configuration:**

```yaml
policy:
  mode: enforce
  requireAttestation: true  # v0.3.1: require verified attestations for run/push
```

**Behavior:**

1. **Enforcement Disabled (default)**: `acc run` and `acc push` work as before (v0.3.0 behavior)
2. **Enforcement Enabled**: Commands check attestations via `acc trust verify` logic
   - If attestations verified: proceed normally (exit 0)
   - If attestations unverified/missing: block execution (exit 1) with remediation steps
3. **Exit Code Preserved**: Enforcement uses exit 1 (same as verification gate)

**Remediation Example:**

```
❌ Attestation requirement not met - workload will NOT run

Remediation:
  1. Verify the workload: acc verify demo-app:latest
  2. Create attestation: acc attest demo-app:latest
  3. Re-run: acc run demo-app:latest
```

**Implementation:**

- `internal/config/config.go`: Added `RequireAttestation` field to `PolicyConfig`
- `internal/runtime/run.go`: Enforcement check before container execution
- `internal/push/push.go`: Enforcement check before registry push
- `scripts/e2e_smoke.sh`: TEST 10 with 4 test cases (baseline, with attestation, without attestation, unknown image)

**Testing Contract (v0.3.1):**

- Exit code semantics: Run/push exit 1 includes "verification failed OR attestation enforcement blocked"
- Default behavior unchanged: `requireAttestation: false` preserves v0.3.0
- Breaking change guardrail: Changing default to `true` would require MAJOR version bump

**Files Modified:**

- `internal/config/config.go` - Added `RequireAttestation bool` field
- `internal/runtime/run.go` - Trust enforcement check with remediation
- `internal/push/push.go` - Trust enforcement check with remediation
- `scripts/e2e_smoke.sh` - TEST 10: Trust Enforcement (4 test cases)
- `docs/testing-contract.md` - v0.3.1 section in version history, updated Run Command guarantees, updated exit code table

**Acceptance Criteria:**

- ✅ Tier 0 tests pass (CLI help matrix)
- ✅ `go test ./...` passes (all unit tests)
- ✅ Code compiles without errors
- ✅ Enforcement is opt-in (default: false)
- ✅ Clear remediation messages provided
- ✅ Exit code semantics preserved

**Release**: v0.3.1 (backward compatible MINOR version bump)

---

### Enhanced - Production Demo with Real CLI Output and Rich Colors

**Summary**: Enhanced interactive demo with accurate CLI output, vibrant colors for visual clarity, and improved user education through `acc --help` and `ls -al` commands.

**What's Enhanced:**

- ✅ **Real acc --help output** - Shows all 16 actual commands (attest, build, completion, config, help, init, inspect, login, policy, promote, push, run, trust, upgrade, verify, version) and complete Flags section (--color, --config, -h, --json, --no-emoji, --policy-pack, -q)
- ✅ **franck@csengineering$ prompt** - Professional cyan-colored prompt instead of generic `csengineering$`
- ✅ **ls -al after init** - Shows created files (.acc/, acc.yaml, policies) to close the loop
- ✅ **Vibrant color scheme** - Bold green for success ("pass", true, EXIT=0), bold red for failures ("fail", violations), bright yellow for warnings, cyan for metadata

**Visual Color Indicators:**

- 🟢 **Green** (bold): "pass" status, true values, EXIT=0, success messages
- 🔴 **Red** (bold): "fail" status, violation names, error messages
- 🟡 **Yellow** (bright): Violation rule names (warning state)
- 🔵 **Cyan** (bright): Counts, project names, metadata

**Educational Improvements:**

1. `acc --help` as first command - Shows users all available CLI commands
2. `ls -al` after `acc init` - Demonstrates what files are actually created
3. Color-coded outputs - Users instantly see success (green) vs failure (red)
4. Real CLI output - Exact match with production `acc` behavior

**Duration**: ~45 seconds
**Terminal**: 100×28
**Commands**: 11 total (added --help and ls -al to original 9)

**Files Modified:**

- `site/public/demo/demo.cast` - Updated with real acc --help output, franck@csengineering$ prompt, and vibrant colors for all outputs
- `demo/demo-script.sh` - Enhanced to use franck@csengineering$ prompt and include acc --help + ls -al

**User Experience**: The demo is now highly readable and visually intuitive. Users can parse success/failure states at a glance and see exactly what commands are available and what files get created.

---

### Fixed - Production Demo Scripts (Exact 9-Command Sequence)

**Summary**: Updated core demo scripts (demo/run.sh, demo/demo-script.sh, demo/record.sh) to implement the exact 9-command production sequence with correct prompt and terminal dimensions.

**What's Fixed:**

- ✅ **demo/run.sh** - Complete production validator with all 9 commands, exit code assertions (PASS=0, FAIL=1), JSON schema validation, and self-validation (exits 1 on any assertion failure)
- ✅ **demo/demo-script.sh** - Updated to execute EXACT 9-command sequence with `csengineering$` prompt (cyan colored), proper pauses, and full trust cycle
- ✅ **demo/record.sh** - Fixed terminal dimensions (100×28 rows, was 30), runs preflight validation via demo/run.sh before recording

**Key Improvements:**

1. **Exact 9 commands**: version → init → build PASS → verify PASS + jq → exit code → build FAIL → verify FAIL + jq → policy explain → full trust cycle (verify → attest → trust status)
2. **Contract compliance**: Validates exit codes per Testing Contract v0.3.0 (PASS=0, FAIL=1)
3. **JSON schema validation**: Asserts `.status`, `.sbomPresent`, `.policyResult.violations[0].rule`, `.attestations|length`
4. **Colored prompt**: `csengineering$` in cyan (ANSI escape codes)
5. **Self-validating**: demo/run.sh fails if acc behavior regresses
6. **Duration**: ~60-85 seconds with readable pauses

**Files Modified:**

- `demo/run.sh` - Production validator (193 lines, validates all 9 commands with assertions)
- `demo/demo-script.sh` - Recording script (105 lines, executes exact 9 commands)
- `demo/record.sh` - Recording orchestration (fixed terminal size to 28 rows)

**Testing**: Run `bash demo/run.sh` to validate all 9 commands work correctly. Exits 0 on success, 1 on any assertion failure.

---

### Added - Production Interactive Demo v2

**Summary**: Production-quality 9-command demo with `csengineering$` prompt proving acc's value in 60-90 seconds. Deterministic, reproducible, and ready for CI/CD.

**What's New:**

- ✅ **Exactly 9 commands** - Follows precise storyline: version → init → build PASS → verify PASS → exit code → build FAIL → verify FAIL → explain → attest
- ✅ **csengineering$ prompt** - Colored cyan prompt (not generic `$`)
- ✅ **60-90 second duration** - Timed with readable pauses between commands
- ✅ **Full CI/CD cycle** - Shows PASS (exit 0), FAIL (exit 1), explainability, and attestation
- ✅ **Deterministic validation** - Validates all 9 commands work correctly with exit codes + JSON schema
- ✅ **Production scripts** - Recording, validation, and deployment infrastructure

**The 9 Commands:**

1. `acc version` - Prove versioned, deterministic tool
2. `acc init demo-project` - Create policy baseline
3. `acc build demo-app:ok` - Build PASSING workload + SBOM (non-root user)
4. `acc verify --json demo-app:ok | jq '.status, .sbomPresent'` - Verify PASS, show JSON fields
5. `echo $?` - Prove exit code 0 (CI gate PASS)
6. `acc build demo-app:root` - Build FAILING workload (runs as root)
7. `acc verify demo-app:root` - Verify FAIL (exit 1, CI gate blocks)
8. `acc policy explain --json | jq ...` - Explainable violation (no-root-user)
9. `acc attest demo-app:ok` - Create attestation after re-verifying PASS

**Files Added:**

- `demo/demo-script-v2.sh` - The 9 commands for asciinema recording
- `demo/run-v2.sh` - Validation script (tests all 9 commands)
- `demo/record-v2.sh` - Recording orchestration with asciinema
- `demo/deploy-to-site.sh` - Easy deployment to website
- `demo/README-v2.md` - Comprehensive usage guide
- `demo/IMPLEMENTATION-SUMMARY.md` - Complete deliverables + specifications

**Files Modified:**

- `site/public/demo/demo.cast` - Deployed improved demo (219 lines vs 45-line placeholder)

**Demo Message:** "acc is a policy verification CLI that turns cloud controls into deterministic, explainable results for CI/CD gates."

---

### Fixed - Interactive Demo CSP

**Summary**: Fixed Content Security Policy blocking asciinema player CDN, enabling the interactive demo to load and play on the website.

**What's Fixed:**

- ✅ **CSP allowlist** - Added `cdn.jsdelivr.net` to `script-src` and `style-src` directives in `site/next.config.js`
- ✅ **DemoPlayer rendering** - Interactive terminal demo now loads and auto-plays on homepage
- ✅ **External resources** - asciinema-player library (v3.7.0) now loads from CDN without CSP violations

**Files Modified:**

- `site/next.config.js` - Updated Content-Security-Policy header to allow cdn.jsdelivr.net

---

### Added - Interactive Demo Infrastructure

**Summary**: New deterministic, reproducible interactive demo using asciinema for showcasing acc's policy verification workflow. Includes validation scripts, CI automation, and web integration.

**What's New:**

- ✅ **Demo validation script** (`demo/run.sh`) - Validates 8 scenarios covering v0.2.7 and v0.3.0 features
- ✅ **Recording infrastructure** (`demo/record.sh`, `demo/demo-script.sh`) - Orchestrates asciinema recordings with preflight validation
- ✅ **CI/CD automation** (`.github/workflows/demo.yml`) - Runs demo validation on PRs and uploads recordings on releases
- ✅ **Web integration** - asciinema-player component for Next.js website with dual source support (asciinema.org + local files)
- ✅ **Demo page fix** (`docs/demo/index.html`) - Fixed auto-play for htmlpreview.github.io viewing with intelligent URL detection

**Demo Features:**

- Deterministic and reproducible workflow (60-90 seconds)
- Shows full acc lifecycle: init, verify (PASS/FAIL), explain, attest, trust status
- Contract compliant with exit codes and JSON output validation
- Works both locally and through GitHub Pages/htmlpreview

**Files Added:**

- `demo/run.sh` - Comprehensive validation with 8 test scenarios
- `demo/record.sh` - Recording orchestration with preflight checks
- `demo/demo-script.sh` - Demo commands for asciinema
- `demo/Dockerfile.ok`, `demo/Dockerfile.root` - Test images (passing/failing)
- `demo/README.md` - Complete documentation
- `.github/workflows/demo.yml` - CI validation workflow
- `site/components/DemoPlayer.tsx` - React component for asciinema-player
- `site/.env.local.example` - Environment configuration template
- `site/public/demo/demo.cast` - Placeholder recording

**Files Modified:**

- `docs/demo/index.html` - Fixed asciinema player for htmlpreview.github.io
- `site/app/page.tsx` - Integrated DemoPlayer component
- `site/public/demo/README.md` - Updated for asciinema approach

---

### Added - Attestation Verification (v0.3.0)

**Summary**: New `acc trust verify` command for local-only, read-only attestation verification. Validates that attestations exist and are valid for a given image.

**What's New:**

- ✅ **New command: `acc trust verify`** - Verify attestations exist and are valid (local-only, read-only)
- ✅ **Exit code contract** - 0=verified, 1=unverified, 2=unknown (cannot resolve digest)
- ✅ **JSON schema v0.3** - Deterministic output with all required fields always present
- ✅ **Schema validation** - Checks attestation JSON has required fields
- ✅ **Digest matching** - Validates attestation subject digest matches image digest
- ✅ **Comprehensive tests** - 3 E2E test cases + 7 unit tests for contract compliance

**Usage:**

```bash
# Verify attestations for an image (with attestations)
acc trust verify demo-app:ok
# Exit 0 (verified)

# Verify unattested image
acc trust verify demo-app:never-verified
# Exit 1 (unverified - no attestations)

# Verify non-existent image
acc trust verify nonexistent:image
# Exit 2 (unknown - cannot resolve digest)

# JSON output
acc trust verify --json demo-app:ok
{
  "schemaVersion": "v0.3",
  "imageRef": "demo-app:ok",
  "imageDigest": "abc123def456",
  "verificationStatus": "verified",
  "attestationCount": 1,
  "attestations": [
    {
      "path": ".acc/attestations/abc123/20250122-120000-attestation.json",
      "timestamp": "2025-01-22T12:00:00Z",
      "verificationStatus": "pass",
      "verificationResultsHash": "sha256:...",
      "validSchema": true,
      "digestMatch": true
    }
  ],
  "errors": []
}
```

**Contract Guarantees:**

- **Local-only**: No network or registry access required
- **Read-only**: No state mutation, no file modifications
- **Deterministic JSON**: All fields always present (arrays never null)
- **Schema validation**: Checks required fields (schemaVersion, timestamp, subject, evidence)
- **Digest matching**: Validates subject.imageDigest matches resolved image digest

**Files Changed:**

- `internal/trust/verify.go` - Core attestation verification logic (~230 lines)
- `internal/trust/verify_test.go` - Unit tests for exit codes and JSON contract (~250 lines)
- `cmd/acc/main.go` - Wire up `acc trust verify` subcommand
- `scripts/e2e_smoke.sh` - Add Test 9 with 3 verification scenarios
- `scripts/cli_help_matrix.sh` - Add trust verify to Tier 0 tests
- `docs/testing-contract.md` - Document v0.3.0 contract and guarantees

**Breaking Changes:** None - v0.3.0 is a backward-compatible minor version bump.

**Future Work (Out of Scope):**
- Cryptographic signature verification
- Registry attestation fetch
- Policy enforcement (blocking acc run if unverified)
- Attestation expiry checks

---

### Enhanced - Trust + Attestation Improvements (v0.2.7)

**Summary**: Strengthened `acc trust status` and `acc attest` as production-ready features with deterministic JSON output, per-image attestation isolation, and comprehensive test coverage.

**What Changed:**

- ✅ **Deterministic JSON schema** - Trust status returns stable, predictable JSON with all required fields always present
- ✅ **Per-image attestation isolation** - Attestations scoped to specific image digests prevent cross-image leakage
- ✅ **Exit code documentation** - Clarified that trust status exit codes remain unchanged (0=pass, 1=fail/warn, 2=unknown)
- ✅ **Enhanced test coverage** - Added 13 new tests (9 unit, 4 golden) for trust and attest features
- ✅ **E2E test improvements** - Strengthened smoke tests validate trust/attest integration and per-image isolation
- ✅ **Documentation updates** - Testing contract, README, and examples now reflect v0.2.7 behavior

**Trust Status Improvements:**

```bash
# Deterministic JSON output with all required fields
acc trust status --json demo-app:latest
{
  "schemaVersion": "v0.2",
  "imageRef": "demo-app:latest",
  "status": "pass",
  "sbomPresent": true,        # Always boolean, never null
  "violations": [],            # Always array, never null
  "warnings": [],              # Always array, never null
  "attestations": [            # Per-image attestations only
    ".acc/attestations/abc123456789/20250122-100000-attestation.json"
  ],
  "timestamp": "2025-01-22T10:00:00Z"
}

# Exit codes (PRESERVED - unchanged):
# 0 = pass
# 1 = fail or warn
# 2 = unknown (cannot compute)
```

**Attestation Improvements:**

- **Digest-based matching** - Uses image digest comparison (not tag strings) for safety
- **Per-image storage** - Attestations stored in `.acc/attestations/<digest-prefix>/`
- **Trust integration** - Attestations appear in `acc trust status` for that specific image only

**Testing Contract Updates:**

- Trust status JSON schema fully specified with required/optional fields
- Attestation safety contract documented (digest matching, mismatch handling)
- Per-image isolation guarantees enforced by tests
- Exit code contract clarified (0 vs 2 distinction)

**New Tests:**

Unit tests (9):
- `TestStatusResultExitCode` - Exit code logic
- `TestStatusUnknown` - Unknown status handling
- `TestStatusWithVerifyState` - State loading
- `TestStatusWithViolations` - Violation extraction
- `TestFindAttestationsForImage` - Per-image discovery
- `TestStatusJSONSchema` - JSON structure
- `TestGetString` - Helper function
- Plus 2 more supporting tests

Golden tests (4):
- `TestTrustStatusJSONGolden` - JSON output validation
- `TestTrustStatusJSONFieldOrdering` - Deterministic ordering
- `TestTrustStatusJSONSchemaVersion` - Schema version stability
- `TestTrustStatusJSONSchemaDrift` - Schema change detection

E2E enhancements:
- Trust status JSON schema validation (8 required fields)
- Per-image attestation isolation checks
- Exit code verification (0=pass, 1=fail/warn, 2=unknown)
- Attestation presence validation after `acc attest`

**Backward Compatibility:**

- ✅ **100% backward compatible** - All v0.2.x behavior preserved
- ✅ **No breaking changes** - New fields added, existing fields unchanged
- ✅ **Exit codes unchanged** - Trust status exit codes remain 0=pass, 1=fail/warn, 2=unknown
- ✅ **JSON schema** - v0.2 schema version unchanged

**Files Modified:**

- `internal/trust/status.go` - Enhanced per-image attestation discovery, deterministic JSON output
- `internal/trust/status_test.go` - Added 9 unit tests
- `internal/trust/status_golden_test.go` - Added 4 golden tests
- `testdata/golden/trust/*.json` - Added 3 golden test files
- `scripts/e2e_smoke.sh` - Enhanced trust/attest test coverage
- `docs/testing-contract.md` - Updated trust/attest contracts for v0.2.7
- `README.md` - Updated trust status and attest examples

---

### Added - Supply-Chain Verification for acc upgrade

#### Optional Cosign Signature & SLSA Provenance Verification (v0.2.7)

**Summary**: Added enterprise-grade supply-chain verification to `acc upgrade` with optional cosign signature verification and SLSA provenance validation.

**What's New:**

- ✅ **Cosign signature verification** - `--verify-signature` flag for cryptographic signature verification
- ✅ **SLSA provenance verification** - `--verify-provenance` flag for build provenance validation
- ✅ **Combined enterprise mode** - Use both flags together for maximum security
- ✅ **100% opt-in** - Default upgrade behavior unchanged (checksum-only)
- ✅ **Clear error messages** - Actionable errors when tools are missing
- ✅ **19 new tests** - Comprehensive test coverage for all verification modes

**New Flags:**

```bash
acc upgrade --verify-signature                    # Verify cosign signature
acc upgrade --verify-signature --cosign-key PATH  # With specific key
acc upgrade --verify-provenance                    # Verify SLSA provenance
acc upgrade --verify-signature --verify-provenance # Both (enterprise mode)
```

**Implementation:**

- `verifyCosignSignature()` - Downloads `.sig` files and runs `cosign verify-blob`
- `verifySLSAProvenance()` - Fetches `.intoto.jsonl` and validates structure
- `findCosignBinary()` - Checks for cosign in PATH
- New fields in `UpgradeOptions`: `VerifySignature`, `CosignKey`, `VerifyProvenance`
- New fields in `UpgradeResult`: `SignatureVerified`, `ProvenanceVerified`

**Verification Flow:**

1. Download release asset (unchanged)
2. Verify SHA256 checksum (unchanged)
3. **[NEW]** Optional: Verify cosign signature if `--verify-signature`
4. **[NEW]** Optional: Verify SLSA provenance if `--verify-provenance`
5. Install binary (unchanged)

**Provenance Checks:**

- ✓ Valid SLSA predicate type (contains "slsa" or "provenance")
- ✓ Builder identity is GitHub Actions
- ✓ Build type is GitHub Actions workflow
- ✓ Source repository is `cloudcwfranck/acc`

**Supported Provenance Formats:**

- `provenance.intoto.jsonl` (global)
- `<tag>.intoto.jsonl` (per-release)
- `<assetName>.intoto.jsonl` (per-asset)

**Requirements:**

- **Cosign verification**: Requires `cosign` in PATH (optional, only if `--verify-signature` used)
- **Provenance verification**: No additional dependencies (pure Go)
- **Default upgrade**: No new requirements (backward compatible)

**Error Handling:**

```bash
# Cosign not installed
$ acc upgrade --verify-signature
Error: cosign is required for signature verification but was not found in PATH.
Install cosign: https://docs.sigstore.dev/cosign/installation/

# Provenance missing
$ acc upgrade --verify-provenance
Error: no SLSA provenance found for this release
(tried: provenance.intoto.jsonl, v0.2.7.intoto.jsonl, acc_0.2.7_linux_amd64.tar.gz.intoto.jsonl)
```

**Output with Verification:**

```
Current version: v0.2.6
Target version:  v0.2.7
Asset:           acc_0.2.7_linux_amd64.tar.gz
Checksum:        a1b2c3d4e5f6...
Signature:       ✓ Verified
Provenance:      ✓ Verified
Installed to:    /usr/local/bin/acc

Successfully upgraded from v0.2.6 to v0.2.7
```

**JSON Output:**

```json
{
  "currentVersion": "v0.2.6",
  "targetVersion": "v0.2.7",
  "updated": true,
  "signatureVerified": true,
  "provenanceVerified": true,
  "installPath": "/usr/local/bin/acc"
}
```

**Tests (19 new, all passing):**

- `TestUpgradeDefaultBehaviorUnchanged` - Verifies no behavior change without flags
- `TestVerifySignatureRequiresCosign` - Cosign missing → actionable error
- `TestVerifyProvenanceMissing` - Provenance missing → clear error
- `TestVerifyProvenanceSuccess` - Valid provenance → success
- `TestVerifyProvenanceInvalidJSON` - Invalid JSON → proper error
- `TestVerifyProvenanceInvalidPredicateType` - Wrong predicate → error
- `TestVerifyProvenanceNonGitHubBuilder` - Non-GitHub builder → error
- `TestUpgradeWithBothVerifications` - Combined mode works
- `TestFindCosignBinary` - Binary detection works

**Files Modified:**

- `internal/upgrade/upgrade.go` (+204 lines)
- `cmd/acc/main.go` (+57 lines)
- `internal/upgrade/upgrade_test.go` (+340 lines)
- `README.md` (+146 lines)

**Documentation:**

- New "Enterprise-Grade Verification" section in README
- Cosign signature verification guide
- SLSA provenance verification guide
- Combined enterprise mode examples
- Release asset conventions

**Backward Compatibility:**

- ✅ Default `acc upgrade` behavior unchanged (checksum-only)
- ✅ No new dependencies required
- ✅ All existing tests pass (100% compatibility)
- ✅ Verification is 100% opt-in via flags
- ✅ No breaking changes to any commands

**No Product Changes:** This release contains NO changes to acc core functionality. All changes are opt-in verification features for the `upgrade` command only.

### Added - Website & Documentation

#### Official Website Launch (Vercel + GitHub Releases Backend)

**Summary**: Launched official acc website with auto-updating release information, deployed on Vercel with GitHub Releases as the backend.

**What's New:**

- ✅ **Official website** - Next.js 14 website deployed on Vercel at `site/`
- ✅ **Auto-updating releases** - ISR (Incremental Static Regeneration) + deploy hooks for real-time updates
- ✅ **GitHub Releases backend** - No separate server, uses GitHub API as data source
- ✅ **Download page** - Platform detection, checksums, grouped downloads by OS/arch
- ✅ **Documentation pages** - Quick start guide, how-to documentation
- ✅ **Release history** - Recent releases with changelogs
- ✅ **Security headers** - CSP, X-Frame-Options, X-Content-Type-Options, Referrer-Policy
- ✅ **Deploy automation** - GitHub Actions workflow triggers Vercel deployment on release

**Website Architecture:**

- **Frontend**: Next.js 14 (App Router, TypeScript, React Server Components)
- **Backend**: GitHub Releases API + repository content (unauthenticated)
- **Hosting**: Vercel
- **Auto-updates**: ISR with 5-minute revalidation + deploy hooks on release publish
- **Pages**: Homepage, Download, Docs, Releases
- **Security**: Strict CSP, security headers, no inline scripts, HTTPS-only

**Auto-Update Mechanism:**

1. **ISR (Incremental Static Regeneration)**:
   - Release data fetched from GitHub API with `revalidate: 300` (5 minutes)
   - Vercel automatically regenerates pages when data changes
   - Fallback if deploy hook not configured

2. **Deploy Hook on Release**:
   - GitHub Actions workflow `.github/workflows/site-deploy.yml` runs on release publish
   - Triggers Vercel deployment via webhook
   - Site rebuilds immediately with new release

**Files Added:**

- `site/package.json` - Next.js dependencies and scripts
- `site/tsconfig.json` - TypeScript configuration
- `site/next.config.js` - Security headers and build configuration
- `site/lib/github.ts` - GitHub API helpers with ISR
- `site/app/layout.tsx` - Root layout with metadata
- `site/app/page.tsx` - Homepage (hero, features, how it works)
- `site/app/download/page.tsx` - Download page with platform detection
- `site/app/docs/page.tsx` - Quick start documentation
- `site/app/releases/page.tsx` - Release history listing
- `site/components/Navigation.tsx` - Sticky header navigation
- `site/components/Footer.tsx` - Footer with links
- `site/components/*.module.css` - Component-scoped styles
- `site/app/globals.css` - Global styles with dark/light mode
- `site/public/demo/` - Demo asset placeholder
- `.github/workflows/site-deploy.yml` - Vercel deploy hook trigger
- `.github/workflows/site-ci.yml` - Site-only CI workflow (lint, typecheck, build)
- `site/README.md` - Website setup and deployment documentation
- `docs/website.md` - Website architecture documentation

**Release Focus:**

This release is a **website and documentation release** that:
- ✅ Launches official acc website on Vercel
- ✅ Provides auto-updating download and release pages
- ✅ Documents website architecture and deployment process
- ✅ Demonstrates production-grade Next.js + Vercel deployment

**No Product Changes:** This release contains NO changes to acc product behavior, CLI semantics, JSON schemas, or exit codes. All changes are website infrastructure and documentation.

**No Test Changes:** This release contains NO changes to test scripts or CI workflows for the CLI tool. All test infrastructure remains unchanged.

#### Enterprise Website Enhancements - Production-Grade Features

**Summary**: Enhanced website with enterprise-grade operational features including stable-by-default downloads, pre-release support, operational health monitoring, and comprehensive testing.

**What's New:**

- ✅ **Stable-by-default downloads** - Download page shows latest stable release by default (prerelease=false, draft=false)
- ✅ **Pre-release toggle** - Optional "Include pre-releases" checkbox with localStorage persistence
- ✅ **Pre-release banner** - Site-wide warning when pre-release is newer than stable
- ✅ **Operational health monitoring** - `/api/health` endpoint + `/status` dashboard for service health
- ✅ **Auto-update optimization** - Reduced ISR interval from 300s to 60s for faster updates
- ✅ **Enhanced release selection** - Intelligent logic to show stable or prerelease based on user preference
- ✅ **Warning indicators** - Clear pre-release warnings: "Not recommended for production use"
- ✅ **Status indicator** - Footer shows pulsing green dot with link to status page
- ✅ **Comprehensive testing** - Jest test suite with 27 test cases for release logic

**Website Architecture Updates:**

- **ISR Interval**: Changed from 300s (5 min) to 60s (1 min) for faster release updates
- **Release Selection**: Dual-mode (stable/prerelease) with automatic date comparison
- **Health Monitoring**: Server-side health checks with 60s caching to prevent API hammering
- **Client Preferences**: localStorage for prerelease toggle + URL parameter support (?prerelease=1)
- **State Persistence**: Banner dismissal per-version in localStorage

**New API Endpoints:**

- `GET /api/health` - Returns JSON with GitHub API health, rate limits, release status
- `GET /api/github/releases` - Server-side releases API with ISR caching

**New Pages:**

- `/status` - Real-time operational dashboard with auto-refresh (60s interval)

**New Components:**

- `PrereleaseBanner.tsx` - Dismissible warning banner for pre-releases
- `PrereleaseBannerWrapper.tsx` - Server component for banner conditional rendering

**GitHub API Library Updates:**

- Added `getLatestStableRelease()` - Filters for !prerelease && !draft
- Added `getLatestPrerelease()` - Finds first prerelease
- Added `isPrereleaseNewer()` - Compares release dates
- Added `prerelease` and `draft` fields to GitHubRelease interface

**Enhanced Downloads Page:**

- Completely rewritten as client component with state management
- Release selector with stable/prerelease badges (green/yellow)
- Toggle for "Include pre-releases" with persistence
- Clear warning messages for pre-release versions
- Info messages when older pre-releases exist
- URL parameter support for sharing pre-release links

**Health Monitoring Features:**

- **Status meanings**: ok (all good), degraded (issues detected), down (GitHub unreachable)
- **Metrics tracked**: GitHub reachability, rate limit remaining, latest stable/prerelease tags, assets validation, checksums presence
- **Caching**: 60-second server-side cache to prevent excessive API requests
- **Troubleshooting**: Context-aware guidance based on detected issues
- **Auto-refresh**: Status page updates every 60 seconds automatically

**Testing Infrastructure:**

- `__tests__/github.test.ts` - 27 comprehensive tests
- `jest.config.js` - Next.js Jest configuration
- `jest.setup.js` - Testing environment setup
- Test coverage: release parsing, asset info, OS/arch display names, stable vs prerelease selection, draft filtering

**Files Added:**

- `site/app/api/health/route.ts` - Health check endpoint
- `site/app/api/github/releases/route.ts` - Releases API route
- `site/app/status/page.tsx` - Status dashboard page
- `site/app/status/status.module.css` - Status page styles
- `site/components/PrereleaseBanner.tsx` - Pre-release banner component
- `site/components/PrereleaseBanner.module.css` - Banner styles
- `site/components/PrereleaseBannerWrapper.tsx` - Server wrapper for banner
- `site/__tests__/github.test.ts` - Test suite
- `site/jest.config.js` - Jest configuration
- `site/jest.setup.js` - Jest setup

**Files Modified:**

- `site/lib/github.ts` - Added stable/prerelease helper functions
- `site/app/download/page.tsx` - Complete rewrite with prerelease toggle
- `site/app/download/download.module.css` - Added release selector, badge, toggle, warning styles
- `site/components/Footer.tsx` - Added status link with pulsing indicator
- `site/components/Footer.module.css` - Added status link and pulse animation
- `site/app/layout.tsx` - Added PrereleaseBannerWrapper
- `site/package.json` - Added test scripts and dependencies (@testing-library/jest-dom, jest, etc.)
- `site/README.md` - Extensive documentation updates (Enterprise Features, Operations, Testing)

**Enterprise Features Documentation:**

- Stable vs Pre-Release Selection behavior
- Operational Health Monitoring details
- Auto-Update Strategy (ISR + Deploy Hooks)
- Pre-Release Banner functionality
- Testing instructions and coverage
- Operations guide (monitoring, common tasks)
- Environment variables reference table

**Pre-Release Support:**

- **Default behavior**: Always shows latest stable release (prerelease=false, draft=false)
- **Opt-in prereleases**: User enables "Include pre-releases" toggle
- **Date comparison**: Only shows prerelease if it's newer than stable
- **Clear labeling**: STABLE (green) vs PRE-RELEASE (yellow) badges
- **Warnings**: "⚠️ This is a pre-release version. Not recommended for production use."
- **Persistence**: Choice saved to localStorage + URL param support
- **Banner**: Site-wide dismissible banner when prerelease is newer than stable

**Operational Monitoring:**

- **Health endpoint**: `/api/health` with ok/degraded/down status
- **Status dashboard**: `/status` page with real-time metrics
- **Rate limit tracking**: Shows remaining GitHub API requests
- **Asset validation**: Verifies binaries and checksums exist
- **Troubleshooting**: Context-aware guidance for issues
- **Footer indicator**: Pulsing green dot linking to status page

**Auto-Update Strategy (Belt + Suspenders):**

1. **ISR (Baseline)**: 60-second revalidation guarantees updates within 1 minute
2. **Deploy Hook (Immediate)**: GitHub Actions triggers Vercel on release:published
3. **Redundancy**: ISR ensures updates even if deploy hook fails

**Test Coverage:**

- Release parsing logic (stable vs prerelease filtering)
- Asset information parsing (OS/arch detection from filenames)
- Display name mapping (linux→Linux, darwin→macOS, amd64→x64)
- Pre-release selection logic (date comparison, toggle behavior)
- Draft release exclusion
- Integration scenarios (stable default, prerelease opt-in, draft filtering)

**Impact:**

- **Production-ready monitoring** - Health checks enable uptime tracking and alerting
- **Enterprise release management** - Stable-by-default with opt-in prerelease access
- **Developer experience** - Clear warnings and status indicators prevent confusion
- **Operational transparency** - Status page provides visibility into system health
- **Faster updates** - 60s ISR interval reduces delay for new releases
- **Comprehensive testing** - 27 tests ensure release logic correctness

**Release Focus:**

This release enhances the **website infrastructure** to enterprise-grade standards:
- ✅ Operational health monitoring with status dashboard
- ✅ Stable-by-default release selection with prerelease support
- ✅ Faster auto-updates (60s ISR interval)
- ✅ Comprehensive test coverage (27 tests)
- ✅ Clear user warnings and status indicators
- ✅ Production-ready operational features

**No Product Changes:** This release contains NO changes to acc product behavior, CLI semantics, JSON schemas, or exit codes. All changes are website infrastructure and operational features.

**No CLI Test Changes:** This release contains NO changes to CLI test scripts or CI workflows for the acc tool. Website tests are isolated under `site/__tests__/`.

#### Release Pipeline & Website Hardening

**Summary**: Added enterprise-grade release validation and comprehensive website testing to ensure release integrity and operational reliability.

**What's New:**

- ✅ **Automated release validation** - CI blocks incomplete releases before publication
- ✅ **Checksum verification required** - All platform archives must have valid SHA256 checksums
- ✅ **Download verification guidance** - Install snippet includes checksum verification commands
- ✅ **Website smoke tests** - Health endpoint, download page, and status page validation
- ✅ **Unit test execution** - 27 Jest tests run on every PR to validate release logic

**Release Pipeline Improvements:**

**Validation Job** (`.github/workflows/release.yml`):
- Added `validate` job that runs after `build` job
- Validates `checksums.txt` exists
- Ensures all 5 platform archives have checksums (Linux/macOS/Windows × AMD64/ARM64)
- Runs `sha256sum -c` to verify checksums match actual files
- Confirms minimum platform coverage
- **Blocks release publication** if validation fails
- Provides actionable error messages for maintainers

**Build Job Output:**
- Added `version` output to build job for artifact download coordination
- Enables validate job to download correct artifacts

**Website Download Hardening:**

**Install Snippet Enhancement** (`site/app/download/page.tsx`):
- Downloads `checksums.txt` in installation commands
- Shows SHA256 verification commands when checksums available:
  - `sha256sum -c checksums.txt --ignore-missing` (Linux)
  - `shasum -a 256 -c checksums.txt --ignore-missing` (macOS)
- Displays warning when checksums missing: `# ⚠️ Checksums not available for this release`
- Adapts dynamically based on release completeness

**Site CI Enhancements:**

**Unit Tests** (`.github/workflows/site-ci.yml`):
- Added test execution to `build-and-test` job
- Runs `npm test -- --ci --coverage --maxWorkers=2`
- Validates stable/prerelease selection logic (27 tests)
- Fails build if tests fail (`continue-on-error: false`)

**Smoke Tests Job**:
- Added new `smoke-tests` job that runs after build
- Starts production Next.js server
- Validates `/api/health` endpoint:
  - Checks JSON structure (status, github fields)
  - Validates status values (ok/degraded/down)
  - Ensures health endpoint responds correctly
- Tests download page loads with "Download" heading
- Tests status page loads successfully
- Graceful cleanup with server process management

**Documentation Updates:**

**site/README.md** - Added "Release Pipeline & Validation" section:
- Release artifact requirements checklist
- Automated validation process explanation
- Manual verification commands (`gh release view`, `sha256sum -c`)
- Website integration details (auto-updates, download verification, health monitoring)
- CI/CD workflow documentation
- Verifying release completeness instructions

**README.md** - Expanded "Website" section:
- Enterprise features overview (stable-by-default, prerelease support, checksum verification)
- "How It Stays Up-to-Date" subsection:
  - Dual update mechanism (deploy hooks + ISR) explanation
  - Result: Updates within 1 minute
- "Release Integrity" subsection:
  - Checksums required, automated validation
  - Download verification, health monitoring
  - User experience for complete vs incomplete releases
- Testing section (Jest + smoke tests)

**Files Changed:**

- `.github/workflows/release.yml` - Added validation job with checksum verification
- `.github/workflows/site-ci.yml` - Added unit tests + smoke tests job
- `site/app/download/page.tsx` - Enhanced install snippet with checksum verification
- `site/README.md` - Added release pipeline documentation
- `README.md` - Expanded website section with enterprise features

**What This Prevents:**

- ❌ Releases without checksums
- ❌ Corrupted archives with mismatched checksums
- ❌ Missing platform binaries
- ❌ Incomplete releases reaching users
- ❌ Website regressions in health endpoint or page loads

**Impact:**

- **Release quality gate** - CI automatically validates every release before publication
- **User security** - Download verification guidance in every install snippet
- **Operational confidence** - Automated smoke tests prevent broken deployments
- **Clear documentation** - Maintainers know how to verify releases manually
- **No manual steps** - Validation runs automatically on every release tag

**Release Focus:**

This enhancement adds **enterprise-grade quality gates** to the release pipeline:
- ✅ Automated release validation (blocks incomplete releases)
- ✅ Checksum verification (SHA256 for all archives)
- ✅ Download security guidance (verification commands in install snippet)
- ✅ Website testing (unit tests + smoke tests on every PR)
- ✅ Comprehensive documentation (pipeline validation + manual verification)

**No Product Changes:** This release contains NO changes to acc product behavior, CLI semantics, JSON schemas, or exit codes. All changes are pipeline validation, website hardening, and documentation.

**No CLI Test Changes:** This release contains NO changes to CLI test scripts. All testing enhancements are for website infrastructure under `site/` and release pipeline under `.github/workflows/`.

#### Semantic Version Sorting & Checksum Detection Fixes

**Summary**: Fixed critical bugs in release selection (semver sorting) and checksum detection to ensure deterministic, correct behavior across all scenarios.

**What's Fixed:**

- ✅ **Semantic version sorting** - v0.2.10 now correctly ranks higher than v0.2.9 (was alphabetical, now proper semver)
- ✅ **Comprehensive checksum detection** - Detects multiple formats: `checksums.txt`, `SHA256SUMS`, `checksums.sha256`, `sha256sums.txt`, `.sha256` files
- ✅ **Checksum mismatch bug** - Fixed bug where download page showed v0.2.5 but checked checksums from v0.2.6
- ✅ **Single source of truth** - All release logic unified in `computeReleaseSelection()` function
- ✅ **ESLint compliance** - Fixed `react/no-unescaped-entities` error in download page

**Bug Fixes:**

**Release Selection (Semver Sorting)**:
- **Before**: Releases sorted by GitHub API order (alphabetical) - v0.2.9 could appear after v0.2.10
- **After**: Proper semantic version comparison - v0.2.10 > v0.2.9 > v0.2.5
- Added `parseSemver()` and `compareSemver()` functions
- Stable versions (no prerelease suffix) rank higher than prereleases with same base version
- Comprehensive tests ensure v0.2.10 > v0.2.9 ordering

**Checksum Detection**:
- **Before**: Only detected `checksums.txt`, missed other common formats
- **After**: Prioritized detection of multiple formats:
  1. `checksums.txt` (highest priority)
  2. `SHA256SUMS`
  3. `checksums.sha256`
  4. `sha256sums.txt`
  5. Per-asset `.sha256` files (fallback)
- Added `detectChecksumAsset()` function with comprehensive tests

**Checksum Mismatch Bug**:
- **Before**: Download page could show stable v0.2.5 while fetching checksums from prerelease v0.2.6
- **After**: `computeReleaseSelection()` returns checksumAsset for selected release only
- Download page always uses `state.checksumAsset` (guaranteed to match selected release)
- Health endpoint uses same selection logic for consistency

**Single Source of Truth**:
- Created `/api/releases/selection` endpoint using `computeReleaseSelection()`
- Download page fetches state from API (no client-side logic)
- Health endpoint uses same function for release selection
- Status page uses same logic
- **Result**: Impossible to have version/checksum mismatch

**New Core Module** (`site/lib/releases.ts`):
```typescript
- parseSemver() - Parse version strings with v prefix handling
- compareSemver() - Proper semver comparison (major.minor.patch + prerelease)
- sortReleasesBySemver() - Sort releases by semver (not GitHub order)
- detectChecksumAsset() - Multi-format checksum detection
- computeReleaseSelection() - SINGLE SOURCE OF TRUTH for all release state
```

**Testing:**

- Added 17 new tests (44 total, all passing)
- `parseSemver` tests: standard/prerelease versions, invalid input
- `compareSemver` tests: **v0.2.10 > v0.2.9** (critical test), stable > prerelease
- `sortReleasesBySemver` tests: correct ordering across multiple versions
- `detectChecksumAsset` tests: all formats, prioritization, fallbacks
- `computeReleaseSelection` tests: stable default, prerelease toggle, draft filtering, checksum detection

**Files Added:**

- `site/lib/releases.ts` - Core semver and release selection logic (167 lines)
- `site/app/api/releases/selection/route.ts` - Server-side release selection API
- `site/__tests__/releases.test.ts` - Comprehensive test suite (334 lines, 17 new tests)

**Files Modified:**

- `site/lib/github.ts` - Added `getAuthHeaders()`, reduced ISR interval to 60s
- `site/app/download/page.tsx` - Rewritten to use `/api/releases/selection`, fixed ESLint quote escaping
- `site/app/api/health/route.ts` - Updated to use `computeReleaseSelection()` for consistency
- `site/README.md` - Documented semver sorting, checksum detection, single source of truth pattern

**Acceptance Criteria (All Met):**

- ✅ Toggle OFF → Shows v0.2.5 stable, checksums match v0.2.5
- ✅ Toggle ON → Shows v0.2.6 prerelease, checksums match v0.2.6
- ✅ No scenario where version shown != checksums checked
- ✅ v0.2.10 correctly appears after v0.2.9 in all views
- ✅ Status page reflects Degraded/Operational correctly based on checksum presence

**Impact:**

- **Correctness** - Release selection now deterministic and semver-compliant
- **Security** - Checksum verification always matches displayed version
- **Reliability** - Single source of truth prevents state inconsistencies
- **Test coverage** - 44 tests ensure no regressions
- **User trust** - Download page shows correct checksums for selected release

**Release Focus:**

This release fixes **critical bugs in release selection and checksum detection**:
- ✅ Semantic version sorting (v0.2.10 > v0.2.9)
- ✅ Multi-format checksum detection
- ✅ Version/checksum mismatch prevention
- ✅ Single source of truth architecture
- ✅ ESLint compliance for CI

**No Product Changes:** This release contains NO changes to acc product behavior, CLI semantics, JSON schemas, or exit codes. All changes are website bug fixes and test enhancements.

**No CLI Test Changes:** This release contains NO changes to CLI test scripts. All testing enhancements are for website infrastructure under `site/__tests__/`.

## [0.2.5] - 2025-12-20

### Fixed - Documentation Accuracy

#### Post-Release Verification & Documentation Refresh

**Summary**: Corrected stale documentation references and verified all test tiers pass reliably after v0.2.3/v0.2.4 regression fixes.

**What Changed:**
- Updated CONTRIBUTING.md to remove outdated "known regressions" warnings
- Removed stale references to Tier 1 test failures (regressions fixed in v0.2.3)
- Clarified that all test tiers (0, 1, Go unit tests) now pass reliably
- Updated pre-push checklist to reflect current expected behavior

**Documentation Corrections:**
- CONTRIBUTING.md line 124: Removed "Tier 1 tests currently FAIL" warning (outdated)
- CONTRIBUTING.md line 189: Removed "Expected to FAIL" comment (outdated)
- CONTRIBUTING.md line 198: Updated to state "MUST all pass" instead of "currently FAILS"

**Verification Performed:**
- ✅ All Go unit tests pass
- ✅ Tier 0 (CLI Help Matrix) passes
- ✅ Tier 1 (E2E Smoke Tests) passes (verified in CI)
- ✅ Documentation now matches implementation reality

**Files Changed:**
- `CONTRIBUTING.md` - Removed stale regression warnings, updated test expectations

**Release Focus:**

v0.2.5 is a **post-release documentation accuracy release** that:
- ✅ Confirms all regressions from v0.2.2 are fixed
- ✅ Updates contributor documentation to match reality
- ✅ Verifies test infrastructure stability
- ✅ Ensures clear guidance for new contributors

**No Product Changes:** This release contains NO changes to acc product behavior, CLI semantics, JSON schemas, or exit codes. All changes are documentation accuracy improvements.

**No Test Changes:** This release contains NO changes to test scripts or CI workflows. All test infrastructure remains unchanged from v0.2.4.

## [0.2.4] - 2025-12-20

### Fixed - Test Infrastructure & Documentation Quality

#### Test Script Quality - ShellCheck Compliance

**Summary**: Fixed all ShellCheck INFO warnings (SC2086, SC2317) in test scripts without changing acc behavior or test assertions.

**What Changed:**
- Test scripts now use array-based command invocation to prevent word-splitting issues
- Helper functions use `set +e / set -e` pattern for safe exit code capture
- All variable references properly quoted in command substitutions
- Zero ShellCheck warnings while maintaining identical test behavior

**Files Changed:**
- `scripts/cli_help_matrix.sh` - Array-based command execution for `test_help_command()` and `test_not_implemented()`
- `scripts/e2e_smoke.sh` - Safe exit code capture in `assert_success()` and `assert_failure()`
- `scripts/registry_integration.sh` - Quoted variable references
- `docs/testing-contract.md` - Documented script implementation patterns and ShellCheck compliance

**Technical Details:**

**SC2086 Fix (Word Splitting):**
```bash
# Before:
output=$($ACC_BIN $cmd_args 2>&1)

# After:
cmd=( "$ACC_BIN" )
cmd+=( $cmd_args )  # Intentional word splitting with disable comment
output=$("${cmd[@]}" 2>&1)
```

**SC2317 Fix (Unreachable Code):**
```bash
# Before (shellcheck thinks exit_code=$? is unreachable):
output=$("$@" 2>&1)
exit_code=$?

# After (explicit set +e):
set +e
output=$("$@" 2>&1)
exit_code=$?
set -e
```

**Documentation Updates:**
- Added "Test Script Implementation Patterns" section to testing-contract.md
- Documented why `|| true` pattern was replaced with `set +e / set -e`
- Clarified that `config` and `login` commands are help-only stubs
- Removed "Known Regressions" section (regressions fixed in v0.2.3)

**Impact:**
- Improved script maintainability and portability
- Better adherence to bash best practices
- No behavior changes to acc product or CI gates
- Zero impact on existing workflows

**Testing:** All scripts pass `bash -n` validation, Tier 0 tests GREEN

#### Documentation Updates

**Summary**: Updated README and documentation to reflect v0.2.4 release with accurate version references and CI testing information.

**What Changed:**
- Updated all version references in README.md from v0.1.0 to v0.2.4
- Added comprehensive "CI Test Tiers" section to README Development guide
- Documented how to run Tier 0, Tier 1, and Tier 2 tests locally
- Added reference to testing-contract.md for behavioral guarantees

**Files Changed:**
- `README.md` - Version references updated, CI test tier documentation added
- `docs/testing-contract.md` - Already updated with script implementation patterns

**Release Focus:**

v0.2.4 is a **quality and verification release** that demonstrates:
- ✅ All Tier 0 and Tier 1 CI tests pass reliably
- ✅ Zero ShellCheck warnings in test scripts
- ✅ Documentation matches implementation reality
- ✅ Enterprise-grade CI infrastructure with strict release gates
- ✅ Clear testing contract with exit code guarantees

**No Product Changes:** This release contains NO changes to acc product behavior, CLI semantics, JSON schemas, or exit codes. All changes are test infrastructure, documentation, and quality improvements.

## [0.2.3] - 2025-12-20

### Fixed - acc build CLI Regressions

**This release fixes critical bugs in acc build that broke backward compatibility and SBOM generation guarantees.**

**Critical Fixes:**

1. **Positional argument handling** - acc build now accepts image reference as positional argument
   - **Bug**: `acc build demo-app:ok` silently ignored positional argument, exited 0 without SBOM
   - **Root Cause**: CLI command didn't parse `args`, only accepted `--tag` flag
   - **Fix**: Accept both `acc build demo-app:ok` (positional) and `acc build --tag demo-app:ok` (flag)
   - **Impact**: Restores v0.1.x backward compatibility for automation scripts
   - **Code**: `cmd/acc/main.go:114-163` - NewBuildCmd with args support

2. **SBOM generation guarantee** - Build now verifies SBOM file exists or fails explicitly
   - **Bug**: acc build could exit successfully without creating SBOM file
   - **Root Cause**: No verification that SBOM file actually existed after syft command
   - **Fix**: Added explicit file existence check after generateSBOM
   - **Impact**: Build ALWAYS produces SBOM or fails with clear error
   - **Code**: `internal/build/build.go:81-84` - SBOM file verification

3. **Help text and examples** - Added usage examples to acc build --help
   - **Bug**: Help text lacked examples, unclear that --tag was accepted
   - **Fix**: Added examples showing both positional and flag usage
   - **Impact**: Users can discover correct syntax via --help
   - **Code**: `cmd/acc/main.go:121-126` - Example usage in help

**Positional Argument Behavior:**
```bash
# All these work now:
acc build demo-app:ok              # positional argument
acc build --tag demo-app:ok        # flag syntax
acc build -t demo-app:ok          # short flag

# If both provided, --tag takes precedence with warning
acc build demo-app:ok --tag other:latest  # Uses other:latest
```

**SBOM Guarantee:**
- If `acc build` exits 0, SBOM MUST exist in `.acc/sbom/`
- If SBOM can't be generated (syft missing, syft fails), build exits non-zero
- Build logs show explicit "Generating SBOM..." and success/failure

**Regression Tests Added:**
- `TestDetectBuildTool` - Verifies build tool detection with clear errors
- `TestBuild_SBOMVerification` - Verifies SBOM exists after successful build
- `TestGenerateSBOM_Contract` - Verifies generateSBOM creates file or errors
- `TestSBOMPath_Consistency` - Verifies SBOM path is predictable

**Files Changed:**
- `cmd/acc/main.go` - Accept positional args, add examples to help
- `internal/build/build.go` - Add SBOM file verification, improve logging
- `internal/build/build_test.go` - New file with 4 regression tests

**Breaking Changes:** None - changes restore v0.1.x compatibility

## [0.2.2] - 2025-12-20

### Fixed - Final Gate Consistency & SBOM Workflow

**This release implements a single authoritative final gate for verify decision consistency.**

**Critical Fixes:**

1. **Final gate consistency** - Implemented single authoritative decision gate
   - **Bug**: `status:"fail"` could occur while `PolicyResult.allow:true` due to early status assignments
   - **Root Cause**: Multiple places set `status = "fail"`, but final gate only set "fail" if still not failed
   - **Fix**: Single authoritative `finalAllow` variable that ALWAYS determines final status, overriding all earlier assignments
   - **Impact**: Guarantees `status` and exit code derive from `PolicyResult.Allow` (the final decision after profile filtering)
   - **Code**: `internal/verify/verify.go:255-286` - Authoritative final gate

2. **SBOM workflow guidance** - Improved error messages and documentation
   - **Bug**: SBOM-required error lacked actionable workflow guidance
   - **Fix**: Error message now includes step-by-step workflow:
     - Option 1: `docker build` → `syft` → `acc verify`
     - Option 2: `acc build` (automatic SBOM generation)
   - **Impact**: Users know exactly how to generate SBOMs
   - **Code**: `internal/verify/verify.go:75-97` - Enhanced error message

3. **README SBOM Workflows section** - Comprehensive workflow documentation
   - **Added**: Dedicated "SBOM Workflows" section with 3 workflows:
     - Workflow 1: `acc build` (recommended, automatic)
     - Workflow 2: `docker build` + manual SBOM generation
     - Workflow 3: CI/CD integration example
   - **Added**: SBOM troubleshooting guide
   - **Impact**: Clear documentation for all use cases

**Regression Tests Added:**
- `TestVerify_FinalGateConsistency` - Verifies status MUST match allow field
- `TestVerify_SBOMMissingErrorMessage` - Verifies error includes workflow guidance

**Design Principle:**
```go
// Single authoritative final gate (v0.2.2)
var finalAllow bool
if result.PolicyResult != nil {
    finalAllow = result.PolicyResult.Allow
} else {
    finalAllow = false
}

// Status ALWAYS derives from final gate
if finalAllow {
    result.Status = "pass"
} else {
    result.Status = "fail"
}
```

**Files Changed:**
- `internal/verify/verify.go` - Single authoritative final gate, improved SBOM error
- `internal/verify/verify_test.go` - Added regression tests, added strings import
- `README.md` - Added comprehensive SBOM Workflows section

**Breaking Changes:** None - all changes maintain backward compatibility

## [0.2.1] - 2025-12-20

### Fixed - v0.2.0 Regression Bugs

**This release fixes 5 critical bugs found in v0.2.0 testing.**

1. **verify status inconsistency** - Fixed verify returning `status:"fail"` when `allow:true` and no violations
   - **Bug**: verify checked violation count instead of `PolicyResult.Allow` field after profile filtering
   - **Fix**: Use `PolicyResult.Allow` as source of truth for status determination
   - **Impact**: Profiles now correctly set status to "pass" when all violations are filtered

2. **sbomPresent false after build** - Fixed SBOM detection after `acc build`
   - **Bug**: checkSBOMExists() required exact filename match, failed on name/format mismatches
   - **Fix**: Added fallback to detect ANY .json file in .acc/sbom/ directory
   - **Impact**: More robust SBOM detection across different configurations

3. **Profile loading broken** - Fixed profile name resolution and error messages
   - **Bug**: .acc/profiles/ directory not created by `acc init`, unclear error messages
   - **Fix**: `acc init` now creates .acc/profiles/, improved error messages with remediation
   - **Impact**: Better user experience when loading profiles by name

4. **trust status image leakage** - Fixed digest resolution to prevent state leakage across images
   - **Bug**: resolveImageDigest() was simplistic, couldn't query container runtimes
   - **Fix**: Properly query Docker/Podman/nerdctl for image digest
   - **Impact**: `acc trust status` now shows correct per-image state, not global state

5. **trust status exit codes wrong** - Fixed exit codes when no state found
   - **Bug**: Returned exit 1 (error) instead of exit 2 (no state) when verification state missing
   - **Fix**: Return StatusResult with status:"unknown" and exit code 2
   - **Impact**: Correct exit code behavior: 0=pass, 1=fail, 2=no state

**Regression Tests Added:**
- `TestVerify_StatusFromAllow` - Verifies status derives from allow field
- `TestCheckSBOMExists_Fallback` - Verifies SBOM fallback detection

**Files Changed:**
- `internal/verify/verify.go` - Fixed status logic, improved SBOM detection
- `internal/profile/profile.go` - Improved error messages
- `internal/config/init.go` - Create .acc/profiles/ directory
- `internal/trust/status.go` - Fixed digest resolution and exit codes
- `internal/verify/verify_test.go` - Added regression tests

## [0.2.0] - 2025-12-20

### Added - Policy Profiles & Trust Status

**This release introduces Policy Profiles, an opt-in configuration layer for post-evaluation violation filtering.**

**What's New:**

- ✅ **Policy Profiles** - YAML-based configuration for filtering violations by rule name or severity
- ✅ **`--profile` flag** - `acc verify --profile <name|path>` for profile-based enforcement
- ✅ **`acc trust status` command** - View verification state with profile and violation details
- ✅ **Post-evaluation filtering** - Profiles filter results AFTER OPA runs, not during
- ✅ **Warning display** - Convert ignored violations to warnings with `warnings.show: true`
- ✅ **Example profiles** - Baseline (dev) and strict (prod) profiles in `.acc/profiles/`
- ✅ **Full backward compatibility** - v0.1.x behavior unchanged when `--profile` not used

**Profile Schema (v1):**

```yaml
schemaVersion: 1
name: baseline
description: Baseline enforcement profile

# Only enforce these policies (allowlist)
policies:
  allow:
    - no-root-user
    - no-latest-tag

# Ignore these violations
violations:
  ignore:
    - informational  # By severity
    - low            # By severity
    - missing-healthcheck  # By rule name

# Warning display
warnings:
  show: true  # Display ignored violations as warnings
```

**Usage Examples:**

```bash
# Verify with profile (name lookup in .acc/profiles/)
acc verify myapp:latest --profile baseline

# Verify with profile (explicit path)
acc verify myapp:latest --profile ./custom.yaml

# View trust status with profile information
acc trust status myapp:latest

# Output:
# Status:         ✓ PASS
# Profile:        baseline
# Warnings (2 ignored):
#   [low] missing-healthcheck: Container lacks health check
```

**New Commands:**

| Command | Description |
|---------|-------------|
| `acc trust status [image]` | View trust status with profile and violation details |

**Trust Status Exit Codes:**

- `0` - Verified (pass)
- `1` - Not verified (fail)
- `2` - No verification state found

**Architecture:**

**Phase 1 - Core Profile Infrastructure:**
- `internal/profile/profile.go` - Profile types, YAML parsing, validation
- Schema version 1 enforcement with strict unknown field rejection
- Profile loading from name (`.acc/profiles/<name>.yaml`) or explicit path
- 12 unit tests

**Phase 2 - Decision Gating:**
- `internal/profile/resolver.go` - Post-evaluation violation filtering
- Allow list filtering (only enforce specified policies)
- Ignore list filtering (by severity or rule name)
- Warning categorization with `warnings.show` support
- 9 unit tests

**Phase 3 - CLI Integration:**
- `acc verify --profile <name|path>` flag
- Profile loading errors with clear remediation messages
- Backward compatibility: `nil` profile parameter for v0.1.x callers
- Warning output to stderr when `warnings.show: true`

**Phase 4 - Trust Status:**
- `acc trust status` command with digest-scoped state loading
- Displays profile used, active violations, and warnings separately
- JSON output with `--json` flag
- Reads from `.acc/state/verify/<digest>.json` or global state

**Phase 5 - Documentation:**
- Comprehensive README section with examples
- Example profiles: baseline (dev), strict (prod)
- Migration guide from v0.1.x
- CHANGELOG entry

**Non-Goals (Explicitly Not Implemented):**

- ❌ Profile auto-discovery or defaults
- ❌ Modifying OPA evaluation or Rego policies
- ❌ Pre-evaluation policy filtering
- ❌ Profile inheritance or composition
- ❌ Remote profile fetching
- ❌ Profile signing/verification

**Backward Compatibility:**

- **v0.1.x behavior preserved** - `acc verify` without `--profile` is identical to v0.1.8
- **JSON output unchanged** - Same JSON structure when profile not used
- **No breaking changes** - All existing commands and flags work unchanged
- **State format extended** - Added optional `profileUsed` field to verification state

**Migration:**

No changes required. Profiles are opt-in:

```bash
# Continue using v0.1.x behavior
acc verify myapp:latest

# Adopt profiles when ready
acc verify myapp:latest --profile baseline
```

**Testing:**

- ✅ 21 unit tests in `internal/profile/` (profile + resolver)
- ✅ Full backward compatibility verified
- ✅ Profile loading errors tested (missing file, invalid YAML, unknown fields)
- ✅ Violation filtering tested (allow list, ignore by severity/rule, warnings)

**Impact:**

- **Environment-specific enforcement** - Different profiles for dev/staging/prod
- **Gradual adoption** - Suppress low-priority violations without policy changes
- **Warning-driven workflows** - See violations without blocking
- **No migration burden** - Opt-in only, no forced changes

## [0.1.8] - 2025-12-19

### Added - Interactive Terminal Walkthrough

**This release adds a 60-second interactive demo for v0.1.8, the final release in the 0.1.x line.**

**What's New:**

- ✅ **Auto-playing terminal demo** - Asciinema-based recording at `docs/demo/index.html`
- ✅ **60-second walkthrough** - Shows init, verify (pass/fail), inspect, policy explain, and attest workflows
- ✅ **Professional presentation** - Real terminal recording with auto-play and loop
- ✅ **Embeddable** - Uses Asciinema player, works in any browser, auto-plays on load
- ✅ **README integration** - New Demo section with direct link to embedded player

**Demo Flow:**

1. Initialize acc project
2. Verify compliant image (PASS)
3. Inspect trust summary
4. Verify non-compliant image (FAIL - runs as root)
5. Explain policy violation with remediation
6. Create attestation for verified image

**Usage:**

Open `docs/demo/index.html` in any browser. The demo auto-plays on load using the Asciinema player.

**Impact:**

- **First-time users** get instant understanding of acc workflows
- **Documentation** includes visual demonstration alongside text
- **Onboarding** simplified with interactive preview before installation

**No Functional Changes:**

This is a documentation-only release. No changes to:
- acc commands or CLI behavior
- Policy evaluation logic
- Verification, attestation, or inspection code
- Any existing functionality

## [0.1.7] - 2025-12-19

### Fixed - Upgrade Backward Compatibility

**This release fixes backward compatibility for `acc upgrade` to support older release archive formats.**

**The Bug:**

`acc upgrade` failed when upgrading/downgrading to older versions due to inconsistent binary naming in release archives. Older releases used `acc-linux-amd64`, `acc-darwin-arm64` naming, while newer releases use `acc`.

**What's Fixed:**

- ✅ **Flexible binary detection** - Searches for executables matching `acc` or `acc-*` patterns
- ✅ **Exactly one enforcement** - Errors if 0 or multiple executables found in archive
- ✅ **Executable validation** - Verifies executable permissions on Unix systems
- ✅ **Better error messages** - Includes version context when `--version` flag is used
- ✅ **Legacy support** - Works with `acc-OS-ARCH` naming from v0.1.0-v0.1.5

**Testing:**

- ✅ Added `TestFindExecutableInDir_StandardName` - Current "acc" naming
- ✅ Added `TestFindExecutableInDir_LegacyName` - Legacy "acc-OS-ARCH" naming
- ✅ Added `TestFindExecutableInDir_MultipleExecutables` - Enforces exactly one
- ✅ Added `TestFindExecutableInDir_NoExecutable` - Handles missing binary
- ✅ Added `TestFindExecutableInDir_NonExecutableFile` - Unix permissions check
- ✅ Added `TestUpgradeWithLegacyArchive` - End-to-end legacy archive test
- ✅ All 16 upgrade package tests pass

**Security Guarantees Preserved:**

- SHA256 checksum verification unchanged
- Atomic binary replacement unchanged
- Backup/rollback mechanism unchanged
- No weakening of validation

**Example:**

```bash
# Now works with any version
acc upgrade --version v0.1.0  # Legacy naming
acc upgrade --version v0.1.6  # Current naming
acc upgrade                   # Latest (any format)
```

## [0.1.6] - 2025-12-19

### Added - Self-Update Capability

**This release adds `acc upgrade` for automatic self-updating with cryptographic verification.**

**What's New:**

- ✅ **Self-update command** - `acc upgrade` downloads and installs the latest stable release from GitHub
- ✅ **Version targeting** - `acc upgrade --version vX.Y.Z` installs specific versions
- ✅ **Dry-run mode** - `acc upgrade --dry-run` shows what would happen without downloading
- ✅ **SHA256 verification** - All downloads verified against checksums.txt before installation
- ✅ **Atomic replacement** - Unix systems use atomic rename with backup/rollback
- ✅ **Platform detection** - Automatic OS/ARCH detection (linux, darwin, windows × amd64, arm64)
- ✅ **Safe installation** - Backup created before replacement, restored on failure
- ✅ **Windows support** - Manual replacement instructions for file lock scenarios
- ✅ **Already-latest detection** - Skips download if current version matches latest

**Usage:**

```bash
# Upgrade to latest version
acc upgrade

# Upgrade to specific version
acc upgrade --version v0.1.7

# Show what would happen without installing
acc upgrade --dry-run

# JSON output for automation
acc upgrade --json
```

**Output Example:**

```
Current version: v0.1.5
Target version:  v0.1.6
Asset:           acc_0.1.6_linux_amd64.tar.gz
Checksum:        a1b2c3d4e5f6...
Installed to:    /usr/local/bin/acc

Successfully upgraded from v0.1.5 to v0.1.6
```

**Windows Behavior:**

On Windows, the running executable cannot be replaced directly due to file locking. The upgrade command downloads the new version to `acc.new.exe` and provides manual replacement instructions:

```
Windows binary downloaded to: C:\path\to\acc.new.exe

To complete upgrade:
1. Close this terminal
2. Rename acc.exe to acc.exe.old
3. Rename acc.new.exe to acc.exe
4. Delete acc.exe.old
```

**Security:**

- All releases downloaded from official GitHub repository (github.com/cloudcwfranck/acc)
- SHA256 checksums verified against signed checksums.txt
- Download failures abort installation (no partial updates)
- Checksum mismatches abort installation
- Backup/rollback on installation failure

### Impact

**Previous versions required manual installation:**
- Users had to download releases manually from GitHub
- No automatic checksum verification
- Risk of partial or corrupted downloads
- No built-in version targeting

**v0.1.6 provides seamless updates:**
- Single command to stay current
- Cryptographic verification of all downloads
- Safe atomic replacement with rollback
- Platform-specific handling for reliability

### Testing

- ✅ Added `TestSelectAsset` - Verifies asset name selection for linux/amd64, darwin/arm64, darwin/amd64, windows/amd64
- ✅ Added `TestNormalizeVersion` - Verifies version string normalization (with/without "v" prefix)
- ✅ Added `TestFetchRelease` - Verifies GitHub API integration with mock server
- ✅ Added `TestFetchReleaseNotFound` - Verifies 404 error handling
- ✅ Added `TestFetchChecksums` - Verifies checksum file parsing (ignores comments/blanks)
- ✅ Added `TestUpgradeAlreadyLatest` - Verifies already-latest returns Updated=false
- ✅ Added `TestUpgradeDryRun` - Verifies dry-run mode doesn't install
- ✅ Added `TestUpgradeAssetNotFound` - Verifies missing asset error
- ✅ Added `TestComputeSHA256` - Verifies checksum computation correctness
- ✅ Added `TestExtractTarGz` - Verifies archive extraction
- ✅ All tests use httptest mock servers (no real internet required)
- ✅ Environment variable overrides for testing (ACC_UPGRADE_API_BASE, ACC_UPGRADE_DOWNLOAD_BASE, ACC_UPGRADE_DISABLE_INSTALL)

## [0.1.5] - 2025-01-19

### Fixed - Attestation UX & Inspect State Correctness

**This release fixes UX and state correctness bugs discovered during v0.1.4 validation.**

**The Bugs:**

1. **Misleading attestation messaging**: `acc attest` printed "Creating attestation..." before validation, even when attestation failed due to missing state or image mismatch
2. **Incorrect inspect status**: `acc inspect <image>` showed the last global verification result instead of per-image status, causing Image A's status to be overwritten by Image B's verification

**What Was Broken:**

- `acc attest` printed "ℹ Creating attestation for demo-app:ok" before checking if verification state exists or matches the image
- When attestation failed validation, the creation message had already appeared, misleading users into thinking an attestation was created
- `acc inspect` loaded from `.acc/state/last_verify.json` (global), not per-image state
- Verifying Image A (PASS), then Image B (FAIL), then inspecting Image A would incorrectly show FAIL

**What's Fixed in v0.1.5:**

- ✅ **Attestation validation first** - All validation checks (state exists, image matches) run BEFORE printing creation message
- ✅ **Clear failure messages** - Failed validation prints errors without misleading "Creating..." message
- ✅ **Digest-scoped state** - Verification state now saved to both global and per-digest files (`.acc/state/verify/<digest>.json`)
- ✅ **Per-image inspect** - `acc inspect` loads digest-scoped state when available, falls back to global
- ✅ **Backward compatible** - Global `last_verify.json` still written for older tools/workflows

### Impact

**v0.1.4 had confusing UX:**
- Attestation printed "Creating..." even when it immediately failed
- Inspect showed wrong status for images after verifying another image

**v0.1.5 provides accurate UX:**
- Attestation only prints creation message after validation succeeds
- Inspect shows correct per-image verification status
- Each image maintains its own verification history

### Testing

- ✅ Added `TestAttest_NoCreationMessageOnFailure` - Verifies no creation message when validation fails
- ✅ Added `TestAttest_CreationMessageOnlyOnSuccess` - Verifies creation message only after validation
- ✅ Added `TestInspect_PerImageVerificationState` - Verifies per-image state loading
- ✅ Added `TestInspect_DoesNotLeakLastVerify` - Verifies no cross-contamination between images
- ✅ All existing tests pass on v0.1.5

## [0.1.4] - 2025-01-19

### Fixed - Panic Prevention & State Persistence

**CRITICAL: This release fixes release-blocking crashes and state persistence bugs discovered in v0.1.3 during Linux/WSL2 testing.**

**The Bugs:**

1. **Panic when OPA missing**: v0.1.3 panicked with nil pointer dereference when OPA was not installed
2. **Security bypass with escape hatch**: `ACC_ALLOW_NO_OPA=1` caused verification to PASS instead of FAIL
3. **Missing state persistence**: `acc policy explain` showed "no verification history" because state wasn't written on failure
4. **Nil result handling**: Main command didn't handle nil results defensively

**What Was Broken in v0.1.3:**

- `Verify()` returned `(nil, error)` when OPA missing, causing `result.ExitCode()` to panic on nil pointer
- `ACC_ALLOW_NO_OPA=1` returned empty violations, making verification pass (security bypass)
- State file wasn't written when `evaluatePolicy()` failed, breaking `acc policy explain`
- `FormatJSON()` and `ExitCode()` methods weren't nil-safe
- Exit code 2 (panic) instead of clean failure with exit code 1

**Example of Broken Behavior (v0.1.3):**
```bash
$ acc verify demo-app:latest  # OPA not installed
panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x1 addr=0x0 pc=0x...]
goroutine 1 [running]:
github.com/cloudcwfranck/acc/internal/verify.(*VerifyResult).ExitCode(...)
        internal/verify/verify.go:227
exit status 2
```

**v0.1.4 behavior (CORRECT):**
```bash
$ acc verify --json demo-app:latest  # OPA not installed
{
  "status": "fail",
  "policyResult": {
    "allow": false,
    "violations": [{
      "rule": "opa-required",
      "severity": "critical",
      "result": "fail",
      "message": "OPA not found. Policy evaluation requires OPA to be installed.\n\nInstall OPA: https://www.openpolicyagent.org/docs/latest/#running-opa"
    }]
  }
}
$ echo $?
1
```

**What's Fixed in v0.1.4:**

- ✅ **Zero panics** - `Verify()` ALWAYS returns valid `VerifyResult`, never nil
- ✅ **OPA missing creates violation** - Returns `opa-required` critical violation instead of error
- ✅ **Escape hatch fixed** - `ACC_ALLOW_NO_OPA=1` still creates violation (not a bypass), just allows tests to run
- ✅ **Nil-safe methods** - `ExitCode()` and `FormatJSON()` handle nil receiver defensively
- ✅ **State persistence on failure** - `saveVerifyState()` called even when policy evaluation fails
- ✅ **PolicyResult always initialized** - Initial `VerifyResult` includes non-nil `PolicyResult`
- ✅ **Main command defensive** - Handles nil result with exit code 2 and error message
- ✅ **Clean error messages** - OPA missing shows installation instructions, not panic trace

### Impact

**v0.1.3 MUST NOT be used** - Panics on systems without OPA installed, making acc unusable.

**v0.1.4 fixes all release-blocking crashes:**
- Systems without OPA get clean failure with installation instructions (no panic)
- JSON output is always valid (no null fields)
- Exit codes are deterministic (0 = pass, 1 = fail, 2 = internal error)
- `acc policy explain` works after failed verification
- `ACC_ALLOW_NO_OPA=1` no longer bypasses security checks

**Users of v0.1.3 MUST upgrade immediately to v0.1.4.**

### Testing

- ✅ Added `TestVerify_NoPanic_WhenOPAIsMissing` - Verifies no panic occurs (FAILS on v0.1.3, PASSES on v0.1.4)
- ✅ Added `TestVerify_ReturnsStructuredFailure_WhenOPAIsMissing` - Verifies structured failure with violations (FAILS on v0.1.3, PASSES on v0.1.4)
- ✅ Added `TestVerifyResultExitCode_NilSafe` - Verifies ExitCode handles nil (FAILS on v0.1.3, PASSES on v0.1.4)
- ✅ Added `TestVerify_WritesState_OnFailure` - Verifies state persists on failure (FAILS on v0.1.3, PASSES on v0.1.4)
- ✅ Updated `TestOPAEscapeHatch` - Verifies escape hatch creates violation, not bypass (FAILS on v0.1.3, PASSES on v0.1.4)
- ✅ All existing tests pass on v0.1.4

## [0.1.3] - 2025-01-19

### Fixed - Policy Input & Evaluation Correctness

**CRITICAL: This release fixes a critical input provisioning bug in v0.1.2's policy evaluation that broke enforcement of input-dependent rules.**

**The Bug:**
v0.1.2 introduced structured violation parsing but **failed to provide the Rego input document** to policy evaluation. This caused all policy deny rules depending on `input.*` fields to silently never trigger, allowing root containers and other violations to pass verification incorrectly.

**What Was Broken in v0.1.2:**
- Rego evaluation didn't receive any input document - `input` was empty/undefined during policy evaluation
- Policy rules depending on `input.config.User`, `input.config.Labels`, etc. never fired
- Root containers (`User == ""`) passed verification when they should have failed
- SBOM presence wasn't exposed via `input.sbom.present` for policy decisions
- Image config inspection silently fell back to empty config on failure
- `acc policy explain` didn't show input document, making policy debugging impossible
- Text-parsing fallback existed as unsafe security bypass

**Example of Broken Behavior:**
```rego
# Policy file: .acc/policy/security.rego
deny contains {
  "rule": "no-root-user",
  "severity": "high",
  "message": "Container runs as root"
} if {
  input.config.User == ""  # This condition NEVER evaluated in v0.1.2
}
```

**v0.1.2 behavior:** Root container **INCORRECTLY PASSED** (input was empty, condition never matched)
**v0.1.3 behavior:** Root container **CORRECTLY FAILS** (input.config.User == "" triggers deny)

**What's Fixed in v0.1.3:**
- ✅ **Rego input document properly constructed** - Full input object with config, sbom, attestation, promotion fields
- ✅ **Image inspection using docker/podman/nerdctl** - Extracts actual User and Labels from image config
- ✅ **Input contract defined**: `{config: {User, Labels}, sbom: {present}, attestation: {present}, promotion}`
- ✅ **Policy evaluation changed to `data.acc.policy.result`** - Evaluates full result object (violations, warnings, allow)
- ✅ **Input persisted in verification state** - `acc policy explain --json` now includes `.result.input` for debuggability
- ✅ **Image inspection failure is a violation** - Missing container tools creates critical `image-inspect-failed` violation (no silent fallback)
- ✅ **OPA is required by default** - Clear error if `opa` command not found, with installation instructions
- ✅ **Escape hatch for dev/testing** - `ACC_ALLOW_NO_OPA=1` allows tests to run without OPA (development only)
- ✅ **Removed text-parsing fallback** - All security decisions now use proper OPA evaluation
- ✅ **Backwards compatibility** - Checks both `result.violations` and `result.deny` in OPA output

### Impact

**Enforcement was BROKEN in v0.1.2** - Any policy deny rule depending on `input.*` fields (User, Labels, SBOM presence, etc.) silently never triggered. This is a **critical security regression** from v0.1.1.

**Explainability was BROKEN in v0.1.2** - Users could not see what input was provided to policies, making policy debugging impossible.

**Users of v0.1.2 MUST upgrade immediately to v0.1.3** to restore correct policy enforcement for input-dependent rules.

**Root containers and other input-dependent violations that incorrectly passed in v0.1.2 will now correctly fail in v0.1.3.**

### Testing

- ✅ Added `TestBuildRegoInput` - Verifies input document construction (FAILS on v0.1.2, PASSES on v0.1.3)
- ✅ Added `TestSBOMPresentField` - Verifies SBOM presence detection in input (FAILS on v0.1.2, PASSES on v0.1.3)
- ✅ Added `TestPolicyExplainIncludesInput` - Verifies input persists in state for policy explain (FAILS on v0.1.2, PASSES on v0.1.3)
- ✅ Added `TestImageInspectFailureCreatesViolation` - Verifies inspection failures create violations (FAILS on v0.1.2, PASSES on v0.1.3)
- ✅ Added `TestOPARequiredError` - Verifies clear error when OPA not found
- ✅ Added `TestOPAEscapeHatch` - Verifies `ACC_ALLOW_NO_OPA=1` allows dev/testing
- ✅ All 6 regression tests pass on v0.1.3

## [0.1.2] - 2025-01-19

### Fixed - Policy Correctness & Explainability

**CRITICAL: This release fixes correctness and explainability bugs in v0.1.1's policy evaluation.**

**The Bug:**
v0.1.1 correctly **enforced** policy deny rules (verification fails when denies exist), but **discarded the actual deny rule details** from Rego policies and replaced them with synthetic placeholder violations. This broke explainability, trust, and correctness of JSON/CLI output.

**What Was Broken in v0.1.1:**
- Custom `deny contains { "rule": "...", "severity": "...", "message": "..." }` objects were parsed but their fields were lost
- All deny violations showed generic `rule: "policy-deny"`, `severity: "critical"`, `message: "Policy deny rule triggered"`
- Actual rule names, custom severities, and policy-specific messages were discarded
- Multiple deny rules sometimes resulted in duplicate violations
- `acc policy explain` showed the same broken generic violations
- JSON output was unreliable for CI/GitOps consumption

**What's Fixed in v0.1.2:**
- ✅ **Structured deny objects propagated verbatim** - Rego deny objects with custom rule, severity, and message fields are now preserved exactly as written
- ✅ **No synthetic violations** - Removed all hardcoded `rule: "policy-deny"` generation
- ✅ **No duplicates** - Each deny rule produces exactly one violation
- ✅ **Faithful CLI output** - Violations display the exact rule names and messages from policy files
- ✅ **Trustworthy JSON** - `policyResult.violations` array accurately reflects policy semantics
- ✅ **Single source of truth** - CLI, `--json`, and `acc policy explain` all use the same PolicyResult

**Example:**
```rego
# Policy file: .acc/policy/security.rego
deny contains {
  "rule": "no-root-user",
  "severity": "high",
  "message": "Container runs as root"
}
```

**v0.1.1 output (WRONG):**
```json
{
  "rule": "policy-deny",
  "severity": "critical",
  "message": "Policy deny rule triggered"
}
```

**v0.1.2 output (CORRECT):**
```json
{
  "rule": "no-root-user",
  "severity": "high",
  "message": "Container runs as root"
}
```

### Impact

**Enforcement was correct in v0.1.1** - deny rules did cause verification to fail as intended. The security model was not broken.

**Explainability was broken in v0.1.1** - users could not see *which* deny rules triggered or *why* verification failed. This made debugging policies nearly impossible.

**Users of v0.1.1 should upgrade to v0.1.2** to restore policy explainability and trust in JSON output.

### Testing

- ✅ Added `TestSingleDenyRuleVerbatim` - Verifies exact field preservation (FAILS on v0.1.1, PASSES on v0.1.2)
- ✅ Added `TestMultipleDenyRules` - Verifies 3 distinct violations, no duplicates (FAILS on v0.1.1, PASSES on v0.1.2)
- ✅ Added `TestAllowAllPolicy` - Verifies allow-all policies pass with no violations
- ✅ Added `TestParseDenyObjects` - Direct parser unit tests
- ✅ Updated all existing tests to use structured deny objects
- ✅ All tests pass on v0.1.2

## [0.1.1] - 2025-01-19

### Security
- **CRITICAL: Policy deny rules now enforced** - Fixed security bug where deny rules in Rego policies were evaluated but not enforced
- Verification now correctly fails when policy deny rules are triggered
- Deny violations properly surfaced in CLI output and JSON responses
- Non-zero exit codes returned on policy failures as expected

### Fixed
- **Policy Evaluation** - `acc verify` now reads and enforces deny rules from `.acc/policy/*.rego` files
- **JSON Output** - `policyResult.allow` correctly set to `false` when deny rules exist
- **Policy Violations** - Deny messages now properly populated in `policyResult.violations` array
- **Attestation Discovery** - `acc inspect` now recursively searches `.acc/attestations/<digest>/` subdirectories
- **Attestation Visibility** - Attestation count and paths now correctly displayed in inspect output

### Impact
This release fixes a critical enforcement gap where policy deny rules were parsed but ignored during verification. Users relying on deny rules for security enforcement **must upgrade immediately** to v0.1.1.

**Before v0.1.1:** `deny` rules had no effect - verification always passed
**After v0.1.1:** `deny` rules are authoritative - verification fails when triggered

**Commands affected:** `acc verify`, `acc run`, `acc push`, `acc promote` (all verification-gated commands)

### Testing
- Added comprehensive tests for policy deny enforcement
- Added tests for JSON output correctness with deny semantics
- Added tests for attestation discovery in subdirectories
- All tests pass on v0.1.1, would fail on v0.1.0

## [0.1.0] - 2025-01-19

### Added
- **Core Commands**
  - `acc init` - Initialize new acc project with config and starter policy
  - `acc build` - Build OCI images with automatic SBOM generation
  - `acc verify` - Verify SBOM, policy compliance, and attestations with strict enforcement
  - `acc inspect` - Display artifact trust summary with verification status
  - `acc attest` - Create cryptographic attestations with canonical hashing
  - `acc push` - Push verified artifacts to registries (verification gated)
  - `acc promote` - Re-verify and promote workloads to environments
  - `acc run` - Run workloads locally with security defaults
  - `acc policy explain` - Display developer-friendly explanation of last verification decision
  - `acc version` - Display version, commit, and build information

- **Verification Gates**
  - Strict verification enforcement: failed verification blocks run/push/promote
  - No bypass flags - security by default
  - Policy mode support (enforce/warn) with explicit configuration
  - SBOM requirement for all builds
  - Image digest validation prevents tag manipulation

- **Policy & Waivers**
  - Rego policy support via `.acc/policy/` directory
  - Policy waivers with strict expiry enforcement
  - Expired waiver = verification failure (no exceptions)
  - Waiver visibility in `inspect` and `policy explain`
  - YAML-based waiver configuration (`.acc/waivers.yaml`)

- **Attestations**
  - Deterministic JSON attestations with `schemaVersion: v0.1`
  - Canonical hashing of verification results with sorted violations
  - Structured attestation storage: `.acc/attestations/<digest>/<timestamp>-attestation.json`
  - Last attestation pointer: `.acc/state/last_attestation.json`
  - Attestation schema includes subject, evidence, and metadata

- **State Management**
  - Persistent verification state: `.acc/state/last_verify.json`
  - State validation for attest and push commands
  - Timestamp tracking for all verification decisions
  - JSON state files with schema versioning

- **Security Features**
  - Runtime security defaults: network isolation, dropped capabilities, no new privileges
  - Multi-tool support: docker/podman/nerdctl/oras
  - Global flags: `--json`, `--color`, `--quiet`, `--no-emoji`
  - Deterministic JSON output with stable field ordering
  - Red output means stop (UI signals critical failures)

- **Testing & Quality**
  - Golden tests for JSON output validation and schema drift detection
  - Comprehensive unit tests for all core functionality
  - CI smoke tests with intentional policy violation examples
  - Test coverage for waiver expiry, attestation creation, and verification gates

- **Documentation**
  - Comprehensive README with examples and workflows
  - Threat model (docs/threat-model.md) defining security boundaries
  - In-scope threats: supply chain integrity, policy bypass, expired waivers
  - Out-of-scope: runtime attacks, secret scanning, SAST/DAST
  - Example projects demonstrating verification gating

### Security
- **Verification Gates Execution Principle**: If verification fails, workloads cannot run, push, or promote
- **No Bypass Flags**: No silent security degradation modes
- **Waiver Expiry Enforcement**: Expired waivers treated as critical violations
- **Canonical Hashing**: Deterministic verification result hashing prevents replay attacks
- **Digest Validation**: Push and attest commands validate image digests match verified state

---

## Versioning Policy

acc follows [Semantic Versioning](https://semver.org/):

- **v0.x.y** (pre-1.0): API is not stable, breaking changes allowed in minor versions
  - **PATCH** (0.1.1): Bug fixes, no new features
  - **MINOR** (0.2.0): New features, may include breaking changes

- **v1.x.y** (1.0+): API is stable
  - **PATCH** (1.0.1): Bug fixes only
  - **MINOR** (1.1.0): New features, backward compatible
  - **MAJOR** (2.0.0): Breaking changes

### What Constitutes a Breaking Change

- Removal or renaming of commands
- Changes to command flags or arguments
- Changes to JSON output schemas (requires schemaVersion bump)
- Changes to file formats (.acc/state/, .acc/attestations/)
- Removal of global flags
- Changes to exit codes

---

[Unreleased]: https://github.com/cloudcwfranck/acc/compare/v0.2.5...HEAD
[0.2.5]: https://github.com/cloudcwfranck/acc/compare/v0.2.4...v0.2.5
[0.2.4]: https://github.com/cloudcwfranck/acc/compare/v0.2.3...v0.2.4
[0.2.3]: https://github.com/cloudcwfranck/acc/compare/v0.2.2...v0.2.3
[0.2.2]: https://github.com/cloudcwfranck/acc/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/cloudcwfranck/acc/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/cloudcwfranck/acc/compare/v0.1.8...v0.2.0
[0.1.8]: https://github.com/cloudcwfranck/acc/compare/v0.1.7...v0.1.8
[0.1.7]: https://github.com/cloudcwfranck/acc/compare/v0.1.6...v0.1.7
[0.1.6]: https://github.com/cloudcwfranck/acc/compare/v0.1.5...v0.1.6
[0.1.5]: https://github.com/cloudcwfranck/acc/compare/v0.1.4...v0.1.5
[0.1.4]: https://github.com/cloudcwfranck/acc/compare/v0.1.3...v0.1.4
[0.1.3]: https://github.com/cloudcwfranck/acc/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/cloudcwfranck/acc/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/cloudcwfranck/acc/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/cloudcwfranck/acc/releases/tag/v0.1.0
