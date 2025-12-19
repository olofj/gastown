package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/style"
)

// MQ command flags
var (
	mqRejectReason string
	mqRejectNotify bool
)

var mqCmd = &cobra.Command{
	Use:   "mq",
	Short: "Merge queue operations",
	Long: `Manage the merge queue for a rig.

The merge queue tracks work branches from polecats waiting to be merged.
Use these commands to view, submit, and manage merge requests.`,
}

var mqRejectCmd = &cobra.Command{
	Use:   "reject <rig> <mr-id-or-branch>",
	Short: "Reject a merge request",
	Long: `Manually reject a merge request.

This closes the MR with a 'rejected' status without merging.
The source issue is NOT closed (work is not done).

Examples:
  gt mq reject gastown polecat/Nux/gt-xyz --reason "Does not meet requirements"
  gt mq reject gastown mr-Nux-12345 --reason "Superseded by other work" --notify`,
	Args: cobra.ExactArgs(2),
	RunE: runMQReject,
}

func init() {
	// Reject flags
	mqRejectCmd.Flags().StringVarP(&mqRejectReason, "reason", "r", "", "Reason for rejection (required)")
	mqRejectCmd.Flags().BoolVar(&mqRejectNotify, "notify", false, "Send mail notification to worker")
	mqRejectCmd.MarkFlagRequired("reason")

	// Add subcommands
	mqCmd.AddCommand(mqRejectCmd)

	rootCmd.AddCommand(mqCmd)
}

func runMQReject(cmd *cobra.Command, args []string) error {
	rigName := args[0]
	mrIDOrBranch := args[1]

	mgr, _, err := getRefineryManager(rigName)
	if err != nil {
		return err
	}

	result, err := mgr.RejectMR(mrIDOrBranch, mqRejectReason, mqRejectNotify)
	if err != nil {
		return fmt.Errorf("rejecting MR: %w", err)
	}

	fmt.Printf("%s Rejected: %s\n", style.Bold.Render("✗"), result.Branch)
	fmt.Printf("  Worker: %s\n", result.Worker)
	fmt.Printf("  Reason: %s\n", mqRejectReason)

	if result.IssueID != "" {
		fmt.Printf("  Issue:  %s %s\n", result.IssueID, style.Dim.Render("(not closed - work not done)"))
	}

	if mqRejectNotify {
		fmt.Printf("  %s\n", style.Dim.Render("Worker notified via mail"))
	}

	return nil
}
