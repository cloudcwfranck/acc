# acc Interactive Demo (Production Version)

**60-90 second, 9-command demo proving:** "acc is a policy verification CLI that turns controls into deterministic, explainable results for CI/CD gates."

## What This Demo Shows

The demo executes exactly **9 commands** in a colored terminal with `csengineering$` prompt:

1. **`acc version`** - Proves versioned, deterministic tool
2. **`acc init demo-project`** - Creates policy baseline
3. **`acc build demo-app:ok`** - Builds PASSING workload + SBOM (non-root user)
4. **`acc verify --json demo-app:ok | jq '.status, .sbomPresent'`** - Verify PASS, show key fields
5. **`echo $?`** - Proves exit code 0 (CI gate PASS)
6. **`acc build demo-app:root`** - Builds FAILING workload (runs as root)
7. **`acc verify demo-app:root`** - Verify FAIL (exit code 1, CI gate blocks)
8. **`acc policy explain --json | jq ...`** - Explains violation (no-root-user)
9. **`acc attest demo-app:ok`** - Creates attestation after re-verifying PASS

## Files

### Demo Scripts (v2)

- **`demo-script-v2.sh`** - The actual 9 commands executed in recording
- **`run-v2.sh`** - Validation script (ensures all 9 commands work correctly)
- **`record-v2.sh`** - Recording orchestration with asciinema

### Test Images

- **`Dockerfile.ok`** - PASSING image (non-root user, uid 1000)
- **`Dockerfile.root`** - FAILING image (runs as root - violates policy)
- **`app.txt`** - Dummy content file

## Prerequisites

**For validation/recording:**
- Docker (or Podman/Buildah)
- jq
- asciinema (for recording only)
- acc binary built (`go build -o ./acc ./cmd/acc`)

**Why Docker?**
The demo builds local container images to prove policy gates work. No network/registry required.

## Usage

### 1. Validate (Test All 9 Commands)

```bash
cd /path/to/acc
bash demo/run-v2.sh
```

**What it does:**
- Builds `./acc` if needed
- Creates temp workdir
- Executes all 9 commands
- Validates exit codes + output format
- Checks:
  - ✓ version displays correctly
  - ✓ init creates .acc/ and acc.yaml
  - ✓ build generates SBOM
  - ✓ verify PASS returns exit 0, status='pass', sbomPresent=true
  - ✓ verify FAIL returns exit 1
  - ✓ policy explain shows violation details
  - ✓ attest creates attestation

**Expected output:**
```
✓ Working directory: /tmp/acc-demo-validate-...
========================================
COMMAND 1: acc version
========================================
✓ version: exit 0
✓ version: includes 'acc version'
...
========================================
SUMMARY
========================================
✓ All 9 commands validated successfully!
```

### 2. Record (Create asciinema Recording)

```bash
cd /path/to/acc
bash demo/record-v2.sh
```

**What it does:**
- Runs preflight validation
- Records with asciinema (100x28 terminal)
- Colored prompt: `csengineering$` in cyan
- Auto-executes all 9 commands with readable pacing
- Outputs `demo/demo-v2.cast`

**Recording specs:**
- Duration: 60-90 seconds
- Prompt: `csengineering$` (colored)
- Theme: Dark with colored success/fail markers
- Commands shown: Exactly 9
- Deterministic: Same input → same output

### 3. Preview Recording

```bash
asciinema play demo/demo-v2.cast
```

### 4. Upload to asciinema.org

```bash
asciinema upload demo/demo-v2.cast
# Copy the ID (e.g., 'abc123')
```

### 5. Integrate into Website

**Option A: Use asciinema.org (recommended)**
```bash
# Configure website
echo 'NEXT_PUBLIC_ASCIINEMA_ID=abc123' >> site/.env.local

# Start dev server
cd site
npm install
npm run dev
# Visit http://localhost:3000
```

**Option B: Use local cast file**
```bash
# Copy recording to website public directory
cp demo/demo-v2.cast site/public/demo/demo.cast

# Start dev server (will use local file)
cd site
npm install
npm run dev
```

## Demo Guarantees

### Determinism
- ✅ No network access required
- ✅ No timestamps in validation
- ✅ Local Docker builds only (alpine:3.19)
- ✅ Same commands → same results

### Contract Compliance
- ✅ Exit codes: 0=pass, 1=fail, 2=unknown
- ✅ JSON schema: `.status`, `.sbomPresent`, `.policyResult.violations[]`
- ✅ Explainability: Every failure includes rule name + message
- ✅ Attestation safety: Digest-based, mismatch-protected

### Security
- ✅ No secrets, tokens, or user info
- ✅ Minimal images (alpine base)
- ✅ Policy violations demonstrated safely

## CI Integration

The validation script is designed to run in GitHub Actions:

```yaml
- name: Validate demo
  run: bash demo/run-v2.sh
  env:
    ACC_BIN: ${{ github.workspace }}/acc
```

See `.github/workflows/demo.yml` for full integration.

## Troubleshooting

### Error: "no OCI build tool found"
**Solution:** Install Docker, Podman, or Buildah
```bash
# macOS
brew install docker

# Ubuntu/Debian
sudo apt-get install docker.io

# Or use Podman
sudo apt-get install podman
```

### Error: "asciinema not found"
**Solution:** Install asciinema
```bash
pip install asciinema
# or
brew install asciinema
```

### Demo validation fails
1. Ensure acc builds: `go build -o ./acc ./cmd/acc`
2. Ensure Docker works: `docker run --rm alpine:3.19 echo test`
3. Run with debug: `bash -x demo/run-v2.sh`

### Recording doesn't match expectations
- Check terminal size: `echo $COLUMNS x $LINES` (should be 100x28)
- Check prompt: `echo $PS1` (should show `csengineering$`)
- Re-run validation first: `bash demo/run-v2.sh`

## Comparison: v1 vs v2

| Feature | v1 (original) | v2 (production) |
|---------|---------------|-----------------|
| Commands shown | 6-8 (variable) | **Exactly 9** |
| Prompt | `$` | **`csengineering$`** (colored) |
| Duration | Variable | **60-90 seconds** |
| Storyline | Basic workflow | **Full CI/CD cycle** |
| Exit codes | Partial | **All exit codes shown** |
| Explainability | Limited | **Policy explain integrated** |
| Validation | Basic | **9-step comprehensive** |

## Contributing

To update the demo:

1. Modify `demo-script-v2.sh` (keep exactly 9 commands)
2. Update validation in `run-v2.sh`
3. Test: `bash demo/run-v2.sh`
4. Record: `bash demo/record-v2.sh`
5. Verify duration: `asciinema play demo/demo-v2.cast` (should be 60-90s)
6. Update this README if needed

## Questions?

See main README.md or run `./acc --help` for command details.
