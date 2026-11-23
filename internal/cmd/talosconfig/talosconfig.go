// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package talosconfig contains the logic to generate talosconfig file
package talosconfig

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/postfinance/topf/internal/topf"
	"github.com/postfinance/topf/pkg/config"
	"github.com/siderolabs/talos/pkg/machinery/config/bundle"
	"github.com/siderolabs/talos/pkg/machinery/config/generate"
	"github.com/siderolabs/talos/pkg/machinery/constants"
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
				KubeVersion: strings.TrimPrefix(cmp.Or(t.Config().KubernetesVersion, constants.DefaultKubernetesVersion), "v"),
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

	var endpoints, targets []string

	for _, node := range nodes {
		targets = append(targets, node.Address())

		endpoint := node.EffectiveEndpoint()
		if node.Role == config.RoleControlPlane && !slices.Contains(endpoints, endpoint) {
			endpoints = append(endpoints, endpoint)
		}
	}

	talosContext := configBundle.TalosCfg.Contexts[clusterName]

	if len(endpoints) > 0 {
		talosContext.Endpoints = endpoints
	}

	if len(targets) > 0 {
		talosContext.Nodes = targets
	}

	return configBundle.TalosCfg.Bytes()
}
