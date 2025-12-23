import Link from 'next/link';
import Image from 'next/image';
import DemoPlayer from '../components/DemoPlayer';

// Get demo source from environment
const ASCIINEMA_ID = process.env.NEXT_PUBLIC_ASCIINEMA_ID;
const LOCAL_CAST = ASCIINEMA_ID ? undefined : '/demo/demo.cast';

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
          <div style={{ textAlign: 'center', marginBottom: '2rem' }}>
            <h2 style={{ fontSize: '2rem', marginBottom: '0.5rem' }}>
              Interactive Demo
            </h2>
            <p style={{ color: 'rgba(var(--foreground-rgb), 0.7)', fontSize: '1.1rem' }}>
              See acc in action with a terminal walkthrough
            </p>
          </div>
          <DemoPlayer asciinemaId={ASCIINEMA_ID} localCastPath={LOCAL_CAST} />
          <div style={{ textAlign: 'center', marginTop: '1.5rem' }}>
            <Link href="/docs" className="btn btn-secondary" style={{ marginRight: '1rem' }}>
              View Quick Start Guide
            </Link>
            <a
              href="https://github.com/cloudcwfranck/acc"
              className="btn btn-secondary"
              target="_blank"
              rel="noopener noreferrer"
            >
              View on GitHub
            </a>
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
