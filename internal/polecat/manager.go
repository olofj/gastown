package polecat

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/steveyegge/gastown/internal/git"
	"github.com/steveyegge/gastown/internal/rig"
)

// Common errors
var (
	ErrPolecatExists   = errors.New("polecat already exists")
	ErrPolecatNotFound = errors.New("polecat not found")
	ErrHasChanges      = errors.New("polecat has uncommitted changes")
)

// Manager handles polecat lifecycle.
type Manager struct {
	rig *rig.Rig
	git *git.Git
}

// NewManager creates a new polecat manager.
func NewManager(r *rig.Rig, g *git.Git) *Manager {
	return &Manager{
		rig: r,
		git: g,
	}
}

// polecatDir returns the directory for a polecat.
func (m *Manager) polecatDir(name string) string {
	return filepath.Join(m.rig.Path, "polecats", name)
}

// stateFile returns the state file path for a polecat.
func (m *Manager) stateFile(name string) string {
	return filepath.Join(m.polecatDir(name), "state.json")
}

// exists checks if a polecat exists.
func (m *Manager) exists(name string) bool {
	_, err := os.Stat(m.polecatDir(name))
	return err == nil
}

// Add creates a new polecat with a clone of the rig.
func (m *Manager) Add(name string) (*Polecat, error) {
	if m.exists(name) {
		return nil, ErrPolecatExists
	}

	polecatPath := m.polecatDir(name)

	// Create polecats directory if needed
	polecatsDir := filepath.Join(m.rig.Path, "polecats")
	if err := os.MkdirAll(polecatsDir, 0755); err != nil {
		return nil, fmt.Errorf("creating polecats dir: %w", err)
	}

	// Clone the rig repo
	if err := m.git.Clone(m.rig.GitURL, polecatPath); err != nil {
		return nil, fmt.Errorf("cloning rig: %w", err)
	}

	// Create working branch
	polecatGit := git.NewGit(polecatPath)
	branchName := fmt.Sprintf("polecat/%s", name)
	if err := polecatGit.CreateBranch(branchName); err != nil {
		os.RemoveAll(polecatPath)
		return nil, fmt.Errorf("creating branch: %w", err)
	}
	if err := polecatGit.Checkout(branchName); err != nil {
		os.RemoveAll(polecatPath)
		return nil, fmt.Errorf("checking out branch: %w", err)
	}

	// Create polecat state
	now := time.Now()
	polecat := &Polecat{
		Name:      name,
		Rig:       m.rig.Name,
		State:     StateIdle,
		ClonePath: polecatPath,
		Branch:    branchName,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Save state
	if err := m.saveState(polecat); err != nil {
		os.RemoveAll(polecatPath)
		return nil, fmt.Errorf("saving state: %w", err)
	}

	return polecat, nil
}

// Remove deletes a polecat.
func (m *Manager) Remove(name string) error {
	if !m.exists(name) {
		return ErrPolecatNotFound
	}

	polecatPath := m.polecatDir(name)
	polecatGit := git.NewGit(polecatPath)

	// Check for uncommitted changes
	hasChanges, err := polecatGit.HasUncommittedChanges()
	if err == nil && hasChanges {
		return ErrHasChanges
	}

	// Remove directory
	if err := os.RemoveAll(polecatPath); err != nil {
		return fmt.Errorf("removing polecat dir: %w", err)
	}

	return nil
}

// List returns all polecats in the rig.
func (m *Manager) List() ([]*Polecat, error) {
	polecatsDir := filepath.Join(m.rig.Path, "polecats")

	entries, err := os.ReadDir(polecatsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading polecats dir: %w", err)
	}

	var polecats []*Polecat
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		polecat, err := m.Get(entry.Name())
		if err != nil {
			continue // Skip invalid polecats
		}
		polecats = append(polecats, polecat)
	}

	return polecats, nil
}

// Get returns a specific polecat by name.
func (m *Manager) Get(name string) (*Polecat, error) {
	if !m.exists(name) {
		return nil, ErrPolecatNotFound
	}

	return m.loadState(name)
}

// SetState updates a polecat's state.
func (m *Manager) SetState(name string, state State) error {
	polecat, err := m.Get(name)
	if err != nil {
		return err
	}

	polecat.State = state
	polecat.UpdatedAt = time.Now()

	return m.saveState(polecat)
}

// AssignIssue assigns an issue to a polecat.
func (m *Manager) AssignIssue(name, issue string) error {
	polecat, err := m.Get(name)
	if err != nil {
		return err
	}

	polecat.Issue = issue
	polecat.State = StateWorking
	polecat.UpdatedAt = time.Now()

	return m.saveState(polecat)
}

// ClearIssue removes the issue assignment from a polecat.
func (m *Manager) ClearIssue(name string) error {
	polecat, err := m.Get(name)
	if err != nil {
		return err
	}

	polecat.Issue = ""
	polecat.State = StateIdle
	polecat.UpdatedAt = time.Now()

	return m.saveState(polecat)
}

// Wake transitions a polecat from idle to active.
func (m *Manager) Wake(name string) error {
	polecat, err := m.Get(name)
	if err != nil {
		return err
	}

	if polecat.State != StateIdle {
		return fmt.Errorf("polecat is not idle (state: %s)", polecat.State)
	}

	return m.SetState(name, StateActive)
}

// Sleep transitions a polecat from active to idle.
func (m *Manager) Sleep(name string) error {
	polecat, err := m.Get(name)
	if err != nil {
		return err
	}

	if polecat.State != StateActive {
		return fmt.Errorf("polecat is not active (state: %s)", polecat.State)
	}

	return m.SetState(name, StateIdle)
}

// saveState persists polecat state to disk.
func (m *Manager) saveState(polecat *Polecat) error {
	data, err := json.MarshalIndent(polecat, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	stateFile := m.stateFile(polecat.Name)
	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		return fmt.Errorf("writing state: %w", err)
	}

	return nil
}

// loadState reads polecat state from disk.
func (m *Manager) loadState(name string) (*Polecat, error) {
	stateFile := m.stateFile(name)

	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Return minimal polecat if state file missing
			return &Polecat{
				Name:      name,
				Rig:       m.rig.Name,
				State:     StateIdle,
				ClonePath: m.polecatDir(name),
			}, nil
		}
		return nil, fmt.Errorf("reading state: %w", err)
	}

	var polecat Polecat
	if err := json.Unmarshal(data, &polecat); err != nil {
		return nil, fmt.Errorf("parsing state: %w", err)
	}

	return &polecat, nil
}
