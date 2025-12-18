package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestTownConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mayor", "town.json")

	original := &TownConfig{
		Type:      "town",
		Version:   1,
		Name:      "test-town",
		CreatedAt: time.Now().Truncate(time.Second),
	}

	if err := SaveTownConfig(path, original); err != nil {
		t.Fatalf("SaveTownConfig: %v", err)
	}

	loaded, err := LoadTownConfig(path)
	if err != nil {
		t.Fatalf("LoadTownConfig: %v", err)
	}

	if loaded.Name != original.Name {
		t.Errorf("Name = %q, want %q", loaded.Name, original.Name)
	}
	if loaded.Type != original.Type {
		t.Errorf("Type = %q, want %q", loaded.Type, original.Type)
	}
}

func TestRigsConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mayor", "rigs.json")

	original := &RigsConfig{
		Version: 1,
		Rigs: map[string]RigEntry{
			"gastown": {
				GitURL:  "git@github.com:steveyegge/gastown.git",
				AddedAt: time.Now().Truncate(time.Second),
				BeadsConfig: &BeadsConfig{
					Repo:   "local",
					Prefix: "gt-",
				},
			},
		},
	}

	if err := SaveRigsConfig(path, original); err != nil {
		t.Fatalf("SaveRigsConfig: %v", err)
	}

	loaded, err := LoadRigsConfig(path)
	if err != nil {
		t.Fatalf("LoadRigsConfig: %v", err)
	}

	if len(loaded.Rigs) != 1 {
		t.Errorf("Rigs count = %d, want 1", len(loaded.Rigs))
	}

	rig, ok := loaded.Rigs["gastown"]
	if !ok {
		t.Fatal("missing 'gastown' rig")
	}
	if rig.BeadsConfig == nil || rig.BeadsConfig.Prefix != "gt-" {
		t.Errorf("BeadsConfig.Prefix = %v, want 'gt-'", rig.BeadsConfig)
	}
}

func TestAgentStateRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	original := &AgentState{
		Role:       "mayor",
		LastActive: time.Now().Truncate(time.Second),
		Session:    "abc123",
		Extra: map[string]any{
			"custom": "value",
		},
	}

	if err := SaveAgentState(path, original); err != nil {
		t.Fatalf("SaveAgentState: %v", err)
	}

	loaded, err := LoadAgentState(path)
	if err != nil {
		t.Fatalf("LoadAgentState: %v", err)
	}

	if loaded.Role != original.Role {
		t.Errorf("Role = %q, want %q", loaded.Role, original.Role)
	}
	if loaded.Session != original.Session {
		t.Errorf("Session = %q, want %q", loaded.Session, original.Session)
	}
}

func TestLoadTownConfigNotFound(t *testing.T) {
	_, err := LoadTownConfig("/nonexistent/path.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestValidationErrors(t *testing.T) {
	// Missing name
	tc := &TownConfig{Type: "town", Version: 1}
	if err := validateTownConfig(tc); err == nil {
		t.Error("expected error for missing name")
	}

	// Wrong type
	tc = &TownConfig{Type: "wrong", Version: 1, Name: "test"}
	if err := validateTownConfig(tc); err == nil {
		t.Error("expected error for wrong type")
	}

	// Missing role
	as := &AgentState{}
	if err := validateAgentState(as); err == nil {
		t.Error("expected error for missing role")
	}
}
