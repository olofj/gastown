// Package session provides polecat session lifecycle management.
package session

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/steveyegge/gastown/internal/rig"
	"github.com/steveyegge/gastown/internal/tmux"
)

// Common errors
var (
	ErrSessionRunning  = errors.New("session already running")
	ErrSessionNotFound = errors.New("session not found")
	ErrPolecatNotFound = errors.New("polecat not found")
)

// Manager handles polecat session lifecycle.
type Manager struct {
	tmux *tmux.Tmux
	rig  *rig.Rig
}

// NewManager creates a new session manager for a rig.
func NewManager(t *tmux.Tmux, r *rig.Rig) *Manager {
	return &Manager{
		tmux: t,
		rig:  r,
	}
}

// StartOptions configures session startup.
type StartOptions struct {
	// WorkDir overrides the default working directory (polecat clone dir).
	WorkDir string

	// Issue is an optional issue ID to work on.
	Issue string

	// Command overrides the default "claude" command.
	Command string
}

// Info contains information about a running session.
type Info struct {
	// Polecat is the polecat name.
	Polecat string `json:"polecat"`

	// SessionID is the tmux session identifier.
	SessionID string `json:"session_id"`

	// Running indicates if the session is currently active.
	Running bool `json:"running"`

	// RigName is the rig this session belongs to.
	RigName string `json:"rig_name"`
}

// sessionName generates the tmux session name for a polecat.
func (m *Manager) sessionName(polecat string) string {
	return fmt.Sprintf("gt-%s-%s", m.rig.Name, polecat)
}

// polecatDir returns the working directory for a polecat.
func (m *Manager) polecatDir(polecat string) string {
	return filepath.Join(m.rig.Path, "polecats", polecat)
}

// hasPolecat checks if the polecat exists in this rig.
func (m *Manager) hasPolecat(polecat string) bool {
	for _, p := range m.rig.Polecats {
		if p == polecat {
			return true
		}
	}
	return false
}

// Start creates and starts a new session for a polecat.
func (m *Manager) Start(polecat string, opts StartOptions) error {
	if !m.hasPolecat(polecat) {
		return fmt.Errorf("%w: %s", ErrPolecatNotFound, polecat)
	}

	sessionID := m.sessionName(polecat)

	// Check if session already exists
	running, err := m.tmux.HasSession(sessionID)
	if err != nil {
		return fmt.Errorf("checking session: %w", err)
	}
	if running {
		return fmt.Errorf("%w: %s", ErrSessionRunning, sessionID)
	}

	// Determine working directory
	workDir := opts.WorkDir
	if workDir == "" {
		workDir = m.polecatDir(polecat)
	}

	// Create session
	if err := m.tmux.NewSession(sessionID, workDir); err != nil {
		return fmt.Errorf("creating session: %w", err)
	}

	// Set environment
	m.tmux.SetEnvironment(sessionID, "GT_RIG", m.rig.Name)
	m.tmux.SetEnvironment(sessionID, "GT_POLECAT", polecat)

	// Send initial command
	command := opts.Command
	if command == "" {
		command = "claude"
	}
	if err := m.tmux.SendKeys(sessionID, command); err != nil {
		return fmt.Errorf("sending command: %w", err)
	}

	// If issue specified, wait a bit then inject it
	if opts.Issue != "" {
		time.Sleep(500 * time.Millisecond)
		prompt := fmt.Sprintf("Work on issue: %s", opts.Issue)
		if err := m.Inject(polecat, prompt); err != nil {
			// Non-fatal, just log
		}
	}

	return nil
}

// Stop terminates a polecat session.
func (m *Manager) Stop(polecat string) error {
	sessionID := m.sessionName(polecat)

	// Check if session exists
	running, err := m.tmux.HasSession(sessionID)
	if err != nil {
		return fmt.Errorf("checking session: %w", err)
	}
	if !running {
		return ErrSessionNotFound
	}

	// Try graceful shutdown first
	m.tmux.SendKeysRaw(sessionID, "C-c") // Ctrl+C
	time.Sleep(100 * time.Millisecond)

	// Kill the session
	if err := m.tmux.KillSession(sessionID); err != nil {
		return fmt.Errorf("killing session: %w", err)
	}

	return nil
}

// IsRunning checks if a polecat session is active.
func (m *Manager) IsRunning(polecat string) (bool, error) {
	sessionID := m.sessionName(polecat)
	return m.tmux.HasSession(sessionID)
}

// List returns information about all sessions for this rig.
func (m *Manager) List() ([]Info, error) {
	sessions, err := m.tmux.ListSessions()
	if err != nil {
		return nil, err
	}

	prefix := fmt.Sprintf("gt-%s-", m.rig.Name)
	var infos []Info

	for _, sessionID := range sessions {
		if !strings.HasPrefix(sessionID, prefix) {
			continue
		}

		polecat := strings.TrimPrefix(sessionID, prefix)
		infos = append(infos, Info{
			Polecat:   polecat,
			SessionID: sessionID,
			Running:   true,
			RigName:   m.rig.Name,
		})
	}

	return infos, nil
}

// Attach attaches to a polecat session.
func (m *Manager) Attach(polecat string) error {
	sessionID := m.sessionName(polecat)

	running, err := m.tmux.HasSession(sessionID)
	if err != nil {
		return fmt.Errorf("checking session: %w", err)
	}
	if !running {
		return ErrSessionNotFound
	}

	return m.tmux.AttachSession(sessionID)
}

// Capture returns the recent output from a polecat session.
func (m *Manager) Capture(polecat string, lines int) (string, error) {
	sessionID := m.sessionName(polecat)

	running, err := m.tmux.HasSession(sessionID)
	if err != nil {
		return "", fmt.Errorf("checking session: %w", err)
	}
	if !running {
		return "", ErrSessionNotFound
	}

	return m.tmux.CapturePane(sessionID, lines)
}

// Inject sends a message to a polecat session.
func (m *Manager) Inject(polecat, message string) error {
	sessionID := m.sessionName(polecat)

	running, err := m.tmux.HasSession(sessionID)
	if err != nil {
		return fmt.Errorf("checking session: %w", err)
	}
	if !running {
		return ErrSessionNotFound
	}

	return m.tmux.SendKeys(sessionID, message)
}

// StopAll terminates all sessions for this rig.
func (m *Manager) StopAll() error {
	infos, err := m.List()
	if err != nil {
		return err
	}

	var lastErr error
	for _, info := range infos {
		if err := m.Stop(info.Polecat); err != nil {
			lastErr = err
		}
	}

	return lastErr
}
