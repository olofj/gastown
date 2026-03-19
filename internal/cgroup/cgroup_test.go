package cgroup

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func TestWrapCommand_Empty(t *testing.T) {
	var buf bytes.Buffer
	cmd, err := WrapCommand("claude --resume", "", "", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd != "claude --resume" {
		t.Errorf("expected unchanged command, got %q", cmd)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no log output for empty limit, got %q", buf.String())
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
		var buf bytes.Buffer
		cmd, err := WrapCommand("claude --resume", tc.limit, "", &buf)
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
		if !strings.Contains(cmd, "-- /bin/sh -c") {
			t.Errorf("WrapCommand(%q): missing /bin/sh -c wrapper, got %q", tc.limit, cmd)
		}
		if !strings.Contains(cmd, "claude --resume") {
			t.Errorf("WrapCommand(%q): missing original command in shell wrapper, got %q", tc.limit, cmd)
		}
		// Verify default MemorySwapMax=1G is applied
		if !strings.Contains(cmd, "MemorySwapMax=1G") {
			t.Errorf("WrapCommand(%q): expected default MemorySwapMax=1G, got %q", tc.limit, cmd)
		}
		if !strings.Contains(buf.String(), "[cgroup] applying memory limit") {
			t.Errorf("WrapCommand(%q): expected log about applying memory limit, got %q", tc.limit, buf.String())
		}
	}
}

func TestWrapCommand_ExplicitSwapMax(t *testing.T) {
	if _, err := exec.LookPath("systemd-run"); err != nil {
		t.Skip("systemd-run not available")
	}

	tests := []struct {
		name        string
		memMax      string
		swapMax     string
		expectSwap  string
	}{
		{"explicit 2G swap", "16G", "2G", "MemorySwapMax=2G"},
		{"explicit 512M swap", "8G", "512M", "MemorySwapMax=512M"},
		{"zero swap (no swap)", "8G", "0", "MemorySwapMax=0"},
		{"default 1G swap", "8G", "", "MemorySwapMax=1G"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			cmd, err := WrapCommand("claude --resume", tc.memMax, tc.swapMax, &buf)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(cmd, tc.expectSwap) {
				t.Errorf("expected %q in result, got %q", tc.expectSwap, cmd)
			}
		})
	}
}

func TestWrapCommand_InvalidLimits(t *testing.T) {
	tests := []string{
		"0G",       // zero
		"-16G",     // negative
		"16GB",     // invalid suffix
		"sixteen",  // non-numeric
		"16 G",     // space
		"16G 8G",   // multiple values
	}

	for _, limit := range tests {
		var buf bytes.Buffer
		_, err := WrapCommand("claude --resume", limit, "", &buf)
		if err == nil {
			t.Errorf("WrapCommand(%q): expected error for invalid limit", limit)
		}
	}
}

func TestWrapCommand_InvalidSwapMax(t *testing.T) {
	tests := []string{
		"-1G",     // negative
		"16GB",    // invalid suffix
		"sixteen", // non-numeric
	}

	for _, swapMax := range tests {
		var buf bytes.Buffer
		_, err := WrapCommand("claude --resume", "16G", swapMax, &buf)
		if err == nil {
			t.Errorf("WrapCommand(swapMax=%q): expected error for invalid swap limit", swapMax)
		}
	}
}

func TestWrapCommand_SystemdRunMissing(t *testing.T) {
	origPath := t.TempDir()
	t.Setenv("PATH", origPath)

	var buf bytes.Buffer
	cmd, err := WrapCommand("claude --resume", "16G", "", &buf)
	if err != nil {
		t.Fatalf("expected graceful degradation (no error), got: %v", err)
	}
	if cmd != "claude --resume" {
		t.Errorf("expected original command returned unwrapped, got %q", cmd)
	}
	logOutput := buf.String()
	if !strings.Contains(logOutput, "WARNING") {
		t.Errorf("expected WARNING in log output, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "systemd-run not found") {
		t.Errorf("expected 'systemd-run not found' in log output, got %q", logOutput)
	}
}

func TestWrapCommand_ShellPipeline(t *testing.T) {
	if _, err := exec.LookPath("systemd-run"); err != nil {
		t.Skip("systemd-run not available")
	}

	shellCmd := "export FOO=bar && exec env BAZ=qux /usr/bin/claude --resume"
	var buf bytes.Buffer
	cmd, err := WrapCommand(shellCmd, "16G", "", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(cmd, "-- /bin/sh -c") {
		t.Errorf("expected /bin/sh -c wrapper for shell pipeline, got %q", cmd)
	}
	if !strings.Contains(cmd, "export FOO=bar") {
		t.Errorf("expected original shell command preserved inside wrapper, got %q", cmd)
	}
}

func TestWrapCommand_SingleQuoteEscaping(t *testing.T) {
	if _, err := exec.LookPath("systemd-run"); err != nil {
		t.Skip("systemd-run not available")
	}

	cmdWithQuote := "echo 'hello world'"
	var buf bytes.Buffer
	cmd, err := WrapCommand(cmdWithQuote, "8G", "", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(cmd, "echo 'hello world'") {
		t.Errorf("single quotes should be escaped in the wrapper, got %q", cmd)
	}
	if !strings.Contains(cmd, "hello world") {
		t.Errorf("command content should be preserved, got %q", cmd)
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
