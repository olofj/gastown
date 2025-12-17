package crew

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/steveyegge/gastown/internal/git"
	"github.com/steveyegge/gastown/internal/rig"
)

func TestManager_workerDir(t *testing.T) {
	r := &rig.Rig{
		Name: "test-rig",
		Path: "/tmp/test-rig",
	}
	m := NewManager(r, nil)

	got := m.workerDir("alice")
	want := "/tmp/test-rig/crew/alice"

	if got != want {
		t.Errorf("workerDir() = %q, want %q", got, want)
	}
}

func TestManager_stateFile(t *testing.T) {
	r := &rig.Rig{
		Name: "test-rig",
		Path: "/tmp/test-rig",
	}
	m := NewManager(r, nil)

	got := m.stateFile("bob")
	want := "/tmp/test-rig/crew/bob/state.json"

	if got != want {
		t.Errorf("stateFile() = %q, want %q", got, want)
	}
}

func TestManager_exists(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	crewDir := filepath.Join(tmpDir, "crew", "existing-worker")
	if err := os.MkdirAll(crewDir, 0755); err != nil {
		t.Fatal(err)
	}

	r := &rig.Rig{
		Name: "test-rig",
		Path: tmpDir,
	}
	m := NewManager(r, nil)

	tests := []struct {
		name   string
		worker string
		want   bool
	}{
		{"existing worker", "existing-worker", true},
		{"non-existing worker", "non-existing", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.exists(tt.worker)
			if got != tt.want {
				t.Errorf("exists(%q) = %v, want %v", tt.worker, got, tt.want)
			}
		})
	}
}

func TestManager_List_Empty(t *testing.T) {
	tmpDir := t.TempDir()

	r := &rig.Rig{
		Name: "test-rig",
		Path: tmpDir,
	}
	m := NewManager(r, git.NewGit(tmpDir))

	workers, err := m.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(workers) != 0 {
		t.Errorf("List() returned %d workers, want 0", len(workers))
	}
}

func TestManager_List_WithWorkers(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some fake worker directories
	workers := []string{"alice", "bob", "charlie"}
	for _, name := range workers {
		workerDir := filepath.Join(tmpDir, "crew", name)
		if err := os.MkdirAll(workerDir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	r := &rig.Rig{
		Name: "test-rig",
		Path: tmpDir,
	}
	m := NewManager(r, git.NewGit(tmpDir))

	gotWorkers, err := m.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(gotWorkers) != len(workers) {
		t.Errorf("List() returned %d workers, want %d", len(gotWorkers), len(workers))
	}
}

func TestManager_Names(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some fake worker directories
	expected := []string{"alice", "bob"}
	for _, name := range expected {
		workerDir := filepath.Join(tmpDir, "crew", name)
		if err := os.MkdirAll(workerDir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	r := &rig.Rig{
		Name: "test-rig",
		Path: tmpDir,
	}
	m := NewManager(r, git.NewGit(tmpDir))

	names, err := m.Names()
	if err != nil {
		t.Fatalf("Names() error = %v", err)
	}

	if len(names) != len(expected) {
		t.Errorf("Names() returned %d names, want %d", len(names), len(expected))
	}
}

func TestWorker_EffectiveBeadsDir(t *testing.T) {
	tests := []struct {
		name     string
		beadsDir string
		rigPath  string
		want     string
	}{
		{
			name:     "custom beads dir",
			beadsDir: "/custom/beads",
			rigPath:  "/tmp/rig",
			want:     "/custom/beads",
		},
		{
			name:     "default beads dir",
			beadsDir: "",
			rigPath:  "/tmp/rig",
			want:     "/tmp/rig/.beads",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Worker{
				BeadsDir: tt.beadsDir,
			}
			got := w.EffectiveBeadsDir(tt.rigPath)
			if got != tt.want {
				t.Errorf("EffectiveBeadsDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWorker_Summary(t *testing.T) {
	w := &Worker{
		Name:   "alice",
		State:  StateActive,
		Branch: "feature/test",
	}

	got := w.Summary()

	if got.Name != w.Name {
		t.Errorf("Summary().Name = %q, want %q", got.Name, w.Name)
	}
	if got.State != w.State {
		t.Errorf("Summary().State = %q, want %q", got.State, w.State)
	}
	if got.Branch != w.Branch {
		t.Errorf("Summary().Branch = %q, want %q", got.Branch, w.Branch)
	}
}
