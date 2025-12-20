# Contributing to acc

Thank you for your interest in contributing to acc! This document provides guidelines and instructions for contributing to the project.

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment for all contributors.

## Getting Started

### Prerequisites

- Go 1.21 or later
- One of: Docker, Podman, Buildah, or nerdctl
- [syft](https://github.com/anchore/syft) for SBOM generation (optional for core development)
- Git

### Setting Up Your Development Environment

```bash
# Clone the repository
git clone https://github.com/cloudcwfranck/acc.git
cd acc

# Build the project
go build -o acc ./cmd/acc

# Run tests
go test ./...

# Verify your build
./acc version
```

## Development Workflow

### 1. Running Tests

All contributions must pass the test suite:

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test ./internal/verify -v
```

**Important:** All pull requests must pass `go test ./...` before merge.

### 1.1 Running CI Tests Locally

Before pushing, you can run the same tests that CI runs to catch issues early:

#### Prerequisites for CI Tests

```bash
# Install OPA (v0.66.0)
curl -L -o opa https://openpolicyagent.org/downloads/v0.66.0/opa_linux_amd64_static
chmod +x opa
sudo mv opa /usr/local/bin/
opa version

# Install syft (for SBOM generation)
curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin
syft version

# Install jq (if not already installed)
sudo apt-get install -y jq  # Ubuntu/Debian
brew install jq             # macOS

# Verify docker is available
docker --version
```

#### Run Tier 0: CLI Help Matrix

Fast validation of all command help text:

```bash
# Build acc first
go build -o acc ./cmd/acc

# Run Tier 0 tests
bash scripts/cli_help_matrix.sh

# Check exit code
echo $?  # Should be 0
```

**What it tests**: All commands exist and show help correctly.

#### Run Tier 1: E2E Smoke Tests

Comprehensive offline functional tests:

```bash
# Build acc first
go build -o acc ./cmd/acc

# Run Tier 1 tests
bash scripts/e2e_smoke.sh

# Check exit code
echo $?  # Should be 0

# View logs on failure
cat /tmp/tier1-*.log

# Inspect workdir on failure
ls -la /tmp/acc-e2e-*/
```

**What it tests**: Full workflow including init, build, verify, attest, inspect, trust status.

**Runtime**: ~60-90 seconds

**Expected Result**: All tests should pass. If any fail, check logs at `/tmp/tier1-*.log`.

#### Run Tier 2: Registry Integration (Optional)

Tests push/promote workflows with GHCR. **Tier 2 never blocks PRs** and is optional for local development.

```bash
# Prerequisites: Login to GHCR
echo $GITHUB_TOKEN | docker login ghcr.io -u <your-username> --password-stdin

# Set environment variables
export GHCR_REPO="<your-username>/acc"
export GHCR_REGISTRY="ghcr.io"
export GITHUB_SHA=$(git rev-parse --short HEAD)

# Build acc first
go build -o acc ./cmd/acc

# Run Tier 2 tests
bash scripts/registry_integration.sh
```

**What it tests**: Push to GHCR, promote, pull and re-verify.

**Runtime**: ~2-5 minutes

**Note**: Script auto-skips if GHCR_REPO not set or not logged in.

#### Validate Scripts (CI Syntax Check)

The exact validation CI runs on scripts:

```bash
# Validate bash syntax
for script in scripts/*.sh; do
  echo "Checking $script"
  bash -n "$script"
done

# Run shellcheck if available (optional)
if command -v shellcheck &> /dev/null; then
  shellcheck scripts/*.sh
else
  echo "shellcheck not installed (optional)"
fi
```

#### Quick Pre-Push Checklist

Run this before pushing to catch CI failures early:

```bash
# 1. Format code
gofmt -w .

# 2. Run Go tests
go test ./...

# 3. Build
go build -o acc ./cmd/acc

# 4. Run Tier 0 (fast)
bash scripts/cli_help_matrix.sh

# 5. Run Tier 1 (comprehensive)
bash scripts/e2e_smoke.sh

# 6. Check CHANGELOG.md updated
git diff origin/main...HEAD -- CHANGELOG.md
```

**Interpreting Results**:
- ✅ Tier 0, Tier 1, and Go tests MUST all pass
- ⏭️  Tier 2 is optional (never blocks PRs)

### 2. Code Formatting

All Go code must be formatted with `gofmt`:

```bash
# Check formatting (what CI runs)
if [ -n "$(gofmt -l .)" ]; then
  echo "Code needs formatting"
  gofmt -d .
fi

# Format all code
gofmt -w .
```

**CI will reject PRs with unformatted code.** Run `gofmt -w .` before committing.

### 3. Updating the CHANGELOG

**All user-facing changes require a CHANGELOG entry.**

Before opening a pull request:

1. Edit `CHANGELOG.md`
2. Add your changes under the `[Unreleased]` section
3. Use the appropriate category:
   - **Added** - New features
   - **Changed** - Changes to existing functionality
   - **Deprecated** - Soon-to-be removed features
   - **Removed** - Removed features
   - **Fixed** - Bug fixes
   - **Security** - Security improvements

Example:

```markdown
## [Unreleased]

### Added
- `acc config get/set` commands for runtime configuration

### Fixed
- Fix race condition in SBOM generation
```

See `docs/releasing.md` for detailed changelog guidelines.

**Note:** CI enforces changelog updates on pull requests. PRs without CHANGELOG updates will fail CI checks.

### 4. Commit Messages

Write clear, descriptive commit messages:

```
Add acc config command for runtime configuration

Implements `acc config get <key>` and `acc config set <key> <value>`
to enable runtime configuration without editing acc.yaml.

Fixes #123
```

**Format:**
- First line: Short summary (50 chars or less)
- Blank line
- Detailed description explaining the "why" and "what"
- Reference related issues

### 5. Opening Pull Requests

**Before opening a PR:**
- ✅ Run `go test ./...` - all tests pass
- ✅ Run `gofmt -w .` - code is formatted
- ✅ Update `CHANGELOG.md` under `[Unreleased]`
- ✅ Update documentation if adding/changing features
- ✅ Ensure security principles are maintained

**PR Description Template:**

```markdown
## Summary
Brief description of what this PR does

## Changes
- Item 1
- Item 2

## Testing
How this was tested

## Checklist
- [ ] Tests pass (`go test ./...`)
- [ ] Code formatted (`gofmt -w .`)
- [ ] CHANGELOG.md updated
- [ ] Documentation updated (if applicable)
```

### 6. Review Process

- All PRs require review before merge
- CI must pass (tests + formatting)
- Address review feedback promptly
- Maintainers may request changes for security, design, or quality reasons

## Project Guidelines

### Security Principles (Non-Negotiable)

acc is a security-focused tool. **All contributions must respect these principles:**

1. **Verification Gates Execution** - If verification fails, workloads cannot run/push/promote
2. **No Bypass Flags** - No `--skip-verify`, `--force`, or similar flags
3. **No Silent Degradation** - Failures are fatal; no fallback to insecure defaults
4. **Explicit Trust** - Trust is cryptographic, not implied

**Pull requests that violate these principles will be rejected.**

### Code Quality Standards

- **No dead code** - Remove unused functions, variables, imports
- **No commented-out code** - Delete it (Git preserves history)
- **Minimal abstractions** - Prefer simple, direct code over premature abstractions
- **Clear error messages** - Include remediation steps where possible
- **Deterministic behavior** - Avoid time-based or random behavior where possible

### Testing Requirements

- **Unit tests required** for all new functionality
- **Golden tests** for JSON output changes (see `testdata/README.md`)
- **Error cases** must be tested
- **No flaky tests** - Tests must pass consistently

### Documentation Requirements

- **Update README.md** if adding new commands or features
- **Add examples** for new functionality
- **Update docs/** if changing architecture or workflows
- **Comment complex logic** (but prefer self-documenting code)

## Repository Structure

```
acc/
├── cmd/acc/           # Main CLI entry point
├── internal/          # Internal packages (not for external import)
│   ├── attest/       # Attestation creation
│   ├── build/        # OCI image builds
│   ├── config/       # Configuration management
│   ├── inspect/      # Artifact inspection
│   ├── policy/       # Policy evaluation
│   ├── push/         # Registry push with gates
│   ├── runtime/      # Secure runtime execution
│   ├── verify/       # Verification logic
│   ├── waivers/      # Policy waivers
│   └── ui/           # Output formatting
├── testdata/         # Test fixtures and golden files
├── docs/             # Documentation
└── .github/          # GitHub Actions workflows
```

## What NOT to Contribute

Please **do not** submit PRs for:

- ❌ Bypass flags (`--skip-verify`, `--force`, etc.)
- ❌ Silent degradation modes
- ❌ Interactive shells or exec functionality
- ❌ Runtime monitoring/EDR features
- ❌ SAST/DAST scanning
- ❌ Secret scanning
- ❌ Kubernetes cluster management
- ❌ Multi-tenancy features

See `docs/threat-model.md` for the explicit non-goals of acc.

## Getting Help

- **Questions?** Open a [GitHub Discussion](https://github.com/cloudcwfranck/acc/discussions)
- **Bug reports?** Open a [GitHub Issue](https://github.com/cloudcwfranck/acc/issues)
- **Security issues?** See [SECURITY.md](./SECURITY.md)

## Release Process

Contributors don't need to worry about releases - maintainers handle versioning and tagging.

If you're curious about the release process, see `docs/releasing.md`.

## License

By contributing to acc, you agree that your contributions will be licensed under the Apache License 2.0.

## Thank You!

Your contributions help make supply chain security more accessible. Thank you for being part of the acc community!
