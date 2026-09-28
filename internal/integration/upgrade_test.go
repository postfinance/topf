// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

//go:build integration

package integration

import (
	"os"
	"regexp"
	"time"

	"github.com/postfinance/topf/internal/cmd/upgrade"
	"github.com/siderolabs/talos/cmd/talosctl/pkg/talos/nodedrain"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
	"github.com/siderolabs/talos/pkg/reporter"
)

// UpgradeSuite upgrades the cluster to the expected Talos version.
type UpgradeSuite struct {
	TopfSuite
}

func (s *UpgradeSuite) TestUpgrade() {
	s.bumpTalosVersion(*expectedVersion)
	s.ReloadRuntime()

	err := upgrade.Execute(s.T().Context(), s.Runtime(), upgrade.Options{
		RebootMode:            machine.RebootRequest_DEFAULT,
		Drain:                 true,
		DrainTimeout:          nodedrain.DefaultDrainTimeout,
		StabilizationDuration: 5 * time.Second,
		ReporterMode:          reporter.OutputModePlain,
	})
	s.Require().NoError(err, "upgrade failed")

	for _, node := range s.Nodes(s.T().Context()) {
		s.NoError(node.Error, "node %s has an error", node.Node.Host)
		s.Equal(runtime.MachineStageRunning, node.MachineStatus.Stage, "node %s not in Running stage", node.Node.Host)
		s.Equal(*expectedVersion, node.RunningVersion(), "node %s not on expected version", node.Node.Host)
	}
}

func (s *TopfSuite) bumpTalosVersion(version string) {
	content, err := os.ReadFile(s.configPath)
	s.Require().NoError(err)

	re := regexp.MustCompile(`(?m)^talosVersion: .*$`)

	if !re.Match(content) {
		s.FailNowf("no talosVersion line found in %s", s.configPath)
	}

	updated := re.ReplaceAll(content, []byte("talosVersion: "+version))
	//nolint:gosec // config path is provided by the test invocation on purpose
	s.Require().NoError(os.WriteFile(s.configPath, updated, 0o600))
}
