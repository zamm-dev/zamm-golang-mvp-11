package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// createLinkCommand creates the link management commands
func (a *App) createLinkCommand(jsonOutput, quiet bool) *cobra.Command {
	linkCmd := &cobra.Command{
		Use:   "link",
		Short: "Manage spec-commit links",
		Long:  "Create and manage links between specifications and Git commits.",
	}

	// link create
	var specID, commitID, repoPath, linkType string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a link between spec and commit",
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoPath == "" {
				repoPath = a.config.Git.DefaultRepo
			}
			if linkType == "" {
				linkType = "implements"
			}

			link, err := a.linkService.LinkSpecToCommit(specID, commitID, repoPath, linkType)
			if err != nil {
				return err
			}

			if jsonOutput {
				return a.outputJSON(link)
			}

			if !quiet {
				fmt.Printf("Created link: %s\n", link.ID)
			}
			return nil
		},
	}
	createCmd.Flags().StringVar(&specID, "spec", "", "Specification ID (required)")
	createCmd.Flags().StringVar(&commitID, "commit", "", "Commit hash (required)")
	createCmd.Flags().StringVar(&repoPath, "repo", "", "Repository path (default: current directory)")
	createCmd.Flags().StringVar(&linkType, "type", "implements", "Link type (implements or references)")
	createCmd.MarkFlagRequired("spec")
	createCmd.MarkFlagRequired("commit")

	// link list-by-spec
	listBySpecCmd := &cobra.Command{
		Use:   "list-by-spec <spec-id>",
		Short: "List commits linked to a specification",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			links, err := a.linkService.GetCommitsForSpec(args[0])
			if err != nil {
				return err
			}

			if jsonOutput {
				return a.outputJSON(links)
			}

			return a.outputLinkTable(links)
		},
	}

	// link list-by-commit
	listByCommitCmd := &cobra.Command{
		Use:   "list-by-commit <commit-hash>",
		Short: "List specs linked to a commit",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoPath == "" {
				repoPath = a.config.Git.DefaultRepo
			}

			specs, err := a.linkService.GetSpecsForCommit(args[0], repoPath)
			if err != nil {
				return err
			}

			if jsonOutput {
				return a.outputJSON(specs)
			}

			return a.outputSpecTable(specs)
		},
	}
	listByCommitCmd.Flags().StringVar(&repoPath, "repo", "", "Repository path (default: current directory)")

	// link delete
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a link between spec and commit",
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoPath == "" {
				repoPath = a.config.Git.DefaultRepo
			}

			if err := a.linkService.UnlinkSpecFromCommit(specID, commitID, repoPath); err != nil {
				return err
			}

			if !quiet {
				fmt.Printf("Deleted link between spec %s and commit %s\n", specID, commitID)
			}
			return nil
		},
	}
	deleteCmd.Flags().StringVar(&specID, "spec", "", "Specification ID (required)")
	deleteCmd.Flags().StringVar(&commitID, "commit", "", "Commit hash (required)")
	deleteCmd.Flags().StringVar(&repoPath, "repo", "", "Repository path (default: current directory)")
	deleteCmd.MarkFlagRequired("spec")
	deleteCmd.MarkFlagRequired("commit")

	linkCmd.AddCommand(createCmd, listBySpecCmd, listByCommitCmd, deleteCmd)
	return linkCmd
}
