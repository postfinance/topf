// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"path/filepath"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestNodeAddressAndEffectiveEndpoint(t *testing.T) {
	tests := []struct {
		name         string
		yaml         string
		wantAddress  string
		wantEndpoint string
	}{
		{
			name:         "host only",
			yaml:         "host: node1.example.com\nrole: worker\n",
			wantAddress:  "node1.example.com",
			wantEndpoint: "node1.example.com",
		},
		{
			name:         "ip takes precedence over host",
			yaml:         "host: node1\nip: 10.0.0.1\nrole: worker\n",
			wantAddress:  "10.0.0.1",
			wantEndpoint: "10.0.0.1",
		},
		{
			name:         "ipv6",
			yaml:         "host: node1\nip: fd00::1\nrole: worker\n",
			wantAddress:  "fd00::1",
			wantEndpoint: "fd00::1",
		},
		{
			name:         "endpoint overrides ip for connecting only",
			yaml:         "host: node1\nip: 10.0.0.1\nendpoint: 127.0.0.1:50001\nrole: control-plane\n",
			wantAddress:  "10.0.0.1",
			wantEndpoint: "127.0.0.1:50001",
		},
		{
			name:         "endpoint overrides host for connecting only",
			yaml:         "host: node1.example.com\nendpoint: lb.example.com\nrole: control-plane\n",
			wantAddress:  "node1.example.com",
			wantEndpoint: "lb.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var n Node
			if err := yaml.Unmarshal([]byte(tt.yaml), &n); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			if got := n.Address(); got != tt.wantAddress {
				t.Errorf("Address() = %q, want %q", got, tt.wantAddress)
			}

			if got := n.EffectiveEndpoint(); got != tt.wantEndpoint {
				t.Errorf("EffectiveEndpoint() = %q, want %q", got, tt.wantEndpoint)
			}
		})
	}
}

func TestNodeTemplateAddressAndEndpoint(t *testing.T) {
	var n Node
	if err := yaml.Unmarshal([]byte("host: node1\nip: 10.0.0.1\nrole: worker\n"), &n); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	path := filepath.Join(t.TempDir(), "patch.yaml.tpl")
	tpl := "{{ .Node.Address }}|{{ .Node.EffectiveEndpoint }}|{{ .Node.Endpoint }}"

	if err := os.WriteFile(path, []byte(tpl), 0o600); err != nil {
		t.Fatal(err)
	}

	out, err := RenderTemplate(path, &PatchContext{Node: &n})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if want := "10.0.0.1|10.0.0.1|"; string(out) != want {
		t.Errorf("rendered %q, want %q", out, want)
	}
}
