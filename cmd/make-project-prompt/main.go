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
)

// Command-line flags
var (
	includePatterns      multiStringFlag
	excludePatterns      multiStringFlag
	forceIncludePatterns multiStringFlag
	question             string
	useClipboard         bool
	questionFile         string
	outputFile           string
	useStdout            bool
	quietMode            bool
	showHelp             bool
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
	flag.Var(&includePatterns, "i", "File path to INCLUDE. Glob patterns are expanded by your shell before being passed to this program. Can be used multiple times.")
	flag.Var(&excludePatterns, "e", "File path to EXCLUDE. Glob patterns are expanded by your shell before being passed to this program. Can be used multiple times.")
	flag.Var(&forceIncludePatterns, "f", "File path to FORCE INCLUDE, bypassing file type and size checks. Glob patterns are expanded by your shell before being passed to this program. Can be used multiple times.")
	flag.StringVar(&question, "q", "[YOUR QUESTION HERE]", "Specifies the question for the LLM.")
	flag.BoolVar(&useClipboard, "c", false, "Use clipboard content as the question for the LLM.")
	flag.StringVar(&questionFile, "qf", "", "Path to a file containing the question for the LLM.")
	flag.StringVar(&outputFile, "output", "", "Write prompt to a file instead of the clipboard (for testing).")
	flag.BoolVar(&useStdout, "stdout", false, "Write prompt to stdout instead of the clipboard.")
	flag.BoolVar(&quietMode, "quiet", false, "Suppress all non-essential output. Useful with --stdout or --output for scripting.")
	flag.BoolVar(&showHelp, "h", false, "Displays help message.")

	// Override usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-i <file_path>] [-e <file_path>] [-f <file_path>] [-q \"question\"] [-c] [-qf file] [--stdout] [--quiet] [--output file] [-h]\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nNote: If multiple question input methods (-q, -c, -qf) are provided, the last one in the command line takes precedence.")
		fmt.Fprintln(os.Stderr, "\nOutput options:")
		fmt.Fprintln(os.Stderr, "  By default, the prompt is copied to the clipboard.")
		fmt.Fprintln(os.Stderr, "  --output <file>: Write prompt to a file instead of the clipboard.")
		fmt.Fprintln(os.Stderr, "  --stdout: Write prompt to stdout instead of the clipboard.")
		fmt.Fprintln(os.Stderr, "  --quiet: Suppress all non-essential output. Useful with --stdout or --output for scripting.")
		fmt.Fprintln(os.Stderr, "\nExamples (with shell glob expansion):")
		fmt.Fprintln(os.Stderr, "  make-project-prompt -i src/**/*.js -e **/__tests__/* -q \"Refactor this React code to use Hooks.\"")
		fmt.Fprintln(os.Stderr, "  make-project-prompt -i *.go -f assets/*.bin -c")
		fmt.Fprintln(os.Stderr, "  make-project-prompt -i *.py -qf question.txt  # Read question from file")
		fmt.Fprintln(os.Stderr, "  make-project-prompt -i *.py -q \"Initial question\" -c  # Clipboard content will be used (last option wins)")
		fmt.Fprintln(os.Stderr, "  make-project-prompt -i *.py -q \"Question\" --stdout --quiet  # Output only the prompt to stdout")
		fmt.Fprintln(os.Stderr, "\nNote: Glob patterns (like *.go) are expanded by your shell before being passed to this program.")
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
		IncludePatterns:      includePatterns,
		ExcludePatterns:      excludePatterns,
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

	printInfo("Found %d files matching the specified patterns.\n", len(fileInfos))

	// Generate prompt
	generator := prompt.NewGenerator(fileInfos, question, quietMode)
	promptText, fileCount, err := generator.Generate()
	if err != nil {
		return "", 0, fmt.Errorf("failed to generate prompt: %w", err)
	}

	if fileCount == 0 {
		return "", 0, fmt.Errorf("no files were included in the prompt. All matched files were either binary, too large, or couldn't be read")
	}

	return promptText, fileCount, nil
}

// customParseArgs parses command-line arguments, collecting all arguments until a new flag is encountered
func customParseArgs() {
	args := os.Args[1:] // Skip the program name

	// Define a helper function to check if an argument is a flag
	isFlag := func(arg string) bool {
		return strings.HasPrefix(arg, "-")
	}

	var currentFlag string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Check if this is a flag
		if isFlag(arg) {
			// Process the flag
			currentFlag = arg

			// Handle boolean flags (like -h, -c, --stdout, --quiet)
			if currentFlag == "-h" || currentFlag == "--h" {
				showHelp = true
				continue
			} else if currentFlag == "-c" || currentFlag == "--c" {
				useClipboard = true
				continue
			} else if currentFlag == "-stdout" || currentFlag == "--stdout" {
				useStdout = true
				continue
			} else if currentFlag == "-quiet" || currentFlag == "--quiet" {
				quietMode = true
				continue
			}

			// For flags that take a value, get the next argument
			if i+1 < len(args) && !isFlag(args[i+1]) {
				value := args[i+1]
				i++ // Skip the value in the next iteration

				// Process the flag and its value
				switch currentFlag {
				case "-q", "--q":
					question = value
				case "-qf", "--qf":
					questionFile = value
				case "-output", "--output":
					outputFile = value
				case "-i", "--i":
					includePatterns = append(includePatterns, value)
				case "-e", "--e":
					excludePatterns = append(excludePatterns, value)
				case "-f", "--f":
					forceIncludePatterns = append(forceIncludePatterns, value)
				}
			}
		} else if currentFlag == "-i" || currentFlag == "--i" {
			// This is a non-flag argument following -i, add it to includePatterns
			includePatterns = append(includePatterns, arg)
		} else if currentFlag == "-e" || currentFlag == "--e" {
			// This is a non-flag argument following -e, add it to excludePatterns
			excludePatterns = append(excludePatterns, arg)
		} else if currentFlag == "-f" || currentFlag == "--f" {
			// This is a non-flag argument following -f, add it to forceIncludePatterns
			forceIncludePatterns = append(forceIncludePatterns, arg)
		}
	}
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
			printInfo("Warning: Optional command '%s' not found. Some features may not work correctly.\n", cmdName)
		}
	}

	return nil
}

// printInfo prints informational messages unless quiet mode is enabled
func printInfo(format string, a ...interface{}) {
	if !quietMode {
		fmt.Printf(format, a...)
	}
}

func main() {
	// Store original args before parsing to determine flag order later
	originalArgs := make([]string, len(os.Args))
	copy(originalArgs, os.Args)

	// Custom argument parsing to handle multiple arguments per flag
	customParseArgs()

	// Show help if requested
	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

	// Validate output options
	if useStdout && outputFile != "" {
		log.Fatalf("Error: Cannot use both --stdout and --output options at the same time.")
	}

	printInfo("Starting make-project-prompt (Go version)...\n")

	// Check dependencies
	if err := checkDependencies(); err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Handle question input strategy (last one wins)
	// Determine which flag was provided last by analyzing original args
	lastQIndex := -1
	lastCIndex := -1
	lastQFIndex := -1

	for i, arg := range originalArgs {
		switch arg {
		case "-q", "--q":
			lastQIndex = i
		case "-c", "--c":
			lastCIndex = i
		case "-qf", "--qf":
			lastQFIndex = i
		}
	}

	// Use the last provided method
	if lastQFIndex > lastQIndex && lastQFIndex > lastCIndex && questionFile != "" {
		// Read from file (it was the last option)
		fileContent, err := os.ReadFile(questionFile)
		if err != nil {
			log.Fatalf("Error reading from file %s: %v\nMake sure the file exists and is readable.", questionFile, err)
		}
		if len(fileContent) == 0 {
			log.Fatalf("Error: File %s is empty. Please provide a file with content.", questionFile)
		}
		question = string(fileContent)
		printInfo("Using question from file %s (last option wins).\n", questionFile)
	} else if lastCIndex > lastQIndex && lastCIndex > lastQFIndex && useClipboard {
		// Read from clipboard (it was the last option)
		clipContent, err := clipboard.ReadAll()
		if err != nil {
			log.Fatalf("Error reading from clipboard: %v\nMake sure you have content in your clipboard.", err)
		}
		if clipContent == "" {
			log.Fatalf("Error: Clipboard is empty. Please copy your question to the clipboard first.")
		}
		question = clipContent
		printInfo("Using question from clipboard (last option wins).\n")
	} else if lastQIndex > lastCIndex && lastQIndex > lastQFIndex && question != "[YOUR QUESTION HERE]" {
		// Using question from -q flag (it was the last option)
		printInfo("Using question from command line (last option wins).\n")
	} else if questionFile != "" {
		// Only file flag was provided
		fileContent, err := os.ReadFile(questionFile)
		if err != nil {
			log.Fatalf("Error reading from file %s: %v\nMake sure the file exists and is readable.", questionFile, err)
		}
		if len(fileContent) == 0 {
			log.Fatalf("Error: File %s is empty. Please provide a file with content.", questionFile)
		}
		question = string(fileContent)
		printInfo("Using question from file %s.\n", questionFile)
	} else if useClipboard {
		// Only clipboard flag was provided
		clipContent, err := clipboard.ReadAll()
		if err != nil {
			log.Fatalf("Error reading from clipboard: %v\nMake sure you have content in your clipboard.", err)
		}
		if clipContent == "" {
			log.Fatalf("Error: Clipboard is empty. Please copy your question to the clipboard first.")
		}
		question = clipContent
		printInfo("Using question from clipboard.\n")
	}

	// Display options
	printInfo("Inclusion patterns: %v\n", includePatterns)
	if len(excludePatterns) > 0 {
		printInfo("Exclusion patterns: %v\n", excludePatterns)
	}
	if len(forceIncludePatterns) > 0 {
		printInfo("Force inclusion patterns: %v\n", forceIncludePatterns)
	}
	printInfo("Question: %s\n", question)

	// Process files and generate prompt
	prompt, fileCount, err := processFilesAndGeneratePrompt()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Handle output based on flags
	if useStdout {
		// Write to stdout
		fmt.Print(prompt)
	} else if outputFile != "" {
		// Write to file
		err = os.WriteFile(outputFile, []byte(prompt), 0644)
		if err != nil {
			log.Fatalf("Error writing to output file: %v", err)
		}
		printInfo("-------------------------------------\n")
		printInfo("Prompt generated and written to %s!\n", outputFile)
	} else {
		// Copy to clipboard (default)
		if err := clipboard.WriteAll(prompt); err != nil {
			log.Fatalf("Error copying to clipboard: %v\nYou may need to install a clipboard manager or run this tool in a graphical environment.", err)
		}
		printInfo("-------------------------------------\n")
		printInfo("Prompt generated and copied to clipboard!\n")
	}

	// User feedback
	printInfo("Number of files included: %d\n", fileCount)
	if question == "[YOUR QUESTION HERE]" {
		printInfo("NOTE: No question specified with -q. Remember to replace '[YOUR QUESTION HERE]'.\n")
	}
	if !useStdout {
		printInfo("Paste (Ctrl+Shift+V or middle-click) into your LLM.\n")
	}
	printInfo("-------------------------------------\n")
}
