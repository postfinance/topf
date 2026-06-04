// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package config contains the config structs and loading logic for Topf
package config

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/postfinance/topf/internal/decryption"
	"github.com/postfinance/topf/pkg/providers"
	"go.yaml.in/yaml/v4"
)

// TopfConfig is the main configuration structure for a Topf cluster
type TopfConfig struct {
	ClusterName       string   `yaml:"clusterName"`
	ClusterEndpoint   Endpoint `yaml:"clusterEndpoint"`
	KubernetesVersion string   `yaml:"kubernetesVersion"`
	TalosVersion      string   `yaml:"talosVersion,omitempty"`
	SchematicID       string   `yaml:"schematicId,omitempty"`

	// Factory is the Talos image factory address (default: factory.talos.dev)
	Factory string `yaml:"factory,omitempty"`
	// Platform is the Talos platform identifier (default: metal)
	Platform string `yaml:"platform,omitempty"`
	// SecureBoot enables the secure boot installer variant
	SecureBoot bool `yaml:"secureboot,omitempty"`

	// SecretsProvider can be optionally set to the path of a binary which is
	// responsible for storing and retrieving secrets.yaml for a cluster. If not
	// set, will use a local secrets.yaml (see SecretsPath) with optional SOPS encryption.
	SecretsProvider string `yaml:"secretsProvider,omitempty"`

	// NodesProvider can be optionally set to the path of a binary which will provide additional nodes
	NodesProvider string `yaml:"nodesProvider,omitempty"`

	// PatchesDir is the directory containing patches and node-specific configurations.
	// Defaults to the directory containing the config file.
	PatchesDir string `yaml:"patchesDir,omitempty"`

	// ConfigDir is deprecated: use PatchesDir instead.
	ConfigDir string `yaml:"configDir,omitempty"`

	// SecretsPath is the path to the secrets.yaml file.
	// Relative paths are resolved against the directory containing the config file.
	// Defaults to "secrets.yaml" next to the config file.
	SecretsPath string `yaml:"secretsPath,omitempty"`

	Nodes []Node `yaml:"nodes"`

	// Data can contain arbitrary data that can be used when templating patches
	Data map[string]any `yaml:"data"`
}

// LoadFromFile loads the TopfConfig from a YAML file.
// PatchesDir defaults to the directory containing the config file.
// SecretsPath defaults to "secrets.yaml" next to the config file (not inside PatchesDir).
// Relative paths for both are resolved against the directory containing the config file.
func LoadFromFile(path string, nodesRegexFilter string, cache *decryption.Cache) (config *TopfConfig, secrets []string, err error) {
	// Read file with automatic SOPS decryption if needed
	var content []byte

	content, secrets, err = cache.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config = &TopfConfig{}

	err = yaml.Unmarshal(content, config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode config: %w", err)
	}

	if config.ConfigDir != "" {
		return nil, nil, fmt.Errorf("deprecated field 'configDir' found in %s: use 'patchesDir' instead", path)
	}

	// Resolve config file directory as absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve config path: %w", err)
	}

	configFileDir := filepath.Dir(absPath)

	// Set default patchesDir if not specified
	if config.PatchesDir == "" {
		config.PatchesDir = configFileDir
	} else if !filepath.IsAbs(config.PatchesDir) {
		config.PatchesDir = filepath.Join(configFileDir, config.PatchesDir)
	}

	// Set default secretsPath if not specified (next to topf.yaml, not inside patchesDir)
	if config.SecretsPath == "" {
		config.SecretsPath = filepath.Join(configFileDir, "secrets.yaml")
	} else if !filepath.IsAbs(config.SecretsPath) {
		config.SecretsPath = filepath.Join(configFileDir, config.SecretsPath)
	}

	nodesFilter := regexp.MustCompile(".*")

	if nodesRegexFilter != "" {
		nodesFilter, err = regexp.Compile(nodesRegexFilter)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid nodes selector regex: %w", err)
		}
	}

	// If a nodes provider is given, add those to the list of nodes
	if config.NodesProvider != "" {
		provider := providers.NewBinaryNodesProvider(config.NodesProvider)

		nodesYAML, err := providers.LoadNodesYAML(provider, config.ClusterName)
		if err != nil {
			return nil, nil, err
		}

		var nodes []Node
		if err := yaml.Unmarshal(nodesYAML, &nodes); err != nil {
			return nil, nil, fmt.Errorf("failed to parse nodes from provider: %w", err)
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

	return config, secrets, err
}

// GetSecretsProvider returns the configured secrets provider, or the default filesystem provider
func (t *TopfConfig) GetSecretsProvider(cache *decryption.Cache) providers.SecretsProvider {
	if t.SecretsProvider != "" {
		return providers.NewBinarySecretsProvider(t.SecretsProvider)
	}

	return providers.NewFilesystemSecretsProvider(t.SecretsPath, cache)
}
