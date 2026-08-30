// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package talosconfig contains the logic to generate talosconfig file
package talosconfig

import (
	"fmt"
	"strings"

	"github.com/postfinance/topf/internal/topf"
	"github.com/postfinance/topf/pkg/config"
	"github.com/siderolabs/talos/pkg/machinery/config/bundle"
	"github.com/siderolabs/talos/pkg/machinery/config/generate"
)

// Generate returns a talosconfig
func Generate(t topf.Topf) ([]byte, error) {
	secretsBundle, err := t.Secrets()
	if err != nil {
		return nil, fmt.Errorf("failed to load secrets bundle: %w", err)
	}

	configBundleOpts := []bundle.Option{
		bundle.WithVerbose(false), // prevent printing "generating PKI and tokens"
		bundle.WithInputOptions(
			&bundle.InputOptions{
				ClusterName: t.Config().ClusterName,
				KubeVersion: strings.TrimPrefix(t.Config().KubernetesVersion, "v"),
				GenOptions: []generate.Option{
					generate.WithSecretsBundle(secretsBundle),
				},
			},
		),
	}

	configBundle, err := bundle.NewBundle(configBundleOpts...)
	if err != nil {
		return nil, err
	}

	clusterName := t.Config().ClusterName
	nodes := t.Config().Nodes

	// For single-node clusters, add the single node as both endpoint and node
	if len(nodes) == 1 {
		endpoint := nodes[0].Endpoint()
		configBundle.TalosCfg.Contexts[clusterName].Endpoints = []string{endpoint}
		configBundle.TalosCfg.Contexts[clusterName].Nodes = []string{endpoint}
	} else {
		// For multi-node clusters:
		// - endpoints: all control-plane nodes
		// - nodes: all nodes
		var (
			endpoints []string
			allNodes  []string
		)

		for _, node := range nodes {
			allNodes = append(allNodes, node.Endpoint())
			if node.Role == config.RoleControlPlane {
				endpoints = append(endpoints, node.Endpoint())
			}
		}

		if len(endpoints) > 0 {
			configBundle.TalosCfg.Contexts[clusterName].Endpoints = endpoints
		}

		if len(allNodes) > 0 {
			configBundle.TalosCfg.Contexts[clusterName].Nodes = allNodes
		}
	}

	return configBundle.TalosCfg.Bytes()
}
