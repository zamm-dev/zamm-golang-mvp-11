package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// createSpecCommand creates the spec management commands
func (a *App) createSpecCommand(jsonOutput, quiet bool) *cobra.Command {
	specCmd := &cobra.Command{
		Use:   "spec",
		Short: "Manage specifications",
		Long:  "Create, read, update, and delete specification nodes.",
	}

	// spec create
	var title, content string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new specification",
		RunE: func(cmd *cobra.Command, args []string) error {
			spec, err := a.specService.CreateSpec(title, content)
			if err != nil {
				return err
			}

			if jsonOutput {
				return a.outputJSON(spec)
			}

			if !quiet {
				fmt.Printf("Created spec: %s\n", spec.ID)
				fmt.Printf("Title: %s\n", spec.Title)
			}
			return nil
		},
	}
	createCmd.Flags().StringVar(&title, "title", "", "Specification title (required)")
	createCmd.Flags().StringVar(&content, "content", "", "Specification content (required)")
	_ = createCmd.MarkFlagRequired("title")
	_ = createCmd.MarkFlagRequired("content")

	// spec list
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all specifications",
		RunE: func(cmd *cobra.Command, args []string) error {
			specs, err := a.specService.ListSpecs()
			if err != nil {
				return err
			}

			if jsonOutput {
				return a.outputJSON(specs)
			}

			return a.outputSpecTable(specs)
		},
	}

	// spec show
	showCmd := &cobra.Command{
		Use:   "show <spec-id>",
		Short: "Show a specification",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spec, err := a.specService.GetSpec(args[0])
			if err != nil {
				return err
			}

			if jsonOutput {
				return a.outputJSON(spec)
			}

			return a.outputSpecDetails(spec)
		},
	}

	// spec update
	updateCmd := &cobra.Command{
		Use:   "update <spec-id>",
		Short: "Update a specification",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spec, err := a.specService.UpdateSpec(args[0], title, content)
			if err != nil {
				return err
			}

			if jsonOutput {
				return a.outputJSON(spec)
			}

			if !quiet {
				fmt.Printf("Updated spec: %s\n", spec.ID)
			}
			return nil
		},
	}
	updateCmd.Flags().StringVar(&title, "title", "", "New specification title")
	updateCmd.Flags().StringVar(&content, "content", "", "New specification content")

	// spec delete
	deleteCmd := &cobra.Command{
		Use:   "delete <spec-id>",
		Short: "Delete a specification",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.specService.DeleteSpec(args[0]); err != nil {
				return err
			}

			if !quiet {
				fmt.Printf("Deleted spec: %s\n", args[0])
			}
			return nil
		},
	}

	specCmd.AddCommand(createCmd, listCmd, showCmd, updateCmd, deleteCmd)
	return specCmd
}
