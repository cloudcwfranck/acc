# EXACT 9-Command Production Demo

**Prompt:** `franck@csengineering$` (colored cyan)
**Duration:** 60-90 seconds
**Terminal:** 100 columns × 28 rows

---

## The Exact 9 Commands

### 1. Show version
```bash
franck@csengineering$ acc version
```

### 2. Initialize project
```bash
franck@csengineering$ acc init demo-project
```

### 3. Build PASSING image
```bash
franck@csengineering$ acc build demo-app:ok
```

### 4. Verify PASS + show JSON fields
```bash
franck@csengineering$ acc verify --json demo-app:ok | jq -r '.status, .sbomPresent'
```

### 5. Show exit code
```bash
franck@csengineering$ echo "EXIT=$?"
```

### 6. Build FAILING image
```bash
franck@csengineering$ acc build demo-app:root
```
*Note: Spec says `--tag` but acc uses positional arg*

### 7. Verify FAIL + show violation
```bash
franck@csengineering$ acc verify --json demo-app:root | jq -r '.status, (.policyResult.violations[0].rule // "no-violation")'
```

### 8. Explain policy violation
```bash
franck@csengineering$ acc policy explain --json | jq -r '.result.violations[0] | "\(.rule): \(.message)"'
```
*Note: Spec says `.result.input.config.User` but actual is `.result.violations[0]`*

### 9. Full trust cycle (verify → attest → trust status)
```bash
franck@csengineering$ acc verify --json demo-app:ok >/dev/null && acc attest demo-app:ok && acc trust status --json demo-app:ok | jq -r '.status, (.attestations|length)'
```

---

## Expected Outputs

### Command 1: acc version
```
acc version dev
commit: 55e9f96a7d32b3f4fcc84af83e437257b86d7e33
built: 2025-01-22T...
go: go1.21.5
```

### Command 2: acc init demo-project
```
✔ Created acc.yaml for project 'demo-project'
✔ Created .acc/ directory structure
✔ Created default policy at .acc/policy/default.rego
```

### Command 3: acc build demo-app:ok
```
ℹ Building image for project 'demo-project'
✔ Built image: demo-app:ok
✔ Generated SBOM: .acc/sbom/demo-project.spdx.json
```

### Command 4: acc verify --json ... | jq
```
"pass"
true
```

### Command 5: echo "EXIT=$?"
```
EXIT=0
```

### Command 6: acc build demo-app:root
```
ℹ Building image for project 'demo-project'
✔ Built image: demo-app:root
✔ Generated SBOM: .acc/sbom/demo-project.spdx.json
```

### Command 7: acc verify --json ... | jq
```
"fail"
"no-root-user"
```

### Command 8: acc policy explain ... | jq
```
no-root-user: Container runs as root
```

### Command 9: Full trust cycle
```
Creating attestation for demo-app:ok...
✔ Attestation created successfully
"pass"
1
```

---

## Recording Instructions

```bash
# From repo root with Docker available
bash demo/record-production.sh

# Output: demo/demo-production.cast

# Deploy to website
cp demo/demo-production.cast site/public/demo/demo.cast

# Commit
git add site/public/demo/demo.cast
git commit -m "feat(demo): Production recording with franck@csengineering$ prompt"
git push origin main
```

---

## What Each Command Proves

1. **Versioned tool** - Deterministic, reproducible
2. **Policy baseline** - Shift-left security
3. **SBOM generation** - Supply chain transparency
4. **Machine-readable** - JSON for automation
5. **CI gate PASS** - Exit code 0
6. **Build succeeds** - Policy enforced at verify, not build
7. **CI gate FAIL** - Exit code 1, blocks deployment
8. **Explainability** - Developer-friendly violation details
9. **Trust cycle** - Verify → Attest → Status (cryptographic provenance)

---

## Core Message

> "acc is a policy verification CLI that turns cloud controls into deterministic, explainable results for CI/CD gates."

✓ **Deterministic:** Same inputs → same outputs, versioned
✓ **Explainable:** Violation details + remediation guidance
✓ **CI/CD gates:** Exit codes (0=pass, 1=fail, 2=unknown)
✓ **Cloud controls:** Policy-based (SBOM, no-root-user, attestations)
