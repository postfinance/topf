// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package progress combines structural slog logging with transient per-node
// progress reporting, backed by talos' console reporter when colorized.
package progress

import (
	"fmt"
	"log/slog"

	"github.com/siderolabs/talos/cmd/talosctl/pkg/talos/safeout"
	"github.com/siderolabs/talos/pkg/reporter"
)

// Progress routes structural events to the embedded logger and transient
// progress to the reporter. Without a colorized reporter, Running forwards
// to debug, Done to info and Fail to error.
type Progress struct {
	*slog.Logger

	reporter *reporter.Reporter
	host     string
}

// New builds a Progress for one node. rep may be nil.
func New(logger *slog.Logger, rep *reporter.Reporter, host string) Progress {
	return Progress{Logger: logger, reporter: rep, host: host}
}

// Running reports transient progress.
func (p Progress) Running(msg string) {
	if p.reporter == nil || !p.reporter.IsColorized() {
		p.Debug("progress", "message", msg)

		return
	}

	p.reporter.Report(reporter.Update{Message: prefix(p.host, msg), Status: reporter.StatusRunning})
}

// Done reports a completed phase.
func (p Progress) Done(msg string) {
	if p.reporter == nil || !p.reporter.IsColorized() {
		p.Info("progress", "message", msg)

		return
	}

	p.reporter.Report(reporter.Update{Message: prefix(p.host, msg), Status: reporter.StatusSucceeded})
}

// Fail reports a failed phase. The error still propagates to the caller.
func (p Progress) Fail(msg string) {
	if p.reporter == nil || !p.reporter.IsColorized() {
		p.Error("progress", "message", msg)

		return
	}

	p.reporter.Report(reporter.Update{Message: prefix(p.host, msg), Status: reporter.StatusError})
}

func prefix(host, msg string) string {
	if host == "" {
		return msg
	}

	return fmt.Sprintf("%s: %s", host, msg)
}

// NewReporter returns nil when plain output was requested or upgrades run in
// parallel, since the reporter renders a single status line.
func NewReporter(mode reporter.OutputMode, concurrency int) *reporter.Reporter {
	rep := reporter.New(
		reporter.WithOutputMode(mode),
		reporter.WithLineFilter(safeout.String),
	)

	if !rep.IsColorized() || concurrency > 1 {
		return nil
	}

	return rep
}
