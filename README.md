# Make Project Prompt (mpp)

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

`make-project-prompt` (or its alias `mpp`) is a simple command-line tool designed to generate a contextual prompt for Large Language Models (LLMs). It analyzes your current project, extracts the directory structure and content of relevant files, and copies everything to your clipboard, ready to be pasted into your LLM interface.

This allows you to provide rich and precise context to the LLM for questions regarding your codebase.

## Features

*   **Project Structure:** Includes the output of the `tree` command to show the organization of files and folders.
*   **File Content:** Retrieves the content of text files in your project.
*   **Respects `.gitignore`:** Uses `git ls-files` to list files, automatically ignoring those specified in your `.gitignore` and other standard Git ignore mechanisms.
*   **Advanced Filtering:**
    *   Selectively includes/excludes files/folders using glob patterns (`-i` and `-e` options).
    *   Force include files/folders regardless of type or size (`-f` option).
    *   Automatically excludes binary files (based on MIME type).
    *   Excludes common directories like `.git`, `node_modules`, etc. from the `tree` output for clarity.
*   **Flexible Output Options:**
    *   Copies the generated prompt directly to the clipboard (default).
    *   Write to a file with the `--output` option.
    *   Output directly to stdout with the `--stdout` option.
    *   Suppress non-essential output with the `--quiet` option for easier scripting and automation.
    *   Perform a dry run with the `--dry-run` option to see which files would be included without generating the prompt.
*   **Question Accumulation:**
    *   Specify questions/text directly via the `-q` option (can be used multiple times - all accumulate).
    *   Use content from your clipboard via the `-c` option.
    *   Read questions from files via the `-qf` option (can be used multiple times).
    *   All question sources accumulate and appear in the order specified.
*   **Raw Mode (`--raw`):**
    *   Removes all pre-written messages for minimal output.
    *   Supports full argument order-based positioning - questions and files appear in the exact order they're specified.
    *   Perfect for crafting custom prompts with precise control.
*   **Alias System:**
    *   Define reusable command aliases in `.mpp.txt` configuration files.
    *   Aliases are loaded recursively from the current directory up to the root.
    *   Use aliases with the `-a` flag to avoid repetitive typing.
    *   List all available aliases with `--list-aliases`.
*   **Cross-Platform:** Written in Go for better performance and cross-platform compatibility.
*   **Packaged with Nix Flakes:** Easy to run, install, and integrate into Nix/NixOS environments.

## Prerequisites

*   **Git:** The tool uses `git ls-files` to list files and respect `.gitignore`.
*   **Tree:** Used to generate the project structure visualization.
*   **File (optional):** Used to detect binary files. If not available, the tool will use heuristics.
*   **xsel (optional):** Used for clipboard operations in Linux. Required for running the functional tests.

For Nix users:
*   **Nix:** You must have [Nix installed](https://nixos.org/download.html) on your system.
*   **Flakes:** The [Nix Flakes](https://nixos.wiki/wiki/Flakes) feature must be enabled (this is often the case by default on recent installations; otherwise, follow the instructions in the Nix documentation).

## Installation and Usage

### Go Installation

If you have Go installed, you can install the tool directly:

```bash
go install github.com/briossant/make-project-prompt/cmd/make-project-prompt@latest
```

This will install the `make-project-prompt` command in your `$GOPATH/bin` directory.

### Nix Installation

You can use `make-project-prompt` in several ways thanks to Nix Flakes:

**1. One-Time Execution (without permanent installation):**

This is ideal for testing or using the tool on an ad-hoc basis. From the root of your Git project:

```bash
nix run github:briossant/make-project-prompt -- [options...]
```

Example:

```bash
# Generates a prompt for all JS files in src/ and asks a question
nix run github:briossant/make-project-prompt -- -i 'src/**/*.js' -q "Can you refactor this React component?"
```

**2. Installation in your User Profile:**

This makes the `make-project-prompt` and `mpp` commands available globally for your user.

```bash
nix profile install github:briossant/make-project-prompt
```

Then, you can simply run it from the root of your Git project:

```bash
# Using the full name
make-project-prompt -q "Describe the architecture of this project."

# Using the alias
mpp -i '*.py' -e 'tests/*'
```

To uninstall: `nix profile remove github:briossant/make-project-prompt` (the exact name may vary; use `nix profile list` to check).

**3. Usage in a Temporary Shell:**

If you just want to have the command available temporarily in your current shell:

```bash
nix shell github:briossant/make-project-prompt
# Now the commands are available:
mpp -h
make-project-prompt
# exit to leave the temporary shell
```

**4. Integration into NixOS or Home Manager:**

Add the flake as an input to your NixOS/Home Manager configuration:

```nix
# flake.nix (of your system/home configuration)
{
  inputs = {
    # ... other inputs ...
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable"; # or another branch
    make-project-prompt.url = "github:briossant/make-project-prompt";
  };

  outputs = { self, nixpkgs, make-project-prompt, ... }@inputs: {
    # Example for NixOS
    nixosConfigurations.yourhostname = nixpkgs.lib.nixosSystem {
      # ...
      environment.systemPackages = with pkgs; [
        # ... other packages ...
        make-project-prompt.packages.${system}.default # <= Add the tool here
      ];
      # ...
    };

    # Example for Home Manager
    homeConfigurations."youruser@yourhostname" = home-manager.lib.homeManagerConfiguration {
       # ...
       home.packages = with pkgs; [
         # ... other packages ...
         make-project-prompt.packages.${system}.default # <= Add the tool here
       ];
       # ...
    };
  };
}
```

After rebuilding (`nixos-rebuild switch` or `home-manager switch`), the commands `make-project-prompt` and `mpp` will be available.

## Command Options

```bash
Usage: make-project-prompt [-i <include_pattern>] [-e <exclude_pattern>] [-f <force_include_pattern>] [-q "text"] [-c] [-qf file] [--raw] [-a "alias"] [--list-aliases] [--stdout] [--quiet] [--dry-run] [--output file] [-h]

Options:
  -i <pattern> : Pattern (glob) to INCLUDE files/folders (default: '*' if no -i is provided).
                 Can be used multiple times (e.g., -i 'src/*' -i '*.py').
                 Supports glob patterns including ** for recursive matching.
  -e <pattern> : Pattern (glob) to EXCLUDE files/folders (e.g., -e '*.log' -e 'tests/data/*').
                 Can be used multiple times.
  -f <pattern> : Pattern (glob) to FORCE INCLUDE files/folders, bypassing file type and size checks.
                 Can be used multiple times (e.g., -f 'assets/*.bin' -f 'data/*.dat').
  -q "text"    : Specifies a question or text for the LLM. Can be used multiple times - all questions will be included.
  -c            : Use clipboard content as a question for the LLM.
  -qf <file>    : Path to a file containing a question for the LLM. Can be used multiple times.
  --raw         : Raw mode: remove pre-written messages and use argument order for positioning.
  -a "alias"    : Use a predefined alias from config files (.mpp.txt).
  --list-aliases : List all available aliases from config files.
  --stdout      : Write prompt to stdout instead of the clipboard.
  --quiet       : Suppress all non-essential output. Useful with --stdout or --output for scripting.
  --dry-run     : Perform a dry run. Lists the files that would be included in the prompt without generating it.
  --output <file> : Write prompt to a file instead of the clipboard.
  -h            : Displays this help message.

Note: Multiple -q and -qf options accumulate (all are included in order).
      In --raw mode, argument order determines positioning in the output.
      For non-combining options, the last occurrence takes precedence.

Examples:
  make-project-prompt -i 'src/**/*.js' -e '**/__tests__/*' -q "Refactor this React code to use Hooks."
  make-project-prompt -i '*.go' -q "First question" -q "Second question"  # Both questions included
  make-project-prompt --raw -q "Header" -i '*.py' -q "Footer"  # Raw mode with positioning
  make-project-prompt -i '*.py' -qf question.txt  # Read question from file
  make-project-prompt -a js_dev -q "Review this code"  # Use the js_dev alias
  make-project-prompt --list-aliases  # List all available aliases
```

## Alias Configuration

You can define reusable command aliases in `.mpp.txt` files. These files are loaded recursively from the current directory up to the file system root.

### Configuration File Format

Create a `.mpp.txt` file in your project root or any parent directory:

```
# Comments start with #
alias_name: options

# Example aliases
js_dev: -i src/**/*.js -e **/__tests__/*
go_files: -i **/*.go -e **/*_test.go
python_review: -i **/*.py -q "Focus on code quality and best practices"
quick_readme: -i README.md -i CONTRIBUTING.md -q "Summarize this project"
```

### Using Aliases

```bash
# Use an alias
mpp -a js_dev -q "Review this code"

# List all available aliases
mpp --list-aliases

# Combine an alias with additional options (options combine or override)
mpp -a go_files -i cmd/**/*.go -q "Explain the command structure"
```

### Alias Precedence

*   Config files are loaded from the current directory up to the root.
*   If the same alias name appears in multiple config files, the first one encountered (closest to current directory) takes precedence.
*   A warning is displayed when duplicate aliases are found.

## Usage Examples

(Make sure you are at the root of your Git project)

```bash
# Generate a prompt for the entire project (tracked by Git)
mpp

# Generate a prompt, include only .py files, and ask a question
mpp -i '*.py' -q "Explain the role of the main class in this Python project."

# Ask multiple questions that accumulate
mpp -i '*.go' -q "What does this code do?" -q "Are there any bugs?"

# Generate a prompt in raw mode with custom positioning
mpp --raw -q "Context: This is a web server." -i 'server/*.go' -q "Question: How can I improve performance?"

# Generate a prompt, include files in 'src' and 'include', exclude test files
mpp -i 'src/*' -i 'include/*' -e '*_test.go' -q "Check if there are any concurrency issues in this Go code."

# Generate a prompt, include Go files and force include binary files in the assets directory
mpp -i '*.go' -f 'assets/**/*.bin' -q "How can I optimize loading these binary assets in my Go application?"

# Generate a prompt using the question from your clipboard
mpp -c

# Generate a prompt using a question from a file
mpp -i '*.go' -qf path/to/question.txt

# Mix multiple question sources (all accumulate)
mpp -i '*.py' -q "Question 1" -qf questions.txt -q "Question 3"

# Perform a dry run to see which files would be included without generating the prompt
mpp -i '*.go' --dry-run

# Use an alias for common workflows
mpp -a python_review -q "Check for potential bugs"
```

## Development

If you want to contribute or modify the code:

1.  Clone the repository: `git clone https://github.com/briossant/make-project-prompt.git`
2.  Enter the directory: `cd make-project-prompt`
3.  Make your changes to the Go code
4.  Build and test: `go build` and `./make-project-prompt`

### Testing

The project includes comprehensive tests for all functionalities. To run the tests:

```bash
go test -v ./...
```

For Nix users, you can also run the tests using the flake output:

```bash
nix run .#test
```

This will run both unit tests and functional tests in the project, including:
- File filtering and pattern matching
- Text file detection
- Project tree generation
- Prompt generation
- Question input methods (command-line, clipboard, file)
- End-to-end functional tests with a test Git repository

#### Linting

The project uses `golangci-lint` to ensure code quality. The same linter configuration from the CI workflow can be run locally using Nix:

```bash
nix run .#lint
```

This command not only checks for linting errors but also automatically fixes them when possible, making it easy to maintain code quality with minimal effort. Make sure to run this command from the root of your project directory to ensure it can properly access and modify your source files.

The project also includes a GitHub Actions workflow that automatically runs tests on push and pull requests. This ensures that all changes are tested before being merged.

### Development Environment

For Nix users, you can start a Nix development shell:

```bash
nix develop .
```

In this shell, you will have access to the dependencies (`git`, `tree`, `file`), and the development tools needed for working on the project.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
