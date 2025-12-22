/**
 * Release selection logic - Single source of truth
 * Handles semver sorting, stable vs prerelease selection, checksum detection
 */

import { GitHubRelease } from './github';

/**
 * Parse semver from tag (handles v0.2.10, v1.2.3, etc.)
 */
export function parseSemver(tag: string): { major: number; minor: number; patch: number; prerelease: string } | null {
  const match = tag.match(/^v?(\d+)\.(\d+)\.(\d+)(?:-(.+))?$/);
  if (!match) return null;

  return {
    major: parseInt(match[1], 10),
    minor: parseInt(match[2], 10),
    patch: parseInt(match[3], 10),
    prerelease: match[4] || '',
  };
}

/**
 * Compare two semver versions
 * Returns: -1 if a < b, 0 if equal, 1 if a > b
 */
export function compareSemver(a: string, b: string): number {
  const versionA = parseSemver(a);
  const versionB = parseSemver(b);

  if (!versionA || !versionB) {
    // Fallback to string comparison if parsing fails
    return a.localeCompare(b);
  }

  // Compare major.minor.patch
  if (versionA.major !== versionB.major) return versionA.major - versionB.major;
  if (versionA.minor !== versionB.minor) return versionA.minor - versionB.minor;
  if (versionA.patch !== versionB.patch) return versionA.patch - versionB.patch;

  // Handle prerelease: no prerelease > prerelease
  if (!versionA.prerelease && versionB.prerelease) return 1;
  if (versionA.prerelease && !versionB.prerelease) return -1;

  // Both prereleases or both stable
  return versionA.prerelease.localeCompare(versionB.prerelease);
}

/**
 * Sort releases by semver (descending - newest first)
 */
export function sortReleasesBySemver(releases: GitHubRelease[]): GitHubRelease[] {
  return [...releases].sort((a, b) => compareSemver(b.tag_name, a.tag_name));
}

/**
 * Checksum file patterns to detect (in priority order)
 */
const CHECKSUM_PATTERNS = [
  'checksums.txt',
  'checksums.sha256',
  'SHA256SUMS',
  'sha256sums.txt',
  'CHECKSUMS',
];

/**
 * Detect if checksums are available for a release
 * Returns the checksum asset if found, null otherwise
 */
export function detectChecksumAsset(release: GitHubRelease): { name: string; url: string } | null {
  // Check for standard checksum files
  for (const pattern of CHECKSUM_PATTERNS) {
    const asset = release.assets.find(a => a.name === pattern || a.name.toLowerCase() === pattern.toLowerCase());
    if (asset) {
      return { name: asset.name, url: asset.browser_download_url };
    }
  }

  // Check for per-asset .sha256 files (fallback)
  const sha256Assets = release.assets.filter(a => a.name.endsWith('.sha256'));
  if (sha256Assets.length > 0) {
    // If we have per-asset sha256 files, checksums are available
    // Return the first one as a signal
    return { name: sha256Assets[0].name, url: sha256Assets[0].browser_download_url };
  }

  return null;
}

/**
 * Release selection state - Single source of truth
 */
export interface ReleaseSelectionState {
  /** All releases sorted by semver (excluding drafts) */
  releases: GitHubRelease[];
  /** Latest stable release (prerelease=false) */
  latestStable: GitHubRelease | null;
  /** Latest prerelease (prerelease=true) */
  latestPrerelease: GitHubRelease | null;
  /** Currently selected release (based on toggle) */
  selectedRelease: GitHubRelease | null;
  /** Whether to include prereleases */
  includePrereleases: boolean;
  /** Checksum asset for selected release */
  checksumAsset: { name: string; url: string } | null;
  /** Whether checksums are available */
  hasChecksums: boolean;
}

/**
 * Compute release selection state - SINGLE SOURCE OF TRUTH
 * All UI (download page, status page, health endpoint) must use this
 */
export function computeReleaseSelection(
  rawReleases: GitHubRelease[],
  includePrereleases: boolean
): ReleaseSelectionState {
  // Filter out drafts
  const releases = rawReleases.filter(r => !r.draft);

  // Sort by semver (newest first)
  const sortedReleases = sortReleasesBySemver(releases);

  // Find latest stable (prerelease=false)
  const latestStable = sortedReleases.find(r => !r.prerelease) || null;

  // Find latest prerelease (prerelease=true)
  const latestPrerelease = sortedReleases.find(r => r.prerelease) || null;

  // Determine selected release
  let selectedRelease: GitHubRelease | null = latestStable;

  if (includePrereleases) {
    // When prereleases enabled, select the highest semver overall
    // This could be stable or prerelease
    selectedRelease = sortedReleases[0] || null;
  }

  // Detect checksums for selected release
  const checksumAsset = selectedRelease ? detectChecksumAsset(selectedRelease) : null;

  return {
    releases: sortedReleases,
    latestStable,
    latestPrerelease,
    selectedRelease,
    includePrereleases,
    checksumAsset,
    hasChecksums: checksumAsset !== null,
  };
}

/**
 * Check if selected release is a prerelease
 */
export function isSelectedPrerelease(state: ReleaseSelectionState): boolean {
  return state.selectedRelease?.prerelease === true;
}

/**
 * Check if a newer prerelease exists than the stable
 */
export function hasNewerPrerelease(state: ReleaseSelectionState): boolean {
  if (!state.latestStable || !state.latestPrerelease) return false;
  return compareSemver(state.latestPrerelease.tag_name, state.latestStable.tag_name) > 0;
}
