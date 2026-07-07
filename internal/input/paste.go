// Package input implements paste handling for TUIOS terminal forwarding.
package input

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/Gaurav-Gosain/tuios/internal/app"
	"github.com/Gaurav-Gosain/tuios/internal/config"
)

const (
	bracketedPasteStart = "\x1b[200~"
	bracketedPasteEnd   = "\x1b[201~"
)

// normalizePasteContent converts Windows-style line endings to Unix newlines.
func normalizePasteContent(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return content
}

// shouldWrapBracketedPaste reports whether paste content should be wrapped with
// bracketed-paste sequences before forwarding to a child PTY.
func shouldWrapBracketedPaste(content string, appBracketedPasteEnabled bool) bool {
	if appBracketedPasteEnabled {
		return true
	}
	// Multiline paste without bracketed paste causes each line to submit as Enter
	// when TUIOS forwards bytes through an extra ConPTY layer.
	return strings.Contains(content, "\n")
}

// buildPastePayload wraps content for safe delivery to a child terminal.
func buildPastePayload(content string, appBracketedPasteEnabled bool) string {
	content = normalizePasteContent(content)
	if !shouldWrapBracketedPaste(content, appBracketedPasteEnabled) {
		return content
	}
	return bracketedPasteStart + content + bracketedPasteEnd
}

func bracketedPasteEnabledForWindow(o *app.OS) bool {
	focusedWindow := o.GetFocusedWindow()
	if focusedWindow == nil || focusedWindow.Terminal == nil {
		return false
	}
	return focusedWindow.Terminal.BracketedPasteEnabled()
}

// handlePasteStart begins buffering host-terminal bracketed paste in terminal mode.
func handlePasteStart(o *app.OS) {
	if o.Mode != app.TerminalMode {
		return
	}
	o.PasteInProgress = true
	o.PasteBuffer = ""
}

// handlePasteEnd finishes bracketed paste, flushing any leaked key events.
func handlePasteEnd(o *app.OS) {
	if o.Mode != app.TerminalMode {
		return
	}

	leaked := o.PasteInProgress && o.PasteBuffer != ""
	if leaked {
		o.ClipboardContent = o.PasteBuffer
		sendPasteToTerminal(o, o.ClipboardContent)
	}

	o.PasteInProgress = false
	o.PasteBuffer = ""
}

// handlePasteContent forwards complete paste payloads from tea.PasteMsg.
func handlePasteContent(o *app.OS, content string) {
	if o.Mode != app.TerminalMode {
		return
	}
	o.PasteInProgress = false
	o.PasteBuffer = ""
	o.ClipboardContent = content
	sendPasteToTerminal(o, content)
}

// accumulatePasteKey buffers key events that leak during host bracketed paste.
// Returns true when the key was consumed.
func accumulatePasteKey(msg tea.KeyPressMsg, o *app.OS) bool {
	if !o.PasteInProgress || o.Mode != app.TerminalMode {
		return false
	}

	key := msg.String()
	switch key {
	case "enter", "shift+enter":
		o.PasteBuffer += "\n"
	case "tab":
		o.PasteBuffer += "\t"
	case "space":
		o.PasteBuffer += " "
	case "backspace":
		if len(o.PasteBuffer) > 0 {
			o.PasteBuffer = o.PasteBuffer[:len(o.PasteBuffer)-1]
		}
	default:
		if msg.Text != "" {
			o.PasteBuffer += msg.Text
		} else if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
			o.PasteBuffer += key
		}
	}
	return true
}

// sendPasteToTerminal delivers paste content to the focused terminal PTY.
func sendPasteToTerminal(o *app.OS, content string) {
	if content == "" {
		o.ShowNotification("Clipboard is empty", "warning", config.NotificationDuration)
		return
	}

	if o.FocusedWindow < 0 || o.FocusedWindow >= len(o.Windows) {
		return
	}

	focusedWindow := o.GetFocusedWindow()
	if focusedWindow == nil {
		return
	}

	isImagePath := strings.Contains(content, "tuios_clipboard_image_") && strings.HasSuffix(content, ".png")

	pasteContent := buildPastePayload(content, bracketedPasteEnabledForWindow(o))
	if err := focusedWindow.SendInput([]byte(pasteContent)); err != nil {
		o.ShowNotification("Paste failed", "error", config.NotificationDuration)
		return
	}

	if isImagePath {
		o.ShowNotification("Pasted clipboard image path", "success", config.NotificationDuration)
	} else {
		o.ShowNotification(fmt.Sprintf("Pasted %d characters", len(content)), "success", config.NotificationDuration)
	}
}

// copyToSystemClipboard copies text to the OS clipboard.
func copyToSystemClipboard(text string) tea.Cmd {
	if writeSystemClipboard(text) {
		return nil
	}
	return tea.SetClipboard(text)
}

// pasteFromSystemClipboard pastes into the focused terminal, preferring native clipboard APIs.
func pasteFromSystemClipboard(o *app.OS) tea.Cmd {
	if content, ok := readSystemClipboard(); ok {
		handlePasteContent(o, content)
		return nil
	}
	return tea.ReadClipboard
}

// requestClipboardPaste returns a command that reads the system clipboard when
// available, falling back to OSC 52 via tea.ReadClipboard.
func requestClipboardPaste(o *app.OS) tea.Cmd {
	return pasteFromSystemClipboard(o)
}
