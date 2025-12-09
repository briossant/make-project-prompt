package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/briossant/make-project-prompt/pkg/files"
)

func TestGenerator_MultipleQuestions(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "prompt_test_multi_q")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Warning: Failed to remove temp directory: %v", err)
		}
	}()

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("Test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	fileInfos := []files.FileInfo{
		{
			Path:      testFile,
			IsText:    true,
			IsForced:  false,
			Size:      int64(len("Test content")),
			IsRegular: true,
		},
	}

	// Test multiple questions accumulation
	t.Run("Multiple questions are accumulated", func(t *testing.T) {
		generator := NewGenerator(fileInfos, "", false)
		
		// Add multiple questions
		generator.AddQuestion("First question", 0)
		generator.AddQuestion("Second question", 1)
		generator.AddQuestion("Third question", 2)

		promptText, fileCount, err := generator.Generate()
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if fileCount != 1 {
			t.Errorf("Expected 1 file in prompt, got %d", fileCount)
		}

		// All questions should be in the prompt
		for _, q := range []string{"First question", "Second question", "Third question"} {
			if !strings.Contains(promptText, q) {
				t.Errorf("Expected prompt to contain %q, but it doesn't", q)
			}
		}

		// Questions should appear after files
		fileIdx := strings.Index(promptText, "--- FILE: ")
		q1Idx := strings.Index(promptText, "First question")
		if fileIdx == -1 || q1Idx == -1 || fileIdx > q1Idx {
			t.Error("Questions should appear after files in default mode")
		}
	})

	// Test Questions array
	t.Run("Questions array works correctly", func(t *testing.T) {
		questions := []ContentItem{
			{Type: "question", Content: "Question A", Order: 0},
			{Type: "question", Content: "Question B", Order: 1},
		}

		generator := NewGenerator(fileInfos, "", false)
		generator.Questions = questions

		promptText, _, err := generator.Generate()
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if !strings.Contains(promptText, "Question A") {
			t.Error("Expected prompt to contain 'Question A'")
		}
		if !strings.Contains(promptText, "Question B") {
			t.Error("Expected prompt to contain 'Question B'")
		}
	})
}

func TestGenerator_RawMode(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "prompt_test_raw")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Warning: Failed to remove temp directory: %v", err)
		}
	}()

	// Create a test file
	testFile := filepath.Join(tempDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	fileInfos := []files.FileInfo{
		{
			Path:      testFile,
			IsText:    true,
			IsForced:  false,
			Size:      int64(len("package main")),
			IsRegular: true,
		},
	}

	t.Run("Raw mode removes pre-written messages", func(t *testing.T) {
		generator := NewGenerator(fileInfos, "", false)
		generator.RawMode = true
		generator.Questions = []ContentItem{
			{Type: "question", Content: "Test question", Order: 0},
		}

		promptText, fileCount, err := generator.Generate()
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if fileCount != 1 {
			t.Errorf("Expected 1 file in prompt, got %d", fileCount)
		}

		// Should NOT contain pre-written messages
		unwantedPhrases := []string{
			"Here is the context of my current project",
			"Based on the context provided above",
			"PROJECT STRUCTURE",
		}

		for _, phrase := range unwantedPhrases {
			if strings.Contains(promptText, phrase) {
				t.Errorf("Raw mode should not contain phrase %q, but it does", phrase)
			}
		}

		// Should still contain file separators and question
		if !strings.Contains(promptText, "--- FILE: ") {
			t.Error("Raw mode should still contain file separators")
		}
		if !strings.Contains(promptText, "Test question") {
			t.Error("Raw mode should contain the question")
		}
	})

	t.Run("Raw mode excludes role message and extra context", func(t *testing.T) {
		generator := NewGenerator(fileInfos, "", false)
		generator.RawMode = true
		generator.RoleMessage = "You are an expert"
		generator.ExtraContext = "Extra info"
		generator.LastWords = "Final words"
		generator.Questions = []ContentItem{
			{Type: "question", Content: "Question", Order: 0},
		}

		promptText, _, err := generator.Generate()
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		// In raw mode, these should be excluded
		// (In current implementation they are not used, but verifying)
		if strings.Contains(promptText, "You are an expert") {
			t.Error("Raw mode should not include role message (currently not implemented to include it)")
		}
	})

	t.Run("Default mode includes all messages", func(t *testing.T) {
		generator := NewGenerator(fileInfos, "", false)
		generator.RawMode = false // Default mode
		generator.RoleMessage = "You are an expert"
		generator.ExtraContext = "Extra info"
		generator.LastWords = "Final words"
		generator.Questions = []ContentItem{
			{Type: "question", Content: "Question", Order: 0},
		}

		promptText, _, err := generator.Generate()
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		// Default mode should include these
		expectedPhrases := []string{
			"You are an expert",
			"Here is the context of my current project",
			"Extra info",
			"Final words",
			"Question",
		}

		for _, phrase := range expectedPhrases {
			if !strings.Contains(promptText, phrase) {
				t.Errorf("Default mode should contain phrase %q, but it doesn't", phrase)
			}
		}
	})
}

func TestGenerator_EmptyQuestions(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "prompt_test_empty_q")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Warning: Failed to remove temp directory: %v", err)
		}
	}()

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	fileInfos := []files.FileInfo{
		{
			Path:      testFile,
			IsText:    true,
			IsForced:  false,
			Size:      int64(len("content")),
			IsRegular: true,
		},
	}

	t.Run("Empty questions array still generates prompt", func(t *testing.T) {
		generator := NewGenerator(fileInfos, "", false)
		generator.Questions = []ContentItem{} // Empty

		promptText, fileCount, err := generator.Generate()
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if fileCount != 1 {
			t.Errorf("Expected 1 file in prompt, got %d", fileCount)
		}

		// Should still contain file content
		if !strings.Contains(promptText, "--- FILE: ") {
			t.Error("Should contain file separators")
		}
	})
}
