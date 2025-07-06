# Functional Tests for make-project-prompt

This directory contains functional tests for the make-project-prompt tool. These tests verify that the tool works correctly from end to end, using a template Git repository created in `/tmp`.

## Test Files

- **setup_test_repo.sh**: Creates a template Git repository in `/tmp` for testing.
- **test_clipboard.sh**: Provides functions to test the clipboard functionality using `xsel`.
- **run_clipboard_tests.sh**: Runs tests for the clipboard functionality.
- **functional_test.go**: Contains Go tests for the basic functionality.

## Running the Tests

### Go Tests

To run the Go functional tests:

```bash
go test -v ./test/functional/...
```

### Bash Tests

To run the bash functional tests:

```bash
./test/functional/run_clipboard_tests.sh
```

Note: The bash functional tests require `xsel` to be installed.

## Test Repository Structure

The template Git repository created by `setup_test_repo.sh` has the following structure:

```
.
├── .gitignore           # Ignores binary files, build directory, and large files
├── binary_file.bin      # A binary file (ignored by Git)
├── build/               # A directory that is ignored by Git
│   └── output.txt
├── docs/                # Documentation files
│   ├── CONTRIBUTING.md
│   └── README.md
├── large_ignored.txt    # A large file that is ignored by Git
├── large_important.txt  # A large file that is included despite being large
└── src/                 # Source code
    ├── main/            # Main code files
    │   ├── app.go
    │   └── utils.go
    └── test/            # Test files
        └── app_test.go
```

This structure allows testing various features of make-project-prompt, including:

- Listing Git files
- Filtering files based on include/exclude patterns
- Forcing inclusion of specific files
- Handling binary files
- Handling large files
- Respecting .gitignore

## Adding New Tests

To add new tests:

1. For Go tests, add new test functions to `functional_test.go`.
2. For bash tests, add new test cases to `run_clipboard_tests.sh`.

When adding new tests, make sure they use the template Git repository created by `setup_test_repo.sh` to ensure a consistent testing environment.