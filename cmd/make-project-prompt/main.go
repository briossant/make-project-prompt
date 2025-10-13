package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/briossant/make-project-prompt/pkg/config"
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
	dryRun               bool
	roleMessage          string
	extraContext         string
	lastWords            string
	aliasName            string
	listAliases          bool
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
	flag.Var(&includePatterns, "i", "Pattern (glob) to INCLUDE files/folders (default: '*' if no -i is provided).\n                 Can be used multiple times (e.g., -i 'src/*' -i '*.py').")
	flag.Var(&excludePatterns, "e", "Pattern (glob) to EXCLUDE files/folders (e.g., -e '*.log' -e 'tests/data/*').\n                 Can be used multiple times.")
	flag.Var(&forceIncludePatterns, "f", "Pattern (glob) to FORCE INCLUDE files/folders, bypassing file type and size checks.\n                 Can be used multiple times (e.g., -f 'assets/*.bin' -f 'data/*.dat').")
	flag.StringVar(&question, "q", "[YOUR QUESTION HERE]", "Specifies the question for the LLM.")
	flag.BoolVar(&useClipboard, "c", false, "Use clipboard content as the question for the LLM.")
	flag.StringVar(&questionFile, "qf", "", "Path to a file containing the question for the LLM.")
	flag.StringVar(&outputFile, "output", "", "Write prompt to a file instead of the clipboard.")
	flag.BoolVar(&useStdout, "stdout", false, "Write prompt to stdout instead of the clipboard.")
	flag.BoolVar(&quietMode, "quiet", false, "Suppress all non-essential output. Useful with --stdout or --output for scripting.")
	flag.BoolVar(&dryRun, "dry-run", false, "Perform a dry run. Lists the files that would be included in the prompt without generating it.")
	flag.BoolVar(&showHelp, "h", false, "Displays this help message.")
	flag.StringVar(&roleMessage, "role-message", "", "Pre-prompt message placed before all files.")
	flag.StringVar(&extraContext, "extra-context", "", "Pre-prompt message placed before the question.")
	flag.StringVar(&lastWords, "last-words", "", "Post-prompt message placed at the end of the whole message.")
	flag.StringVar(&aliasName, "a", "", "Use a predefined alias from config files.")
	flag.BoolVar(&listAliases, "list-aliases", false, "List all available aliases from config files.")

	// Override usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-i <include_pattern>] [-e <exclude_pattern>] [-f <force_include_pattern>] [-q \"question\"] [-c] [-qf file] [--role-message \"msg\"] [--extra-context \"msg\"] [--last-words \"msg\"] [-a \"alias\"] [--list-aliases] [--stdout] [--quiet] [--dry-run] [--output file] [-h]\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Options:")
		// Custom print defaults to match README style
		fmt.Fprintf(os.Stderr, "  -i <pattern> : %s\n", flag.Lookup("i").Usage)
		fmt.Fprintf(os.Stderr, "  -e <pattern> : %s\n", flag.Lookup("e").Usage)
		fmt.Fprintf(os.Stderr, "  -f <pattern> : %s\n", flag.Lookup("f").Usage)
		fmt.Fprintf(os.Stderr, "  -q \"question\" : %s\n", flag.Lookup("q").Usage)
		fmt.Fprintf(os.Stderr, "  -c            : %s\n", flag.Lookup("c").Usage)
		fmt.Fprintf(os.Stderr, "  -qf <file>    : %s\n", flag.Lookup("qf").Usage)
		fmt.Fprintf(os.Stderr, "  --role-message \"msg\" : %s\n", flag.Lookup("role-message").Usage)
		fmt.Fprintf(os.Stderr, "  --extra-context \"msg\" : %s\n", flag.Lookup("extra-context").Usage)
		fmt.Fprintf(os.Stderr, "  --last-words \"msg\" : %s\n", flag.Lookup("last-words").Usage)
		fmt.Fprintf(os.Stderr, "  -a \"alias\"    : %s\n", flag.Lookup("a").Usage)
		fmt.Fprintf(os.Stderr, "  --list-aliases : %s\n", flag.Lookup("list-aliases").Usage)
		fmt.Fprintf(os.Stderr, "  --stdout      : %s\n", flag.Lookup("stdout").Usage)
		fmt.Fprintf(os.Stderr, "  --quiet       : %s\n", flag.Lookup("quiet").Usage)
		fmt.Fprintf(os.Stderr, "  --dry-run     : %s\n", flag.Lookup("dry-run").Usage)
		fmt.Fprintf(os.Stderr, "  --output <file> : %s\n", flag.Lookup("output").Usage)
		fmt.Fprintf(os.Stderr, "  -h            : %s\n", flag.Lookup("h").Usage)

		fmt.Fprintln(os.Stderr, "\nNote: If multiple question input methods (-q, -c, -qf) are provided, the last one in the command line takes precedence.")
		fmt.Fprintln(os.Stderr, "      For non-combining options (all except -i, -e, -f), the last occurrence takes precedence.")
		fmt.Fprintln(os.Stderr, "\nAliases:")
		fmt.Fprintln(os.Stderr, "  Define aliases in .mpp.txt files using the format: alias_name: options")
		fmt.Fprintln(os.Stderr, "  Example: js_dev: --role-message \"You are a JS expert\" -i **.js")
		fmt.Fprintln(os.Stderr, "\nExamples:")
		fmt.Fprintln(os.Stderr, "  make-project-prompt -i 'src/**/*.js' -e '**/__tests__/*' -q \"Refactor this React code to use Hooks.\"")
		fmt.Fprintln(os.Stderr, "  make-project-prompt -i '*.go' -f 'assets/*.bin' -c")
		fmt.Fprintln(os.Stderr, "  make-project-prompt -i '*.py' -qf question.txt  # Read question from file")
		fmt.Fprintln(os.Stderr, "  make-project-prompt -i '*.py' -q \"Initial question\" -c  # Clipboard content will be used (last option wins)")
		fmt.Fprintln(os.Stderr, "  make-project-prompt -a js_dev -q \"Review this code\"  # Use the js_dev alias")
		fmt.Fprintln(os.Stderr, "  make-project-prompt --list-aliases  # List all available aliases")
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
	generator.RoleMessage = roleMessage
	generator.ExtraContext = extraContext
	generator.LastWords = lastWords
	promptText, fileCount, err := generator.Generate()
	if err != nil {
		return "", 0, fmt.Errorf("failed to generate prompt: %w", err)
	}

	if fileCount == 0 {
		return "", 0, fmt.Errorf("no files were included in the prompt. All matched files were either binary, too large, or couldn't be read")
	}

	return promptText, fileCount, nil
}

// expandAliasesInArgs expands any alias arguments in the command line
func expandAliasesInArgs(args []string) ([]string, error) {
	// Load aliases from config files
	cfg, err := config.LoadAliases()
	if err != nil {
		return nil, fmt.Errorf("failed to load aliases: %w", err)
	}

	var expanded []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		
		// Check if this is the -a or --a flag
		if arg == "-a" || arg == "--a" {
			// Get the alias name from next argument
			if i+1 >= len(args) {
				return nil, fmt.Errorf("flag -a requires an argument")
			}
			i++
			aliasName := args[i]
			
			// Look up the alias
			alias, exists := cfg.GetAlias(aliasName)
			if !exists {
				return nil, fmt.Errorf("alias '%s' not found", aliasName)
			}
			
			// Expand the alias options
			aliasArgs := config.ExpandAlias(alias.Options)
			expanded = append(expanded, aliasArgs...)
		} else {
			// Regular argument
			expanded = append(expanded, arg)
		}
	}
	
	return expanded, nil
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
			} else if currentFlag == "-dry-run" || currentFlag == "--dry-run" {
				dryRun = true
				continue
			} else if currentFlag == "-list-aliases" || currentFlag == "--list-aliases" {
				listAliases = true
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
				case "-role-message", "--role-message":
					roleMessage = value
				case "-extra-context", "--extra-context":
					extraContext = value
				case "-last-words", "--last-words":
					lastWords = value
				case "-a", "--a":
					aliasName = value
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
	requiredCommands := []string{"git"}
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
	optionalCommands := []string{"file", "tree"}
	for _, cmdName := range optionalCommands {
		if _, err := exec.LookPath(cmdName); err != nil {
			printInfo("Warning: Optional command '%s' not found. Some features may not work correctly.\n", cmdName)
			// Set an environment variable to indicate that the command is not available
			os.Setenv("MPP_NO_"+strings.ToUpper(cmdName), "1")
		}
	}

	return nil
}

// printInfo prints informational messages unless quiet mode is enabled or stdout is used
func printInfo(format string, a ...interface{}) {
	if !quietMode && !useStdout {
		fmt.Printf(format, a...)
	}
}

func main() {
	// Store original args before parsing to determine flag order later
	originalArgs := make([]string, len(os.Args))
	copy(originalArgs, os.Args)

	// Check if --list-aliases is requested before expanding aliases
	for _, arg := range os.Args[1:] {
		if arg == "-list-aliases" || arg == "--list-aliases" {
			listAliases = true
			break
		}
	}

	// Handle --list-aliases early
	if listAliases {
		cfg, err := config.LoadAliases()
		if err != nil {
			log.Fatalf("Error loading aliases: %v", err)
		}
		
		aliases := cfg.ListAliases()
		if len(aliases) == 0 {
			fmt.Println("No aliases found in .mpp.txt config files.")
			os.Exit(0)
		}
		
		fmt.Println("Available aliases:")
		for _, alias := range aliases {
			fmt.Printf("  %s: %s\n", alias.Name, alias.Options)
			fmt.Printf("    (defined in %s)\n", alias.Source)
		}
		os.Exit(0)
	}

	// Expand any aliases in the arguments
	expandedArgs, err := expandAliasesInArgs(os.Args[1:])
	if err != nil {
		log.Fatalf("Error expanding aliases: %v", err)
	}
	
	// Replace os.Args with expanded arguments for parsing
	os.Args = append([]string{os.Args[0]}, expandedArgs...)

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

	// If dry-run is requested, list files and exit.
	if dryRun {
		printInfo("--- Performing a dry run ---\n")
		fileConfig := files.Config{
			IncludePatterns:      includePatterns,
			ExcludePatterns:      excludePatterns,
			ForceIncludePatterns: forceIncludePatterns,
		}
		fileInfos, err := files.ListGitFiles(fileConfig)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}

		if len(fileInfos) == 0 {
			log.Fatalf("Dry run: No files would be included with the current filters.")
		}

		fmt.Println("The following files would be included in the prompt:")
		for _, info := range fileInfos {
			fmt.Println("- " + info.Path)
		}
		fmt.Printf("\nTotal files: %d\n", len(fileInfos))
		os.Exit(0) // Exit successfully after the dry run
	}

	// Process files and generate prompt
	prompt, fileCount, err := processFilesAndGeneratePrompt()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Handle output based on flags
	if useStdout {
		// Write to stdout and exit. This is critical for clean scripting output.
		fmt.Print(prompt)
		os.Exit(0)
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
