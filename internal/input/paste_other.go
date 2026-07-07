//go:build !windows

package input

import "github.com/Gaurav-Gosain/tuios/internal/app"

// readSystemClipboard is only implemented on Windows; other platforms use OSC 52.
func readSystemClipboard(o *app.OS) (string, bool) {
	_ = o
	return "", false
}

// writeSystemClipboard is only implemented on Windows; other platforms use OSC 52.
func writeSystemClipboard(text string) bool {
	_ = text
	return false
}
