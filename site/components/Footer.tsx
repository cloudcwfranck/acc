import styles from './Footer.module.css';

export default function Footer() {
  return (
    <footer className={styles.footer}>
      <div className={styles.container}>
        <div className={styles.content}>
          <div>
            <p className={styles.copyright}>
              © {new Date().getFullYear()} acc. Open source software.
            </p>
          </div>
          <div className={styles.links}>
            <a
              href="https://github.com/cloudcwfranck/acc"
              target="_blank"
              rel="noopener noreferrer"
            >
              GitHub
            </a>
            <a
              href="https://github.com/cloudcwfranck/acc/issues"
              target="_blank"
              rel="noopener noreferrer"
            >
              Issues
            </a>
            <a
              href="https://github.com/cloudcwfranck/acc/blob/main/docs/testing-contract.md"
              target="_blank"
              rel="noopener noreferrer"
            >
              Testing Contract
            </a>
          </div>
        </div>
      </div>
    </footer>
  );
}
