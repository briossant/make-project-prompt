# JUNIE Project Guidelines

These are the essential rules for contributing to this project. Following them ensures our codebase remains clean, maintainable, and robust.

### 1. We Live and Breathe by Our Tests

This project has two test suites. You must understand which to use and when.

*   **Unit Tests (`pkg/`)**: These test individual functions and components in isolation. They are fast and focused.
    *   **When to use:** When you add or change a function in a `pkg/` package (e.g., a function in `pkg/files/files.go`).
    *   **How to run all:** `nix run .#test` (runs both unit and functional tests)

*   **Functional Tests (`test/functional/`)**: These test the entire application from end-to-end, simulating how a user would run it from the command line. They are slower but test the full integration of all components.
    *   **When to use:** When you add or change a CLI flag, modify the final prompt output, or alter any other user-facing behavior.
    *   **How to run:** `nix run .#test` (runs both unit and functional tests)

**The Golden Rule: Write a Failing Test First**

This is our non-negotiable workflow for any code change:

1.  **Choose the right test type:** Is this an isolated function (Unit) or a user-facing change (Functional)? Often, you may need to add both.
2.  **Write a failing test:** Add a test case that proves the new feature works or that a bug exists. Run it to confirm it fails as expected.
3.  **Write code to pass the test:** Implement the simplest, cleanest code required to make the test pass.
4.  **Run all tests:** After your change, run the *entire* test suite (`nix run .#test`) to ensure you haven't introduced any regressions.

### 2. Documentation is Not an Afterthought

Your code is not "done" until the documentation is updated.

-   If you add, change, or remove a user-facing feature (like a CLI flag), the `README.md` **must** be updated in the *same* pull request.
-   If you change the project's direction or goals, update the `ROADMAP.md`.

### 3. Keep the Code Clean and Idiomatic

We value simplicity and clarity, following standard Go practices.

-   **Formatting is automatic:** All code must be formatted with `gofmt`. Our linter enforces this and other quality rules. Run it before committing:
    ```bash
    # This will check and automatically fix issues
    nix run .#lint
    ```
-   **Handle errors explicitly:** Never ignore an error with `_`. If a function can fail, it must return an `error`. The caller must check it and either handle it or return it up the call stack.
-   **Keep it simple:** Favor Go's standard library. Only add a third-party dependency if it provides significant value and is well-maintained.
-   **Package organization matters:**
    -   `cmd/`: Contains the `main` package for the executable. This is the application entry point.
    -   `pkg/`: Contains all reusable library code. Code in `pkg` should never import code from `cmd`.
