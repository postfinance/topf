// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

//go:build integration

// Package integration contains end-to-end tests for topf, run against a live
// Talos cluster behind the "integration" build tag.
package integration

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/postfinance/topf/internal/topf"
	"github.com/stretchr/testify/suite"
)

var (
	configPath       = flag.String("topf.config", "topf.yaml", "path to the topf config file")                          //nolint:gochecknoglobals // standard flag pattern
	expectedVersion  = flag.String("expected.talos.version", "", "expected Talos version after the upgrade (required)") //nolint:gochecknoglobals // standard flag pattern
	expectedNodesNum = flag.Int("expected.nodes", 3, "number of nodes in the cluster")                                  //nolint:gochecknoglobals // standard flag pattern
)

// TopfSuite is the shared base for all suites. The runtime is rebuilt via
// ReloadRuntime whenever the config file is mutated on disk.
type TopfSuite struct {
	suite.Suite

	configPath string

	runtime topf.Topf
}

func (s *TopfSuite) SetupSuite() {
	t, err := buildRuntime(s.configPath)
	s.Require().NoError(err, "failed to build topf runtime")

	s.runtime = t
}

func (s *TopfSuite) Runtime() topf.Topf {
	return s.runtime
}

func (s *TopfSuite) ReloadRuntime() {
	t, err := buildRuntime(s.configPath)
	s.Require().NoError(err, "failed to rebuild topf runtime after config change")

	s.runtime = t
}

func buildRuntime(configPath string) (topf.Topf, error) {
	return topf.NewTopfRuntime(topf.RuntimeConfig{
		ConfigPath:  configPath,
		Logger:      slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})),
		Redact:      true,
		Confirm:     false,
		TopfVersion: "integration-test",
	})
}

func (s *TopfSuite) Nodes(ctx context.Context) []*topf.Node {
	nodes, err := s.Runtime().FilteredNodes(ctx)
	s.Require().NoError(err)

	s.Len(nodes, *expectedNodesNum, "unexpected node count")

	return nodes
}
