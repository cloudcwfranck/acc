<!-- START EDITING HERE -->
# Testing Contract

This document defines the **stable behavioral contract** for `acc`. It specifies what behaviors are guaranteed, what tests enforce them, and how to safely evolve the contract over time.

## Purpose

The Testing Contract serves as:

1. **Release Gate**: Tier 0 and Tier 1 tests MUST pass before merging PRs
2. **Behavioral Documentation**: Defines expected outputs, exit codes, and JSON schemas
3. **Regression Prevention**: Tests encode assumptions that must not break
4. **Change Management**: Provides a process for intentional breaking changes

---

## Test Tiers

### Tier 0: CLI Help Matrix (BLOCKS PRs)

**Runtime**: Fast (~10-20 seconds)
**When**: Every PR, every push to main
**Script**: `scripts/cli_help_matrix.sh`

**Contract**:
- All commands MUST exist and respond to `--help`
- Help text MUST exit with code 0
- Help output MUST be non-empty

**Commands Under Test**:
```bash
# Root command
acc --help

# Core commands
acc init --help
acc build --help
acc verify --help
acc run --help
acc push --help
acc promote --help
acc attest --help
acc inspect --help
acc version --help
acc upgrade --help

# Subcommands
acc trust --help
acc trust status --help
acc policy --help
acc policy explain --help

# Possibly not-implemented (must return clear message)
acc config --help    # May return "not implemented" with non-zero exit
acc login --help     # May return "not implemented" with non-zero exit
```

**Not-Implemented Commands**:
- `acc config --help`: Currently behaves as help-only stub (exit 0, shows help text)
- `acc login --help`: Currently behaves as help-only stub (exit 0, shows help text)

**Contract for Stub Commands**:
- Stub commands MAY return help text and exit 0 (indicates future implementation planned)
- Stub commands MAY return "not implemented" error with non-zero exit (indicates not yet available)
- Tier 0 tests verify command exists and either shows help OR clear "not implemented" message
- Changing from stub (exit 0) to error (exit 1) is a MINOR version bump (more restrictive)
- Implementing full functionality is a MINOR version bump (backward compatible)

**Breaking Changes**:
- Removing a command is a MAJOR version bump
- Changing help exit code from 0 to non-zero is a MAJOR version bump
- Adding new commands is a MINOR version bump

---

### Tier 1: E2E Smoke Tests (BLOCKS PRs)

**Runtime**: Medium (~60-90 seconds)
**When**: Every PR, every push to main
**Script**: `scripts/e2e_smoke.sh`
**Dependencies**: docker, opa v0.66.0, jq, syft

**Contract**: End-to-end workflow with local images (no external registry)

#### 1. Project Initialization

```bash
acc init <project-name>
```

**Guarantees**:
- MUST create `.acc/` directory
- MUST create `acc.yaml` configuration file
- MUST create `.acc/profiles/` directory (as of v0.2.1)
- Exit code: 0 on success

**Breaking Changes**:
- Removing `.acc/profiles/` creation is a MAJOR version bump (reverts v0.2.1 fix)
- Changing default directory structure is a MAJOR version bump

#### 2. Build Workflow

```bash
acc build <image>           # Positional argument (v0.2.3+)
acc build --tag <image>     # Flag syntax (always supported)
acc build -t <image>        # Short flag (always supported)
```

**Guarantees** (v0.2.3+):
- MUST accept both positional and flag syntax for image reference
- MUST generate SBOM in `.acc/sbom/<project>.<format>.json`
- MUST verify SBOM file exists after generation
- MUST exit non-zero if SBOM generation fails
- If exit 0, SBOM MUST exist

**Exit Codes**:
- 0: Build succeeded, SBOM created
- 1: Build failed (docker/podman/buildah error, syft error, etc.)

**Breaking Changes**:
- Removing positional argument support is a MAJOR version bump (breaks v0.1.x scripts)
- Changing SBOM filename pattern is a MAJOR version bump
- Allowing build to succeed without SBOM is a CRITICAL BUG (v0.2.3 regression)

#### 3. Verify Workflow

```bash
acc verify --json <image>
```

**Guarantees**:
- MUST return valid JSON when `--json` flag is used
- MUST set `.status` field to exactly "pass" or "fail"
- MUST include `.policyResult` object when policy evaluation occurs
- MUST include `.sbomPresent` boolean field
- Status MUST match `policyResult.allow` (v0.2.2+)

**Exit Codes**:
- 0: Verification passed (status: "pass")
- 1: Verification failed (status: "fail", policy violations)
- 2: Verification could not complete (SBOM missing in enforce mode, image not found)

**JSON Schema** (stable fields):
```json
{
  "schemaVersion": "v0.2",
  "status": "pass|fail",
  "sbomPresent": boolean,
  "policyResult": {
    "allow": boolean,
    "violations": [
      {
        "rule": string,
        "severity": "low|medium|high|critical",
        "message": string
      }
    ]
  }
}
```

**Critical Invariants** (v0.2.2+):
- `status == "pass"` IF AND ONLY IF `policyResult.allow == true`
- `status == "fail"` IF AND ONLY IF `policyResult.allow == false`
- This is the **Single Authoritative Final Gate** contract

**Breaking Changes**:
- Changing JSON field names is a MAJOR version bump
- Changing `schemaVersion` indicates potential breaking change
- Removing fields is a MAJOR version bump (add deprecation warnings first)
- Violating status/allow consistency is a CRITICAL BUG (v0.2.2 regression)

#### 4. Policy Violations

**Test**: Verify image running as root user

**Contract**:
- Default policy MUST include `no-root-user` rule
- Root user MUST cause verification to fail
- Violation MUST appear in `policyResult.violations[]` with `rule: "no-root-user"`

**Breaking Changes**:
- Removing `no-root-user` from default policy is a MAJOR version bump
- Changing violation rule names is a MAJOR version bump

#### 5. Policy Explain

```bash
acc policy explain --json
```

**Guarantees**:
- MUST return valid JSON when `--json` flag is used
- MUST include `.result.input` field
- `.result.input` MUST be an object (not string, not null)
- Should show the most recent verification decision

**Exit Codes**:
- 0: Explanation available
- Non-zero: No verification history (implementation may vary)

**Breaking Changes**:
- Changing `.result.input` type from object is a MAJOR version bump
- Removing `.result.input` field is a MAJOR version bump

#### 6. Attest UX Contract

**Scenario 1**: Attest after verifying different image (mismatch) (v0.2.7)
```bash
acc verify image-a
acc attest image-b   # Should FAIL
```

**Guarantees**:
- MUST exit non-zero (image mismatch)
- MUST NOT print "Creating attestation" message
- Error message MUST indicate digest mismatch and provide remediation steps
- Comparison MUST use digest matching (not tag string matching)

**Scenario 2**: Attest after verifying same image (success) (v0.2.7)
```bash
acc verify image-a
acc attest image-a   # Should SUCCEED
```

**Guarantees**:
- MUST exit 0
- MUST print "Creating attestation" message to stdout/stderr
- MUST create attestation file in `.acc/attestations/<digest>/`
- Attestation stored per-image digest (12-char digest prefix as directory name)
- Attestation appears in `acc trust status` output for that specific image only

**Digest Matching** (v0.2.7):
- Attestation safety uses digest comparison, not tag comparison
- Prevents accidental attestation of wrong image even if tags are reused
- If digest cannot be resolved, falls back to tag string comparison

**Attestation Storage** (v0.2.7):
- Location: `.acc/attestations/<digest-prefix>/YYYYMMDD-HHMMSS-attestation.json`
- Digest prefix: First 12 characters of image digest
- Per-image isolation: Each image digest has its own directory
- Multiple attestations: Same image can have multiple timestamped attestations

**Integration with Trust Status** (v0.2.7):
- After `acc attest demo-app:ok`, `acc trust status demo-app:ok` shows the attestation
- Attestations are per-image: `acc trust status demo-app:root` will NOT show demo-app:ok attestations

**Breaking Changes**:
- Changing UX messages is a MINOR version bump
- Removing mismatch detection is a MAJOR version bump (security regression)
- v0.2.7 clarifies digest-based matching (behavior unchanged, documentation improved)

#### 7. Inspect Per-Image State

```bash
acc inspect --json <image>
```

**Guarantees**:
- MUST return state specific to the requested image digest
- MUST NOT leak state from other images (v0.2.1 fix)
- MUST return valid JSON when `--json` flag is used
- `.status` MUST match the verification status for that specific image

**Exit Codes**:
- 0: Inspection succeeded (regardless of pass/fail status)

**Critical Invariant**: No cross-image state leakage

**Breaking Changes**:
- Reverting per-image state isolation is a CRITICAL BUG (v0.2.1 regression)

#### 8. Trust Status

```bash
acc trust status --json <image>
```

**Guarantees** (v0.2.7):
- MUST return deterministic JSON with stable keys
- MUST reflect per-image state (no leakage between different image digests)
- MUST handle "never verified" case gracefully

**Exit Codes** (v0.2.7):
- 0: Trust status can be computed (even if status is fail)
- 2: Trust status cannot be computed (missing state, corrupted data, missing image)
- Implementation note: Exit 0 with `status: "unknown"` is also acceptable for backward compatibility

**JSON Schema** (v0.2.7):
```json
{
  "schemaVersion": "v0.2",
  "imageRef": string,
  "status": "pass|fail|unknown",
  "sbomPresent": boolean,
  "violations": array,
  "warnings": array,
  "attestations": array,
  "timestamp": string
}
```

**Required Fields** (v0.2.7):
- `schemaVersion`: Always "v0.2"
- `imageRef`: Image reference provided
- `status`: One of "pass", "fail", "unknown"
- `sbomPresent`: Boolean (always set, never null)
- `violations`: Array of violations (empty array [] if none, never null)
- `warnings`: Array of warnings (empty array [] if none, never null)
- `attestations`: Array of attestation paths for this specific image (per-image isolation)
- `timestamp`: ISO 8601 timestamp (empty string "" for unknown status)

**Optional Fields**:
- `profileUsed`: Policy profile name (if applicable)

**Per-Image Isolation** (v0.2.7):
- Attestations are scoped to specific image digests
- `acc trust status demo-app:ok` only shows attestations for demo-app:ok
- `acc trust status demo-app:root` only shows attestations for demo-app:root
- No cross-image state leakage

**Breaking Changes**:
- Changing "never verified" exit code is a MINOR version bump (document behavior)
- Changing JSON schema is a MAJOR version bump
- v0.2.7 added required fields: sbomPresent, attestations, timestamp (backward compatible)

#### 9. Run Command

```bash
acc run <image> [-- command args...]
```

**Guarantees**:
- MUST verify image before running
- MUST exit non-zero if verification fails
- If run is not fully implemented, MUST show help or clear error

**Exit Codes**:
- 0: Container ran successfully
- 1: Verification failed
- 2: Runtime error
- Non-zero: Not implemented or container failed

**Contract Status**: Partially implemented - validation in Tier 0, functional test optional

---

### Tier 2: Registry Integration (NEVER BLOCKS)

**Runtime**: Slow (~2-5 minutes)
**When**: Nightly scheduled, on tags, manual main branch pushes
**Script**: `scripts/registry_integration.sh`
**Dependencies**: docker, opa, jq, syft, GHCR access

**Contract**: Tests push/promote workflows with live registry

**Auto-Skip Conditions**:
- `GHCR_REPO` environment variable not set
- Not logged in to GHCR
- No network access

**Tests**:
1. Push verified image to GHCR
2. Promote image to environment (if supported)
3. Pull from registry and re-verify

**Guarantees**:
- Tier 2 failures MUST NOT block PR merges
- Script MUST skip gracefully when prerequisites not met
- Script MUST log clear skip messages

**Breaking Changes**: N/A (Tier 2 never blocks)

---

## Stable Behaviors Summary

### Exit Code Contract

**CRITICAL**: Exit codes are part of the stable contract. Changing exit codes is a **MAJOR version bump**.

| Command | Exit 0 | Exit 1 | Exit 2 | Notes |
|---------|--------|--------|--------|-------|
| `acc --help` | Help displayed | N/A | N/A | All help commands MUST exit 0 |
| `acc init` | Project created | Initialization failed | N/A | |
| `acc build` | Build succeeded + SBOM created | Build or SBOM generation failed | N/A | If exit 0, SBOM MUST exist |
| `acc verify` | Verification passed (`status:"pass"`) | Verification failed (`status:"fail"`) | Cannot complete (SBOM missing, image not found) | **Exit code MUST match `.status` field** |
| `acc attest` | Attestation created | Mismatch or verification state missing | N/A | MUST fail when image != last verified |
| `acc inspect` | Inspection succeeded | N/A | N/A | Always exit 0, check `.status` field |
| `acc trust status` | Status available | N/A | No verification state found | Some versions may return 0 with `status:"unknown"` |
| `acc run` | Container ran successfully | Verification failed | Runtime error | Verification gate enforced |
| `acc push` | Push succeeded | Push failed or verification gate blocked | N/A | Verification gate enforced |
| `acc policy explain` | Explanation available | Varies by implementation | Varies by implementation | |

**Exit Code Semantics**:
- **Exit 0**: Operation succeeded (for verify: policy passed)
- **Exit 1**: Operation failed (for verify: policy violations found)
- **Exit 2**: Operation could not complete (missing prerequisites, no state)

**Regression Detection**: Tests enforce exit codes match JSON output. Mismatches are CRITICAL BUGS.

---

### Test Script Implementation Patterns

**Script Quality Standards**: All test scripts follow ShellCheck best practices to ensure robustness and portability.

#### Array-Based Command Invocation

Test scripts use array-based command execution to avoid word-splitting issues (ShellCheck SC2086):

```bash
# cli_help_matrix.sh pattern for dynamic commands
test_help_command() {
    local cmd_args="$1"

    # Build command array
    local cmd=( "$ACC_BIN" )
    # shellcheck disable=SC2206
    cmd+=( $cmd_args )  # Intentional word splitting

    # Execute safely
    set +e
    output=$("${cmd[@]}" 2>&1)
    exit_code=$?
    set -e
}
```

#### Safe Exit Code Capture

All helper functions use `set +e` / `set -e` pattern to capture exit codes without triggering ShellCheck SC2317 (unreachable code) warnings:

```bash
assert_success() {
    local description="$1"
    shift

    local output
    local exit_code

    # Disable errexit to capture actual exit code
    set +e
    output=$("$@" 2>&1)
    exit_code=$?
    set -e  # Re-enable errexit

    if [ $exit_code -eq 0 ]; then
        log_success "$description"
    else
        log_error "$description (exit code: $exit_code)"
    fi
}
```

**Why This Pattern**:
- Using `|| true` masks actual exit codes (`$?` always captures 0)
- `set +e` allows command to fail without terminating script
- `$?` correctly captures the command's exit code
- `set -e` restores fail-fast behavior for subsequent commands

#### Variable Quoting

All variable references in command invocations are properly quoted to prevent word splitting:

```bash
# Correct: Quote variables in command substitution
log "✓ $tool: $(command -v "$tool")"

# Correct: Variables passed through function parameters
assert_success "description" "$ACC_BIN" verify --json "$image"
```

**ShellCheck Compliance**: Test scripts aim for zero ShellCheck warnings. Any suppressions (`# shellcheck disable=SCxxxx`) are documented with justification.

---

### JSON Output Stability

**Guaranteed Fields** (MUST NOT remove without major version bump):
```
verify output:
  .status (string: "pass"|"fail")
  .policyResult.allow (boolean)
  .policyResult.violations (array)
  .sbomPresent (boolean)
  .schemaVersion (string)

inspect output:
  .status (string)
  .imageRef (string)
  .schemaVersion (string)

trust status output:
  .status (string)
  .imageRef (string)
  .schemaVersion (string)
```

**Allowed Changes**:
- Adding new fields: MINOR version bump
- Adding new optional fields: PATCH version bump
- Removing fields: MAJOR version bump
- Changing field types: MAJOR version bump
- Adding enum values: MINOR version bump
- Removing enum values: MAJOR version bump

---

## Versioning and Breaking Changes

### Contract Versioning

This Testing Contract follows semantic versioning aligned with acc releases:

- **MAJOR** (`v1.0.0` → `v2.0.0`): Breaking changes to stable contract
  - Remove command
  - Remove JSON field
  - Change exit code semantics
  - Break Tier 0 or Tier 1 guarantees

- **MINOR** (`v0.2.0` → `v0.3.0`): Backward-compatible additions
  - Add new command
  - Add new JSON field
  - Add new enum value
  - Enhance error messages

- **PATCH** (`v0.2.2` → `v0.2.3`): Bug fixes and clarifications
  - Fix regression (restore v0.1.x behavior)
  - Improve test coverage
  - Documentation updates

### Breaking Change Process

If you need to make a breaking change:

1. **Document the change** in this contract under a new version section
2. **Add deprecation warnings** in the release before removal (if applicable)
3. **Update CHANGELOG.md** with breaking change notice
4. **Add migration guide** in release notes
5. **Bump MAJOR version** when releasing
6. **Update all tests** to reflect new contract

### Example: Removing a Command

**Bad** (immediate removal):
```bash
# v0.2.x
acc foo --help  # works

# v0.3.0
acc foo --help  # ERROR: unknown command
```

**Good** (deprecation path):
```bash
# v0.2.5
acc foo --help  # works, prints deprecation warning

# v0.3.0
acc foo --help  # works, prints deprecation warning

# v1.0.0
acc foo --help  # ERROR: command removed in v1.0.0
```

---

## How to Run Tests Locally

### Prerequisites

Install required tools:
```bash
# OPA v0.66.0
curl -L -o opa https://openpolicyagent.org/downloads/v0.66.0/opa_linux_amd64_static
chmod +x opa
sudo mv opa /usr/local/bin/

# jq
sudo apt-get install jq  # or: brew install jq

# syft
curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin

# docker
# Follow: https://docs.docker.com/get-docker/
```

### Run Individual Tiers

**Tier 0** (fast, no dependencies):
```bash
# Build acc first
go build -o acc ./cmd/acc

# Run Tier 0
bash scripts/cli_help_matrix.sh
```

**Tier 1** (requires docker + tools):
```bash
# Build acc first
go build -o acc ./cmd/acc

# Run Tier 1
bash scripts/e2e_smoke.sh

# View logs if failed
cat /tmp/tier1-e2e-*.log
```

**Tier 2** (requires GHCR access):
```bash
# Build acc first
go build -o acc ./cmd/acc

# Login to GHCR
echo $GITHUB_TOKEN | docker login ghcr.io -u <username> --password-stdin

# Set environment
export GHCR_REPO="<owner>/<repo>"
export GITHUB_SHA=$(git rev-parse --short HEAD)

# Run Tier 2
bash scripts/registry_integration.sh
```

### Run All Tests

```bash
# Build
go build -o acc ./cmd/acc

# Tier 0
bash scripts/cli_help_matrix.sh

# Tier 1
bash scripts/e2e_smoke.sh

# Tier 2 (optional)
bash scripts/registry_integration.sh
```

---

## CI/CD Integration

### GitHub Actions Workflow

Tests run automatically via `.github/workflows/ci.yml`:

- **On PR**: Tier 0 + Tier 1 (must pass to merge)
- **On push to main**: Tier 0 + Tier 1 (must pass)
- **Nightly**: Tier 0 + Tier 1 + Tier 2
- **On tags**: Tier 0 + Tier 1 + Tier 2

### Required Status Checks

Configure branch protection for `main`:
- ✅ Tier 0: CLI Help Matrix
- ✅ Tier 1: E2E Smoke Tests
- ✅ Changelog Check
- ❌ Tier 2: Registry Integration (optional, never blocks)

### Test Artifacts

On failure, tests upload artifacts:
- Tier 0: `/tmp/tier0-*.log`
- Tier 1: `/tmp/tier1-*.log` + workdir `/tmp/acc-e2e-*/`
- Tier 2: `/tmp/tier2-*.log` + workdir `/tmp/acc-registry-*/`

Artifacts retained for 7 days.

---

## Updating This Contract

### When to Update

Update this contract when:
1. Adding new commands or features
2. Changing stable behavior intentionally
3. Fixing bugs that restore v0.1.x behavior
4. Adding new test coverage

### How to Update

1. **Add new section** for the feature/behavior
2. **Define guarantees** (exit codes, JSON schema, invariants)
3. **Add tests** to enforce the contract
4. **Update CHANGELOG.md** to reference contract changes
5. **Version appropriately** (MAJOR/MINOR/PATCH)

### Contract Review Process

- All contract changes MUST be reviewed in PR
- Contract changes MUST align with semantic versioning
- Breaking changes MUST have migration guide
- Tests MUST be updated to match new contract

---

## FAQ

### Q: Why do Tier 0 and Tier 1 block PRs, but not Tier 2?

**A**: Tier 0 and Tier 1 test stable local behaviors that should never regress. Tier 2 tests external integrations (registry push/pull) which may fail due to network issues, credentials, or rate limits that are outside the developer's control.

### Q: What if I need to break the contract?

**A**: Follow the Breaking Change Process above. Document the change, add deprecation warnings, provide migration guide, and bump the MAJOR version.

### Q: Can I add new tests without updating the contract?

**A**: Yes, if the test enforces an existing guarantee. If the test adds a new guarantee or changes expected behavior, update the contract.

### Q: What if a test is flaky?

**A**: Fix the test or remove it. Flaky tests erode trust in the CI system. If behavior is non-deterministic, document it in the contract and adjust test expectations.

### Q: How do I know if my change is MAJOR, MINOR, or PATCH?

**A**:
- Changes command availability, exit codes, or JSON schema = MAJOR
- Adds new commands or fields = MINOR
- Fixes bugs or improves docs = PATCH
- When in doubt, ask in PR review

---

## Version History

### v0.2.7 (2025-12-22)
- **Contract**: Trust Status JSON schema stabilized with required fields
  - `sbomPresent` MUST always be set as boolean (never null)
  - `violations`, `warnings`, `attestations` MUST be arrays (never null)
  - `timestamp` MUST be string (empty "" for unknown status)
- **Contract**: Trust Status exit codes clarified
  - Exit 0: status can be computed (pass, fail, or unknown)
  - Exit 2: status cannot be computed (missing state, corrupted data)
- **Contract**: Trust Status per-image attestation isolation
  - Attestations scoped to specific image digests
  - No cross-image attestation leakage
- **Contract**: Attest digest-based matching documented
  - Uses digest comparison (not tag string matching)
  - Provides clear remediation messages on mismatch
- **Added**: Unit tests for trust package (9 tests)
- **Added**: Golden tests for trust status JSON output (4 tests)
- **Enhanced**: E2E tests validate trust/attest integration and per-image isolation

### v0.2.3 (2025-12-20)
- **Added**: Tier 0/1/2 test infrastructure
- **Added**: Testing Contract document
- **Contract**: Build MUST accept positional arguments (v0.1.x compatibility)
- **Contract**: Build MUST verify SBOM file exists or fail
- **Contract**: Help text MUST exit 0 for all commands

### v0.2.2 (2025-12-20)
- **Contract**: Single Authoritative Final Gate - status MUST match policyResult.allow
- **Contract**: Verify MUST never return status:fail with allow:true

### v0.2.1 (2025-12-20)
- **Contract**: acc init MUST create .acc/profiles/ directory
- **Contract**: Inspect MUST NOT leak state across images
- **Contract**: Trust status MUST resolve image digest correctly

### v0.2.0 (2025-12-19)
- **Contract**: Policy Profiles and Trust Status features added
- **Contract**: Profile filtering applies AFTER policy evaluation

### v0.1.x
- Initial stable contract (implicit)
- Build accepted positional arguments
- Verify enforced SBOM requirement
- Basic policy evaluation with OPA

---

**Last Updated**: 2025-12-22
**Contract Version**: v0.2.7
**Maintained By**: acc core team
