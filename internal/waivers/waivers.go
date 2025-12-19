package waivers

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Waiver represents a policy waiver with expiry
type Waiver struct {
	RuleID        string `yaml:"ruleId" json:"ruleId"`
	Justification string `yaml:"justification" json:"justification"`
	Expiry        string `yaml:"expiry" json:"expiry"`
	ApprovedBy    string `yaml:"approvedBy,omitempty" json:"approvedBy,omitempty"`
}

// WaiversFile represents the structure of .acc/waivers.yaml
type WaiversFile struct {
	Waivers []Waiver `yaml:"waivers"`
}

// IsExpired checks if the waiver has expired
func (w *Waiver) IsExpired() bool {
	if w.Expiry == "" {
		return false
	}

	expiryTime, err := time.Parse(time.RFC3339, w.Expiry)
	if err != nil {
		// If we can't parse the expiry, treat it as expired for safety
		return true
	}

	return time.Now().UTC().After(expiryTime)
}

// LoadWaivers loads policy waivers from .acc/waivers.yaml
func LoadWaivers() ([]Waiver, error) {
	waiversPath := filepath.Join(".acc", "waivers.yaml")

	// If waivers file doesn't exist, return empty list (no waivers)
	if _, err := os.Stat(waiversPath); os.IsNotExist(err) {
		return []Waiver{}, nil
	}

	data, err := os.ReadFile(waiversPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read waivers file: %w", err)
	}

	var waiversFile WaiversFile
	if err := yaml.Unmarshal(data, &waiversFile); err != nil {
		return nil, fmt.Errorf("failed to parse waivers file: %w", err)
	}

	return waiversFile.Waivers, nil
}

// GetWaiverForRule returns the waiver for a specific rule, if one exists
func GetWaiverForRule(waivers []Waiver, ruleID string) *Waiver {
	for i := range waivers {
		if waivers[i].RuleID == ruleID {
			return &waivers[i]
		}
	}
	return nil
}
