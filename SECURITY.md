# Security Policy

## Supported Versions

The following versions of acc are currently supported with security updates:

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |
| < 0.1   | :x:                |

**Note:** As a v0.x project, acc is in active development. We will backport critical security fixes to the latest v0.x minor release, but we encourage users to stay on the latest version.

When acc reaches v1.0.0, we will establish a formal LTS (Long-Term Support) policy.

## Reporting a Vulnerability

**We take security seriously.** If you discover a security vulnerability in acc, please report it responsibly.

### Where to Report

**Preferred method: GitHub Security Advisories**

Report vulnerabilities privately through GitHub Security Advisories:

1. Go to [https://github.com/cloudcwfranck/acc/security/advisories/new](https://github.com/cloudcwfranck/acc/security/advisories/new)
2. Fill out the vulnerability details
3. Submit the advisory

This method ensures:
- Private communication with maintainers
- Coordinated disclosure timeline
- CVE assignment (if applicable)

**Alternative method: Email**

If you prefer email, contact: **security@example.com**
*(Note: Replace with actual security contact before v0.1.0 release)*

**Please encrypt sensitive reports using our PGP key:** *(Add PGP key or omit if not available)*

### What to Include in Your Report

To help us triage and fix the vulnerability quickly, please include:

- **Description** of the vulnerability
- **Steps to reproduce** the issue
- **Impact assessment** (what can an attacker do?)
- **Affected versions** (if known)
- **Suggested fix** (if you have one)
- **Proof of concept** (if applicable)

### What NOT to Report

The following are **not** security vulnerabilities:

- ❌ **Intentional policy enforcement** - acc is designed to block unverified workloads
- ❌ **Missing features** - Use GitHub Issues for feature requests
- ❌ **Supply chain issues in user images** - acc detects these; it's not a vulnerability in acc itself
- ❌ **Policy violations detected by acc** - This is expected behavior

### Response Timeline

- **Initial response:** Within 3 business days
- **Status update:** Within 7 days
- **Fix timeline:** Depends on severity (see below)

### Severity Levels and Response

| Severity | Description | Target Fix Timeline |
|----------|-------------|---------------------|
| **Critical** | RCE, privilege escalation, verification bypass | 7 days |
| **High** | Information disclosure, authentication bypass | 14 days |
| **Medium** | DoS, limited impact vulnerabilities | 30 days |
| **Low** | Minor issues with workarounds | Next release |

**Note:** Timelines are targets, not guarantees. Complex issues may take longer.

### Disclosure Policy

We follow **coordinated disclosure**:

1. You report the vulnerability privately
2. We confirm receipt and assess severity
3. We develop and test a fix
4. We prepare a security advisory
5. We release the fix and publish the advisory
6. **90-day embargo** - We ask that you wait 90 days or until the fix is released (whichever is sooner) before public disclosure

If you plan to publish research or speak about the vulnerability, please coordinate timing with us.

## Security Features of acc

acc is designed with security as a core principle:

### Threat Model

See [docs/threat-model.md](./docs/threat-model.md) for the complete threat model, including:

- **In-scope threats:** Supply chain attacks, policy bypass, config tampering
- **Out-of-scope threats:** Runtime attacks, registry compromise, DoS
- **Trust boundaries:** Filesystem (trusted), images (untrusted), network (untrusted)

### Security Guarantees

acc provides the following security guarantees:

1. **Verification Gates** - Failed verification blocks run/push/promote operations
2. **No Bypass Flags** - No `--skip-verify` or `--force` options
3. **No Silent Degradation** - Security failures are fatal, not warnings
4. **Waiver Expiry Enforcement** - Expired policy waivers cause verification failure
5. **Digest Validation** - Image digests are validated to prevent tag manipulation

### Known Limitations

acc explicitly does NOT:

- Prevent runtime attacks (use runtime security tools like Falco)
- Detect registry compromise (verify artifacts after pull)
- Provide DoS protection (deploy appropriate infrastructure controls)
- Scan for secrets in images (use dedicated secret scanners)
- Perform SAST/DAST (use dedicated code analysis tools)

See [docs/threat-model.md](./docs/threat-model.md) for complete non-goals.

## Security Updates

Security updates will be:

- Released as new patch versions (e.g., v0.1.1)
- Announced via GitHub Security Advisories
- Documented in [CHANGELOG.md](./CHANGELOG.md) under `### Security`
- Tagged with `[SECURITY]` prefix in commit messages

Subscribe to release notifications to stay informed:
- Watch this repository → Custom → Releases

## Security Best Practices for acc Users

When using acc in production:

1. **Pin versions** - Use specific versions, not `latest`
2. **Verify checksums** - Always verify release artifact checksums
3. **Use enforce mode** - Set `policy.mode: enforce` in production
4. **Audit waivers** - Review `.acc/waivers.yaml` regularly
5. **Rotate waivers** - Set expiry dates on all policy waivers
6. **Monitor state files** - Track `.acc/state/last_verify.json` in version control
7. **Review policies** - Audit `.acc/policy/` regularly for drift

## Responsible Disclosure Hall of Fame

We recognize and thank security researchers who responsibly disclose vulnerabilities:

*(List will be added as vulnerabilities are reported and fixed)*

## Questions?

For non-security questions, see [CONTRIBUTING.md](./CONTRIBUTING.md) or open a [GitHub Issue](https://github.com/cloudcwfranck/acc/issues).

For security issues, use the reporting methods above.

Thank you for helping keep acc secure!
