package providers

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

// DataProvider is an interface for getting cluster data from an external source
type DataProvider interface {
	// GetData returns the YAML representation of data for the given cluster
	GetData(clusterName string) ([]byte, error)
}

// LoadDataYAML gets the YAML bytes from a provider and validates it
func LoadDataYAML(provider DataProvider, clusterName string) (map[string]any, error) {
	dataYAML, err := provider.GetData(clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to load data from provider: %w", err)
	}

	var data map[string]any
	if err := yaml.Unmarshal(dataYAML, &data); err != nil {
		return nil, fmt.Errorf("failed to parse data provider output: invalid YAML: %w", err)
	}

	// Validate it's actually a map (not null, array, or scalar)
	if data == nil {
		return nil, errors.New("data provider returned null/empty data")
	}

	return data, nil
}
