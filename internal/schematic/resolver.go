// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package schematic resolves @-prefixed schematic IDs by reading schematic files
// and computing or submitting them to the Talos image factory API.
package schematic

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/postfinance/topf/pkg/config"
	factoryclient "github.com/siderolabs/image-factory/pkg/client"
	"github.com/siderolabs/image-factory/pkg/schematic"
	"golang.org/x/sync/singleflight"
)

// Resolver resolves @-prefixed schematic IDs by reading schematic definition files,
// optionally rendering them as Go templates, and computing or submitting schematic IDs.
// By default, schematic IDs are computed locally from the canonical YAML representation.
// When SubmitToFactory is set, schematics are submitted to the image factory API instead.
type Resolver struct {
	httpClient      *http.Client
	submitToFactory bool
	clients         map[string]*factoryclient.Client
	submitted       singleflight.Group
	mu              sync.Mutex
	logger          *slog.Logger
	configDir       string
	topfVersion     string
}

// Option configures a Resolver.
type Option func(*Resolver)

// WithHTTPClient sets a custom HTTP client for factory API calls.
func WithHTTPClient(c *http.Client) Option {
	return func(r *Resolver) {
		r.httpClient = c
	}
}

// WithLogger sets the logger for the resolver.
func WithLogger(l *slog.Logger) Option {
	return func(r *Resolver) {
		r.logger = l
	}
}

// WithSubmitToFactory enables submitting schematics to the image factory API.
// By default, schematic IDs are computed locally without network calls.
func WithSubmitToFactory(submit bool) Option {
	return func(r *Resolver) {
		r.submitToFactory = submit
	}
}

// NewResolver creates a new Resolver. configDir is used to resolve relative file paths.
func NewResolver(configDir, topfVersion string, opts ...Option) *Resolver {
	r := &Resolver{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		clients:     make(map[string]*factoryclient.Client),
		logger:      slog.Default(),
		configDir:   configDir,
		topfVersion: topfVersion,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

func (r *Resolver) getOrCreateClient(factory string) (*factoryclient.Client, error) {
	r.mu.Lock()
	if c, ok := r.clients[factory]; ok {
		r.mu.Unlock()

		return c, nil
	}
	r.mu.Unlock()

	baseURL := "https://" + factory

	c, err := factoryclient.New(baseURL, factoryclient.WithClient(*r.httpClient), withUserAgent(r.topfVersion))
	if err != nil {
		return nil, fmt.Errorf("failed to create factory client: %w", err)
	}

	r.mu.Lock()
	r.clients[factory] = c
	r.mu.Unlock()

	return c, nil
}

// Resolve resolves a schematic ID. If the ID starts with @, it is treated as a path
// to a schematic file (relative to configDir). Files ending in .tpl are rendered
// through Go templates using the provided PatchContext as data.
// By default, the schematic ID is computed locally from the canonical YAML hash.
// When SubmitToFactory is set, the schematic is submitted to the image factory API instead.
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

	sc, err := schematic.Unmarshal(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse schematic file %s: %w", path, err)
	}

	id, err := sc.ID()
	if err != nil {
		return "", fmt.Errorf("failed to compute schematic ID: %w", err)
	}

	if r.submitToFactory {
		return id, r.submitSchematic(ctx, factory, id, sc)
	}

	return id, nil
}

func (r *Resolver) submitSchematic(ctx context.Context, factory, id string, sc *schematic.Schematic) error {
	key := factory + ":" + id

	_, err, shared := r.submitted.Do(key, func() (any, error) {
		r.logger.Debug("submitting schematic to factory", "factory", factory, "id", id, "key", key)

		start := time.Now()

		c, err := r.getOrCreateClient(factory)
		if err != nil {
			return nil, err
		}

		if _, _, err := c.SchematicCreate(ctx, *sc); err != nil {
			if factoryclient.IsInvalidSchematicError(err) {
				return nil, fmt.Errorf("invalid schematic submitted to factory %s: %w", factory, err)
			}

			return nil, fmt.Errorf("failed to submit schematic to factory %s: %w", factory, err)
		}

		r.logger.Debug("schematic submitted to factory", "factory", factory, "id", id, "elapsed", time.Since(start))

		return true, nil
	})

	r.logger.Debug("submitSchematic result", "factory", factory, "id", id, "shared", shared, "err", err)

	return err
}

func (r *Resolver) readFile(path string, tmplData *config.PatchContext) ([]byte, error) {
	if !strings.HasSuffix(path, ".tpl") {
		//nolint:gosec // loading arbitrary schematic files is by design
		return os.ReadFile(path)
	}

	return config.RenderTemplate(path, tmplData)
}

func withUserAgent(version string) factoryclient.Option {
	return func(o *factoryclient.Options) {
		if o.ExtraHeaders == nil {
			o.ExtraHeaders = http.Header{}
		}

		o.ExtraHeaders.Set("User-Agent", "topf/"+version)
	}
}
