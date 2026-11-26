// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package talosconfig

import (
	stdx509 "crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"testing"
	"time"

	"github.com/postfinance/topf/internal/topf"
	"github.com/postfinance/topf/pkg/config"
	clientconfig "github.com/siderolabs/talos/pkg/machinery/client/config"
	talosconfig "github.com/siderolabs/talos/pkg/machinery/config"
	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
)

type fakeTopf struct {
	topf.Topf
	cfg     *config.TopfConfig
	secrets *secrets.Bundle
}

func (f *fakeTopf) Config() *config.TopfConfig { return f.cfg }

func (f *fakeTopf) Secrets() (*secrets.Bundle, error) { return f.secrets, nil }

func newFakeTopf(t *testing.T) *fakeTopf {
	t.Helper()

	bundle, err := secrets.NewBundle(secrets.NewFixedClock(time.Now()), talosconfig.TalosVersionCurrent)
	if err != nil {
		t.Fatalf("failed to generate secrets bundle: %v", err)
	}

	return &fakeTopf{
		cfg:     &config.TopfConfig{ClusterName: "test"},
		secrets: bundle,
	}
}

func parseClientCert(t *testing.T, raw []byte) *stdx509.Certificate {
	t.Helper()

	cfg, err := clientconfig.FromBytes(raw)
	if err != nil {
		t.Fatalf("failed to parse talosconfig: %v", err)
	}

	crtPEM, err := base64.StdEncoding.DecodeString(cfg.Contexts["test"].Crt)
	if err != nil {
		t.Fatalf("failed to decode certificate: %v", err)
	}

	block, _ := pem.Decode(crtPEM)
	if block == nil {
		t.Fatal("failed to decode certificate PEM")
	}

	crt, err := stdx509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	return crt
}

func TestGenerateRolesAndValidity(t *testing.T) {
	raw, err := Generate(newFakeTopf(t), []string{"os:reader", "os:etcd:backup"}, 2*time.Hour)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	crt := parseClientCert(t, raw)

	if got := crt.NotAfter.Sub(crt.NotBefore); got != 2*time.Hour {
		t.Errorf("certificate validity = %s, want 2h", got)
	}

	want := map[string]bool{"os:reader": true, "os:etcd:backup": true}
	if len(crt.Subject.Organization) != len(want) {
		t.Fatalf("certificate roles = %v, want os:reader and os:etcd:backup", crt.Subject.Organization)
	}

	for _, org := range crt.Subject.Organization {
		if !want[org] {
			t.Errorf("unexpected role %q in certificate", org)
		}
	}
}

func TestGenerateInvalidInput(t *testing.T) {
	tests := map[string]struct {
		roles    []string
		validity time.Duration
	}{
		"unknown role":      {roles: []string{"os:reeder"}, validity: time.Hour},
		"no roles":          {roles: []string{}, validity: time.Hour},
		"zero validity":     {roles: []string{"os:admin"}, validity: 0},
		"negative validity": {roles: []string{"os:admin"}, validity: -time.Hour},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := Generate(&fakeTopf{}, tc.roles, tc.validity); err == nil {
				t.Error("Generate() expected error, got nil")
			}
		})
	}
}
