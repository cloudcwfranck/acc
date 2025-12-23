#!/usr/bin/env bash
# demo/deploy-to-site.sh - Deploy demo recording to website
# Run this after recording the v2 demo in a Docker environment

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
V2_CAST="${V2_CAST:-$SCRIPT_DIR/demo-v2.cast}"
SITE_CAST="$REPO_ROOT/site/public/demo/demo.cast"

echo "========================================="
echo "acc Demo Deployment to Website"
echo "========================================="
echo ""

# Check if v2 cast exists
if [ ! -f "$V2_CAST" ]; then
    echo "❌ Error: $V2_CAST not found"
    echo ""
    echo "You need to record the demo first:"
    echo "  cd $REPO_ROOT"
    echo "  bash demo/record-v2.sh"
    echo ""
    echo "Or use an existing demo as fallback:"
    echo "  cp docs/demo/acc.cast site/public/demo/demo.cast"
    exit 1
fi

# Verify it's a valid asciinema file
if ! head -1 "$V2_CAST" | grep -q '"version"'; then
    echo "❌ Error: $V2_CAST doesn't appear to be a valid asciinema file"
    exit 1
fi

# Get recording stats
lines=$(wc -l < "$V2_CAST")
duration=$(tail -1 "$V2_CAST" | grep -oE '^\[[0-9.]+' | tr -d '[' || echo "unknown")

echo "Source: $V2_CAST"
echo "Lines: $lines"
echo "Duration: ${duration}s"
echo ""

# Copy to website
echo "Deploying to website..."
cp "$V2_CAST" "$SITE_CAST"

echo "✓ Copied to: $SITE_CAST"
echo ""
echo "Next steps:"
echo "1. Preview locally:"
echo "   cd site && npm run dev"
echo "   Visit http://localhost:3000"
echo ""
echo "2. Commit the new demo:"
echo "   git add site/public/demo/demo.cast"
echo "   git commit -m 'feat(demo): Update to production v2 recording'"
echo "   git push"
echo ""
echo "Demo deployed! 🚀"
