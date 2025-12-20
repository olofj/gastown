package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/beads"
	"github.com/steveyegge/gastown/internal/style"
)

// Molecule command flags
var (
	moleculeJSON          bool
	moleculeInstParent    string
	moleculeInstContext   []string
)

var moleculeCmd = &cobra.Command{
	Use:   "molecule",
	Short: "Molecule workflow commands",
	Long: `Manage molecule workflow templates.

Molecules are composable workflow patterns stored as beads issues.
When instantiated on a parent issue, they create child beads forming a DAG.`,
}

var moleculeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List molecules",
	Long: `List all molecule definitions.

Molecules are issues with type=molecule.`,
	RunE: runMoleculeList,
}

var moleculeShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show molecule with parsed steps",
	Long: `Show a molecule definition with its parsed steps.

Displays the molecule's title, description structure, and all defined steps
with their dependencies.`,
	Args: cobra.ExactArgs(1),
	RunE: runMoleculeShow,
}

var moleculeParseCmd = &cobra.Command{
	Use:   "parse <id>",
	Short: "Validate and show parsed structure",
	Long: `Parse and validate a molecule definition.

This command parses the molecule's step definitions and reports any errors.
Useful for debugging molecule definitions before instantiation.`,
	Args: cobra.ExactArgs(1),
	RunE: runMoleculeParse,
}

var moleculeInstantiateCmd = &cobra.Command{
	Use:   "instantiate <mol-id>",
	Short: "Create steps from molecule template",
	Long: `Instantiate a molecule on a parent issue.

Creates child issues for each step defined in the molecule, wiring up
dependencies according to the Needs: declarations.

Template variables ({{variable}}) can be substituted using --context flags.

Examples:
  gt molecule instantiate mol-xyz --parent=gt-abc
  gt molecule instantiate mol-xyz --parent=gt-abc --context feature=auth --context file=login.go`,
	Args: cobra.ExactArgs(1),
	RunE: runMoleculeInstantiate,
}

var moleculeInstancesCmd = &cobra.Command{
	Use:   "instances <mol-id>",
	Short: "Show all instantiations of a molecule",
	Long: `Show all parent issues that have instantiated this molecule.

Lists each instantiation with its status and progress.`,
	Args: cobra.ExactArgs(1),
	RunE: runMoleculeInstances,
}

func init() {
	// List flags
	moleculeListCmd.Flags().BoolVar(&moleculeJSON, "json", false, "Output as JSON")

	// Show flags
	moleculeShowCmd.Flags().BoolVar(&moleculeJSON, "json", false, "Output as JSON")

	// Parse flags
	moleculeParseCmd.Flags().BoolVar(&moleculeJSON, "json", false, "Output as JSON")

	// Instantiate flags
	moleculeInstantiateCmd.Flags().StringVar(&moleculeInstParent, "parent", "", "Parent issue ID (required)")
	moleculeInstantiateCmd.Flags().StringArrayVar(&moleculeInstContext, "context", nil, "Context variable (key=value)")
	moleculeInstantiateCmd.MarkFlagRequired("parent")

	// Instances flags
	moleculeInstancesCmd.Flags().BoolVar(&moleculeJSON, "json", false, "Output as JSON")

	// Add subcommands
	moleculeCmd.AddCommand(moleculeListCmd)
	moleculeCmd.AddCommand(moleculeShowCmd)
	moleculeCmd.AddCommand(moleculeParseCmd)
	moleculeCmd.AddCommand(moleculeInstantiateCmd)
	moleculeCmd.AddCommand(moleculeInstancesCmd)

	rootCmd.AddCommand(moleculeCmd)
}

func runMoleculeList(cmd *cobra.Command, args []string) error {
	workDir, err := findBeadsWorkDir()
	if err != nil {
		return fmt.Errorf("not in a beads workspace: %w", err)
	}

	b := beads.New(workDir)
	issues, err := b.List(beads.ListOptions{
		Type:     "molecule",
		Status:   "all",
		Priority: -1,
	})
	if err != nil {
		return fmt.Errorf("listing molecules: %w", err)
	}

	if moleculeJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(issues)
	}

	// Human-readable output
	fmt.Printf("%s Molecules (%d)\n\n", style.Bold.Render("🧬"), len(issues))

	if len(issues) == 0 {
		fmt.Printf("  %s\n", style.Dim.Render("(no molecules defined)"))
		return nil
	}

	for _, mol := range issues {
		statusMarker := ""
		if mol.Status == "closed" {
			statusMarker = " " + style.Dim.Render("[closed]")
		}

		// Parse steps to show count
		steps, _ := beads.ParseMoleculeSteps(mol.Description)
		stepCount := ""
		if len(steps) > 0 {
			stepCount = fmt.Sprintf(" (%d steps)", len(steps))
		}

		fmt.Printf("  %s: %s%s%s\n", style.Bold.Render(mol.ID), mol.Title, stepCount, statusMarker)
	}

	return nil
}

func runMoleculeShow(cmd *cobra.Command, args []string) error {
	molID := args[0]

	workDir, err := findBeadsWorkDir()
	if err != nil {
		return fmt.Errorf("not in a beads workspace: %w", err)
	}

	b := beads.New(workDir)
	mol, err := b.Show(molID)
	if err != nil {
		return fmt.Errorf("getting molecule: %w", err)
	}

	if mol.Type != "molecule" {
		return fmt.Errorf("%s is not a molecule (type: %s)", molID, mol.Type)
	}

	// Parse steps
	steps, parseErr := beads.ParseMoleculeSteps(mol.Description)

	// For JSON, include parsed steps
	if moleculeJSON {
		type moleculeOutput struct {
			*beads.Issue
			Steps      []beads.MoleculeStep `json:"steps,omitempty"`
			ParseError string               `json:"parse_error,omitempty"`
		}
		out := moleculeOutput{Issue: mol, Steps: steps}
		if parseErr != nil {
			out.ParseError = parseErr.Error()
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	// Human-readable output
	fmt.Printf("\n%s: %s\n", style.Bold.Render(mol.ID), mol.Title)
	fmt.Printf("Type: %s\n", mol.Type)

	if parseErr != nil {
		fmt.Printf("\n%s Parse error: %s\n", style.Bold.Render("⚠"), parseErr)
	}

	// Show steps
	fmt.Printf("\nSteps (%d):\n", len(steps))
	if len(steps) == 0 {
		fmt.Printf("  %s\n", style.Dim.Render("(no steps defined)"))
	} else {
		// Find which steps are ready (no dependencies)
		for _, step := range steps {
			needsStr := ""
			if len(step.Needs) == 0 {
				needsStr = style.Dim.Render("(ready first)")
			} else {
				needsStr = fmt.Sprintf("Needs: %s", strings.Join(step.Needs, ", "))
			}

			tierStr := ""
			if step.Tier != "" {
				tierStr = fmt.Sprintf(" [%s]", step.Tier)
			}

			fmt.Printf("  %-12s → %s%s\n", step.Ref, needsStr, tierStr)
		}
	}

	// Count instances
	instances, _ := findMoleculeInstances(b, molID)
	fmt.Printf("\nInstances: %d\n", len(instances))

	return nil
}

func runMoleculeParse(cmd *cobra.Command, args []string) error {
	molID := args[0]

	workDir, err := findBeadsWorkDir()
	if err != nil {
		return fmt.Errorf("not in a beads workspace: %w", err)
	}

	b := beads.New(workDir)
	mol, err := b.Show(molID)
	if err != nil {
		return fmt.Errorf("getting molecule: %w", err)
	}

	// Validate the molecule
	validationErr := beads.ValidateMolecule(mol)

	// Parse steps regardless of validation
	steps, parseErr := beads.ParseMoleculeSteps(mol.Description)

	if moleculeJSON {
		type parseOutput struct {
			Valid           bool                 `json:"valid"`
			ValidationError string               `json:"validation_error,omitempty"`
			ParseError      string               `json:"parse_error,omitempty"`
			Steps           []beads.MoleculeStep `json:"steps"`
		}
		out := parseOutput{
			Valid: validationErr == nil,
			Steps: steps,
		}
		if validationErr != nil {
			out.ValidationError = validationErr.Error()
		}
		if parseErr != nil {
			out.ParseError = parseErr.Error()
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	// Human-readable output
	fmt.Printf("\n%s: %s\n\n", style.Bold.Render(mol.ID), mol.Title)

	if validationErr != nil {
		fmt.Printf("%s Validation failed: %s\n\n", style.Bold.Render("✗"), validationErr)
	} else {
		fmt.Printf("%s Valid molecule\n\n", style.Bold.Render("✓"))
	}

	if parseErr != nil {
		fmt.Printf("Parse error: %s\n\n", parseErr)
	}

	fmt.Printf("Parsed Steps (%d):\n", len(steps))
	for i, step := range steps {
		fmt.Printf("\n  [%d] %s\n", i+1, style.Bold.Render(step.Ref))
		if step.Title != step.Ref {
			fmt.Printf("      Title: %s\n", step.Title)
		}
		if len(step.Needs) > 0 {
			fmt.Printf("      Needs: %s\n", strings.Join(step.Needs, ", "))
		}
		if step.Tier != "" {
			fmt.Printf("      Tier: %s\n", step.Tier)
		}
		if step.Instructions != "" {
			// Show first line of instructions
			firstLine := strings.SplitN(step.Instructions, "\n", 2)[0]
			if len(firstLine) > 60 {
				firstLine = firstLine[:57] + "..."
			}
			fmt.Printf("      Instructions: %s\n", style.Dim.Render(firstLine))
		}
	}

	return nil
}

func runMoleculeInstantiate(cmd *cobra.Command, args []string) error {
	molID := args[0]

	workDir, err := findBeadsWorkDir()
	if err != nil {
		return fmt.Errorf("not in a beads workspace: %w", err)
	}

	b := beads.New(workDir)

	// Get the molecule
	mol, err := b.Show(molID)
	if err != nil {
		return fmt.Errorf("getting molecule: %w", err)
	}

	if mol.Type != "molecule" {
		return fmt.Errorf("%s is not a molecule (type: %s)", molID, mol.Type)
	}

	// Validate molecule
	if err := beads.ValidateMolecule(mol); err != nil {
		return fmt.Errorf("invalid molecule: %w", err)
	}

	// Get the parent issue
	parent, err := b.Show(moleculeInstParent)
	if err != nil {
		return fmt.Errorf("getting parent issue: %w", err)
	}

	// Parse context variables
	ctx := make(map[string]string)
	for _, kv := range moleculeInstContext {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid context format %q (expected key=value)", kv)
		}
		ctx[parts[0]] = parts[1]
	}

	// Instantiate the molecule
	opts := beads.InstantiateOptions{Context: ctx}
	steps, err := b.InstantiateMolecule(mol, parent, opts)
	if err != nil {
		return fmt.Errorf("instantiating molecule: %w", err)
	}

	fmt.Printf("%s Created %d steps from %s on %s\n\n",
		style.Bold.Render("✓"), len(steps), molID, moleculeInstParent)

	for _, step := range steps {
		fmt.Printf("  %s: %s\n", style.Dim.Render(step.ID), step.Title)
	}

	return nil
}

func runMoleculeInstances(cmd *cobra.Command, args []string) error {
	molID := args[0]

	workDir, err := findBeadsWorkDir()
	if err != nil {
		return fmt.Errorf("not in a beads workspace: %w", err)
	}

	b := beads.New(workDir)

	// Verify the molecule exists
	mol, err := b.Show(molID)
	if err != nil {
		return fmt.Errorf("getting molecule: %w", err)
	}

	if mol.Type != "molecule" {
		return fmt.Errorf("%s is not a molecule (type: %s)", molID, mol.Type)
	}

	// Find all instances
	instances, err := findMoleculeInstances(b, molID)
	if err != nil {
		return fmt.Errorf("finding instances: %w", err)
	}

	if moleculeJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(instances)
	}

	// Human-readable output
	fmt.Printf("\n%s Instances of %s (%d)\n\n",
		style.Bold.Render("📋"), molID, len(instances))

	if len(instances) == 0 {
		fmt.Printf("  %s\n", style.Dim.Render("(no instantiations found)"))
		return nil
	}

	fmt.Printf("%-16s %-12s %s\n",
		style.Bold.Render("Parent"),
		style.Bold.Render("Status"),
		style.Bold.Render("Created"))
	fmt.Println(strings.Repeat("-", 50))

	for _, inst := range instances {
		// Calculate progress from children
		progress := ""
		if len(inst.Children) > 0 {
			closed := 0
			for _, childID := range inst.Children {
				child, err := b.Show(childID)
				if err == nil && child.Status == "closed" {
					closed++
				}
			}
			progress = fmt.Sprintf(" (%d/%d complete)", closed, len(inst.Children))
		}

		statusStr := inst.Status
		if inst.Status == "closed" {
			statusStr = style.Dim.Render("done")
		} else if inst.Status == "in_progress" {
			statusStr = "active"
		}

		created := ""
		if inst.CreatedAt != "" {
			// Parse and format date
			created = inst.CreatedAt[:10] // Just the date portion
		}

		fmt.Printf("%-16s %-12s %s%s\n", inst.ID, statusStr, created, progress)
	}

	return nil
}

// moleculeInstance represents an instantiation of a molecule.
type moleculeInstance struct {
	*beads.Issue
}

// findMoleculeInstances finds all parent issues that have steps instantiated from the given molecule.
func findMoleculeInstances(b *beads.Beads, molID string) ([]*beads.Issue, error) {
	// Get all issues and look for ones with children that have instantiated_from metadata
	// This is a brute-force approach - could be optimized with better queries

	// Strategy: search for issues whose descriptions contain "instantiated_from: <molID>"
	allIssues, err := b.List(beads.ListOptions{Status: "all", Priority: -1})
	if err != nil {
		return nil, err
	}

	// Find issues that reference this molecule
	parentIDs := make(map[string]bool)
	for _, issue := range allIssues {
		if strings.Contains(issue.Description, fmt.Sprintf("instantiated_from: %s", molID)) {
			// This is a step - find its parent
			if issue.Parent != "" {
				parentIDs[issue.Parent] = true
			}
		}
	}

	// Fetch the parent issues
	var parents []*beads.Issue
	for parentID := range parentIDs {
		parent, err := b.Show(parentID)
		if err == nil {
			parents = append(parents, parent)
		}
	}

	return parents, nil
}
