# Releasing acc

This document describes the release process for acc, including versioning policy, changelog management, and how to cut releases.

## Table of Contents

- [Versioning Policy](#versioning-policy)
- [Changelog Guidelines](#changelog-guidelines)
- [Cutting a Release](#cutting-a-release)
- [Release Artifacts](#release-artifacts)
- [Post-Release Tasks](#post-release-tasks)
- [Starting v1.0](#starting-v10)

---

## Versioning Policy

acc follows [Semantic Versioning](https://semver.org/):

```
MAJOR.MINOR.PATCH
```

### v0 Releases (0.x.y)

During the v0 phase, the API is **not stable**:

- **PATCH version** (e.g., 0.1.1 → 0.1.2): Bug fixes only, no new features
- **MINOR version** (e.g., 0.1.0 → 0.2.0): New features, **may include breaking changes**

Breaking changes are allowed in minor versions during v0 but should be documented clearly in CHANGELOG.md.

### v1+ Releases (1.x.y)

Once we reach v1.0, the API is **stable**:

- **PATCH version** (e.g., 1.0.0 → 1.0.1): Bug fixes only, backward compatible
- **MINOR version** (e.g., 1.0.0 → 1.1.0): New features, backward compatible
- **MAJOR version** (e.g., 1.0.0 → 2.0.0): Breaking changes

### What Constitutes a Breaking Change

A breaking change is any modification that could break existing user workflows:

**Command-level:**
- Removing or renaming commands
- Changing command behavior in incompatible ways
- Removing or renaming flags
- Changing flag defaults that affect behavior

**Output-level:**
- Changes to JSON output schemas (requires `schemaVersion` bump)
- Changes to exit codes
- Removal of output formats

**File format-level:**
- Changes to `.acc/state/*.json` format
- Changes to `.acc/attestations/` format
- Changes to `.acc/waivers.yaml` format
- Changes to `acc.yaml` format (if not backward compatible)

**Not breaking:**
- Adding new commands
- Adding new flags (optional)
- Adding new fields to JSON output (backward compatible)
- Bug fixes that restore intended behavior
- Internal refactoring without user-facing changes

---

## Changelog Guidelines

We follow the [Keep a Changelog](https://keepachangelog.com/) format.

### Structure

```markdown
## [Unreleased]

### Added
- New feature descriptions

### Changed
- Changes to existing functionality

### Deprecated
- Soon-to-be-removed features

### Removed
- Removed features

### Fixed
- Bug fixes

### Security
- Security fixes and improvements

## [0.1.0] - 2025-01-19

### Added
- Initial release features...
```

### Categories

Use these categories in order:

1. **Added** - New features, commands, or capabilities
2. **Changed** - Changes to existing functionality
3. **Deprecated** - Features that will be removed in future versions
4. **Removed** - Features that have been removed
5. **Fixed** - Bug fixes
6. **Security** - Security-related changes

### Writing Good Changelog Entries

**Good:**
```markdown
### Added
- `acc push` command for pushing verified artifacts to registries with verification gates
- Waiver expiry enforcement - expired waivers now cause verification failure
- Golden tests for JSON output validation to catch schema drift
```

**Bad:**
```markdown
### Added
- Added push
- Fixed bug
- Updated code
```

**Guidelines:**
- Write in imperative mood ("Add feature" not "Added feature")
- Be specific and descriptive
- Include command names in backticks
- Explain *what* and *why*, not *how*
- Group related changes together
- Link to issues/PRs when relevant

### Updating the Changelog

**Every PR must update CHANGELOG.md** under the `[Unreleased]` section.

1. Open `CHANGELOG.md`
2. Find the `[Unreleased]` section at the top
3. Add your change under the appropriate category
4. If the category doesn't exist, add it
5. Keep entries concise but descriptive

Example workflow:

```bash
# Make your changes
git checkout -b feature/new-command

# Edit CHANGELOG.md
vim CHANGELOG.md

# Add entry under [Unreleased] -> Added:
# - `acc mycommand` for doing something useful

# Commit with changelog
git add CHANGELOG.md
git commit -m "Add mycommand for X functionality"
```

**CI will fail if CHANGELOG.md is not updated in a PR.**

---

## Cutting a Release

### Prerequisites

- You must have push access to the repository
- All tests must be passing on `main`
- CHANGELOG.md must be up to date with all unreleased changes

### Release Process

#### 1. Prepare the Changelog

Update `CHANGELOG.md` to prepare for release:

```bash
# Checkout main and pull latest
git checkout main
git pull origin main

# Edit CHANGELOG.md
vim CHANGELOG.md
```

Change the `[Unreleased]` section to the new version with date:

**Before:**
```markdown
## [Unreleased]

### Added
- New feature X
- New feature Y
```

**After:**
```markdown
## [Unreleased]

### Added
- Nothing yet

## [0.2.0] - 2025-01-20

### Added
- New feature X
- New feature Y
```

Update the comparison links at the bottom:

```markdown
[Unreleased]: https://github.com/cloudcwfranck/acc/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/cloudcwfranck/acc/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/cloudcwfranck/acc/releases/tag/v0.1.0
```

Commit the changelog update:

```bash
git add CHANGELOG.md
git commit -m "Release v0.2.0"
git push origin main
```

#### 2. Create and Push the Tag

```bash
# Create an annotated tag
git tag -a v0.2.0 -m "Release v0.2.0"

# Push the tag to GitHub
git push origin v0.2.0
```

**Important:** Use annotated tags (`-a` flag), not lightweight tags.

#### 3. Wait for CI

The release workflow will automatically:
1. Run all tests
2. Build cross-platform binaries (Linux, macOS, Windows)
3. Package them as tar.gz/zip
4. Generate SHA256 checksums
5. Extract release notes from CHANGELOG.md
6. Create a GitHub Release
7. Upload all artifacts

Monitor the workflow: https://github.com/cloudcwfranck/acc/actions

#### 4. Verify the Release

Once the workflow completes:

1. Visit: https://github.com/cloudcwfranck/acc/releases
2. Verify the release is published
3. Check that all artifacts are attached:
   - `acc-0.2.0-linux-amd64.tar.gz`
   - `acc-0.2.0-linux-arm64.tar.gz`
   - `acc-0.2.0-darwin-amd64.tar.gz`
   - `acc-0.2.0-darwin-arm64.tar.gz`
   - `acc-0.2.0-windows-amd64.zip`
   - `checksums.txt`
4. Verify release notes are populated from CHANGELOG.md
5. Download and test a binary:

```bash
# Download Linux binary
wget https://github.com/cloudcwfranck/acc/releases/download/v0.2.0/acc-0.2.0-linux-amd64.tar.gz

# Verify checksum
wget https://github.com/cloudcwfranck/acc/releases/download/v0.2.0/checksums.txt
sha256sum -c checksums.txt --ignore-missing

# Extract and test
tar -xzf acc-0.2.0-linux-amd64.tar.gz
./acc-linux-amd64 version
```

---

## Release Artifacts

Each release produces the following artifacts:

### Binaries

| Platform | Architecture | File |
|----------|-------------|------|
| Linux | AMD64 | `acc-VERSION-linux-amd64.tar.gz` |
| Linux | ARM64 | `acc-VERSION-linux-arm64.tar.gz` |
| macOS | AMD64 (Intel) | `acc-VERSION-darwin-amd64.tar.gz` |
| macOS | ARM64 (Apple Silicon) | `acc-VERSION-darwin-arm64.tar.gz` |
| Windows | AMD64 | `acc-VERSION-windows-amd64.zip` |

### Checksums

`checksums.txt` contains SHA256 checksums for all binaries:

```
a1b2c3... acc-0.1.0-linux-amd64.tar.gz
d4e5f6... acc-0.1.0-linux-arm64.tar.gz
...
```

### Version Information

Each binary is compiled with build-time metadata:

```bash
$ acc version
acc version v0.2.0
commit: 3ffc4fa1234567890abcdef
built: 2025-01-20T10:30:00Z
```

This information is injected via `-ldflags` during the build.

---

## Post-Release Tasks

After creating a release:

1. **Announce the release** (if applicable):
   - Update documentation site
   - Post to community channels
   - Update README with latest version

2. **Monitor for issues**:
   - Watch for bug reports related to the release
   - Check download stats to ensure artifacts are accessible

3. **Plan next release**:
   - Review roadmap
   - Triage issues for next milestone

---

## Starting v1.0

When acc is ready for v1.0:

### 1. API Stability Review

Before releasing v1.0, ensure:
- Command interface is stable
- JSON schemas are finalized
- File formats are stable
- Documentation is complete
- Breaking changes are resolved

### 2. Update Versioning Policy

In `CHANGELOG.md`, add a section explaining v1.0 stability guarantees:

```markdown
## [1.0.0] - 2025-XX-XX

**This is the first stable release of acc.**

Starting with v1.0:
- API is stable - no breaking changes without major version bump
- Semantic versioning strictly followed
- PATCH: bug fixes only
- MINOR: new features, backward compatible
- MAJOR: breaking changes

### Added
- ...
```

### 3. Create the v1.0.0 Release

Follow the normal release process:

```bash
# Update CHANGELOG.md
vim CHANGELOG.md

# Commit
git add CHANGELOG.md
git commit -m "Release v1.0.0 - First stable release"
git push origin main

# Tag
git tag -a v1.0.0 -m "Release v1.0.0 - First stable release"
git push origin v1.0.0
```

### 4. Post-v1.0 Development

After v1.0.0:
- Breaking changes require v2.0.0
- Deprecate features for at least one minor version before removal
- Maintain backward compatibility for JSON schemas (use `schemaVersion`)
- Consider long-term support (LTS) releases if needed

---

## Troubleshooting

### Release workflow failed

1. Check GitHub Actions logs
2. Common issues:
   - Tests failing (fix and re-run)
   - Changelog parsing error (check CHANGELOG.md format)
   - Permission issues (check GITHUB_TOKEN)

### Need to fix a release

If a release has critical bugs:

1. **Patch release:**
   ```bash
   # Fix the bug
   # Update CHANGELOG.md with [0.2.1] section
   git tag -a v0.2.1 -m "Release v0.2.1"
   git push origin v0.2.1
   ```

2. **Delete a bad release:**
   ```bash
   # Delete tag locally
   git tag -d v0.2.0

   # Delete tag remotely
   git push origin :refs/tags/v0.2.0

   # Delete GitHub release (via web UI or gh CLI)
   gh release delete v0.2.0
   ```

### Changelog enforcement blocking PR

If CI blocks your PR for missing changelog entry:

1. Add entry to `CHANGELOG.md` under `[Unreleased]`
2. Commit and push
3. CI will re-run and pass

If your PR genuinely doesn't need a changelog entry (docs-only changes, typo fixes):
- Still add a minimal entry or
- Add `[skip changelog]` in PR description (manual review needed)

---

## Questions?

For release-related questions:
- Open an issue: https://github.com/cloudcwfranck/acc/issues
- Include `[release]` in the title
