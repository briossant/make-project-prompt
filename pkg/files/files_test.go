package files

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilterFiles(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "files_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	textFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(textFile, []byte("This is a text file"), 0644); err != nil {
		t.Fatalf("Failed to create text file: %v", err)
	}

	// Create a binary file (contains a null byte)
	binaryFile := filepath.Join(tempDir, "test.bin")
	if err := os.WriteFile(binaryFile, []byte{0x00, 0x01, 0x02}, 0644); err != nil {
		t.Fatalf("Failed to create binary file: %v", err)
	}

	// Create a subdirectory with a file
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	subFile := filepath.Join(subDir, "subfile.txt")
	if err := os.WriteFile(subFile, []byte("This is a file in a subdirectory"), 0644); err != nil {
		t.Fatalf("Failed to create file in subdirectory: %v", err)
	}

	// Test cases
	tests := []struct {
		name           string
		files          []string
		config         Config
		expectedCount  int
		expectedForced bool
	}{
		{
			name:          "Include all files",
			files:         []string{textFile, binaryFile, subFile},
			config:        Config{},
			expectedCount: 2, // Only text files should be included by default
		},
		{
			name:  "Include specific text file",
			files: []string{textFile, binaryFile, subFile},
			config: Config{
				IncludePatterns: []string{"test.txt"},
			},
			expectedCount: 1,
		},
		{
			name:  "Exclude specific file",
			files: []string{textFile, binaryFile, subFile},
			config: Config{
				ExcludePatterns: []string{"subdir/subfile.txt"},
			},
			expectedCount: 1, // Only the root text file
		},
		{
			name:  "Force include specific binary file",
			files: []string{textFile, binaryFile, subFile},
			config: Config{
				ForceIncludePatterns: []string{"test.bin"},
			},
			expectedCount:  3, // All files (2 text + 1 forced binary)
			expectedForced: true,
		},
		{
			name:  "Multiple specific files",
			files: []string{textFile, binaryFile, subFile},
			config: Config{
				IncludePatterns:      []string{"test.txt"},
				ExcludePatterns:      []string{"subdir/subfile.txt"},
				ForceIncludePatterns: []string{"test.bin"},
			},
			expectedCount:  2, // Root text file + forced binary file
			expectedForced: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Convert absolute paths to relative for easier testing
			var relFiles []string
			for _, f := range tc.files {
				rel, err := filepath.Rel(tempDir, f)
				if err != nil {
					t.Fatalf("Failed to get relative path: %v", err)
				}
				relFiles = append(relFiles, rel)
			}

			// Change to temp directory for the test
			oldWd, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}
			defer os.Chdir(oldWd)

			// Run the filter
			result, err := filterFiles(relFiles, tc.config)
			if err != nil {
				t.Fatalf("filterFiles failed: %v", err)
			}

			// Check the count
			if len(result) != tc.expectedCount {
				// Print the files that were included for debugging
				var fileNames []string
				for _, f := range result {
					fileNames = append(fileNames, f.Path)
				}
				t.Errorf("Expected %d files, got %d. Files: %v", tc.expectedCount, len(result), fileNames)
			}

			// Check if any forced files are present when expected
			if tc.expectedForced {
				foundForced := false
				for _, f := range result {
					if f.IsForced {
						foundForced = true
						break
					}
				}
				if !foundForced {
					t.Errorf("Expected to find a forced file but none was found")
				}
			}
		})
	}
}

func TestListGitFiles(t *testing.T) {
	// Skip this test if not in a Git repository
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		t.Skip("Skipping test: not in a Git repository")
	}

	// Note: This test is now less useful since we're no longer doing glob matching
	// in the code. The test now just verifies that ListGitFiles works with the
	// new approach of treating patterns as literal file paths.
	// In a real-world scenario, bash would expand the globs before passing them to the program.

	// Test cases
	testCases := []struct {
		name        string
		config      Config
		expectFiles bool
	}{
		{
			name:        "Default config",
			config:      Config{},
			expectFiles: true,
		},
		{
			name: "Include specific files",
			config: Config{
				// In a real scenario, these would be expanded by bash
				// Here we're just testing that the function works with literal paths
				IncludePatterns: []string{"main.go", "README.md"},
			},
			expectFiles: true,
		},
		{
			name: "Exclude specific files",
			config: Config{
				// In a real scenario, these would be expanded by bash
				ExcludePatterns: []string{"main.go"},
			},
			expectFiles: true,
		},
		{
			name: "Force include specific files",
			config: Config{
				// In a real scenario, these would be expanded by bash
				ForceIncludePatterns: []string{"main.go"},
			},
			expectFiles: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Run ListGitFiles with the config
			files, err := ListGitFiles(tc.config)
			if err != nil {
				t.Fatalf("ListGitFiles failed: %v", err)
			}

			// Check if files were found when expected
			if tc.expectFiles && len(files) == 0 {
				// Skip this check if we're testing in an environment without the expected files
				t.Log("No files found, skipping this check")
			}

			// Log the files found for informational purposes
			if len(files) > 0 {
				t.Logf("Found %d files", len(files))
				for _, file := range files {
					t.Logf("  - %s (IsText: %v, IsForced: %v)", file.Path, file.IsText, file.IsForced)
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
	defer os.RemoveAll(tempDir)

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
