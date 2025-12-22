# Demo Assets

This directory contains demo assets for the acc website homepage.

## Expected Files

The homepage (`site/app/page.tsx`) uses an embedded asciinema player that can load demos from:

1. **asciinema.org** (preferred) - Set `NEXT_PUBLIC_ASCIINEMA_ID` in `.env.local`
2. **Local cast file** - `demo.cast` in this directory (fallback)

## Creating the Demo

The demo is created using the infrastructure in the root `demo/` directory:

### Option 1: Record and Upload to asciinema.org (Recommended)

```bash
# From repo root
bash demo/record.sh

# Upload to asciinema.org
asciinema upload demo/demo.cast

# Copy the ID from the URL (e.g., https://asciinema.org/a/ABC123 → ID is ABC123)

# Create .env.local in site/
echo "NEXT_PUBLIC_ASCIINEMA_ID=ABC123" > site/.env.local
```

### Option 2: Use Local Cast File

```bash
# From repo root
bash demo/record.sh

# Copy cast file to this directory
cp demo/demo.cast site/public/demo/

# No .env.local needed - player auto-detects local file
```

## Recommended Demo Flow

Show a 30-60 second workflow demonstrating:

1. `acc init` - Initialize project
2. `acc verify demo-app:ok` - Verify compliant image (✓ PASS)
3. `acc inspect demo-app:ok` - Show trust summary
4. `acc verify demo-app:fail` - Verify non-compliant image (✗ FAIL)
5. `acc policy explain` - Show why it failed
6. `acc attest demo-app:ok` - Create attestation

## Prerequisites for Recording

See `demo/README.md` in the repo root for full prerequisites:

- Docker (for running the demo workflow)
- asciinema (for recording)
- OPA v0.66.0
- syft
- jq

## Demo Details

- **Format**: asciinema cast v2
- **Terminal size**: 100x30
- **Duration**: 60-90 seconds
- **Commands shown**: 6 core workflows

The demo validates deterministic exit codes and machine-readable JSON output.

## Current Status

A placeholder `demo.cast` file exists. To create the actual recording:

1. **Validate** the demo works: `bash demo/run.sh` (requires Docker)
2. **Record** the demo: `bash demo/record.sh`
3. **Publish** to asciinema.org OR copy to this directory
4. **Update** `.env.local` with ID (if using asciinema.org)

The homepage will automatically embed the demo using the asciinema player.

## References

- [asciinema](https://asciinema.org/) - Terminal session recorder
- [agg](https://github.com/asciinema/agg) - Asciicast to GIF converter
- [termtosvg](https://github.com/nbedos/termtosvg) - Terminal to SVG
- [ttygif](https://github.com/icholy/ttygif) - ttyrec to GIF
- [gifsicle](https://www.lcdf.org/gifsicle/) - GIF optimizer
