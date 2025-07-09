# JUNIE Project Guidelines

**Attention Junie:** These are the mandatory, non-negotiable rules for working on this project. You MUST follow these instructions precisely to ensure correctness and consistency with the CI pipeline.

### 1. The Single Source of Truth for Testing

This project uses Nix Flakes to create a perfectly reproducible environment. All testing and validation **MUST** be performed through the Nix commands defined in `flake.nix`.

**The Universal Test Command**

To run **ALL** tests (both Unit and Functional), you **MUST** use the following command from the project root:

```bash
nix run .#test
```

**Explicit Prohibitions:**

*   You **MUST NOT** run `go test` directly (e.g., `go test -v ./...` or `go test ./test/functional/...`).
*   **Reason:** This is critical because our functional tests depend on the specific shell environment and dependencies managed by Nix. Running `go test` directly will cause these tests to fail due to incorrect shell behavior (like glob expansion) and missing dependencies. The command `nix run .#test` guarantees the correct environment.

**Test Suites (For Your Information Only):**

*   **Unit Tests (`pkg/`)**: These test individual functions. You still run them with `nix run .#test`.
*   **Functional Tests (`test/functional/`)**: These test the full application. You still run them with `nix run .#test`.

**The Golden Workflow:**

1.  **Write a failing test:** Add a unit or functional test case that proves the bug exists or the feature is missing.
2.  **Confirm it fails:** Run `nix run .#test` and verify that the new test fails as expected.
3.  **Write code to pass the test:** Implement the simplest code required to make the test pass.
4.  **Run all tests again:** After your change, run `nix run .#test` to ensure all tests now pass and no regressions were introduced.

### 2. Documentation is Not an Afterthought

Your code is not "done" until the documentation is updated.

-   If you add, change, or remove a user-facing feature (like a CLI flag), the `README.md` **MUST** be updated in the *same* pull request.
-   If you change the project's direction or goals, update the `ROADMAP.md`.

### 3. Keep the Code Clean and Idiomatic

We value simplicity and clarity, following standard Go practices.

-   **Formatting is Mandatory and Automated:** All code must be formatted and linted according to project standards. You **MUST** run the linting command before committing. This command not only checks for issues but automatically fixes many of them.

    ```bash
    # This is the ONLY command to use for linting and formatting.
    nix run .#lint
    ```

-   **Handle errors explicitly:** Never ignore an error with `_`. If a function can fail, it must return an `error`. The caller must check it and either handle it or return it up the call stack.
-   **Keep it simple:** Favor Go's standard library. Only add a third-party dependency if it provides significant value and is well-maintained.
-   **Package organization matters:**
    -   `cmd/`: Contains the `main` package for the executable. This is the application entry point.
    -   `pkg/`: Contains all reusable library code. Code in `pkg` should never import code from `cmd`.
