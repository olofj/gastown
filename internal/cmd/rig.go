// Package cmd provides CLI commands for the gt tool.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/config"
	"github.com/steveyegge/gastown/internal/git"
	"github.com/steveyegge/gastown/internal/polecat"
	"github.com/steveyegge/gastown/internal/refinery"
	"github.com/steveyegge/gastown/internal/rig"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/workspace"
)

var rigCmd = &cobra.Command{
	Use:   "rig",
	Short: "Manage rigs in the workspace",
	Long: `Manage rigs (project containers) in the Gas Town workspace.

A rig is a container for managing a project and its agents:
  - refinery/rig/  Canonical main clone (Refinery's working copy)
  - mayor/rig/     Mayor's working clone for this rig
  - crew/<name>/   Human workspace(s)
  - witness/       Witness agent (no clone)
  - polecats/      Worker directories
  - .beads/        Rig-level issue tracking`,
}

var rigAddCmd = &cobra.Command{
	Use:   "add <name> <git-url>",
	Short: "Add a new rig to the workspace",
	Long: `Add a new rig by cloning a repository.

This creates a rig container with:
  - config.json           Rig configuration
  - .beads/               Rig-level issue tracking (initialized)
  - refinery/rig/         Canonical main clone
  - mayor/rig/            Mayor's working clone
  - crew/main/            Default human workspace
  - witness/              Witness agent directory
  - polecats/             Worker directory (empty)

Example:
  gt rig add gastown https://github.com/steveyegge/gastown
  gt rig add my-project git@github.com:user/repo.git --prefix mp`,
	Args: cobra.ExactArgs(2),
	RunE: runRigAdd,
}

var rigListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all rigs in the workspace",
	RunE:  runRigList,
}

var rigRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a rig from the registry (does not delete files)",
	Args:  cobra.ExactArgs(1),
	RunE:  runRigRemove,
}

var rigInfoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show detailed information about a rig",
	Long: `Show detailed status information for a specific rig.

Displays:
  - Rig path and git URL
  - Active polecats with status
  - Refinery status
  - Witness status
  - Beads summary (open issues count)

Example:
  gt rig info gastown`,
	Args: cobra.ExactArgs(1),
	RunE: runRigInfo,
}

// Flags
var (
	rigAddPrefix string
	rigAddCrew   string
)

func init() {
	rootCmd.AddCommand(rigCmd)
	rigCmd.AddCommand(rigAddCmd)
	rigCmd.AddCommand(rigListCmd)
	rigCmd.AddCommand(rigRemoveCmd)
	rigCmd.AddCommand(rigInfoCmd)

	rigAddCmd.Flags().StringVar(&rigAddPrefix, "prefix", "", "Beads issue prefix (default: derived from name)")
	rigAddCmd.Flags().StringVar(&rigAddCrew, "crew", "main", "Default crew workspace name")
}

func runRigAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	gitURL := args[1]

	// Find workspace
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	// Load rigs config
	rigsPath := filepath.Join(townRoot, "mayor", "rigs.json")
	rigsConfig, err := config.LoadRigsConfig(rigsPath)
	if err != nil {
		// Create new if doesn't exist
		rigsConfig = &config.RigsConfig{
			Version: 1,
			Rigs:    make(map[string]config.RigEntry),
		}
	}

	// Create rig manager
	g := git.NewGit(townRoot)
	mgr := rig.NewManager(townRoot, rigsConfig, g)

	fmt.Printf("Creating rig %s...\n", style.Bold.Render(name))
	fmt.Printf("  Repository: %s\n", gitURL)

	startTime := time.Now()

	// Add the rig
	newRig, err := mgr.AddRig(rig.AddRigOptions{
		Name:        name,
		GitURL:      gitURL,
		BeadsPrefix: rigAddPrefix,
		CrewName:    rigAddCrew,
	})
	if err != nil {
		return fmt.Errorf("adding rig: %w", err)
	}

	// Save updated rigs config
	if err := config.SaveRigsConfig(rigsPath, rigsConfig); err != nil {
		return fmt.Errorf("saving rigs config: %w", err)
	}

	elapsed := time.Since(startTime)

	fmt.Printf("\n%s Rig created in %.1fs\n", style.Success.Render("✓"), elapsed.Seconds())
	fmt.Printf("\nStructure:\n")
	fmt.Printf("  %s/\n", name)
	fmt.Printf("  ├── config.json\n")
	fmt.Printf("  ├── .beads/           (prefix: %s)\n", newRig.Config.Prefix)
	fmt.Printf("  ├── refinery/rig/     (canonical main)\n")
	fmt.Printf("  ├── mayor/rig/        (mayor's clone)\n")
	fmt.Printf("  ├── crew/%s/        (your workspace)\n", rigAddCrew)
	fmt.Printf("  ├── witness/\n")
	fmt.Printf("  └── polecats/\n")

	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  cd %s/crew/%s    # Work in your clone\n", filepath.Join(townRoot, name), rigAddCrew)
	fmt.Printf("  bd ready                 # See available work\n")

	return nil
}

func runRigList(cmd *cobra.Command, args []string) error {
	// Find workspace
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	// Load rigs config
	rigsPath := filepath.Join(townRoot, "mayor", "rigs.json")
	rigsConfig, err := config.LoadRigsConfig(rigsPath)
	if err != nil {
		fmt.Println("No rigs configured.")
		return nil
	}

	if len(rigsConfig.Rigs) == 0 {
		fmt.Println("No rigs configured.")
		fmt.Printf("\nAdd one with: %s\n", style.Dim.Render("gt rig add <name> <git-url>"))
		return nil
	}

	// Create rig manager to get details
	g := git.NewGit(townRoot)
	mgr := rig.NewManager(townRoot, rigsConfig, g)

	fmt.Printf("Rigs in %s:\n\n", townRoot)

	for name := range rigsConfig.Rigs {
		r, err := mgr.GetRig(name)
		if err != nil {
			fmt.Printf("  %s %s\n", style.Warning.Render("!"), name)
			continue
		}

		summary := r.Summary()
		fmt.Printf("  %s\n", style.Bold.Render(name))
		fmt.Printf("    Polecats: %d  Crew: %d\n", summary.PolecatCount, summary.CrewCount)

		agents := []string{}
		if summary.HasRefinery {
			agents = append(agents, "refinery")
		}
		if summary.HasWitness {
			agents = append(agents, "witness")
		}
		if r.HasMayor {
			agents = append(agents, "mayor")
		}
		if len(agents) > 0 {
			fmt.Printf("    Agents: %v\n", agents)
		}
		fmt.Println()
	}

	return nil
}

func runRigRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Find workspace
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	// Load rigs config
	rigsPath := filepath.Join(townRoot, "mayor", "rigs.json")
	rigsConfig, err := config.LoadRigsConfig(rigsPath)
	if err != nil {
		return fmt.Errorf("loading rigs config: %w", err)
	}

	// Create rig manager
	g := git.NewGit(townRoot)
	mgr := rig.NewManager(townRoot, rigsConfig, g)

	if err := mgr.RemoveRig(name); err != nil {
		return fmt.Errorf("removing rig: %w", err)
	}

	// Save updated config
	if err := config.SaveRigsConfig(rigsPath, rigsConfig); err != nil {
		return fmt.Errorf("saving rigs config: %w", err)
	}

	fmt.Printf("%s Rig %s removed from registry\n", style.Success.Render("✓"), name)
	fmt.Printf("\nNote: Files at %s were NOT deleted.\n", filepath.Join(townRoot, name))
	fmt.Printf("To delete: %s\n", style.Dim.Render(fmt.Sprintf("rm -rf %s", filepath.Join(townRoot, name))))

	return nil
}

// Helper to check if path exists
func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func runRigInfo(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Find workspace
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	// Load rigs config
	rigsPath := filepath.Join(townRoot, "mayor", "rigs.json")
	rigsConfig, err := config.LoadRigsConfig(rigsPath)
	if err != nil {
		return fmt.Errorf("loading rigs config: %w", err)
	}

	// Create rig manager and get the rig
	g := git.NewGit(townRoot)
	mgr := rig.NewManager(townRoot, rigsConfig, g)

	r, err := mgr.GetRig(name)
	if err != nil {
		return fmt.Errorf("rig not found: %s", name)
	}

	// Print rig header
	fmt.Printf("%s\n", style.Bold.Render(r.Name))
	fmt.Printf("  Path: %s\n", r.Path)
	fmt.Printf("  Git:  %s\n", r.GitURL)
	if r.Config != nil && r.Config.Prefix != "" {
		fmt.Printf("  Beads prefix: %s\n", r.Config.Prefix)
	}
	fmt.Println()

	// Show polecats
	fmt.Printf("%s\n", style.Bold.Render("Polecats"))
	polecatMgr := polecat.NewManager(r, g)
	polecats, err := polecatMgr.List()
	if err != nil || len(polecats) == 0 {
		fmt.Printf("  %s\n", style.Dim.Render("(none)"))
	} else {
		for _, p := range polecats {
			stateStr := formatPolecatState(p.State)
			if p.Issue != "" {
				fmt.Printf("  %s  %s  %s\n", p.Name, stateStr, style.Dim.Render(p.Issue))
			} else {
				fmt.Printf("  %s  %s\n", p.Name, stateStr)
			}
		}
	}
	fmt.Println()

	// Show crew workers
	fmt.Printf("%s\n", style.Bold.Render("Crew"))
	if len(r.Crew) == 0 {
		fmt.Printf("  %s\n", style.Dim.Render("(none)"))
	} else {
		for _, c := range r.Crew {
			fmt.Printf("  %s\n", c)
		}
	}
	fmt.Println()

	// Show refinery status
	fmt.Printf("%s\n", style.Bold.Render("Refinery"))
	if r.HasRefinery {
		refMgr := refinery.NewManager(r)
		refStatus, err := refMgr.Status()
		if err != nil {
			fmt.Printf("  %s %s\n", style.Warning.Render("!"), "Error loading status")
		} else {
			stateStr := formatRefineryState(refStatus.State)
			fmt.Printf("  Status: %s\n", stateStr)
			if refStatus.State == refinery.StateRunning && refStatus.PID > 0 {
				fmt.Printf("  PID: %d\n", refStatus.PID)
			}
			if refStatus.CurrentMR != nil {
				fmt.Printf("  Current: %s (%s)\n", refStatus.CurrentMR.Branch, refStatus.CurrentMR.Worker)
			}
			if refStatus.Stats.TotalMerged > 0 || refStatus.Stats.TotalFailed > 0 {
				fmt.Printf("  Stats: %d merged, %d failed\n", refStatus.Stats.TotalMerged, refStatus.Stats.TotalFailed)
			}
		}
	} else {
		fmt.Printf("  %s\n", style.Dim.Render("(not configured)"))
	}
	fmt.Println()

	// Show witness status
	fmt.Printf("%s\n", style.Bold.Render("Witness"))
	if r.HasWitness {
		witnessState := loadWitnessState(r.Path)
		if witnessState != nil {
			fmt.Printf("  Last active: %s\n", formatTimeAgo(witnessState.LastActive))
			if witnessState.Session != "" {
				fmt.Printf("  Session: %s\n", witnessState.Session)
			}
		} else {
			fmt.Printf("  %s\n", style.Success.Render("configured"))
		}
	} else {
		fmt.Printf("  %s\n", style.Dim.Render("(not configured)"))
	}
	fmt.Println()

	// Show mayor status
	fmt.Printf("%s\n", style.Bold.Render("Mayor"))
	if r.HasMayor {
		mayorState := loadMayorState(r.Path)
		if mayorState != nil {
			fmt.Printf("  Last active: %s\n", formatTimeAgo(mayorState.LastActive))
		} else {
			fmt.Printf("  %s\n", style.Success.Render("configured"))
		}
	} else {
		fmt.Printf("  %s\n", style.Dim.Render("(not configured)"))
	}
	fmt.Println()

	// Show beads summary
	fmt.Printf("%s\n", style.Bold.Render("Beads"))
	beadsStats := getBeadsSummary(r.Path)
	if beadsStats != nil {
		fmt.Printf("  Open: %d  In Progress: %d  Closed: %d\n",
			beadsStats.Open, beadsStats.InProgress, beadsStats.Closed)
		if beadsStats.Blocked > 0 {
			fmt.Printf("  Blocked: %d\n", beadsStats.Blocked)
		}
	} else {
		fmt.Printf("  %s\n", style.Dim.Render("(beads not initialized)"))
	}

	return nil
}

// formatPolecatState returns a styled string for polecat state.
func formatPolecatState(state polecat.State) string {
	switch state {
	case polecat.StateIdle:
		return style.Dim.Render("idle")
	case polecat.StateActive:
		return style.Info.Render("active")
	case polecat.StateWorking:
		return style.Success.Render("working")
	case polecat.StateDone:
		return style.Success.Render("done")
	case polecat.StateStuck:
		return style.Warning.Render("stuck")
	default:
		return style.Dim.Render(string(state))
	}
}

// formatRefineryState returns a styled string for refinery state.
func formatRefineryState(state refinery.State) string {
	switch state {
	case refinery.StateStopped:
		return style.Dim.Render("stopped")
	case refinery.StateRunning:
		return style.Success.Render("running")
	case refinery.StatePaused:
		return style.Warning.Render("paused")
	default:
		return style.Dim.Render(string(state))
	}
}

// loadWitnessState loads the witness state.json.
func loadWitnessState(rigPath string) *config.AgentState {
	statePath := filepath.Join(rigPath, "witness", "state.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil
	}
	var state config.AgentState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil
	}
	return &state
}

// loadMayorState loads the mayor state.json.
func loadMayorState(rigPath string) *config.AgentState {
	statePath := filepath.Join(rigPath, "mayor", "state.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil
	}
	var state config.AgentState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil
	}
	return &state
}

// formatTimeAgo formats a time as a human-readable "ago" string.
func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day ago"
	}
	return fmt.Sprintf("%d days ago", days)
}

// BeadsSummary contains counts of issues by status.
type BeadsSummary struct {
	Open       int
	InProgress int
	Closed     int
	Blocked    int
}

// getBeadsSummary runs bd stats to get beads summary.
func getBeadsSummary(rigPath string) *BeadsSummary {
	// Check if .beads directory exists
	beadsDir := filepath.Join(rigPath, ".beads")
	if _, err := os.Stat(beadsDir); os.IsNotExist(err) {
		return nil
	}

	// Try running bd stats --json (it may exit with code 1 but still output JSON)
	cmd := exec.Command("bd", "stats", "--json")
	cmd.Dir = rigPath
	output, _ := cmd.CombinedOutput()

	// Parse JSON output (bd stats --json may exit with error but still produce valid JSON)
	var stats struct {
		Open       int `json:"open_issues"`
		InProgress int `json:"in_progress_issues"`
		Closed     int `json:"closed_issues"`
		Blocked    int `json:"blocked_issues"`
	}
	if err := json.Unmarshal(output, &stats); err != nil {
		// JSON parsing failed, try fallback
		return getBeadsSummaryFallback(rigPath)
	}

	return &BeadsSummary{
		Open:       stats.Open,
		InProgress: stats.InProgress,
		Closed:     stats.Closed,
		Blocked:    stats.Blocked,
	}
}

// getBeadsSummaryFallback counts issues by parsing bd list output.
func getBeadsSummaryFallback(rigPath string) *BeadsSummary {
	summary := &BeadsSummary{}

	// Count open issues
	if count := countBeadsIssues(rigPath, "open"); count >= 0 {
		summary.Open = count
	}

	// Count in_progress issues
	if count := countBeadsIssues(rigPath, "in_progress"); count >= 0 {
		summary.InProgress = count
	}

	// Count closed issues
	if count := countBeadsIssues(rigPath, "closed"); count >= 0 {
		summary.Closed = count
	}

	// Count blocked issues
	cmd := exec.Command("bd", "blocked")
	cmd.Dir = rigPath
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		// Filter out empty lines and header
		count := 0
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "Blocked") && !strings.HasPrefix(line, "---") {
				count++
			}
		}
		summary.Blocked = count
	}

	return summary
}

// countBeadsIssues counts issues with a given status.
func countBeadsIssues(rigPath, status string) int {
	cmd := exec.Command("bd", "list", "--status="+status)
	cmd.Dir = rigPath
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	// Count non-empty lines (each line is one issue)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}
