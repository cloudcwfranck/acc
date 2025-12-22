'use client';

import { Suspense, useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import Link from 'next/link';
import styles from './download.module.css';

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
  const [stableRelease, setStableRelease] = useState<Release | null>(null);
  const [prereleaseRelease, setPrereleaseRelease] = useState<Release | null>(null);
  const [selectedRelease, setSelectedRelease] = useState<Release | null>(null);
  const [showPrerelease, setShowPrerelease] = useState(false);
  const [checksums, setChecksums] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Check URL param or localStorage for prerelease preference
    const urlPrerelease = searchParams?.get('prerelease') === '1';
    const storedPrerelease = localStorage.getItem('show-prereleases') === 'true';
    const initialPrerelease = urlPrerelease || storedPrerelease;
    setShowPrerelease(initialPrerelease);

    fetchReleases(initialPrerelease);
  }, [searchParams]);

  const fetchReleases = async (includePrereleases: boolean) => {
    try {
      const response = await fetch('/api/github/releases?limit=20');
      const releases = await response.json();

      const stable = releases.find((r: Release) => !r.prerelease && !r.draft);
      const prerelease = releases.find((r: Release) => r.prerelease && !r.draft);

      setStableRelease(stable || null);
      setPrereleaseRelease(prerelease || null);

      // Determine which release to show
      let selected = stable;
      if (includePrereleases && prerelease) {
        // Show prerelease if it's newer than stable
        const prereleaseDate = new Date(prerelease.published_at);
        const stableDate = stable ? new Date(stable.published_at) : new Date(0);
        if (prereleaseDate > stableDate) {
          selected = prerelease;
        }
      }

      setSelectedRelease(selected);

      // Fetch checksums for selected release
      if (selected) {
        const checksumsAsset = selected.assets.find((a: Asset) => a.name === 'checksums.txt');
        if (checksumsAsset) {
          const checksumsResponse = await fetch(checksumsAsset.browser_download_url);
          const checksumsText = await checksumsResponse.text();
          setChecksums(checksumsText);
        }
      }

      setLoading(false);
    } catch (error) {
      console.error('Failed to fetch releases:', error);
      setLoading(false);
    }
  };

  const togglePrerelease = () => {
    const newValue = !showPrerelease;
    setShowPrerelease(newValue);
    localStorage.setItem('show-prereleases', newValue.toString());
    fetchReleases(newValue);
  };

  if (loading) {
    return (
      <div className="container">
        <h1>Download acc</h1>
        <p>Loading release information...</p>
      </div>
    );
  }

  if (!selectedRelease) {
    return (
      <div className="container">
        <h1>Download acc</h1>
        <p>
          No releases available. Visit the{' '}
          <a href="https://github.com/cloudcwfranck/acc/releases">GitHub Releases page</a>.
        </p>
      </div>
    );
  }

  const publishedDate = new Date(selectedRelease.published_at).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });

  // Group assets by OS
  const binaries = selectedRelease.assets
    .map(asset => {
      const match = asset.name.match(/acc_[\d.]+_(\w+)_(\w+)\.(tar\.gz|zip)/);
      if (!match) return null;
      return {
        ...asset,
        os: match[1],
        arch: match[2],
        format: match[3],
      };
    })
    .filter(Boolean);

  const osGroups: Record<string, any[]> = {};
  binaries.forEach(binary => {
    if (binary) {
      if (!osGroups[binary.os]) {
        osGroups[binary.os] = [];
      }
      osGroups[binary.os].push(binary);
    }
  });

  const getOSName = (os: string) => {
    const names: Record<string, string> = {
      linux: 'Linux',
      darwin: 'macOS',
      windows: 'Windows',
    };
    return names[os] || os;
  };

  const getArchName = (arch: string) => {
    const names: Record<string, string> = {
      amd64: 'x64',
      arm64: 'ARM64',
    };
    return names[arch] || arch;
  };

  return (
    <div className="container">
      <div className={styles.header}>
        <h1>Download acc</h1>

        {/* Release selector */}
        <div className={styles.releaseSelector}>
          <div className={styles.releaseInfo}>
            <span className={`${styles.version} ${selectedRelease.prerelease ? styles.prerelease : styles.stable}`}>
              {selectedRelease.tag_name}
              {selectedRelease.prerelease && <span className={styles.badge}>PRE-RELEASE</span>}
              {!selectedRelease.prerelease && <span className={styles.badge}>STABLE</span>}
            </span>
            <span className={styles.date}>Released {publishedDate}</span>
          </div>

          <label className={styles.toggle}>
            <input
              type="checkbox"
              checked={showPrerelease}
              onChange={togglePrerelease}
            />
            <span>Include pre-releases</span>
          </label>
        </div>

        {selectedRelease.prerelease && (
          <div className={styles.warning}>
            ⚠️ This is a pre-release version. Not recommended for production use.
          </div>
        )}

        {!selectedRelease.prerelease && prereleaseRelease && showPrerelease && (
          <div className={styles.info}>
            ℹ️ A pre-release ({prereleaseRelease.tag_name}) is available but older than the stable release.
          </div>
        )}

        <p className={styles.description}>
          Policy verification CLI for deterministic, explainable results
        </p>
      </div>

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
curl -LO "https://github.com/cloudcwfranck/acc/releases/download/${selectedRelease.tag_name}/checksums.txt"

# Verify checksum
${checksums ? 'sha256sum -c checksums.txt --ignore-missing  # Linux\n# shasum -a 256 -c checksums.txt --ignore-missing  # macOS' : '# ⚠️  Checksums not available for this release'}

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
      {checksums ? (
        <section className={styles.section}>
          <h2>Verify Downloads (SHA256)</h2>
          <p style={{ marginBottom: '1rem', color: 'rgba(var(--foreground-rgb), 0.7)' }}>
            Verify the integrity of your download:
          </p>
          <pre className={styles.checksums}>{checksums}</pre>
          <div className={styles.verifySnippet}>
            <h3>Verification command:</h3>
            <pre>
              {`# Linux
sha256sum -c checksums.txt --ignore-missing

# macOS
shasum -a 256 -c checksums.txt --ignore-missing`}
            </pre>
          </div>
        </section>
      ) : (
        <section className={styles.section}>
          <h2>Checksums</h2>
          <div className={styles.warning}>
            ⚠️ Checksums not published for this release. Download verification unavailable.
          </div>
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
