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

		input, _ := reader.ReadString('\n')
		input = strings.ToLower(strings.TrimSpace(input))

		switch input {
		case "y", "yes":
			return 'y'
		case "n", "no":
			return 'n'
		default:
			fmt.Println("Please enter y or n")
		}
	}
}
