package functional

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var mppBinaryPath string

// TestMain compiles the binary once and cleans it up after tests.
func TestMain(m *testing.M) {
	var err error
	// Create a temporary file for the binary
	tempFile, err := os.CreateTemp("", "mpp-test-binary")
	if err != nil {
		fmt.Printf("Failed to create temp file for binary: %v\n", err)
		os.Exit(1)
	}
	mppBinaryPath = tempFile.Name()
	tempFile.Close() // Close the file so the build command can write to it

	// Get project root to find the main package
	// This assumes the test is in a sub-directory of the project root.
	wd, _ := os.Getwd() // e.g., /path/to/project/test/functional
	projectRoot := filepath.Join(wd, "..", "..")

	// Compile the binary
	buildCmd := exec.Command("go", "build", "-o", mppBinaryPath, "./cmd/make-project-prompt")
	buildCmd.Dir = projectRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		fmt.Printf("Failed to build binary: %v\nOutput: %s\n", err, string(output))
		os.Remove(mppBinaryPath)
		os.Exit(1)
	}

	// Run the tests
	exitCode := m.Run()

	// Cleanup
	os.Remove(mppBinaryPath)
	os.Exit(exitCode)
}

// setupTestRepo creates a test Git repository and returns its path
func setupTestRepo(t *testing.T) string {
	t.Helper()
	// Get the path to the setup script
	scriptPath := filepath.Join(".", "setup_test_repo.sh")
	cmd := exec.Command("bash", scriptPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run setup_test_repo.sh: %v\nOutput: %s", err, string(output))
	}

	// Extract the last line, which contains the repository path
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	repoPath := lines[len(lines)-1]

	t.Logf("Test repository created at: %s", repoPath)
	return repoPath
}

// cleanupTestRepo removes the test repository
func cleanupTestRepo(t *testing.T, repoPath string) {
	t.Helper()
	if repoPath != "" && strings.HasPrefix(repoPath, os.TempDir()) {
		os.RemoveAll(repoPath)
	}
}

func TestFunctionalMPP(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	// Create a question file for one of the tests
	questionFilePath := filepath.Join(repoPath, "question.txt")
	os.WriteFile(questionFilePath, []byte("What is the role of app.go?"), 0644)

	testCases := []struct {
		name                 string
		args                 string // Args will be passed to bash -c for glob expansion
		expectedToContain    []string
		expectedToNotContain []string
	}{
		{
			name: "Default - all tracked files",
			args: `-q "Default test"`,
			expectedToContain: []string{
				"--- FILE: src/main/app.go ---",
				"--- FILE: src/test/app_test.go ---",
				"--- FILE: docs/README.md ---",
				"--- FILE: .gitignore ---",
				"Based on the context provided above, answer the following question:\n\nDefault test",
			},
			expectedToNotContain: []string{
				"--- FILE: binary_file.bin ---", // Ignored by .gitignore and binary
				"--- FILE: build/output.txt ---", // Ignored by .gitignore
			},
		},
		{
			name: "Include only main go files",
			args: `-i src/main/app.go -i src/main/utils.go -q "Include Go files"`,
			expectedToContain: []string{
				"--- FILE: src/main/app.go ---",
				"--- FILE: src/main/utils.go ---",
				"Include Go files",
			},
			expectedToNotContain: []string{
				"--- FILE: src/test/app_test.go ---",
				"--- FILE: docs/README.md ---",
			},
		},
		{
			name: "Exclude test files",
			args: `-e src/test/app_test.go -q "Exclude tests"`,
			expectedToContain: []string{
				"--- FILE: src/main/app.go ---",
				"--- FILE: docs/README.md ---",
				"Exclude tests",
			},
			expectedToNotContain: []string{
				"--- FILE: src/test/app_test.go ---",
			},
		},
		{
			name: "Force include binary file",
			args: `-f binary_file.bin -q "Force include binary"`,
			expectedToContain: []string{
				"--- FILE: binary_file.bin ---",
				"Force include binary",
			},
			expectedToNotContain: []string{},
		},
		{
			name: "Question from file",
			args: fmt.Sprintf(`-qf "%s"`, questionFilePath),
			expectedToContain: []string{
				"What is the role of app.go?",
			},
			expectedToNotContain: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			outputFile, err := os.CreateTemp("", "mpp-output-*.txt")
			if err != nil {
				t.Fatalf("Failed to create temp output file: %v", err)
			}
			defer os.Remove(outputFile.Name())
			outputFile.Close() // Close file so the command can write to it

			// Construct the command to be run inside the test repo
			// This is the key: we use 'bash -c' to ensure glob expansion
			// happens exactly as it would for a real user.
			commandString := fmt.Sprintf("%s --output %s %s", mppBinaryPath, outputFile.Name(), tc.args)
			t.Logf("Running command: %s", commandString)
			t.Logf("From directory: %s", repoPath)

			cmd := exec.Command("bash", "-c", commandString)
			cmd.Dir = repoPath // Run the command from within the test repository

			// Run and check for errors
			output, err := cmd.CombinedOutput()
			t.Logf("Command output:\n%s", string(output))
			if err != nil {
				t.Fatalf("Command failed: %v\nOutput:\n%s", err, string(output))
			}

			// Read the generated prompt
			promptBytes, err := os.ReadFile(outputFile.Name())
			if err != nil {
				t.Fatalf("Failed to read prompt output file: %v", err)
			}
			promptContent := string(promptBytes)
			previewLen := 200
			if len(promptContent) < previewLen {
				previewLen = len(promptContent)
			}
			t.Logf("Prompt content (first %d chars):\n%s", previewLen, promptContent[:previewLen])

			// Perform assertions
			for _, expected := range tc.expectedToContain {
				if !strings.Contains(promptContent, expected) {
					t.Errorf("Expected prompt to contain:\n---\n%s\n---\n...but it did not.", expected)
				}
			}
			for _, notExpected := range tc.expectedToNotContain {
				if strings.Contains(promptContent, notExpected) {
					t.Errorf("Expected prompt to NOT contain:\n---\n%s\n---\n...but it did.", notExpected)
				}
			}

			// Example of a more complex assertion using regex
			// Checks for the tree structure
			treeRegex := regexp.MustCompile(`\.\n(├──|└──)`)
			if !treeRegex.MatchString(promptContent) {
				t.Errorf("Prompt does not appear to contain a valid 'tree' structure.")
			}
		})
	}
}
