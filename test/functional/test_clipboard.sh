#!/bin/bash

# test_clipboard.sh
# Tests the clipboard functionality of make-project-prompt using xsel

set -e  # Exit on error

# Check if xsel is installed
if ! command -v xsel &> /dev/null; then
    echo "xsel is not installed. Please install it to run this test."
    exit 1
fi

# Create a temporary file for the clipboard content
CLIPBOARD_FILE=$(mktemp -t mpp-clipboard-XXXXXXXXXX)
trap "rm -f $CLIPBOARD_FILE" EXIT

# Function to get clipboard content
get_clipboard_content() {
    xsel -b -o > "$CLIPBOARD_FILE"
    echo "$CLIPBOARD_FILE"
}

# Function to check if clipboard contains a string
clipboard_contains() {
    local content_file=$1
    local search_string=$2
    grep -q "$search_string" "$content_file"
    return $?
}

# If this script is run directly, print usage
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    echo "This script is meant to be sourced by other scripts, not run directly."
    echo "Usage:"
    echo "  source $(basename "${BASH_SOURCE[0]}")"
    echo "  clipboard_file=\$(get_clipboard_content)"
    echo "  if clipboard_contains \"\$clipboard_file\" \"search string\"; then"
    echo "    echo \"Clipboard contains the search string\""
    echo "  else"
    echo "    echo \"Clipboard does not contain the search string\""
    echo "  fi"
    exit 1
fi