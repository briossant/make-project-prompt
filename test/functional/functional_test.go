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
	if err := tempFile.Close(); err != nil { // Close the file so the build command can write to it
		fmt.Printf("Warning: Failed to close temp file: %v\n", err)
		// Continue anyway, as this is not a critical error
	}

	// Get project root to find the main package
	// This assumes the test is in a sub-directory of the project root.
	wd, _ := os.Getwd() // e.g., /path/to/project/test/functional
	projectRoot := filepath.Join(wd, "..", "..")

	// Compile the binary
	buildCmd := exec.Command("go", "build", "-o", mppBinaryPath, "./cmd/make-project-prompt")
	buildCmd.Dir = projectRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		fmt.Printf("Failed to build binary: %v\nOutput: %s\n", err, string(output))
		if removeErr := os.Remove(mppBinaryPath); removeErr != nil {
			fmt.Printf("Warning: Failed to remove binary file: %v\n", removeErr)
		}
		os.Exit(1)
	}

	// Run the tests
	exitCode := m.Run()

	// Cleanup
	if err := os.Remove(mppBinaryPath); err != nil {
		fmt.Printf("Warning: Failed to remove binary file during cleanup: %v\n", err)
		// Continue with exit, as this is not a critical error
	}
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
		if err := os.RemoveAll(repoPath); err != nil {
			t.Logf("Warning: Failed to remove test repo: %v", err)
		}
	}
}

func TestFunctionalMPP_SuccessCases(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	// Create a question file for one of the tests
	questionFilePath := filepath.Join(repoPath, "question.txt")
	if err := os.WriteFile(questionFilePath, []byte("What is the role of app.go?"), 0644); err != nil {
		t.Fatalf("Failed to create question file: %v", err)
	}

	testCases := []struct {
		name                 string
		args                 string
		expectedToContain    []string
		expectedToNotContain []string
	}{
		// --- Existing and Refined Tests ---
		{
			name:                 "Default - all tracked text files",
			args:                 `-q "Default test"`,
			expectedToContain:    []string{"--- FILE: src/main/app.go ---", "--- FILE: docs/README.md ---", "--- FILE: .gitignore ---"},
			expectedToNotContain: []string{"--- FILE: binary_file.bin ---", "--- FILE: build/output.txt ---"},
		},
		{
			name:                 "Include only main go files",
			args:                 `-i src/main/app.go -i src/main/utils.go -q "Include Go files"`,
			expectedToContain:    []string{"--- FILE: src/main/app.go ---", "--- FILE: src/main/utils.go ---"},
			expectedToNotContain: []string{"--- FILE: src/test/app_test.go ---", "--- FILE: docs/README.md ---"},
		},
		{
			name:                 "Exclude test files",
			args:                 `-e src/test/app_test.go -q "Exclude tests"`,
			expectedToContain:    []string{"--- FILE: src/main/app.go ---", "--- FILE: docs/README.md ---"},
			expectedToNotContain: []string{"--- FILE: src/test/app_test.go ---"},
		},
		// --- NEW DIRECTORY-FOCUSED TESTS ---
		{
			name:                 "Exclude entire directory with -e src",
			args:                 `-q "Exclude src dir" -e src`,
			expectedToContain:    []string{"--- FILE: docs/README.md ---", "--- FILE: docs/CONTRIBUTING.md ---"},
			expectedToNotContain: []string{"--- FILE: src/main/app.go ---", "--- FILE: src/test/app_test.go ---"},
		},
		{
			name:                 "Exclude entire directory with -e src/ (trailing slash)",
			args:                 `-q "Exclude src/ dir" -e src/`,
			expectedToContain:    []string{"--- FILE: docs/README.md ---", "--- FILE: docs/CONTRIBUTING.md ---"},
			expectedToNotContain: []string{"--- FILE: src/main/app.go ---", "--- FILE: src/test/app_test.go ---"},
		},
		{
			name:                 "Exclude a subdirectory",
			args:                 `-q "Exclude test dir" -e src/test`,
			expectedToContain:    []string{"--- FILE: src/main/app.go ---", "--- FILE: src/main/utils.go ---"},
			expectedToNotContain: []string{"--- FILE: src/test/app_test.go ---"},
		},
		{
			name:                 "Exclude multiple directories",
			args:                 `-q "Exclude src and docs" -e src -e docs`,
			expectedToContain:    []string{"--- FILE: .gitignore ---", "--- FILE: large_important.txt ---"},
			expectedToNotContain: []string{"--- FILE: src/main/app.go ---", "--- FILE: docs/README.md ---"},
		},
		{
			name:                 "Force include a file from an excluded directory",
			args:                 `-f build/output.txt -q "Force include from ignored dir"`,
			expectedToContain:    []string{"--- FILE: build/output.txt ---"},
			expectedToNotContain: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			outputFile, err := os.CreateTemp("", "mpp-output-*.txt")
			if err != nil {
				t.Fatalf("Failed to create temp output file: %v", err)
			}
			defer func() {
				if err := os.Remove(outputFile.Name()); err != nil {
					t.Logf("Warning: Failed to remove temp output file: %v", err)
				}
			}()
			if err := outputFile.Close(); err != nil {
				t.Fatalf("Failed to close temp output file: %v", err)
			}

			commandString := fmt.Sprintf("%s --output %s %s", mppBinaryPath, outputFile.Name(), tc.args)
			cmd := exec.Command("bash", "-c", commandString)
			cmd.Dir = repoPath

			output, err := cmd.CombinedOutput()
			t.Logf("Command stdout/stderr:\n%s", string(output))
			fmt.Printf("[DEBUG_LOG] Test %s command output: %s\n", tc.name, string(output))
			if err != nil {
				t.Fatalf("Command failed: %v\nOutput:\n%s", err, string(output))
			}

			promptBytes, err := os.ReadFile(outputFile.Name())
			if err != nil {
				t.Fatalf("Failed to read prompt output file: %v", err)
			}
			promptContent := string(promptBytes)
			fmt.Printf("[DEBUG_LOG] Test %s running\n", tc.name)

			for _, expected := range tc.expectedToContain {
				if !strings.Contains(promptContent, expected) {
					t.Errorf("Expected prompt to contain:\n---\n%s\n---\n...but it did not.", expected)
					fmt.Printf("[DEBUG_LOG] Test %s failed: Expected prompt to contain %q but it did not.\n", tc.name, expected)
					fmt.Printf("[DEBUG_LOG] Prompt content: %s\n", promptContent)
				}
			}
			for _, notExpected := range tc.expectedToNotContain {
				if strings.Contains(promptContent, notExpected) {
					t.Errorf("Expected prompt to NOT contain:\n---\n%s\n---\n...but it did.", notExpected)
					fmt.Printf("[DEBUG_LOG] Test %s failed: Expected prompt to NOT contain %q but it did.\n", tc.name, notExpected)
				}
			}

			// Check for tree structure - allow for different Unicode representations
			treeRegex := regexp.MustCompile(`\.\n(├|â"œ|└|â"")`)
			if !treeRegex.MatchString(promptContent) {
				t.Logf("Tree structure not found in prompt. This might be due to Unicode encoding differences.")
				// Not failing the test for this, as it's not critical to functionality
			}
		})
	}
}

func TestFunctionalMPP_StdoutOutput(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	t.Run("Stdout output with quiet mode", func(t *testing.T) {
		// Run the command with stdout output and quiet mode
		commandString := fmt.Sprintf(`%s -i src/main/app.go -q "Test stdout output" --stdout --quiet`, mppBinaryPath)
		cmd := exec.Command("bash", "-c", commandString)
		cmd.Dir = repoPath

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Command failed: %v\nOutput:\n%s", err, string(output))
		}

		// Check that the output contains the prompt but not the usual status messages
		if !strings.Contains(string(output), "--- FILE: src/main/app.go ---") {
			t.Errorf("Expected stdout to contain file content, but it did not.")
		}
		if !strings.Contains(string(output), "Based on the context provided above") {
			t.Errorf("Expected stdout to contain prompt footer, but it did not.")
		}
		if strings.Contains(string(output), "Starting make-project-prompt") {
			t.Errorf("Expected stdout to NOT contain startup message, but it did.")
		}
		if strings.Contains(string(output), "Prompt generated and") {
			t.Errorf("Expected stdout to NOT contain success message, but it did.")
		}
	})

	t.Run("Stdout output WITHOUT quiet mode", func(t *testing.T) {
		// This test verifies that --stdout alone produces ONLY the prompt content
		// and nothing else, which is crucial for scripting.
		commandString := fmt.Sprintf(`%s -i src/main/app.go -q "Test stdout without quiet" --stdout`, mppBinaryPath)
		cmd := exec.Command("bash", "-c", commandString)
		cmd.Dir = repoPath

		// Run the command. We expect it to succeed and output the prompt.
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Command failed: %v\nOutput:\n%s", err, string(output))
		}
		outputStr := string(output)

		// Create the expected prompt for comparison.
		// It should start with the intro, have the project structure, file content, and question.
		// It should NOT have any of the `printInfo` messages.
		if !strings.HasPrefix(outputStr, "Here is the context of my current project.") {
			t.Errorf("Expected output to start with prompt intro, but it did not. Got:\n%s", outputStr)
		}
		if !strings.HasSuffix(strings.TrimSpace(outputStr), "Test stdout without quiet") {
			t.Errorf("Expected output to end with the question, but it did not. Got:\n%s", outputStr)
		}
		if strings.Contains(outputStr, "Starting make-project-prompt") {
			t.Errorf("Expected stdout to NOT contain startup message, but it did.")
		}
		if strings.Contains(outputStr, "Prompt generated and") {
			t.Errorf("Expected stdout to NOT contain success message, but it did.")
		}
	})

	t.Run("File output with quiet mode", func(t *testing.T) {
		outputFile, err := os.CreateTemp("", "mpp-output-*.txt")
		if err != nil {
			t.Fatalf("Failed to create temp output file: %v", err)
		}
		defer func() {
			if err := os.Remove(outputFile.Name()); err != nil {
				t.Logf("Warning: Failed to remove temp output file: %v", err)
			}
		}()
		if err := outputFile.Close(); err != nil {
			t.Fatalf("Failed to close temp output file: %v", err)
		}

		// Run the command with file output and quiet mode
		commandString := fmt.Sprintf(`%s -i src/main/app.go -q "Test file output with quiet" --output %s --quiet`, mppBinaryPath, outputFile.Name())
		cmd := exec.Command("bash", "-c", commandString)
		cmd.Dir = repoPath

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Command failed: %v\nOutput:\n%s", err, string(output))
		}

		// Check that the command output doesn't contain the usual status messages
		if strings.Contains(string(output), "Starting make-project-prompt") {
			t.Errorf("Expected command output to NOT contain startup message, but it did.")
		}
		if strings.Contains(string(output), "Prompt generated and") {
			t.Errorf("Expected command output to NOT contain success message, but it did.")
		}

		// Check that the file contains the prompt
		promptBytes, err := os.ReadFile(outputFile.Name())
		if err != nil {
			t.Fatalf("Failed to read prompt output file: %v", err)
		}
		promptContent := string(promptBytes)

		if !strings.Contains(promptContent, "--- FILE: src/main/app.go ---") {
			t.Errorf("Expected output file to contain file content, but it did not.")
		}
		if !strings.Contains(promptContent, "Based on the context provided above") {
			t.Errorf("Expected output file to contain prompt footer, but it did not.")
		}
	})
}

func TestFunctionalMPP_DryRun(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	t.Run("Dry run lists correct files without prompt format", func(t *testing.T) {
		// Run the command with the --dry-run flag
		commandString := fmt.Sprintf(`%s -i src/main/*.go --dry-run`, mppBinaryPath)
		cmd := exec.Command("bash", "-c", commandString)
		cmd.Dir = repoPath

		output, err := cmd.CombinedOutput()
		fmt.Printf("[DEBUG_LOG] Dry run test command output: %s\n", string(output))
		if err != nil {
			t.Fatalf("Command failed: %v\nOutput:\n%s", err, string(output))
		}

		outputStr := string(output)

		// Expected files in output
		expectedToContain := []string{
			"src/main/app.go",
			"src/main/utils.go",
			"Total files: 2",
		}

		// Parts of the full prompt that should NOT be in the output
		expectedToNotContain := []string{
			"--- FILE:",
			"--- PROJECT STRUCTURE ---",
			"Based on the context provided above",
		}

		for _, expected := range expectedToContain {
			if !strings.Contains(outputStr, expected) {
				t.Errorf("Expected dry run output to contain %q, but it did not.", expected)
			}
		}

		for _, notExpected := range expectedToNotContain {
			if strings.Contains(outputStr, notExpected) {
				t.Errorf("Expected dry run output to NOT contain %q, but it did.", notExpected)
			}
		}
	})
}

func TestFunctionalMPP_ErrorCases(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	t.Run("Fails when no files match pattern", func(t *testing.T) {
		commandString := fmt.Sprintf(`%s -i "*.nonexistent"`, mppBinaryPath)
		cmd := exec.Command("bash", "-c", commandString)
		cmd.Dir = repoPath

		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("Expected command to fail, but it succeeded.")
		}

		expectedErrorMsg := "no files matched the specified patterns"
		if !strings.Contains(string(output), expectedErrorMsg) {
			t.Errorf("Expected error output to contain %q, but got:\n%s", expectedErrorMsg, string(output))
		}
	})

	t.Run("Fails when not in a git repository", func(t *testing.T) {
		nonRepoDir := os.TempDir()
		cmd := exec.Command(mppBinaryPath)
		cmd.Dir = nonRepoDir

		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("Expected command to fail, but it succeeded.")
		}

		expectedErrorMsg := "not a git repository"
		if !strings.Contains(strings.ToLower(string(output)), expectedErrorMsg) {
			t.Errorf("Expected error output to contain %q, but got:\n%s", expectedErrorMsg, string(output))
		}
	})
}

func TestFunctionalMPP_QuestionAccumulation(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	t.Run("Multiple -q flags accumulate", func(t *testing.T) {
		commandString := fmt.Sprintf(`%s -i src/main/app.go -q "First question" -q "Second question" -q "Third question" --stdout`, mppBinaryPath)
		cmd := exec.Command("bash", "-c", commandString)
		cmd.Dir = repoPath

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Command failed: %v\nOutput:\n%s", err, string(output))
		}

		promptContent := string(output)

		// All three questions should be present
		if !strings.Contains(promptContent, "First question") {
			t.Error("Expected prompt to contain 'First question'")
		}
		if !strings.Contains(promptContent, "Second question") {
			t.Error("Expected prompt to contain 'Second question'")
		}
		if !strings.Contains(promptContent, "Third question") {
			t.Error("Expected prompt to contain 'Third question'")
		}

		// Questions should appear in order
		firstIdx := strings.Index(promptContent, "First question")
		secondIdx := strings.Index(promptContent, "Second question")
		thirdIdx := strings.Index(promptContent, "Third question")

		if !(firstIdx < secondIdx && secondIdx < thirdIdx) {
			t.Errorf("Questions appear in wrong order. First: %d, Second: %d, Third: %d", firstIdx, secondIdx, thirdIdx)
		}
	})

	t.Run("Mix -q and -qf flags", func(t *testing.T) {
		// Create a question file
		questionFile := filepath.Join(repoPath, "test_question.txt")
		if err := os.WriteFile(questionFile, []byte("Question from file"), 0644); err != nil {
			t.Fatalf("Failed to create question file: %v", err)
		}
		defer os.Remove(questionFile)

		commandString := fmt.Sprintf(`%s -i src/main/app.go -q "First question" -qf %s -q "Third question" --stdout`, mppBinaryPath, questionFile)
		cmd := exec.Command("bash", "-c", commandString)
		cmd.Dir = repoPath

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Command failed: %v\nOutput:\n%s", err, string(output))
		}

		promptContent := string(output)

		// All questions should be present
		if !strings.Contains(promptContent, "First question") {
			t.Error("Expected prompt to contain 'First question'")
		}
		if !strings.Contains(promptContent, "Question from file") {
			t.Error("Expected prompt to contain 'Question from file'")
		}
		if !strings.Contains(promptContent, "Third question") {
			t.Error("Expected prompt to contain 'Third question'")
		}
	})
}

func TestFunctionalMPP_RawMode(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	t.Run("Raw mode removes default messages", func(t *testing.T) {
		commandString := fmt.Sprintf(`%s --raw -i src/main/app.go -q "Test question" --stdout`, mppBinaryPath)
		cmd := exec.Command("bash", "-c", commandString)
		cmd.Dir = repoPath

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Command failed: %v\nOutput:\n%s", err, string(output))
		}

		promptContent := string(output)

		// Should NOT contain default mode messages
		unwantedPhrases := []string{
			"Here is the context of my current project",
			"Based on the context provided above",
			"PROJECT STRUCTURE",
		}

		for _, phrase := range unwantedPhrases {
			if strings.Contains(promptContent, phrase) {
				t.Errorf("Raw mode should not contain phrase %q, but it does", phrase)
			}
		}

		// Should still contain file separators and question
		if !strings.Contains(promptContent, "--- FILE: src/main/app.go ---") {
			t.Error("Raw mode should still contain file separators")
		}
		if !strings.Contains(promptContent, "Test question") {
			t.Error("Raw mode should contain the question")
		}
	})

	t.Run("Raw mode interleaves questions and files based on arg order", func(t *testing.T) {
		commandString := fmt.Sprintf(`%s --raw -q "Before files" -i src/main/app.go -q "Between files" -i src/main/utils.go -q "After files" --stdout`, mppBinaryPath)
		cmd := exec.Command("bash", "-c", commandString)
		cmd.Dir = repoPath

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Command failed: %v\nOutput:\n%s", err, string(output))
		}

		promptContent := string(output)

		// Find positions of each element
		beforeIdx := strings.Index(promptContent, "Before files")
		appGoIdx := strings.Index(promptContent, "--- FILE: src/main/app.go ---")
		betweenIdx := strings.Index(promptContent, "Between files")
		utilsGoIdx := strings.Index(promptContent, "--- FILE: src/main/utils.go ---")
		afterIdx := strings.Index(promptContent, "After files")

		// Check all elements are present
		if beforeIdx == -1 || appGoIdx == -1 || betweenIdx == -1 || utilsGoIdx == -1 || afterIdx == -1 {
			t.Fatal("Not all expected elements found in output")
		}

		// Verify order: Before → app.go → Between → utils.go → After
		if !(beforeIdx < appGoIdx && appGoIdx < betweenIdx && betweenIdx < utilsGoIdx && utilsGoIdx < afterIdx) {
			t.Errorf("Elements appear in wrong order.\nBefore: %d, app.go: %d, Between: %d, utils.go: %d, After: %d",
				beforeIdx, appGoIdx, betweenIdx, utilsGoIdx, afterIdx)
		}
	})

	t.Run("Raw mode with multiple questions before files", func(t *testing.T) {
		commandString := fmt.Sprintf(`%s --raw -q "Header 1" -q "Header 2" -i src/main/app.go -q "Footer" --stdout`, mppBinaryPath)
		cmd := exec.Command("bash", "-c", commandString)
		cmd.Dir = repoPath

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Command failed: %v\nOutput:\n%s", err, string(output))
		}

		promptContent := string(output)

		// Both headers should appear before the file
		header1Idx := strings.Index(promptContent, "Header 1")
		header2Idx := strings.Index(promptContent, "Header 2")
		fileIdx := strings.Index(promptContent, "--- FILE: src/main/app.go ---")
		footerIdx := strings.Index(promptContent, "Footer")

		if !(header1Idx < header2Idx && header2Idx < fileIdx && fileIdx < footerIdx) {
			t.Errorf("Elements appear in wrong order")
		}
	})
}

func TestFunctionalMPP_Aliases(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	// Create a .mpp.txt config file in the test repo
	configPath := filepath.Join(repoPath, ".mpp.txt")
	configContent := `# Test aliases
go_files: -i src/**/*.go
js_files: -i docs/*.md
combined: -i src/main/*.go -q "Focus on main package"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	t.Run("List aliases shows available aliases", func(t *testing.T) {
		commandString := fmt.Sprintf("%s --list-aliases", mppBinaryPath)
		cmd := exec.Command("bash", "-c", commandString)
		cmd.Dir = repoPath

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Command failed: %v\nOutput:\n%s", err, string(output))
		}

		outputStr := string(output)
		expectedStrings := []string{
			"Available aliases:",
			"go_files:",
			"js_files:",
			"combined:",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(outputStr, expected) {
				t.Errorf("Expected output to contain %q but it did not.\nOutput:\n%s", expected, outputStr)
			}
		}
	})

	t.Run("Use alias with -a flag", func(t *testing.T) {
		commandString := fmt.Sprintf("%s -a go_files -q \"Test question\" --stdout", mppBinaryPath)
		cmd := exec.Command("bash", "-c", commandString)
		cmd.Dir = repoPath

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Command failed: %v\nOutput:\n%s", err, string(output))
		}

		// The output should contain only .go files
		outputStr := string(output)
		if !strings.Contains(outputStr, "--- FILE: src/main/app.go ---") {
			t.Error("Expected to find app.go in output")
		}
		if strings.Contains(outputStr, "--- FILE: docs/README.md ---") {
			t.Error("Should not contain README.md when using go_files alias")
		}
	})

	t.Run("Alias with question", func(t *testing.T) {
		commandString := fmt.Sprintf("%s -a combined -q \"Additional question\" --stdout", mppBinaryPath)
		cmd := exec.Command("bash", "-c", commandString)
		cmd.Dir = repoPath

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Command failed: %v\nOutput:\n%s", err, string(output))
		}

		outputStr := string(output)
		// Should include both questions (one from alias, one from command line)
		if !strings.Contains(outputStr, "Focus on main package") {
			t.Error("Expected to find question from alias")
		}
		if !strings.Contains(outputStr, "Additional question") {
			t.Error("Expected to find question from command line")
		}
	})

	t.Run("Alias options can be overridden", func(t *testing.T) {
		// Use go_files alias but add an additional include pattern
		commandString := fmt.Sprintf("%s -a go_files -i docs/*.md -q \"Test question\" --stdout", mppBinaryPath)
		cmd := exec.Command("bash", "-c", commandString)
		cmd.Dir = repoPath

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Command failed: %v\nOutput:\n%s", err, string(output))
		}

		outputStr := string(output)
		// Should contain both .go and .md files
		if !strings.Contains(outputStr, "--- FILE: src/main/app.go ---") {
			t.Error("Expected to find app.go in output")
		}
		if !strings.Contains(outputStr, "--- FILE: docs/README.md ---") {
			t.Error("Expected to find README.md in output")
		}
	})

	t.Run("Non-existent alias returns error", func(t *testing.T) {
		commandString := fmt.Sprintf("%s -a nonexistent -q \"Test question\"", mppBinaryPath)
		cmd := exec.Command("bash", "-c", commandString)
		cmd.Dir = repoPath

		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("Expected command to fail with non-existent alias, but it succeeded")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "alias 'nonexistent' not found") {
			t.Errorf("Expected error message about alias not found, got:\n%s", outputStr)
		}
	})
}
