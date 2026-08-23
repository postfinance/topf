// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package topf

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/postfinance/topf/pkg/config"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/client"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
)

// ControlPlaneNode returns the first reachable control-plane node matching
// the nodes-filter regex, with live node info collected. Nodes in
// maintenance mode are skipped because cluster-level RPCs (e.g. etcd) are
// not available there.
func (t *topf) ControlPlaneNode(ctx context.Context) (*Node, error) {
	for _, node := range t.filterNodes() {
		if node.Node.Role != config.RoleControlPlane {
			continue
		}

		if err := node.collectNodeInfo(ctx); err != nil {
			t.logger.Warn("failed to query control-plane node, trying next", "node", node.Node.Host, "error", err)
			continue
		}

		if node.MachineStatus.Stage == runtime.MachineStageMaintenance {
			t.logger.Warn("control-plane node is in maintenance mode, trying next", "node", node.Node.Host)
			continue
		}

		return node, nil
	}

	return nil, errors.New("no reachable control-plane node available (matching the nodes-filter)")
}

// EtcdSnapshot opens a stream reading a consistent snapshot of the etcd
// database from this node. Closing the returned ReadCloser also closes the
// underlying client connection.
func (n *Node) EtcdSnapshot(ctx context.Context) (io.ReadCloser, error) {
	c, err := n.Client(ctx)
	if err != nil {
		return nil, err
	}

	stream, err := c.EtcdSnapshot(ctx, &machine.EtcdSnapshotRequest{})
	if err != nil {
		c.Close()

		return nil, fmt.Errorf("failed to open etcd snapshot stream: %w", err)
	}

	return &clientStream{ReadCloser: stream, client: c}, nil
}

// clientStream wraps a streaming response so that closing it also closes the
// client connection the stream was opened on.
type clientStream struct {
	io.ReadCloser
	client *client.Client
}

func (s *clientStream) Close() error {
	return errors.Join(s.ReadCloser.Close(), s.client.Close())
}
