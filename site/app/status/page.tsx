'use client';

import { useEffect, useState } from 'react';
import styles from './status.module.css';

interface HealthStatus {
  status: 'ok' | 'degraded' | 'down';
  timestamp: string;
  github: {
    reachable: boolean;
    rateLimitRemaining: number | null;
    latestStableTag: string | null;
    latestPrereleaseTag: string | null;
    assetsOk: boolean;
    checksumsPresent: boolean;
    checksumAsset?: string | null;
    checksumSource?: 'api' | 'legacy' | null;
    error?: string;
  };
}

export default function StatusPage() {
  const [health, setHealth] = useState<HealthStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [lastChecked, setLastChecked] = useState<string>('');

  const fetchHealth = async () => {
    try {
      const response = await fetch('/api/health');
      const data = await response.json();
      setHealth(data);
      setLastChecked(new Date().toLocaleString());
      setLoading(false);
    } catch (error) {
      console.error('Failed to fetch health status:', error);
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchHealth();
    // Refresh every 60 seconds
    const interval = setInterval(fetchHealth, 60000);
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <div className="container">
        <div className={styles.header}>
          <h1>System Status</h1>
          <p>Loading status information...</p>
        </div>
      </div>
    );
  }

  if (!health) {
    return (
      <div className="container">
        <div className={styles.header}>
          <h1>System Status</h1>
          <p className={styles.error}>Failed to load status information</p>
        </div>
      </div>
    );
  }

  const getStatusDisplay = (status: string) => {
    switch (status) {
      case 'ok':
        return { text: '✓ Operational', className: styles.statusOk };
      case 'degraded':
        return { text: '⚠ Degraded', className: styles.statusDegraded };
      case 'down':
        return { text: '✗ Down', className: styles.statusDown };
      default:
        return { text: '? Unknown', className: styles.statusUnknown };
    }
  };

  const statusDisplay = getStatusDisplay(health.status);

  return (
    <div className="container">
      <div className={styles.header}>
        <h1>System Status</h1>
        <div className={`${styles.statusBadge} ${statusDisplay.className}`}>
          {statusDisplay.text}
        </div>
        <p className={styles.lastChecked}>Last checked: {lastChecked}</p>
      </div>

      <section className={styles.section}>
        <h2>GitHub Releases Backend</h2>

        <div className={styles.metrics}>
          <div className={styles.metric}>
            <span className={styles.label}>API Reachable:</span>
            <span className={health.github.reachable ? styles.valueOk : styles.valueError}>
              {health.github.reachable ? 'Yes' : 'No'}
            </span>
          </div>

          {health.github.error && (
            <div className={styles.metric}>
              <span className={styles.label}>Error:</span>
              <span className={styles.valueError}>{health.github.error}</span>
            </div>
          )}

          {health.github.rateLimitRemaining !== null && (
            <div className={styles.metric}>
              <span className={styles.label}>Rate Limit Remaining:</span>
              <span className={styles.value}>{health.github.rateLimitRemaining}</span>
            </div>
          )}

          <div className={styles.metric}>
            <span className={styles.label}>Latest Stable Release:</span>
            <span className={health.github.latestStableTag ? styles.valueOk : styles.valueWarn}>
              {health.github.latestStableTag || 'None found'}
            </span>
          </div>

          <div className={styles.metric}>
            <span className={styles.label}>Latest Pre-release:</span>
            <span className={styles.value}>
              {health.github.latestPrereleaseTag || 'None'}
            </span>
          </div>

          <div className={styles.metric}>
            <span className={styles.label}>Assets Present:</span>
            <span className={health.github.assetsOk ? styles.valueOk : styles.valueError}>
              {health.github.assetsOk ? 'Yes' : 'No'}
            </span>
          </div>

          <div className={styles.metric}>
            <span className={styles.label}>Checksums Available:</span>
            <span className={health.github.checksumsPresent ? styles.valueOk : styles.valueWarn}>
              {health.github.checksumsPresent ? 'Yes' : 'No'}
              {health.github.checksumAsset && (
                <span style={{ marginLeft: '0.5rem', opacity: 0.7 }}>
                  ({health.github.checksumAsset})
                </span>
              )}
            </span>
          </div>

          {health.github.checksumSource && (
            <div className={styles.metric}>
              <span className={styles.label}>Checksum Source:</span>
              <span className={health.github.checksumSource === 'api' ? styles.valueOk : styles.value}>
                {health.github.checksumSource === 'api' ? 'API (checksums.json)' : 'Legacy format'}
              </span>
            </div>
          )}
        </div>
      </section>

      {health.status !== 'ok' && (
        <section className={styles.section}>
          <h2>Troubleshooting</h2>
          <div className={styles.troubleshooting}>
            {!health.github.reachable && (
              <div className={styles.issue}>
                <h3>GitHub API Unreachable</h3>
                <p>The website cannot connect to GitHub&apos;s API. This may be due to:</p>
                <ul>
                  <li>Network connectivity issues</li>
                  <li>GitHub API outage (check <a href="https://www.githubstatus.com/" target="_blank" rel="noopener noreferrer">githubstatus.com</a>)</li>
                  <li>Rate limiting (if many requests made recently)</li>
                </ul>
              </div>
            )}

            {health.github.reachable && !health.github.latestStableTag && (
              <div className={styles.issue}>
                <h3>No Stable Releases Found</h3>
                <p>No stable (non-prerelease) releases are available. This may indicate:</p>
                <ul>
                  <li>Only pre-release versions have been published</li>
                  <li>Releases are marked as drafts</li>
                </ul>
              </div>
            )}

            {health.github.latestStableTag && !health.github.assetsOk && (
              <div className={styles.issue}>
                <h3>Missing Release Assets</h3>
                <p>Expected binary assets are missing from the latest stable release.</p>
                <ul>
                  <li>Release may still be uploading</li>
                  <li>Build artifacts failed to upload</li>
                </ul>
              </div>
            )}

            {health.github.latestStableTag && !health.github.checksumsPresent && (
              <div className={styles.issue}>
                <h3>Checksums Not Published</h3>
                <p>No checksum files found in the latest release.</p>
                <ul>
                  <li>Download verification will not be available</li>
                  <li>Recommended: Publish checksums.json (first-class API)</li>
                  <li>Alternative: Include legacy checksum files (checksums.txt, SHA256SUMS, etc.)</li>
                </ul>
              </div>
            )}
          </div>
        </section>
      )}

      <section className={styles.section}>
        <h2>About This Page</h2>
        <p>
          This status page monitors the operational health of the acc website&apos;s
          GitHub Releases backend. The website fetches release information, download
          links, and checksums directly from GitHub&apos;s API.
        </p>
        <p>
          Status checks run every 60 seconds and are cached for performance.
          A status of &quot;Operational&quot; means downloads and release information are
          available. &quot;Degraded&quot; indicates partial functionality (e.g., missing
          checksums). &quot;Down&quot; means the backend is unreachable.
        </p>
      </section>

      <div className={styles.actions}>
        <button onClick={fetchHealth} className={styles.refreshBtn}>
          Refresh Status
        </button>
      </div>
    </div>
  );
}
