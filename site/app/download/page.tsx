import { getLatestRelease, getChecksums, parseAssetInfo, getOSDisplayName, getArchDisplayName } from '@/lib/github';
import Link from 'next/link';
import styles from './download.module.css';

export const metadata = {
  title: 'Download acc - Latest Release',
  description: 'Download the latest version of acc for your platform',
};

export default async function DownloadPage() {
  const release = await getLatestRelease();

  if (!release) {
    return (
      <div className="container">
        <h1>Download acc</h1>
        <p>Unable to fetch release information. Please visit the <a href="https://github.com/cloudcwfranck/acc/releases">GitHub Releases page</a>.</p>
      </div>
    );
  }

  const checksums = await getChecksums(release);
  const publishedDate = new Date(release.published_at).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });

  // Group assets by OS and arch
  const binaries = release.assets
    .map(asset => {
      const info = parseAssetInfo(asset.name);
      if (!info) return null;
      return { ...asset, ...info };
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

  return (
    <div className="container">
      <div className={styles.header}>
        <h1>Download acc</h1>
        <div className={styles.releaseInfo}>
          <span className={styles.version}>{release.tag_name}</span>
          <span className={styles.date}>Released {publishedDate}</span>
        </div>
        <p className={styles.description}>
          Policy verification CLI for deterministic, explainable results
        </p>
      </div>

      {/* Download Buttons */}
      <section className={styles.downloads}>
        <h2>Download Binaries</h2>
        {Object.entries(osGroups).map(([os, assets]) => (
          <div key={os} className={styles.osGroup}>
            <h3>{getOSDisplayName(os)}</h3>
            <div className={styles.assetButtons}>
              {assets.map((asset: any) => (
                <a
                  key={asset.name}
                  href={asset.browser_download_url}
                  className={styles.downloadBtn}
                  download
                >
                  <span className={styles.arch}>{getArchDisplayName(asset.arch)}</span>
                  <span className={styles.format}>.{asset.format}</span>
                  <span className={styles.size}>
                    {(asset.size / 1024 / 1024).toFixed(1)} MB
                  </span>
                </a>
              ))}
            </div>
          </div>
        ))}
      </section>

      {/* Installation Instructions */}
      <section className={styles.section}>
        <h2>Quick Install</h2>

        <div className={styles.instructions}>
          <h3>Linux / macOS</h3>
          <pre>
{`# Download for your platform
VERSION="${release.tag_name.replace('v', '')}"
OS="linux"    # or darwin
ARCH="amd64"  # or arm64

curl -LO "https://github.com/cloudcwfranck/acc/releases/download/${release.tag_name}/acc_\${VERSION}_\${OS}_\${ARCH}.tar.gz"

# Extract and install
tar -xzf "acc_\${VERSION}_\${OS}_\${ARCH}.tar.gz"
sudo mv acc /usr/local/bin/acc
chmod +x /usr/local/bin/acc

# Verify installation
acc version`}
          </pre>
        </div>

        <div className={styles.instructions}>
          <h3>Windows (PowerShell)</h3>
          <pre>
{`# Download
$VERSION = "${release.tag_name.replace('v', '')}"
Invoke-WebRequest -Uri "https://github.com/cloudcwfranck/acc/releases/download/${release.tag_name}/acc_\${VERSION}_windows_amd64.zip" -OutFile "acc.zip"

# Extract
Expand-Archive -Path "acc.zip" -DestinationPath .

# Run
.\\acc.exe version`}
          </pre>
        </div>

        <div className={styles.instructions}>
          <h3>Using acc upgrade (if you have a previous version)</h3>
          <pre>{`acc upgrade --version ${release.tag_name}`}</pre>
        </div>
      </section>

      {/* Checksums */}
      {checksums && (
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
      )}

      {/* Release Notes */}
      <section className={styles.section}>
        <h2>Release Notes</h2>
        <a href={release.html_url} target="_blank" rel="noopener noreferrer" className={styles.releaseLink}>
          View full release notes on GitHub →
        </a>
      </section>
    </div>
  );
}
