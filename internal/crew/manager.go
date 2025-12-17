package crew

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
	ErrWorkerExists   = errors.New("crew worker already exists")
	ErrWorkerNotFound = errors.New("crew worker not found")
	ErrHasChanges     = errors.New("crew worker has uncommitted changes")
)

// Manager handles crew worker lifecycle.
type Manager struct {
	rig *rig.Rig
	git *git.Git
}

// NewManager creates a new crew manager.
func NewManager(r *rig.Rig, g *git.Git) *Manager {
	return &Manager{
		rig: r,
		git: g,
	}
}

// workerDir returns the directory for a crew worker.
func (m *Manager) workerDir(name string) string {
	return filepath.Join(m.rig.Path, "crew", name)
}

// stateFile returns the state file path for a crew worker.
func (m *Manager) stateFile(name string) string {
	return filepath.Join(m.workerDir(name), "state.json")
}

// exists checks if a crew worker exists.
func (m *Manager) exists(name string) bool {
	_, err := os.Stat(m.workerDir(name))
	return err == nil
}

// Add creates a new crew worker with a clone of the rig.
func (m *Manager) Add(name string) (*Worker, error) {
	if m.exists(name) {
		return nil, ErrWorkerExists
	}

	workerPath := m.workerDir(name)

	// Create crew directory if needed
	crewDir := filepath.Join(m.rig.Path, "crew")
	if err := os.MkdirAll(crewDir, 0755); err != nil {
		return nil, fmt.Errorf("creating crew dir: %w", err)
	}

	// Clone the rig repo
	if err := m.git.Clone(m.rig.GitURL, workerPath); err != nil {
		return nil, fmt.Errorf("cloning rig: %w", err)
	}

	// Create crew worker state
	now := time.Now()
	worker := &Worker{
		Name:      name,
		Rig:       m.rig.Name,
		State:     StateActive,
		ClonePath: workerPath,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Save state
	if err := m.saveState(worker); err != nil {
		os.RemoveAll(workerPath)
		return nil, fmt.Errorf("saving state: %w", err)
	}

	return worker, nil
}

// AddWithConfig creates a new crew worker with custom configuration.
func (m *Manager) AddWithConfig(name string, beadsDir string) (*Worker, error) {
	worker, err := m.Add(name)
	if err != nil {
		return nil, err
	}

	// Update with custom config
	if beadsDir != "" {
		worker.BeadsDir = beadsDir
		if err := m.saveState(worker); err != nil {
			return nil, fmt.Errorf("saving config: %w", err)
		}
	}

	return worker, nil
}

// Remove deletes a crew worker.
func (m *Manager) Remove(name string) error {
	if !m.exists(name) {
		return ErrWorkerNotFound
	}

	workerPath := m.workerDir(name)
	workerGit := git.NewGit(workerPath)

	// Check for uncommitted changes
	hasChanges, err := workerGit.HasUncommittedChanges()
	if err == nil && hasChanges {
		return ErrHasChanges
	}

	// Remove directory
	if err := os.RemoveAll(workerPath); err != nil {
		return fmt.Errorf("removing crew worker dir: %w", err)
	}

	return nil
}

// RemoveForce deletes a crew worker even with uncommitted changes.
func (m *Manager) RemoveForce(name string) error {
	if !m.exists(name) {
		return ErrWorkerNotFound
	}

	workerPath := m.workerDir(name)
	if err := os.RemoveAll(workerPath); err != nil {
		return fmt.Errorf("removing crew worker dir: %w", err)
	}

	return nil
}

// List returns all crew workers in the rig.
func (m *Manager) List() ([]*Worker, error) {
	crewDir := filepath.Join(m.rig.Path, "crew")

	entries, err := os.ReadDir(crewDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading crew dir: %w", err)
	}

	var workers []*Worker
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		worker, err := m.Get(entry.Name())
		if err != nil {
			continue // Skip invalid workers
		}
		workers = append(workers, worker)
	}

	return workers, nil
}

// Get returns a specific crew worker by name.
func (m *Manager) Get(name string) (*Worker, error) {
	if !m.exists(name) {
		return nil, ErrWorkerNotFound
	}

	return m.loadState(name)
}

// SetState updates a crew worker's state.
func (m *Manager) SetState(name string, state State) error {
	worker, err := m.Get(name)
	if err != nil {
		return err
	}

	worker.State = state
	worker.UpdatedAt = time.Now()

	return m.saveState(worker)
}

// SetBeadsDir updates the custom beads directory for a crew worker.
func (m *Manager) SetBeadsDir(name, beadsDir string) error {
	worker, err := m.Get(name)
	if err != nil {
		return err
	}

	worker.BeadsDir = beadsDir
	worker.UpdatedAt = time.Now()

	return m.saveState(worker)
}

// saveState persists crew worker state to disk.
func (m *Manager) saveState(worker *Worker) error {
	data, err := json.MarshalIndent(worker, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	stateFile := m.stateFile(worker.Name)
	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		return fmt.Errorf("writing state: %w", err)
	}

	return nil
}

// loadState reads crew worker state from disk.
func (m *Manager) loadState(name string) (*Worker, error) {
	stateFile := m.stateFile(name)

	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Return minimal worker if state file missing
			return &Worker{
				Name:      name,
				Rig:       m.rig.Name,
				State:     StateActive,
				ClonePath: m.workerDir(name),
			}, nil
		}
		return nil, fmt.Errorf("reading state: %w", err)
	}

	var worker Worker
	if err := json.Unmarshal(data, &worker); err != nil {
		return nil, fmt.Errorf("parsing state: %w", err)
	}

	return &worker, nil
}

// Names returns just the names of all crew workers.
func (m *Manager) Names() ([]string, error) {
	crewDir := filepath.Join(m.rig.Path, "crew")

	entries, err := os.ReadDir(crewDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading crew dir: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}

	return names, nil
}
