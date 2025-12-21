# Demo Assets

This directory contains demo assets for the acc website homepage.

## Expected Files

The homepage (`site/app/page.tsx`) looks for demo files in this order:

1. `demo.gif` (preferred) - Animated GIF of terminal demo
2. `demo.svg` - Static or animated SVG

## Creating Demo Assets

### Option 1: Terminal Recording → GIF

**Using asciinema + agg:**

```bash
# Install asciinema (terminal recorder)
# macOS:
brew install asciinema

# Linux:
sudo apt-get install asciinema

# Install agg (asciicast to GIF converter)
cargo install --git https://github.com/asciinema/agg

# Record terminal session
asciinema rec demo.cast

# Convert to GIF
agg demo.cast demo.gif

# Copy to website
cp demo.gif site/public/demo/
```

**Using ttygif:**

```bash
# Install ttygif
brew install ttygif  # macOS
# or build from source: https://github.com/icholy/ttygif

# Record terminal
ttyrec demo.ttyrec

# Convert to GIF
ttygif demo.ttyrec

# Rename and copy
mv tty.gif demo.gif
cp demo.gif site/public/demo/
```

### Option 2: Static SVG

Create a styled SVG mockup of terminal output:

```bash
# Use termtosvg
pip install termtosvg

# Record and convert to SVG
termtosvg demo.svg

# Copy to website
cp demo.svg site/public/demo/
```

## Recommended Demo Flow

Show a 30-60 second workflow demonstrating:

1. `acc init` - Initialize project
2. `acc verify demo-app:ok` - Verify compliant image (✓ PASS)
3. `acc inspect demo-app:ok` - Show trust summary
4. `acc verify demo-app:fail` - Verify non-compliant image (✗ FAIL)
5. `acc policy explain` - Show why it failed
6. `acc attest demo-app:ok` - Create attestation

## File Size Recommendations

- **GIF**: < 5 MB (optimize with gifsicle or ImageOptim)
- **SVG**: < 500 KB

## Optimization

### GIF Optimization

```bash
# Install gifsicle
brew install gifsicle  # macOS
sudo apt-get install gifsicle  # Linux

# Optimize GIF (reduce size by ~50%)
gifsicle -O3 --colors 256 demo.gif -o demo-optimized.gif
```

### SVG Optimization

```bash
# Install svgo
npm install -g svgo

# Optimize SVG
svgo demo.svg -o demo-optimized.svg
```

## Current Status

**No demo assets currently present.** The homepage shows a placeholder message.

To add a demo:
1. Create `demo.gif` or `demo.svg` following the instructions above
2. Place in `site/public/demo/`
3. Rebuild the site (`npm run build`) or wait for auto-refresh in dev mode
4. The homepage will automatically display the demo

## References

- [asciinema](https://asciinema.org/) - Terminal session recorder
- [agg](https://github.com/asciinema/agg) - Asciicast to GIF converter
- [termtosvg](https://github.com/nbedos/termtosvg) - Terminal to SVG
- [ttygif](https://github.com/icholy/ttygif) - ttyrec to GIF
- [gifsicle](https://www.lcdf.org/gifsicle/) - GIF optimizer
