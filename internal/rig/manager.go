package rig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/steveyegge/gastown/internal/config"
	"github.com/steveyegge/gastown/internal/git"
)

// Common errors
var (
	ErrRigNotFound = errors.New("rig not found")
	ErrRigExists   = errors.New("rig already exists")
)

// Manager handles rig discovery, loading, and creation.
type Manager struct {
	townRoot string
	config   *config.RigsConfig
	git      *git.Git
}

// NewManager creates a new rig manager.
func NewManager(townRoot string, rigsConfig *config.RigsConfig, g *git.Git) *Manager {
	return &Manager{
		townRoot: townRoot,
		config:   rigsConfig,
		git:      g,
	}
}

// DiscoverRigs returns all rigs registered in the workspace.
func (m *Manager) DiscoverRigs() ([]*Rig, error) {
	var rigs []*Rig

	for name, entry := range m.config.Rigs {
		rig, err := m.loadRig(name, entry)
		if err != nil {
			// Log error but continue with other rigs
			continue
		}
		rigs = append(rigs, rig)
	}

	return rigs, nil
}

// GetRig returns a specific rig by name.
func (m *Manager) GetRig(name string) (*Rig, error) {
	entry, ok := m.config.Rigs[name]
	if !ok {
		return nil, ErrRigNotFound
	}

	return m.loadRig(name, entry)
}

// RigExists checks if a rig is registered.
func (m *Manager) RigExists(name string) bool {
	_, ok := m.config.Rigs[name]
	return ok
}

// loadRig loads rig details from the filesystem.
func (m *Manager) loadRig(name string, entry config.RigEntry) (*Rig, error) {
	rigPath := filepath.Join(m.townRoot, name)

	// Verify directory exists
	info, err := os.Stat(rigPath)
	if err != nil {
		return nil, fmt.Errorf("rig directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", rigPath)
	}

	rig := &Rig{
		Name:   name,
		Path:   rigPath,
		GitURL: entry.GitURL,
		Config: entry.BeadsConfig,
	}

	// Scan for polecats
	polecatsDir := filepath.Join(rigPath, "polecats")
	if entries, err := os.ReadDir(polecatsDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				rig.Polecats = append(rig.Polecats, e.Name())
			}
		}
	}

	// Scan for crew workers
	crewDir := filepath.Join(rigPath, "crew")
	if entries, err := os.ReadDir(crewDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				rig.Crew = append(rig.Crew, e.Name())
			}
		}
	}

	// Check for witness
	witnessPath := filepath.Join(rigPath, "witness", "rig")
	if _, err := os.Stat(witnessPath); err == nil {
		rig.HasWitness = true
	}

	// Check for refinery
	refineryPath := filepath.Join(rigPath, "refinery", "rig")
	if _, err := os.Stat(refineryPath); err == nil {
		rig.HasRefinery = true
	}

	// Check for mayor clone
	mayorPath := filepath.Join(rigPath, "mayor", "rig")
	if _, err := os.Stat(mayorPath); err == nil {
		rig.HasMayor = true
	}

	return rig, nil
}

// AddRig clones a repository and registers it as a rig.
func (m *Manager) AddRig(name, gitURL string) (*Rig, error) {
	if m.RigExists(name) {
		return nil, ErrRigExists
	}

	rigPath := filepath.Join(m.townRoot, name)

	// Check if directory already exists
	if _, err := os.Stat(rigPath); err == nil {
		return nil, fmt.Errorf("directory already exists: %s", rigPath)
	}

	// Clone repository
	if err := m.git.Clone(gitURL, rigPath); err != nil {
		return nil, fmt.Errorf("cloning repository: %w", err)
	}

	// Create agent directories
	if err := m.createAgentDirs(rigPath); err != nil {
		// Cleanup on failure
		os.RemoveAll(rigPath)
		return nil, fmt.Errorf("creating agent directories: %w", err)
	}

	// Update git exclude
	if err := m.updateGitExclude(rigPath); err != nil {
		// Non-fatal, continue
	}

	// Register in config
	m.config.Rigs[name] = config.RigEntry{
		GitURL: gitURL,
	}

	return m.loadRig(name, m.config.Rigs[name])
}

// RemoveRig unregisters a rig (does not delete files).
func (m *Manager) RemoveRig(name string) error {
	if !m.RigExists(name) {
		return ErrRigNotFound
	}

	delete(m.config.Rigs, name)
	return nil
}

// createAgentDirs creates the standard agent directory structure.
func (m *Manager) createAgentDirs(rigPath string) error {
	for _, dir := range AgentDirs {
		dirPath := filepath.Join(rigPath, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
	}
	return nil
}

// updateGitExclude adds agent directories to .git/info/exclude.
func (m *Manager) updateGitExclude(rigPath string) error {
	excludePath := filepath.Join(rigPath, ".git", "info", "exclude")

	// Read existing content
	content, err := os.ReadFile(excludePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Append agent dirs
	additions := "\n# Gas Town agent directories\n"
	for _, dir := range AgentDirs {
		additions += dir + "/\n"
	}

	// Write back
	return os.WriteFile(excludePath, append(content, []byte(additions)...), 0644)
}

// ListRigNames returns the names of all registered rigs.
func (m *Manager) ListRigNames() []string {
	names := make([]string, 0, len(m.config.Rigs))
	for name := range m.config.Rigs {
		names = append(names, name)
	}
	return names
}
