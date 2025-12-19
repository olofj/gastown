package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/config"
	"github.com/steveyegge/gastown/internal/git"
	"github.com/steveyegge/gastown/internal/polecat"
	"github.com/steveyegge/gastown/internal/rig"
	"github.com/steveyegge/gastown/internal/session"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/tmux"
	"github.com/steveyegge/gastown/internal/workspace"
)

// polecatNames are Mad Max: Fury Road themed names for auto-generated polecats.
var polecatNames = []string{
	"Nux", "Toast", "Capable", "Cheedo", "Dag", "Rictus", "Slit", "Morsov",
	"Ace", "Coma", "Valkyrie", "Keeper", "Vuvalini", "Organic", "Immortan",
	"Corpus", "Doof", "Scabrous", "Splendid", "Fragile",
}

// Spawn command flags
var (
	spawnIssue   string
	spawnMessage string
	spawnNoStart bool
)

var spawnCmd = &cobra.Command{
	Use:     "spawn <rig/polecat> | <rig>",
	Aliases: []string{"sp"},
	Short:   "Spawn a polecat with work assignment",
	Long: `Spawn a polecat with a work assignment.

Creates a fresh polecat worktree, assigns an issue or task, and starts
a session. Polecats are ephemeral - they exist only while working.

If no polecat name is specified, generates a random name. If the specified
name already exists as a non-working polecat, it will be replaced with
a fresh worktree.

Examples:
  gt spawn gastown --issue gt-abc          # auto-generate polecat name
  gt spawn gastown/Toast --issue gt-def    # use specific name
  gt spawn gastown/Nux -m "Fix the tests"  # free-form task`,
	Args: cobra.ExactArgs(1),
	RunE: runSpawn,
}

func init() {
	spawnCmd.Flags().StringVar(&spawnIssue, "issue", "", "Beads issue ID to assign")
	spawnCmd.Flags().StringVarP(&spawnMessage, "message", "m", "", "Free-form task description")
	spawnCmd.Flags().BoolVar(&spawnNoStart, "no-start", false, "Assign work but don't start session")

	rootCmd.AddCommand(spawnCmd)
}

// BeadsIssue represents a beads issue from JSON output.
type BeadsIssue struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
	Type        string `json:"issue_type"`
	Status      string `json:"status"`
}

func runSpawn(cmd *cobra.Command, args []string) error {
	if spawnIssue == "" && spawnMessage == "" {
		return fmt.Errorf("must specify --issue or -m/--message")
	}

	// Parse address: rig/polecat or just rig
	rigName, polecatName, err := parseSpawnAddress(args[0])
	if err != nil {
		return err
	}

	// Find workspace and rig
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	rigsConfigPath := filepath.Join(townRoot, "mayor", "rigs.json")
	rigsConfig, err := config.LoadRigsConfig(rigsConfigPath)
	if err != nil {
		rigsConfig = &config.RigsConfig{Rigs: make(map[string]config.RigEntry)}
	}

	g := git.NewGit(townRoot)
	rigMgr := rig.NewManager(townRoot, rigsConfig, g)
	r, err := rigMgr.GetRig(rigName)
	if err != nil {
		return fmt.Errorf("rig '%s' not found", rigName)
	}

	// Get polecat manager
	polecatGit := git.NewGit(r.Path)
	polecatMgr := polecat.NewManager(r, polecatGit)

	// Ephemeral model: always create fresh polecat
	// If no name specified, generate one
	if polecatName == "" {
		polecatName = generatePolecatName(polecatMgr)
		fmt.Printf("Generated polecat name: %s\n", polecatName)
	}

	// Check if polecat already exists
	pc, err := polecatMgr.Get(polecatName)
	if err == nil {
		// Polecat exists - check if working
		if pc.State == polecat.StateWorking {
			return fmt.Errorf("polecat '%s' is already working on %s", polecatName, pc.Issue)
		}
		// Existing polecat not working - remove and recreate fresh
		fmt.Printf("Removing stale polecat %s for fresh worktree...\n", polecatName)
		if err := polecatMgr.Remove(polecatName, true); err != nil {
			return fmt.Errorf("removing stale polecat: %w", err)
		}
	} else if err != polecat.ErrPolecatNotFound {
		return fmt.Errorf("checking polecat: %w", err)
	}

	// Create fresh polecat with new worktree from main
	fmt.Printf("Creating fresh polecat %s...\n", polecatName)
	pc, err = polecatMgr.Add(polecatName)
	if err != nil {
		return fmt.Errorf("creating polecat: %w", err)
	}

	// Initialize beads in the new worktree
	fmt.Printf("Initializing beads in worktree...\n")
	if err := initBeadsInWorktree(pc.ClonePath); err != nil {
		// Non-fatal - beads might already be initialized
		fmt.Printf("  %s\n", style.Dim.Render(fmt.Sprintf("(beads init: %v)", err)))
	}

	// Get issue details if specified
	var issue *BeadsIssue
	if spawnIssue != "" {
		issue, err = fetchBeadsIssue(r.Path, spawnIssue)
		if err != nil {
			return fmt.Errorf("fetching issue %s: %w", spawnIssue, err)
		}
	}

	// Assign issue/task to polecat
	assignmentID := spawnIssue
	if assignmentID == "" {
		assignmentID = "task:" + time.Now().Format("20060102-150405")
	}
	if err := polecatMgr.AssignIssue(polecatName, assignmentID); err != nil {
		return fmt.Errorf("assigning issue: %w", err)
	}

	fmt.Printf("%s Assigned %s to %s/%s\n",
		style.Bold.Render("✓"),
		assignmentID, rigName, polecatName)

	// Stop here if --no-start
	if spawnNoStart {
		fmt.Printf("\n  %s\n", style.Dim.Render("Use 'gt session start' to start the session"))
		return nil
	}

	// Start session
	t := tmux.NewTmux()
	sessMgr := session.NewManager(t, r)

	// Check if already running
	running, _ := sessMgr.IsRunning(polecatName)
	if running {
		// Just inject the context
		fmt.Printf("Session already running, injecting context...\n")
	} else {
		// Start new session
		fmt.Printf("Starting session for %s/%s...\n", rigName, polecatName)
		if err := sessMgr.Start(polecatName, session.StartOptions{}); err != nil {
			return fmt.Errorf("starting session: %w", err)
		}
		// Wait for Claude to fully initialize (needs 4-5s for prompt)
		fmt.Printf("Waiting for Claude to initialize...\n")
		time.Sleep(5 * time.Second)
	}

	// Inject initial context
	context := buildSpawnContext(issue, spawnMessage)
	fmt.Printf("Injecting work assignment...\n")
	if err := sessMgr.Inject(polecatName, context); err != nil {
		return fmt.Errorf("injecting context: %w", err)
	}

	fmt.Printf("%s Session started. Attach with: %s\n",
		style.Bold.Render("✓"),
		style.Dim.Render(fmt.Sprintf("gt session at %s/%s", rigName, polecatName)))

	return nil
}

// parseSpawnAddress parses "rig/polecat" or "rig".
func parseSpawnAddress(addr string) (rigName, polecatName string, err error) {
	if strings.Contains(addr, "/") {
		parts := strings.SplitN(addr, "/", 2)
		if parts[0] == "" {
			return "", "", fmt.Errorf("invalid address: missing rig name")
		}
		return parts[0], parts[1], nil
	}
	return addr, "", nil
}

// generatePolecatName generates a unique polecat name that doesn't conflict with existing ones.
func generatePolecatName(mgr *polecat.Manager) string {
	existing, _ := mgr.List()
	existingNames := make(map[string]bool)
	for _, p := range existing {
		existingNames[p.Name] = true
	}

	// Try to find an unused name from the list
	// Shuffle to avoid always picking the same name
	shuffled := make([]string, len(polecatNames))
	copy(shuffled, polecatNames)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	for _, name := range shuffled {
		if !existingNames[name] {
			return name
		}
	}

	// All names taken, generate one with a number suffix
	base := shuffled[0]
	for i := 2; ; i++ {
		name := fmt.Sprintf("%s%d", base, i)
		if !existingNames[name] {
			return name
		}
	}
}

// initBeadsInWorktree initializes beads in a new polecat worktree.
func initBeadsInWorktree(worktreePath string) error {
	cmd := exec.Command("bd", "init")
	cmd.Dir = worktreePath

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return fmt.Errorf("%s", errMsg)
		}
		return err
	}

	return nil
}

// fetchBeadsIssue gets issue details from beads CLI.
func fetchBeadsIssue(rigPath, issueID string) (*BeadsIssue, error) {
	cmd := exec.Command("bd", "show", issueID, "--json")
	cmd.Dir = rigPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return nil, fmt.Errorf("%s", errMsg)
		}
		return nil, err
	}

	// bd show --json returns an array, take the first element
	var issues []BeadsIssue
	if err := json.Unmarshal(stdout.Bytes(), &issues); err != nil {
		return nil, fmt.Errorf("parsing issue: %w", err)
	}
	if len(issues) == 0 {
		return nil, fmt.Errorf("issue not found: %s", issueID)
	}

	return &issues[0], nil
}

// buildSpawnContext creates the initial context message for the polecat.
func buildSpawnContext(issue *BeadsIssue, message string) string {
	var sb strings.Builder

	sb.WriteString("[SPAWN] You have been assigned work.\n\n")

	if issue != nil {
		sb.WriteString(fmt.Sprintf("Issue: %s\n", issue.ID))
		sb.WriteString(fmt.Sprintf("Title: %s\n", issue.Title))
		sb.WriteString(fmt.Sprintf("Priority: P%d\n", issue.Priority))
		sb.WriteString(fmt.Sprintf("Type: %s\n", issue.Type))
		if issue.Description != "" {
			sb.WriteString(fmt.Sprintf("\nDescription:\n%s\n", issue.Description))
		}
	} else if message != "" {
		sb.WriteString(fmt.Sprintf("Task: %s\n", message))
	}

	sb.WriteString("\nWork on this task. When complete, commit your changes and signal DONE.\n")

	return sb.String()
}
