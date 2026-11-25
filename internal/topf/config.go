// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package topf

import (
	"fmt"
	"path/filepath"

	"github.com/postfinance/topf/internal/secrets"
	"github.com/postfinance/topf/pkg/config"
	"github.com/postfinance/topf/pkg/providers"
)

// loadConfig reads topf.yaml with secrets resolved and adds the nodes of the
// configured nodes provider, with vals references resolved.
func loadConfig(path string, resolver secrets.Resolver) (*config.TopfConfig, error) {
	content, err := resolver.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve config path: %w", err)
	}

	cfg, err := config.Parse(content, filepath.Dir(absPath))
	if err != nil {
		return nil, fmt.Errorf("config file %s: %w", path, err)
	}

	if cfg.NodesProvider == "" {
		return cfg, nil
	}

	nodesYAML, err := providers.LoadNodesYAML(providers.NewBinaryNodesProvider(cfg.NodesProvider), cfg.ClusterName)
	if err != nil {
		return nil, err
	}

	nodesYAML, err = resolver.Eval(nodesYAML)
	if err != nil {
		return nil, fmt.Errorf("resolve vals references in nodes provider output: %w", err)
	}

	nodes, err := config.ParseNodes(nodesYAML)
	if err != nil {
		return nil, fmt.Errorf("failed to parse nodes from provider: %w", err)
	}

	cfg.AddNodes(nodes)

	return cfg, nil
}
