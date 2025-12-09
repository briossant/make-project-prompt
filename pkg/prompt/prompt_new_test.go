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

	t.Run("Raw mode excludes default mode messages", func(t *testing.T) {
		generator := NewGenerator(fileInfos, "", false)
		generator.RawMode = true
		generator.Questions = []ContentItem{
			{Type: "question", Content: "Question", Order: 0},
		}

		promptText, _, err := generator.Generate()
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		// In raw mode, these default mode messages should be excluded
		if strings.Contains(promptText, "Here is the context of my current project") {
			t.Error("Raw mode should not include default intro message")
		}
		if strings.Contains(promptText, "PROJECT STRUCTURE") {
			t.Error("Raw mode should not include project structure")
		}
	})

	t.Run("Default mode includes all expected sections", func(t *testing.T) {
		generator := NewGenerator(fileInfos, "", false)
		generator.RawMode = false // Default mode
		generator.Questions = []ContentItem{
			{Type: "question", Content: "Question", Order: 0},
		}

		promptText, _, err := generator.Generate()
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		// Default mode should include these
		expectedPhrases := []string{
			"Here is the context of my current project",
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

func TestGenerator_RawModeInterleaving(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "prompt_test_interleave")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Warning: Failed to remove temp directory: %v", err)
		}
	}()

	file1 := filepath.Join(tempDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("Content of file 1"), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}

	file2 := filepath.Join(tempDir, "file2.txt")
	if err := os.WriteFile(file2, []byte("Content of file 2"), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	fileInfos1 := []files.FileInfo{
		{
			Path:      file1,
			IsText:    true,
			IsForced:  false,
			Size:      int64(len("Content of file 1")),
			IsRegular: true,
		},
	}

	fileInfos2 := []files.FileInfo{
		{
			Path:      file2,
			IsText:    true,
			IsForced:  false,
			Size:      int64(len("Content of file 2")),
			IsRegular: true,
		},
	}

	t.Run("Raw mode interleaves questions and files correctly", func(t *testing.T) {
		generator := NewGenerator([]files.FileInfo{}, "", false)
		generator.RawMode = true
		generator.ContentItems = []ContentItem{
			{Type: "question", Content: "Header text", Order: 0},
			{Type: "file_group", Files: fileInfos1, Order: 1},
			{Type: "question", Content: "Middle text", Order: 2},
			{Type: "file_group", Files: fileInfos2, Order: 3},
			{Type: "question", Content: "Footer text", Order: 4},
		}

		promptText, fileCount, err := generator.Generate()
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if fileCount != 2 {
			t.Errorf("Expected 2 files in prompt, got %d", fileCount)
		}

		// Check order of content
		headerIdx := strings.Index(promptText, "Header text")
		file1Idx := strings.Index(promptText, "Content of file 1")
		middleIdx := strings.Index(promptText, "Middle text")
		file2Idx := strings.Index(promptText, "Content of file 2")
		footerIdx := strings.Index(promptText, "Footer text")

		if headerIdx == -1 || file1Idx == -1 || middleIdx == -1 || file2Idx == -1 || footerIdx == -1 {
			t.Fatal("Not all content items found in prompt")
		}

		// Verify order
		if !(headerIdx < file1Idx && file1Idx < middleIdx && middleIdx < file2Idx && file2Idx < footerIdx) {
			t.Error("Content items are not in the correct order")
			t.Logf("Order: header=%d, file1=%d, middle=%d, file2=%d, footer=%d",
				headerIdx, file1Idx, middleIdx, file2Idx, footerIdx)
		}
	})
}
