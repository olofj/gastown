package polecat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

const (
	// PoolSize is the number of reusable names in the pool.
	PoolSize = 50

	// NamePrefix is the prefix for pooled polecat names.
	NamePrefix = "polecat-"
)

// NamePool manages a bounded pool of reusable polecat names.
// Names in the pool are polecat-01 through polecat-50.
// When the pool is exhausted, overflow names use rigname-N format.
type NamePool struct {
	mu sync.RWMutex

	// RigName is the rig this pool belongs to.
	RigName string `json:"rig_name"`

	// InUse tracks which pool indices are currently in use.
	// Key is the pool index (1-50), value is true if in use.
	InUse map[int]bool `json:"in_use"`

	// OverflowNext is the next overflow sequence number.
	// Starts at PoolSize+1 (51) and increments.
	OverflowNext int `json:"overflow_next"`

	// stateFile is the path to persist pool state.
	stateFile string
}

// NewNamePool creates a new name pool for a rig.
func NewNamePool(rigPath, rigName string) *NamePool {
	return &NamePool{
		RigName:      rigName,
		InUse:        make(map[int]bool),
		OverflowNext: PoolSize + 1,
		stateFile:    filepath.Join(rigPath, ".gastown", "namepool.json"),
	}
}

// Load loads the pool state from disk.
func (p *NamePool) Load() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	data, err := os.ReadFile(p.stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Initialize with empty state
			p.InUse = make(map[int]bool)
			p.OverflowNext = PoolSize + 1
			return nil
		}
		return err
	}

	var loaded NamePool
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}

	p.InUse = loaded.InUse
	if p.InUse == nil {
		p.InUse = make(map[int]bool)
	}
	p.OverflowNext = loaded.OverflowNext
	if p.OverflowNext < PoolSize+1 {
		p.OverflowNext = PoolSize + 1
	}

	return nil
}

// Save persists the pool state to disk.
func (p *NamePool) Save() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	dir := filepath.Dir(p.stateFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(p.stateFile, data, 0644)
}

// Allocate returns a name from the pool.
// It prefers lower-numbered pool slots, and falls back to overflow names
// when the pool is exhausted.
func (p *NamePool) Allocate() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Try to find first available slot in pool (prefer low numbers)
	for i := 1; i <= PoolSize; i++ {
		if !p.InUse[i] {
			p.InUse[i] = true
			return p.formatPoolName(i), nil
		}
	}

	// Pool exhausted, use overflow naming
	name := p.formatOverflowName(p.OverflowNext)
	p.OverflowNext++
	return name, nil
}

// Release returns a pooled name to the pool.
// For overflow names, this is a no-op (they are not reusable).
func (p *NamePool) Release(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	idx := p.parsePoolIndex(name)
	if idx > 0 && idx <= PoolSize {
		delete(p.InUse, idx)
	}
	// Overflow names are not reusable, so we don't track them
}

// IsPoolName returns true if the name is a pool name (polecat-NN format).
func (p *NamePool) IsPoolName(name string) bool {
	idx := p.parsePoolIndex(name)
	return idx > 0 && idx <= PoolSize
}

// ActiveCount returns the number of names currently in use from the pool.
func (p *NamePool) ActiveCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.InUse)
}

// ActiveNames returns a sorted list of names currently in use from the pool.
func (p *NamePool) ActiveNames() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var names []string
	for idx := range p.InUse {
		names = append(names, p.formatPoolName(idx))
	}
	sort.Strings(names)
	return names
}

// MarkInUse marks a name as in use (for reconciling with existing polecats).
func (p *NamePool) MarkInUse(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	idx := p.parsePoolIndex(name)
	if idx > 0 && idx <= PoolSize {
		p.InUse[idx] = true
	}
}

// Reconcile updates the pool state based on existing polecat directories.
// This should be called on startup to sync pool state with reality.
func (p *NamePool) Reconcile(existingPolecats []string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Clear current state
	p.InUse = make(map[int]bool)

	// Mark all existing polecats as in use
	for _, name := range existingPolecats {
		idx := p.parsePoolIndex(name)
		if idx > 0 && idx <= PoolSize {
			p.InUse[idx] = true
		}
	}
}

// formatPoolName formats a pool index as a name.
func (p *NamePool) formatPoolName(idx int) string {
	return fmt.Sprintf("%s%02d", NamePrefix, idx)
}

// formatOverflowName formats an overflow sequence number as a name.
func (p *NamePool) formatOverflowName(seq int) string {
	return fmt.Sprintf("%s-%d", p.RigName, seq)
}

// parsePoolIndex extracts the pool index from a pool name.
// Returns 0 if not a valid pool name.
func (p *NamePool) parsePoolIndex(name string) int {
	if len(name) < len(NamePrefix)+2 {
		return 0
	}
	if name[:len(NamePrefix)] != NamePrefix {
		return 0
	}

	var idx int
	_, err := fmt.Sscanf(name[len(NamePrefix):], "%d", &idx)
	if err != nil {
		return 0
	}
	return idx
}
