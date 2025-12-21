# acc Website Architecture

This document describes the architecture, deployment, and maintenance of the official acc website.

## Overview

The acc website is a Next.js 14 application deployed on Vercel that provides:
- Auto-updating download links from GitHub Releases
- Release history with changelogs
- Quick start documentation
- Product information and features

**URL**: https://acc.vercel.app *(after v0.2.6 deployment)*

## Architecture

### Technology Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **Frontend Framework** | Next.js 14 (App Router) | React-based server-side rendering |
| **Language** | TypeScript | Type-safe development |
| **Styling** | CSS Modules | Component-scoped styles |
| **Data Source** | GitHub Releases API | Release metadata and assets |
| **Hosting** | Vercel | Edge deployment and CDN |
| **Auto-updates** | ISR + Deploy Hooks | Real-time release synchronization |

### Directory Structure

```
site/
├── app/                    # Next.js App Router pages
│   ├── layout.tsx          # Root layout with metadata
│   ├── page.tsx            # Homepage
│   ├── download/page.tsx   # Download page
│   ├── docs/page.tsx       # Quick start docs
│   ├── releases/page.tsx   # Release history
│   └── globals.css         # Global styles
├── components/             # Reusable components
│   ├── Navigation.tsx      # Sticky header
│   ├── Footer.tsx          # Footer with links
│   └── *.module.css        # Component styles
├── lib/                    # Core utilities
│   └── github.ts           # GitHub API helpers
├── public/                 # Static assets
│   └── demo/               # Demo assets
├── package.json            # Dependencies and scripts
├── tsconfig.json           # TypeScript config
├── next.config.js          # Next.js config + security headers
└── README.md               # Setup and deployment guide
```

## Data Flow

### Release Information Flow

```
GitHub Release Published
        ↓
.github/workflows/site-deploy.yml triggers
        ↓
POST request to Vercel Deploy Hook
        ↓
Vercel rebuilds site
        ↓
New release appears on website (immediate)
```

**Fallback (if deploy hook not configured):**

```
GitHub Release Published
        ↓
(5 minutes pass)
        ↓
ISR revalidation triggers on next request
        ↓
Vercel fetches fresh data from GitHub API
        ↓
New release appears on website
```

### GitHub API Integration

**API Endpoints Used:**
- `GET /repos/cloudcwfranck/acc/releases/latest` - Latest release
- `GET /repos/cloudcwfranck/acc/releases` - Recent releases
- `GET /repos/cloudcwfranck/acc/releases/assets/{id}` - Release assets

**Authentication:**
- Unauthenticated by default (60 requests/hour per IP)
- Optional `GITHUB_TOKEN` env var for authenticated requests (5000 requests/hour)

**Caching Strategy:**
- ISR with `revalidate: 300` (5 minutes)
- Server-side fetching minimizes rate limit impact
- Stale-while-revalidate ensures fast responses

## Pages

### Homepage (`/`)

**Sections:**
1. **Hero** - Product tagline and CTA
2. **Features** - Key capabilities (trust, CI-ready, explainable)
3. **How It Works** - 3-step workflow
4. **Demo** - Placeholder for terminal recording

**Data:** Static content (no API calls)

### Download Page (`/download`)

**Features:**
- Latest release version and date
- Platform-grouped downloads (Linux, macOS, Windows)
- OS/arch detection with highlighted recommendations
- SHA256 checksums for verification
- Installation instructions

**Data Source:** GitHub Releases API
**Cache:** ISR with 5-minute revalidation

### Docs Page (`/docs`)

**Content:**
- 4-step quick start guide
- Core concepts explanation
- Links to full documentation (README, testing-contract)

**Data:** Static content

### Releases Page (`/releases`)

**Features:**
- Last 10 releases
- Release tag, date, and changelog preview
- Links to full release notes on GitHub

**Data Source:** GitHub Releases API
**Cache:** ISR with 5-minute revalidation

## Auto-Update Mechanism

### 1. ISR (Incremental Static Regeneration)

**How it works:**
```typescript
// site/lib/github.ts
export const REVALIDATE_INTERVAL = 300; // 5 minutes

export async function getLatestRelease() {
  const response = await fetch(
    `${GITHUB_API_BASE}/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest`,
    {
      next: { revalidate: REVALIDATE_INTERVAL },
      headers: { 'Accept': 'application/vnd.github.v3+json' }
    }
  );
  return response.ok ? response.json() : null;
}
```

**Behavior:**
- First request: Fetches fresh data from GitHub API
- Subsequent requests (within 5 minutes): Serves cached data
- After 5 minutes: Next request triggers background revalidation
- User always gets fast response (cached or revalidated)

**Trade-offs:**
- ✅ Low rate limit usage (max ~12 requests/hour)
- ✅ Fast response times (cached)
- ⚠️ Up to 5-minute delay for new releases

### 2. Deploy Hooks (Immediate Updates)

**How it works:**
1. GitHub Actions workflow `.github/workflows/site-deploy.yml` listens for `release.published` event
2. Workflow checks if `VERCEL_DEPLOY_HOOK_URL` secret exists
3. If exists, sends POST request to Vercel deploy hook URL
4. Vercel triggers full rebuild
5. New release appears immediately (no 5-minute delay)

**Setup:**
1. Create deploy hook in Vercel project settings
2. Add webhook URL to GitHub repo secrets as `VERCEL_DEPLOY_HOOK_URL`
3. Workflow auto-runs on future releases

**Behavior if not configured:**
- Workflow skips with exit 0 (not an error)
- ISR still works as fallback
- Users see new releases within 5 minutes

## Security

### Headers (configured in `next.config.js`)

```javascript
{
  'X-Frame-Options': 'DENY',
  'X-Content-Type-Options': 'nosniff',
  'Referrer-Policy': 'strict-origin-when-cross-origin',
  'Content-Security-Policy':
    "default-src 'self'; " +
    "script-src 'self' 'unsafe-eval' 'unsafe-inline'; " +
    "connect-src 'self' https://api.github.com;"
}
```

**Policy:**
- No inline scripts (except Next.js necessities)
- No third-party scripts
- Only GitHub API allowed for fetch
- No iframes allowed
- Strict referrer policy

### Rate Limiting

**Unauthenticated (default):**
- 60 requests/hour per IP
- Sufficient for ISR with 5-minute revalidation
- Server-side fetching means single IP

**Authenticated (optional):**
- Set `GITHUB_TOKEN` environment variable in Vercel
- 5000 requests/hour
- Use if traffic increases or rate limits become an issue

### No Secrets in Client Code

- All API requests happen server-side (React Server Components)
- No tokens, keys, or secrets sent to client
- GitHub API accessed from Vercel edge functions only

## Deployment

### Initial Setup

1. **Import to Vercel:**
   - Go to [vercel.com](https://vercel.com)
   - Import `cloudcwfranck/acc` repository
   - Set **Root Directory** to `site/`
   - Framework Preset: Next.js
   - Deploy

2. **Configure Deploy Hook (optional but recommended):**
   - Vercel project settings → Git
   - Create Deploy Hook (name: "Release Published")
   - Copy webhook URL
   - Add to GitHub repo secrets:
     - Settings → Secrets and variables → Actions
     - New secret: `VERCEL_DEPLOY_HOOK_URL`
     - Paste webhook URL

3. **Set Environment Variables (optional):**
   - Vercel project settings → Environment Variables
   - Add `GITHUB_TOKEN` with personal access token (if rate limiting becomes an issue)

### Deployment Triggers

**Automatic deployments on:**
- Push to `main` branch (Vercel default)
- New release published (via deploy hook)
- Manual deploy via Vercel dashboard

**Branch-based deployments:**
- `main` → Production (acc.vercel.app)
- Other branches → Preview deployments (unique URLs)

## Local Development

### Setup

```bash
cd site

# Install dependencies
npm install

# Run development server
npm run dev

# Open http://localhost:3000
```

### Build and Test Locally

```bash
# Production build
npm run build

# Test production build
npm run start
```

### Environment Variables (optional)

Create `site/.env.local`:

```bash
# Optional: Use authenticated GitHub API
GITHUB_TOKEN=ghp_your_personal_access_token_here
```

**Note:** Unauthenticated API is sufficient for local development and production (with ISR caching).

## Monitoring and Troubleshooting

### Check ISR is Working

1. Visit `/download` page
2. Note the release version
3. Publish a new GitHub release
4. Wait 5 minutes
5. Revisit `/download` page
6. New release should appear

### Check Deploy Hook is Working

1. Publish a new GitHub release
2. Check GitHub Actions workflow run for `site-deploy.yml`
3. Check Vercel deployment logs
4. Visit website immediately (no 5-minute wait)
5. New release should appear

### Common Issues

**Problem:** Release data not updating

**Possible causes:**
1. GitHub API rate limit hit (check response headers)
2. ISR not working (check Next.js build logs)
3. Deploy hook not configured (check GitHub Actions logs)

**Solutions:**
1. Add `GITHUB_TOKEN` environment variable
2. Trigger manual deploy in Vercel
3. Set up deploy hook (see Initial Setup)

---

**Problem:** Build failures

**Possible causes:**
1. Missing dependencies
2. TypeScript errors
3. Invalid GitHub API responses

**Solutions:**
1. Run `npm run build` locally to reproduce
2. Check Vercel build logs for specific errors
3. Verify GitHub API is accessible

---

**Problem:** Deploy hook not triggering

**Possible causes:**
1. `VERCEL_DEPLOY_HOOK_URL` secret not set
2. Secret has wrong value
3. Vercel webhook endpoint changed

**Solutions:**
1. Verify secret exists in GitHub repo settings
2. Regenerate deploy hook in Vercel and update secret
3. Check GitHub Actions logs for HTTP response codes

## Maintenance

### Updating Dependencies

```bash
cd site

# Check for outdated dependencies
npm outdated

# Update to latest compatible versions
npm update

# Update to latest versions (may include breaking changes)
npm install next@latest react@latest react-dom@latest

# Test locally
npm run build
npm run start
```

### Updating ISR Interval

Edit `site/lib/github.ts`:

```typescript
export const REVALIDATE_INTERVAL = 300; // Change to desired seconds
```

Shorter intervals = more fresh data, higher API usage
Longer intervals = less API usage, staler data

**Recommended:** 300 seconds (5 minutes) for good balance

### Adding New Pages

1. Create `site/app/new-page/page.tsx`
2. Add navigation link in `site/components/Navigation.tsx`
3. Add footer link in `site/components/Footer.tsx` (if appropriate)
4. Test locally with `npm run dev`
5. Commit and push (Vercel auto-deploys)

## Future Enhancements

**Potential improvements (not implemented in v0.2.6):**

- **Blog** - Release announcements and tutorials
- **Interactive demo** - Browser-based acc playground
- **Search** - Full-text search across documentation
- **Analytics** - Privacy-respecting usage analytics
- **Dark mode toggle** - User-controlled theme switching (currently auto-detects)
- **Multi-language** - Internationalization support

## References

- [Next.js Documentation](https://nextjs.org/docs)
- [Vercel Deployment Docs](https://vercel.com/docs)
- [GitHub REST API](https://docs.github.com/en/rest)
- [ISR Documentation](https://nextjs.org/docs/app/building-your-application/data-fetching/incremental-static-regeneration)
- [Vercel Deploy Hooks](https://vercel.com/docs/concepts/git/deploy-hooks)
