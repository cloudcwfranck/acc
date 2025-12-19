# Changelog

All notable changes to acc will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Distribution Channels**
  - Homebrew Formula draft in `packaging/homebrew/acc.rb` for future tap publishing
  - Container image support via `Dockerfile.release` for GHCR distribution
  - CI/CD usage examples in README for automated pipeline integration
  - Comprehensive distribution guide in `packaging/README.md`
- **Documentation**
  - Complete Windows PowerShell installation instructions
  - macOS Intel detailed installation steps
  - Multi-architecture container build documentation

### Changed
- **Release Artifacts Naming** - Changed from dashes to underscores for canonical format:
  - `acc-0.1.0-linux-amd64.tar.gz` → `acc_0.1.0_linux_amd64.tar.gz`
  - Aligns with industry standards (Homebrew, GoReleaser)
- **GitHub Release Notes** - Automatic installation instructions appended to release notes
- **README Install Section** - Updated all artifact URLs to match new naming convention

### Fixed
- Nothing yet

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
