// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package kubeconfig

import (
	stdx509 "crypto/x509"
	"encoding/pem"
	"slices"
	"testing"
	"time"

	"github.com/postfinance/topf/internal/topf"
	"github.com/postfinance/topf/pkg/config"
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

func TestGenerateUserAndGroups(t *testing.T) {
	bundle, err := secrets.NewBundle(secrets.NewFixedClock(time.Now()), talosconfig.TalosVersionCurrent)
	if err != nil {
		t.Fatalf("failed to generate secrets bundle: %v", err)
	}

	f := &fakeTopf{cfg: &config.TopfConfig{ClusterName: "test"}, secrets: bundle}

	tests := map[string]struct {
		groups []string
		want   []string
	}{
		"groups":           {groups: []string{"topf:viewers", "topf:auditors"}, want: []string{"topf:viewers", "topf:auditors"}},
		"empty group name": {groups: []string{""}, want: nil},
		"no groups":        {groups: nil, want: nil},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cfg, err := Generate(f, "alice", tc.groups, time.Hour)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			if cfg.CurrentContext != "alice@test" {
				t.Errorf("current context = %q, want alice@test", cfg.CurrentContext)
			}

			block, _ := pem.Decode(cfg.AuthInfos["alice@test"].ClientCertificateData)
			if block == nil {
				t.Fatal("failed to decode certificate PEM")
			}

			crt, err := stdx509.ParseCertificate(block.Bytes)
			if err != nil {
				t.Fatalf("failed to parse certificate: %v", err)
			}

			if crt.Subject.CommonName != "alice" {
				t.Errorf("certificate user = %q, want alice", crt.Subject.CommonName)
			}

			if !slices.Equal(crt.Subject.Organization, tc.want) {
				t.Errorf("certificate groups = %q, want %q", crt.Subject.Organization, tc.want)
			}
		})
	}
}

func TestGenerateEmptyUser(t *testing.T) {
	if _, err := Generate(&fakeTopf{}, "", []string{DefaultGroup}, time.Hour); err == nil {
		t.Error("Generate() expected error, got nil")
	}
}
