// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package yamlutils provides shared YAML utilities for walking nested structures
// and extracting values by path. It is used by the sops and vals packages to
// discover secret values for output redaction.
package yamlutils

import (
	"bytes"
	"errors"
	"io"

	"gopkg.in/yaml.v3"
)

// ResolvePath follows a sequence of map keys (string) and slice indices (int)
// to reach a value in a nested YAML structure.
func ResolvePath(data any, path []any) (any, bool) {
	current := data

	for _, segment := range path {
		switch key := segment.(type) {
		case string:
			m, ok := current.(map[string]any)
			if !ok {
				return nil, false
			}

			current, ok = m[key]
			if !ok {
				return nil, false
			}

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

// CollectPaths recursively walks a YAML structure and records the path to every
// leaf string value for which the predicate returns true. Paths are copied to
// avoid slice aliasing.
func CollectPaths(data any, path []any, paths *[][]any, predicate func(string) bool) {
	switch v := data.(type) {
	case map[string]any:
		for key, val := range v {
			CollectPaths(val, append(path, key), paths, predicate)
		}
	case []any:
		for i, val := range v {
			CollectPaths(val, append(path, i), paths, predicate)
		}
	case string:
		if predicate(v) {
			p := make([]any, len(path))
			copy(p, path)
			*paths = append(*paths, p)
		}
	}
}

// ExtractSecrets walks paired YAML documents (before and after transformation)
// and returns the plaintext values at every path where the "before" document
// has a leaf string matching the predicate. This is used to collect secret
// values for output redaction after SOPS decryption or vals evaluation.
func ExtractSecrets(before, after []byte, predicate func(string) bool) []string {
	beforeDec := yaml.NewDecoder(bytes.NewReader(before))
	afterDec := yaml.NewDecoder(bytes.NewReader(after))

	var secrets []string

	for {
		var beforeData, afterData any

		beforeErr := beforeDec.Decode(&beforeData)
		afterErr := afterDec.Decode(&afterData)

		if errors.Is(beforeErr, io.EOF) || errors.Is(afterErr, io.EOF) {
			break
		}

		if beforeErr != nil || afterErr != nil {
			return secrets
		}

		var paths [][]any
		CollectPaths(beforeData, nil, &paths, predicate)

		for _, path := range paths {
			if val, ok := ResolvePath(afterData, path); ok {
				if s, ok := val.(string); ok && s != "" {
					secrets = append(secrets, s)
				}
			}
		}
	}

	return secrets
}
