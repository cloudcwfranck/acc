# Changelog

All notable changes to acc will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Nothing yet

### Changed
- Nothing yet

### Fixed
- **acc upgrade backward compatibility** - Fixed upgrade/downgrade to older versions by implementing flexible binary detection that searches for executables matching `acc` or `acc-*` patterns in release archives, supporting both current (`acc`) and legacy (`acc-linux-amd64`) naming conventions

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

[Unreleased]: https://github.com/cloudcwfranck/acc/compare/v0.1.6...HEAD
[0.1.6]: https://github.com/cloudcwfranck/acc/compare/v0.1.5...v0.1.6
[0.1.5]: https://github.com/cloudcwfranck/acc/compare/v0.1.4...v0.1.5
[0.1.4]: https://github.com/cloudcwfranck/acc/compare/v0.1.3...v0.1.4
[0.1.3]: https://github.com/cloudcwfranck/acc/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/cloudcwfranck/acc/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/cloudcwfranck/acc/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/cloudcwfranck/acc/releases/tag/v0.1.0
