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
	"strings"
	"sync"

	"github.com/postfinance/topf/internal/maskedwriter"
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

	// MaskedPrinter returns a writer that redacts secrets from output.
	// Before secrets are loaded it passes through to os.Stdout unchanged.
	MaskedPrinter() io.Writer
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
}

// NewTopfRuntime creates a new Topf runtime from the given configuration
func NewTopfRuntime(cfg RuntimeConfig) (Topf, error) {
	topfConfig, secrets, err := config.LoadFromFile(cfg.ConfigPath, cfg.NodesRegexFilter)
	if err != nil {
		return nil, err
	}

	// Validate configDir exists and is a directory
	if stat, err := os.Stat(topfConfig.ConfigDir); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config directory does not exist: %s", topfConfig.ConfigDir)
		}

		return nil, fmt.Errorf("failed to access config directory: %w", err)
	} else if !stat.IsDir() {
		return nil, fmt.Errorf("config path is not a directory: %s", topfConfig.ConfigDir)
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

	mp := maskedwriter.NewMaskedWriter(os.Stdout, secrets)

	return &topf{
		TopfConfig:    topfConfig,
		configDir:     topfConfig.ConfigDir,
		logger:        logger,
		maskedPrinter: mp,
	}, nil
}

type topf struct {
	*config.TopfConfig
	mu sync.Mutex

	configDir     string
	secretsBundle *secrets.Bundle
	logger        *slog.Logger
	maskedPrinter *maskedwriter.Writer
}

func (t *topf) Config() *config.TopfConfig {
	return t.TopfConfig
}

// Logger returns the configured logger for this runtime
func (t *topf) Logger() *slog.Logger {
	return t.logger
}

// MaskedPrinter returns the writer that redacts loaded secrets from output
func (t *topf) MaskedPrinter() io.Writer {
	return t.maskedPrinter
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
