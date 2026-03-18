package cgroup

import (
	"os/exec"
	"strings"
	"testing"
)

func TestWrapCommand_Empty(t *testing.T) {
	cmd, err := WrapCommand("claude --resume", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd != "claude --resume" {
		t.Errorf("expected unchanged command, got %q", cmd)
	}
}

func TestWrapCommand_ValidLimits(t *testing.T) {
	if _, err := exec.LookPath("systemd-run"); err != nil {
		t.Skip("systemd-run not available")
	}

	tests := []struct {
		limit    string
		expected string
	}{
		{"16G", "MemoryMax=16G"},
		{"8192M", "MemoryMax=8192M"},
		{"1T", "MemoryMax=1T"},
		{"4096000K", "MemoryMax=4096000K"},
		{"16g", "MemoryMax=16G"}, // lowercase normalized
		{"1024m", "MemoryMax=1024M"},
		{"1073741824", "MemoryMax=1073741824"}, // raw bytes
	}

	for _, tc := range tests {
		cmd, err := WrapCommand("claude --resume", tc.limit)
		if err != nil {
			t.Errorf("WrapCommand(%q): unexpected error: %v", tc.limit, err)
			continue
		}
		if !strings.Contains(cmd, tc.expected) {
			t.Errorf("WrapCommand(%q): expected %q in result, got %q", tc.limit, tc.expected, cmd)
		}
		if !strings.HasPrefix(cmd, "systemd-run --user --scope --collect") {
			t.Errorf("WrapCommand(%q): missing systemd-run prefix, got %q", tc.limit, cmd)
		}
		if !strings.HasSuffix(cmd, "-- claude --resume") {
			t.Errorf("WrapCommand(%q): missing original command suffix, got %q", tc.limit, cmd)
		}
	}
}

func TestWrapCommand_InvalidLimits(t *testing.T) {
	tests := []string{
		"0G",       // zero
		"-16G",     // negative
		"16GB",     // invalid suffix
		"sixteen",  // non-numeric
		"",         // empty (handled separately but testing regex)
		"16 G",     // space
		"16G 8G",   // multiple values
	}

	for _, limit := range tests {
		if limit == "" {
			continue // empty is a no-op, not an error
		}
		_, err := WrapCommand("claude --resume", limit)
		if err == nil {
			t.Errorf("WrapCommand(%q): expected error for invalid limit", limit)
		}
	}
}

func TestValidMemoryLimitRegex(t *testing.T) {
	valid := []string{"1K", "16G", "8192M", "1T", "1073741824", "512m", "2g"}
	for _, v := range valid {
		if !validMemoryLimit.MatchString(v) {
			t.Errorf("expected %q to be valid", v)
		}
	}

	invalid := []string{"0G", "G16", "16GB", "-1G", "abc", "16 G"}
	for _, v := range invalid {
		if validMemoryLimit.MatchString(v) {
			t.Errorf("expected %q to be invalid", v)
		}
	}
}
