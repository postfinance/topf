// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package sops provides utilities for reading SOPS-encrypted files
package sops

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v3"
)

// ReadFileWithSOPS reads a file and automatically decrypts it with SOPS if encrypted.
// In addition to the decrypted content, it returns the plaintext values of any fields
// that were SOPS-encrypted (identified by the "ENC[" prefix in the encrypted file).
// Returns nil content if the file doesn't exist (not an error).
func ReadFileWithSOPS(path string) (content []byte, secrets []string, err error) {
	// Check if file exists
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return nil, nil, nil
	} else if err != nil {
		return nil, nil, err
	}

	// Check if file is encrypted with SOPS
	// #nosec G204 required as long as we don't inline sops decryption
	statusCmd := exec.Command("sops", "filestatus", path)
	statusOutput, statusErr := statusCmd.Output()

	var status struct {
		Encrypted bool `json:"encrypted"`
	}

	isEncrypted := statusErr == nil &&
		json.Unmarshal(statusOutput, &status) == nil &&
		status.Encrypted

	if !isEncrypted {
		//nolint:gosec // files read through a variable in our control
		content, err = os.ReadFile(path)
		return content, nil, err
	}

	// Read encrypted content to discover which fields are encrypted
	//nolint:gosec // files read through a variable in our control
	encryptedBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	// Decrypt
	// #nosec G204 required as long as we don't inline sops encryption
	decryptCmd := exec.Command("sops", "decrypt", path)

	content, err = decryptCmd.Output()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return nil, nil, fmt.Errorf("sops decryption failed: %s", string(exitErr.Stderr))
		}

		return nil, nil, fmt.Errorf("failed to run sops decrypt: %w", err)
	}

	secrets = extractEncryptedValues(encryptedBytes, content)

	return content, secrets, nil
}

// extractEncryptedValues finds values that were SOPS-encrypted by comparing
// the encrypted and decrypted YAML. It walks each document in the encrypted YAML
// to find leaf values starting with "ENC[", records their paths, then resolves
// those paths in the corresponding decrypted YAML document to return the plaintext values.
func extractEncryptedValues(encrypted, decrypted []byte) []string {
	encDec := yaml.NewDecoder(bytes.NewReader(encrypted))
	decDec := yaml.NewDecoder(bytes.NewReader(decrypted))

	var secrets []string

	for {
		var encData, decData any

		encErr := encDec.Decode(&encData)
		decErr := decDec.Decode(&decData)

		if errors.Is(encErr, io.EOF) || errors.Is(decErr, io.EOF) {
			break
		}

		if encErr != nil || decErr != nil {
			return secrets
		}

		var paths [][]any
		collectEncryptedPaths(encData, nil, &paths)

		for _, path := range paths {
			if val, ok := resolvePath(decData, path); ok {
				if s, ok := val.(string); ok && s != "" {
					secrets = append(secrets, s)
				}
			}
		}
	}

	return secrets
}

// collectEncryptedPaths recursively walks a YAML structure and records the
// path to every leaf string value that starts with "ENC[".
func collectEncryptedPaths(data any, path []any, paths *[][]any) {
	switch v := data.(type) {
	case map[string]any:
		for key, val := range v {
			collectEncryptedPaths(val, append(path, key), paths)
		}
	case []any:
		for i, val := range v {
			collectEncryptedPaths(val, append(path, i), paths)
		}
	case string:
		if strings.HasPrefix(v, "ENC[") {
			// Copy path to avoid slice aliasing
			p := make([]any, len(path))
			copy(p, path)
			*paths = append(*paths, p)
		}
	}
}

// resolvePath follows a sequence of map keys and slice indices to reach
// a value in a nested YAML structure.
func resolvePath(data any, path []any) (any, bool) {
	current := data

	for _, segment := range path {
		switch key := segment.(type) {
		// string means we look up a key in a map
		case string:
			m, ok := current.(map[string]any)
			if !ok {
				return nil, false
			}

			current, ok = m[key]
			if !ok {
				return nil, false
			}

		// integer means we index into a slice
		case int:
			s, ok := current.([]any)
			if !ok || key >= len(s) {
				return nil, false
			}

			current = s[key]
		default:
			return nil, false
		}
	}

	return current, true
}
