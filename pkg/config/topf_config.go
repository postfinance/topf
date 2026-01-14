// Package config contains the config structs and loading logic for Topf
package config

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/postfinance/topf/pkg/providers"
	"github.com/postfinance/topf/pkg/sops"
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

	// DataProvider can be optionally set to the path of a binary which will provide
	// additional data that gets merged with the data field. Provider data overrides
	// topf.yaml data.
	DataProvider string `yaml:"dataProvider,omitempty"`

	Nodes []Node `yaml:"nodes"`

	// Data can contain arbitrary data that can be used when templating patches
	Data map[string]any `yaml:"data"`
}

// LoadFromFile loads the TopfConfig from a YAML file
func LoadFromFile(path string, nodesRegexFilter string) (config *TopfConfig, err error) {
	// Read file with automatic SOPS decryption if needed
	content, err := sops.ReadFileWithSOPS(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if content == nil {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	config = &TopfConfig{}

	err = yaml.Unmarshal(content, config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	// If a data provider is given, merge its data with the config data
	if config.DataProvider != "" {
		provider := providers.NewBinaryDataProvider(config.DataProvider)

		providerData, err := providers.LoadDataYAML(provider, config.ClusterName)
		if err != nil {
			return nil, err
		}

		// Initialize Data map if nil
		if config.Data == nil {
			config.Data = make(map[string]any)
		}
		// Deep merge: provider overrides
		config.Data = deepMerge(config.Data, providerData)
	}

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

		config.Nodes = append(config.Nodes, nodes...)
	}

	config.Nodes = slices.DeleteFunc(config.Nodes, func(n Node) bool {
		return !nodesFilter.MatchString(n.Host)
	})

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
