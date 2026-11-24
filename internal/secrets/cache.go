// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package secrets

import (
	"fmt"
	"sync"

	"golang.org/x/sync/singleflight"
)

type cache struct {
	mu      sync.RWMutex
	entries map[string][]byte
	group   singleflight.Group
}

func newCache() *cache {
	return &cache{entries: make(map[string][]byte)}
}

// get returns the cached value for key, or calls load exactly once even with
// concurrent callers. Errors are not cached.
func (c *cache) get(key string, load func() ([]byte, error)) ([]byte, error) {
	if v, ok := c.lookup(key); ok {
		return v, nil
	}

	res, err, _ := c.group.Do(key, func() (any, error) {
		// another caller may have finished between lookup and Do
		if v, ok := c.lookup(key); ok {
			return v, nil
		}

		v, err := load()
		if err != nil {
			return nil, err
		}

		c.mu.Lock()
		c.entries[key] = v
		c.mu.Unlock()

		return v, nil
	})
	if err != nil {
		return nil, err
	}

	v, ok := res.([]byte)
	if !ok {
		return nil, fmt.Errorf("unexpected cache value type %T", res)
	}

	return v, nil
}

func (c *cache) lookup(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	v, ok := c.entries[key]

	return v, ok
}
