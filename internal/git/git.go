// Package git provides a wrapper for git operations via subprocess.
package git

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Common errors
var (
	ErrNotARepo       = errors.New("not a git repository")
	ErrMergeConflict  = errors.New("merge conflict")
	ErrAuthFailure    = errors.New("authentication failed")
	ErrRebaseConflict = errors.New("rebase conflict")
)

// Git wraps git operations for a working directory.
type Git struct {
	workDir string
}

// NewGit creates a new Git wrapper for the given directory.
func NewGit(workDir string) *Git {
	return &Git{workDir: workDir}
}

// run executes a git command and returns stdout.
func (g *Git) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = g.workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", g.wrapError(err, stderr.String(), args)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// wrapError wraps git errors with context.
func (g *Git) wrapError(err error, stderr string, args []string) error {
	stderr = strings.TrimSpace(stderr)

	// Detect specific error types
	if strings.Contains(stderr, "not a git repository") {
		return ErrNotARepo
	}
	if strings.Contains(stderr, "CONFLICT") || strings.Contains(stderr, "Merge conflict") {
		return ErrMergeConflict
	}
	if strings.Contains(stderr, "Authentication failed") || strings.Contains(stderr, "could not read Username") {
		return ErrAuthFailure
	}
	if strings.Contains(stderr, "needs merge") || strings.Contains(stderr, "rebase in progress") {
		return ErrRebaseConflict
	}

	if stderr != "" {
		return fmt.Errorf("git %s: %s", args[0], stderr)
	}
	return fmt.Errorf("git %s: %w", args[0], err)
}

// Clone clones a repository to the destination.
func (g *Git) Clone(url, dest string) error {
	cmd := exec.Command("git", "clone", url, dest)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return g.wrapError(err, stderr.String(), []string{"clone", url})
	}
	return nil
}

// Checkout checks out the given ref.
func (g *Git) Checkout(ref string) error {
	_, err := g.run("checkout", ref)
	return err
}

// Fetch fetches from the remote.
func (g *Git) Fetch(remote string) error {
	_, err := g.run("fetch", remote)
	return err
}

// Pull pulls from the remote branch.
func (g *Git) Pull(remote, branch string) error {
	_, err := g.run("pull", remote, branch)
	return err
}

// Push pushes to the remote branch.
func (g *Git) Push(remote, branch string, force bool) error {
	args := []string{"push", remote, branch}
	if force {
		args = append(args, "--force")
	}
	_, err := g.run(args...)
	return err
}

// Add stages files for commit.
func (g *Git) Add(paths ...string) error {
	args := append([]string{"add"}, paths...)
	_, err := g.run(args...)
	return err
}

// Commit creates a commit with the given message.
func (g *Git) Commit(message string) error {
	_, err := g.run("commit", "-m", message)
	return err
}

// CommitAll stages all changes and commits.
func (g *Git) CommitAll(message string) error {
	_, err := g.run("commit", "-am", message)
	return err
}

// GitStatus represents the status of the working directory.
type GitStatus struct {
	Clean    bool
	Modified []string
	Added    []string
	Deleted  []string
	Untracked []string
}

// Status returns the current git status.
func (g *Git) Status() (*GitStatus, error) {
	out, err := g.run("status", "--porcelain")
	if err != nil {
		return nil, err
	}

	status := &GitStatus{Clean: true}
	if out == "" {
		return status, nil
	}

	status.Clean = false
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 3 {
			continue
		}
		code := line[:2]
		file := line[3:]

		switch {
		case strings.Contains(code, "M"):
			status.Modified = append(status.Modified, file)
		case strings.Contains(code, "A"):
			status.Added = append(status.Added, file)
		case strings.Contains(code, "D"):
			status.Deleted = append(status.Deleted, file)
		case strings.Contains(code, "?"):
			status.Untracked = append(status.Untracked, file)
		}
	}

	return status, nil
}

// CurrentBranch returns the current branch name.
func (g *Git) CurrentBranch() (string, error) {
	return g.run("rev-parse", "--abbrev-ref", "HEAD")
}

// HasUncommittedChanges returns true if there are uncommitted changes.
func (g *Git) HasUncommittedChanges() (bool, error) {
	status, err := g.Status()
	if err != nil {
		return false, err
	}
	return !status.Clean, nil
}

// RemoteURL returns the URL for the given remote.
func (g *Git) RemoteURL(remote string) (string, error) {
	return g.run("remote", "get-url", remote)
}

// Merge merges the given branch into the current branch.
func (g *Git) Merge(branch string) error {
	_, err := g.run("merge", branch)
	return err
}

// Rebase rebases the current branch onto the given ref.
func (g *Git) Rebase(onto string) error {
	_, err := g.run("rebase", onto)
	return err
}

// AbortMerge aborts a merge in progress.
func (g *Git) AbortMerge() error {
	_, err := g.run("merge", "--abort")
	return err
}

// AbortRebase aborts a rebase in progress.
func (g *Git) AbortRebase() error {
	_, err := g.run("rebase", "--abort")
	return err
}

// CreateBranch creates a new branch.
func (g *Git) CreateBranch(name string) error {
	_, err := g.run("branch", name)
	return err
}

// DeleteBranch deletes a branch.
func (g *Git) DeleteBranch(name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := g.run("branch", flag, name)
	return err
}

// Rev returns the commit hash for the given ref.
func (g *Git) Rev(ref string) (string, error) {
	return g.run("rev-parse", ref)
}

// IsAncestor checks if ancestor is an ancestor of descendant.
func (g *Git) IsAncestor(ancestor, descendant string) (bool, error) {
	_, err := g.run("merge-base", "--is-ancestor", ancestor, descendant)
	if err != nil {
		// Exit code 1 means not an ancestor, not an error
		if strings.Contains(err.Error(), "exit status 1") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
