package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateDebugLogPath(t *testing.T) {
	path, err := generateDebugLogPath()
	if err != nil {
		t.Fatalf("generateDebugLogPath() failed: %v", err)
	}

	// Check that path contains expected components
	if !strings.Contains(path, ".zamm/logs") {
		t.Errorf("Expected path to contain '.zamm/logs', got: %s", path)
	}

	if !strings.Contains(path, "zamm-debug-") {
		t.Errorf("Expected path to contain 'zamm-debug-', got: %s", path)
	}

	if !strings.HasSuffix(path, ".log") {
		t.Errorf("Expected path to end with '.log', got: %s", path)
	}

	// Check that path is absolute
	if !filepath.IsAbs(path) {
		t.Errorf("Expected absolute path, got: %s", path)
	}
}

func TestCreateDebugLogFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "zamm-debug-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Test creating debug log file
	file, err := createDebugLogFile()
	if err != nil {
		t.Fatalf("createDebugLogFile() failed: %v", err)
	}
	defer file.Close()

	// Verify file was created
	if file == nil {
		t.Fatal("Expected non-nil file handle")
	}

	// Verify file exists
	stat, err := file.Stat()
	if err != nil {
		t.Fatalf("Failed to stat debug log file: %v", err)
	}

	if stat.Size() < 0 {
		t.Errorf("Expected file size >= 0, got: %d", stat.Size())
	}

	// Verify logs directory was created
	logsDir := filepath.Join(tempDir, ".zamm", "logs")
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		t.Errorf("Expected logs directory to be created at: %s", logsDir)
	}
}

func TestCreateDebugLogFilePermissionError(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "zamm-debug-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a read-only directory to simulate permission error
	readOnlyDir := filepath.Join(tempDir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0444); err != nil {
		t.Fatalf("Failed to create read-only dir: %v", err)
	}

	// Override home directory to point to read-only directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", readOnlyDir)
	defer os.Setenv("HOME", originalHome)

	// Test creating debug log file should fail
	file, err := createDebugLogFile()
	if err == nil {
		if file != nil {
			file.Close()
		}
		t.Fatal("Expected createDebugLogFile() to fail with permission error")
	}

	// Verify error message contains expected information
	if !strings.Contains(err.Error(), "failed to create logs directory") {
		t.Errorf("Expected error to mention logs directory creation, got: %v", err)
	}
}
