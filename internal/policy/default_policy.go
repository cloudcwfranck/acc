package policy

// DefaultPolicyContent returns the default Rego policy for new projects
const DefaultPolicyContent = `# Default acc Policy
# This policy enforces basic security best practices for OCI workloads

package acc.policy

import rego.v1

# Default decision: deny
default allow := false

# Track all violations
violations := data.violations

# Rule: Container must not run as root
deny contains msg if {
	input.config.User == ""
	msg := {
		"rule": "no-root-user",
		"severity": "high",
		"result": "fail",
		"message": "Container runs as root (no USER directive found)",
	}
}

deny contains msg if {
	input.config.User == "root"
	msg := {
		"rule": "no-root-user",
		"severity": "high",
		"result": "fail",
		"message": "Container explicitly runs as root",
	}
}

deny contains msg if {
	input.config.User == "0"
	msg := {
		"rule": "no-root-user",
		"severity": "high",
		"result": "fail",
		"message": "Container runs as UID 0 (root)",
	}
}

# Rule: SBOM must be present
deny contains msg if {
	not input.sbom.present
	msg := {
		"rule": "sbom-required",
		"severity": "critical",
		"result": "fail",
		"message": "SBOM is required but not found",
	}
}

# Rule: Image must have labels
warn contains msg if {
	count(input.config.Labels) == 0
	msg := {
		"rule": "image-labels",
		"severity": "low",
		"result": "warn",
		"message": "Image has no labels (recommended for metadata)",
	}
}

# Rule: Attestation should be present for promotion
deny contains msg if {
	input.promotion == true
	not input.attestation.present
	msg := {
		"rule": "attestation-required-for-promotion",
		"severity": "critical",
		"result": "fail",
		"message": "Attestation required for promotion but not found",
	}
}

# Allow if no denials
allow if {
	count(deny) == 0
}

# Overall policy result
result := {
	"allow": allow,
	"violations": deny,
	"warnings": warn,
}
`
