// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package secrets

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	t.Run("concurrent callers load once", func(t *testing.T) {
		c := newCache()

		var (
			calls atomic.Int32
			wg    sync.WaitGroup
		)

		load := func() ([]byte, error) {
			calls.Add(1)
			time.Sleep(10 * time.Millisecond) // let callers pile up in singleflight

			return []byte("value"), nil
		}

		for range 10 {
			wg.Go(func() {
				if v, err := c.get("key", load); err != nil || string(v) != "value" {
					t.Errorf("get() = %q, %v", v, err)
				}
			})
		}

		wg.Wait()

		if n := calls.Load(); n != 1 {
			t.Errorf("load called %d times, want 1", n)
		}
	})

	t.Run("errors are not cached", func(t *testing.T) {
		c := newCache()
		errLoad := errors.New("boom")

		if _, err := c.get("key", func() ([]byte, error) { return nil, errLoad }); !errors.Is(err, errLoad) {
			t.Fatalf("expected load error, got %v", err)
		}

		v, err := c.get("key", func() ([]byte, error) { return []byte("value"), nil })
		if err != nil || string(v) != "value" {
			t.Fatalf("get() = %q, %v, want value", v, err)
		}
	})
}
