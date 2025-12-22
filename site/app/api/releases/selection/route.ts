/**
 * Release selection API - Single source of truth
 * Returns computed release selection state including semver-sorted releases,
 * stable/prerelease selection, and checksum detection
 */

import { NextRequest, NextResponse } from 'next/server';
import { getReleases } from '@/lib/github';
import { computeReleaseSelection, ReleaseSelectionState } from '@/lib/releases';

export const runtime = 'nodejs';

export async function GET(request: NextRequest) {
  try {
    // Get query params
    const searchParams = request.nextUrl.searchParams;
    const includePrereleases = searchParams.get('includePrereleases') === 'true';

    // Fetch raw releases from GitHub
    const rawReleases = await getReleases(30);

    // Compute selection state (single source of truth)
    const state: ReleaseSelectionState = computeReleaseSelection(rawReleases, includePrereleases);

    // Return state
    return NextResponse.json(state, {
      headers: {
        'Cache-Control': 's-maxage=60, stale-while-revalidate=30',
      },
    });
  } catch (error) {
    console.error('Error in release selection API:', error);
    return NextResponse.json(
      { error: 'Failed to compute release selection' },
      { status: 500 }
    );
  }
}
