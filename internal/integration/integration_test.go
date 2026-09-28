// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

//go:build integration

package integration

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// TestIntegration runs all suites sequentially: each phase depends on the
// state left behind by the previous one.
func TestIntegration(t *testing.T) {
	if *expectedVersion == "" {
		t.Fatal("--expected.talos.version is required")
	}

	suites := []suite.TestingSuite{
		&NodesSuite{TopfSuite: TopfSuite{configPath: *configPath}},
		&BootstrapSuite{TopfSuite: TopfSuite{configPath: *configPath}},
		&ConfigSuite{TopfSuite: TopfSuite{configPath: *configPath}},
		&UpgradeSuite{TopfSuite: TopfSuite{configPath: *configPath}},
	}

	for _, s := range suites {
		suite.Run(t, s)
	}
}
