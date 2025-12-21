import Link from 'next/link';
import styles from './Navigation.module.css';

export default function Navigation() {
  return (
    <nav className={styles.nav}>
      <div className={styles.container}>
        <Link href="/" className={styles.logo}>
          <span className={styles.logoText}>acc</span>
        </Link>
        <div className={styles.links}>
          <Link href="/download">Download</Link>
          <Link href="/docs">Docs</Link>
          <Link href="/releases">Releases</Link>
          <a
            href="https://github.com/cloudcwfranck/acc"
            target="_blank"
            rel="noopener noreferrer"
          >
            GitHub
          </a>
        </div>
      </div>
    </nav>
  );
}
