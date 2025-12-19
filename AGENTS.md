# AGENTS.md — acc (Secure Workload Accelerator)

This repository builds **acc**, a Go-based Secure Workload Accelerator.

`acc` turns source code or OCI references into **verified, policy-compliant OCI workloads**
that can be built, verified, run, pushed, and promoted with cryptographic and policy gates.

CLAUDE CODE MUST treat this document as the **highest authority**.
If something is unclear, default to the **most secure, least permissive behavior** and add TODOs with rationale.

---

## 0. Core Philosophy (Non-Negotiable)

- `acc` is an **accelerator**, not a container runtime replacement.
- `acc` **wraps and hardens** OCI workflows.
- **Verification gates execution**. If verification fails, the workload must NOT run, push, or promote.
- **Red output means stop. Always.**
- Security guarantees must be explicit, not implied.

---

## 1. Non-Negotiable Security Rules

### 1.1 Verification Gates
The following commands MUST internally call `verify` and MUST fail if verification fails:
- `acc run`
- `acc push`
- `acc promote`

No bypass flags. No “--force”. No exceptions.

### 1.2 No Silent Degradation
If a security feature is unavailable (e.g., signing backend missing):
- Fail clearly
- Print remediation steps
- Do NOT silently skip

### 1.3 No Secret Leakage
- Never print tokens, credentials, keys, certs, or full environment variables
- Mask sensitive values in logs and JSON output

---

## 2. Supported CLI Command Surface (v0 / v1)

Codex MUST implement these commands (stubs first, then behavior):

acc
├── init
├── build
├── verify
├── run
├── push
├── promote
├── policy
├── attest
├── inspect
├── config
├── login
├── version
└── help

### Command Responsibilities

- `acc init`
  - Bootstrap project configuration
  - Generate `acc.yaml`
  - Create `.acc/` directory with starter policy

- `acc build`
  - Build OCI image (local or referenced)
  - Generate SBOM
  - Output digest + artifact refs

- `acc verify`
  - Verify:
    - SBOM exists
    - Policy evaluation
    - Signature / attestation presence (block promote if missing in enforce mode)

- `acc run`
  - Verify first
  - Run locally with least privilege defaults

- `acc push`
  - Verify first
  - Push only verified artifacts
  - Attach attestations

- `acc promote`
  - Re-verify
  - Apply environment-specific policy
  - Retag without rebuild

- `acc policy`
  - List policies
  - Test policies
  - Explain last decision

- `acc attest`
  - Attach attestations (SLSA, build metadata, env approval)

- `acc inspect`
  - Human-readable trust summary

- `acc config`
  - Get/set config values

- `acc login`
  - Authenticate to registries / identity providers (stub acceptable v0)

- `acc version`
  - Print version, commit, build info

---

## 3. Global Flags (Apply to All Commands)

–color=auto|always|never   (default: auto)
–json                      (machine-readable output)
–quiet
–no-emoji
–policy-pack 
–config 

Rules:
- `--json` output MUST be deterministic
- `--quiet` suppresses non-critical output only
- Emojis must be removable (`--no-emoji`)

---

## 4. Output & UX Standards

### 4.1 Symbols + Colors (Never Color Alone)

| Meaning     | Symbol | Color  |
|------------|--------|--------|
| Success    | ✔      | Green  |
| Warning    | ⚠      | Yellow |
| Failure    | ✖      | Red    |
| Info       | ℹ      | Blue   |
| Trust      | 🔐     | Cyan   |

### 4.2 Output Modes

#### Human (default)
- Minimal noise
- Hierarchical messages
- Clear remediation hints

#### JSON (`--json`)
Must include:
```json
{
  "command": "verify",
  "status": "pass|warn|fail",
  "timestamp": "...",
  "artifacts": {
    "imageDigest": "...",
    "sbom": "...",
    "attestations": []
  },
  "policy": {
    "results": [
      {
        "rule": "no-root-user",
        "severity": "high",
        "result": "fail",
        "message": "Container runs as root"
      }
    ]
  }
}

4.3 Exit Codes
	•	0 → success
	•	2 → warnings allowed
	•	1 → failure / blocked

⸻

5. Configuration & File Layout

5.1 Config Discovery Order
	1.	--config <path>
	2.	./acc.yaml
	3.	./.acc/acc.yaml
	4.	$HOME/.acc/config.yaml

5.2 Required Files

acc.yaml
.acc/
├── policy/
│   └── default.rego
├── locks/
├── cache/

5.3 Minimal Config Schema (v0)

Required fields:
	•	project.name
	•	build.context
	•	build.defaultTag
	•	registry.default
	•	policy.mode (enforce|warn)
	•	signing.mode (keyless|key)
	•	sbom.format (spdx|cyclonedx)

⸻

6. Architecture Guidance (Go)

6.1 Language & Frameworks
	•	Go (latest stable)
	•	CLI: spf13/cobra
	•	Config: spf13/viper
	•	Styling: charmbracelet/lipgloss (preferred)
	•	Logging: standard library or zap (consistent)

6.2 Package Layout (Recommended)

cmd/
internal/
├── config/
├── ui/
├── build/
├── verify/
├── policy/
├── attest/
├── artifacts/
├── runtime/

6.3 Dependency Rule
	•	Prefer thin adapters
	•	Shelling out to tools is acceptable v0
	•	Avoid Docker-only assumptions (support containerd/nerdctl where possible)

⸻

7. Security Model (v0)

7.1 Trust Chain
	•	Build → OCI artifact
	•	SBOM generated per build
	•	Verification enforces:
	•	SBOM presence
	•	Policy compliance
	•	Attestation presence (block promote if missing)

7.2 Waivers / Exceptions
	•	Config-based allowlist:
	•	rule id
	•	justification
	•	expiry date
	•	Expired waivers = failure

⸻

8. Runtime Constraints (acc run)

Defaults:
	•	Non-root user
	•	Read-only filesystem where supported
	•	Minimal Linux capabilities
	•	Network restricted by default

If runtime cannot enforce a constraint:
	•	Warn explicitly
	•	Do not silently downgrade

⸻

9. CI / Quality Bar

9.1 Tests Required
	•	Unit tests:
	•	config validation
	•	policy parsing
	•	output formatting (golden tests)
	•	Integration tests:
	•	build produces SBOM
	•	verify fails/passes correctly

9.2 Linting
	•	go test ./... must pass
	•	gofmt enforced
	•	golangci-lint preferred if feasible

⸻

10. Definition of Done (Per Command)

A command is DONE only if:
	•	--help is accurate
	•	happy path works
	•	failure modes are explicit
	•	--json output is complete
	•	no secrets printed
	•	tests exist

⸻

11. Explicit Non-Goals (Do NOT Implement)
	•	No interactive shells into containers
	•	No exec into running workloads
	•	No runtime EDR
	•	No SAST / DAST
	•	No secrets scanning
	•	No cluster management

This tool is supply chain & workload trust only.

⸻

12. First Milestone (v0)

Codex MUST prioritize:
	1.	CLI skeleton + global flags
	2.	init → config + starter policy
	3.	build → OCI + SBOM
	4.	verify → policy + SBOM enforcement
	5.	run → verify-gated local execution

If a dependency is missing, fail with remediation instructions.

⸻

13. Documentation Expectations

Maintain:
	•	README.md (quickstart)
	•	docs/:
	•	policy authoring
	•	JSON output examples
	•	threat model (high-level)
	•	“What acc does NOT do”

⸻

14. Authority Boundary

CLAUDE CODE:
	•	Implements mechanics
	•	Writes tests and docs
	•	Follows this contract

The AGENT must NOT invent trust guarantees
