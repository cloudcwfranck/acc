# acc Website

This directory contains the Next.js website for acc, deployed on Vercel.

## Architecture

- **Frontend**: Next.js 14 (App Router, TypeScript)
- **Backend**: GitHub Releases API + repository content
- **Hosting**: Vercel
- **Auto-updates**: ISR (Incremental Static Regeneration) + Deploy Hooks

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

## Contributing

The website is isolated under `site/` and does not affect the CLI tool.

Changes to the website:
- Edit files under `site/`
- Test locally with `npm run dev`
- Commit and push (Vercel auto-deploys on push to main)

Do not modify:
- CLI tool code (`internal/`, `cmd/`, etc.)
- Existing CI workflows (except to add site-specific checks in a separate job)
