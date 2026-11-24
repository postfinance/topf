// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package topf

import (
	"strings"
	"testing"

	"github.com/postfinance/topf/internal/tmpl"
	"github.com/postfinance/topf/pkg/config"
	"github.com/siderolabs/talos/pkg/machinery/version"
)

func TestNodeTalosVersion(t *testing.T) {
	tests := []struct {
		name    string
		cluster string
		node    string
		running string
		want    string
	}{
		{name: "bundled default", want: strings.TrimPrefix(version.Tag, "v")},
		{name: "cluster", cluster: "v1.12.0", want: "1.12.0"},
		{name: "node overrides cluster", cluster: "1.12.0", node: "1.13.0", want: "1.13.0"},
		{name: "running overrides node", cluster: "1.12.0", node: "1.13.0", running: "1.11.0", want: "1.11.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Node{
				t:           &topf{TopfConfig: &config.TopfConfig{TalosVersion: tt.cluster}},
				Node:        &config.Node{TalosVersion: tt.node},
				runtimeData: tmpl.RuntimeData{TalosVersion: tt.running},
			}

			if got := n.TalosVersion(); got != tt.want {
				t.Errorf("TalosVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}
