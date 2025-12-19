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
- Nothing yet

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

[Unreleased]: https://github.com/cloudcwfranck/acc/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/cloudcwfranck/acc/releases/tag/v0.1.0
