package files

import (
	"os"
	"path/filepath"
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
			name:  "Include only text files",
			files: []string{textFile, binaryFile, subFile},
			config: Config{
				IncludePatterns: []string{"test.txt"},
			},
			expectedCount: 1,
		},
		{
			name:  "Exclude subdirectory",
			files: []string{textFile, binaryFile, subFile},
			config: Config{
				ExcludePatterns: []string{"subdir/*"},
			},
			expectedCount: 1, // Only the root text file
		},
		{
			name:  "Force include binary file",
			files: []string{textFile, binaryFile, subFile},
			config: Config{
				ForceIncludePatterns: []string{"*.bin"},
			},
			expectedCount:  3, // All files (2 text + 1 forced binary)
			expectedForced: true,
		},
		{
			name:  "Complex pattern matching",
			files: []string{textFile, binaryFile, subFile},
			config: Config{
				IncludePatterns:     []string{"test.txt"},
				ExcludePatterns:     []string{"subdir/*"},
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
