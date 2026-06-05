// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package nodepool provides shared helpers for running per-node operations
// across a set of cluster nodes, including bounded-concurrency execution and
// role-based partitioning.
package nodepool

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/postfinance/topf/internal/topf"
	"github.com/postfinance/topf/pkg/config"
)

// BatchSize describes how many nodes may be processed concurrently. It is either
// an absolute count or a percentage of the total node count.
type BatchSize struct {
	Value   int
	Percent bool
}

// ParseBatchSize parses a batch-size flag value, which is either a positive
// integer (e.g. "5") or a percentage of the total node count (e.g. "25%").
func ParseBatchSize(value string) (BatchSize, error) {
	value = strings.TrimSpace(value)

	if pct, ok := strings.CutSuffix(value, "%"); ok {
		n, err := strconv.Atoi(strings.TrimSpace(pct))
		if err != nil || n <= 0 || n > 100 {
			return BatchSize{}, fmt.Errorf("invalid batch-size %q: percentage must be an integer between 1 and 100", value)
		}

		return BatchSize{Value: n, Percent: true}, nil
	}

	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return BatchSize{}, fmt.Errorf("invalid batch-size %q: must be a positive integer or a percentage (e.g. \"5\" or \"25%%\")", value)
	}

	return BatchSize{Value: n}, nil
}

// Resolve returns the effective concurrency for the given total node count. For
// percentages the result is rounded up, and the result is always at least 1 and
// never exceeds the total.
func (b BatchSize) Resolve(total int) int {
	n := b.Value

	if b.Percent {
		n = int(math.Ceil(float64(total) * float64(b.Value) / 100.0))
	}

	if n < 1 {
		n = 1
	}

	if n > total && total > 0 {
		n = total
	}

	return n
}

// PartitionByRole splits nodes into control-plane and worker groups, preserving
// the input order within each group.
func PartitionByRole(nodes []*topf.Node) (controlPlane, workers []*topf.Node) {
	for _, node := range nodes {
		if node.Node.Role == config.RoleControlPlane {
			controlPlane = append(controlPlane, node)
		} else {
			workers = append(workers, node)
		}
	}

	return controlPlane, workers
}

// NodeFunc is a per-node operation. The provided logger already carries the
// node's attributes.
type NodeFunc func(ctx context.Context, node *topf.Node, logger *slog.Logger) error

// RunRolling runs fn over nodes using a rolling pool of at most n concurrent
// operations: as soon as one operation finishes the next node is started. If any
// operation fails, no new operations are started, the in-flight operations are
// allowed to complete, and the joined errors are returned. The provided context
// is not cancelled on failure, so in-flight operations run to completion.
func RunRolling(ctx context.Context, nodes []*topf.Node, n int, fn NodeFunc, logger *slog.Logger) error {
	if len(nodes) == 0 {
		return nil
	}

	if n < 1 {
		n = 1
	}

	var (
		wg   sync.WaitGroup
		stop atomic.Bool
	)

	sem := make(chan struct{}, n)
	errs := make(chan error, len(nodes))

	for _, node := range nodes {
		if stop.Load() {
			break
		}

		sem <- struct{}{}

		// A previously launched operation may have failed while we waited for a
		// free slot; if so, don't start any new operations.
		if stop.Load() {
			<-sem
			break
		}

		wg.Add(1)

		go func(node *topf.Node) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := fn(ctx, node, logger.With(node.Attrs())); err != nil {
				errs <- err

				stop.Store(true)
			}
		}(node)
	}

	wg.Wait()
	close(errs)

	var runErrs []error
	for err := range errs {
		runErrs = append(runErrs, err)
	}

	return errors.Join(runErrs...)
}
