# acc Interactive Demo

This directory contains the infrastructure for creating and validating deterministic terminal demos of acc.

## Prerequisites

- **Docker** - For building container images
- **jq** - For JSON parsing and validation
- **OPA** (v0.66.0) - Policy engine
- **syft** - SBOM generation
- **asciinema** (for recording only) - Install with `pip install asciinema`

## Files

- `Dockerfile.ok` - Passing image (non-root user)
- `Dockerfile.root` - Failing image (runs as root)
- `app.txt` - Demo application placeholder
- `run.sh` - Validation script (CI gate)
- `record.sh` - Recording script
- `demo-script.sh` - The actual demo commands (executed inside asciinema)
- `demo.cast` - The recorded demo (generated)

## Usage

### 1. Validate Demo (CI Gate)

This validates that the demo workflow works correctly:

```bash
cd /path/to/acc
bash demo/run.sh
```

**What it validates:**
- `acc init` creates `.acc/`, `.acc/profiles/`, and `acc.yaml`
- `acc verify` on passing image returns exit 0 and status "pass"
- `acc verify` on failing image returns exit 1 and status "fail"
- Policy violations include the `no-root-user` rule
- `acc policy explain` returns JSON with `.result.input`
- `acc attest` enforces digest-mismatch safety
- `acc trust status` for never-verified image returns exit 2 and includes `sbomPresent` field
- `acc trust verify` (v0.3.0) validates attestations

**Exit codes:**
- `0` - All validations passed
- `1` - One or more validations failed (workdir preserved for debugging)

### 2. Record Demo

This creates the `demo.cast` recording:

```bash
cd /path/to/acc
bash demo/record.sh
```

**Recording details:**
- Terminal size: 100x30
- Duration: ~60-90 seconds
- Commands shown: 6 core workflows
- Clean prompt (PS1='$ ')
- No color output (NO_COLOR=1)

**Preview the recording:**
```bash
asciinema play demo/demo.cast
```

### 3. Publish to asciinema.org

Upload the recording to get an embeddable ID:

```bash
asciinema upload demo/demo.cast
```

This will output a URL like `https://asciinema.org/a/ABC123`. The ID is `ABC123`.

### 4. Preview in Website

#### Option A: Using asciinema.org ID

1. Create `site/.env.local`:
   ```bash
   NEXT_PUBLIC_ASCIINEMA_ID=ABC123
   ```

2. Start the site:
   ```bash
   cd site
   npm install
   npm run dev
   ```

3. Open http://localhost:3000

#### Option B: Using local cast file

1. Copy the cast file:
   ```bash
   mkdir -p site/public/demo
   cp demo/demo.cast site/public/demo/
   ```

2. Start the site (without env var):
   ```bash
   cd site
   npm install
   npm run dev
   ```

3. Open http://localhost:3000

The demo player will automatically detect which source to use.

## Demo Flow

The recorded demo shows:

1. **Version** - `acc version` shows the CLI version
2. **Init** - `acc init demo-project` creates project structure
3. **PASS verification** - Build and verify non-root image (exit 0)
4. **FAIL verification** - Build and verify root image (exit 1)
5. **Explain violations** - Show the `no-root-user` policy violation
6. **Machine-readable output** - Show JSON output with `.status` field

## Contract Guarantees

All commands follow the testing contract in `docs/testing-contract.md`:

- **Exit codes**: 0=pass, 1=fail, 2=cannot complete
- **JSON output**: Stable schema with required fields
- **Deterministic**: Same input → same output (no timestamps in comparisons)
- **Local-only**: No network access required

## CI Integration

The `run.sh` script is designed to run in CI as a gate:

```yaml
# .github/workflows/demo.yml
- name: Validate demo
  run: bash demo/run.sh
```

On releases, you can optionally:
- Upload `demo.cast` as a release artifact
- Auto-publish to asciinema.org (requires API key)
- Update website with new demo ID

## Troubleshooting

**Demo validation fails:**
- Check that docker daemon is running
- Ensure OPA v0.66.0 is installed: `opa version`
- Ensure syft is installed: `syft version`
- Run with preserved workdir: `bash demo/run.sh` (workdir printed on failure)

**Recording fails:**
- Install asciinema: `pip install asciinema`
- Run preflight: `bash demo/run.sh` (must pass first)
- Check terminal size: Recording uses 100x30

**Website embed not showing:**
- Check browser console for errors
- Verify `.env.local` has correct ID (if using asciinema.org)
- Verify `demo.cast` exists in `site/public/demo/` (if using local)
- Restart dev server after adding env vars

## Updating the Demo

1. Modify `demo-script.sh` to change the recorded commands
2. Run validation: `bash demo/run.sh`
3. Re-record: `bash demo/record.sh`
4. Preview: `asciinema play demo/demo.cast`
5. Publish and update website

Keep the demo under 90 seconds and show no more than 6 core commands.
