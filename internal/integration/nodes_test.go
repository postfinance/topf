// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

//go:build integration

package integration

import (
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
)

// NodesSuite asserts the initial state: unconfigured nodes in maintenance mode.
type NodesSuite struct {
	TopfSuite
}

func (s *NodesSuite) TestNodesInMaintenance() {
	ctx := s.T().Context()

	for _, node := range s.Nodes(ctx) {
		s.NoError(node.Error, "node %s has an error", node.Node.Host)
		s.Equal(runtime.MachineStageMaintenance, node.MachineStatus.Stage,
			"node %s is not in maintenance mode; was the cluster already configured?", node.Node.Host)
	}
}
