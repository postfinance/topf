// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package interactive contains code for CLI interactions
package interactive

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/mattn/go-isatty"
)

var promptMu sync.Mutex //nolint:gochecknoglobals // guards stdin during interactive prompts

var nonTTYWarned sync.Once //nolint:gochecknoglobals // warns once per process

// ConfirmPrompt asks the user a y/n question.
func ConfirmPrompt(prompt string) rune {
	promptMu.Lock()
	defer promptMu.Unlock()

	if !isatty.IsTerminal(os.Stdin.Fd()) {
		nonTTYWarned.Do(func() {
			fmt.Fprintln(os.Stderr, "\nWARNING: stdin is not a terminal, declining confirmations automatically — pass --confirm=false to skip prompts")
		})

		return 'n'
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/n]: ", prompt)

		input, err := reader.ReadString('\n')
		input = strings.ToLower(strings.TrimSpace(input))

		switch input {
		case "y", "yes":
			return 'y'
		case "n", "no":
			return 'n'
		default:
			// no further read can succeed, so decline instead of reprompting forever
			if err != nil {
				fmt.Println()

				return 'n'
			}

			fmt.Println("Please enter y or n")
		}
	}
}
