// Package prompt provides functionality for generating prompts for LLMs.
// It handles formatting file content and project structure into a prompt.
package prompt

import (
	"fmt"
	"os"
	"strings"

	"github.com/briossant/make-project-prompt/pkg/files"
)

// Generator handles prompt generation
type Generator struct {
	Files        []files.FileInfo
	Question     string
	MaxFileSize  int64
	QuietMode    bool
	RoleMessage  string
	ExtraContext string
	LastWords    string
}

// NewGenerator creates a new prompt generator
func NewGenerator(fileInfos []files.FileInfo, question string, quietMode bool) *Generator {
	return &Generator{
		Files:       fileInfos,
		Question:    question,
		MaxFileSize: 1048576, // 1MB default max file size
		QuietMode:   quietMode,
	}
}

// SetMaxFileSize sets the maximum file size for inclusion in the prompt
func (g *Generator) SetMaxFileSize(size int64) {
	g.MaxFileSize = size
}

// Generate creates the prompt with file content and project structure
func (g *Generator) Generate() (string, int, error) {
	var promptContent strings.Builder
	fileCounter := 0

	// Role message (if provided)
	if g.RoleMessage != "" {
		promptContent.WriteString(g.RoleMessage + "\n\n")
	}

	// Introduction
	promptContent.WriteString("Here is the context of my current project. Analyze the structure and content of the provided files to answer my question.\n\n")

	// Project structure via 'tree'
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

	// Content of relevant files
	promptContent.WriteString("--- FILE CONTENT (based on git ls-files, respecting .gitignore and -i/-e/-f options) ---\n")

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
		promptContent.WriteString("\n--- FILE: " + file.Path + " ---\n")
		promptContent.Write(content)
		promptContent.WriteString("\n--- END FILE: " + file.Path + " ---\n")

		fileCounter++
	}

	promptContent.WriteString("\n--- END OF FILE CONTENT ---\n")

	// Extra context (if provided)
	if g.ExtraContext != "" {
		promptContent.WriteString("\n" + g.ExtraContext + "\n")
	}

	// Final question
	promptContent.WriteString("\nBased on the context provided above, answer the following question:\n\n")
	promptContent.WriteString(g.Question + "\n")

	// Last words (if provided)
	if g.LastWords != "" {
		promptContent.WriteString("\n" + g.LastWords + "\n")
	}

	return promptContent.String(), fileCounter, nil
}
