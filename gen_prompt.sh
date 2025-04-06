#!/bin/bash

# Script to generate an LLM prompt describing the current project
# and copy it to the clipboard (via xsel).
#
# Options:
#   -i <pattern> : Glob pattern to INCLUDE files/folders (default: '*')
#                  Can be used multiple times (implicitly combined).
#                  The pattern is passed to 'git ls-files'.
#   -e <pattern> : Glob pattern to EXCLUDE files/folders.
#                  Can be used multiple times.
#                  Uses the --exclude option of 'git ls-files'.
#   -q "question" : Specifies the question to ask the LLM.
#   -h            : Displays this help message.
#
# PREREQUISITES: git, tree, xsel, file

# --- Default Variables ---
INCLUDE_PATTERNS=() # Array for include patterns
EXCLUDE_OPTS=()     # Array for --exclude options
LLM_QUESTION="[YOUR QUESTION HERE]" # Default placeholder question
TREE_IGNORE_DIRS=".git|node_modules|vendor|dist|build" # Dirs to ignore in 'tree' output
TREE_MAX_DEPTH="" # Max depth for 'tree' (e.g., "-L 3")

# --- Help Function ---
show_usage() {
  echo "Usage: $(basename "$0") [-i <include_pattern>] [-e <exclude_pattern>] [-q \"question\"] [-h]"
  echo ""
  echo "Options:"
  echo "  -i <pattern> : Glob pattern to INCLUDE files/folders (default: '*' if no -i is provided)."
  echo "                 Can be used multiple times (e.g., -i 'src/*' -i '*.py')."
  echo "  -e <pattern> : Glob pattern to EXCLUDE files/folders (e.g., -e '*.log' -e 'tests/data/*')."
  echo "                 Can be used multiple times."
  echo "  -q \"question\" : Specifies the question for the LLM."
  echo "  -h            : Displays this help message."
  echo ""
  echo "Example: $(basename "$0") -i 'src/**/*.js' -e '**/__tests__/*' -q \"Refactor this React code to use Hooks.\""
  exit 1
}

# --- Parse Command Line Options ---
while getopts "hi:e:q:" opt; do
  case $opt in
    h) show_usage ;;
    i) INCLUDE_PATTERNS+=("$OPTARG") ;;
    e) EXCLUDE_OPTS+=("--exclude=${OPTARG}") ;; # Prepare the option directly for git ls-files
    q) LLM_QUESTION="$OPTARG" ;;
    \?) echo "Invalid option: -$OPTARG" >&2; show_usage ;;
    :) echo "Option -$OPTARG requires an argument." >&2; show_usage ;;
  esac
done
shift $((OPTIND-1)) # Remove processed options

# --- Set Default Include Pattern if Necessary ---
# If no include patterns were provided, set a default
if [ ${#INCLUDE_PATTERNS[@]} -eq 0 ]; then
  # *** FIX: Default to '*' to include all files if no -i pattern is specified ***
  INCLUDE_PATTERNS=("*")
fi

# --- Check necessary commands ---
command -v git >/dev/null 2>&1 || { echo >&2 "ERROR: Command 'git' not found. Please install it."; exit 1; }
command -v tree >/dev/null 2>&1 || { echo >&2 "ERROR: Command 'tree' not found. Please install it."; exit 1; }
command -v xsel >/dev/null 2>&1 || { echo >&2 "ERROR: Command 'xsel' not found. Please install it."; exit 1; }
command -v file >/dev/null 2>&1 || { echo >&2 "ERROR: Command 'file' not found. Please install it."; exit 1; }

# --- Check if inside a Git repository ---
if ! git rev-parse --is-inside-work-tree > /dev/null 2>&1; then
  echo >&2 "ERROR: You are not inside a Git repository."
  echo >&2 "This script uses 'git ls-files' to list files and respect .gitignore."
  exit 1
fi

# --- Build Prompt ---
echo "Generating prompt..."
echo "Inclusion patterns: ${INCLUDE_PATTERNS[@]}"
if [ ${#EXCLUDE_OPTS[@]} -gt 0 ]; then
    # Extract just the patterns for display
    exclude_patterns_display=$(printf "%s\n" "${EXCLUDE_OPTS[@]}" | sed 's/^--exclude=//')
    echo "Exclusion patterns: $exclude_patterns_display"
fi
echo "Question: $LLM_QUESTION"

PROMPT_CONTENT="Here is the context of my current project. Analyze the structure and content of the provided files to answer my question.\n\n"

# 2. Project structure via 'tree'
PROMPT_CONTENT+="--- PROJECT STRUCTURE (based on 'tree', may differ slightly from included files) ---\n"
TREE_CMD="tree"
if [ -n "$TREE_MAX_DEPTH" ]; then
  TREE_CMD+=" $TREE_MAX_DEPTH"
fi
# The -I option for tree doesn't handle complex patterns like git ls-files,
# so we keep a simple exclusion based on TREE_IGNORE_DIRS.
# User's -e patterns are NOT applied to tree here for simplicity.
if [ -n "$TREE_IGNORE_DIRS" ]; then
  # Check if the -I option is supported (newer versions of tree)
  if tree --help | grep -q "\-I pattern"; then
    TREE_CMD+=" -I '$TREE_IGNORE_DIRS'"
  else
    echo >&2 "Warning: Your version of 'tree' might not support -I. Attempting without specific directory exclusion for tree."
  fi
fi
# Execute the tree command. Use eval to handle quotes in -I if present. Redirect stderr.
PROJECT_TREE=$(eval $TREE_CMD 2>/dev/null || echo "Error running tree.")
PROMPT_CONTENT+="$PROJECT_TREE\n\n"

# 3. Content of relevant files (via git ls-files)
PROMPT_CONTENT+="--- FILE CONTENT (based on git ls-files, respecting .gitignore and -i/-e options) ---\n"
FILE_COUNTER=0
TOTAL_SIZE=0 # Could be calculated if needed

# Build git ls-files command
# -c: cached (tracked) / -o: others (untracked but not ignored)
# --exclude-standard: respects .gitignore, .git/info/exclude, etc.
# "${EXCLUDE_OPTS[@]}": adds the --exclude=... options specified by the user
# -- : separates options from path patterns
# "${INCLUDE_PATTERNS[@]}": the patterns to include specified by the user
GIT_LS_FILES_CMD=(git ls-files -co --exclude-standard "${EXCLUDE_OPTS[@]}" -- "${INCLUDE_PATTERNS[@]}")

while IFS= read -r file || [[ -n "$file" ]]; do
  if [ ! -f "$file" ] || [ ! -r "$file" ]; then
    echo >&2 "Warning: File '$file' listed by git but is not a regular readable file. Skipping."
    continue
  fi

  MIME_TYPE=$(file -b --mime-type "$file")
   if [[ "$MIME_TYPE" != text/* && \
         "$MIME_TYPE" != application/json && \
         "$MIME_TYPE" != application/xml && \
         "$MIME_TYPE" != application/javascript && \
         "$MIME_TYPE" != application/x-sh && \
         "$MIME_TYPE" != application/x-shellscript && \
         "$MIME_TYPE" != application/x-python* && \
         "$MIME_TYPE" != application/x-php && \
         "$MIME_TYPE" != application/x-ruby && \
         "$MIME_TYPE" != application/toml && \
         "$MIME_TYPE" != application/yaml ]]; then
     echo "Info: Skipping file '$file' (non-text MIME type: $MIME_TYPE)"
     continue
  fi

  FILE_SIZE=$(stat -c%s "$file" 2>/dev/null || stat -f %z "$file" 2>/dev/null || echo 0) 
  if [ "$FILE_SIZE" -gt 1048576 ]; then # 1 MiB
    echo "Info: Skipping file '$file' because it is too large (> 1MiB)."
    continue
  fi

  PROMPT_CONTENT+="\n--- FILE: $file ---\n"
  FILE_CONTENT=$(cat "$file" 2>/dev/null)
  if [ $? -ne 0 ]; then
      echo >&2 "Warning: Failed to read content of '$file' with cat. Skipping."
      PROMPT_CONTENT="${PROMPT_CONTENT%--- FILE: $file ---*}"
      continue
  fi
  PROMPT_CONTENT+="$FILE_CONTENT"
  PROMPT_CONTENT+="\n--- END FILE: $file ---\n"
  ((FILE_COUNTER++))
  # Could add logic here to calculate and check total prompt size
done < <("${GIT_LS_FILES_CMD[@]}")


PROMPT_CONTENT+="\n--- END OF FILE CONTENT ---\n"

# 4. Final question
PROMPT_CONTENT+="\nBased on the context provided above, answer the following question:\n\n$LLM_QUESTION\n"

# --- Copy to Clipboard ---
printf "%s" "$PROMPT_CONTENT" | xsel -ib

# --- User Feedback ---
echo "-------------------------------------"
echo "Prompt generated and copied to clipboard!"
echo "Number of files included: $FILE_COUNTER"
# TODO: display total size here if calculated
if [[ "$LLM_QUESTION" == "[YOUR QUESTION HERE]" ]]; then
    echo "NOTE: No question specified with -q. Remember to replace '[YOUR QUESTION HERE]'."
fi
echo "Paste (Ctrl+Shift+V or middle-click) into your LLM."
echo "-------------------------------------"

exit 0
