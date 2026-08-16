// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package interactive contains code for CLI interactions
package interactive

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ConfirmPrompt asks the user a y/n question
func ConfirmPrompt(prompt string) rune {
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
