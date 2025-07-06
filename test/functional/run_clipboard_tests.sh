#!/bin/bash

# run_clipboard_tests.sh
# Runs tests for the clipboard functionality of make-project-prompt

set -e  # Exit on error

# Get the directory of this script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Source the clipboard test functions
source "$SCRIPT_DIR/test_clipboard.sh"

# Run the setup script to create a test repository
echo "Setting up test repository..."
TEST_REPO=$("$SCRIPT_DIR/setup_test_repo.sh" | tail -n 1)
echo "Test repository created at: $TEST_REPO"

# Change to the test repository
cd "$TEST_REPO"

# Function to run a test
run_test() {
    local test_name=$1
    local mpp_args=$2
    local expected_content=$3
    
    echo "Running test: $test_name"
    echo "Command: go run $SCRIPT_DIR/../../cmd/make-project-prompt/main.go $mpp_args"
    
    # Run make-project-prompt
    go run "$SCRIPT_DIR/../../cmd/make-project-prompt/main.go" $mpp_args
    
    # Get clipboard content
    clipboard_file=$(get_clipboard_content)
    
    # Check if clipboard contains the expected content
    if clipboard_contains "$clipboard_file" "$expected_content"; then
        echo "✅ Test passed: Clipboard contains '$expected_content'"
    else
        echo "❌ Test failed: Clipboard does not contain '$expected_content'"
        exit 1
    fi
    
    echo ""
}

# Test 1: Basic functionality
run_test "Basic functionality" "-q 'Test question'" "Test question"

# Test 2: Include patterns
run_test "Include patterns" "-i 'src/main/*.go' -q 'Test include patterns'" "Test include patterns"

# Test 3: Exclude patterns
run_test "Exclude patterns" "-e 'src/test/*' -q 'Test exclude patterns'" "Test exclude patterns"

# Test 4: Force include patterns
run_test "Force include patterns" "-f 'binary_file.bin' -q 'Test force include patterns'" "Test force include patterns"

# Test 5: Question from file
echo "Creating question file..."
echo "Question from file" > "$TEST_REPO/question.txt"
run_test "Question from file" "-qf '$TEST_REPO/question.txt'" "Question from file"

# Test 6: Combined options
run_test "Combined options" "-i 'src/main/*.go' -e 'src/main/utils.go' -f 'binary_file.bin' -q 'Test combined options'" "Test combined options"

# Clean up
echo "Cleaning up test repository..."
rm -rf "$TEST_REPO"

echo "All tests passed! 🎉"