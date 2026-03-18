// Package cgroup provides helpers for wrapping commands in cgroups v2 resource scopes.
package cgroup

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// validMemoryLimit matches values like "16G", "8192M", "1T", "4096000K".
// Supports K, M, G, T suffixes (case-insensitive) or raw byte counts.
var validMemoryLimit = regexp.MustCompile(`(?i)^[1-9]\d*[KMGT]?$`)

// WrapCommand wraps a shell command string with systemd-run to enforce a
// MemoryMax limit via cgroups v2. The entire process tree spawned by the
// command is confined to a transient user scope. If the limit is exceeded,
// the kernel OOM-kills within the scope only — other system processes are
// unaffected.
//
// Returns the original command unchanged if memoryMax is empty.
// Returns an error if memoryMax is set but invalid, or if systemd-run
// is not available on the system.
func WrapCommand(command, memoryMax string) (string, error) {
	if memoryMax == "" {
		return command, nil
	}

	if !validMemoryLimit.MatchString(memoryMax) {
		return "", fmt.Errorf("invalid memory_max value %q: must be a positive integer with optional K/M/G/T suffix (e.g. 16G, 8192M)", memoryMax)
	}

	// Normalize suffix to uppercase for systemd consistency.
	memoryMax = strings.ToUpper(memoryMax)

	if _, err := exec.LookPath("systemd-run"); err != nil {
		return "", fmt.Errorf("systemd-run not found: cgroup memory limits require systemd (memory_max=%s)", memoryMax)
	}

	// --user: run in user session (no root required)
	// --scope: the command inherits stdio (required for tmux pane)
	// --collect: auto-clean the scope unit after it exits
	// -p MemoryMax=<limit>: the cgroup v2 memory ceiling
	return fmt.Sprintf("systemd-run --user --scope --collect -p MemoryMax=%s -- %s", memoryMax, command), nil
}
