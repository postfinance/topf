// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package topf

import (
	"context"
	"sync"
)

// Render generates machine config bundles for all configured nodes.
// When online is true, live nodes are queried for their actual running Talos version,
// which is then used for version contract selection in config generation.
// Errors during config generation for individual nodes are recorded in the Node.Error field.
func (t *topf) Render(ctx context.Context, online bool) ([]*Node, error) {
	cfg := t.Config()

	nodes := make([]*Node, 0, len(cfg.Nodes))

	for _, node := range cfg.Nodes {
		nodes = append(nodes, &Node{Node: &node, t: t})
	}

	if online {
		var wg sync.WaitGroup

		for _, node := range nodes {
			wg.Add(1)

			go func(node *Node) {
				defer wg.Done()

				if err := node.collectNodeInfo(ctx); err != nil {
					node.Error = err
				}
			}(node)
		}

		wg.Wait()
	}

	for _, node := range nodes {
		if node.Error != nil {
			continue
		}

		if err := t.generateNodeConfig(node); err != nil {
			node.Error = err
		}
	}

	return nodes, nil
}
