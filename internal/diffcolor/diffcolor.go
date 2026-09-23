// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package diffcolor colorizes unified diff text with ANSI escape codes.
// Color codes wrap whole lines only, so embedded secrets remain contiguous
// and can still be redacted downstream.
package diffcolor

import (
	"strings"
)

const (
	reset = "\x1b[0m"
	red   = "\x1b[31m"
	green = "\x1b[32m"
	cyan  = "\x1b[36m"
	bold  = "\x1b[1m"
)

// Colorize returns diff with additions in green, deletions in red,
// hunk headers in cyan and file headers in bold.
func Colorize(diff string) string {
	lines := strings.Split(diff, "\n")

	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			lines[i] = bold + line + reset
		case strings.HasPrefix(line, "@@"):
			lines[i] = cyan + line + reset
		case strings.HasPrefix(line, "+"):
			lines[i] = green + line + reset
		case strings.HasPrefix(line, "-"):
			lines[i] = red + line + reset
		}
	}

	return strings.Join(lines, "\n")
}
