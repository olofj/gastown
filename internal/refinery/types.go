// Package refinery provides the merge queue processing agent.
package refinery

import "time"

// State represents the refinery's running state.
type State string

const (
	// StateStopped means the refinery is not running.
	StateStopped State = "stopped"

	// StateRunning means the refinery is actively processing.
	StateRunning State = "running"

	// StatePaused means the refinery is paused (not processing new items).
	StatePaused State = "paused"
)

// Refinery represents a rig's merge queue processor.
type Refinery struct {
	// RigName is the rig this refinery processes.
	RigName string `json:"rig_name"`

	// State is the current running state.
	State State `json:"state"`

	// PID is the process ID if running in background.
	PID int `json:"pid,omitempty"`

	// StartedAt is when the refinery was started.
	StartedAt *time.Time `json:"started_at,omitempty"`

	// CurrentMR is the merge request currently being processed.
	CurrentMR *MergeRequest `json:"current_mr,omitempty"`

	// LastMergeAt is when the last successful merge happened.
	LastMergeAt *time.Time `json:"last_merge_at,omitempty"`

	// Stats contains cumulative statistics.
	Stats RefineryStats `json:"stats"`
}

// MergeRequest represents a branch waiting to be merged.
type MergeRequest struct {
	// ID is a unique identifier for this merge request.
	ID string `json:"id"`

	// Branch is the source branch name (e.g., "polecat/Toast/gt-abc").
	Branch string `json:"branch"`

	// Worker is the polecat that created this branch.
	Worker string `json:"worker"`

	// IssueID is the beads issue being worked on.
	IssueID string `json:"issue_id"`

	// SwarmID is the swarm this work belongs to (if any).
	SwarmID string `json:"swarm_id,omitempty"`

	// TargetBranch is where this should merge (usually integration or main).
	TargetBranch string `json:"target_branch"`

	// CreatedAt is when the MR was queued.
	CreatedAt time.Time `json:"created_at"`

	// Status is the current status of the merge request.
	Status MRStatus `json:"status"`

	// Error contains error details if Status is MRFailed.
	Error string `json:"error,omitempty"`
}

// MRStatus represents the status of a merge request.
type MRStatus string

const (
	// MRPending means the MR is waiting to be processed.
	MRPending MRStatus = "pending"

	// MRProcessing means the MR is currently being merged.
	MRProcessing MRStatus = "processing"

	// MRMerged means the MR was successfully merged.
	MRMerged MRStatus = "merged"

	// MRFailed means the merge failed (conflict or error).
	MRFailed MRStatus = "failed"

	// MRSkipped means the MR was skipped (duplicate, outdated, etc).
	MRSkipped MRStatus = "skipped"
)

// RefineryStats contains cumulative refinery statistics.
type RefineryStats struct {
	// TotalMerged is the total number of successful merges.
	TotalMerged int `json:"total_merged"`

	// TotalFailed is the total number of failed merges.
	TotalFailed int `json:"total_failed"`

	// TotalSkipped is the total number of skipped MRs.
	TotalSkipped int `json:"total_skipped"`

	// TodayMerged is the number of merges today.
	TodayMerged int `json:"today_merged"`

	// TodayFailed is the number of failures today.
	TodayFailed int `json:"today_failed"`
}

// QueueItem represents an item in the merge queue for display.
type QueueItem struct {
	Position  int       `json:"position"`
	MR        *MergeRequest `json:"mr"`
	Age       string    `json:"age"`
}
