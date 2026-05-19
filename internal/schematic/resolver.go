// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package schematic resolves @-prefixed schematic IDs by reading schematic files
// and submitting them to the Talos image factory API.
package schematic

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/postfinance/topf/pkg/config"
	"github.com/siderolabs/image-factory/pkg/schematic"
)

// Resolver resolves @-prefixed schematic IDs by reading schematic definition files,
// optionally rendering them as Go templates, and submitting them to the image factory API.
type Resolver struct {
	configDir string
	version   string
}

type cacheEntry struct {
	id  string
	err error
}

// Option configures a Resolver.
type Option func(*Resolver)

// NewResolver creates a new Resolver. configDir is used to resolve relative file paths.
func NewResolver(configDir, topfVersion string, opts ...Option) *Resolver {
	r := &Resolver{
		configDir: configDir,
		version:   topfVersion,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Resolve resolves a schematic ID. If the ID starts with @, it is treated as a path
// to a schematic file (relative to configDir). Files ending in .tpl are rendered
// through Go templates using the provided PatchContext as data.
// The rendered schematic is POSTed to https://<factory>/schematics and the returned
// hash ID is cached and returned.
// Non-@-prefixed IDs are returned unchanged.
func (r *Resolver) Resolve(ctx context.Context, schematicID string, tmplData *config.PatchContext) (string, error) {
	if !strings.HasPrefix(schematicID, "@") {
		return schematicID, nil
	}

	ref := strings.TrimPrefix(schematicID, "@")

	path := ref
	if !filepath.IsAbs(path) {
		path = filepath.Join(r.configDir, path)
	}

	content, err := r.readFile(path, tmplData)
	if err != nil {
		return "", fmt.Errorf("failed to read schematic file %s: %w", path, err)
	}

	scheme, err := schematic.Unmarshal(content)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal schematic file %s: %w", path, err)
	}
	id, err := scheme.ID()
	if err != nil {
		return "", fmt.Errorf("failed to determine id of schematic file %s: %w", path, err)
	}

	return id, err
}

func (r *Resolver) readFile(path string, tmplData *config.PatchContext) ([]byte, error) {
	if !strings.HasSuffix(path, ".tpl") {
		//nolint:gosec // loading arbitrary schematic files is by design
		return os.ReadFile(path)
	}

	return config.RenderTemplate(path, tmplData)
}
