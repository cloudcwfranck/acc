/**
 * Tests for release selection logic - semver sorting, stable vs prerelease, checksum detection
 */

import {
  parseSemver,
  compareSemver,
  sortReleasesBySemver,
  detectChecksumAsset,
  computeReleaseSelection,
  isSelectedPrerelease,
  hasNewerPrerelease,
} from '../lib/releases';
import { GitHubRelease } from '../lib/github';

describe('parseSemver', () => {
  it('parses standard semver with v prefix', () => {
    const result = parseSemver('v1.2.3');
    expect(result).toEqual({ major: 1, minor: 2, patch: 3, prerelease: '' });
  });

  it('parses semver without v prefix', () => {
    const result = parseSemver('0.2.5');
    expect(result).toEqual({ major: 0, minor: 2, patch: 5, prerelease: '' });
  });

  it('parses prerelease versions', () => {
    const result = parseSemver('v1.0.0-alpha.1');
    expect(result).toEqual({ major: 1, minor: 0, patch: 0, prerelease: 'alpha.1' });
  });

  it('returns null for invalid versions', () => {
    expect(parseSemver('invalid')).toBeNull();
    expect(parseSemver('v1.2')).toBeNull();
  });
});

describe('compareSemver', () => {
  it('correctly orders v0.2.10 > v0.2.9', () => {
    expect(compareSemver('v0.2.10', 'v0.2.9')).toBeGreaterThan(0);
    expect(compareSemver('v0.2.9', 'v0.2.10')).toBeLessThan(0);
  });

  it('correctly orders v0.3.0 > v0.2.10', () => {
    expect(compareSemver('v0.3.0', 'v0.2.10')).toBeGreaterThan(0);
  });

  it('correctly orders v1.0.0 > v0.9.9', () => {
    expect(compareSemver('v1.0.0', 'v0.9.9')).toBeGreaterThan(0);
  });

  it('treats stable versions as higher than prereleases', () => {
    expect(compareSemver('v1.0.0', 'v1.0.0-alpha')).toBeGreaterThan(0);
    expect(compareSemver('v1.0.0-alpha', 'v1.0.0')).toBeLessThan(0);
  });

  it('returns 0 for equal versions', () => {
    expect(compareSemver('v1.2.3', 'v1.2.3')).toBe(0);
  });
});

describe('sortReleasesBySemver', () => {
  const createRelease = (tag: string): GitHubRelease => ({
    id: 1,
    tag_name: tag,
    name: tag,
    body: '',
    published_at: '2025-01-01T00:00:00Z',
    html_url: '',
    prerelease: tag.includes('-'),
    draft: false,
    assets: [],
  });

  it('sorts releases by semver descending (newest first)', () => {
    const releases = [
      createRelease('v0.2.5'),
      createRelease('v0.2.10'),
      createRelease('v0.2.9'),
      createRelease('v0.3.0'),
      createRelease('v0.2.8'),
    ];

    const sorted = sortReleasesBySemver(releases);

    expect(sorted.map(r => r.tag_name)).toEqual([
      'v0.3.0',
      'v0.2.10',
      'v0.2.9',
      'v0.2.8',
      'v0.2.5',
    ]);
  });

  it('places stable versions before prereleases of same version', () => {
    const releases = [
      createRelease('v1.0.0-alpha'),
      createRelease('v1.0.0'),
      createRelease('v1.0.0-beta'),
    ];

    const sorted = sortReleasesBySemver(releases);

    expect(sorted[0].tag_name).toBe('v1.0.0');
    expect(sorted[1].tag_name).toBe('v1.0.0-beta');
    expect(sorted[2].tag_name).toBe('v1.0.0-alpha');
  });
});

describe('detectChecksumAsset', () => {
  const createReleaseWithAssets = (assetNames: string[]): GitHubRelease => ({
    id: 1,
    tag_name: 'v1.0.0',
    name: 'v1.0.0',
    body: '',
    published_at: '2025-01-01T00:00:00Z',
    html_url: '',
    prerelease: false,
    draft: false,
    assets: assetNames.map(name => ({
      name,
      browser_download_url: `https://example.com/${name}`,
      size: 1024,
      download_count: 0,
    })),
  });

  it('detects checksums.txt', () => {
    const release = createReleaseWithAssets(['binary.tar.gz', 'checksums.txt']);
    const result = detectChecksumAsset(release);

    expect(result).not.toBeNull();
    expect(result?.name).toBe('checksums.txt');
  });

  it('detects SHA256SUMS', () => {
    const release = createReleaseWithAssets(['binary.tar.gz', 'SHA256SUMS']);
    const result = detectChecksumAsset(release);

    expect(result).not.toBeNull();
    expect(result?.name).toBe('SHA256SUMS');
  });

  it('detects checksums.sha256', () => {
    const release = createReleaseWithAssets(['binary.tar.gz', 'checksums.sha256']);
    const result = detectChecksumAsset(release);

    expect(result).not.toBeNull();
    expect(result?.name).toBe('checksums.sha256');
  });

  it('detects per-asset .sha256 files', () => {
    const release = createReleaseWithAssets([
      'binary-linux.tar.gz',
      'binary-linux.tar.gz.sha256',
      'binary-darwin.tar.gz',
      'binary-darwin.tar.gz.sha256',
    ]);
    const result = detectChecksumAsset(release);

    expect(result).not.toBeNull();
    expect(result?.name).toContain('.sha256');
  });

  it('returns null when no checksums found', () => {
    const release = createReleaseWithAssets(['binary.tar.gz', 'README.md']);
    const result = detectChecksumAsset(release);

    expect(result).toBeNull();
  });

  it('prioritizes checksums.txt over other legacy formats', () => {
    const release = createReleaseWithAssets([
      'SHA256SUMS',
      'checksums.txt',
      'binary.tar.gz.sha256',
    ]);
    const result = detectChecksumAsset(release);

    expect(result?.name).toBe('checksums.txt');
    expect(result?.source).toBe('legacy');
  });

  // First-class API tests
  it('detects checksums.json as first-class API', () => {
    const release = createReleaseWithAssets(['binary.tar.gz', 'checksums.json']);
    const result = detectChecksumAsset(release);

    expect(result).not.toBeNull();
    expect(result?.name).toBe('checksums.json');
    expect(result?.source).toBe('api');
  });

  it('prioritizes checksums.json over all legacy formats', () => {
    const release = createReleaseWithAssets([
      'checksums.txt',
      'SHA256SUMS',
      'checksums.json',
      'binary.tar.gz.sha256',
    ]);
    const result = detectChecksumAsset(release);

    expect(result?.name).toBe('checksums.json');
    expect(result?.source).toBe('api');
  });

  it('falls back to legacy checksums.txt when checksums.json missing', () => {
    const release = createReleaseWithAssets([
      'checksums.txt',
      'SHA256SUMS',
      'binary.tar.gz',
    ]);
    const result = detectChecksumAsset(release);

    expect(result?.name).toBe('checksums.txt');
    expect(result?.source).toBe('legacy');
  });

  it('marks SHA256SUMS as legacy source', () => {
    const release = createReleaseWithAssets(['binary.tar.gz', 'SHA256SUMS']);
    const result = detectChecksumAsset(release);

    expect(result?.name).toBe('SHA256SUMS');
    expect(result?.source).toBe('legacy');
  });

  it('marks per-asset .sha256 files as legacy source', () => {
    const release = createReleaseWithAssets([
      'binary-linux.tar.gz',
      'binary-linux.tar.gz.sha256',
    ]);
    const result = detectChecksumAsset(release);

    expect(result?.source).toBe('legacy');
  });
});

describe('computeReleaseSelection', () => {
  const createRelease = (tag: string, prerelease: boolean, assets: string[] = ['checksums.txt']): GitHubRelease => ({
    id: parseInt(tag.replace(/\D/g, '')),
    tag_name: tag,
    name: tag,
    body: '',
    published_at: '2025-01-01T00:00:00Z',
    html_url: '',
    prerelease,
    draft: false,
    assets: assets.map(name => ({
      name,
      browser_download_url: `https://example.com/${name}`,
      size: 1024,
      download_count: 0,
    })),
  });

  it('selects latest stable release by default (includePrereleases=false)', () => {
    const releases = [
      createRelease('v0.2.6', true),  // prerelease
      createRelease('v0.2.5', false), // stable
      createRelease('v0.2.4', false), // stable
    ];

    const state = computeReleaseSelection(releases, false);

    expect(state.selectedRelease?.tag_name).toBe('v0.2.5');
    expect(state.latestStable?.tag_name).toBe('v0.2.5');
    expect(state.latestPrerelease?.tag_name).toBe('v0.2.6');
    expect(state.includePrereleases).toBe(false);
  });

  it('selects highest semver overall when includePrereleases=true', () => {
    const releases = [
      createRelease('v0.2.6', true),  // prerelease (highest)
      createRelease('v0.2.5', false), // stable
    ];

    const state = computeReleaseSelection(releases, true);

    expect(state.selectedRelease?.tag_name).toBe('v0.2.6');
    expect(state.includePrereleases).toBe(true);
  });

  it('filters out draft releases', () => {
    const releases: GitHubRelease[] = [
      { ...createRelease('v0.2.6', false), draft: true },
      createRelease('v0.2.5', false),
    ];

    const state = computeReleaseSelection(releases, false);

    expect(state.selectedRelease?.tag_name).toBe('v0.2.5');
    expect(state.releases.length).toBe(1); // Only non-draft
  });

  it('correctly detects checksums for selected release', () => {
    const releases = [
      createRelease('v0.2.5', false, ['checksums.txt']),
    ];

    const state = computeReleaseSelection(releases, false);

    expect(state.hasChecksums).toBe(true);
    expect(state.checksumAsset?.name).toBe('checksums.txt');
  });

  it('returns hasChecksums=false when selected release has no checksums', () => {
    const releases = [
      createRelease('v0.2.5', false, ['binary.tar.gz']), // No checksums
    ];

    const state = computeReleaseSelection(releases, false);

    expect(state.hasChecksums).toBe(false);
    expect(state.checksumAsset).toBeNull();
  });

  it('sorts releases by semver', () => {
    const releases = [
      createRelease('v0.2.5', false),
      createRelease('v0.2.10', false),
      createRelease('v0.2.9', false),
    ];

    const state = computeReleaseSelection(releases, false);

    expect(state.releases[0].tag_name).toBe('v0.2.10');
    expect(state.releases[1].tag_name).toBe('v0.2.9');
    expect(state.releases[2].tag_name).toBe('v0.2.5');
  });

  // Checksum API tests
  it('sets checksumSource to "api" when checksums.json is present', () => {
    const releases = [
      createRelease('v0.2.5', false, ['checksums.json', 'binary.tar.gz']),
    ];

    const state = computeReleaseSelection(releases, false);

    expect(state.hasChecksums).toBe(true);
    expect(state.checksumAsset?.name).toBe('checksums.json');
    expect(state.checksumSource).toBe('api');
  });

  it('sets checksumSource to "legacy" when using checksums.txt', () => {
    const releases = [
      createRelease('v0.2.5', false, ['checksums.txt', 'binary.tar.gz']),
    ];

    const state = computeReleaseSelection(releases, false);

    expect(state.hasChecksums).toBe(true);
    expect(state.checksumAsset?.name).toBe('checksums.txt');
    expect(state.checksumSource).toBe('legacy');
  });

  it('sets checksumSource to null when no checksums present', () => {
    const releases = [
      createRelease('v0.2.5', false, ['binary.tar.gz']),
    ];

    const state = computeReleaseSelection(releases, false);

    expect(state.hasChecksums).toBe(false);
    expect(state.checksumAsset).toBeNull();
    expect(state.checksumSource).toBeNull();
  });

  // User acceptance criteria tests
  it('stable release with checksum API → no warning (hasChecksums=true, source=api)', () => {
    const releases = [
      createRelease('v1.0.0', false, ['checksums.json', 'binary.tar.gz']),
    ];

    const state = computeReleaseSelection(releases, false);

    expect(state.selectedRelease?.prerelease).toBe(false);
    expect(state.hasChecksums).toBe(true);
    expect(state.checksumSource).toBe('api');
    expect(state.checksumAsset?.name).toBe('checksums.json');
  });

  it('missing checksum API → warning (hasChecksums=false, source=null)', () => {
    const releases = [
      createRelease('v1.0.0', false, ['binary.tar.gz']), // No checksums
    ];

    const state = computeReleaseSelection(releases, false);

    expect(state.selectedRelease?.prerelease).toBe(false);
    expect(state.hasChecksums).toBe(false);
    expect(state.checksumSource).toBeNull();
    expect(state.checksumAsset).toBeNull();
  });

  it('pre-release + checksum API → allowed but flagged (prerelease=true, source=api)', () => {
    const releases = [
      createRelease('v1.0.0-beta.1', true, ['checksums.json', 'binary.tar.gz']),
    ];

    const state = computeReleaseSelection(releases, true);

    expect(state.selectedRelease?.prerelease).toBe(true);
    expect(state.hasChecksums).toBe(true);
    expect(state.checksumSource).toBe('api');
    expect(state.checksumAsset?.name).toBe('checksums.json');
  });

  it('prefers checksums.json over legacy formats in selected release', () => {
    const releases = [
      createRelease('v1.0.0', false, ['checksums.json', 'checksums.txt', 'SHA256SUMS']),
    ];

    const state = computeReleaseSelection(releases, false);

    expect(state.checksumAsset?.name).toBe('checksums.json');
    expect(state.checksumSource).toBe('api');
  });
});

describe('isSelectedPrerelease', () => {
  it('returns true when selected release is a prerelease', () => {
    const state = computeReleaseSelection([
      { id: 1, tag_name: 'v1.0.0-alpha', name: '', body: '', published_at: '', html_url: '', prerelease: true, draft: false, assets: [] },
    ], true);

    expect(isSelectedPrerelease(state)).toBe(true);
  });

  it('returns false when selected release is stable', () => {
    const state = computeReleaseSelection([
      { id: 1, tag_name: 'v1.0.0', name: '', body: '', published_at: '', html_url: '', prerelease: false, draft: false, assets: [] },
    ], false);

    expect(isSelectedPrerelease(state)).toBe(false);
  });
});

describe('hasNewerPrerelease', () => {
  const createRelease = (tag: string, prerelease: boolean): GitHubRelease => ({
    id: 1,
    tag_name: tag,
    name: '',
    body: '',
    published_at: '',
    html_url: '',
    prerelease,
    draft: false,
    assets: [],
  });

  it('returns true when prerelease is newer than stable', () => {
    const state = computeReleaseSelection([
      createRelease('v0.2.6', true),  // prerelease
      createRelease('v0.2.5', false), // stable
    ], false);

    expect(hasNewerPrerelease(state)).toBe(true);
  });

  it('returns false when stable is newer than prerelease', () => {
    const state = computeReleaseSelection([
      createRelease('v0.2.5', false), // stable
      createRelease('v0.2.4', true),  // prerelease
    ], false);

    expect(hasNewerPrerelease(state)).toBe(false);
  });

  it('returns false when no prerelease exists', () => {
    const state = computeReleaseSelection([
      createRelease('v0.2.5', false), // stable only
    ], false);

    expect(hasNewerPrerelease(state)).toBe(false);
  });
});
