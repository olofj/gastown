package refinery

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/steveyegge/gastown/internal/rig"
)

// Common errors
var (
	ErrNotRunning    = errors.New("refinery not running")
	ErrAlreadyRunning = errors.New("refinery already running")
	ErrNoQueue       = errors.New("no items in queue")
)

// Manager handles refinery lifecycle and queue operations.
type Manager struct {
	rig     *rig.Rig
	workDir string
}

// NewManager creates a new refinery manager for a rig.
func NewManager(r *rig.Rig) *Manager {
	return &Manager{
		rig:     r,
		workDir: r.Path,
	}
}

// stateFile returns the path to the refinery state file.
func (m *Manager) stateFile() string {
	return filepath.Join(m.rig.Path, ".gastown", "refinery.json")
}

// loadState loads refinery state from disk.
func (m *Manager) loadState() (*Refinery, error) {
	data, err := os.ReadFile(m.stateFile())
	if err != nil {
		if os.IsNotExist(err) {
			return &Refinery{
				RigName: m.rig.Name,
				State:   StateStopped,
			}, nil
		}
		return nil, err
	}

	var ref Refinery
	if err := json.Unmarshal(data, &ref); err != nil {
		return nil, err
	}

	return &ref, nil
}

// saveState persists refinery state to disk.
func (m *Manager) saveState(ref *Refinery) error {
	dir := filepath.Dir(m.stateFile())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(ref, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.stateFile(), data, 0644)
}

// Status returns the current refinery status.
func (m *Manager) Status() (*Refinery, error) {
	ref, err := m.loadState()
	if err != nil {
		return nil, err
	}

	// If running, verify process is still alive
	if ref.State == StateRunning && ref.PID > 0 {
		if !processExists(ref.PID) {
			ref.State = StateStopped
			ref.PID = 0
			m.saveState(ref)
		}
	}

	return ref, nil
}

// Start starts the refinery.
// If foreground is true, runs in the current process (blocking).
// Otherwise, spawns a background process.
func (m *Manager) Start(foreground bool) error {
	ref, err := m.loadState()
	if err != nil {
		return err
	}

	if ref.State == StateRunning && ref.PID > 0 && processExists(ref.PID) {
		return ErrAlreadyRunning
	}

	now := time.Now()
	ref.State = StateRunning
	ref.StartedAt = &now
	ref.PID = os.Getpid() // For foreground mode; background would set actual PID

	if err := m.saveState(ref); err != nil {
		return err
	}

	if foreground {
		// Run the processing loop (blocking)
		return m.run(ref)
	}

	// Background mode: spawn a new process
	// For MVP, we just mark as running - actual daemon implementation in gt-ov2
	return nil
}

// Stop stops the refinery.
func (m *Manager) Stop() error {
	ref, err := m.loadState()
	if err != nil {
		return err
	}

	if ref.State != StateRunning {
		return ErrNotRunning
	}

	// If we have a PID, try to stop it gracefully
	if ref.PID > 0 && ref.PID != os.Getpid() {
		// Send SIGTERM
		if proc, err := os.FindProcess(ref.PID); err == nil {
			proc.Signal(os.Interrupt)
		}
	}

	ref.State = StateStopped
	ref.PID = 0

	return m.saveState(ref)
}

// Queue returns the current merge queue.
func (m *Manager) Queue() ([]QueueItem, error) {
	// Discover branches that look like polecat work branches
	branches, err := m.discoverWorkBranches()
	if err != nil {
		return nil, err
	}

	// Load any pending MRs from state
	ref, err := m.loadState()
	if err != nil {
		return nil, err
	}

	// Build queue items
	var items []QueueItem
	pos := 1

	// Add current processing item
	if ref.CurrentMR != nil {
		items = append(items, QueueItem{
			Position: 0, // 0 = currently processing
			MR:       ref.CurrentMR,
			Age:      formatAge(ref.CurrentMR.CreatedAt),
		})
	}

	// Add discovered branches as pending
	for _, branch := range branches {
		mr := m.branchToMR(branch)
		if mr != nil {
			items = append(items, QueueItem{
				Position: pos,
				MR:       mr,
				Age:      formatAge(mr.CreatedAt),
			})
			pos++
		}
	}

	return items, nil
}

// discoverWorkBranches finds branches that look like polecat work.
func (m *Manager) discoverWorkBranches() ([]string, error) {
	cmd := exec.Command("git", "branch", "-r", "--list", "origin/polecat/*")
	cmd.Dir = m.workDir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, nil // No remote branches
	}

	var branches []string
	for _, line := range strings.Split(stdout.String(), "\n") {
		branch := strings.TrimSpace(line)
		if branch != "" && !strings.Contains(branch, "->") {
			// Remove origin/ prefix
			branch = strings.TrimPrefix(branch, "origin/")
			branches = append(branches, branch)
		}
	}

	return branches, nil
}

// branchToMR converts a branch name to a merge request.
func (m *Manager) branchToMR(branch string) *MergeRequest {
	// Expected format: polecat/<worker>/<issue> or polecat/<worker>
	pattern := regexp.MustCompile(`^polecat/([^/]+)(?:/(.+))?$`)
	matches := pattern.FindStringSubmatch(branch)
	if matches == nil {
		return nil
	}

	worker := matches[1]
	issueID := ""
	if len(matches) > 2 {
		issueID = matches[2]
	}

	return &MergeRequest{
		ID:           fmt.Sprintf("mr-%s-%d", worker, time.Now().Unix()),
		Branch:       branch,
		Worker:       worker,
		IssueID:      issueID,
		TargetBranch: "main", // Default; swarm would use integration branch
		CreatedAt:    time.Now(), // Would ideally get from git
		Status:       MRPending,
	}
}

// run is the main processing loop (for foreground mode).
func (m *Manager) run(ref *Refinery) error {
	// MVP: Just a stub that returns immediately
	// Full implementation in gt-ov2
	fmt.Println("Refinery running (stub mode)...")
	fmt.Println("Press Ctrl+C to stop")

	// Would normally loop here processing the queue
	select {}
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

// formatAge formats a duration since the given time.
func formatAge(t time.Time) string {
	d := time.Since(t)

	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}
