package files

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Helper function to set up a test repo for this package's tests.
// It's good practice to keep helpers close to the tests that use them.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	// Assumes test is run from project root, or CI environment is set up correctly.
	// We need to find the script relative to the current file.
	wd, _ := os.Getwd() // e.g., /path/to/project/pkg/files
	scriptPath := filepath.Join(wd, "..", "..", "test", "functional", "setup_test_repo.sh")

	cmd := exec.Command("bash", scriptPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run setup_test_repo.sh: %v\nOutput: %s", err, string(output))
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	repoPath := lines[len(lines)-1]
	t.Logf("Test repository created at: %s", repoPath)
	return repoPath
}

func TestListGitFiles_Hermetic(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer func() {
		if err := os.RemoveAll(repoPath); err != nil {
			t.Logf("Warning: Failed to remove test repo: %v", err)
		}
	}()

	// Change working directory to the test repo for the duration of the test
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	if err := os.Chdir(repoPath); err != nil {
		t.Fatalf("Failed to change directory to test repo: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Logf("Warning: Failed to change back to original directory: %v", err)
		}
	}() // Change back when done

	testCases := []struct {
		name                string
		config              Config
		expectedFiles       map[string]bool // Use a map for easy lookup
		expectedForcedFiles map[string]bool
	}{
		{
			name:   "Default config lists all tracked text files",
			config: Config{},
			expectedFiles: map[string]bool{
				".gitignore":           true,
				"docs/CONTRIBUTING.md": true,
				"docs/README.md":       true,
				"large_important.txt":  true,
				"src/main/app.go":      true,
				"src/main/utils.go":    true,
				"src/test/app_test.go": true,
			},
		},
		{
			name: "Include only main go files",
			config: Config{
				// Note: These are not globs, they are literal paths because the
				// shell would have expanded them.
				IncludePatterns: []string{"src/main/app.go", "src/main/utils.go"},
			},
			expectedFiles: map[string]bool{
				"src/main/app.go":   true,
				"src/main/utils.go": true,
			},
		},
		{
			name: "Exclude test files",
			config: Config{
				ExcludePatterns: []string{"src/test/app_test.go"},
			},
			expectedFiles: map[string]bool{
				".gitignore":           true,
				"docs/CONTRIBUTING.md": true,
				"docs/README.md":       true,
				"large_important.txt":  true,
				"src/main/app.go":      true,
				"src/main/utils.go":    true,
			},
		},
		{
			name: "Force include an ignored binary file",
			config: Config{
				ForceIncludePatterns: []string{"binary_file.bin"},
			},
			// Only the forced file is returned
			expectedFiles: map[string]bool{
				"binary_file.bin": true, // The forced file
			},
			expectedForcedFiles: map[string]bool{"binary_file.bin": true},
		},
		{
			name: "Force include markdown files",
			config: Config{
				ForceIncludePatterns: []string{"docs/README.md", "docs/CONTRIBUTING.md"},
			},
			// Only the forced files should be returned
			expectedFiles: map[string]bool{
				"docs/README.md":       true,
				"docs/CONTRIBUTING.md": true,
			},
			expectedForcedFiles: map[string]bool{
				"docs/README.md":       true,
				"docs/CONTRIBUTING.md": true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			infos, err := ListGitFiles(tc.config)
			if err != nil {
				t.Fatalf("ListGitFiles failed: %v", err)
			}

			if len(infos) != len(tc.expectedFiles) {
				t.Errorf("Expected %d files, but got %d", len(tc.expectedFiles), len(infos))
				var foundFiles []string
				for _, info := range infos {
					foundFiles = append(foundFiles, info.Path)
				}
				t.Logf("Found files: %v", foundFiles)
			}

			for _, info := range infos {
				if _, ok := tc.expectedFiles[info.Path]; !ok {
					t.Errorf("Got unexpected file in result: %s", info.Path)
				}

				// Check if the forced status is correct
				isForced := tc.expectedForcedFiles != nil && tc.expectedForcedFiles[info.Path]
				if info.IsForced != isForced {
					t.Errorf("File %s: expected IsForced=%v, got %v", info.Path, isForced, info.IsForced)
				}
			}
		})
	}
}

func TestGetProjectTree(t *testing.T) {
	// Skip this test if the tree command is not available
	_, err := exec.LookPath("tree")
	if err != nil {
		t.Skip("Skipping test: tree command not available")
	}

	// Get the project tree
	tree, err := GetProjectTree()
	if err != nil {
		t.Fatalf("GetProjectTree failed: %v", err)
	}

	// Verify that the tree is not empty
	if len(tree) == 0 {
		t.Error("Expected non-empty project tree, but got empty string")
	}

	// Verify that the tree contains some expected elements
	expectedElements := []string{
		".",
		"├──",
		"└──",
	}

	for _, element := range expectedElements {
		if !strings.Contains(tree, element) {
			t.Errorf("Expected project tree to contain %q, but it doesn't", element)
		}
	}
}

func TestIsTextFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "istext_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Warning: Failed to remove temp directory: %v", err)
		}
	}()

	// Test cases
	testCases := []struct {
		name     string
		content  []byte
		ext      string
		expected bool
	}{
		{
			name:     "Text file with .txt extension",
			content:  []byte("This is a text file"),
			ext:      ".txt",
			expected: true,
		},
		{
			name:     "Go source file",
			content:  []byte("package main\n\nfunc main() {}\n"),
			ext:      ".go",
			expected: true,
		},
		{
			name:     "Binary file",
			content:  []byte{0, 1, 2, 3, 0, 5, 6},
			ext:      ".bin",
			expected: false,
		},
		{
			name:     "Text file with unknown extension",
			content:  []byte("This is a text file with unknown extension"),
			ext:      ".unknown",
			expected: true,
		},
		{
			name:     "Go module file",
			content:  []byte("module example.com/mymodule\n\ngo 1.21\n"),
			ext:      ".mod",
			expected: true,
			// This test will create a file named "test.mod", but IsTextFile has a special case for "go.mod"
			// We'll handle this in the test function
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test file
			var filePath string
			if tc.name == "Go module file" {
				// Special case for Go module file
				filePath = filepath.Join(tempDir, "go.mod")
			} else {
				filePath = filepath.Join(tempDir, "test"+tc.ext)
			}

			err := os.WriteFile(filePath, tc.content, 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Test the IsTextFile function
			result := IsTextFile(filePath)
			if result != tc.expected {
				t.Errorf("IsTextFile(%q) = %v, want %v", filePath, result, tc.expected)
			}
		})
	}
}
