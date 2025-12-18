// Package config provides configuration types and serialization for Gas Town.
package config

import "time"

// TownConfig represents the main town configuration (mayor/town.json).
type TownConfig struct {
	Type      string    `json:"type"`       // "town"
	Version   int       `json:"version"`    // schema version
	Name      string    `json:"name"`       // town identifier
	CreatedAt time.Time `json:"created_at"`
}

// RigsConfig represents the rigs registry (mayor/rigs.json).
type RigsConfig struct {
	Version int                 `json:"version"`
	Rigs    map[string]RigEntry `json:"rigs"`
}

// RigEntry represents a single rig in the registry.
type RigEntry struct {
	GitURL      string       `json:"git_url"`
	AddedAt     time.Time    `json:"added_at"`
	BeadsConfig *BeadsConfig `json:"beads,omitempty"`
}

// BeadsConfig represents beads configuration for a rig.
type BeadsConfig struct {
	Repo   string `json:"repo"`   // "local" | path | git-url
	Prefix string `json:"prefix"` // issue prefix
}

// AgentState represents an agent's current state (*/state.json).
type AgentState struct {
	Role       string         `json:"role"`              // "mayor", "witness", etc.
	LastActive time.Time      `json:"last_active"`
	Session    string         `json:"session,omitempty"`
	Extra      map[string]any `json:"extra,omitempty"`
}

// CurrentTownVersion is the current schema version for TownConfig.
const CurrentTownVersion = 1

// CurrentRigsVersion is the current schema version for RigsConfig.
const CurrentRigsVersion = 1
