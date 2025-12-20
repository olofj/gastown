package cmd

import (
	"testing"
)

func TestAddIntegrationBranchField(t *testing.T) {
	tests := []struct {
		name        string
		description string
		branchName  string
		want        string
	}{
		{
			name:        "empty description",
			description: "",
			branchName:  "integration/gt-epic",
			want:        "integration_branch: integration/gt-epic",
		},
		{
			name:        "simple description",
			description: "Epic for authentication",
			branchName:  "integration/gt-auth",
			want:        "integration_branch: integration/gt-auth\nEpic for authentication",
		},
		{
			name:        "existing integration_branch field",
			description: "integration_branch: integration/old-epic\nSome description",
			branchName:  "integration/new-epic",
			want:        "integration_branch: integration/new-epic\nSome description",
		},
		{
			name:        "multiline description",
			description: "Line 1\nLine 2\nLine 3",
			branchName:  "integration/gt-xyz",
			want:        "integration_branch: integration/gt-xyz\nLine 1\nLine 2\nLine 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := addIntegrationBranchField(tt.description, tt.branchName)
			if got != tt.want {
				t.Errorf("addIntegrationBranchField() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseBranchName(t *testing.T) {
	tests := []struct {
		name       string
		branch     string
		wantIssue  string
		wantWorker string
	}{
		{
			name:       "polecat branch format",
			branch:     "polecat/Nux/gt-xyz",
			wantIssue:  "gt-xyz",
			wantWorker: "Nux",
		},
		{
			name:       "polecat branch with subtask",
			branch:     "polecat/Worker/gt-abc.1",
			wantIssue:  "gt-abc.1",
			wantWorker: "Worker",
		},
		{
			name:       "simple issue branch",
			branch:     "gt-xyz",
			wantIssue:  "gt-xyz",
			wantWorker: "",
		},
		{
			name:       "feature branch with issue",
			branch:     "feature/gt-abc-impl",
			wantIssue:  "gt-abc",
			wantWorker: "",
		},
		{
			name:       "no issue pattern",
			branch:     "main",
			wantIssue:  "",
			wantWorker: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := parseBranchName(tt.branch)
			if info.Issue != tt.wantIssue {
				t.Errorf("parseBranchName() Issue = %q, want %q", info.Issue, tt.wantIssue)
			}
			if info.Worker != tt.wantWorker {
				t.Errorf("parseBranchName() Worker = %q, want %q", info.Worker, tt.wantWorker)
			}
		})
	}
}

func TestFormatMRAge(t *testing.T) {
	tests := []struct {
		name      string
		createdAt string
		wantOk    bool // just check it doesn't panic/error
	}{
		{
			name:      "RFC3339 format",
			createdAt: "2025-01-01T12:00:00Z",
			wantOk:    true,
		},
		{
			name:      "alternative format",
			createdAt: "2025-01-01T12:00:00",
			wantOk:    true,
		},
		{
			name:      "invalid format",
			createdAt: "not-a-date",
			wantOk:    true, // returns "?" for invalid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMRAge(tt.createdAt)
			if tt.wantOk && result == "" {
				t.Errorf("formatMRAge() returned empty for %s", tt.createdAt)
			}
		})
	}
}

func TestGetDescriptionWithoutMRFields(t *testing.T) {
	tests := []struct {
		name        string
		description string
		want        string
	}{
		{
			name:        "empty description",
			description: "",
			want:        "",
		},
		{
			name:        "only MR fields",
			description: "branch: polecat/Nux/gt-xyz\ntarget: main\nworker: Nux",
			want:        "",
		},
		{
			name:        "mixed content",
			description: "branch: polecat/Nux/gt-xyz\nSome custom notes\ntarget: main",
			want:        "Some custom notes",
		},
		{
			name:        "no MR fields",
			description: "Just a regular description\nWith multiple lines",
			want:        "Just a regular description\nWith multiple lines",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getDescriptionWithoutMRFields(tt.description)
			if got != tt.want {
				t.Errorf("getDescriptionWithoutMRFields() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{
			name:   "short string",
			s:      "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exact length",
			s:      "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "needs truncation",
			s:      "hello world",
			maxLen: 8,
			want:   "hello...",
		},
		{
			name:   "very short max",
			s:      "hello",
			maxLen: 3,
			want:   "hel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateString() = %q, want %q", got, tt.want)
			}
		})
	}
}
