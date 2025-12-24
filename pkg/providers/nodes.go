package providers

import (
	"fmt"
)

// NodesProvider is an interface for getting nodes from an external source
type NodesProvider interface {
	// GetNodes returns the YAML representation of nodes for the given cluster
	GetNodes(clusterName string) ([]byte, error)
}

// LoadNodesYAML gets the YAML bytes from a provider
func LoadNodesYAML(provider NodesProvider, clusterName string) ([]byte, error) {
	nodesYAML, err := provider.GetNodes(clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to load nodes from provider: %w", err)
	}

	return nodesYAML, nil
}
