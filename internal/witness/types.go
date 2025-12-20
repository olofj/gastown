// Package witness provides the polecat monitoring agent.
package witness

import (
	"time"
)

// State represents the witness's running state.
type State string

const (
	// StateStopped means the witness is not running.
	StateStopped State = "stopped"

	// StateRunning means the witness is actively monitoring.
	StateRunning State = "running"

	// StatePaused means the witness is paused (not monitoring).
	StatePaused State = "paused"
)

// Witness represents a rig's polecat monitoring agent.
type Witness struct {
	// RigName is the rig this witness monitors.
	RigName string `json:"rig_name"`

	// State is the current running state.
	State State `json:"state"`

	// PID is the process ID if running in background.
	PID int `json:"pid,omitempty"`

	// StartedAt is when the witness was started.
	StartedAt *time.Time `json:"started_at,omitempty"`

	// MonitoredPolecats tracks polecats being monitored.
	MonitoredPolecats []string `json:"monitored_polecats,omitempty"`

	// LastCheckAt is when the last health check was performed.
	LastCheckAt *time.Time `json:"last_check_at,omitempty"`

	// Stats contains cumulative statistics.
	Stats WitnessStats `json:"stats"`
}

// WitnessStats contains cumulative witness statistics.
type WitnessStats struct {
	// TotalChecks is the total number of health checks performed.
	TotalChecks int `json:"total_checks"`

	// TotalNudges is the total number of nudges sent to polecats.
	TotalNudges int `json:"total_nudges"`

	// TotalEscalations is the total number of escalations to mayor.
	TotalEscalations int `json:"total_escalations"`

	// TodayChecks is the number of checks today.
	TodayChecks int `json:"today_checks"`

	// TodayNudges is the number of nudges today.
	TodayNudges int `json:"today_nudges"`
}
