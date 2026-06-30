//go:build windows

package config

import "testing"

func TestWindowsShellCandidatesPreferPwsh(t *testing.T) {
	candidates := windowsShellCandidates()
	if len(candidates) == 0 || candidates[0] != "pwsh.exe" {
		t.Fatalf("expected pwsh.exe first, got %v", candidates)
	}
}