// Package cgroup provides helpers for wrapping commands in cgroups v2 resource scopes.
package cgroup

import (
	"fmt"
	"io"
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
// memorySwapMax sets MemorySwapMax to prevent swap from bypassing the memory
// ceiling. If empty, defaults to "1G" when memoryMax is set. Pass "0" to
// disable swap entirely within the scope.
//
// Returns the original command unchanged if memoryMax is empty.
// Returns an error only if memoryMax or memorySwapMax is set but invalid.
//
// Graceful degradation: if systemd-run is not available, logs a warning to
// logw and returns the original command unwrapped. The polecat spawns without
// memory limits rather than failing entirely.
func WrapCommand(command, memoryMax, memorySwapMax string, logw io.Writer) (string, error) {
	if memoryMax == "" {
		return command, nil
	}

	if !validMemoryLimit.MatchString(memoryMax) {
		return "", fmt.Errorf("invalid memory_max value %q: must be a positive integer with optional K/M/G/T suffix (e.g. 16G, 8192M)", memoryMax)
	}

	// Normalize suffix to uppercase for systemd consistency.
	memoryMax = strings.ToUpper(memoryMax)

	// Default swap limit to 1G if not explicitly set. This prevents swap from
	// bypassing the MemoryMax ceiling (stress-ng --vm-bytes 10G completes under
	// an 8G MemoryMax without this because excess spills into swap).
	if memorySwapMax == "" {
		memorySwapMax = "1G"
	}

	// "0" is a valid value meaning "no swap allowed" — skip regex validation for it.
	if memorySwapMax != "0" && !validMemoryLimit.MatchString(memorySwapMax) {
		return "", fmt.Errorf("invalid memory_swap_max value %q: must be a positive integer with optional K/M/G/T suffix (e.g. 1G, 512M) or \"0\"", memorySwapMax)
	}
	memorySwapMax = strings.ToUpper(memorySwapMax)

	if _, err := exec.LookPath("systemd-run"); err != nil {
		fmt.Fprintf(logw, "[cgroup] WARNING: systemd-run not found — memory limit %s will NOT be enforced. "+
			"Install systemd or remove memory_max from rig config to silence this warning.\n", memoryMax)
		return command, nil
	}

	fmt.Fprintf(logw, "[cgroup] applying memory limit: MemoryMax=%s MemorySwapMax=%s via systemd-run --user --scope\n", memoryMax, memorySwapMax)

	// --user: run in user session (no root required)
	// --scope: the command inherits stdio (required for tmux pane)
	// --collect: auto-clean the scope unit after it exits
	// -p MemoryMax=<limit>: the cgroup v2 memory ceiling
	// -p MemorySwapMax=<limit>: cap on swap usage to prevent swap bypass
	//
	// The command is passed via /bin/sh -c because polecat startup commands
	// are shell pipelines with export statements and && chains that cannot
	// be executed directly by systemd-run's -- separator.
	//
	// NOTE: If a process inside this scope is OOM-killed, it will appear as
	// a normal crash (exit code 137/SIGKILL). Check `journalctl --user` or
	// `dmesg` for "oom_reaper" / "Killed process" to confirm it hit the
	// cgroup memory ceiling rather than a system-wide OOM.
	escaped := strings.ReplaceAll(command, "'", "'\"'\"'")
	wrapped := fmt.Sprintf("systemd-run --user --scope --collect -p MemoryMax=%s -p MemorySwapMax=%s -- /bin/sh -c '%s'",
		memoryMax, memorySwapMax, escaped)
	fmt.Fprintf(logw, "[cgroup] wrapped command: systemd-run --user --scope --collect -p MemoryMax=%s -p MemorySwapMax=%s -- /bin/sh -c '...'\n",
		memoryMax, memorySwapMax)
	return wrapped, nil
}
