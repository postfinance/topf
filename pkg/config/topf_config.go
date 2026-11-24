// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package config contains the config structs and loading logic for Topf
package config

import (
	"errors"
	"fmt"
	"path/filepath"
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

// Parse decodes a topf.yaml document. Relative PatchesDir and SecretsPath are
// resolved against baseDir, the absolute directory containing the config file.
// Nodes are sorted control-plane first, then by host.
// Parse does not access the filesystem.
func Parse(content []byte, baseDir string) (*TopfConfig, error) {
	if !filepath.IsAbs(baseDir) {
		return nil, fmt.Errorf("base directory %q is not absolute", baseDir)
	}

	cfg := &TopfConfig{}
	if err := yaml.Unmarshal(content, cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	if cfg.ConfigDir != "" {
		return nil, errors.New("deprecated field 'configDir' found: use 'patchesDir' instead")
	}

	cfg.PatchesDir = resolvePath(baseDir, cfg.PatchesDir, ".")
	cfg.SecretsPath = resolvePath(baseDir, cfg.SecretsPath, "secrets.yaml")
	sortNodes(cfg.Nodes)

	return cfg, nil
}

// ParseNodes decodes a YAML list of nodes, as returned by a nodes provider.
func ParseNodes(content []byte) ([]Node, error) {
	var nodes []Node
	if err := yaml.Unmarshal(content, &nodes); err != nil {
		return nil, fmt.Errorf("failed to decode nodes: %w", err)
	}

	return nodes, nil
}

// AddNodes appends nodes and restores the node order.
func (t *TopfConfig) AddNodes(nodes []Node) {
	t.Nodes = append(t.Nodes, nodes...)
	sortNodes(t.Nodes)
}

// resolvePath resolves path against baseDir, using fallback if path is empty.
func resolvePath(baseDir, path, fallback string) string {
	if path == "" {
		path = fallback
	}

	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(baseDir, path)
}

// sortNodes sorts control-plane nodes first, then by host.
func sortNodes(nodes []Node) {
	slices.SortStableFunc(nodes, func(a, b Node) int {
		if a.Role == RoleControlPlane && b.Role == RoleWorker {
			return -1
		} else if a.Role == RoleWorker && b.Role == RoleControlPlane {
			return 1
		}

		return strings.Compare(a.Host, b.Host)
	})
}

// LoadFromFile loads the TopfConfig from a YAML file.
// PatchesDir defaults to the directory containing the config file.
// SecretsPath defaults to "secrets.yaml" next to the config file (not inside PatchesDir).
// Relative paths for both are resolved against the directory containing the config file.
func LoadFromFile(path string) (config *TopfConfig, secrets []string, err error) {
	var content []byte

	content, secrets, err = decryption.ReadFileWithSecrets(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read config file: %w", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve config path: %w", err)
	}

	config, err = Parse(content, filepath.Dir(absPath))
	if err != nil {
		return nil, nil, fmt.Errorf("config file %s: %w", path, err)
	}

	if config.NodesProvider != "" {
		provider := providers.NewBinaryNodesProvider(config.NodesProvider)

		nodesYAML, err := providers.LoadNodesYAML(provider, config.ClusterName)
		if err != nil {
			return nil, nil, err
		}

		nodes, err := ParseNodes(nodesYAML)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse nodes from provider: %w", err)
		}

		config.AddNodes(nodes)
	}

	return config, secrets, nil
}
