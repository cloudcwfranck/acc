import Link from 'next/link';
import Image from 'next/image';

export default function Home() {
  return (
    <>
      {/* Hero Section */}
      <section className="hero">
        <h1>Policy Verification for CI/CD</h1>
        <p>
          acc is a policy verification CLI that turns cloud controls into
          deterministic, explainable results for CI/CD.
        </p>
        <div className="cta-buttons">
          <Link href="/download" className="btn btn-primary">
            Download
          </Link>
          <Link href="/docs" className="btn btn-secondary">
            Documentation
          </Link>
        </div>
      </section>

      {/* Demo Section */}
      <section className="container">
        <div className="demo-container">
          <div className="demo-placeholder">
            <div style={{ fontSize: '3rem', marginBottom: '1rem' }}>🎬</div>
            <h3 style={{ marginBottom: '1rem', fontSize: '1.5rem' }}>
              Interactive Demo
            </h3>
            <p style={{ marginBottom: '2rem', fontSize: '1rem', lineHeight: '1.6' }}>
              See acc in action with a terminal walkthrough showing policy
              verification, trust inspection, and attestation workflows.
            </p>
            <Link href="/docs" className="btn btn-primary">
              View Quick Start Guide
            </Link>
            <p
              style={{
                marginTop: '2rem',
                fontSize: '0.875rem',
                color: 'rgba(var(--foreground-rgb), 0.4)',
              }}
            >
              Terminal demo recording coming soon •{' '}
              <a
                href="https://github.com/cloudcwfranck/acc"
                style={{ color: 'rgb(var(--primary-rgb))' }}
              >
                View on GitHub
              </a>
            </p>
          </div>
        </div>
      </section>

      {/* Trust / CI-ready Section */}
      <section className="section">
        <div className="container">
          <h2>Built for CI/CD Trust</h2>
          <div className="features">
            <div className="feature">
              <h3>Deterministic Exit Codes</h3>
              <p>
                Predictable exit codes (0=pass, 1=fail, 2=cannot complete)
                ensure your CI pipelines behave consistently.
              </p>
            </div>
            <div className="feature">
              <h3>Explainable Findings</h3>
              <p>
                Every violation includes rule name, severity, and clear
                descriptions—no black-box decisions.
              </p>
            </div>
            <div className="feature">
              <h3>Machine-Readable JSON</h3>
              <p>
                Structured JSON output with schema versioning makes integration
                with downstream tools trivial.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* How It Works Section */}
      <section className="section">
        <div className="container">
          <h2>How It Works</h2>
          <div className="steps">
            <div className="step">
              <div className="step-number">1</div>
              <div className="step-content">
                <h3>Define Policies</h3>
                <p>
                  Write policies in Rego (OPA) that enforce your security and
                  compliance requirements.
                </p>
              </div>
            </div>
            <div className="step">
              <div className="step-number">2</div>
              <div className="step-content">
                <h3>Verify Artifacts</h3>
                <p>
                  Run <code>acc verify</code> to check your images, SBOMs, and
                  configurations against policies.
                </p>
              </div>
            </div>
            <div className="step">
              <div className="step-number">3</div>
              <div className="step-content">
                <h3>Gate Deployments</h3>
                <p>
                  Use exit codes and JSON output in CI/CD to block
                  non-compliant workloads from production.
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Quick Start CTA */}
      <section className="section">
        <div className="container" style={{ textAlign: 'center' }}>
          <h2>Get Started</h2>
          <p style={{ marginBottom: '2rem', color: 'rgba(var(--foreground-rgb), 0.7)' }}>
            Download acc and run your first verification in minutes
          </p>
          <Link href="/download" className="btn btn-primary">
            Download Latest Release
          </Link>
        </div>
      </section>
    </>
  );
}
