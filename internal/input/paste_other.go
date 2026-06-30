//go:build !windows

package input

// readSystemClipboard is only implemented on Windows; other platforms use OSC 52.
func readSystemClipboard() (string, bool) {
	return "", false
}

// writeSystemClipboard is only implemented on Windows; other platforms use OSC 52.
func writeSystemClipboard(text string) bool {
	_ = text
	return false
}
