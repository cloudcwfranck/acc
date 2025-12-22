# acc Website

This directory contains the Next.js website for acc, deployed on Vercel.

## Architecture

- **Frontend**: Next.js 14 (App Router, TypeScript, React Server Components)
- **Backend**: GitHub Releases API + repository content (unauthenticated by default)
- **Hosting**: Vercel (edge deployment)
- **Auto-updates**: Dual mechanism - ISR (60s revalidation) + Deploy Hooks (immediate)
- **Operational Monitoring**: `/api/health` endpoint + `/status` dashboard
- **Release Selection**: Stable-by-default with optional pre-release toggle

## Local Development

### Prerequisites

- Node.js 18+ and npm

### Setup

```bash
cd site

# Install dependencies
npm install

# Run development server
npm run dev

# Open http://localhost:3000
```

### Build

```bash
# Production build
npm run build

# Test production build locally
npm run start
```

### Testing

```bash
# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Run tests with coverage
npm test -- --coverage
```

**Test Coverage:**
- ✅ Semver sorting (v0.2.10 > v0.2.9)
- ✅ Stable vs prerelease selection
- ✅ Checksum detection (multiple formats: checksums.txt, SHA256SUMS, .sha256 files)
- ✅ Single source of truth logic
- ✅ Draft filtering

## Deployment

### Initial Vercel Setup

1. **Import to Vercel**:
   - Go to [vercel.com](https://vercel.com)
   - Import the `cloudcwfranck/acc` repository
   - Set the **Root Directory** to `site/`
   - Framework Preset: Next.js
   - Deploy

2. **Configure Deploy Hook** (for auto-updates on release):
   - In Vercel project settings → Git
   - Create a Deploy Hook (name: "Release Published")
   - Copy the webhook URL
   - Add to GitHub repo secrets as `VERCEL_DEPLOY_HOOK_URL`:
     - Go to GitHub repo → Settings → Secrets and variables → Actions
     - New repository secret: `VERCEL_DEPLOY_HOOK_URL`
     - Paste the Vercel deploy hook URL

### How Auto-Updates Work

1. **ISR (Incremental Static Regeneration)**:
   - Release data is fetched from GitHub API with `revalidate: 300` (5 minutes)
   - Vercel automatically regenerates pages every 5 minutes if data changes
   - Configure interval in `site/lib/github.ts` → `REVALIDATE_INTERVAL`

2. **Deploy Hook on Release**:
   - When a new release is published on GitHub
   - GitHub Actions workflow `.github/workflows/site-deploy.yml` runs
   - Triggers Vercel deploy hook
   - Vercel rebuilds the entire site immediately
   - New release appears on the website

3. **Fallback**:
   - If deploy hook fails or is not configured, ISR still works
   - Site will update within 5 minutes of a new release

## Environment Variables

No environment variables required for basic functionality (uses unauthenticated GitHub API).

**Optional** (if GitHub API rate limiting becomes an issue):
- `GITHUB_TOKEN`: Personal access token for authenticated API requests (increases rate limit to 5000/hour)

To add in Vercel:
- Project Settings → Environment Variables
- Add `GITHUB_TOKEN` with your token

## Rate Limiting

- **Unauthenticated**: 60 requests/hour per IP
- **Authenticated** (with `GITHUB_TOKEN`): 5000 requests/hour

The site uses server-side fetching with ISR caching, so rate limits should not be an issue for normal traffic. If needed, add authentication via environment variable.

## Pages

- `/` - Homepage (hero, features, how it works)
- `/download` - Download latest release with OS/arch buttons
- `/docs` - Quick start guide + links to full docs
- `/releases` - List of recent releases with changelogs

## Security Headers

Configured in `next.config.js`:
- Content Security Policy (CSP)
- X-Frame-Options: DENY
- X-Content-Type-Options: nosniff
- Referrer-Policy: strict-origin-when-cross-origin

## Assets

### Demo Video/GIF

Place your terminal demo at:
- `site/public/demo/demo.gif` (preferred), or
- `site/public/demo/demo.svg`

Homepage will automatically display it. Currently shows a placeholder.

## Troubleshooting

### Deploy Hook Not Triggering

1. Check GitHub Actions logs for the `site-deploy.yml` workflow
2. Verify `VERCEL_DEPLOY_HOOK_URL` secret is set correctly
3. Check Vercel deployment logs

### Release Data Not Updating

1. Check GitHub API rate limits (should be visible in Next.js build logs)
2. Verify ISR is working: trigger a manual deploy in Vercel
3. Check browser console for fetch errors

### Build Failures

1. Ensure all dependencies are in `package.json`
2. Run `npm run build` locally to reproduce
3. Check Vercel build logs for specific errors

## Enterprise Features

### Stable vs Pre-Release Selection

The website implements stable-by-default download behavior with **enterprise-grade semver sorting** and a **single source of truth** for all release logic.

**Stable Release (Default)**:
- Downloads page shows the latest stable release (where `prerelease: false` and `draft: false`)
- Primary CTA buttons link to stable release assets
- Recommended for production use
- **Guaranteed correct even with v0.2.10 > v0.2.9 scenarios**

**Pre-Release Toggle**:
- Users can enable "Include pre-releases" checkbox on `/download` page
- When enabled, shows the **highest semver overall** (stable or prerelease)
- Pre-releases clearly labeled with warning: "Not recommended for production use"
- Preference persisted in `localStorage` and via URL parameter `?prerelease=1`

**Semver Sorting**:
- All releases sorted by semantic version (not GitHub API order)
- Correctly handles: v0.2.10 > v0.2.9 > v0.2.5
- Stable versions ranked higher than prereleases of same version (v1.0.0 > v1.0.0-alpha)
- Draft releases completely filtered out

**Checksum Detection**:
- Detects multiple checksum formats:
  - `checksums.txt` (preferred)
  - `SHA256SUMS`
  - `checksums.sha256`
  - `sha256sums.txt`
  - Per-asset `.sha256` files (fallback)
- **Always checks against selected release** (no stable/prerelease mismatch)
- Download page shows warning if checksums missing

**Single Source of Truth**:
- All logic in `lib/releases.ts` → `computeReleaseSelection()`
- Returns deterministic state:
  - `latestStable` (highest stable version)
  - `latestPrerelease` (highest prerelease version)
  - `selectedRelease` (based on toggle)
  - `hasChecksums` (checksum availability for selected release)
- Used by:
  - `/api/releases/selection` (API route)
  - `/api/health` (health endpoint)
  - Download page UI
  - Status page

**Implementation**:
- `lib/releases.ts` → Core semver logic and release selection
- `lib/github.ts` → GitHub API fetching (with optional server-side GITHUB_TOKEN)
- `app/api/releases/selection/route.ts` → Single source of truth API
- Download page fetches from API (no client-side release logic)
- **Comprehensive test coverage**: 40+ tests in `__tests__/releases.test.ts`

### Operational Health Monitoring

The website includes enterprise-grade health monitoring:

**Health Check Endpoint**: `GET /api/health`

Returns JSON with operational status:
```json
{
  "status": "ok|degraded|down",
  "timestamp": "2025-12-22T...",
  "github": {
    "reachable": true,
    "rateLimitRemaining": 58,
    "latestStableTag": "v0.2.5",
    "latestPrereleaseTag": "v0.2.6",
    "assetsOk": true,
    "checksumsPresent": true
  }
}
```

**Status Meanings**:
- `ok`: GitHub API reachable, stable release exists with valid assets
- `degraded`: Reachable but issues (missing checksums, no stable release, etc.)
- `down`: GitHub API unreachable or returning errors

**Status Dashboard**: `/status`
- Real-time health metrics
- Rate limit information
- Asset validation status
- Troubleshooting guidance
- Auto-refreshes every 60 seconds

**Caching**:
- Health checks cached for 60 seconds server-side
- Prevents hammering GitHub API
- Configurable cache TTL in `app/api/health/route.ts`

**Status Indicator**:
- Footer shows "● Status" link with pulsing green dot
- Links to `/status` page for detailed metrics

### Auto-Update Strategy

**Dual Mechanism (Belt + Suspenders)**:

1. **ISR (Baseline)**:
   - Server-side data fetching with `revalidate: 60` (60 seconds)
   - Automatic background revalidation
   - No deploy hook required
   - Guarantees updates within 60 seconds

2. **Deploy Hook (Immediate)**:
   - GitHub Actions workflow triggers on `release: published`
   - POST request to Vercel deploy hook
   - Full site rebuild in 30-60 seconds
   - Requires `VERCEL_DEPLOY_HOOK_URL` secret

**Why Both?**:
- ISR provides reliable fallback if deploy hook fails
- Deploy hook provides near-instant updates
- Redundancy ensures site stays current

### Pre-Release Banner

Global banner displayed when:
- A pre-release exists
- Pre-release is newer than stable release

**Features**:
- Clear warning: "Pre-release available: vX.Y.Z (not recommended for production)"
- CTA to view pre-release on download page
- Dismissible (persisted per version in `localStorage`)
- Responsive design
- Dark/light mode compatible

### Testing

Run tests locally:
```bash
npm test              # Run all tests
npm run test:watch   # Watch mode
```

**Test Coverage**:
- Release parsing logic (stable vs pre-release)
- Asset information parsing (OS/arch detection)
- Display name mapping
- Pre-release selection logic

**Test Files**:
- `__tests__/github.test.ts` - Core GitHub API helper tests

## Operations

### Monitoring Production Health

1. **Check status page**: https://your-site.vercel.app/status
2. **API health check**: `curl https://your-site.vercel.app/api/health`
3. **Vercel deployment logs**: Check for errors
4. **GitHub Actions**: Verify deploy hook workflow runs

### Common Operational Tasks

**Verify Pre-Release Toggle**:
1. Publish a pre-release on GitHub (check "This is a pre-release")
2. Wait 60 seconds (ISR) or trigger deploy hook
3. Visit `/download` - should show stable by default
4. Enable "Include pre-releases" - should switch to pre-release
5. Verify warning displays: "Not recommended for production use"

**Test Health Check**:
```bash
# Check health status
curl https://your-site.vercel.app/api/health | jq

# Should return "ok" if GitHub is reachable and releases exist
# Returns "degraded" if issues detected
# Returns "down" if GitHub API unreachable
```

**Simulate GitHub Outage**:
- Health endpoint will return `"status": "down"`
- Status page shows troubleshooting guidance
- Download page gracefully degrades (shows error message)

### Environment Variables

| Variable | Required | Purpose | Default |
|----------|----------|---------|---------|
| `GITHUB_TOKEN` | No | Increase rate limit to 5000/hour | Unauthenticated (60/hour) |
| `VERCEL_DEPLOY_HOOK_URL` | No | Enable immediate deploy on release | ISR fallback only |

**Setting in Vercel**:
- Project Settings → Environment Variables
- Add variables for Production, Preview, Development as needed
- Redeploy after adding variables

## Release Pipeline & Validation

### Release Artifact Requirements

Every acc release MUST include:
- ✅ Cross-platform binaries (Linux, macOS, Windows × AMD64, ARM64)
- ✅ **checksums.txt** with SHA256 for all archives
- ✅ Release notes from CHANGELOG.md
- ✅ Proper prerelease marking (v0.x or versions with `-` suffix)

### Automated Validation

The release workflow includes automated validation to ensure release quality:

**Build Job** (`.github/workflows/release.yml`):
1. Builds cross-platform binaries
2. Packages as `.tar.gz` (Linux/macOS) and `.zip` (Windows)
3. Generates `checksums.txt` with SHA256 for all archives
4. Uploads artifacts to GitHub Release

**Validation Job** (runs after build):
1. ✅ Verifies `checksums.txt` exists
2. ✅ Ensures all 5 platform archives have checksums
3. ✅ Validates checksums match actual files (`sha256sum -c`)
4. ✅ Confirms minimum 5 archives present
5. ❌ Fails release if any check fails

**What Happens on Failure:**
- Release job fails with clear error message
- GitHub Release is not published
- Maintainer receives actionable guidance on what to fix

### Website Integration with Releases

**Automatic Updates:**
1. **On Release Published**: GitHub Actions triggers Vercel deploy hook
2. **ISR Fallback**: 60-second revalidation ensures updates even if hook fails
3. **Download Page**: Automatically shows new release within 1 minute

**Download Verification:**
- Download page includes checksum verification in install snippet
- If checksums missing: Shows warning "⚠️ Checksums not available for this release"
- If checksums present: Displays full SHA256 checksums with verification commands

**Health Monitoring:**
- `/api/health` endpoint validates latest release has required assets
- Status degrades to "degraded" if checksums missing
- `/status` page shows asset validation status

### Verifying a Release Manually

**Check Release Completeness:**
```bash
# List assets for a release
gh release view v0.2.6 --json assets --jq '.assets[].name'

# Expected output:
# acc_0.2.6_darwin_amd64.tar.gz
# acc_0.2.6_darwin_arm64.tar.gz
# acc_0.2.6_linux_amd64.tar.gz
# acc_0.2.6_linux_arm64.tar.gz
# acc_0.2.6_windows_amd64.zip
# checksums.txt
```

**Verify Checksums:**
```bash
# Download and verify
curl -LO https://github.com/cloudcwfranck/acc/releases/download/v0.2.6/checksums.txt
curl -LO https://github.com/cloudcwfranck/acc/releases/download/v0.2.6/acc_0.2.6_linux_amd64.tar.gz

# Verify checksum matches
sha256sum -c checksums.txt --ignore-missing
# Expected: acc_0.2.6_linux_amd64.tar.gz: OK
```

**Check Website Auto-Update:**
```bash
# 1. Note current version shown on /download
# 2. Publish new release on GitHub
# 3. Wait 60 seconds (ISR interval)
# 4. Reload /download - should show new version
# 5. Check /status - should show new stable tag
```

### CI/CD Workflows

**Site CI** (`.github/workflows/site-ci.yml`):
- Runs on PRs that modify `site/**`
- Linting, type checking, build validation
- **Unit tests**: Validates stable/prerelease selection logic (27 tests)
- **Smoke tests**: Health endpoint, download page, status page

**Site Deploy** (`.github/workflows/site-deploy.yml`):
- Triggers on `release:published`
- POSTs to Vercel deploy hook (if configured)
- Gracefully skips if `VERCEL_DEPLOY_HOOK_URL` not set
- ISR provides fallback auto-update mechanism

**Release** (`.github/workflows/release.yml`):
- Triggers on `v*` tags
- Builds cross-platform binaries
- Generates and validates checksums
- Publishes GitHub Release with artifacts
- **Critical**: Blocks release if validation fails

## Contributing

The website is isolated under `site/` and does not affect the CLI tool.

Changes to the website:
- Edit files under `site/`
- Test locally with `npm run dev`
- Commit and push (Vercel auto-deploys on push to main)

Do not modify:
- CLI tool code (`internal/`, `cmd/`, etc.)
- Existing CI workflows (except to add site-specific checks in a separate job)
