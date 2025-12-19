# acc - Secure Workload Accelerator

`acc` is a secure workload accelerator that turns source code and OCI artifacts into cryptographically verifiable, policy-compliant workloads.

`acc` wraps and hardens OCI workflows with verification gates - ensuring that only verified, policy-compliant workloads can be built, run, pushed, or promoted.

## Core Principles

- **Verification gates execution** - If verification fails, workloads cannot run, push, or promote
- **Red output means stop** - Always
- **Security by default** - No bypass flags, no silent degradation
- **Explicit guarantees** - Trust is cryptographic, not implied

## Features

- **Policy-gated builds** - OCI image builds with automatic SBOM generation
- **Verification enforcement** - SBOM validation, policy compliance, attestation checking
- **Secure runtime** - Run workloads locally with least-privilege defaults
- **Cryptographic attestations** - Sign and verify build provenance
- **Multi-tool support** - Works with Docker, Podman, Buildah, and nerdctl

## Quick Start

### Prerequisites

- Go 1.21 or later
- One of: Docker, Podman, or Buildah
- [syft](https://github.com/anchore/syft) for SBOM generation

### Installation

#### Option 1: Download Pre-built Binaries (Recommended)

Download the latest release from [GitHub Releases](https://github.com/cloudcwfranck/acc/releases):

**Linux (AMD64):**
```bash
# Download the latest release
VERSION="0.1.0"
curl -LO "https://github.com/cloudcwfranck/acc/releases/download/v${VERSION}/acc_${VERSION}_linux_amd64.tar.gz"

# Verify checksum (recommended)
curl -LO "https://github.com/cloudcwfranck/acc/releases/download/v${VERSION}/checksums.txt"
sha256sum -c checksums.txt --ignore-missing

# Extract and install
tar -xzf "acc_${VERSION}_linux_amd64.tar.gz"
sudo mv acc-linux-amd64 /usr/local/bin/acc
chmod +x /usr/local/bin/acc

# Verify installation
acc version
```

**macOS (Apple Silicon):**
```bash
# Download the latest release
VERSION="0.1.0"
curl -LO "https://github.com/cloudcwfranck/acc/releases/download/v${VERSION}/acc_${VERSION}_darwin_arm64.tar.gz"

# Verify checksum (recommended)
curl -LO "https://github.com/cloudcwfranck/acc/releases/download/v${VERSION}/checksums.txt"
shasum -a 256 -c checksums.txt --ignore-missing

# Extract and install
tar -xzf "acc_${VERSION}_darwin_arm64.tar.gz"
sudo mv acc-darwin-arm64 /usr/local/bin/acc
chmod +x /usr/local/bin/acc

# Verify installation
acc version
```

**macOS (Intel):**
```bash
# Use acc_${VERSION}_darwin_amd64.tar.gz instead
VERSION="0.1.0"
curl -LO "https://github.com/cloudcwfranck/acc/releases/download/v${VERSION}/acc_${VERSION}_darwin_amd64.tar.gz"
curl -LO "https://github.com/cloudcwfranck/acc/releases/download/v${VERSION}/checksums.txt"
shasum -a 256 -c checksums.txt --ignore-missing
tar -xzf "acc_${VERSION}_darwin_amd64.tar.gz"
sudo mv acc-darwin-amd64 /usr/local/bin/acc
chmod +x /usr/local/bin/acc
acc version
```

**Windows (AMD64):**
```powershell
# Download the latest release
$VERSION = "0.1.0"
Invoke-WebRequest -Uri "https://github.com/cloudcwfranck/acc/releases/download/v$VERSION/acc_${VERSION}_windows_amd64.zip" -OutFile "acc_${VERSION}_windows_amd64.zip"

# Download checksums for verification
Invoke-WebRequest -Uri "https://github.com/cloudcwfranck/acc/releases/download/v$VERSION/checksums.txt" -OutFile "checksums.txt"

# Extract
Expand-Archive -Path "acc_${VERSION}_windows_amd64.zip" -DestinationPath .

# Verify
.\acc-windows-amd64.exe version

# Add to PATH (optional - requires admin)
# Move-Item .\acc-windows-amd64.exe C:\Windows\System32\acc.exe
```

**CI/CD Usage:**
```bash
# GitHub Actions / GitLab CI / Jenkins
VERSION="0.1.0"
OS="linux"  # or darwin, windows
ARCH="amd64"  # or arm64

# Download binary
curl -LO "https://github.com/cloudcwfranck/acc/releases/download/v${VERSION}/acc_${VERSION}_${OS}_${ARCH}.tar.gz"
curl -LO "https://github.com/cloudcwfranck/acc/releases/download/v${VERSION}/checksums.txt"

# Verify checksum
sha256sum -c checksums.txt --ignore-missing || shasum -a 256 -c checksums.txt --ignore-missing

# Extract
tar -xzf "acc_${VERSION}_${OS}_${ARCH}.tar.gz"

# Make executable and add to PATH
chmod +x acc-${OS}-${ARCH}
sudo mv acc-${OS}-${ARCH} /usr/local/bin/acc

# Use in pipeline
acc version
acc verify myimage:latest
```

#### Option 2: Build from Source

```bash
# Prerequisites: Go 1.21+
git clone https://github.com/cloudcwfranck/acc.git
cd acc

# Build
go build -o acc ./cmd/acc

# Install (optional)
sudo mv acc /usr/local/bin/

# Verify
acc version
```

### Basic Usage

#### 1. Initialize a project

```bash
# Create a new acc project
acc init my-project

# This creates:
# - acc.yaml (project configuration)
# - .acc/policy/default.rego (starter policy)
```

#### 2. Review configuration

```bash
cat acc.yaml
```

Example `acc.yaml`:
```yaml
project:
  name: my-project

build:
  context: .
  defaultTag: latest

registry:
  default: localhost:5000

policy:
  mode: enforce  # enforce|warn

signing:
  mode: keyless  # keyless|key

sbom:
  format: spdx   # spdx|cyclonedx
```

#### 3. Build an image

```bash
# Build with SBOM generation
acc build

# Or specify a custom tag
acc build --tag myregistry.io/myapp:v1.0.0
```

The build command will:
- Build the OCI image using available tools (docker/podman/buildah)
- Generate an SBOM using syft
- Store artifacts in `.acc/sbom/`

#### 4. Verify compliance

```bash
# Verify SBOM and policy compliance
acc verify

# JSON output
acc verify --json
```

Verification checks:
- SBOM presence
- Policy compliance (using Rego policies in `.acc/policy/`)
- Attestations (for promotion workflows)

#### 5. Run workload (with verification gate)

```bash
# Run with verification - will fail if verification fails
acc run myimage:latest

# Run with custom security settings
acc run myimage:latest --user 1000 --network bridge --read-only

# Run with specific capabilities
acc run myimage:latest --cap-add NET_ADMIN
```

**Important**: `acc run` always verifies before execution. If verification fails, the workload will NOT run.

## Commands

| Command | Description |
|---------|-------------|
| `init` | Initialize a new acc project |
| `build` | Build OCI image with SBOM generation |
| `verify` | Verify SBOM, policy compliance, and attestations |
| `run` | Verify and run workload locally with security defaults |
| `inspect` | Inspect artifact trust summary with verification status |
| `attest` | Create attestation for artifact with build metadata |
| `push` | Verify and push verified artifacts to registry |
| `promote` | Re-verify and promote workload to environment |
| `policy explain` | Explain last verification decision |
| `upgrade` | Upgrade acc to the latest version with checksum verification |
| `config` | Get or set configuration values (coming soon) |
| `login` | Authenticate to registries (coming soon) |
| `version` | Print version information |

## Global Flags

```
--color string       Colorize output (auto|always|never) [default: auto]
--json              Output in JSON format
--quiet, -q         Suppress non-critical output
--no-emoji          Disable emoji in output
--policy-pack path  Path to policy pack
--config path       Path to config file
```

## Upgrade

`acc` includes built-in self-update functionality with cryptographic verification to ensure you're always running the latest stable release.

### Upgrade to Latest Version

```bash
# Upgrade to the latest stable release
acc upgrade
```

**What happens:**
1. Fetches latest release information from GitHub
2. Checks if you're already running the latest version
3. Downloads the appropriate binary for your OS/ARCH
4. Verifies SHA256 checksum against official checksums.txt
5. Atomically replaces the current binary (with backup on Unix)
6. Displays upgrade summary with version and checksum

**Output:**
```
Current version: v0.1.5
Target version:  v0.1.6
Asset:           acc_0.1.6_linux_amd64.tar.gz
Checksum:        a1b2c3d4e5f6...
Installed to:    /usr/local/bin/acc

Successfully upgraded from v0.1.5 to v0.1.6
```

### Upgrade to Specific Version

```bash
# Install a specific version
acc upgrade --version v0.1.5

# Or without the 'v' prefix
acc upgrade --version 0.1.5
```

Use this to:
- Pin to a known-good version in CI/CD
- Downgrade to a previous version if needed
- Test pre-release versions

### Dry Run

Preview what would happen without actually downloading or installing:

```bash
acc upgrade --dry-run

# Example output:
# Would upgrade from v0.1.5 to v0.1.6 using acc_0.1.6_linux_amd64.tar.gz
```

### JSON Output

For automation and CI/CD integration:

```bash
acc upgrade --json
```

**Output:**
```json
{
  "currentVersion": "v0.1.5",
  "targetVersion": "v0.1.6",
  "updated": true,
  "message": "Successfully upgraded from v0.1.5 to v0.1.6",
  "assetName": "acc_0.1.6_linux_amd64.tar.gz",
  "checksum": "a1b2c3d4e5f67890...",
  "installPath": "/usr/local/bin/acc"
}
```

### Platform Support

The upgrade command automatically detects your platform and downloads the correct binary:

| OS | Architecture | Asset Pattern |
|----|--------------|---------------|
| Linux | amd64 | `acc_<version>_linux_amd64.tar.gz` |
| Linux | arm64 | `acc_<version>_linux_arm64.tar.gz` |
| macOS | amd64 (Intel) | `acc_<version>_darwin_amd64.tar.gz` |
| macOS | arm64 (Apple Silicon) | `acc_<version>_darwin_arm64.tar.gz` |
| Windows | amd64 | `acc_<version>_windows_amd64.zip` |

### Windows Special Handling

On Windows, running executables cannot be replaced directly due to file locking. The upgrade command handles this by:

1. Downloading the new version to `acc.new.exe`
2. Providing manual replacement instructions:

```
Windows binary downloaded to: C:\path\to\acc.new.exe

To complete upgrade:
1. Close this terminal
2. Rename acc.exe to acc.exe.old
3. Rename acc.new.exe to acc.exe
4. Delete acc.exe.old
```

### Security

The upgrade process includes multiple security checks:

- **Official sources only** - Downloads from `github.com/cloudcwfranck/acc` releases
- **SHA256 verification** - All downloads verified against official checksums.txt
- **Checksum mismatch = abort** - Installation blocked on verification failure
- **Download failure = abort** - No partial or corrupted updates
- **Atomic replacement** - Unix systems use atomic rename (non-Windows)
- **Backup/rollback** - Failed installations restore previous binary

### Already Up-to-Date

If you're already running the latest version:

```bash
$ acc upgrade
Already up-to-date (version v0.1.6)
```

Exit code is `0` (success) when already up-to-date.

### CI/CD Usage

Pin versions in CI/CD pipelines for reproducibility:

```yaml
# GitHub Actions example
- name: Install acc
  run: |
    curl -sSfL https://github.com/cloudcwfranck/acc/releases/download/v0.1.6/acc_0.1.6_linux_amd64.tar.gz | tar xz
    chmod +x acc
    sudo mv acc /usr/local/bin/

# Or use acc upgrade for latest
- name: Upgrade acc
  run: acc upgrade --version 0.1.6
```

### Troubleshooting

**Issue: "checksum mismatch"**
- The downloaded binary's checksum doesn't match official checksums.txt
- This could indicate network corruption or a compromised download
- Solution: Retry the upgrade, check network connection

**Issue: "no release asset found"**
- Your OS/ARCH combination doesn't have a pre-built binary
- Solution: Build from source (see Installation section)

**Issue: Permission denied (Unix)**
- The binary is installed in a protected directory (e.g., `/usr/local/bin/`)
- Solution: Run with sudo: `sudo acc upgrade`

**Issue: Cannot replace running executable (Windows)**
- Expected behavior on Windows
- Solution: Follow the manual replacement instructions provided

## Security Model

### Verification Chain

1. **Build** → OCI artifact + SBOM
2. **Verify** → Policy evaluation + SBOM check + state persistence
3. **Inspect** → Trust summary with verification status
4. **Attest** → Cryptographic attestation of verification results
5. **Push** → Push verified artifacts to registry (verification gated)
6. **Promote** → Re-verify and promote to environment (verification gated)
7. **Run** → Execute workload locally (verification gated)

### Runtime Security Defaults

When using `acc run`, the following security defaults are applied:

- **Network isolation** - `--network none` by default
- **Capability dropping** - All Linux capabilities dropped by default
- **No new privileges** - Prevents privilege escalation
- **Optional read-only root** - Use `--read-only` flag

### Policy Enforcement

Policies are written in Rego and stored in `.acc/policy/`. The default policy enforces:

- No root user execution
- SBOM required for all builds
- Attestations required for promotion

To customize policies, edit `.acc/policy/default.rego` or add new `.rego` files.

## Exit Codes

- `0` - Success
- `1` - Failure / Blocked
- `2` - Warnings (allowed in warn mode)

## Examples

### Build and verify a project

```bash
# Initialize
acc init web-app

# Add a Dockerfile to your project
cat > Dockerfile <<EOF
FROM alpine:latest
RUN apk add --no-cache nginx
USER nginx
EXPOSE 8080
EOF

# Build with SBOM
acc build --tag myapp:latest

# Verify
acc verify
```

### Run with custom security settings

```bash
# Run with bridge network and specific user
acc run myapp:latest --network bridge --user nginx

# Run with read-only filesystem
acc run myapp:latest --read-only

# Run with specific capabilities
acc run myapp:latest --cap-add NET_BIND_SERVICE --user www-data
```

### JSON output for CI/CD

```bash
# Initialize with JSON output
acc init --json my-project

# Build with JSON output
acc build --json --tag myapp:latest

# Verify with JSON output
acc verify --json
```

### Inspect artifact trust

```bash
# Inspect an image to see trust summary
acc inspect myapp:latest

# View trust summary with JSON output
acc inspect myapp:latest --json

# Shows:
# - Image digest and reference
# - SBOM presence and location
# - Attestations found
# - Last verification status
# - Policy mode and waivers
```

### Create attestations

Attestations capture verification results as deterministic, auditable artifacts:

```bash
# First, build and verify the image
acc build --tag myapp:latest
acc verify myapp:latest

# Inspect trust summary
acc inspect myapp:latest

# Create attestation (requires verification state)
acc attest myapp:latest

# View attestation in JSON
acc attest myapp:latest --json
```

**How attestations work:**

1. **Requires verification state** - `acc attest` will fail if `.acc/state/last_verify.json` doesn't exist
2. **Image reference validation** - Ensures the image matches the last verified image
3. **Canonical hashing** - Creates deterministic hash of verification results with sorted violations
4. **Structured storage** - Saves to `.acc/attestations/<image>/<timestamp>-attestation.json`
5. **State tracking** - Updates `.acc/state/last_attestation.json` pointer

**Attestation schema:**

```json
{
  "schemaVersion": "v0.1",
  "command": "attest",
  "timestamp": "2025-01-15T10:30:00Z",
  "subject": {
    "imageRef": "myapp:latest",
    "imageDigest": "sha256:abc123..."
  },
  "evidence": {
    "sbomRef": ".acc/sbom/myapp-latest.spdx.json",
    "policyPack": ".acc/policy",
    "policyMode": "enforce",
    "verificationStatus": "pass",
    "verificationResultsHash": "sha256:def456..."
  },
  "metadata": {
    "tool": "acc",
    "toolVersion": "v0.1.0",
    "gitCommit": "abc123def"
  }
}
```

The `verificationResultsHash` is computed using canonical JSON ordering, ensuring that identical verification results always produce the same hash regardless of field order.

### Push verified artifacts

Push images to registries with verification gates:

```bash
# First, build and verify the image
acc build --tag registry.io/myapp:v1.0.0
acc verify registry.io/myapp:v1.0.0

# Push only if verification passed
acc push registry.io/myapp:v1.0.0

# View push result in JSON
acc push registry.io/myapp:v1.0.0 --json
```

**How push works:**

1. **Requires verification state** - `acc push` will fail if `.acc/state/last_verify.json` doesn't exist
2. **Blocks failed verification** - Cannot push if last verification status is "fail"
3. **Image reference validation** - Ensures the image matches the last verified digest
4. **Attestation reference** - If attestation exists, includes reference in output
5. **Tool detection** - Uses nerdctl, docker, or oras (in that order)

**Push workflow:**

```bash
# Complete verified push workflow
acc build --tag myregistry.io/myapp:v1.0.0
acc verify myregistry.io/myapp:v1.0.0
acc inspect myregistry.io/myapp:v1.0.0
acc attest myregistry.io/myapp:v1.0.0
acc push myregistry.io/myapp:v1.0.0
```

This ensures that only verified, policy-compliant workloads with attestations can be pushed to registries.

### Promote workloads to environments

```bash
# Promote to production (requires verification to pass)
acc promote myapp:dev --to prod

# Promotion:
# 1. Re-verifies with prod-specific policy
# 2. Blocks if verification fails
# 3. Re-tags image without rebuild
# 4. Verifies digest unchanged
```

### Environment-specific configuration

Add to `acc.yaml`:

```yaml
environments:
  prod:
    policy:
      mode: enforce
    registry:
      default: prod.registry.io
  staging:
    policy:
      mode: warn
    registry:
      default: staging.registry.io
```

### Explain policy decisions

```bash
# View explanation of last verification
acc policy explain

# Shows:
# - Image and timestamp
# - Pass/fail status
# - Violations with remediation
# - Warnings
# - Policy decision details

# JSON output for automation
acc policy explain --json
```

### Testing policy failures

See `examples/intentional-failure/` for a Dockerfile that demonstrates verification gating by intentionally violating security policies.

## What acc Does NOT Do

Per the design specification, `acc` explicitly does NOT:

- Provide interactive shells into containers
- Execute into running workloads
- Perform runtime EDR/monitoring
- Perform SAST/DAST scanning
- Scan for secrets
- Manage Kubernetes clusters

`acc` focuses exclusively on supply chain security and workload trust.

## Development

### Running tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/config -v
```

### Building

```bash
# Build for current platform
go build -o acc ./cmd/acc

# Build with version info
go build -ldflags "-X main.version=v0.1.0 -X main.commit=$(git rev-parse HEAD) -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o acc ./cmd/acc
```

## Documentation

See [AGENTS.md](./AGENTS.md) for the complete specification and design principles.

## License

See LICENSE file for details.

## Contributing

Contributions are welcome! Please ensure:

- All tests pass (`go test ./...`)
- Code is formatted (`gofmt`)
- Security principles are maintained
- No bypass mechanisms are added

## Support

For issues and feature requests, please open an issue on GitHub.
