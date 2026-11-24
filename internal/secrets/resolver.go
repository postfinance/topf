// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package secrets provides a Resolver that decrypts and resolves secrets from SOPS and vals,
// caches the results, and reports discovered plaintext secrets back via callback
package secrets

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/postfinance/topf/internal/sops"
	"github.com/postfinance/topf/internal/vals"
)

// Resolver returns plaintext content with SOPS and vals secrets resolved.
// Discovered secret values are reported back via a callback.
// Returned content is shared between callers and must not be modified.
type Resolver interface {
	// ReadFile reads a file, decrypting it with SOPS if encrypted and resolving vals references.
	// ReadFile can be safely called concurrently. Results are cached by resolved absolute file path.
	ReadFile(path string) ([]byte, error)
	// Eval resolves vals references in content. SOPS is not applied.
	// Eval can be safely called concurrently. Results are cached by content.
	Eval(content []byte) ([]byte, error)
}

// ResolverCallback is the callback called for plaintext secrets read during ReadFile or Eval.
// Can be called concurrently.
type ResolverCallback func([]string)

// NewResolver returns a Resolver which can decrypt SOPS and resolve vals references
func NewResolver(callback ResolverCallback) Resolver {
	// no-op fallback
	if callback == nil {
		callback = func([]string) {}
	}

	return &resolver{
		callback:  callback,
		sopsCache: newCache(),
		valsCache: newCache(),
	}
}

type resolver struct {
	callback  ResolverCallback
	sopsCache *cache // path → SOPS-decrypted content
	valsCache *cache // sha256(content) → vals-evaluated content
}

// ReadFile reads a file, decrypting it with SOPS if encrypted and resolving vals references.
func (r *resolver) ReadFile(path string) ([]byte, error) {
	// make sure we cache if the file is accessed through different symlinks or relative paths
	key, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	content, err := r.sopsCache.get(key, func() ([]byte, error) {
		return r.decrypt(path)
	})
	if err != nil {
		return nil, err
	}

	res, err := r.Eval(content)
	if err != nil {
		return nil, fmt.Errorf("resolve vals references in %s: %w", path, err)
	}

	return res, nil
}

// Eval resolves vals references in content
func (r *resolver) Eval(content []byte) ([]byte, error) {
	sum := sha256.Sum256(content)

	return r.valsCache.get(string(sum[:]), func() ([]byte, error) {
		res, found, err := vals.EvalContent(content)
		if err != nil {
			return nil, err
		}

		r.callback(found)

		return res, nil
	})
}

// decrypt reads a file and decrypts it with SOPS if it is encrypted.
// Secrets discovered by SOPS are reported via the callback.
func (r *resolver) decrypt(path string) ([]byte, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	isEncrypted, err := sops.IsEncrypted(path)
	if err != nil {
		return nil, fmt.Errorf("check sops status for %s: %w", path, err)
	}

	if !isEncrypted {
		return os.ReadFile(path) //nolint:gosec // files read through a variable in our control
	}

	content, found, err := sops.Decrypt(path)
	if err != nil {
		return nil, fmt.Errorf("decrypt %s with sops: %w", path, err)
	}

	r.callback(found)

	return content, nil
}
