# Intentional Policy Failure Example

This example demonstrates **verification gating** in acc by intentionally violating security policies.

## What This Example Shows

- How acc blocks unsafe workloads
- Policy violation detection
- Clear error messages with remediation
- "Red output means stop" principle

## Violations in This Example

1. **No USER directive** - Container runs as root (violates `no-root-user` policy)
2. **No SBOM** - Missing Software Bill of Materials
3. **No labels** - Missing metadata (warning only)

## How to Test

```bash
# Initialize acc project
acc init policy-test

# Build the image (will succeed)
acc build -t policy-test:latest

# Verify (will FAIL due to policy violations)
acc verify policy-test:latest

# Try to run (will be BLOCKED by verification gate)
acc run policy-test:latest

# Explain the failure
acc policy explain
```

## Expected Output

The verify and run commands should **fail** with clear error messages:

```
✖ Container runs as root (no USER directive found)
✖ SBOM is required but not found
```

## How to Fix

Add to Dockerfile:

```dockerfile
# Fix 1: Add non-root user
RUN adduser -D appuser
USER appuser

# Fix 2: Add labels
LABEL maintainer="your-email@example.com"
LABEL version="1.0"
```

Then rebuild:

```bash
acc build -t policy-test:latest
acc verify policy-test:latest  # Should now pass
```

## Key Principle

**Verification gates execution** - If verification fails, acc will NOT allow the workload to run, push, or promote. No bypass flags, no exceptions.
