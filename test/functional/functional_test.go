package functional

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestMain sets up the test environment and runs all tests
func TestMain(m *testing.M) {
	// Run tests
	exitCode := m.Run()

	// Exit with the same code
	os.Exit(exitCode)
}

// setupTestRepo creates a test Git repository and returns its path
func setupTestRepo(t *testing.T) string {
	t.Helper()

	// Get the path to the setup script relative to this file
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	scriptPath := filepath.Join(dir, "setup_test_repo.sh")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Fatalf("Setup script not found at %s: %v", scriptPath, err)
	}

	// Run the setup script
	cmd := exec.Command(scriptPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to run setup script: %v\nStdout: %s\nStderr: %s", err, stdout.String(), stderr.String())
	}

	// Get the repository path from the last line of stdout
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	repoPath := lines[len(lines)-1]
	if repoPath == "" {
		t.Fatalf("Failed to get repository path from setup script output")
	}

	t.Logf("Test repository created at: %s", repoPath)
	return repoPath
}

// cleanupTestRepo removes the test repository
func cleanupTestRepo(t *testing.T, repoPath string) {
	t.Helper()
	if repoPath != "" && strings.HasPrefix(repoPath, "/tmp/") {
		err := os.RemoveAll(repoPath)
		if err != nil {
			t.Logf("Warning: Failed to remove test repository: %v", err)
		}
	}
}

// runMPP runs the make-project-prompt command with the given arguments
func runMPP(t *testing.T, repoPath string, args ...string) (string, string, error) {
	t.Helper()

	// Build the command
	cmd := exec.Command("go", append([]string{"run", "cmd/make-project-prompt/main.go"}, args...)...)
	cmd.Dir = repoPath

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()

	return stdout.String(), stderr.String(), err
}

// TestBasicFunctionality tests the basic functionality of make-project-prompt
func TestBasicFunctionality(t *testing.T) {
	// Setup test repository
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	// Test with default options
	stdout, stderr, err := runMPP(t, repoPath, "-q", "Test question")
	if err != nil {
		t.Fatalf("Failed to run make-project-prompt: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
	}

	// Check that the output contains expected information
	if !strings.Contains(stdout, "Prompt generated and copied to clipboard!") {
		t.Errorf("Expected output to contain 'Prompt generated and copied to clipboard!', got: %s", stdout)
	}

	// We can't easily check the clipboard content, but we can check that the command ran successfully
	t.Logf("make-project-prompt ran successfully with default options")
}

// TestIncludePatterns tests the -i flag
func TestIncludePatterns(t *testing.T) {
	// Setup test repository
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	// Test with include patterns
	stdout, stderr, err := runMPP(t, repoPath, "-i", "src/main/*.go", "-q", "Test include patterns")
	if err != nil {
		t.Fatalf("Failed to run make-project-prompt: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
	}

	// Check that the output contains expected information
	if !strings.Contains(stdout, "Inclusion patterns: [src/main/*.go]") {
		t.Errorf("Expected output to contain 'Inclusion patterns: [src/main/*.go]', got: %s", stdout)
	}

	t.Logf("make-project-prompt ran successfully with include patterns")
}

// TestExcludePatterns tests the -e flag
func TestExcludePatterns(t *testing.T) {
	// Setup test repository
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	// Test with exclude patterns
	stdout, stderr, err := runMPP(t, repoPath, "-e", "src/test/*", "-q", "Test exclude patterns")
	if err != nil {
		t.Fatalf("Failed to run make-project-prompt: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
	}

	// Check that the output contains expected information
	if !strings.Contains(stdout, "Exclusion patterns: [src/test/*]") {
		t.Errorf("Expected output to contain 'Exclusion patterns: [src/test/*]', got: %s", stdout)
	}

	t.Logf("make-project-prompt ran successfully with exclude patterns")
}

// TestForceIncludePatterns tests the -f flag
func TestForceIncludePatterns(t *testing.T) {
	// Setup test repository
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	// Test with force include patterns
	stdout, stderr, err := runMPP(t, repoPath, "-f", "binary_file.bin", "-q", "Test force include patterns")
	if err != nil {
		t.Fatalf("Failed to run make-project-prompt: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
	}

	// Check that the output contains expected information
	if !strings.Contains(stdout, "Force inclusion patterns: [binary_file.bin]") {
		t.Errorf("Expected output to contain 'Force inclusion patterns: [binary_file.bin]', got: %s", stdout)
	}

	t.Logf("make-project-prompt ran successfully with force include patterns")
}

// TestQuestionFromFile tests the -qf flag
func TestQuestionFromFile(t *testing.T) {
	// Setup test repository
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	// Create a question file
	questionFile := filepath.Join(repoPath, "question.txt")
	err := os.WriteFile(questionFile, []byte("Question from file"), 0644)
	if err != nil {
		t.Fatalf("Failed to create question file: %v", err)
	}

	// Test with question from file
	stdout, stderr, err := runMPP(t, repoPath, "-qf", questionFile)
	if err != nil {
		t.Fatalf("Failed to run make-project-prompt: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
	}

	// Check that the output contains expected information
	if !strings.Contains(stdout, fmt.Sprintf("Using question from file %s", questionFile)) {
		t.Errorf("Expected output to contain 'Using question from file %s', got: %s", questionFile, stdout)
	}

	t.Logf("make-project-prompt ran successfully with question from file")
}

// TestCombinedOptions tests combining multiple options
func TestCombinedOptions(t *testing.T) {
	// Setup test repository
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	// Test with combined options
	stdout, stderr, err := runMPP(t, repoPath, "-i", "src/main/*.go", "-e", "src/main/utils.go", "-f", "binary_file.bin", "-q", "Test combined options")
	if err != nil {
		t.Fatalf("Failed to run make-project-prompt: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
	}

	// Check that the output contains expected information
	if !strings.Contains(stdout, "Inclusion patterns: [src/main/*.go]") {
		t.Errorf("Expected output to contain 'Inclusion patterns: [src/main/*.go]', got: %s", stdout)
	}
	if !strings.Contains(stdout, "Exclusion patterns: [src/main/utils.go]") {
		t.Errorf("Expected output to contain 'Exclusion patterns: [src/main/utils.go]', got: %s", stdout)
	}
	if !strings.Contains(stdout, "Force inclusion patterns: [binary_file.bin]") {
		t.Errorf("Expected output to contain 'Force inclusion patterns: [binary_file.bin]', got: %s", stdout)
	}

	t.Logf("make-project-prompt ran successfully with combined options")
}
