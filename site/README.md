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

The website implements stable-by-default download behavior:

**Stable Release (Default)**:
- Downloads page shows the latest stable release (where `prerelease: false` and `draft: false`)
- Primary CTA buttons link to stable release assets
- Recommended for production use

**Pre-Release Toggle**:
- Users can enable "Include pre-releases" checkbox on `/download` page
- When enabled, shows the latest pre-release if it's newer than stable
- Pre-releases clearly labeled with warning: "Not recommended for production use"
- Preference persisted in `localStorage` and via URL parameter `?prerelease=1`

**Implementation**:
- `lib/github.ts` → `getLatestStableRelease()` and `getLatestPrerelease()`
- Download page uses client-side toggle with server-side data fetching
- Pre-release banner shows site-wide if pre-release is newer than stable

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

## Contributing

The website is isolated under `site/` and does not affect the CLI tool.

Changes to the website:
- Edit files under `site/`
- Test locally with `npm run dev`
- Commit and push (Vercel auto-deploys on push to main)

Do not modify:
- CLI tool code (`internal/`, `cmd/`, etc.)
- Existing CI workflows (except to add site-specific checks in a separate job)
