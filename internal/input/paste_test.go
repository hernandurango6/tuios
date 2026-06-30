package input

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Gaurav-Gosain/tuios/internal/app"
)

func TestShouldWrapBracketedPaste(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		appEnabled bool
		want       bool
	}{
		{name: "app enabled single line", content: "hello", appEnabled: true, want: true},
		{name: "app disabled single line", content: "hello", appEnabled: false, want: false},
		{name: "app disabled multiline", content: "line1\nline2", appEnabled: false, want: true},
		{name: "app enabled multiline", content: "line1\nline2", appEnabled: true, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldWrapBracketedPaste(tt.content, tt.appEnabled); got != tt.want {
				t.Fatalf("shouldWrapBracketedPaste() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildPastePayload(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		appEnabled bool
		want       string
	}{
		{
			name:       "single line without wrap",
			content:    "hello",
			appEnabled: false,
			want:       "hello",
		},
		{
			name:       "multiline wrapped",
			content:    "a\nb",
			appEnabled: false,
			want:       bracketedPasteStart + "a\nb" + bracketedPasteEnd,
		},
		{
			name:       "windows line endings normalized",
			content:    "a\r\nb",
			appEnabled: false,
			want:       bracketedPasteStart + "a\nb" + bracketedPasteEnd,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildPastePayload(tt.content, tt.appEnabled); got != tt.want {
				t.Fatalf("buildPastePayload() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAccumulatePasteKey(t *testing.T) {
	o := &app.OS{
		Mode:            app.TerminalMode,
		PasteInProgress: true,
	}

	if !accumulatePasteKey(tea.KeyPressMsg{Code: 'a', Text: "a"}, o) {
		t.Fatal("expected printable key to be consumed")
	}
	if !accumulatePasteKey(tea.KeyPressMsg{Code: tea.KeyEnter, Text: ""}, o) {
		t.Fatal("expected enter to be consumed")
	}
	if o.PasteBuffer != "a\n" {
		t.Fatalf("PasteBuffer = %q, want %q", o.PasteBuffer, "a\n")
	}
}

func TestHandlePasteStartEnd(t *testing.T) {
	o := &app.OS{Mode: app.TerminalMode}

	handlePasteStart(o)
	if !o.PasteInProgress {
		t.Fatal("expected paste in progress after start")
	}

	o.PasteBuffer = "leaked"
	handlePasteEnd(o)
	if o.PasteInProgress {
		t.Fatal("expected paste to finish after end")
	}
	if o.PasteBuffer != "" {
		t.Fatal("expected paste buffer to be cleared")
	}
}
