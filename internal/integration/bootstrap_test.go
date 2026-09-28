// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

//go:build integration

package integration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/postfinance/topf/internal/cmd/apply"
	"github.com/postfinance/topf/internal/topf"
	"github.com/siderolabs/go-retry/retry"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
)

// BootstrapSuite applies the cluster configuration to the maintenance nodes
// and bootstraps etcd.
type BootstrapSuite struct {
	TopfSuite
}

const waitForReadyTimeout = 5 * time.Minute

// testify runs methods alphabetically, so the dry-run assertion and the apply
// share one method to guarantee their order.
func (s *BootstrapSuite) TestBootstrap() {
	ctx := s.T().Context()

	err := apply.Execute(ctx, s.Runtime(), apply.Options{
		DryRun: true,
		Mode:   machine.ApplyConfigurationRequest_AUTO,
	})
	s.ErrorIs(err, topf.ErrDryRunChangesDetected, "dry-run against unconfigured nodes must report changes")

	err = apply.Execute(ctx, s.Runtime(), apply.Options{
		AutoBootstrap:         true,
		AllowNotReady:         true,
		StabilizationDuration: 5 * time.Second,
		Mode:                  machine.ApplyConfigurationRequest_AUTO,
	})
	s.Require().NoError(err, "apply with auto-bootstrap failed")

	s.WaitForReady(ctx)
}

func (s *TopfSuite) WaitForReady(ctx context.Context) {
	var lastState string

	err := retry.Constant(waitForReadyTimeout, retry.WithUnits(10*time.Second)).Retry(func() error {
		nodes, err := s.Runtime().FilteredNodes(ctx)
		if err != nil {
			return retry.ExpectedError(err)
		}

		lastState = describeNodes(nodes)

		for _, node := range nodes {
			if node.Error != nil {
				return retry.ExpectedErrorf("node %s has error: %v", node.Node.Host, node.Error)
			}

			if !node.MachineStatus.Status.Ready {
				return retry.ExpectedErrorf("node %s not ready: %s", node.Node.Host, describeUnmet(node))
			}

			if node.MachineStatus.Stage != runtime.MachineStageRunning {
				return retry.ExpectedErrorf("node %s in stage %s", node.Node.Host, node.MachineStatus.Stage)
			}
		}

		return nil
	})
	if err != nil {
		s.FailNowf("nodes did not become ready within %s\nlast state:\n%s",
			waitForReadyTimeout.String(), lastState)
	}
}

func describeNodes(nodes []*topf.Node) string {
	var sb strings.Builder

	for _, node := range nodes {
		fmt.Fprintf(&sb, "- %s: stage=%s ready=%t version=%s error=%v\n",
			node.Node.Host, node.MachineStatus.Stage, node.MachineStatus.Status.Ready, node.RunningVersion(), node.Error)
	}

	return sb.String()
}

func describeUnmet(node *topf.Node) string {
	if len(node.MachineStatus.Status.UnmetConditions) == 0 {
		return "no unmet conditions reported"
	}

	conditions := make([]string, 0, len(node.MachineStatus.Status.UnmetConditions))
	for _, cond := range node.MachineStatus.Status.UnmetConditions {
		conditions = append(conditions, cond.Name+": "+cond.Reason)
	}

	return strings.Join(conditions, "; ")
}
