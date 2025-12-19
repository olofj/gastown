// Package beads provides a wrapper for the bd (beads) CLI.
package beads

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// BuiltinMolecule defines a built-in molecule template.
type BuiltinMolecule struct {
	ID          string // Well-known ID (e.g., "mol-engineer-in-box")
	Title       string
	Description string
}

// BuiltinMolecules returns all built-in molecule definitions.
func BuiltinMolecules() []BuiltinMolecule {
	return []BuiltinMolecule{
		EngineerInBoxMolecule(),
		QuickFixMolecule(),
		ResearchMolecule(),
	}
}

// EngineerInBoxMolecule returns the engineer-in-box molecule definition.
// This is a full workflow from design to merge.
func EngineerInBoxMolecule() BuiltinMolecule {
	return BuiltinMolecule{
		ID:    "mol-engineer-in-box",
		Title: "Engineer in a Box",
		Description: `Full workflow from design to merge.

## Step: design
Think carefully about architecture. Consider:
- Existing patterns in the codebase
- Trade-offs between approaches
- Testability and maintainability

Write a brief design summary before proceeding.

## Step: implement
Write the code. Follow codebase conventions.
Needs: design

## Step: review
Self-review the changes. Look for:
- Bugs and edge cases
- Style issues
- Missing error handling
Needs: implement

## Step: test
Write and run tests. Cover happy path and edge cases.
Fix any failures before proceeding.
Needs: implement

## Step: submit
Submit for merge via refinery.
Needs: review, test`,
	}
}

// QuickFixMolecule returns the quick-fix molecule definition.
// This is a fast path for small changes.
func QuickFixMolecule() BuiltinMolecule {
	return BuiltinMolecule{
		ID:    "mol-quick-fix",
		Title: "Quick Fix",
		Description: `Fast path for small changes.

## Step: implement
Make the fix. Keep it focused.

## Step: test
Run relevant tests. Fix any regressions.
Needs: implement

## Step: submit
Submit for merge.
Needs: test`,
	}
}

// ResearchMolecule returns the research molecule definition.
// This is an investigation workflow.
func ResearchMolecule() BuiltinMolecule {
	return BuiltinMolecule{
		ID:    "mol-research",
		Title: "Research",
		Description: `Investigation workflow.

## Step: investigate
Explore the question. Search code, read docs,
understand context. Take notes.

## Step: document
Write up findings. Include:
- What you learned
- Recommendations
- Open questions
Needs: investigate`,
	}
}

// jsonlIssue represents an issue in the JSONL format.
// This struct matches the beads JSONL schema for direct file writes.
type jsonlIssue struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Priority    int    `json:"priority"`
	IssueType   string `json:"issue_type"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// SeedBuiltinMolecules creates all built-in molecules in the beads database.
// It skips molecules that already exist (by ID match).
// Returns the number of molecules created.
//
// Note: Since the bd CLI doesn't support the "molecule" type, this function
// writes directly to the JSONL file to create molecules with the proper type.
func (b *Beads) SeedBuiltinMolecules() (int, error) {
	molecules := BuiltinMolecules()
	created := 0

	// Find the JSONL file
	jsonlPath := filepath.Join(b.workDir, ".beads", "issues.jsonl")
	if _, err := os.Stat(jsonlPath); os.IsNotExist(err) {
		return 0, fmt.Errorf("beads JSONL not found: %s", jsonlPath)
	}

	// Read existing issues to check for duplicates
	existingIDs, err := readExistingIDs(jsonlPath)
	if err != nil {
		return 0, fmt.Errorf("reading existing issues: %w", err)
	}

	// Prepare new molecules to add
	var newMolecules []jsonlIssue
	now := time.Now().Format(time.RFC3339Nano)

	for _, mol := range molecules {
		if existingIDs[mol.ID] {
			continue // Already exists
		}

		newMolecules = append(newMolecules, jsonlIssue{
			ID:          mol.ID,
			Title:       mol.Title,
			Description: mol.Description,
			Status:      "open",
			Priority:    2, // Medium priority
			IssueType:   "molecule",
			CreatedAt:   now,
			UpdatedAt:   now,
		})
		created++
	}

	if len(newMolecules) == 0 {
		return 0, nil
	}

	// Append new molecules to the JSONL file
	f, err := os.OpenFile(jsonlPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return 0, fmt.Errorf("opening JSONL for append: %w", err)
	}
	defer f.Close()

	for _, mol := range newMolecules {
		line, err := json.Marshal(mol)
		if err != nil {
			return created, fmt.Errorf("marshaling molecule %s: %w", mol.ID, err)
		}
		if _, err := f.Write(append(line, '\n')); err != nil {
			return created, fmt.Errorf("writing molecule %s: %w", mol.ID, err)
		}
	}

	return created, nil
}

// readExistingIDs reads the JSONL file and returns a set of existing issue IDs.
func readExistingIDs(jsonlPath string) (map[string]bool, error) {
	ids := make(map[string]bool)

	f, err := os.Open(jsonlPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Increase buffer size for long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Just extract the ID field - we don't need to parse the full issue
		var partial struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(line, &partial); err != nil {
			continue // Skip malformed lines
		}
		if partial.ID != "" {
			ids[partial.ID] = true
		}
	}

	return ids, scanner.Err()
}
