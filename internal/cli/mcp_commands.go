package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/mcp"
)

func (a *App) createMCPCommand() *cobra.Command {
	var transport string
	var address string

	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server",
		Long:  "Start a Model Context Protocol server that provides tools for creating child specifications.",
		RunE: func(cmd *cobra.Command, args []string) error {
			server := mcp.NewServer(a.specService)

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

			errChan := make(chan error, 1)
			go func() {
				errChan <- server.Start(transport, address)
			}()

			select {
			case err := <-errChan:
				if err != nil {
					return fmt.Errorf("MCP server error: %w", err)
				}
			case sig := <-sigChan:
				fmt.Printf("\nReceived signal %v, shutting down MCP server...\n", sig)
				if err := server.Stop(); err != nil {
					return fmt.Errorf("error stopping MCP server: %w", err)
				}
			}

			return nil
		},
	}

	mcpCmd.Flags().StringVar(&transport, "transport", "stdio", "Transport type (stdio or http)")
	mcpCmd.Flags().StringVar(&address, "address", ":8080", "Address to bind HTTP server (only used with http transport)")

	return mcpCmd
}
