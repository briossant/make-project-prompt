// Package prompt provides functionality for generating prompts for LLMs.
// It handles formatting file content and project structure into a prompt.
package prompt

import (
	"fmt"
	"os"
	"strings"

	"github.com/briossant/make-project-prompt/pkg/files"
)

// ContentItem represents a piece of content to include in the prompt
type ContentItem struct {
	Type    string // "question", "file_pattern", "tree"
	Content string // The actual content or pattern
	Order   int    // Original position in args (for --raw mode)
}

// Generator handles prompt generation
type Generator struct {
	Files         []files.FileInfo
	Question      string // Deprecated: use Questions for new code
	Questions     []ContentItem
	MaxFileSize   int64
	QuietMode     bool
	RoleMessage   string
	ExtraContext  string
	LastWords     string
	RawMode       bool
	FilePatterns  []ContentItem // For --raw mode: track file patterns with order
	IncludeTree   bool          // Whether to include project tree
}

// NewGenerator creates a new prompt generator
func NewGenerator(fileInfos []files.FileInfo, question string, quietMode bool) *Generator {
	questions := []ContentItem{}
	if question != "" && question != "[YOUR QUESTION HERE]" {
		questions = append(questions, ContentItem{
			Type:    "question",
			Content: question,
			Order:   0,
		})
	}
	return &Generator{
		Files:       fileInfos,
		Question:    question, // Keep for backward compatibility
		Questions:   questions,
		MaxFileSize: 1048576, // 1MB default max file size
		QuietMode:   quietMode,
		IncludeTree: true,
		RawMode:     false,
	}
}

// AddQuestion adds a question to the generator (for accumulation strategy)
func (g *Generator) AddQuestion(content string, order int) {
	g.Questions = append(g.Questions, ContentItem{
		Type:    "question",
		Content: content,
		Order:   order,
	})
}

// SetMaxFileSize sets the maximum file size for inclusion in the prompt
func (g *Generator) SetMaxFileSize(size int64) {
	g.MaxFileSize = size
}

// Generate creates the prompt with file content and project structure
func (g *Generator) Generate() (string, int, error) {
	if g.RawMode {
		return g.generateRawMode()
	}
	return g.generateDefaultMode()
}

// generateDefaultMode creates the prompt in default mode (with pre-written messages)
func (g *Generator) generateDefaultMode() (string, int, error) {
	var promptContent strings.Builder
	fileCounter := 0

	// Role message (if provided)
	if g.RoleMessage != "" {
		promptContent.WriteString(g.RoleMessage + "\n\n")
	}

	// Introduction
	promptContent.WriteString("Here is the context of my current project. Analyze the structure and content of the provided files to answer my question.\n\n")

	// Project structure via 'tree'
	if g.IncludeTree {
		promptContent.WriteString("--- PROJECT STRUCTURE (based on 'tree', may differ slightly from included files) ---\n")
		projectTree, err := files.GetProjectTree()
		if err != nil {
			if !g.QuietMode {
				fmt.Fprintf(os.Stderr, "Warning: Failed to get project tree: %v\n", err)
			}
			promptContent.WriteString("Error running tree command.\n")
		} else {
			promptContent.WriteString(projectTree)
		}
		promptContent.WriteString("\n")
	}

	// Content of relevant files
	promptContent.WriteString("--- FILE CONTENT (based on git ls-files, respecting .gitignore and -i/-e/-f options) ---\n")

	fileCounter = g.writeFiles(&promptContent)

	promptContent.WriteString("\n--- END OF FILE CONTENT ---\n")

	// Extra context (if provided)
	if g.ExtraContext != "" {
		promptContent.WriteString("\n" + g.ExtraContext + "\n")
	}

	// Final question(s) - accumulate all questions
	if len(g.Questions) > 0 {
		promptContent.WriteString("\nBased on the context provided above, answer the following question:\n\n")
		for _, q := range g.Questions {
			promptContent.WriteString(q.Content + "\n")
		}
	} else if g.Question != "" && g.Question != "[YOUR QUESTION HERE]" {
		// Backward compatibility: use old Question field if Questions is empty
		promptContent.WriteString("\nBased on the context provided above, answer the following question:\n\n")
		promptContent.WriteString(g.Question + "\n")
	}

	// Last words (if provided)
	if g.LastWords != "" {
		promptContent.WriteString("\n" + g.LastWords + "\n")
	}

	return promptContent.String(), fileCounter, nil
}

// generateRawMode creates the prompt in raw mode (minimal formatting, position-aware)
func (g *Generator) generateRawMode() (string, int, error) {
	var promptContent strings.Builder
	fileCounter := 0

	// In raw mode, we interleave questions and files based on order
	// For simplicity in this version: show all files, then all questions
	// A more complex implementation would require tracking file pattern order
	
	// Write all files
	fileCounter = g.writeFiles(&promptContent)

	// Write all questions in order
	for _, q := range g.Questions {
		promptContent.WriteString("\n" + q.Content + "\n")
	}

	return promptContent.String(), fileCounter, nil
}

// writeFiles writes file content to the builder and returns the count
func (g *Generator) writeFiles(builder *strings.Builder) int {
	fileCounter := 0

	for _, file := range g.Files {
		// Skip if not a regular file
		if !file.IsRegular {
			if !g.QuietMode {
				fmt.Fprintf(os.Stderr, "Warning: File '%s' is not a regular file. Skipping.\n", file.Path)
			}
			continue
		}

		// Skip if file is too large (unless force included)
		if !file.IsForced && file.Size > g.MaxFileSize {
			if !g.QuietMode {
				fmt.Fprintf(os.Stderr, "Info: Skipping file '%s' because it is too large (> 1MiB).\n", file.Path)
			}
			continue
		}

		// Skip if not a text file (unless force included)
		if !file.IsForced && !file.IsText {
			if !g.QuietMode {
				fmt.Fprintf(os.Stderr, "Info: Skipping file '%s' (non-text file).\n", file.Path)
			}
			continue
		}

		// Read file content
		content, err := os.ReadFile(file.Path)
		if err != nil {
			if !g.QuietMode {
				fmt.Fprintf(os.Stderr, "Warning: Failed to read content of '%s': %v. Skipping.\n", file.Path, err)
			}
			continue
		}

		// Add file content to prompt
		builder.WriteString("\n--- FILE: " + file.Path + " ---\n")
		builder.Write(content)
		builder.WriteString("\n--- END FILE: " + file.Path + " ---\n")

		fileCounter++
	}

	return fileCounter
}
