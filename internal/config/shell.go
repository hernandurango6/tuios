package config

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// DetectShell returns the shell executable to spawn in new terminal windows.
func DetectShell() string {
	if cfg, err := LoadUserConfig(); err == nil && cfg.Appearance.PreferredShell != "" {
		preferredShell := cfg.Appearance.PreferredShell
		if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(preferredShell), ".exe") {
			preferredShell += ".exe"
		}

		var shellExists bool
		if runtime.GOOS == "windows" {
			_, err = exec.LookPath(preferredShell)
			shellExists = err == nil
		} else {
			_, err = os.Stat(preferredShell)
			shellExists = err == nil
		}

		if shellExists {
			return preferredShell
		}
		fmt.Fprintf(os.Stderr, "Warning: Configured shell '%s' not found. Falling back to defaults.\n", preferredShell)
	}

	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}

	return defaultShellForPlatform()
}

func defaultShellForPlatform() string {
	if runtime.GOOS == "windows" {
		for _, shell := range windowsShellCandidates() {
			if _, err := exec.LookPath(shell); err == nil {
				return shell
			}
		}
		return "cmd.exe"
	}

	for _, shell := range []string{"/bin/bash", "/bin/zsh", "/bin/fish", "/bin/sh"} {
		if _, err := os.Stat(shell); err == nil {
			return shell
		}
	}
	return "/bin/sh"
}

func windowsShellCandidates() []string {
	return []string{
		"pwsh.exe",
		"powershell.exe",
		"cmd.exe",
	}
}