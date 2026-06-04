// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package decryption provides a unified interface for reading files with
// automatic SOPS decryption and vals reference evaluation.
package decryption

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"

	"golang.org/x/sync/singleflight"

	"github.com/postfinance/topf/internal/sops"
	"github.com/postfinance/topf/internal/vals"
)

// decryptResult holds the result of reading and processing a file.
type decryptResult struct {
	content []byte
	secrets []string
}

// Cache stores the result of ReadFile calls, keyed by file path.
// It is safe for concurrent use by multiple goroutines.
type Cache struct {
	mu    sync.RWMutex
	cache map[string]decryptResult
	sf    singleflight.Group
}

// NewCache returns a new, empty Cache.
func NewCache() *Cache {
	return &Cache{
		cache: make(map[string]decryptResult),
	}
}

// ReadFile reads a file, automatically decrypts it with SOPS if encrypted,
// then evaluates any vals references. It returns the final content and
// a combined list of plaintext secret values discovered during both
// SOPS decryption and vals evaluation (for output redaction).
// Results are cached and deduplicated: concurrent calls for the same path
// will only fork sops/vals once.
//
// The returned content and secrets slices are aliases of the cached data
// and share the same underlying arrays. Callers must not append to, resize,
// or write into the returned slices; do so would corrupt the cache and
// race with other concurrent callers.
//
// Returns an error wrapping fs.ErrNotExist if the file doesn't exist.
func (c *Cache) ReadFile(path string) ([]byte, []string, error) {
	c.mu.RLock()

	res, cached := c.cache[path]

	c.mu.RUnlock()

	if cached {
		return res.content, res.secrets, nil
	}

	raw, err, _ := c.sf.Do(path, func() (any, error) {
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, fmt.Errorf("read file %q: %w", path, fs.ErrNotExist)
			}

			return nil, fmt.Errorf("read file %q: %w", path, err)
		}

		var (
			content     []byte
			sopsSecrets []string
		)

		isEncrypted, err := sops.IsEncrypted(path)
		if err != nil {
			return nil, err
		}

		if isEncrypted {
			content, sopsSecrets, err = sops.Decrypt(path)
			if err != nil {
				return nil, err
			}
		} else {
			//nolint:gosec // files read through a variable in our control
			content, err = os.ReadFile(path)
			if err != nil {
				return nil, err
			}
		}

		valsContent, valsSecrets, err := vals.EvalContent(content)
		if err != nil {
			return nil, err
		}

		allSecrets := make([]string, 0, len(sopsSecrets)+len(valsSecrets))
		allSecrets = append(allSecrets, sopsSecrets...)
		allSecrets = append(allSecrets, valsSecrets...)

		decrypted := decryptResult{content: valsContent, secrets: allSecrets}

		c.mu.Lock()
		c.cache[path] = decrypted
		c.mu.Unlock()

		return decrypted, nil
	})
	if err != nil {
		return nil, nil, err
	}

	out, ok := raw.(decryptResult)
	if !ok {
		return nil, nil, fmt.Errorf("unexpected type %T from singleflight for path %q", raw, path)
	}

	return out.content, out.secrets, nil
}
