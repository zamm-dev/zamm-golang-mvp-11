package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// createDebugLogFile creates a debug log file in ~/.zamm/logs directory
// Returns the file handle and any error encountered
func createDebugLogFile() (*os.File, error) {
	logPath, err := generateDebugLogPath()
	if err != nil {
		return nil, fmt.Errorf("failed to generate debug log path: %w", err)
	}

	// Ensure the logs directory exists
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory %s: %w", logDir, err)
	}

	// Create the debug log file
	file, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create debug log file %s: %w", logPath, err)
	}

	return file, nil
}

// generateDebugLogPath generates the path for a debug log file
// Format: ~/.zamm/logs/zamm-debug-YYYY-MM-DD-HH-MM-SS.log
func generateDebugLogPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Generate timestamp for unique filename
	timestamp := time.Now().Format("2006-01-02-15-04-05")
	filename := fmt.Sprintf("zamm-debug-%s.log", timestamp)

	logPath := filepath.Join(homeDir, ".zamm", "logs", filename)
	return logPath, nil
}
