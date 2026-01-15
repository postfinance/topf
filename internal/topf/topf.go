// Package topf contains the internal implementations of Topf
package topf

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

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
	topfConfig, err := config.LoadFromFile(cfg.ConfigPath, cfg.NodesRegexFilter)
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

	return &topf{
		TopfConfig: topfConfig,
		configDir:  topfConfig.ConfigDir,
		logger:     logger,
	}, nil
}

type topf struct {
	*config.TopfConfig
	mu sync.Mutex

	configDir     string
	secretsBundle *secrets.Bundle
	logger        *slog.Logger
}

func (t *topf) Config() *config.TopfConfig {
	return t.TopfConfig
}

// Logger returns the configured logger for this runtime
func (t *topf) Logger() *slog.Logger {
	return t.logger
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
