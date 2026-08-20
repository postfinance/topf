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

	"github.com/postfinance/topf/internal/sops"
	"github.com/postfinance/topf/internal/vals"
)

// ReadFileWithSecrets reads a file, automatically decrypts it with SOPS and vals.
// It returns the final content and list of plaintext secret values discovered.
func ReadFileWithSecrets(path string) ([]byte, []string, error) {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil, fmt.Errorf("read file %q: %w", path, fs.ErrNotExist)
		}

		return nil, nil, fmt.Errorf("read file %q: %w", path, err)
	}

	var (
		content     []byte
		sopsSecrets []string
	)

	// first pass with sops
	isEncrypted, err := sops.IsEncrypted(path)
	if err != nil {
		return nil, nil, err
	}

	if isEncrypted {
		content, sopsSecrets, err = sops.Decrypt(path)
		if err != nil {
			return nil, nil, err
		}
	} else {
		//nolint:gosec // files read through a variable in our control
		content, err = os.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
	}

	// second pass with vals
	content, valsSecrets, err := vals.EvalContent(content)
	if err != nil {
		return nil, nil, err
	}

	allSecrets := make([]string, 0, len(sopsSecrets)+len(valsSecrets))
	allSecrets = append(allSecrets, sopsSecrets...)
	allSecrets = append(allSecrets, valsSecrets...)

	return content, allSecrets, nil
}
