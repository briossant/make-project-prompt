package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
)

// Command-line flags
var (
	includePatterns multiStringFlag
	excludePatterns multiStringFlag
	question        string
	showHelp        bool
)

// multiStringFlag is a custom flag type that can be specified multiple times
type multiStringFlag []string

func (m *multiStringFlag) String() string {
	return strings.Join(*m, ", ")
}

func (m *multiStringFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

// Initialize flags
func init() {
	flag.Var(&includePatterns, "i", "Pattern (glob) to INCLUDE files/folders. Can be used multiple times.")
	flag.Var(&excludePatterns, "e", "Pattern (glob) to EXCLUDE files/folders. Can be used multiple times.")
	flag.StringVar(&question, "q", "[YOUR QUESTION HERE]", "Specifies the question for the LLM.")
	flag.BoolVar(&showHelp, "h", false, "Displays help message.")

	// Override usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-i <include_pattern>] [-e <exclude_pattern>] [-q \"question\"] [-h]\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nExample: make-project-prompt -i 'src/**/*.js' -e '**/__tests__/*' -q \"Refactor this React code to use Hooks.\"")
	}
}

// listGitFiles returns a list of files tracked by Git
func listGitFiles(includePatterns, excludePatterns []string) ([]string, error) {
	// Build git ls-files command
	args := []string{"ls-files", "-co", "--exclude-standard"}

	// Add -- separator and include patterns
	args = append(args, "--")
	if len(includePatterns) > 0 {
		args = append(args, includePatterns...)
	} else {
		// Default to all files if no include patterns specified
		args = append(args, "*")
	}

	// Uncomment for debugging
	// fmt.Printf("Debug - git command: git %s\n", strings.Join(args, " "))

	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("failed to run git ls-files: %s: %w", strings.TrimSpace(stderr.String()), err)
		}
		return nil, fmt.Errorf("failed to run git ls-files: %w", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return []string{}, nil
	}

	files := strings.Split(output, "\n")

	// Apply exclude patterns
	if len(excludePatterns) > 0 {
		var filteredFiles []string
		for _, file := range files {
			excluded := false
			for _, pattern := range excludePatterns {
				matched, err := filepath.Match(pattern, file)
				if err != nil {
					return nil, fmt.Errorf("invalid exclude pattern '%s': %w", pattern, err)
				}
				if matched {
					excluded = true
					break
				}
			}
			if !excluded {
				filteredFiles = append(filteredFiles, file)
			}
		}
		files = filteredFiles
	}

	return files, nil
}

// getProjectTree returns the output of the tree command
func getProjectTree() (string, error) {
	// Directories to ignore in tree output
	ignorePattern := ".git|node_modules|vendor|dist|build"

	cmd := exec.Command("tree", "-I", ignorePattern)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("failed to run tree command: %s: %w", strings.TrimSpace(stderr.String()), err)
		}
		return "", fmt.Errorf("failed to run tree command: %w", err)
	}

	return stdout.String(), nil
}

// isTextFile checks if a file is a text file based on its MIME type
func isTextFile(filePath string) bool {
	// Special case for Go module files
	if filepath.Base(filePath) == "go.mod" || filepath.Base(filePath) == "go.sum" {
		return true
	}

	// Get file extension
	ext := filepath.Ext(filePath)

	// Check MIME type based on extension
	mimeType := mime.TypeByExtension(ext)

	// If MIME type couldn't be determined by extension, use file command if available
	if mimeType == "" {
		// Check if 'file' command is available
		_, err := exec.LookPath("file")
		if err == nil {
			cmd := exec.Command("file", "-b", "--mime-type", filePath)
			var out bytes.Buffer
			cmd.Stdout = &out
			err := cmd.Run()
			if err == nil {
				mimeType = strings.TrimSpace(out.String())
			}
		} else {
			// If 'file' command is not available, make a best guess based on extension
			knownTextExtensions := map[string]bool{
				".txt": true, ".md": true, ".go": true, ".py": true, ".js": true, 
				".html": true, ".css": true, ".json": true, ".xml": true, ".yaml": true, 
				".yml": true, ".toml": true, ".sh": true, ".bash": true, ".c": true, 
				".cpp": true, ".h": true, ".hpp": true, ".java": true, ".rb": true, 
				".php": true, ".ts": true, ".jsx": true, ".tsx": true, ".vue": true,
				".rs": true, ".swift": true, ".kt": true, ".scala": true, ".clj": true,
				".ex": true, ".exs": true, ".erl": true, ".hs": true, ".lua": true,
				".pl": true, ".pm": true, ".r": true, ".dart": true, ".gradle": true,
				".ini": true, ".cfg": true, ".conf": true, ".properties": true,
				".gitignore": true, ".dockerignore": true, ".env": true, ".mod": true,
				".sum": true, ".lock": true,
			}

			if knownTextExtensions[strings.ToLower(ext)] {
				return true
			}

			// Try to read a small portion of the file to check if it's text
			f, err := os.Open(filePath)
			if err == nil {
				defer f.Close()

				// Read first 512 bytes
				buf := make([]byte, 512)
				n, err := f.Read(buf)
				if err == nil && n > 0 {
					// Check if the content appears to be text (no null bytes)
					for i := 0; i < n; i++ {
						if buf[i] == 0 {
							return false // Contains null byte, likely binary
						}
					}
					return true // No null bytes found, likely text
				}
			}
		}
	}

	// Check if it's a text file
	if strings.HasPrefix(mimeType, "text/") {
		return true
	}

	// Check for other common text-based formats
	textBasedTypes := []string{
		"application/json",
		"application/xml",
		"application/javascript",
		"application/x-sh",
		"application/x-shellscript",
		"application/x-python",
		"application/x-php",
		"application/x-ruby",
		"application/toml",
		"application/yaml",
	}

	for _, textType := range textBasedTypes {
		if strings.HasPrefix(mimeType, textType) {
			return true
		}
	}

	return false
}

// generatePrompt generates the LLM prompt
func generatePrompt(files []string, question string) (string, int, error) {
	var promptContent strings.Builder
	fileCounter := 0

	// Introduction
	promptContent.WriteString("Here is the context of my current project. Analyze the structure and content of the provided files to answer my question.\n\n")

	// Project structure via 'tree'
	promptContent.WriteString("--- PROJECT STRUCTURE (based on 'tree', may differ slightly from included files) ---\n")
	projectTree, err := getProjectTree()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to get project tree: %v\n", err)
		promptContent.WriteString("Error running tree command.\n")
	} else {
		promptContent.WriteString(projectTree)
	}
	promptContent.WriteString("\n")

	// Content of relevant files
	promptContent.WriteString("--- FILE CONTENT (based on git ls-files, respecting .gitignore and -i/-e options) ---\n")

	for _, file := range files {
		// Skip if not a regular file or not readable
		fileInfo, err := os.Stat(file)
		if err != nil || !fileInfo.Mode().IsRegular() {
			fmt.Fprintf(os.Stderr, "Warning: File '%s' is not a regular readable file. Skipping.\n", file)
			continue
		}

		// Skip if file is too large (> 1MB)
		if fileInfo.Size() > 1048576 {
			fmt.Fprintf(os.Stderr, "Info: Skipping file '%s' because it is too large (> 1MiB).\n", file)
			continue
		}

		// Skip if not a text file
		if !isTextFile(file) {
			fmt.Fprintf(os.Stderr, "Info: Skipping file '%s' (non-text file).\n", file)
			continue
		}

		// Read file content
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to read content of '%s': %v. Skipping.\n", file, err)
			continue
		}

		// Add file content to prompt
		promptContent.WriteString("\n--- FILE: " + file + " ---\n")
		promptContent.Write(content)
		promptContent.WriteString("\n--- END FILE: " + file + " ---\n")

		fileCounter++
	}

	promptContent.WriteString("\n--- END OF FILE CONTENT ---\n")

	// Final question
	promptContent.WriteString("\nBased on the context provided above, answer the following question:\n\n")
	promptContent.WriteString(question + "\n")

	return promptContent.String(), fileCounter, nil
}

// processFilesAndGeneratePrompt handles file processing and prompt generation
func processFilesAndGeneratePrompt(includePatterns, excludePatterns []string, question string) (string, int, error) {
	// List Git files with include/exclude patterns
	files, err := listGitFiles(includePatterns, excludePatterns)
	if err != nil {
		return "", 0, fmt.Errorf("failed to list Git files: %w", err)
	}

	if len(files) == 0 {
		if len(includePatterns) > 0 {
			return "", 0, fmt.Errorf("no files matched the specified include patterns: %v\nTry using different patterns or check if the files exist", includePatterns)
		} else {
			return "", 0, fmt.Errorf("no files found in the Git repository. Make sure you have committed or staged some files")
		}
	}

	fmt.Printf("Found %d files matching the specified patterns.\n", len(files))

	// Generate prompt
	prompt, fileCount, err := generatePrompt(files, question)
	if err != nil {
		return "", 0, fmt.Errorf("failed to generate prompt: %w", err)
	}

	if fileCount == 0 {
		return "", 0, fmt.Errorf("no text files were included in the prompt. All matched files were either binary, too large, or couldn't be read")
	}

	return prompt, fileCount, nil
}

// checkDependencies checks if all required dependencies are available
func checkDependencies() error {
	// Check if inside a Git repository
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("%s\nThis tool uses 'git ls-files' to list files and respect .gitignore", strings.TrimSpace(stderr.String()))
		} else {
			return fmt.Errorf("you are not inside a Git repository or git is not installed.\nThis tool uses 'git ls-files' to list files and respect .gitignore")
		}
	}

	// Check for required commands
	requiredCommands := []string{"git", "tree"}
	missingCommands := []string{}
	for _, cmdName := range requiredCommands {
		if _, err := exec.LookPath(cmdName); err != nil {
			missingCommands = append(missingCommands, cmdName)
		}
	}

	if len(missingCommands) > 0 {
		return fmt.Errorf("required command(s) not found: %s\nPlease install the missing command(s) to use this tool", strings.Join(missingCommands, ", "))
	}

	// Check for optional commands
	optionalCommands := []string{"file"}
	for _, cmdName := range optionalCommands {
		if _, err := exec.LookPath(cmdName); err != nil {
			fmt.Printf("Warning: Optional command '%s' not found. Some features may not work correctly.\n", cmdName)
		}
	}

	return nil
}

func main() {
	// Parse command-line flags
	flag.Parse()

	// Show help if requested
	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

	fmt.Println("Starting make-project-prompt (Go version)...")

	// Check dependencies
	if err := checkDependencies(); err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Display options
	fmt.Println("Inclusion patterns:", includePatterns)
	if len(excludePatterns) > 0 {
		fmt.Println("Exclusion patterns:", excludePatterns)
	}
	fmt.Println("Question:", question)

	// Process files and generate prompt
	prompt, fileCount, err := processFilesAndGeneratePrompt(includePatterns, excludePatterns, question)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Copy to clipboard
	if err := clipboard.WriteAll(prompt); err != nil {
		log.Fatalf("Error copying to clipboard: %v\nYou may need to install a clipboard manager or run this tool in a graphical environment.", err)
	}

	// User feedback
	fmt.Println("-------------------------------------")
	fmt.Println("Prompt generated and copied to clipboard!")
	fmt.Println("Number of files included:", fileCount)
	if question == "[YOUR QUESTION HERE]" {
		fmt.Println("NOTE: No question specified with -q. Remember to replace '[YOUR QUESTION HERE]'.")
	}
	fmt.Println("Paste (Ctrl+Shift+V or middle-click) into your LLM.")
	fmt.Println("-------------------------------------")
}
