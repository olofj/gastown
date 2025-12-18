package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/config"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/templates"
	"github.com/steveyegge/gastown/internal/workspace"
)

var (
	installForce    bool
	installName     string
	installNoBeads  bool
	installGit      bool
	installGitHub   string
	installPrivate  bool
)

var installCmd = &cobra.Command{
	Use:   "install [path]",
	Short: "Create a new Gas Town harness (workspace)",
	Long: `Create a new Gas Town harness at the specified path.

A harness is the top-level directory where Gas Town is installed. It contains:
  - CLAUDE.md            Mayor role context (Mayor runs from harness root)
  - mayor/               Mayor config, state, and mail
  - rigs/                Managed rig clones (created by 'gt rig add')
  - .beads/redirect      (optional) Default beads location

If path is omitted, uses the current directory.

Examples:
  gt install ~/gt                         # Create harness at ~/gt
  gt install . --name my-workspace        # Initialize current dir
  gt install ~/gt --no-beads              # Skip .beads/redirect setup
  gt install ~/gt --git                   # Also init git with .gitignore
  gt install ~/gt --github=user/repo      # Also create GitHub repo`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInstall,
}

func init() {
	installCmd.Flags().BoolVarP(&installForce, "force", "f", false, "Overwrite existing harness")
	installCmd.Flags().StringVarP(&installName, "name", "n", "", "Town name (defaults to directory name)")
	installCmd.Flags().BoolVar(&installNoBeads, "no-beads", false, "Skip .beads/redirect setup")
	installCmd.Flags().BoolVar(&installGit, "git", false, "Initialize git with .gitignore")
	installCmd.Flags().StringVar(&installGitHub, "github", "", "Create GitHub repo (format: owner/repo)")
	installCmd.Flags().BoolVar(&installPrivate, "private", false, "Make GitHub repo private (use with --github)")
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	// Determine target path
	targetPath := "."
	if len(args) > 0 {
		targetPath = args[0]
	}

	// Expand ~ and resolve to absolute path
	if targetPath[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("getting home directory: %w", err)
		}
		targetPath = filepath.Join(home, targetPath[1:])
	}

	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Determine town name
	townName := installName
	if townName == "" {
		townName = filepath.Base(absPath)
	}

	// Check if already a workspace
	if isWS, _ := workspace.IsWorkspace(absPath); isWS && !installForce {
		return fmt.Errorf("directory is already a Gas Town harness (use --force to reinitialize)")
	}

	// Check if inside an existing workspace
	if existingRoot, _ := workspace.Find(absPath); existingRoot != "" && existingRoot != absPath {
		fmt.Printf("%s Warning: Creating harness inside existing workspace at %s\n",
			style.Dim.Render("⚠"), existingRoot)
	}

	fmt.Printf("%s Creating Gas Town harness at %s\n\n",
		style.Bold.Render("🏭"), style.Dim.Render(absPath))

	// Create directory structure
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Create mayor directory (holds config, state, and mail)
	mayorDir := filepath.Join(absPath, "mayor")
	if err := os.MkdirAll(mayorDir, 0755); err != nil {
		return fmt.Errorf("creating mayor directory: %w", err)
	}
	fmt.Printf("   ✓ Created mayor/\n")

	// Create town.json in mayor/
	townConfig := &config.TownConfig{
		Type:      "town",
		Version:   config.CurrentTownVersion,
		Name:      townName,
		CreatedAt: time.Now(),
	}
	townPath := filepath.Join(mayorDir, "town.json")
	if err := config.SaveTownConfig(townPath, townConfig); err != nil {
		return fmt.Errorf("writing town.json: %w", err)
	}
	fmt.Printf("   ✓ Created mayor/town.json\n")

	// Create rigs.json in mayor/
	rigsConfig := &config.RigsConfig{
		Version: config.CurrentRigsVersion,
		Rigs:    make(map[string]config.RigEntry),
	}
	rigsPath := filepath.Join(mayorDir, "rigs.json")
	if err := config.SaveRigsConfig(rigsPath, rigsConfig); err != nil {
		return fmt.Errorf("writing rigs.json: %w", err)
	}
	fmt.Printf("   ✓ Created mayor/rigs.json\n")

	// Create rigs directory (for managed rig clones)
	rigsDir := filepath.Join(absPath, "rigs")
	if err := os.MkdirAll(rigsDir, 0755); err != nil {
		return fmt.Errorf("creating rigs directory: %w", err)
	}
	fmt.Printf("   ✓ Created rigs/\n")

	// Create mayor mail directory
	mailDir := filepath.Join(mayorDir, "mail")
	if err := os.MkdirAll(mailDir, 0755); err != nil {
		return fmt.Errorf("creating mail directory: %w", err)
	}

	// Create empty inbox
	inboxPath := filepath.Join(mailDir, "inbox.jsonl")
	if err := os.WriteFile(inboxPath, []byte{}, 0644); err != nil {
		return fmt.Errorf("creating inbox: %w", err)
	}
	fmt.Printf("   ✓ Created mayor/mail/inbox.jsonl\n")

	// Create mayor state.json
	mayorState := &config.AgentState{
		Role:       "mayor",
		LastActive: time.Now(),
	}
	statePath := filepath.Join(mayorDir, "state.json")
	if err := config.SaveAgentState(statePath, mayorState); err != nil {
		return fmt.Errorf("writing mayor state: %w", err)
	}
	fmt.Printf("   ✓ Created mayor/state.json\n")

	// Create Mayor CLAUDE.md at harness root (Mayor runs from there)
	if err := createMayorCLAUDEmd(absPath, absPath); err != nil {
		fmt.Printf("   %s Could not create CLAUDE.md: %v\n", style.Dim.Render("⚠"), err)
	} else {
		fmt.Printf("   ✓ Created CLAUDE.md\n")
	}

	// Create .beads directory with redirect (optional)
	if !installNoBeads {
		beadsDir := filepath.Join(absPath, ".beads")
		if err := os.MkdirAll(beadsDir, 0755); err != nil {
			fmt.Printf("   %s Could not create .beads/: %v\n", style.Dim.Render("⚠"), err)
		} else {
			// Create redirect file with placeholder
			redirectPath := filepath.Join(beadsDir, "redirect")
			redirectContent := "# Redirect to your main rig's beads\n# Example: gastown/.beads\n"
			if err := os.WriteFile(redirectPath, []byte(redirectContent), 0644); err != nil {
				fmt.Printf("   %s Could not create redirect: %v\n", style.Dim.Render("⚠"), err)
			} else {
				fmt.Printf("   ✓ Created .beads/redirect (configure for your main rig)\n")
			}
		}
	}

	// Initialize git if requested (--git or --github implies --git)
	if installGit || installGitHub != "" {
		fmt.Println()
		if err := InitGitForHarness(absPath, installGitHub, installPrivate); err != nil {
			return fmt.Errorf("git initialization failed: %w", err)
		}
	}

	fmt.Printf("\n%s Harness created successfully!\n", style.Bold.Render("✓"))
	fmt.Println()
	fmt.Println("Next steps:")
	if !installGit && installGitHub == "" {
		fmt.Printf("  1. Initialize git: %s\n", style.Dim.Render("gt git-init"))
	}
	fmt.Printf("  %s. Add a rig: %s\n", nextStepNum(installGit || installGitHub != ""), style.Dim.Render("gt rig add <name> <git-url>"))
	fmt.Printf("  %s. Configure beads redirect: %s\n", nextStepNum2(installGit || installGitHub != ""), style.Dim.Render("edit .beads/redirect"))
	fmt.Printf("  %s. Start the Mayor: %s\n", nextStepNum3(installGit || installGitHub != ""), style.Dim.Render("cd "+absPath+" && gt prime"))

	return nil
}

func nextStepNum(gitDone bool) string {
	if gitDone {
		return "1"
	}
	return "2"
}

func nextStepNum2(gitDone bool) string {
	if gitDone {
		return "2"
	}
	return "3"
}

func nextStepNum3(gitDone bool) string {
	if gitDone {
		return "3"
	}
	return "4"
}

func createMayorCLAUDEmd(harnessRoot, townRoot string) error {
	tmpl, err := templates.New()
	if err != nil {
		return err
	}

	data := templates.RoleData{
		Role:     "mayor",
		TownRoot: townRoot,
		WorkDir:  harnessRoot,
	}

	content, err := tmpl.RenderRole("mayor", data)
	if err != nil {
		return err
	}

	claudePath := filepath.Join(harnessRoot, "CLAUDE.md")
	return os.WriteFile(claudePath, []byte(content), 0644)
}

func writeJSON(path string, data interface{}) error {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, content, 0644)
}
