// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package tmpl

import (
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/postfinance/topf/pkg/config"
)

func testContext(t *testing.T) *Context {
	t.Helper()

	endpoint, err := url.Parse("https://cp.example.com:6443")
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.TopfConfig{
		ClusterName:     "test-cluster",
		ClusterEndpoint: config.Endpoint(*endpoint),
		SchematicID:     "@schematic.yaml",
		Data:            map[string]any{"region": "eu-west-1"},
		Nodes: []config.Node{
			{Host: "cp1", Role: config.RoleControlPlane},
			{
				Host:    "w1",
				Role:    config.RoleWorker,
				Factory: "factory.example.com",
				Data:    map[string]any{"disk": "/dev/vda"},
			},
		},
	}

	return &Context{
		TopfConfig: cfg,
		Node: Node{
			Node:                cfg.Nodes[1],
			ResolvedSchematicID: "abc123",
			RuntimeData:         RuntimeData{TalosVersion: "1.14.2"},
		},
	}
}

func TestRender(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		want    string
		wantErr bool
	}{
		{name: "cluster field", src: "{{ .ClusterName }}", want: "test-cluster"},
		{name: "cluster endpoint prints as URL", src: "{{ .ClusterEndpoint }}", want: "https://cp.example.com:6443"},
		{name: "cluster schematic is the raw config value", src: "{{ .SchematicID }}", want: "@schematic.yaml"},
		{name: "sprig pipeline", src: "{{ .Data.region | upper | trunc 2 }}", want: "EU"},
		{name: "node config field", src: "{{ .Node.Factory }}", want: "factory.example.com"},
		{name: "node data", src: "{{ .Node.Data.disk }}", want: "/dev/vda"},
		{name: "node role comparison", src: `{{ if eq .Node.Role "worker" }}w{{ end }}`, want: "w"},
		{name: "resolved schematic", src: "{{ .Node.ResolvedSchematicID }}", want: "abc123"},
		{name: "runtime data semver", src: `{{ semverCompare ">= 1.14.0-0" .Node.RuntimeData.TalosVersion }}`, want: "true"},
		{name: "all nodes", src: "{{ range .Nodes }}{{ .Host }},{{ end }}", want: "cp1,w1,"},
		{name: "control-plane nodes", src: `{{ range .Nodes }}{{ if eq .Role "control-plane" }}{{ .Host }}{{ end }}{{ end }}`, want: "cp1"},
		{name: "missing data key", src: "{{ .Data.nope }}", wantErr: true},
		{name: "unknown field", src: "{{ .PatchesDirr }}", wantErr: true},
		{name: "unknown node field", src: "{{ .Node.Nope }}", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Render("test.yaml.tpl", []byte(tt.src), testContext(t))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got output %q", got)
				}

				return
			}

			if err != nil {
				t.Fatalf("Render() error: %v", err)
			}

			if string(got) != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderEnv(t *testing.T) {
	t.Setenv("TOPF_TEST_VAR", "hello-world")

	got, err := Render("env.yaml.tpl", []byte(`{{ env "TOPF_TEST_VAR" }}`), testContext(t))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if string(got) != "hello-world" {
		t.Errorf("got %q, want %q", got, "hello-world")
	}
}

func TestRenderEncryptedTemplate(t *testing.T) {
	src := []byte(`{"data": "ENC[AES256_GCM,data:abc,iv:def,tag:ghi,type:str]", "sops": {}}`)

	_, err := Render("secret.yaml.tpl", src, testContext(t))
	if !errors.Is(err, ErrEncryptedTemplate) {
		t.Fatalf("expected ErrEncryptedTemplate, got %v", err)
	}
}

func TestRenderParseErrorContainsName(t *testing.T) {
	_, err := Render("all/01-broken.yaml.tpl", []byte("{{ if }}"), testContext(t))
	if err == nil || !strings.Contains(err.Error(), "all/01-broken.yaml.tpl") {
		t.Fatalf("expected parse error mentioning the template name, got %v", err)
	}
}
