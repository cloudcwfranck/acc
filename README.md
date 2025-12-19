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

```bash
# Clone the repository
git clone https://github.com/cloudcwfranck/acc.git
cd acc

# Build
go build -o acc ./cmd/acc

# Install (optional)
sudo mv acc /usr/local/bin/
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
| `push` | Verify and push verified artifacts (coming soon) |
| `promote` | Re-verify and promote workload (coming soon) |
| `policy` | Manage and test policies (coming soon) |
| `attest` | Attach attestations to artifacts (coming soon) |
| `inspect` | Inspect artifact trust summary (coming soon) |
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

## Security Model

### Verification Chain

1. **Build** → OCI artifact + SBOM
2. **Verify** → Policy evaluation + SBOM check
3. **Run/Push/Promote** → Gated by verification

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
