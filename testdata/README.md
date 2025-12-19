# Test Data

This directory contains golden files for deterministic JSON output testing.

## Golden Tests

Golden tests ensure that JSON output remains stable and deterministic. They catch:
- Schema drift (added/removed fields)
- Field ordering changes
- Type changes
- Unexpected output variations

### Directory Structure

```
testdata/
├── golden/
│   ├── verify/           # Verify command JSON outputs
│   │   ├── pass.json
│   │   ├── fail-no-sbom.json
│   │   └── fail-policy-violations.json
│   └── inspect/          # Inspect command JSON outputs
│       ├── basic.json
│       ├── with-waivers.json
│       └── no-sbom.json
└── README.md
```

### How Golden Tests Work

1. **Golden files** contain expected JSON output for specific scenarios
2. **Test code** generates actual JSON output from code
3. **Comparison** verifies actual output matches golden file exactly
4. **Failure** occurs if output differs, indicating schema drift or bugs

### Timestamp Normalization

Some JSON outputs include timestamps that vary by test execution time:
- `inspect` results include `timestamp` field
- `inspect` metadata may include `lastVerified` field

Golden tests handle this by:
- Using fixed timestamps in test fixtures
- Comparing structure and values (timestamps are stable in fixtures)
- In production, timestamps are deterministic based on execution time

### Running Golden Tests

```bash
# Run all golden tests
go test ./internal/verify -v -run Golden
go test ./internal/inspect -v -run Golden

# Run all tests (including golden)
go test ./...
```

### Updating Golden Files

If schema changes are intentional:

1. Review the schema change carefully
2. Update the golden files to match new schema
3. Update `schemaVersion` if breaking change
4. Document the change in commit message

**Warning**: Never update golden files without understanding why they changed. Schema drift should be intentional and documented.

### Test Coverage

**Verify Golden Tests:**
- ✅ Pass scenario (all checks pass)
- ✅ Fail scenario (SBOM missing)
- ✅ Fail scenario (policy violations)
- ✅ Field ordering stability
- ✅ Schema version presence

**Inspect Golden Tests:**
- ✅ Basic scenario (SBOM + attestations)
- ✅ Waivers scenario (multiple waivers with expiry)
- ✅ No SBOM scenario (missing artifacts)
- ✅ Field ordering stability
- ✅ Schema version validation (v0.1)
- ✅ Schema drift detection

## Adding New Golden Tests

1. Create golden JSON file in appropriate directory
2. Add test case to `*_golden_test.go` file
3. Run test to verify it passes
4. Commit both golden file and test code

Example:

```go
{
    name:       "new-scenario",
    goldenFile: "testdata/golden/verify/new-scenario.json",
    result:     /* your test fixture */,
}
```

## Schema Version Policy

- `v0.1` - Initial schema version
- Increment minor version for backward-compatible additions
- Increment major version for breaking changes
- Golden tests ensure version is present and correct
