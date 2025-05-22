# Project Roadmap: make-project-prompt Refactor

This document outlines the plan for refactoring and enhancing the `make-project-prompt` tool. The primary goal is to move from the current bash script to a more robust and portable programming language, addressing current limitations and paving the way for future improvements.

## 1. Language and Technology Stack

**Recommendation:** Rewrite the project in **Go**. ✓

**Rationale:**
*   **Performance:** Go is a compiled language known for its speed and efficiency, which can be beneficial for a CLI tool.
*   **Single Binaries:** Go compiles to a single, statically-linked executable, making distribution and deployment extremely simple across different operating systems without requiring users to install a runtime.
*   **Concurrency:** Go has excellent built-in support for concurrency (goroutines and channels), which could be leveraged for future features involving parallel processing.
*   **Strong Standard Library:** Go's standard library is comprehensive, covering areas like file system operations, string manipulation, and HTTP.
*   **Readability & Maintainability:** While different from Python, Go's syntax is designed to be simple and clean, promoting readable and maintainable code.
*   **Cross-Platform Compilation:** Go makes it easy to cross-compile for different target operating systems and architectures from a single development machine.
*   **Testing:** Go has built-in support for testing (`go test`).
*   **Growing Ecosystem:** The Go ecosystem is mature and continues to grow.

**Alternative:** **Python** remains a viable option, particularly if rapid prototyping or a vast number of third-party libraries are primary concerns.

## 2. Core Refactoring Goals

*   **Replicate Existing Functionality:** ✓
    *   Integrate with Git to list tracked files (e.g., using `os/exec` to call git commands or a Go Git library). ✓
    *   Include/exclude files and folders based on glob patterns (e.g., using `path/filepath` package's `Match` or a third-party library). ✓
    *   Concatenate file contents with appropriate headers (filepath, tree structure). ✓
    *   Incorporate a user-defined question into the prompt. ✓
    *   Copy the generated prompt to the clipboard (e.g., using a cross-platform clipboard library like `github.com/atotto/clipboard`). ✓
*   **Improve Argument Parsing:** Implement a more user-friendly and robust CLI argument parsing system (e.g., using Go's `flag` package or a library like Cobra or Viper). ✓
*   **Enhance File Handling:** ✓
    *   More reliable file discovery and pattern matching. ✓
    *   Better handling of different file encodings. ✓
*   **Cross-Platform Compatibility:** Ensure the tool works seamlessly on major operating systems. ✓
*   **Robust Error Handling:** Provide clear, informative error messages for various failure scenarios (e.g., invalid patterns, file access issues, missing dependencies). ✓
*   **Code Structure and Modularity:** Organize the code into logical packages for better maintainability and testability. ✓

## 3. New Features and Enhancements

*   **Configuration File:** Allow users to define default include/exclude patterns, preferred LLM questions, or other settings in a configuration file (e.g., YAML, TOML, or JSON).
*   **Support for Different LLMs:** Potentially allow formatting the output for different LLMs or allow users to specify custom prompt templates.
*   **Token Counting/Estimation:** Add an option to estimate the token count of the generated prompt (useful for context window limits).
*   **Ignoring Binary Files:** Automatically detect and skip binary files more effectively.
*   **Interactive Mode:** An interactive mode to help users select files or refine the prompt.
*   **Plugin System:** (Long-term) Allow for extensions or plugins.

## 4. Development Process and Best Practices

*   **Version Control:** Continue using Git.
*   **Dependency Management:** Use Go Modules (`go mod`).
*   **Testing:**
    *   Implement unit tests for individual functions/packages using `go test`.
    *   Add integration tests for end-to-end CLI behavior.
*   **Linting and Formatting:** Use tools like `gofmt` (standard), `golint`, or `golangci-lint` to maintain code quality and consistency.
*   **Documentation:**
    *   Update `README.md` with new usage instructions. ✓
    *   Add inline code comments and docstrings.
*   **Continuous Integration (CI):** Set up CI (e.g., GitHub Actions) to automate testing, linting, and potentially builds.

## 5. Phased Rollout (Suggested)

*   **Phase 1: Core Rewrite (Go)** ✓
    *   Set up Go project structure (Go Modules). ✓
    *   Implement basic Git file listing. ✓
    *   Implement file inclusion/exclusion logic. ✓
    *   Implement prompt generation and clipboard copying. ✓
    *   Basic argument parsing. ✓
*   **Phase 2: Robustness and Usability** ✓
    *   Implement comprehensive error handling. ✓
    *   Write unit tests for core features. ✓
    *   Refine CLI arguments and user experience. ✓
    *   Improve cross-platform compatibility. ✓
*   **Phase 3: Feature Enhancement**
    *   Implement configuration file support.
    *   Begin adding new features from section 3 based on priority.
*   **Phase 4: Documentation and Release**
    *   Finalize documentation.
    *   Package the application for easier distribution (provide compiled binaries for different platforms).
    *   Update Nix flake integration to build the Go version instead of the bash script, updating dependencies and descriptions accordingly. ✓

This roadmap provides a high-level overview. Details can be refined as the project progresses.
