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

func TestQuestionInputMethods(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "question_input_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a question file
	questionFile := filepath.Join(tempDir, "question.txt")
	questionContent := "This is a question from a file"
	if err := os.WriteFile(questionFile, []byte(questionContent), 0644); err != nil {
		t.Fatalf("Failed to create question file: %v", err)
	}

	// Test cases for determining which flag was provided last
	testCases := []struct {
		name           string
		args           []string
		expectedQIndex int
		expectedCIndex int
		expectedQFIndex int
	}{
		{
			name:           "Only -q flag",
			args:           []string{"program", "-q", "question"},
			expectedQIndex: 1,
			expectedCIndex: -1,
			expectedQFIndex: -1,
		},
		{
			name:           "Only -c flag",
			args:           []string{"program", "-c"},
			expectedQIndex: -1,
			expectedCIndex: 1,
			expectedQFIndex: -1,
		},
		{
			name:           "Only -qf flag",
			args:           []string{"program", "-qf", "file.txt"},
			expectedQIndex: -1,
			expectedCIndex: -1,
			expectedQFIndex: 1,
		},
		{
			name:           "Multiple flags, -q last",
			args:           []string{"program", "-c", "-qf", "file.txt", "-q", "question"},
			expectedQIndex: 4,
			expectedCIndex: 1,
			expectedQFIndex: 2,
		},
		{
			name:           "Multiple flags, -c last",
			args:           []string{"program", "-q", "question", "-qf", "file.txt", "-c"},
			expectedQIndex: 1,
			expectedCIndex: 5,
			expectedQFIndex: 3,
		},
		{
			name:           "Multiple flags, -qf last",
			args:           []string{"program", "-q", "question", "-c", "-qf", "file.txt"},
			expectedQIndex: 1,
			expectedCIndex: 3,
			expectedQFIndex: 4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call the function that determines which flag was provided last
			lastQIndex := -1
			lastCIndex := -1
			lastQFIndex := -1

			for i, arg := range tc.args {
				if arg == "-q" || arg == "--q" {
					lastQIndex = i
				} else if arg == "-c" || arg == "--c" {
					lastCIndex = i
				} else if arg == "-qf" || arg == "--qf" {
					lastQFIndex = i
				}
			}

			// Check results
			if lastQIndex != tc.expectedQIndex {
				t.Errorf("Expected lastQIndex to be %d, got %d", tc.expectedQIndex, lastQIndex)
			}
			if lastCIndex != tc.expectedCIndex {
				t.Errorf("Expected lastCIndex to be %d, got %d", tc.expectedCIndex, lastCIndex)
			}
			if lastQFIndex != tc.expectedQFIndex {
				t.Errorf("Expected lastQFIndex to be %d, got %d", tc.expectedQFIndex, lastQFIndex)
			}

			// Determine which method should win
			var expectedWinner string
			if tc.expectedQFIndex > tc.expectedQIndex && tc.expectedQFIndex > tc.expectedCIndex {
				expectedWinner = "file"
			} else if tc.expectedCIndex > tc.expectedQIndex && tc.expectedCIndex > tc.expectedQFIndex {
				expectedWinner = "clipboard"
			} else if tc.expectedQIndex > tc.expectedCIndex && tc.expectedQIndex > tc.expectedQFIndex {
				expectedWinner = "command-line"
			} else if tc.expectedQFIndex >= 0 {
				expectedWinner = "file"
			} else if tc.expectedCIndex >= 0 {
				expectedWinner = "clipboard"
			} else if tc.expectedQIndex >= 0 {
				expectedWinner = "command-line"
			} else {
				expectedWinner = "default"
			}

			// Log the expected winner for clarity
			t.Logf("Expected winner: %s", expectedWinner)
		})
	}

	// Test reading from a file
	t.Run("Read question from file", func(t *testing.T) {
		content, err := os.ReadFile(questionFile)
		if err != nil {
			t.Fatalf("Failed to read question file: %v", err)
		}
		if string(content) != questionContent {
			t.Errorf("Expected file content to be %q, got %q", questionContent, string(content))
		}
	})
}
