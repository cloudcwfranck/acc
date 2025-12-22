import Link from 'next/link';
import styles from './docs.module.css';

export const metadata = {
  title: 'Documentation - acc',
  description: 'Get started with acc policy verification CLI',
};

export default function DocsPage() {
  return (
    <div className="container">
      <div className={styles.header}>
        <h1>Documentation</h1>
        <p>Get started with acc in minutes</p>
      </div>

      <section className={styles.quickstart}>
        <h2>Quick Start</h2>
        <div className={styles.steps}>
          <div className={styles.step}>
            <h3>1. Install acc</h3>
            <pre>{`# Download latest release
curl -LO https://github.com/cloudcwfranck/acc/releases/latest/download/acc_linux_amd64.tar.gz

# Extract and install
tar -xzf acc_*.tar.gz
sudo mv acc /usr/local/bin/`}</pre>
          </div>

          <div className={styles.step}>
            <h3>2. Initialize a project</h3>
            <pre>{`acc init my-project

# This creates:
# .acc/           - Configuration directory
# .acc/profiles/  - Policy profiles
# acc.yaml        - Project config`}</pre>
          </div>

          <div className={styles.step}>
            <h3>3. Verify an artifact</h3>
            <pre>{`# Verify with JSON output
acc verify --json myimage:latest

# Check exit code:
# 0 = pass, 1 = fail, 2 = cannot complete
echo $?`}</pre>
          </div>

          <div className={styles.step}>
            <h3>4. Use in CI/CD</h3>
            <pre>{`# GitHub Actions example
- name: Verify image
  run: |
    acc verify --json myapp:latest
  # Fails the job if verification fails`}</pre>
          </div>
        </div>
      </section>

      <section className={styles.section}>
        <h2>Core Concepts</h2>
        <div className={styles.concepts}>
          <div className={styles.concept}>
            <h3>Policies</h3>
            <p>
              Write policies in Rego (OPA) stored in <code>.acc/policy/</code>.
              Policies define what&apos;s allowed and what&apos;s not.
            </p>
          </div>
          <div className={styles.concept}>
            <h3>Verification</h3>
            <p>
              Run <code>acc verify</code> to check artifacts against policies.
              Results are deterministic and machine-readable.
            </p>
          </div>
          <div className={styles.concept}>
            <h3>Exit Codes</h3>
            <p>
              <strong>0</strong>: Pass (compliant)
              <br />
              <strong>1</strong>: Fail (violations found)
              <br />
              <strong>2</strong>: Cannot complete (missing prerequisites)
            </p>
          </div>
        </div>
      </section>

      <section className={styles.section}>
        <h2>Further Reading</h2>
        <div className={styles.links}>
          <a
            href="https://github.com/cloudcwfranck/acc/blob/main/README.md"
            target="_blank"
            rel="noopener noreferrer"
            className={styles.docLink}
          >
            <span className={styles.linkTitle}>Full README</span>
            <span className={styles.linkDesc}>
              Complete guide including all commands, examples, and workflows
            </span>
          </a>

          <a
            href="https://github.com/cloudcwfranck/acc/blob/main/docs/testing-contract.md"
            target="_blank"
            rel="noopener noreferrer"
            className={styles.docLink}
          >
            <span className={styles.linkTitle}>Testing Contract</span>
            <span className={styles.linkDesc}>
              Exit codes, JSON schemas, and behavioral guarantees
            </span>
          </a>

          <a
            href="https://github.com/cloudcwfranck/acc/blob/main/CONTRIBUTING.md"
            target="_blank"
            rel="noopener noreferrer"
            className={styles.docLink}
          >
            <span className={styles.linkTitle}>Contributing Guide</span>
            <span className={styles.linkDesc}>
              How to run tests locally, development workflow, and contribution guidelines
            </span>
          </a>

          <a
            href="https://github.com/cloudcwfranck/acc/tree/main/docs"
            target="_blank"
            rel="noopener noreferrer"
            className={styles.docLink}
          >
            <span className={styles.linkTitle}>All Documentation</span>
            <span className={styles.linkDesc}>
              Browse all docs including architecture, threat model, and more
            </span>
          </a>
        </div>
      </section>

      <section className={styles.cta}>
        <h2>Ready to get started?</h2>
        <Link href="/download" className="btn btn-primary">
          Download Latest Release
        </Link>
      </section>
    </div>
  );
}
