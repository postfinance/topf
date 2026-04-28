// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package topf

import (
	"cmp"
	"strings"

	"github.com/siderolabs/talos/pkg/machinery/version"
)

// Render generates machine config bundles for all configured nodes without connecting to a cluster.
// Falls back to the bundled Talos version when talosVersion is not set in topf.yaml.
// Errors during config generation for individual nodes are recorded in the Node.Error field.
func (t *topf) Render() ([]*Node, error) {
	cfg := t.Config()

	nodes := make([]*Node, 0, len(cfg.Nodes))

	for _, node := range cfg.Nodes {
		nodes = append(nodes, &Node{Node: &node, t: t})
	}

	talosVersion := strings.TrimPrefix(cmp.Or(cfg.TalosVersion, version.Tag), "v")

	for _, node := range nodes {
		node.TalosVersion = talosVersion
		node.Schematic = cmp.Or(cfg.SchematicID, DefaultSchematic)

		if err := t.generateNodeConfig(node); err != nil {
			node.Error = err
		}
	}

	return nodes, nil
}
