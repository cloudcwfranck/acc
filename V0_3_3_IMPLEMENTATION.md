# v0.3.3 Implementation Guide: Remote Attestation Validity + Policy Thresholds

## Status: In Progress

### Completed ✅
1. **Schema inspection** - Analyzed current attestation and verify result schemas
2. **Unit tests created** - `/home/user/acc/internal/trust/validate_test.go`
3. **Validation module implemented** - `/home/user/acc/internal/trust/validate.go`
4. **Config extended** - Added `TrustConfig` and `AttestationRequirements` to `config.go`

### Remaining Tasks

#### 1. Wire Validation into Trust Commands

**File: `/home/user/acc/internal/trust/verify.go`**

Add ResultsHashMatch field to AttestationDetail:
```go
type AttestationDetail struct {
	Path                    string `json:"path"`
	Timestamp               string `json:"timestamp"`
	VerificationStatus      string `json:"verificationStatus"`
	VerificationResultsHash string `json:"verificationResultsHash"`
	ValidSchema             bool   `json:"validSchema"`
	DigestMatch             bool   `json:"digestMatch"`
	ResultsHashMatch        bool   `json:"resultsHashMatch"`  // v0.3.3: ADD THIS
	InvalidReason           string `json:"invalidReason"`     // v0.3.3: ADD THIS
}
```

Update VerifyResult to include counts:
```go
type VerifyResult struct {
	SchemaVersion      string              `json:"schemaVersion"`
	ImageRef           string              `json:"imageRef"`
	ImageDigest        string              `json:"imageDigest"`
	VerificationStatus string              `json:"verificationStatus"`
	AttestationCount   int                 `json:"attestationCount"`
	ValidCount         int                 `json:"validCount"`         // v0.3.3: ADD THIS
	InvalidCount       int                 `json:"invalidCount"`       // v0.3.3: ADD THIS
	Attestations       []AttestationDetail `json:"attestations"`
	Errors             []string            `json:"errors"`
}
```

Modify `VerifyAttestations()` to:
1. Load config to check trust requirements
2. Compute expected results hash from verify state if available
3. Use `ValidateAttestationWithHash()` instead of `validateAttestation()`
4. Apply trust policy thresholds to determine final verification status
5. Set ValidCount and InvalidCount

#### 2. Update VerifyAttestations Implementation

Replace the validation loop (lines 86-98) with:
```go
// Step 3: Load config for trust requirements (v0.3.3)
cfg, err := config.Load("")
var trustReqs *config.AttestationRequirements
if err == nil && cfg.Trust.RequireAttestations != nil {
	trustReqs = cfg.Trust.RequireAttestations
}

// Step 3a: Compute expected results hash if verify state exists
expectedResultsHash := ""
if verifyState, err := loadLastVerifyState(); err == nil {
	if hash, err := computeResultsHash(verifyState); err == nil {
		expectedResultsHash = hash
	}
}

// Step 4: Validate each attestation with hash checking
validCount := 0
invalidCount := 0
for _, path := range attestPaths {
	detail := ValidateAttestationWithHash(path, digest, expectedResultsHash)

	// Convert to AttestationDetail for backwards compatibility
	legacyDetail := AttestationDetail{
		Path:                    detail.Path,
		Timestamp:               detail.Timestamp,
		VerificationStatus:      detail.VerificationStatus,
		VerificationResultsHash: detail.VerificationResultsHash,
		ValidSchema:             detail.ValidSchema,
		DigestMatch:             detail.DigestMatch,
		ResultsHashMatch:        detail.ResultsHashMatch,
		InvalidReason:           detail.InvalidReason,
	}
	result.Attestations = append(result.Attestations, legacyDetail)

	// Count valid vs invalid
	if IsAttestationValid(detail) {
		validCount++
	} else {
		invalidCount++
		result.Errors = append(result.Errors,
			fmt.Sprintf("Invalid attestation: %s (%s)",
				filepath.Base(path), detail.InvalidReason))
	}
}

result.ValidCount = validCount
result.InvalidCount = invalidCount

// Step 5: Apply trust requirements to determine status
if trustReqs != nil && trustReqs.Enabled {
	if validCount >= trustReqs.MinCount {
		result.VerificationStatus = "verified"
	} else {
		result.VerificationStatus = "unverified"
		result.Errors = append(result.Errors,
			fmt.Sprintf("Trust requirements not met: %d valid attestations, need %d",
				validCount, trustReqs.MinCount))
		return result, fmt.Errorf("trust requirements not met")
	}
} else {
	// Default behavior (backwards compatible): require all valid
	if validCount > 0 && invalidCount == 0 {
		result.VerificationStatus = "verified"
	} else {
		result.VerificationStatus = "unverified"
		return result, fmt.Errorf("attestation validation failed")
	}
}
```

#### 3. Add Helper Functions to verify.go

```go
// loadLastVerifyState loads the last verification state for results hash computation
func loadLastVerifyState() (map[string]interface{}, error) {
	stateFile := filepath.Join(".acc", "state", "last_verify.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, err
	}

	var state map[string]interface{}
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return state, nil
}

// computeResultsHash computes the canonical hash of verification results
// This matches the logic in internal/attest/attest.go:computeCanonicalHash
func computeResultsHash(state map[string]interface{}) (string, error) {
	result, ok := state["result"].(map[string]interface{})
	if !ok {
		result = make(map[string]interface{})
	}

	// Build canonical structure (must match attest.go logic)
	canonical := map[string]interface{}{
		"status":       state["status"],
		"violations":   result["violations"],
		"waivers":      result["waivers"],
		"sbomPresent":  result["sbomPresent"],
		"attestations": result["attestations"],
	}

	data, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", hash), nil
}
```

Don't forget to add imports:
```go
import (
	"crypto/sha256"
	"github.com/cloudcwfranck/acc/internal/config"
)
```

#### 4. Update Trust Status Command

**File: `/home/user/acc/internal/trust/status.go`**

Update StatusResult to include validity counts:
```go
type StatusResult struct {
	SchemaVersion string      `json:"schemaVersion"`
	ImageRef      string      `json:"imageRef"`
	Status        string      `json:"status"`
	ProfileUsed   string      `json:"profileUsed,omitempty"`
	Violations    []Violation `json:"violations"`
	Warnings      []Violation `json:"warnings"`
	SBOMPresent   bool        `json:"sbomPresent"`
	Attestations  []string    `json:"attestations"`
	ValidCount    int         `json:"validCount"`    // v0.3.3
	InvalidCount  int         `json:"invalidCount"`  // v0.3.3
}
```

#### 5. Add Tier 1 E2E Tests

**File: `/home/user/acc/scripts/e2e_smoke.sh`**

Add test cases for trust verification with v0.3.3:

```bash
#==========================================
# TEST 11: Trust Verification with Thresholds (v0.3.3)
#==========================================
test_trust_requirements() {
    log_test "TEST 11: Trust Verification with Attestation Requirements"

    # Create config with trust requirements
    cat > acc.yaml <<EOF
project:
  name: trust-test
policy:
  mode: enforce
trust:
  requireAttestations:
    enabled: true
    minCount: 1
    sources: ["local"]
    requireDigestMatch: true
    requireValidSchema: true
    requireResultsHashMatch: true
    mode: enforce
EOF

    # Test 11.1: No attestations - should fail
    run_test "verify without attestation should fail" \
        "$ACC_BIN trust verify --json test:latest" \
        1 \
        "unverified"

    # Test 11.2: Create valid attestation
    run_test "verify image" "$ACC_BIN verify test:latest"
    run_test "create attestation" "$ACC_BIN attest test:latest"

    # Test 11.3: Verify with valid attestation
    run_test "trust verify with valid attestation" \
        "$ACC_BIN trust verify --json test:latest" \
        0 \
        "verified"

    # Verify counts in JSON output
    OUTPUT=$($ACC_BIN trust verify --json test:latest)
    VALID_COUNT=$(echo "$OUTPUT" | jq -r '.validCount')
    INVALID_COUNT=$(echo "$OUTPUT" | jq -r '.invalidCount')

    if [ "$VALID_COUNT" != "1" ] || [ "$INVALID_COUNT" != "0" ]; then
        log_error "Expected validCount=1, invalidCount=0, got validCount=$VALID_COUNT, invalidCount=$INVALID_COUNT"
        return 1
    fi

    log_success "TEST 11 passed"
}
```

#### 6. Add Tier 2 Integration Tests

**File: `/home/user/acc/scripts/registry_integration.sh`**

Add cache corruption test:

```bash
# TEST 6.5: Cache corruption handling
log_test "Step 6.5: Test remote cache corruption handling"

# Tamper with a cached attestation
CACHE_DIR=".acc/attestations/${DIGEST_PREFIX}/remote/${GHCR_REGISTRY}/${GHCR_REPO}"
if [ -d "$CACHE_DIR" ]; then
    CACHE_FILE=$(ls "$CACHE_DIR"/*.json 2>/dev/null | head -1)
    if [ -n "$CACHE_FILE" ]; then
        # Corrupt the JSON
        echo '{"invalid": "json"' > "$CACHE_FILE"

        # Verify should handle gracefully
        $ACC_BIN trust verify --remote --json "$GHCR_IMAGE" || true

        # Should mark as invalid but not crash
        OUTPUT=$($ACC_BIN trust verify --remote --json "$GHCR_IMAGE" || echo '{}')
        INVALID_COUNT=$(echo "$OUTPUT" | jq -r '.invalidCount // 0')

        if [ "$INVALID_COUNT" -lt 1 ]; then
            log_error "Expected at least 1 invalid attestation after corruption"
            exit 1
        fi

        log_success "Cache corruption handled gracefully"
    fi
fi
```

#### 7. Update CHANGELOG.md

Add v0.3.3 section:

```markdown
### Added - v0.3.3: Attestation Validity and Policy Thresholds

**Summary**: Attestation validity checks and configurable trust requirements enable deterministic, policy-driven trust verification without cryptographic signatures.

**What's New:**

- ✅ **Results Hash Validation** - Attestations must reference correct verification results hash (integrity binding)
- ✅ **Configurable Trust Requirements** - Set minimum valid attestation count and validity checks via `trust.requireAttestations` in config
- ✅ **Multiple Attestation Support** - Evaluate and count valid vs invalid attestations deterministically
- ✅ **Cache Poisoning Protection** - Handle malformed remote attestations gracefully with structured error reporting
- ✅ **Extended JSON Output** - Added `validCount`, `invalidCount`, `resultsHashMatch`, `invalidReason` fields

**Configuration:**
```yaml
trust:
  requireAttestations:
    enabled: true
    minCount: 1
    sources: ["local", "remote"]
    requireDigestMatch: true
    requireValidSchema: true
    requireResultsHashMatch: true  # v0.3.3: integrity binding
    mode: enforce
```

**Validation Checks:**
1. **Schema**: Required fields present and correct types
2. **Digest Match**: `subject.imageDigest` matches resolved image digest
3. **Results Hash**: `evidence.verificationResultsHash` matches computed hash from verify state

**JSON Output Changes:**
- `VerifyResult`: Added `validCount`, `invalidCount`
- `AttestationDetail`: Added `resultsHashMatch`, `invalidReason`

**Backward Compatibility:**
- Default behavior unchanged (trust requirements disabled by default)
- Exit codes preserved (0/1/2)
- Existing fields unchanged

**Files Modified:**
- `internal/trust/validate.go` - New validation module
- `internal/trust/verify.go` - Integrated validity checks
- `internal/trust/status.go` - Added count fields
- `internal/config/config.go` - Added TrustConfig
- `scripts/e2e_smoke.sh` - TEST 11 for trust requirements
- `scripts/registry_integration.sh` - Cache corruption tests

**Testing:**
- Tier 0: Help output unchanged
- Tier 1: E2E tests with trust requirements
- Tier 2: Remote cache corruption handling
```

## Test Commands

### Run Unit Tests
```bash
go test ./internal/trust/... -v
```

### Run Tier 1 E2E
```bash
bash scripts/e2e_smoke.sh
```

### Run Tier 2 Integration
```bash
# Requires GHCR_REPO and credentials
export GHCR_REPO="owner/repo"
bash scripts/registry_integration.sh
```

### Build and Manual Test
```bash
go build -o acc ./cmd/acc

# Test without trust requirements (default)
./acc verify test:latest
./acc attest test:latest
./acc trust verify --json test:latest

# Test with trust requirements
cat > acc.yaml <<EOF
trust:
  requireAttestations:
    enabled: true
    minCount: 1
    requireResultsHashMatch: true
EOF

./acc trust verify --json test:latest
```

## Example JSON Output (v0.3.3)

```json
{
  "schemaVersion": "v0.3",
  "imageRef": "test:latest",
  "imageDigest": "abc123def456",
  "verificationStatus": "verified",
  "attestationCount": 2,
  "validCount": 2,
  "invalidCount": 0,
  "attestations": [
    {
      "path": ".acc/attestations/abc123/local/attestation1.json",
      "timestamp": "2025-01-01T00:00:00Z",
      "verificationStatus": "pass",
      "verificationResultsHash": "sha256:correct123",
      "validSchema": true,
      "digestMatch": true,
      "resultsHashMatch": true,
      "invalidReason": ""
    },
    {
      "path": ".acc/attestations/abc123/remote/ghcr.io/owner/repo/attestation2.json",
      "timestamp": "2025-01-01T01:00:00Z",
      "verificationStatus": "pass",
      "verificationResultsHash": "sha256:correct123",
      "validSchema": true,
      "digestMatch": true,
      "resultsHashMatch": true,
      "invalidReason": ""
    }
  ],
  "errors": []
}
```

## Migration Notes

1. **No breaking changes** - Trust requirements disabled by default
2. **JSON schema extended** - New fields added, existing fields unchanged
3. **Config addition** - New `trust` section is optional
4. **Exit codes preserved** - 0 (verified), 1 (unverified), 2 (unknown)

## Acceptance Criteria Checklist

- [ ] Tier 0/1/2 all pass
- [ ] No new top-level commands
- [ ] Default behavior unchanged unless config enables requirements
- [ ] Deterministic JSON output with stable ordering
- [ ] Exit codes preserved (0/1/2)
- [ ] Validity signals in both human and JSON output
- [ ] Cache corruption handled gracefully
- [ ] Results hash validation working
- [ ] Multiple attestations counted correctly
- [ ] Trust requirements enforced when enabled
