// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package schematicids contains the logic for the schematic-ids command
package schematicids

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/postfinance/topf/internal/topf"
	"github.com/postfinance/topf/pkg/config"
)

// Execute resolves all schematic IDs for the configured nodes and prints
// them to w, one per line, deduplicated and sorted. Per-node schematicId
// overrides are honored. @-prefixed references are resolved via the
// schematic resolver.
func Execute(ctx context.Context, t topf.Topf, w io.Writer) error {
	cfg := t.Config()

	ids := make(map[string]struct{})

	for _, node := range cfg.Nodes {
		factory := cmp.Or(node.Factory, cfg.Factory, topf.DefaultFactory)
		schematicID := cmp.Or(node.SchematicID, cfg.SchematicID, topf.DefaultSchematic)

		patchCtx := &config.PatchContext{
			ClusterName:       cfg.ClusterName,
			ClusterEndpoint:   cfg.ClusterEndpoint.String(),
			KubernetesVersion: cfg.KubernetesVersion,
			TalosVersion:      cfg.TalosVersion,
			SchematicID:       schematicID,
			Data:              cfg.Data,
			Node:              &node,
		}

		resolved, err := t.ResolveSchematic(ctx, factory, schematicID, patchCtx)
		if err != nil {
			return fmt.Errorf("failed to resolve schematic for node %s: %w", node.Host, err)
		}

		ids[resolved] = struct{}{}
	}

	sorted := make([]string, 0, len(ids))
	for id := range ids {
		sorted = append(sorted, id)
	}

	sort.Strings(sorted)

	for _, id := range sorted {
		fmt.Fprintln(w, id)
	}

	return nil
}
