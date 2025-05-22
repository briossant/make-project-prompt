package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIsTextFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "mpp-test")
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test file
			filePath := filepath.Join(tempDir, "test"+tc.ext)
			err := os.WriteFile(filePath, tc.content, 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Test the isTextFile function
			result := isTextFile(filePath)
			if result != tc.expected {
				t.Errorf("isTextFile(%q) = %v, want %v", filePath, result, tc.expected)
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

	// Test with default include pattern
	files, err := listGitFiles([]string{}, []string{})
	if err != nil {
		t.Fatalf("listGitFiles failed: %v", err)
	}

	// Verify that at least some files were found
	if len(files) == 0 {
		t.Error("listGitFiles returned no files, expected at least some files")
	}

	// Test with a specific include pattern that should match at least one file
	files, err = listGitFiles([]string{"*.go"}, []string{})
	if err != nil {
		t.Fatalf("listGitFiles with include pattern failed: %v", err)
	}

	// Verify that at least one .go file was found
	if len(files) == 0 {
		t.Error("listGitFiles with '*.go' pattern returned no files, expected at least one")
	}

	// Verify that all files have .go extension
	for _, file := range files {
		if filepath.Ext(file) != ".go" {
			t.Errorf("listGitFiles with '*.go' pattern returned a non-go file: %s", file)
		}
	}
}
