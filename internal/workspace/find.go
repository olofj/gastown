// Package workspace provides workspace detection and management.
package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrNotFound indicates no workspace was found.
var ErrNotFound = errors.New("not in a Gas Town workspace")

// Markers used to detect a Gas Town workspace.
const (
	// PrimaryMarker is the main config file that identifies a workspace.
	PrimaryMarker = "config/town.json"

	// SecondaryMarker is an alternative indicator at the town level.
	SecondaryMarker = "mayor"
)

// Find locates the town root by walking up from the given directory.
// It looks for config/town.json (primary) or mayor/ directory (secondary).
// Returns the absolute path to the town root, or empty string if not found.
func Find(startDir string) (string, error) {
	// Resolve to absolute path and follow symlinks
	absDir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	absDir, err = filepath.EvalSymlinks(absDir)
	if err != nil {
		return "", fmt.Errorf("evaluating symlinks: %w", err)
	}

	// Walk up the directory tree
	current := absDir
	for {
		// Check for primary marker (config/town.json)
		primaryPath := filepath.Join(current, PrimaryMarker)
		if _, err := os.Stat(primaryPath); err == nil {
			return current, nil
		}

		// Check for secondary marker (mayor/ directory)
		secondaryPath := filepath.Join(current, SecondaryMarker)
		info, err := os.Stat(secondaryPath)
		if err == nil && info.IsDir() {
			return current, nil
		}

		// Move to parent directory
		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root
			return "", nil
		}
		current = parent
	}
}

// FindOrError is like Find but returns a user-friendly error if not found.
func FindOrError(startDir string) (string, error) {
	root, err := Find(startDir)
	if err != nil {
		return "", err
	}
	if root == "" {
		return "", ErrNotFound
	}
	return root, nil
}

// FindFromCwd locates the town root from the current working directory.
func FindFromCwd() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting current directory: %w", err)
	}
	return Find(cwd)
}

// FindFromCwdOrError is like FindFromCwd but returns an error if not found.
func FindFromCwdOrError() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting current directory: %w", err)
	}
	return FindOrError(cwd)
}

// IsWorkspace checks if the given directory is a Gas Town workspace root.
func IsWorkspace(dir string) (bool, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false, fmt.Errorf("resolving path: %w", err)
	}

	// Check for primary marker
	primaryPath := filepath.Join(absDir, PrimaryMarker)
	if _, err := os.Stat(primaryPath); err == nil {
		return true, nil
	}

	// Check for secondary marker
	secondaryPath := filepath.Join(absDir, SecondaryMarker)
	info, err := os.Stat(secondaryPath)
	if err == nil && info.IsDir() {
		return true, nil
	}

	return false, nil
}
