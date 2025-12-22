'use client';

import { Suspense, useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import Link from 'next/link';
import styles from './download.module.css';

interface ReleaseSelectionState {
  releases: Release[];
  latestStable: Release | null;
  latestPrerelease: Release | null;
  selectedRelease: Release | null;
  includePrereleases: boolean;
  checksumAsset: { name: string; url: string; source: 'api' | 'legacy' } | null;
  hasChecksums: boolean;
  checksumSource: 'api' | 'legacy' | null;
}

interface Release {
  tag_name: string;
  name: string;
  published_at: string;
  prerelease: boolean;
  draft: boolean;
  html_url: string;
  assets: Asset[];
}

interface Asset {
  name: string;
  browser_download_url: string;
  size: number;
}

function DownloadContent() {
  const searchParams = useSearchParams();
  const [state, setState] = useState<ReleaseSelectionState | null>(null);
  const [showPrerelease, setShowPrerelease] = useState(false);
  const [checksums, setChecksums] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Check URL param or localStorage for prerelease preference
    const urlPrerelease = searchParams?.get('prerelease') === '1';
    const storedPrerelease = localStorage.getItem('show-prereleases') === 'true';
    const initialPrerelease = urlPrerelease || storedPrerelease;
    setShowPrerelease(initialPrerelease);

    fetchReleaseState(initialPrerelease);
  }, [searchParams]);

  const fetchReleaseState = async (includePrereleases: boolean) => {
    try {
      // Fetch release selection state from API (single source of truth)
      const response = await fetch(`/api/releases/selection?includePrereleases=${includePrereleases}`);
      const selectionState: ReleaseSelectionState = await response.json();

      setState(selectionState);

      // Fetch checksums content if available
      if (selectionState.checksumAsset) {
        const checksumsResponse = await fetch(selectionState.checksumAsset.url);
        const checksumsText = await checksumsResponse.text();
        setChecksums(checksumsText);
      } else {
        setChecksums(null);
      }

      setLoading(false);
    } catch (error) {
      console.error('Failed to fetch release state:', error);
      setLoading(false);
    }
  };

  const togglePrerelease = () => {
    const newValue = !showPrerelease;
    setShowPrerelease(newValue);
    localStorage.setItem('show-prereleases', newValue.toString());
    fetchReleaseState(newValue);
  };

  if (loading) {
    return (
      <div className="container">
        <h1>Download acc</h1>
        <p>Loading release information...</p>
      </div>
    );
  }

  if (!state || !state.selectedRelease) {
    return (
      <div className="container">
        <h1>Download acc</h1>
        <p>No releases available.</p>
      </div>
    );
  }

  const { selectedRelease, latestStable, latestPrerelease, hasChecksums, checksumAsset, checksumSource } = state;

  // Group assets by OS
  const binaryAssets = selectedRelease.assets.filter(asset =>
    asset.name.match(/acc_.*\.(tar\.gz|zip)/)
  );

  const osGroups: Record<string, any[]> = {};
  binaryAssets.forEach(asset => {
    const match = asset.name.match(/acc_[\d.]+_(\w+)_(\w+)\.(tar\.gz|zip)/);
    if (match) {
      const [, os, arch, format] = match;
      if (!osGroups[os]) osGroups[os] = [];
      osGroups[os].push({ ...asset, os, arch, format });
    }
  });

  const getOSName = (os: string) => {
    const names: Record<string, string> = {
      'linux': 'Linux',
      'darwin': 'macOS',
      'windows': 'Windows',
    };
    return names[os] || os;
  };

  const getArchName = (arch: string) => {
    const names: Record<string, string> = {
      'amd64': 'x64',
      'arm64': 'ARM64',
    };
    return names[arch] || arch;
  };

  return (
    <div className="container">
      <h1>Download acc</h1>

      {/* Release selector */}
      <div className={styles.releaseSelector}>
        <div className={styles.version + ' ' + (selectedRelease.prerelease ? styles.prerelease : styles.stable)}>
          {selectedRelease.tag_name}
          <span className={styles.badge}>
            {selectedRelease.prerelease ? 'PRE-RELEASE' : 'STABLE'}
          </span>
        </div>

        <label className={styles.toggle}>
          <input
            type="checkbox"
            checked={showPrerelease}
            onChange={togglePrerelease}
          />
          <span className={styles.toggleLabel}>Include pre-releases</span>
        </label>
      </div>

      {/* Prerelease warning */}
      {selectedRelease.prerelease && (
        <div className={styles.warning}>
          ⚠️ This is a pre-release version. Not recommended for production use.
        </div>
      )}

      {/* Info message if prerelease exists but not selected */}
      {!showPrerelease && latestPrerelease && latestStable && (
        <div className={styles.info}>
          A pre-release version ({latestPrerelease.tag_name}) is available.
          Enable &quot;Include pre-releases&quot; to download it.
        </div>
      )}

      {/* Download buttons */}
      <section className={styles.downloads}>
        <h2>Download Binaries</h2>
        {Object.entries(osGroups).map(([os, assets]) => (
          <div key={os} className={styles.osGroup}>
            <h3>{getOSName(os)}</h3>
            <div className={styles.assetButtons}>
              {assets.map((asset: any) => (
                <a
                  key={asset.name}
                  href={asset.browser_download_url}
                  className={styles.downloadBtn}
                  download
                >
                  <span className={styles.arch}>{getArchName(asset.arch)}</span>
                  <span className={styles.format}>.{asset.format}</span>
                  <span className={styles.size}>{(asset.size / 1024 / 1024).toFixed(1)} MB</span>
                </a>
              ))}
            </div>
          </div>
        ))}
      </section>

      {/* Installation instructions */}
      <section className={styles.section}>
        <h2>Quick Install</h2>
        <div className={styles.instructions}>
          <h3>Linux / macOS</h3>
          <pre>
            {`# Download for your platform
VERSION="${selectedRelease.tag_name.replace('v', '')}"
OS="linux"    # or darwin
ARCH="amd64"  # or arm64

curl -LO "https://github.com/cloudcwfranck/acc/releases/download/${selectedRelease.tag_name}/acc_\${VERSION}_\${OS}_\${ARCH}.tar.gz"
${hasChecksums ? `curl -LO "https://github.com/cloudcwfranck/acc/releases/download/${selectedRelease.tag_name}/${checksumAsset?.name}"

# Verify checksum
sha256sum -c ${checksumAsset?.name} --ignore-missing  # Linux
# shasum -a 256 -c ${checksumAsset?.name} --ignore-missing  # macOS` : '# ⚠️  Checksums not available for this release'}

# Extract and install
tar -xzf "acc_\${VERSION}_\${OS}_\${ARCH}.tar.gz"
sudo mv acc /usr/local/bin/acc
chmod +x /usr/local/bin/acc

# Verify installation
acc version`}
          </pre>
        </div>
      </section>

      {/* Checksums */}
      {hasChecksums ? (
        <section className={styles.section}>
          <h2>Verify Downloads (SHA256)</h2>
          <p style={{ marginBottom: '1rem', color: 'rgba(var(--foreground-rgb), 0.7)' }}>
            Checksums available: <strong>Yes</strong> ({checksumAsset?.name})
            {checksumSource === 'api' && (
              <span style={{ marginLeft: '0.5rem', color: 'rgba(var(--foreground-rgb), 0.5)' }}>
                • API (checksums.json)
              </span>
            )}
            {checksumSource === 'legacy' && (
              <span style={{ marginLeft: '0.5rem', color: 'rgba(var(--foreground-rgb), 0.5)' }}>
                • Legacy format
              </span>
            )}
          </p>
          {checksums && <pre className={styles.checksums}>{checksums}</pre>}
          <div className={styles.verifySnippet}>
            <h3>Verification command:</h3>
            <pre>
              {`# Linux
sha256sum -c ${checksumAsset?.name} --ignore-missing

# macOS
shasum -a 256 -c ${checksumAsset?.name} --ignore-missing`}
            </pre>
          </div>
        </section>
      ) : (
        <section className={styles.section}>
          <h2>Checksums</h2>
          <div className={styles.warning}>
            ⚠️ Checksums not published for this release. Download verification unavailable.
          </div>
          <p style={{ marginTop: '1rem', color: 'rgba(var(--foreground-rgb), 0.7)' }}>
            Release maintainers should ensure checksums are included in future releases.
          </p>
        </section>
      )}

      {/* Release notes */}
      <section className={styles.section}>
        <h2>Release Notes</h2>
        <a
          href={selectedRelease.html_url}
          target="_blank"
          rel="noopener noreferrer"
          className={styles.releaseLink}
        >
          View full release notes on GitHub →
        </a>
      </section>
    </div>
  );
}

export default function DownloadPage() {
  return (
    <Suspense fallback={<div className="container"><p>Loading...</p></div>}>
      <DownloadContent />
    </Suspense>
  );
}
