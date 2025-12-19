package witness

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/steveyegge/gastown/internal/rig"
)

// Common errors
var (
	ErrNotRunning     = errors.New("witness not running")
	ErrAlreadyRunning = errors.New("witness already running")
)

// Manager handles witness lifecycle and monitoring operations.
type Manager struct {
	rig     *rig.Rig
	workDir string
}

// NewManager creates a new witness manager for a rig.
func NewManager(r *rig.Rig) *Manager {
	return &Manager{
		rig:     r,
		workDir: r.Path,
	}
}

// stateFile returns the path to the witness state file.
func (m *Manager) stateFile() string {
	return filepath.Join(m.rig.Path, ".gastown", "witness.json")
}

// loadState loads witness state from disk.
func (m *Manager) loadState() (*Witness, error) {
	data, err := os.ReadFile(m.stateFile())
	if err != nil {
		if os.IsNotExist(err) {
			return &Witness{
				RigName: m.rig.Name,
				State:   StateStopped,
			}, nil
		}
		return nil, err
	}

	var w Witness
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, err
	}

	return &w, nil
}

// saveState persists witness state to disk.
func (m *Manager) saveState(w *Witness) error {
	dir := filepath.Dir(m.stateFile())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.stateFile(), data, 0644)
}

// Status returns the current witness status.
func (m *Manager) Status() (*Witness, error) {
	w, err := m.loadState()
	if err != nil {
		return nil, err
	}

	// If running, verify process is still alive
	if w.State == StateRunning && w.PID > 0 {
		if !processExists(w.PID) {
			w.State = StateStopped
			w.PID = 0
			m.saveState(w)
		}
	}

	// Update monitored polecats list
	w.MonitoredPolecats = m.rig.Polecats

	return w, nil
}

// Start starts the witness.
// If foreground is true, runs in the current process (blocking).
// Otherwise, spawns a background process.
func (m *Manager) Start(foreground bool) error {
	w, err := m.loadState()
	if err != nil {
		return err
	}

	if w.State == StateRunning && w.PID > 0 && processExists(w.PID) {
		return ErrAlreadyRunning
	}

	now := time.Now()
	w.State = StateRunning
	w.StartedAt = &now
	w.PID = os.Getpid() // For foreground mode; background would set actual PID
	w.MonitoredPolecats = m.rig.Polecats

	if err := m.saveState(w); err != nil {
		return err
	}

	if foreground {
		// Run the monitoring loop (blocking)
		return m.run(w)
	}

	// Background mode: spawn a new process
	// For MVP, we just mark as running - actual daemon implementation later
	return nil
}

// Stop stops the witness.
func (m *Manager) Stop() error {
	w, err := m.loadState()
	if err != nil {
		return err
	}

	if w.State != StateRunning {
		return ErrNotRunning
	}

	// If we have a PID, try to stop it gracefully
	if w.PID > 0 && w.PID != os.Getpid() {
		// Send SIGTERM
		if proc, err := os.FindProcess(w.PID); err == nil {
			proc.Signal(os.Interrupt)
		}
	}

	w.State = StateStopped
	w.PID = 0

	return m.saveState(w)
}

// run is the main monitoring loop (for foreground mode).
func (m *Manager) run(w *Witness) error {
	fmt.Println("Witness running...")
	fmt.Println("Press Ctrl+C to stop")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Perform health check
			if err := m.healthCheck(w); err != nil {
				fmt.Printf("Health check error: %v\n", err)
			}
		}
	}
}

// healthCheck performs a health check on all monitored polecats.
func (m *Manager) healthCheck(w *Witness) error {
	now := time.Now()
	w.LastCheckAt = &now
	w.Stats.TotalChecks++
	w.Stats.TodayChecks++

	// For MVP, just update state
	// Future: check keepalive files, nudge idle polecats, escalate stuck ones

	return m.saveState(w)
}

// processExists checks if a process with the given PID exists.
func processExists(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds; signal 0 tests existence
	err = proc.Signal(nil)
	return err == nil
}
