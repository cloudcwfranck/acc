/**
 * GitHub API helpers for fetching acc releases
 * Uses unauthenticated requests to avoid token management
 * Rate limit: 60 requests/hour per IP (unauthenticated)
 */

const GITHUB_API_BASE = 'https://api.github.com';
const REPO_OWNER = 'cloudcwfranck';
const REPO_NAME = 'acc';

// ISR revalidation interval (seconds)
export const REVALIDATE_INTERVAL = 300; // 5 minutes

export interface GitHubRelease {
  id: number;
  tag_name: string;
  name: string;
  body: string;
  published_at: string;
  html_url: string;
  prerelease: boolean;
  draft: boolean;
  assets: GitHubAsset[];
}

export interface GitHubAsset {
  name: string;
  browser_download_url: string;
  size: number;
  download_count: number;
}

/**
 * Fetch the latest stable release (excludes prereleases and drafts)
 */
export async function getLatestStableRelease(): Promise<GitHubRelease | null> {
  try {
    const releases = await getReleases(20); // Fetch more to find stable
    const stable = releases.find(r => !r.prerelease && !r.draft);
    return stable || null;
  } catch (error) {
    console.error('Error fetching latest stable release:', error);
    return null;
  }
}

/**
 * Fetch the latest prerelease (if any)
 */
export async function getLatestPrerelease(): Promise<GitHubRelease | null> {
  try {
    const releases = await getReleases(20);
    const prerelease = releases.find(r => r.prerelease && !r.draft);
    return prerelease || null;
  } catch (error) {
    console.error('Error fetching latest prerelease:', error);
    return null;
  }
}

/**
 * Determine if a prerelease is newer than stable
 */
export function isPrereleaseNewer(prerelease: GitHubRelease, stable: GitHubRelease): boolean {
  const prereleaseDate = new Date(prerelease.published_at);
  const stableDate = new Date(stable.published_at);
  return prereleaseDate > stableDate;
}

/**
 * Fetch the latest release (could be stable or prerelease)
 * This is the GitHub API /releases/latest endpoint behavior
 */
export async function getLatestRelease(): Promise<GitHubRelease | null> {
  try {
    const response = await fetch(
      `${GITHUB_API_BASE}/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest`,
      {
        next: { revalidate: REVALIDATE_INTERVAL },
        headers: {
          'Accept': 'application/vnd.github.v3+json',
        },
      }
    );

    if (!response.ok) {
      console.error('Failed to fetch latest release:', response.status);
      return null;
    }

    return response.json();
  } catch (error) {
    console.error('Error fetching latest release:', error);
    return null;
  }
}

/**
 * Fetch all releases (limit to last N)
 */
export async function getReleases(limit: number = 10): Promise<GitHubRelease[]> {
  try {
    const response = await fetch(
      `${GITHUB_API_BASE}/repos/${REPO_OWNER}/${REPO_NAME}/releases?per_page=${limit}`,
      {
        next: { revalidate: REVALIDATE_INTERVAL },
        headers: {
          'Accept': 'application/vnd.github.v3+json',
        },
      }
    );

    if (!response.ok) {
      console.error('Failed to fetch releases:', response.status);
      return [];
    }

    return response.json();
  } catch (error) {
    console.error('Error fetching releases:', error);
    return [];
  }
}

/**
 * Fetch checksums.txt content from a release
 */
export async function getChecksums(release: GitHubRelease): Promise<string | null> {
  const checksumsAsset = release.assets.find(asset => asset.name === 'checksums.txt');

  if (!checksumsAsset) {
    return null;
  }

  try {
    const response = await fetch(checksumsAsset.browser_download_url, {
      next: { revalidate: REVALIDATE_INTERVAL },
    });

    if (!response.ok) {
      return null;
    }

    return response.text();
  } catch (error) {
    console.error('Error fetching checksums:', error);
    return null;
  }
}

/**
 * Parse platform/arch from asset name
 * Examples: acc_0.2.5_linux_amd64.tar.gz, acc_0.2.5_darwin_arm64.tar.gz
 */
export function parseAssetInfo(assetName: string): {
  os: string;
  arch: string;
  format: string;
} | null {
  const match = assetName.match(/acc_[\d.]+_(\w+)_(\w+)\.(tar\.gz|zip)/);

  if (!match) {
    return null;
  }

  return {
    os: match[1],
    arch: match[2],
    format: match[3],
  };
}

/**
 * Get display name for OS
 */
export function getOSDisplayName(os: string): string {
  const names: Record<string, string> = {
    'linux': 'Linux',
    'darwin': 'macOS',
    'windows': 'Windows',
  };
  return names[os] || os;
}

/**
 * Get display name for architecture
 */
export function getArchDisplayName(arch: string): string {
  const names: Record<string, string> = {
    'amd64': 'x64',
    'arm64': 'ARM64',
  };
  return names[arch] || arch;
}
