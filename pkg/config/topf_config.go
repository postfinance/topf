// Package config contains the config structs and loading logic for Topf
package config

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/postfinance/topf/pkg/providers"
	"go.yaml.in/yaml/v4"
)

// TopfConfig is the main configuration structure for a Topf cluster
type TopfConfig struct {
	ClusterName       string   `yaml:"clusterName"`
	ClusterEndpoint   Endpoint `yaml:"clusterEndpoint"`
	KubernetesVersion string   `yaml:"kubernetesVersion"`

	// SecretsProvider can be optionally set to the path of a binary which is
	// responsible for storing and retrieving secrets.yaml for a cluster. If not
	// set, will use a local secrest.yaml with optinoal SOPS encryption.
	SecretsProvider string `yaml:"secretsProvider,omitempty"`

	// NodesProvider can be optionally set the path of a binary which will provide additional noddes
	NodesProvider string `yaml:"nodesProvider,omitempty"`

	Nodes []Node `yaml:"nodes"`
}

// LoadFromFile loads the TopfConfig from a YAML file
func LoadFromFile(path string, nodesRegexFilter string) (config *TopfConfig, err error) {
	//nolint:gosec // we allow loading the config from arbitrary paths by design
	configFile, err := os.Open(path)
	if err != nil {
		return config, err
	}

	config = &TopfConfig{}

	err = yaml.NewDecoder(configFile).Decode(config)

	nodesFilter := regexp.MustCompile(".*")

	if nodesRegexFilter != "" {
		nodesFilter, err = regexp.Compile(nodesRegexFilter)
		if err != nil {
			return nil, fmt.Errorf("invalid nodes selector regex: %w", err)
		}
	}

	// If a nodes provider is given, add those to the list of nodes
	if config.NodesProvider != "" {
		provider := providers.NewBinaryNodesProvider(config.NodesProvider)

		nodesYAML, err := providers.LoadNodesYAML(provider, config.ClusterName)
		if err != nil {
			return nil, err
		}

		var nodes []Node
		if err := yaml.Unmarshal(nodesYAML, &nodes); err != nil {
			return nil, fmt.Errorf("failed to parse nodes from provider: %w", err)
		}

		for _, n := range nodes {
			if nodesFilter.MatchString(n.Host) {
				config.Nodes = append(config.Nodes, n)
			}
		}
	}

	// Sort nodes by role and hostname, such that control plane nodes come first
	slices.SortStableFunc(config.Nodes, func(a, b Node) int {
		if a.Role == RoleControlPlane && b.Role == RoleWorker {
			return -1
		} else if a.Role == RoleWorker && b.Role == RoleControlPlane {
			return 1
		}

		return strings.Compare(a.Host, b.Host)
	})

	return config, err
}

// GetSecretsProvider returns the configured secrets provider, or the default filesystem provider
func (t *TopfConfig) GetSecretsProvider() providers.SecretsProvider {
	if t.SecretsProvider != "" {
		return providers.NewBinarySecretsProvider(t.SecretsProvider)
	}

	return providers.NewFilesystemSecretsProvider()
}
