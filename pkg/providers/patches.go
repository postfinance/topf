// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package providers

import "context"

// PatchesProvider is an interface for getting machine config patches for a node from an external source
type PatchesProvider interface {
	// GetPatches returns the YAML representation of the patches for a single
	// node. patchContext is the JSON serialized template context of that node.
	GetPatches(ctx context.Context, patchContext []byte) ([]byte, error)
}
