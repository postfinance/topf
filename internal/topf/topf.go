// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package topf contains the internal implementations of Topf
package topf

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/postfinance/topf/internal/maskedwriter"
	"github.com/postfinance/topf/internal/schematic"
	"github.com/postfinance/topf/pkg/config"
	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
)

// Topf is the main interface to interact with a Topf cluster runtime
type Topf interface {
	// Config returns the cluster configuration
	Config() *config.TopfConfig

	// Secrets returns the lazy loaded secrets bundle
	Secrets() (*secrets.Bundle, error)

	// Logger returns the configured logger
	Logger() *slog.Logger

	// Nodes returns the list of nodes with additional information
	Nodes(context.Context) ([]*Node, error)

	// Render generates machine config bundles for all nodes.
	// When online is true, live nodes are queried for their actual running Talos version.
	Render(context.Context, bool) ([]*Node, error)

	// Writer returns a writer targeting os.Stdout. When the runtime was
	// created with Redact=true, secrets and certificates are replaced with
	// "*** redacted ***" before being written.
	Writer() io.Writer

	// AddSecretsToMask registers additional sensitive strings for redaction.
	// Has no effect when redaction is disabled.
	AddSecretsToMask(sensitive []string)

	// Confirm returns whether confirmation prompts are enabled
	Confirm() bool

	// TopfVersion returns the topf version string
	TopfVersion() string

	// ResolveSchematic resolves a schematic ID string. If the ID starts with @,
	// it is treated as a path to a schematic file (relative to the directory
	// containing topf.yaml) and the resolved hash is returned. Non-@-prefixed
	// IDs are returned unchanged.
	ResolveSchematic(ctx context.Context, factory, schematicID string, patchCtx *config.PatchContext) (string, error)
}

// RuntimeConfig contains configuration for creating a Topf runtime
type RuntimeConfig struct {
	// ConfigPath is the path to the topf.yaml configuration file
	ConfigPath string

	// NodesRegexFilter is an optional regex pattern to filter which nodes to operate on
	// Empty string means all nodes
	NodesRegexFilter string

	// LogLevel sets the logging verbosity (debug, info, warn, error)
	LogLevel string

	// Redact controls whether sensitive values are masked in output
	Redact bool

	// Confirm controls whether confirmation prompts are shown before destructive actions
	Confirm bool

	// SubmitToFactory controls whether schematics are submitted to the image factory API.
	// By default, schematic IDs are computed locally.
	SubmitToFactory bool

	// TopfVersion is the topf version string
	TopfVersion string
}

// NewTopfRuntime creates a new Topf runtime from the given configuration
func NewTopfRuntime(cfg RuntimeConfig) (Topf, error) {
	topfConfig, secrets, err := config.LoadFromFile(cfg.ConfigPath, cfg.NodesRegexFilter)
	if err != nil {
		return nil, err
	}

	// Validate patchesDir exists and is a directory
	if stat, err := os.Stat(topfConfig.PatchesDir); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("patches directory does not exist: %s", topfConfig.PatchesDir)
		}

		return nil, fmt.Errorf("failed to access patches directory: %w", err)
	} else if !stat.IsDir() {
		return nil, fmt.Errorf("patches path is not a directory: %s", topfConfig.PatchesDir)
	}

	// Parse log level
	level, err := parseLogLevel(cfg.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	// Create logger with TextHandler
	opts := &slog.HandlerOptions{
		Level: level,
	}
	handler := slog.NewTextHandler(os.Stderr, opts)
	logger := slog.New(handler)

	var mw *maskedwriter.Writer
	if cfg.Redact {
		mw = maskedwriter.New(os.Stdout, secrets)
	}

	return &topf{
		TopfConfig:   topfConfig,
		patchesDir:   topfConfig.PatchesDir,
		logger:       logger,
		maskedWriter: mw,
		confirm:      cfg.Confirm,
		version:      cfg.TopfVersion,
		resolver:     schematic.NewResolver(filepath.Dir(cfg.ConfigPath), cfg.TopfVersion, schematic.WithSubmitToFactory(cfg.SubmitToFactory), schematic.WithLogger(logger)),
	}, nil
}

type topf struct {
	*config.TopfConfig
	mu sync.Mutex

	patchesDir    string
	secretsBundle *secrets.Bundle
	logger        *slog.Logger
	maskedWriter  *maskedwriter.Writer
	confirm       bool
	version       string
	resolver      *schematic.Resolver
}

func (t *topf) Config() *config.TopfConfig {
	return t.TopfConfig
}

func (t *topf) TopfVersion() string {
	return t.version
}

// Logger returns the configured logger for this runtime
func (t *topf) Logger() *slog.Logger {
	return t.logger
}

func (t *topf) Writer() io.Writer {
	if t.maskedWriter != nil {
		return t.maskedWriter
	}

	return os.Stdout
}

func (t *topf) AddSecretsToMask(sensitive []string) {
	if t.maskedWriter != nil {
		t.maskedWriter.AddSecrets(sensitive)
	}
}

func (t *topf) Confirm() bool {
	return t.confirm
}

func (t *topf) ResolveSchematic(ctx context.Context, factory, schematicID string, patchCtx *config.PatchContext) (string, error) {
	return t.resolver.Resolve(ctx, factory, schematicID, patchCtx)
}

// parseLogLevel converts a string log level to slog.Level
func parseLogLevel(levelStr string) (slog.Level, error) {
	switch strings.ToLower(levelStr) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unknown log level %q, valid levels: debug, info, warn, error", levelStr)
	}
}
