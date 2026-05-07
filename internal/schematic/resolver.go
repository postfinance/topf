// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package schematic resolves @-prefixed schematic IDs by reading schematic files
// and submitting them to the Talos image factory API.
package schematic

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/postfinance/topf/pkg/config"
)

// Resolver resolves @-prefixed schematic IDs by reading schematic definition files,
// optionally rendering them as Go templates, and submitting them to the image factory API.
type Resolver struct {
	client    *http.Client
	cache     map[string]cacheEntry
	mu        sync.Mutex
	configDir string
	version   string
}

type cacheEntry struct {
	id  string
	err error
}

// Option configures a Resolver.
type Option func(*Resolver)

// WithClient sets a custom HTTP client on the Resolver.
func WithClient(c *http.Client) Option {
	return func(r *Resolver) {
		r.client = c
	}
}

// NewResolver creates a new Resolver. configDir is used to resolve relative file paths.
func NewResolver(configDir, topfVersion string, opts ...Option) *Resolver {
	r := &Resolver{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache:     make(map[string]cacheEntry),
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
func (r *Resolver) Resolve(ctx context.Context, factory, schematicID string, tmplData *config.PatchContext) (string, error) {
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

	cacheKey := factory + ":" + fmt.Sprintf("%x", sha256.Sum256(content))

	r.mu.Lock()
	defer r.mu.Unlock()

	if entry, ok := r.cache[cacheKey]; ok {
		return entry.id, entry.err
	}

	id, err := r.postSchematic(ctx, factory, content)
	if err != nil {
		err = fmt.Errorf("failed to submit schematic to factory %s: %w", factory, err)
	}

	r.cache[cacheKey] = cacheEntry{id: id, err: err}

	return id, err
}

func (r *Resolver) readFile(path string, tmplData *config.PatchContext) ([]byte, error) {
	if !strings.HasSuffix(path, ".tpl") {
		//nolint:gosec // loading arbitrary schematic files is by design
		return os.ReadFile(path)
	}

	return config.RenderTemplate(path, tmplData)
}

func (r *Resolver) postSchematic(ctx context.Context, factory string, content []byte) (string, error) {
	url := "https://" + factory + "/schematics"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(content))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/yaml")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "topf/"+r.version)

	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("factory returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if result.ID == "" {
		return "", errors.New("factory returned empty schematic ID")
	}

	return result.ID, nil
}
