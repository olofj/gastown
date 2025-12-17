// Package beads provides a wrapper for the bd (beads) CLI.
package beads

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Common errors
var (
	ErrNotInstalled = errors.New("bd not installed: run 'pip install beads-cli' or see https://github.com/anthropics/beads")
	ErrNotARepo     = errors.New("not a beads repository (no .beads directory found)")
	ErrSyncConflict = errors.New("beads sync conflict")
	ErrNotFound     = errors.New("issue not found")
)

// Issue represents a beads issue.
type Issue struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Priority    int      `json:"priority"`
	Type        string   `json:"issue_type"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	ClosedAt    string   `json:"closed_at,omitempty"`
	Parent      string   `json:"parent,omitempty"`
	Assignee    string   `json:"assignee,omitempty"`
	Children    []string `json:"children,omitempty"`
	DependsOn   []string `json:"depends_on,omitempty"`
	Blocks      []string `json:"blocks,omitempty"`
	BlockedBy   []string `json:"blocked_by,omitempty"`

	// Counts from list output
	DependencyCount int `json:"dependency_count,omitempty"`
	DependentCount  int `json:"dependent_count,omitempty"`
	BlockedByCount  int `json:"blocked_by_count,omitempty"`

	// Detailed dependency info from show output
	Dependencies []IssueDep `json:"dependencies,omitempty"`
	Dependents   []IssueDep `json:"dependents,omitempty"`
}

// IssueDep represents a dependency or dependent issue with its relation.
type IssueDep struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	Priority       int    `json:"priority"`
	Type           string `json:"issue_type"`
	DependencyType string `json:"dependency_type,omitempty"`
}

// ListOptions specifies filters for listing issues.
type ListOptions struct {
	Status   string // "open", "closed", "all"
	Type     string // "task", "bug", "feature", "epic"
	Priority int    // 0-4, -1 for no filter
	Parent   string // filter by parent ID
}

// CreateOptions specifies options for creating an issue.
type CreateOptions struct {
	Title       string
	Type        string // "task", "bug", "feature", "epic"
	Priority    int    // 0-4
	Description string
	Parent      string
}

// UpdateOptions specifies options for updating an issue.
type UpdateOptions struct {
	Title       *string
	Status      *string
	Priority    *int
	Description *string
	Assignee    *string
}

// SyncStatus represents the sync status of the beads repository.
type SyncStatus struct {
	Branch    string
	Ahead     int
	Behind    int
	Conflicts []string
}

// Beads wraps bd CLI operations for a working directory.
type Beads struct {
	workDir string
}

// New creates a new Beads wrapper for the given directory.
func New(workDir string) *Beads {
	return &Beads{workDir: workDir}
}

// run executes a bd command and returns stdout.
func (b *Beads) run(args ...string) ([]byte, error) {
	cmd := exec.Command("bd", args...)
	cmd.Dir = b.workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, b.wrapError(err, stderr.String(), args)
	}

	return stdout.Bytes(), nil
}

// wrapError wraps bd errors with context.
func (b *Beads) wrapError(err error, stderr string, args []string) error {
	stderr = strings.TrimSpace(stderr)

	// Check for bd not installed
	if execErr, ok := err.(*exec.Error); ok && errors.Is(execErr.Err, exec.ErrNotFound) {
		return ErrNotInstalled
	}

	// Detect specific error types from stderr
	if strings.Contains(stderr, "not a beads repository") ||
		strings.Contains(stderr, "No .beads directory") ||
		strings.Contains(stderr, ".beads") && strings.Contains(stderr, "not found") {
		return ErrNotARepo
	}
	if strings.Contains(stderr, "sync conflict") || strings.Contains(stderr, "CONFLICT") {
		return ErrSyncConflict
	}
	if strings.Contains(stderr, "not found") || strings.Contains(stderr, "Issue not found") {
		return ErrNotFound
	}

	if stderr != "" {
		return fmt.Errorf("bd %s: %s", strings.Join(args, " "), stderr)
	}
	return fmt.Errorf("bd %s: %w", strings.Join(args, " "), err)
}

// List returns issues matching the given options.
func (b *Beads) List(opts ListOptions) ([]*Issue, error) {
	args := []string{"list", "--json"}

	if opts.Status != "" {
		args = append(args, "--status="+opts.Status)
	}
	if opts.Type != "" {
		args = append(args, "--type="+opts.Type)
	}
	if opts.Priority >= 0 {
		args = append(args, fmt.Sprintf("--priority=%d", opts.Priority))
	}
	if opts.Parent != "" {
		args = append(args, "--parent="+opts.Parent)
	}

	out, err := b.run(args...)
	if err != nil {
		return nil, err
	}

	var issues []*Issue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, fmt.Errorf("parsing bd list output: %w", err)
	}

	return issues, nil
}

// Ready returns issues that are ready to work (not blocked).
func (b *Beads) Ready() ([]*Issue, error) {
	out, err := b.run("ready", "--json")
	if err != nil {
		return nil, err
	}

	var issues []*Issue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, fmt.Errorf("parsing bd ready output: %w", err)
	}

	return issues, nil
}

// Show returns detailed information about an issue.
func (b *Beads) Show(id string) (*Issue, error) {
	out, err := b.run("show", id, "--json")
	if err != nil {
		return nil, err
	}

	// bd show --json returns an array with one element
	var issues []*Issue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, fmt.Errorf("parsing bd show output: %w", err)
	}

	if len(issues) == 0 {
		return nil, ErrNotFound
	}

	return issues[0], nil
}

// Blocked returns issues that are blocked by dependencies.
func (b *Beads) Blocked() ([]*Issue, error) {
	out, err := b.run("blocked", "--json")
	if err != nil {
		return nil, err
	}

	var issues []*Issue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, fmt.Errorf("parsing bd blocked output: %w", err)
	}

	return issues, nil
}

// Create creates a new issue and returns it.
func (b *Beads) Create(opts CreateOptions) (*Issue, error) {
	args := []string{"create", "--json"}

	if opts.Title != "" {
		args = append(args, "--title="+opts.Title)
	}
	if opts.Type != "" {
		args = append(args, "--type="+opts.Type)
	}
	if opts.Priority >= 0 {
		args = append(args, fmt.Sprintf("--priority=%d", opts.Priority))
	}
	if opts.Description != "" {
		args = append(args, "--description="+opts.Description)
	}
	if opts.Parent != "" {
		args = append(args, "--parent="+opts.Parent)
	}

	out, err := b.run(args...)
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := json.Unmarshal(out, &issue); err != nil {
		return nil, fmt.Errorf("parsing bd create output: %w", err)
	}

	return &issue, nil
}

// Update updates an existing issue.
func (b *Beads) Update(id string, opts UpdateOptions) error {
	args := []string{"update", id}

	if opts.Title != nil {
		args = append(args, "--title="+*opts.Title)
	}
	if opts.Status != nil {
		args = append(args, "--status="+*opts.Status)
	}
	if opts.Priority != nil {
		args = append(args, fmt.Sprintf("--priority=%d", *opts.Priority))
	}
	if opts.Description != nil {
		args = append(args, "--description="+*opts.Description)
	}
	if opts.Assignee != nil {
		args = append(args, "--assignee="+*opts.Assignee)
	}

	_, err := b.run(args...)
	return err
}

// Close closes one or more issues.
func (b *Beads) Close(ids ...string) error {
	if len(ids) == 0 {
		return nil
	}

	args := append([]string{"close"}, ids...)
	_, err := b.run(args...)
	return err
}

// CloseWithReason closes one or more issues with a reason.
func (b *Beads) CloseWithReason(reason string, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}

	args := append([]string{"close"}, ids...)
	args = append(args, "--reason="+reason)
	_, err := b.run(args...)
	return err
}

// AddDependency adds a dependency: issue depends on dependsOn.
func (b *Beads) AddDependency(issue, dependsOn string) error {
	_, err := b.run("dep", "add", issue, dependsOn)
	return err
}

// RemoveDependency removes a dependency.
func (b *Beads) RemoveDependency(issue, dependsOn string) error {
	_, err := b.run("dep", "remove", issue, dependsOn)
	return err
}

// Sync syncs beads with remote.
func (b *Beads) Sync() error {
	_, err := b.run("sync")
	return err
}

// SyncFromMain syncs beads updates from main branch.
func (b *Beads) SyncFromMain() error {
	_, err := b.run("sync", "--from-main")
	return err
}

// SyncStatus returns the sync status without performing a sync.
func (b *Beads) SyncStatus() (*SyncStatus, error) {
	out, err := b.run("sync", "--status", "--json")
	if err != nil {
		// If sync branch doesn't exist, return empty status
		if strings.Contains(err.Error(), "does not exist") {
			return &SyncStatus{}, nil
		}
		return nil, err
	}

	var status SyncStatus
	if err := json.Unmarshal(out, &status); err != nil {
		return nil, fmt.Errorf("parsing bd sync status output: %w", err)
	}

	return &status, nil
}

// Stats returns repository statistics.
func (b *Beads) Stats() (string, error) {
	out, err := b.run("stats")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// IsBeadsRepo checks if the working directory is a beads repository.
func (b *Beads) IsBeadsRepo() bool {
	_, err := b.run("list", "--limit=1")
	return err == nil || !errors.Is(err, ErrNotARepo)
}
