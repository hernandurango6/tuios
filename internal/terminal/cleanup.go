package terminal

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// ResetTerminal sends escape sequences to reset the terminal to a clean state.
// This should be called when exiting the application to restore the terminal.
func ResetTerminal() {
	fmt.Print(
		"\033c" + // Reset terminal to initial state
			"\033[?1000l" + // Disable normal mouse tracking
			"\033[?1002l" + // Disable button event tracking
			"\033[?1003l" + // Disable all motion tracking
			"\033[?1004l" + // Disable focus tracking
			"\033[?1006l" + // Disable SGR extended mouse mode
			"\033[?25h" + // Show cursor
			"\033[?47l" + // Exit alternate screen buffer
			"\033[0m" + // Reset all text attributes
			"\r\n", // Clean line ending
	)
	// Sync can hang on Windows when stdout is a pipe or other non-console handle.
	if term.IsTerminal(int(os.Stdout.Fd())) {
		_ = os.Stdout.Sync()
	}
}
