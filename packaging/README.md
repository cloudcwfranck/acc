# acc Distribution Packaging

This directory contains packaging metadata for distributing acc through various package managers.

## Homebrew (macOS/Linux)

**Status:** Draft formula ready, NOT published yet

**Location:** `packaging/homebrew/acc.rb`

### How to Publish to Homebrew

**Option 1: Create a Homebrew Tap (Recommended for v0)**

1. Create a new repository: `homebrew-acc`
2. Copy `packaging/homebrew/acc.rb` to `Formula/acc.rb`
3. After releasing v0.1.0, update SHA256 checksums in the formula:
   ```bash
   # Download checksums from release
   curl -L https://github.com/cloudcwfranck/acc/releases/download/v0.1.0/checksums.txt

   # Extract checksums for each platform and update acc.rb
   ```
4. Commit and push to GitHub
5. Users can install with:
   ```bash
   brew tap cloudcwfranck/acc
   brew install acc
   ```

**Option 2: Submit to Homebrew Core (For v1.0+)**

Once acc reaches v1.0.0 and has proven stability:

1. Fork https://github.com/Homebrew/homebrew-core
2. Add `Formula/acc.rb` with actual checksums
3. Test thoroughly with `brew install --build-from-source ./Formula/acc.rb`
4. Run `brew audit --strict --online acc`
5. Submit PR to homebrew-core

**Current Requirements:**
- Homebrew core requires projects to be "notable" (high usage, stable)
- v0.x releases should use a tap (cloudcwfranck/homebrew-acc)
- v1.0+ can be submitted to core

### Updating the Formula for New Releases

After cutting a new release (e.g., v0.1.1):

1. Update version number in `acc.rb`
2. Download `checksums.txt` from GitHub Releases
3. Replace placeholder checksums with actual values from `checksums.txt`:
   - `acc_X.Y.Z_darwin_arm64.tar.gz` → darwin arm64 checksum
   - `acc_X.Y.Z_darwin_amd64.tar.gz` → darwin amd64 checksum
   - `acc_X.Y.Z_linux_arm64.tar.gz` → linux arm64 checksum
   - `acc_X.Y.Z_linux_amd64.tar.gz` → linux amd64 checksum
4. Test locally: `brew install --build-from-source ./Formula/acc.rb`
5. Commit and push to tap

### Testing the Formula Locally

```bash
# Install from local formula
brew install --build-from-source packaging/homebrew/acc.rb

# Test the formula
brew test acc

# Audit the formula
brew audit --strict packaging/homebrew/acc.rb

# Uninstall
brew uninstall acc
```

## Container Images (Docker/Podman/OCI)

**Status:** Dockerfile ready, NOT published yet

**Location:** `../Dockerfile.release`

### How to Build and Publish

See detailed instructions in `Dockerfile.release` header.

**Quick Start:**
```bash
# 1. Download release binary
VERSION=0.1.0
curl -LO "https://github.com/cloudcwfranck/acc/releases/download/v${VERSION}/acc_${VERSION}_linux_amd64.tar.gz"
curl -LO "https://github.com/cloudcwfranck/acc/releases/download/v${VERSION}/checksums.txt"

# 2. Verify checksum
sha256sum -c checksums.txt --ignore-missing

# 3. Extract binary
tar -xzf "acc_${VERSION}_linux_amd64.tar.gz"

# 4. Build image
docker build -f Dockerfile.release --build-arg BINARY=acc-linux-amd64 -t ghcr.io/cloudcwfranck/acc:${VERSION} .
docker tag ghcr.io/cloudcwfranck/acc:${VERSION} ghcr.io/cloudcwfranck/acc:latest

# 5. Test image
docker run --rm ghcr.io/cloudcwfranck/acc:${VERSION} version

# 6. Push to GHCR (requires authentication)
echo $GITHUB_TOKEN | docker login ghcr.io -u cloudcwfranck --password-stdin
docker push ghcr.io/cloudcwfranck/acc:${VERSION}
docker push ghcr.io/cloudcwfranck/acc:latest
```

**Multi-Architecture Images:**

For publishing both linux/amd64 and linux/arm64:

```bash
# Download both binaries
VERSION=0.1.0
curl -LO "https://github.com/cloudcwfranck/acc/releases/download/v${VERSION}/acc_${VERSION}_linux_amd64.tar.gz"
curl -LO "https://github.com/cloudcwfranck/acc/releases/download/v${VERSION}/acc_${VERSION}_linux_arm64.tar.gz"
tar -xzf "acc_${VERSION}_linux_amd64.tar.gz"
tar -xzf "acc_${VERSION}_linux_arm64.tar.gz"

# Create buildx builder
docker buildx create --name acc-builder --use

# Build and push multi-arch image
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg BINARY=acc-linux-amd64 \
  -f Dockerfile.release \
  -t ghcr.io/cloudcwfranck/acc:${VERSION} \
  -t ghcr.io/cloudcwfranck/acc:latest \
  --push \
  .
```

**Using the Container Image:**

```bash
# Run acc in container
docker run --rm ghcr.io/cloudcwfranck/acc:latest version

# Run with docker socket mounted (for build/verify)
docker run --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $(pwd):/workspace \
  ghcr.io/cloudcwfranck/acc:latest verify myimage:latest

# Use in CI/CD
# GitHub Actions example:
#   - uses: docker://ghcr.io/cloudcwfranck/acc:0.1.0
#     with:
#       args: verify myimage:latest
```

**Image Properties:**
- Base: Alpine Linux 3.19
- Size: ~25MB (Alpine + acc binary)
- User: Non-root (uid=1000, gid=1000)
- Entrypoint: `/usr/local/bin/acc`
- Working directory: `/workspace`

## Other Package Managers

### NOT Supported (Explicit Non-Goals)

- ❌ **PyPI** - acc is not a Python package
- ❌ **npm** - acc is not a Node.js package
- ❌ **apt/yum/dnf** - Use GitHub Releases or Homebrew instead
- ❌ **Snap/Flatpak** - Not applicable for CLI tools in this domain
- ❌ **Chocolatey (Windows)** - May consider for v1.0+
- ❌ **Winget (Windows)** - May consider for v1.0+

### Might Support in Future (v1.0+)

- 🤔 **asdf** - Version manager support
- 🤔 **Scoop (Windows)** - Windows package manager
- 🤔 **MacPorts** - Alternative to Homebrew on macOS

## Distribution Checklist

Before publishing to any package manager:

- [ ] v0.1.0 is released and tagged
- [ ] GitHub Release artifacts are published
- [ ] `checksums.txt` is verified
- [ ] All platform binaries tested
- [ ] Formula/Dockerfile has actual checksums (not placeholders)
- [ ] Installation tested on target platform
- [ ] `acc version` shows correct version

## Questions?

See `docs/releasing.md` for the complete release process.
