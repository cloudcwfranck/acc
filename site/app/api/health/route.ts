import { NextResponse } from 'next/server';

const GITHUB_API_BASE = 'https://api.github.com';
const REPO_OWNER = 'cloudcwfranck';
const REPO_NAME = 'acc';

// Cache health check results for 60 seconds
let healthCache: {
  data: any;
  timestamp: number;
} | null = null;

const CACHE_TTL = 60 * 1000; // 60 seconds

export const dynamic = 'force-dynamic';

export async function GET() {
  // Return cached result if fresh
  if (healthCache && Date.now() - healthCache.timestamp < CACHE_TTL) {
    return NextResponse.json(healthCache.data);
  }

  const timestamp = new Date().toISOString();
  const headers: Record<string, string> = {
    'Accept': 'application/vnd.github.v3+json',
  };

  // Use GITHUB_TOKEN if available (server-side only)
  if (process.env.GITHUB_TOKEN) {
    headers['Authorization'] = `Bearer ${process.env.GITHUB_TOKEN}`;
  }

  try {
    // Fetch all releases to check stable vs prerelease
    const releasesResponse = await fetch(
      `${GITHUB_API_BASE}/repos/${REPO_OWNER}/${REPO_NAME}/releases?per_page=10`,
      {
        headers,
        next: { revalidate: 0 }, // Don't cache the fetch itself
      }
    );

    if (!releasesResponse.ok) {
      const result = {
        status: 'down',
        timestamp,
        github: {
          reachable: false,
          error: `HTTP ${releasesResponse.status}`,
          rateLimitRemaining: null,
          latestStableTag: null,
          latestPrereleaseTag: null,
          assetsOk: false,
          checksumsPresent: false,
        },
      };
      healthCache = { data: result, timestamp: Date.now() };
      return NextResponse.json(result);
    }

    const releases = await releasesResponse.json();
    const rateLimitRemaining = releasesResponse.headers.get('x-ratelimit-remaining');

    // Find latest stable and latest prerelease
    const stableReleases = releases.filter((r: any) => !r.prerelease && !r.draft);
    const prereleaseReleases = releases.filter((r: any) => r.prerelease && !r.draft);

    const latestStable = stableReleases[0];
    const latestPrerelease = prereleaseReleases[0];

    // Check assets and checksums for latest stable
    let assetsOk = false;
    let checksumsPresent = false;

    if (latestStable) {
      const assets = latestStable.assets || [];
      // Expect at least one binary asset
      const hasBinaries = assets.some((a: any) =>
        a.name.match(/acc_.*\.(tar\.gz|zip)/)
      );
      const hasChecksums = assets.some((a: any) => a.name === 'checksums.txt');

      assetsOk = hasBinaries;
      checksumsPresent = hasChecksums;
    }

    // Determine overall status
    let status = 'ok';
    if (!latestStable) {
      status = 'degraded'; // No stable releases
    } else if (!assetsOk) {
      status = 'degraded'; // Missing expected assets
    } else if (!checksumsPresent) {
      status = 'degraded'; // Missing checksums
    }

    const result = {
      status,
      timestamp,
      github: {
        reachable: true,
        rateLimitRemaining: rateLimitRemaining ? parseInt(rateLimitRemaining) : null,
        latestStableTag: latestStable?.tag_name || null,
        latestPrereleaseTag: latestPrerelease?.tag_name || null,
        assetsOk,
        checksumsPresent,
      },
    };

    healthCache = { data: result, timestamp: Date.now() };
    return NextResponse.json(result);
  } catch (error) {
    const result = {
      status: 'down',
      timestamp,
      github: {
        reachable: false,
        error: error instanceof Error ? error.message : 'Unknown error',
        rateLimitRemaining: null,
        latestStableTag: null,
        latestPrereleaseTag: null,
        assetsOk: false,
        checksumsPresent: false,
      },
    };
    healthCache = { data: result, timestamp: Date.now() };
    return NextResponse.json(result);
  }
}
