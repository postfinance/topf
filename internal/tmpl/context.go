// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package tmpl

import "github.com/postfinance/topf/pkg/config"

// Context is the data available to patch and schematic templates: the
// complete topf.yaml configuration plus the node being rendered.
type Context struct {
	*config.TopfConfig

	// Node is the node being rendered.
	Node Node
}

// Node is the node being rendered: its configuration plus values
// determined at runtime.
type Node struct {
	config.Node

	// ResolvedSchematicID is the schematic ID after @-references are resolved.
	// Empty while rendering a schematic template.
	ResolvedSchematicID string
	// RuntimeData contains values collected from the live node.
	RuntimeData RuntimeData
}

// RuntimeData contains values collected from the live node.
// All fields are empty when rendering offline.
type RuntimeData struct {
	// TalosVersion is the Talos version reported by the live node (without v prefix).
	TalosVersion string
	// SchematicID is the schematic ID reported by the live node.
	SchematicID string
	// Stage is the machine stage reported by the live node.
	Stage string
}
