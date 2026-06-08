// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package nodepool

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/postfinance/topf/internal/topf"
	"github.com/postfinance/topf/pkg/config"
)

func nodes(n int) []*topf.Node {
	out := make([]*topf.Node, n)
	for i := range out {
		out[i] = &topf.Node{Node: &config.Node{Host: fmt.Sprintf("node-%d", i)}}
	}

	return out
}

func TestRun(t *testing.T) {
	const total, n = 12, 4

	var (
		mu            sync.Mutex
		current, peak int
		processed     atomic.Int64
	)

	fn := func(_ context.Context, _ *topf.Node, _ *slog.Logger) error {
		mu.Lock()
		current++
		peak = max(peak, current)
		mu.Unlock()

		time.Sleep(20 * time.Millisecond)

		mu.Lock()
		current--
		mu.Unlock()
		processed.Add(1)

		return nil
	}

	if err := RunConcurrent(context.Background(), nodes(total), n, fn, slog.New(slog.DiscardHandler)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if processed.Load() != total {
		t.Errorf("processed = %d, want %d", processed.Load(), total)
	}

	if peak != n {
		t.Errorf("peak concurrency = %d, want %d", peak, n)
	}
}

func TestRunStopsAfterFailure(t *testing.T) {
	wantErr := errors.New("boom")

	var processed atomic.Int64

	fn := func(_ context.Context, node *topf.Node, _ *slog.Logger) error {
		processed.Add(1)

		time.Sleep(50 * time.Millisecond)

		t.Log(node.Node.Host)
		// node-0 is dequeued first (FIFO) and fails immediately.
		if node.Node.Host == "node-0" {
			return wantErr
		}

		return nil
	}

	err := RunConcurrent(context.Background(), nodes(50), 2, fn, slog.New(slog.DiscardHandler))
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}

	if processed.Load() >= 4 {
		t.Errorf("processed = %d, want fewer than 4 (pool should stop after failure)", processed.Load())
	}
}
