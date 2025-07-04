package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/briossant/make-project-prompt/pkg/files"
)

func TestGenerator_Generate(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "prompt_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	textFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(textFile, []byte("This is a text file"), 0644); err != nil {
		t.Fatalf("Failed to create text file: %v", err)
	}

	goFile := filepath.Join(tempDir, "test.go")
	if err := os.WriteFile(goFile, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("Failed to create Go file: %v", err)
	}

	// Create a large file that exceeds the default max size
	largeFile := filepath.Join(tempDir, "large.txt")
	largeContent := strings.Repeat("Large file content\n", 100000) // More than 1MB
	if err := os.WriteFile(largeFile, []byte(largeContent), 0644); err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	// Create file info objects
	fileInfos := []files.FileInfo{
		{
			Path:      textFile,
			IsText:    true,
			IsForced:  false,
			Size:      int64(len("This is a text file")),
			IsRegular: true,
		},
		{
			Path:      goFile,
			IsText:    true,
			IsForced:  false,
			Size:      int64(len("package main\n\nfunc main() {}\n")),
			IsRegular: true,
		},
		{
			Path:      largeFile,
			IsText:    true,
			IsForced:  false,
			Size:      int64(len(largeContent)),
			IsRegular: true,
		},
		{
			Path:      largeFile + ".forced",
			IsText:    true,
			IsForced:  true, // Force include this large file
			Size:      int64(len(largeContent)),
			IsRegular: true,
		},
	}

	// Test cases
	testCases := []struct {
		name           string
		question       string
		maxFileSize    int64
		expectedFiles  int
		expectedPhrase string
	}{
		{
			name:           "Default max file size",
			question:       "Test question",
			maxFileSize:    0, // Use default
			expectedFiles:  2, // Only the two small files
			expectedPhrase: "Test question",
		},
		{
			name:           "Custom max file size",
			question:       "Another question",
			maxFileSize:    int64(len(largeContent) + 1),
			expectedFiles:  3, // All three files (including large file)
			expectedPhrase: "Another question",
		},
		{
			name:           "Force included file",
			question:       "Force include question",
			maxFileSize:    0, // Use default
			expectedFiles:  3, // Two small files + forced large file
			expectedPhrase: "Force include question",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create generator
			generator := NewGenerator(fileInfos, tc.question)
			
			// Set custom max file size if specified
			if tc.maxFileSize > 0 {
				generator.SetMaxFileSize(tc.maxFileSize)
			}

			// Generate prompt
			promptText, fileCount, err := generator.Generate()
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			// Check file count
			if fileCount != tc.expectedFiles {
				t.Errorf("Expected %d files in prompt, got %d", tc.expectedFiles, fileCount)
			}

			// Check that the question is included
			if !strings.Contains(promptText, tc.expectedPhrase) {
				t.Errorf("Expected prompt to contain %q, but it doesn't", tc.expectedPhrase)
			}

			// Check that the prompt contains the expected sections
			expectedSections := []string{
				"PROJECT STRUCTURE",
				"FILE CONTENT",
				"Based on the context provided above",
			}

			for _, section := range expectedSections {
				if !strings.Contains(promptText, section) {
					t.Errorf("Expected prompt to contain section %q, but it doesn't", section)
				}
			}
		})
	}
}