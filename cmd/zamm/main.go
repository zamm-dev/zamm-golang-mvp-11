package main

import (
	"fmt"
	"os"

	"github.com/yourorg/zamm-mvp/internal/cli"
	"github.com/yourorg/zamm-mvp/internal/models"
)

func main() {
	app, err := cli.NewApp()
	if err != nil {
		handleError(err)
		os.Exit(2)
	}
	defer app.Close()

	rootCmd := app.CreateRootCommand()
	if err := rootCmd.Execute(); err != nil {
		handleError(err)
		os.Exit(getExitCode(err))
	}
}

// handleError prints error messages in a user-friendly format
func handleError(err error) {
	if zammErr, ok := err.(*models.ZammError); ok {
		fmt.Fprintf(os.Stderr, "Error: %s\n", zammErr.Message)
		if zammErr.Details != "" {
			fmt.Fprintf(os.Stderr, "Details: %s\n", zammErr.Details)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
	}
}

// getExitCode returns appropriate exit code based on error type
func getExitCode(err error) int {
	if zammErr, ok := err.(*models.ZammError); ok {
		switch zammErr.Type {
		case models.ErrTypeValidation, models.ErrTypeNotFound, models.ErrTypeConflict, models.ErrTypeGit:
			return 1 // User error
		case models.ErrTypeStorage, models.ErrTypeSystem:
			return 2 // System error
		default:
			return 2
		}
	}
	return 2 // Default to system error
}
