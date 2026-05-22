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
	"slices"
	"strings"

	"github.com/postfinance/topf/internal/yamlutils"
	"gopkg.in/yaml.v3"
)

// refPattern matches vals reference expressions like ref+vault://..., ref+sops://..., etc.
var refPattern = regexp.MustCompile(`^ref\+[a-zA-Z0-9]+://`)

// EvalContent evaluates vals references in YAML content by piping it through
// the vals binary. It returns the evaluated content and a list of plaintext
// values that were resolved from vals references (for output redaction).
// Returns an error if the content contains vals references but the vals
// binary is not installed.
func EvalContent(content []byte) ([]byte, []string, error) {
	if !hasValsRefs(content) {
		return content, nil, nil
	}

	valsPath, err := exec.LookPath("vals")
	if err != nil {
		return nil, nil, fmt.Errorf("content contains vals references but vals binary not found in PATH: %w", err)
	}

	// #nosec G204 required as long as we don't inline vals evaluation
	cmd := exec.Command(valsPath, "eval", "-f", "-")

	cmd.Stdin = bytes.NewReader(content)

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, nil, fmt.Errorf("vals evaluation failed: %s: %w", strings.TrimSpace(stderr.String()), err)
	}

	evaluated := stdout.Bytes()
	secrets := yamlutils.ExtractSecrets(content, evaluated, refPattern.MatchString)

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
		if slices.ContainsFunc(v, containsValsRef) {
			return true
		}
	case string:
		return refPattern.MatchString(v)
	}

	return false
}
