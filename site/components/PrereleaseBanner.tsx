'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';
import styles from './PrereleaseBanner.module.css';

interface PrereleaseBannerProps {
  version: string;
  onEnablePrereleases?: () => void;
}

export default function PrereleaseBanner({ version, onEnablePrereleases }: PrereleaseBannerProps) {
  const [dismissed, setDismissed] = useState(false);

  useEffect(() => {
    // Check if user dismissed this specific prerelease banner
    const dismissedVersion = localStorage.getItem('prerelease-banner-dismissed');
    if (dismissedVersion === version) {
      setDismissed(true);
    }
  }, [version]);

  const handleDismiss = () => {
    localStorage.setItem('prerelease-banner-dismissed', version);
    setDismissed(true);
  };

  if (dismissed) {
    return null;
  }

  return (
    <div className={styles.banner}>
      <div className={styles.content}>
        <span className={styles.icon}>⚠️</span>
        <div className={styles.message}>
          <strong>Pre-release available: {version}</strong>
          <span className={styles.subtext}> (not recommended for production)</span>
        </div>
        <div className={styles.actions}>
          <Link href="/download?prerelease=1" className={styles.link}>
            View pre-release
          </Link>
          <button onClick={handleDismiss} className={styles.dismissBtn} aria-label="Dismiss banner">
            ✕
          </button>
        </div>
      </div>
    </div>
  );
}
