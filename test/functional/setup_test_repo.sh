#!/bin/bash

# setup_test_repo.sh
# Creates a template Git repository in /tmp for testing make-project-prompt

set -e  # Exit on error

# Create a unique temporary directory
REPO_DIR=$(mktemp -d -t mpp-test-repo-XXXXXXXXXX)
echo "Creating test repository in $REPO_DIR"

# Initialize Git repository
cd "$REPO_DIR"
git init
git config --local user.name "Test User"
git config --local user.email "test@example.com"

# Create a .gitignore file
cat > .gitignore << EOF
# Ignore binary files
*.bin
# Ignore build directory
build/
# Ignore large files
large_*.txt
# But don't ignore this specific large file
!large_important.txt
EOF

# Create various text files
mkdir -p src/main
mkdir -p src/test
mkdir -p docs
mkdir -p build

# Main code files
cat > src/main/app.go << EOF
package main

import "fmt"

func main() {
    fmt.Println("Hello, world!")
}

func Add(a, b int) int {
    return a + b
}
EOF

cat > src/main/utils.go << EOF
package main

func Multiply(a, b int) int {
    return a * b
}
EOF

# Test files
cat > src/test/app_test.go << EOF
package main

import "testing"

func TestAdd(t *testing.T) {
    if Add(2, 3) != 5 {
        t.Error("Expected 2 + 3 to equal 5")
    }
}
EOF

# Documentation
cat > docs/README.md << EOF
# Test Project

This is a test project for make-project-prompt functional tests.

## Features

- Feature 1
- Feature 2
EOF

cat > docs/CONTRIBUTING.md << EOF
# Contributing

Please follow these guidelines when contributing to this project.

1. Fork the repository
2. Create a feature branch
3. Submit a pull request
EOF

# Create a binary file
dd if=/dev/urandom of=binary_file.bin bs=1024 count=10

# Create a large text file that should be ignored
yes "This is a large file that should be ignored." | head -n 10000 > large_ignored.txt

# Create a large text file that should be included (via !large_important.txt in .gitignore)
yes "This is a large file that should be included." | head -n 10000 > large_important.txt

# Create a file in the build directory (should be ignored)
echo "This file should be ignored" > build/output.txt

# Add all files to Git (except those in .gitignore)
git add .
git commit -m "Initial commit"

# Print the repository path
echo "$REPO_DIR"