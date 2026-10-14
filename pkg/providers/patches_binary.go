// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package providers

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
)

// NewBinaryPatchesProvider returns a PatchesProvider that delegates generating patches to a binary
func NewBinaryPatchesProvider(binaryPath string) PatchesProvider {
	return &binaryPatches{
		binaryPath: binaryPath,
	}
}

type binaryPatches struct {
	binaryPath string
}

func (p *binaryPatches) GetPatches(ctx context.Context, patchContext []byte) ([]byte, error) {
	//nolint:gosec // launching arbitrary binary is part of the design
	cmd := exec.CommandContext(ctx, p.binaryPath, "patches")
	cmd.Stdin = bytes.NewReader(patchContext)
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute patches provider command: %w", err)
	}

	return output, nil
}
