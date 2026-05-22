// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package vals provides utilities for evaluating vals references in YAML content
package vals

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// refPattern matches vals reference expressions like ref+vault://..., ref+sops://..., etc.
var refPattern = regexp.MustCompile(`^ref\+[a-zA-Z0-9]+://`)

// EvalContent evaluates vals references in YAML content by piping it through
// the vals binary. It returns the evaluated content and a list of plaintext
// values that were resolved from vals references (for output redaction).
// If the vals binary is not found, the content is returned unchanged.
func EvalContent(content []byte) ([]byte, []string, error) {
	// Check if content contains any vals references before spawning a process
	if !hasValsRefs(content) {
		return content, nil, nil
	}

	// #nosec G204 required as long as we don't inline vals evaluation
	cmd := exec.Command("vals", "eval", "-f", "-")

	cmd.Stdin = bytes.NewReader(content)

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, nil, fmt.Errorf("vals evaluation failed: %s", strings.TrimSpace(stderr.String()))
	}

	evaluated := stdout.Bytes()
	secrets := extractValsSecrets(content, evaluated)

	return evaluated, secrets, nil
}

// hasValsRefs checks whether the content contains any vals reference patterns.
func hasValsRefs(content []byte) bool {
	dec := yaml.NewDecoder(bytes.NewReader(content))

	for {
		var doc any

		if err := dec.Decode(&doc); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return false
		}

		if doc != nil && containsValsRef(doc) {
			return true
		}
	}

	return false
}

// containsValsRef recursively checks if a YAML structure contains any vals ref.
func containsValsRef(data any) bool {
	switch v := data.(type) {
	case map[string]any:
		for _, val := range v {
			if containsValsRef(val) {
				return true
			}
		}
	case []any:
		for _, val := range v {
			if containsValsRef(val) {
				return true
			}
		}
	case string:
		return refPattern.MatchString(v)
	}

	return false
}

// extractValsSecrets finds values that were vals-references by comparing
// the pre-evaluated and post-evaluated YAML. It walks each document in the
// pre-evaluated YAML to find leaf values matching the ref+ pattern, records
// their paths, then resolves those paths in the post-evaluated YAML to
// return the plaintext values.
func extractValsSecrets(preEval, postEval []byte) []string {
	preDec := yaml.NewDecoder(bytes.NewReader(preEval))
	postDec := yaml.NewDecoder(bytes.NewReader(postEval))

	var secrets []string

	for {
		var preData, postData any

		preErr := preDec.Decode(&preData)
		postErr := postDec.Decode(&postData)

		if errors.Is(preErr, io.EOF) || errors.Is(postErr, io.EOF) {
			break
		}

		if preErr != nil || postErr != nil {
			return secrets
		}

		var paths [][]any
		collectValsPaths(preData, nil, &paths)

		for _, path := range paths {
			if val, ok := resolvePath(postData, path); ok {
				if s, ok := val.(string); ok && s != "" {
					secrets = append(secrets, s)
				}
			}
		}
	}

	return secrets
}

// collectValsPaths recursively walks a YAML structure and records the
// path to every leaf string value that matches the vals ref pattern.
func collectValsPaths(data any, path []any, paths *[][]any) {
	switch v := data.(type) {
	case map[string]any:
		for key, val := range v {
			collectValsPaths(val, append(path, key), paths)
		}
	case []any:
		for i, val := range v {
			collectValsPaths(val, append(path, i), paths)
		}
	case string:
		if refPattern.MatchString(v) {
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
