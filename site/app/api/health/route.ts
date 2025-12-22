import { NextResponse } from 'next/server';
import { getReleases } from '@/lib/github';
import { computeReleaseSelection } from '@/lib/releases';

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

  try {
    // Fetch releases (uses server-side GITHUB_TOKEN if available)
    const releases = await getReleases(30);

    if (releases.length === 0) {
      const result = {
        status: 'down',
        timestamp,
        github: {
          reachable: false,
          error: 'No releases found',
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

    // Compute release selection state (single source of truth)
    const state = computeReleaseSelection(releases, false); // Check stable by default

    // Check assets for latest stable
    let assetsOk = false;
    if (state.latestStable) {
      const hasBinaries = state.latestStable.assets.some(a =>
        a.name.match(/acc_.*\.(tar\.gz|zip)/)
      );
      assetsOk = hasBinaries;
    }

    // Determine overall status
    let status: 'ok' | 'degraded' | 'down' = 'ok';
    if (!state.latestStable) {
      status = 'degraded'; // No stable releases
    } else if (!assetsOk) {
      status = 'degraded'; // Missing expected assets
    } else if (!state.hasChecksums) {
      status = 'degraded'; // Missing checksums
    }

    const result = {
      status,
      timestamp,
      github: {
        reachable: true,
        rateLimitRemaining: null, // Can't get from getReleases, but API is working
        latestStableTag: state.latestStable?.tag_name || null,
        latestPrereleaseTag: state.latestPrerelease?.tag_name || null,
        assetsOk,
        checksumsPresent: state.hasChecksums,
        checksumAsset: state.checksumAsset?.name || null,
        checksumSource: state.checksumSource || null,
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
