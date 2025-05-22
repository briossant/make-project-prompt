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
    *   Automatically excludes binary files (based on MIME type).
    *   Excludes common directories like `.git`, `node_modules`, etc. from the `tree` output for clarity.
*   **Easy Integration:** Copies the generated prompt directly to the clipboard.
*   **Direct Question:** Allows you to specify the question for the LLM directly via the `-q` option.
*   **Cross-Platform:** Written in Go for better performance and cross-platform compatibility.
*   **Packaged with Nix Flakes:** Easy to run, install, and integrate into Nix/NixOS environments.

## Prerequisites

*   **Git:** The tool uses `git ls-files` to list files and respect `.gitignore`.
*   **Tree:** Used to generate the project structure visualization.
*   **File (optional):** Used to detect binary files. If not available, the tool will use heuristics.

For Nix users:
*   **Nix:** You must have [Nix installed](https://nixos.org/download.html) on your system.
*   **Flakes:** The [Nix Flakes](https://nixos.wiki/wiki/Flakes) feature must be enabled (this is often the case by default on recent installations; otherwise, follow the instructions in the Nix documentation).

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
Usage: make-project-prompt [-i <include_pattern>] [-e <exclude_pattern>] [-q "question"] [-h]

Options:
  -i <pattern> : Pattern (glob) to INCLUDE files/folders (default: '*' if no -i is provided).
                 Can be used multiple times (e.g., -i 'src/*' -i '*.py').
  -e <pattern> : Pattern (glob) to EXCLUDE files/folders (e.g., -e '*.log' -e 'tests/data/*').
                 Can be used multiple times.
  -q "question" : Specifies the question for the LLM.
  -h            : Displays this help message.

Example: make-project-prompt -i 'src/**/*.js' -e '**/__tests__/*' -q "Refactor this React code to use Hooks."
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
```

## Development

If you want to contribute or modify the code:

1.  Clone the repository: `git clone https://github.com/briossant/make-project-prompt.git`
2.  Enter the directory: `cd make-project-prompt`
3.  Make your changes to the Go code
4.  Build and test: `go build` and `./make-project-prompt`

For Nix users, you can start a Nix development shell:

```bash
nix develop .
```

In this shell, you will have access to the dependencies (`git`, `tree`, `file`), and the development tools needed for working on the project.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
