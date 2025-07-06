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
*   **Easy Integration:** Copies the generated prompt directly to the clipboard.
*   **Flexible Question Input:** 
    *   Specify the question directly via the `-q` option.
    *   Use content from your clipboard via the `-c` option.
    *   Read question from a file via the `-qf` option.
    *   Multiple input methods supported with "last one wins" precedence.
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

## Testing

The project includes both unit tests and functional tests:

### Unit Tests

Unit tests are located alongside the code they test, following Go conventions. They test individual components of the application.

To run the unit tests:

```bash
go test -v ./...
```

### Functional Tests

Functional tests are located in the `test/functional` directory. They test the entire application workflow from end to end, using a template Git repository created in `/tmp`.

The functional tests include:

1. **Go tests** (`test/functional/functional_test.go`): Test the basic functionality of the application.
2. **Bash tests** (`test/functional/run_clipboard_tests.sh`): Test the clipboard functionality using `xsel`.

To run the Go functional tests:

```bash
go test -v ./test/functional/...
```

To run the bash functional tests:

```bash
./test/functional/run_clipboard_tests.sh
```

Note: The bash functional tests require `xsel` to be installed.

## Installation and Usage

### Go Installation

If you have Go installed, you can install the tool directly:

```bash
go install github.com/briossant/make-project-prompt@latest
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
Usage: make-project-prompt [-i <include_pattern>] [-e <exclude_pattern>] [-f <force_include_pattern>] [-q "question"] [-c] [-qf file] [-h]

Options:
  -i <pattern> : Pattern (glob) to INCLUDE files/folders (default: '*' if no -i is provided).
                 Can be used multiple times (e.g., -i 'src/*' -i '*.py').
  -e <pattern> : Pattern (glob) to EXCLUDE files/folders (e.g., -e '*.log' -e 'tests/data/*').
                 Can be used multiple times.
  -f <pattern> : Pattern (glob) to FORCE INCLUDE files/folders, bypassing file type and size checks.
                 Can be used multiple times (e.g., -f 'assets/*.bin' -f 'data/*.dat').
  -q "question" : Specifies the question for the LLM.
  -c            : Use clipboard content as the question for the LLM.
  -qf <file>    : Path to a file containing the question for the LLM.
  -h            : Displays this help message.

Note: If multiple question input methods (-q, -c, -qf) are provided, the last one in the command line takes precedence.

Examples:
  make-project-prompt -i 'src/**/*.js' -e '**/__tests__/*' -q "Refactor this React code to use Hooks."
  make-project-prompt -i '*.go' -f 'assets/*.bin' -c
  make-project-prompt -i '*.py' -qf question.txt  # Read question from file
  make-project-prompt -i '*.py' -q "Initial question" -c  # Clipboard content will be used (last option wins)
```

## Usage Examples

(Make sure you are at the root of your Git project)

```bash
# Generate a prompt for the entire project (tracked by Git)
mpp

# Generate a prompt, include only .py files, and ask a question
mpp -i '*.py' -q "Explain the role of the main class in this Python project."

# Generate a prompt, include files in 'src' and 'include', exclude test files
mpp -i 'src/*' -i 'include/*' -e '*_test.go' -q "Check if there are any concurrency issues in this Go code."

# Generate a prompt, include Go files and force include binary files in the assets directory
mpp -i '*.go' -f 'assets/**/*.bin' -q "How can I optimize loading these binary assets in my Go application?"

# Generate a prompt using the question from your clipboard
mpp -c

# Generate a prompt using a question from a file
mpp -i '*.go' -qf path/to/question.txt

# Generate a prompt with multiple question input methods (clipboard content will be used)
mpp -q "This question will be overridden" -c
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

This will run all tests in the project, including:
- File filtering and pattern matching
- Text file detection
- Project tree generation
- Prompt generation
- Question input methods (command-line, clipboard, file)

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
