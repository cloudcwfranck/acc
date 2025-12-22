import { isPrereleaseNewer, parseAssetInfo, getOSDisplayName, getArchDisplayName } from '../lib/github';

describe('GitHub API Helpers', () => {
  describe('isPrereleaseNewer', () => {
    it('returns true when prerelease is newer than stable', () => {
      const prerelease = {
        id: 2,
        tag_name: 'v0.2.6',
        name: 'v0.2.6',
        body: '',
        published_at: '2025-12-22T00:00:00Z',
        html_url: '',
        prerelease: true,
        draft: false,
        assets: [],
      };

      const stable = {
        id: 1,
        tag_name: 'v0.2.5',
        name: 'v0.2.5',
        body: '',
        published_at: '2025-12-20T00:00:00Z',
        html_url: '',
        prerelease: false,
        draft: false,
        assets: [],
      };

      expect(isPrereleaseNewer(prerelease, stable)).toBe(true);
    });

    it('returns false when prerelease is older than stable', () => {
      const prerelease = {
        id: 1,
        tag_name: 'v0.2.4-rc1',
        name: 'v0.2.4-rc1',
        body: '',
        published_at: '2025-12-18T00:00:00Z',
        html_url: '',
        prerelease: true,
        draft: false,
        assets: [],
      };

      const stable = {
        id: 2,
        tag_name: 'v0.2.5',
        name: 'v0.2.5',
        body: '',
        published_at: '2025-12-20T00:00:00Z',
        html_url: '',
        prerelease: false,
        draft: false,
        assets: [],
      };

      expect(isPrereleaseNewer(prerelease, stable)).toBe(false);
    });
  });

  describe('parseAssetInfo', () => {
    it('parses Linux amd64 asset correctly', () => {
      const result = parseAssetInfo('acc_0.2.5_linux_amd64.tar.gz');
      expect(result).toEqual({
        os: 'linux',
        arch: 'amd64',
        format: 'tar.gz',
      });
    });

    it('parses macOS arm64 asset correctly', () => {
      const result = parseAssetInfo('acc_0.2.6_darwin_arm64.tar.gz');
      expect(result).toEqual({
        os: 'darwin',
        arch: 'arm64',
        format: 'tar.gz',
      });
    });

    it('parses Windows amd64 asset correctly', () => {
      const result = parseAssetInfo('acc_0.2.5_windows_amd64.zip');
      expect(result).toEqual({
        os: 'windows',
        arch: 'amd64',
        format: 'zip',
      });
    });

    it('returns null for invalid asset name', () => {
      const result = parseAssetInfo('README.md');
      expect(result).toBeNull();
    });
  });

  describe('getOSDisplayName', () => {
    it('returns "Linux" for linux', () => {
      expect(getOSDisplayName('linux')).toBe('Linux');
    });

    it('returns "macOS" for darwin', () => {
      expect(getOSDisplayName('darwin')).toBe('macOS');
    });

    it('returns "Windows" for windows', () => {
      expect(getOSDisplayName('windows')).toBe('Windows');
    });

    it('returns original string for unknown OS', () => {
      expect(getOSDisplayName('freebsd')).toBe('freebsd');
    });
  });

  describe('getArchDisplayName', () => {
    it('returns "x64" for amd64', () => {
      expect(getArchDisplayName('amd64')).toBe('x64');
    });

    it('returns "ARM64" for arm64', () => {
      expect(getArchDisplayName('arm64')).toBe('ARM64');
    });

    it('returns original string for unknown arch', () => {
      expect(getArchDisplayName('ppc64')).toBe('ppc64');
    });
  });
});

// Integration test concept (would require test doubles/mocks in practice)
describe('Release Selection Logic', () => {
  it('selects stable release by default', () => {
    const releases = [
      {
        id: 2,
        tag_name: 'v0.2.6',
        published_at: '2025-12-22T00:00:00Z',
        prerelease: true,
        draft: false,
      },
      {
        id: 1,
        tag_name: 'v0.2.5',
        published_at: '2025-12-20T00:00:00Z',
        prerelease: false,
        draft: false,
      },
    ];

    const stable = releases.find(r => !r.prerelease && !r.draft);
    expect(stable?.tag_name).toBe('v0.2.5');
  });

  it('includes prerelease when toggle is enabled and prerelease is newer', () => {
    const releases = [
      {
        id: 2,
        tag_name: 'v0.2.6',
        published_at: '2025-12-22T00:00:00Z',
        prerelease: true,
        draft: false,
      },
      {
        id: 1,
        tag_name: 'v0.2.5',
        published_at: '2025-12-20T00:00:00Z',
        prerelease: false,
        draft: false,
      },
    ];

    const includePrereleases = true;
    const stable = releases.find(r => !r.prerelease && !r.draft);
    const prerelease = releases.find(r => r.prerelease && !r.draft);

    let selected = stable;
    if (includePrereleases && prerelease && stable) {
      const prereleaseDate = new Date(prerelease.published_at);
      const stableDate = new Date(stable.published_at);
      if (prereleaseDate > stableDate) {
        selected = prerelease;
      }
    }

    expect(selected?.tag_name).toBe('v0.2.6');
    expect(selected?.prerelease).toBe(true);
  });

  it('excludes draft releases', () => {
    const releases = [
      {
        id: 3,
        tag_name: 'v0.2.7',
        published_at: '2025-12-23T00:00:00Z',
        prerelease: false,
        draft: true,
      },
      {
        id: 2,
        tag_name: 'v0.2.6',
        published_at: '2025-12-22T00:00:00Z',
        prerelease: true,
        draft: false,
      },
      {
        id: 1,
        tag_name: 'v0.2.5',
        published_at: '2025-12-20T00:00:00Z',
        prerelease: false,
        draft: false,
      },
    ];

    const stable = releases.find(r => !r.prerelease && !r.draft);
    expect(stable?.tag_name).toBe('v0.2.5'); // Should skip draft v0.2.7
  });
});
