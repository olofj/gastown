package polecat

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNamePool_Allocate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "namepool-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	pool := NewNamePool(tmpDir, "testrig")

	// First allocation should be polecat-01
	name, err := pool.Allocate()
	if err != nil {
		t.Fatalf("Allocate error: %v", err)
	}
	if name != "polecat-01" {
		t.Errorf("expected polecat-01, got %s", name)
	}

	// Second allocation should be polecat-02
	name, err = pool.Allocate()
	if err != nil {
		t.Fatalf("Allocate error: %v", err)
	}
	if name != "polecat-02" {
		t.Errorf("expected polecat-02, got %s", name)
	}
}

func TestNamePool_Release(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "namepool-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	pool := NewNamePool(tmpDir, "testrig")

	// Allocate first two
	name1, _ := pool.Allocate()
	name2, _ := pool.Allocate()

	if name1 != "polecat-01" || name2 != "polecat-02" {
		t.Fatalf("unexpected allocations: %s, %s", name1, name2)
	}

	// Release first one
	pool.Release("polecat-01")

	// Next allocation should reuse polecat-01
	name, _ := pool.Allocate()
	if name != "polecat-01" {
		t.Errorf("expected polecat-01 to be reused, got %s", name)
	}
}

func TestNamePool_PrefersLowNumbers(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "namepool-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	pool := NewNamePool(tmpDir, "testrig")

	// Allocate first 5
	for i := 0; i < 5; i++ {
		pool.Allocate()
	}

	// Release 03 and 01
	pool.Release("polecat-03")
	pool.Release("polecat-01")

	// Next allocation should be 01 (lowest available)
	name, _ := pool.Allocate()
	if name != "polecat-01" {
		t.Errorf("expected polecat-01 (lowest), got %s", name)
	}

	// Next should be 03
	name, _ = pool.Allocate()
	if name != "polecat-03" {
		t.Errorf("expected polecat-03, got %s", name)
	}
}

func TestNamePool_Overflow(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "namepool-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	pool := NewNamePool(tmpDir, "gastown")

	// Exhaust the pool
	for i := 0; i < PoolSize; i++ {
		pool.Allocate()
	}

	// Next allocation should be overflow format
	name, err := pool.Allocate()
	if err != nil {
		t.Fatalf("Allocate error: %v", err)
	}
	expected := "gastown-51"
	if name != expected {
		t.Errorf("expected overflow name %s, got %s", expected, name)
	}

	// Next overflow
	name, _ = pool.Allocate()
	if name != "gastown-52" {
		t.Errorf("expected gastown-52, got %s", name)
	}
}

func TestNamePool_OverflowNotReusable(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "namepool-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	pool := NewNamePool(tmpDir, "gastown")

	// Exhaust the pool
	for i := 0; i < PoolSize; i++ {
		pool.Allocate()
	}

	// Get overflow name
	overflow1, _ := pool.Allocate()
	if overflow1 != "gastown-51" {
		t.Fatalf("expected gastown-51, got %s", overflow1)
	}

	// Release it - should not be reused
	pool.Release(overflow1)

	// Next allocation should be gastown-52, not gastown-51
	name, _ := pool.Allocate()
	if name != "gastown-52" {
		t.Errorf("expected gastown-52 (overflow increments), got %s", name)
	}
}

func TestNamePool_SaveLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "namepool-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	pool := NewNamePool(tmpDir, "testrig")

	// Allocate some names
	pool.Allocate() // 01
	pool.Allocate() // 02
	pool.Allocate() // 03
	pool.Release("polecat-02")

	// Save state
	if err := pool.Save(); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Create new pool and load
	pool2 := NewNamePool(tmpDir, "testrig")
	if err := pool2.Load(); err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// Should have 01 and 03 in use
	if pool2.ActiveCount() != 2 {
		t.Errorf("expected 2 active, got %d", pool2.ActiveCount())
	}

	// Next allocation should be 02 (released slot)
	name, _ := pool2.Allocate()
	if name != "polecat-02" {
		t.Errorf("expected polecat-02, got %s", name)
	}
}

func TestNamePool_Reconcile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "namepool-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	pool := NewNamePool(tmpDir, "testrig")

	// Simulate existing polecats from filesystem
	existing := []string{"polecat-03", "polecat-07", "some-other-name"}

	pool.Reconcile(existing)

	if pool.ActiveCount() != 2 {
		t.Errorf("expected 2 active after reconcile, got %d", pool.ActiveCount())
	}

	// Should allocate 01 first (not 03 or 07)
	name, _ := pool.Allocate()
	if name != "polecat-01" {
		t.Errorf("expected polecat-01, got %s", name)
	}
}

func TestNamePool_IsPoolName(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "namepool-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	pool := NewNamePool(tmpDir, "testrig")

	tests := []struct {
		name     string
		expected bool
	}{
		{"polecat-01", true},
		{"polecat-50", true},
		{"polecat-51", false}, // > PoolSize
		{"gastown-51", false}, // overflow format
		{"Nux", false},        // legacy name
		{"polecat-", false},   // invalid
		{"polecat-abc", false},
	}

	for _, tc := range tests {
		result := pool.IsPoolName(tc.name)
		if result != tc.expected {
			t.Errorf("IsPoolName(%q) = %v, expected %v", tc.name, result, tc.expected)
		}
	}
}

func TestNamePool_ActiveNames(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "namepool-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	pool := NewNamePool(tmpDir, "testrig")

	pool.Allocate() // 01
	pool.Allocate() // 02
	pool.Allocate() // 03
	pool.Release("polecat-02")

	names := pool.ActiveNames()
	if len(names) != 2 {
		t.Errorf("expected 2 active names, got %d", len(names))
	}
	if names[0] != "polecat-01" || names[1] != "polecat-03" {
		t.Errorf("expected [polecat-01, polecat-03], got %v", names)
	}
}

func TestNamePool_MarkInUse(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "namepool-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	pool := NewNamePool(tmpDir, "testrig")

	// Mark some slots as in use
	pool.MarkInUse("polecat-05")
	pool.MarkInUse("polecat-10")

	// Allocate should skip those
	name, _ := pool.Allocate()
	if name != "polecat-01" {
		t.Errorf("expected polecat-01, got %s", name)
	}

	// Mark more and verify count
	if pool.ActiveCount() != 3 { // 01, 05, 10
		t.Errorf("expected 3 active, got %d", pool.ActiveCount())
	}
}

func TestNamePool_StateFilePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "namepool-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	pool := NewNamePool(tmpDir, "testrig")
	pool.Allocate()
	if err := pool.Save(); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Verify file was created in expected location
	expectedPath := filepath.Join(tmpDir, ".gastown", "namepool.json")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Errorf("state file not found at expected path: %v", err)
	}
}
