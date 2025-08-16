package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (a *App) createOrganizeCommand(jsonOutput, quiet bool) *cobra.Command {
	return &cobra.Command{
		Use:   "organize [node-id]",
		Short: "Organize nodes into hierarchical file structure",
		Long: `Move nodes from generic .zamm/nodes/<UUID>.md locations to hierarchical paths 
based on their parent-child relationships. Uses slug metadata for consistent path computation.

Root nodes are placed at docs/index.md, and child nodes are organized under their 
parent's slug as either folders (for nodes with children) or files (for leaf nodes).

If a node ID is provided, only that specific node will be organized.
If no node ID is provided, all nodes starting from the root will be organized.

The command will:
1. Generate slugs for nodes that don't have them (based on titles)
2. Compute hierarchical paths using parent-child relationships
3. Move files to new locations
4. Update node-files.csv to track new paths`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var nodeID string
			if len(args) > 0 {
				nodeID = args[0]
			}

			if err := a.specService.OrganizeNodes(nodeID); err != nil {
				return fmt.Errorf("failed to organize nodes: %w", err)
			}

			if !quiet {
				if nodeID != "" {
					fmt.Printf("Successfully organized node %s into hierarchical structure\n", nodeID)
				} else {
					fmt.Println("Successfully organized nodes into hierarchical structure")
				}
				fmt.Println("Updated node-files.csv with new file paths")
			}

			if jsonOutput {
				result := map[string]interface{}{
					"success": true,
					"message": "Nodes organized successfully",
				}
				if nodeID != "" {
					result["node_id"] = nodeID
				}
				return a.outputJSON(result)
			}

			return nil
		},
	}
}
