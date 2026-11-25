// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package config

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestNodeEmptyClass(t *testing.T) {
	var n Node

	err := yaml.Unmarshal([]byte(`
host: n1
role: worker
classes: [gpu, ""]
`), &n)
	if err == nil || !strings.Contains(err.Error(), "empty class") {
		t.Fatalf("expected empty class error, got %v", err)
	}
}
