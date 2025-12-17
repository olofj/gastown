// Package crew provides crew worker management.
// Crew workers are user-managed persistent workspaces within a rig,
// as opposed to polecats which are AI-managed workers.
package crew

import "time"

// State represents the current state of a crew worker.
type State string

const (
	// StateActive means the crew workspace is active.
	StateActive State = "active"

	// StateInactive means the crew workspace is inactive.
	StateInactive State = "inactive"
)

// Worker represents a crew member's workspace in a rig.
// Unlike polecats, crew workers are managed by the Overseer (human),
// not by AI agents.
type Worker struct {
	// Name is the crew worker identifier.
	Name string `json:"name"`

	// Rig is the rig this worker belongs to.
	Rig string `json:"rig"`

	// State is the current state.
	State State `json:"state"`

	// ClonePath is the path to the worker's clone.
	ClonePath string `json:"clone_path"`

	// Branch is the current git branch (if any).
	Branch string `json:"branch,omitempty"`

	// BeadsDir is an optional custom beads directory.
	// If empty, defaults to the rig's .beads/ directory.
	BeadsDir string `json:"beads_dir,omitempty"`

	// CreatedAt is when the worker was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the worker was last updated.
	UpdatedAt time.Time `json:"updated_at"`
}

// Summary provides a concise view of crew worker status.
type Summary struct {
	Name   string `json:"name"`
	State  State  `json:"state"`
	Branch string `json:"branch,omitempty"`
}

// Summary returns a Summary for this worker.
func (w *Worker) Summary() Summary {
	return Summary{
		Name:   w.Name,
		State:  w.State,
		Branch: w.Branch,
	}
}

// EffectiveBeadsDir returns the beads directory to use.
// Returns the custom BeadsDir if set, otherwise returns the rig default path.
func (w *Worker) EffectiveBeadsDir(rigPath string) string {
	if w.BeadsDir != "" {
		return w.BeadsDir
	}
	return rigPath + "/.beads"
}
