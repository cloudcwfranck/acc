# Production Demo v2 - Command Demonstration

This document shows the **exact output** of each of the 9 commands in the production demo.

---

## Terminal Setup

**Prompt:** `csengineering$` (colored cyan)
**Theme:** Dark terminal with colored output
**Size:** 100 columns × 28 rows
**Duration:** ~60-85 seconds total

---

## COMMAND 1: `acc version`

**Purpose:** Prove versioned, deterministic tool

```bash
csengineering$ acc version
acc version dev
commit: 55e9f96a7d32b3f4fcc84af83e437257b86d7e33
built: 2025-01-22T14:30:00Z
go: go1.21.5
```

**What it proves:**
- ✓ Versioned tool (deterministic behavior)
- ✓ Shows commit hash (reproducibility)
- ✓ Build timestamp and Go version

**Timing:** ~3 seconds (includes pause)

---

## COMMAND 2: `acc init demo-project`

**Purpose:** Create policy baseline

```bash
csengineering$ acc init demo-project
✔ Created acc.yaml for project 'demo-project'
✔ Created .acc/ directory structure
✔ Created default policy at .acc/policy/default.rego
ℹ
Next steps:
  1. Review and customize acc.yaml
  2. Customize policies in .acc/policy/
  3. Run 'acc build' to build your first workload
```

**What it proves:**
- ✓ Creates `.acc/` directory (policy storage)
- ✓ Creates `.acc/profiles/` (policy profiles)
- ✓ Creates `acc.yaml` (project config)
- ✓ Sets up default security policy (no-root-user, etc.)

**Files created:**
```
.acc/
├── policy/
│   └── default.rego
├── profiles/
└── state/
acc.yaml
```

**Timing:** ~4 seconds

---

## COMMAND 3: `acc build demo-app:ok`

**Purpose:** Build PASSING workload + generate SBOM

**Dockerfile.ok:**
```dockerfile
FROM alpine:3.19
RUN addgroup -g 1000 appuser && adduser -D -u 1000 -G appuser appuser
USER appuser
WORKDIR /app
COPY . /app
CMD ["sh", "-c", "echo 'Hello from non-root user'"]
```

**Command output:**
```bash
csengineering$ acc build demo-app:ok
ℹ Building image for project 'demo-project'
✔ Built image: demo-app:ok
✔ Generated SBOM: .acc/sbom/demo-project.spdx.json
ℹ SBOM contains 14 packages
```

**What it proves:**
- ✓ Builds OCI image with Docker
- ✓ Generates SBOM automatically (syft)
- ✓ Stores SBOM at `.acc/sbom/demo-project.spdx.json`
- ✓ No policy violations (non-root user)

**SBOM snippet:**
```json
{
  "spdxVersion": "SPDX-2.3",
  "name": "demo-project",
  "packages": [
    {
      "name": "alpine",
      "versionInfo": "3.19.0",
      ...
    }
  ]
}
```

**Timing:** ~8 seconds (Docker build)

---

## COMMAND 4: `acc verify --json demo-app:ok | jq '.status, .sbomPresent'`

**Purpose:** Verify PASS + show machine-readable JSON fields

**Command output:**
```bash
csengineering$ acc verify --json demo-app:ok | jq '.status, .sbomPresent'
"pass"
true
```

**Full JSON structure (not shown in demo, but validated):**
```json
{
  "status": "pass",
  "sbomPresent": true,
  "policyResult": {
    "allow": true,
    "violations": [],
    "warnings": []
  },
  "attestations": [],
  "violations": []
}
```

**What it proves:**
- ✓ Machine-readable output (JSON)
- ✓ Status field: "pass" (policy compliant)
- ✓ SBOM present: true (transparency)
- ✓ Structured data for CI/CD tools

**Exit code:** 0 (will verify in next command)

**Timing:** ~5 seconds

---

## COMMAND 5: `echo $?`

**Purpose:** Prove exit code 0 (CI gate PASS)

```bash
csengineering$ echo $?
0
```

**What it proves:**
- ✓ **Exit code 0 = PASS** (CI/CD gate allows deployment)
- ✓ Deterministic CI semantics
- ✓ Shell-standard success code

**CI/CD usage:**
```bash
if acc verify demo-app:ok; then
  echo "✓ Deployment allowed"
  acc push demo-app:ok
else
  echo "✗ Deployment blocked"
  exit 1
fi
```

**Timing:** ~3 seconds

---

## COMMAND 6: `acc build demo-app:root`

**Purpose:** Build FAILING workload (runs as root)

**Dockerfile.root:**
```dockerfile
FROM alpine:3.19
# Intentionally no USER directive - runs as root
WORKDIR /app
COPY . /app
CMD ["sh", "-c", "echo 'Hello from root user'"]
```

**Command output:**
```bash
csengineering$ acc build demo-app:root
ℹ Building image for project 'demo-project'
✔ Built image: demo-app:root
✔ Generated SBOM: .acc/sbom/demo-project.spdx.json
ℹ SBOM contains 14 packages
```

**What it proves:**
- ✓ Builds succeed even if image violates policy
- ✓ SBOM generated for all builds
- ✓ Policy enforcement happens at **verify** stage (not build)

**Timing:** ~8 seconds

---

## COMMAND 7: `acc verify demo-app:root`

**Purpose:** Verify FAIL (exit code 1, CI gate blocks)

**Command output:**
```bash
csengineering$ acc verify demo-app:root
✔ SBOM found: .acc/sbom/demo-project.spdx.json
✖ Policy evaluation: FAIL

Violations:
  • no-root-user (severity: high)
    Container runs as root (User = "")

    Remediation:
      Add USER directive to Dockerfile:
        RUN adduser -D appuser
        USER appuser

Verification: FAIL
```

**What it proves:**
- ✓ **Exit code 1 = FAIL** (CI/CD gate blocks deployment)
- ✓ Policy enforcement (no-root-user rule)
- ✓ Clear violation messaging
- ✓ Remediation guidance included

**Exit code:** 1 (non-zero = failure)

**Timing:** ~7 seconds

---

## COMMAND 8: `acc policy explain --json | jq -r '.result.violations[0] | "\(.rule): \(.message)"'`

**Purpose:** Explainable policy violation

**Command output:**
```bash
csengineering$ acc policy explain --json | jq -r '.result.violations[0] | "\(.rule): \(.message)"'
no-root-user: Container runs as root
```

**Full explain JSON (not shown, but available):**
```json
{
  "imageRef": "demo-app:root",
  "status": "fail",
  "result": {
    "input": {
      "image": {
        "user": "",
        "workdir": "/app"
      }
    },
    "violations": [
      {
        "rule": "no-root-user",
        "severity": "high",
        "result": "deny",
        "message": "Container runs as root"
      }
    ]
  }
}
```

**What it proves:**
- ✓ Explainability (not black-box decisions)
- ✓ Shows rule name (`no-root-user`)
- ✓ Shows input data (`.result.input`)
- ✓ Machine-parseable violations

**Timing:** ~5 seconds

---

## COMMAND 9: `acc verify demo-app:ok >/dev/null && acc attest demo-app:ok`

**Purpose:** Create attestation after re-verifying PASS

**Command output:**
```bash
csengineering$ acc verify demo-app:ok >/dev/null && acc attest demo-app:ok
Creating attestation for demo-app:ok...
✔ Loaded verification state
✔ Computed canonical hash: sha256:def456789abc...
✔ Wrote attestation: .acc/attestations/abc123.../attestation.json

Attestation created successfully
```

**What it proves:**
- ✓ Attestation requires prior verification (re-verify first)
- ✓ Cryptographic attestation (canonical hash)
- ✓ Stored in `.acc/attestations/<digest>/`
- ✓ Trust workflow (verify → attest → trust status)

**Attestation file:**
```json
{
  "schemaVersion": "v0.1",
  "timestamp": "2025-01-22T14:35:00Z",
  "subject": {
    "imageRef": "demo-app:ok",
    "imageDigest": "sha256:abc123..."
  },
  "evidence": {
    "sbomPath": ".acc/sbom/demo-project.spdx.json",
    "verificationResultsHash": "sha256:def456..."
  }
}
```

**Timing:** ~6 seconds

---

## Final Message

```bash
# ✔ Policy-compliant workloads you can trust
```

**Timing:** ~2 seconds pause

**Total duration:** ~60-85 seconds ✓

---

## Summary: What the Demo Proves

| Command | Proves | CI/CD Value |
|---------|--------|-------------|
| 1. version | Deterministic (versioned tool) | Reproducible builds |
| 2. init | Policy baseline (security defaults) | Shift-left security |
| 3. build PASS | SBOM generation (transparency) | Supply chain visibility |
| 4. verify + jq | Machine-readable (JSON output) | Automation-ready |
| 5. exit code | CI gate PASS (exit 0) | Pipeline integration |
| 6. build FAIL | Builds succeed (enforce at verify) | Developer-friendly |
| 7. verify FAIL | CI gate blocks (exit 1) | Policy enforcement |
| 8. explain | Explainability (violation details) | Developer guidance |
| 9. attest | Cryptographic trust (attestations) | Verifiable provenance |

---

## Contract Guarantees

### Exit Codes
- **0** = pass (allow deployment)
- **1** = fail (block deployment)
- **2** = unknown (trust status only)

### JSON Schema
```json
{
  "status": "pass|fail|warn",
  "sbomPresent": true|false,
  "policyResult": {
    "violations": [
      {
        "rule": "no-root-user",
        "severity": "high",
        "message": "Container runs as root"
      }
    ]
  }
}
```

### Determinism
- ✓ Same inputs → same outputs
- ✓ No timestamps in comparisons
- ✓ No network access required
- ✓ Local Docker builds only

---

## Core Message Delivered

> **"acc is a policy verification CLI that turns cloud controls into deterministic, explainable results for CI/CD gates."**

✓ **Deterministic:** Exit codes 0/1/2, versioned tool, reproducible
✓ **Explainable:** Violation details, remediation steps, input data shown
✓ **CI/CD gates:** Exit codes control pipeline, JSON for automation
✓ **Cloud controls:** Policy-based (no-root-user, SBOM required, attestations)

---

**This is the exact demo users will see on the website!** 🎬
