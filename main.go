package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/briossant/make-project-prompt/pkg/files"
	"github.com/briossant/make-project-prompt/pkg/prompt"
	"github.com/gobwas/glob"
)

// Command-line flags
var (
	includePatterns     multiStringFlag
	excludePatterns     multiStringFlag
	forceIncludePatterns multiStringFlag
	question            string
	useClipboard        bool
	showHelp            bool
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
	flag.Var(&forceIncludePatterns, "f", "Pattern (glob) to FORCE INCLUDE files/folders, bypassing file type and size checks. Can be used multiple times.")
	flag.StringVar(&question, "q", "[YOUR QUESTION HERE]", "Specifies the question for the LLM.")
	flag.BoolVar(&showHelp, "h", false, "Displays help message.")

	// Override usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-i <include_pattern>] [-e <exclude_pattern>] [-f <force_include_pattern>] [-q \"question\"] [-h]\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nExamples:")
		fmt.Fprintln(os.Stderr, "  make-project-prompt -i 'src/**/*.js' -e '**/__tests__/*' -q \"Refactor this React code to use Hooks.\"")
		fmt.Fprintln(os.Stderr, "  make-project-prompt -i '*.go' -f 'assets/*.bin' -q \"How can I optimize this binary asset loading?\"")
	}
}

// The functionality of these functions has been moved to the files and prompt packages:
// - listGitFiles -> pkg/files/files.go:ListGitFiles
// - getProjectTree -> pkg/files/files.go:GetProjectTree
// - isTextFile -> pkg/files/files.go:IsTextFile
// - generatePrompt -> pkg/prompt/prompt.go:Generator.Generate

// processFilesAndGeneratePrompt handles file processing and prompt generation
func processFilesAndGeneratePrompt() (string, int, error) {
	// Create file config
	fileConfig := files.Config{
		IncludePatterns:     includePatterns,
		ExcludePatterns:     excludePatterns,
		ForceIncludePatterns: forceIncludePatterns,
	}

	// List Git files with include/exclude/force patterns
	fileInfos, err := files.ListGitFiles(fileConfig)
	if err != nil {
		return "", 0, fmt.Errorf("failed to list Git files: %w", err)
	}

	if len(fileInfos) == 0 {
		if len(includePatterns) > 0 || len(forceIncludePatterns) > 0 {
			allPatterns := append([]string{}, includePatterns...)
			allPatterns = append(allPatterns, forceIncludePatterns...)
			return "", 0, fmt.Errorf("no files matched the specified patterns: %v\nTry using different patterns or check if the files exist", allPatterns)
		} else {
			return "", 0, fmt.Errorf("no files found in the Git repository. Make sure you have committed or staged some files")
		}
	}

	fmt.Printf("Found %d files matching the specified patterns.\n", len(fileInfos))

	// Generate prompt
	generator := prompt.NewGenerator(fileInfos, question)
	promptText, fileCount, err := generator.Generate()
	if err != nil {
		return "", 0, fmt.Errorf("failed to generate prompt: %w", err)
	}

	if fileCount == 0 {
		return "", 0, fmt.Errorf("no files were included in the prompt. All matched files were either binary, too large, or couldn't be read")
	}

	return promptText, fileCount, nil
}

// checkDependencies checks if all required dependencies are available
func checkDependencies() error {
	// Check if inside a Git repository
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	var stderr strings.Builder
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
	if len(forceIncludePatterns) > 0 {
		fmt.Println("Force inclusion patterns:", forceIncludePatterns)
	}
	fmt.Println("Question:", question)

	// Process files and generate prompt
	prompt, fileCount, err := processFilesAndGeneratePrompt()
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
