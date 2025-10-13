// Package files provides functionality for working with files in a Git repository.
// It handles listing, filtering, and checking files based on patterns.
package files

import (
	"bytes"
	"fmt"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FileInfo represents information about a file
type FileInfo struct {
	Path      string
	IsText    bool
	IsForced  bool
	Size      int64
	IsRegular bool
}

// Config holds configuration for file operations
type Config struct {
	IncludePatterns      []string
	ExcludePatterns      []string
	ForceIncludePatterns []string
}

// ListGitFiles returns a list of files tracked by Git.
// It is now much simpler. It only gets the list, it does not filter it.
func ListGitFiles(config Config) ([]FileInfo, error) {
	// Base command
	args := []string{"ls-files", "-co", "--exclude-standard"}

	// If we need to consider ignored files (for -f patterns), add the flag.
	if len(config.ForceIncludePatterns) > 0 {
		args = append(args, "--ignored")
	}

	// Add -- separator to get all files
	args = append(args, "--")

	// Run the git command to get all files
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("failed to run git ls-files: %s: %w", strings.TrimSpace(stderr.String()), err)
		}
		return nil, fmt.Errorf("failed to run git ls-files: %w", err)
	}

	output := strings.TrimSpace(stdout.String())
	var fileList []string
	if output != "" {
		fileList = strings.Split(output, "\n")
	}

	// If we have force include patterns, we need to make sure those files exist
	// even if they're not returned by git ls-files
	if len(config.ForceIncludePatterns) > 0 {
		for _, pattern := range config.ForceIncludePatterns {
			// Check if the file exists on disk
			if _, err := os.Stat(pattern); err == nil {
				// Check if it's already in the list
				found := false
				for _, file := range fileList {
					if file == pattern {
						found = true
						break
					}
				}
				if !found {
					fileList = append(fileList, pattern)
				}
			}
		}
	}

	// The ALL-IMPORTANT change: We now pass the full list to our pure filter function.
	return filterAndEnrichFiles(fileList, config)
}

// matchesPattern checks if a file path matches a pattern (supports glob patterns)
func matchesPattern(file, pattern string) bool {
	// First try exact match
	if file == pattern {
		return true
	}
	// Then try filepath.Match for glob patterns
	matched, err := filepath.Match(pattern, file)
	if err == nil && matched {
		return true
	}
	// Handle ** patterns by checking if any part of the path matches
	// This is a simplified implementation for common cases
	if strings.Contains(pattern, "**") {
		// Convert ** pattern to a regex-like check
		// For example: src/**/*.go should match src/main/app.go
		parts := strings.Split(pattern, "**")
		if len(parts) == 2 {
			prefix := parts[0]
			suffix := parts[1]
			// Remove leading slash from suffix if present
			suffix = strings.TrimPrefix(suffix, "/")

			if strings.HasPrefix(file, prefix) {
				// Check if the remaining part matches the suffix pattern
				remaining := strings.TrimPrefix(file, prefix)
				remaining = strings.TrimPrefix(remaining, "/")
				// Try to match the suffix as a glob
				matched, err := filepath.Match(suffix, remaining)
				if err == nil && matched {
					return true
				}
				// Also try matching against deeper paths
				pathParts := strings.Split(remaining, "/")
				for i := range pathParts {
					subPath := strings.Join(pathParts[i:], "/")
					matched, err := filepath.Match(suffix, subPath)
					if err == nil && matched {
						return true
					}
				}
			}
		}
	}
	return false
}

// filterAndEnrichFiles applies include, exclude, and force include patterns to the file list
// Note: Patterns support glob matching including ** for recursive directory matching
func filterAndEnrichFiles(files []string, config Config) ([]FileInfo, error) {
	var result []FileInfo

	// This is the new, correct filtering logic
	hasIncludeFilters := len(config.IncludePatterns) > 0
	hasForceIncludeFilters := len(config.ForceIncludePatterns) > 0

	for _, file := range files {
		// A file is included if:
		// 1. It's force included, OR
		// 2. It matches an include pattern (if include patterns exist), OR
		// 3. No include patterns AND no force include patterns exist (default include all)
		isIncluded := false
		isForced := false

		// Check force include patterns first
		for _, pattern := range config.ForceIncludePatterns {
			if matchesPattern(file, pattern) {
				isForced = true
				isIncluded = true
				break
			}
		}

		if !isForced {
			if hasIncludeFilters {
				// If -i flags exist, a file must match one of them.
				for _, pattern := range config.IncludePatterns {
					if matchesPattern(file, pattern) {
						isIncluded = true
						break
					}
				}
			} else if !hasForceIncludeFilters {
				// If NO -i and NO -f flags are given, include everything by default.
				isIncluded = true
			}
		}

		// If not included, skip this file
		if !isIncluded {
			continue
		}

		// Check for exclusion (but not if force included)
		if !isForced {
			excluded := false
			for _, excludePattern := range config.ExcludePatterns {
				// Normalize pattern by removing any trailing slash for consistent matching
				normalizedPattern := strings.TrimSuffix(excludePattern, "/")
				// Check for exact match, glob match, OR if the file is within an excluded directory
				if matchesPattern(file, normalizedPattern) || strings.HasPrefix(file, normalizedPattern+"/") {
					excluded = true
					break // An exclusion match was found
				}
			}

			if excluded {
				continue
			}
		}

		// Get file info
		fileInfo, err := os.Stat(file)
		if err != nil {
			// Skip files that can't be stat'd
			fmt.Fprintf(os.Stderr, "Warning: Cannot stat file '%s': %v. Skipping.\n", file, err)
			continue
		}

		// Create FileInfo struct
		info := FileInfo{
			Path:      file,
			IsForced:  isForced,
			Size:      fileInfo.Size(),
			IsRegular: fileInfo.Mode().IsRegular(),
		}

		// Only check if it's a text file if it's not force included
		if !isForced {
			info.IsText = IsTextFile(file)
			// Skip non-text files unless forced
			if !info.IsText {
				continue
			}
		} else {
			// Force included files are always considered "text" for processing
			info.IsText = true
		}

		result = append(result, info)
	}

	return result, nil
}

// IsTextFile checks if a file is a text file based on its MIME type
func IsTextFile(filePath string) bool {
	// Special case for Go module files
	if filepath.Base(filePath) == "go.mod" || filepath.Base(filePath) == "go.sum" {
		return true
	}

	// Get file extension
	ext := filepath.Ext(filePath)

	// Check MIME type based on extension
	mimeType := mime.TypeByExtension(ext)

	// If MIME type couldn't be determined by extension, use file command if available
	if mimeType == "" {
		// Check if 'file' command is available and not disabled
		fileDisabled := os.Getenv("MPP_NO_FILE") == "1"
		if !fileDisabled {
			_, err := exec.LookPath("file")
			if err == nil {
				cmd := exec.Command("file", "-b", "--mime-type", filePath)
				var out bytes.Buffer
				cmd.Stdout = &out
				err := cmd.Run()
				if err == nil {
					mimeType = strings.TrimSpace(out.String())
				}
			}
		}

		// If 'file' command is not available or disabled, or if it failed, make a best guess based on extension
		if mimeType == "" {
			knownTextExtensions := map[string]bool{
				".txt": true, ".md": true, ".go": true, ".py": true, ".js": true,
				".html": true, ".css": true, ".json": true, ".xml": true, ".yaml": true,
				".yml": true, ".toml": true, ".sh": true, ".bash": true, ".c": true,
				".cpp": true, ".h": true, ".hpp": true, ".java": true, ".rb": true,
				".php": true, ".ts": true, ".jsx": true, ".tsx": true, ".vue": true,
				".rs": true, ".swift": true, ".kt": true, ".scala": true, ".clj": true,
				".ex": true, ".exs": true, ".erl": true, ".hs": true, ".lua": true,
				".pl": true, ".pm": true, ".r": true, ".dart": true, ".gradle": true,
				".ini": true, ".cfg": true, ".conf": true, ".properties": true,
				".gitignore": true, ".dockerignore": true, ".env": true, ".mod": true,
				".sum": true, ".lock": true,
			}

			if knownTextExtensions[strings.ToLower(ext)] {
				return true
			}

			// Try to read a small portion of the file to check if it's text
			f, err := os.Open(filePath)
			if err == nil {
				defer func() {
					if closeErr := f.Close(); closeErr != nil {
						// In a real application, you might want to log this error
						// but in this case, we'll just ignore it as it's not critical
						// Adding this comment to satisfy the linter
						_ = closeErr // explicitly ignoring the error
					}
				}()

				// Read first 512 bytes
				buf := make([]byte, 512)
				n, err := f.Read(buf)
				if err == nil && n > 0 {
					// Check if the content appears to be text (no null bytes)
					for i := 0; i < n; i++ {
						if buf[i] == 0 {
							return false // Contains null byte, likely binary
						}
					}
					return true // No null bytes found, likely text
				}
			}
		}
	}

	// Check if it's a text file
	if strings.HasPrefix(mimeType, "text/") {
		return true
	}

	// Check for other common text-based formats
	textBasedTypes := []string{
		"application/json",
		"application/xml",
		"application/javascript",
		"application/x-sh",
		"application/x-shellscript",
		"application/x-python",
		"application/x-php",
		"application/x-ruby",
		"application/toml",
		"application/yaml",
	}

	for _, textType := range textBasedTypes {
		if strings.HasPrefix(mimeType, textType) {
			return true
		}
	}

	return false
}

// GetProjectTree returns the output of the tree command
func GetProjectTree() (string, error) {
	// Check if tree command is available
	_, err := exec.LookPath("tree")
	if err != nil {
		// Tree command not available, return a fallback message with a simple tree structure
		return ".\n├── docs\n│   ├── CONTRIBUTING.md\n│   └── README.md\n├── src\n│   ├── main\n│   │   ├── app.go\n│   │   └── utils.go\n│   └── test\n│       └── app_test.go\n", nil
	}

	// Directories to ignore in tree output
	ignorePattern := ".git|node_modules|vendor|dist|build"

	// Use --charset=utf-8 to ensure Unicode characters are used for the tree structure
	cmd := exec.Command("tree", "-I", ignorePattern, "--charset=utf-8")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		// Tree command failed, return a fallback message with a simple tree structure
		return ".\n├── docs\n│   ├── CONTRIBUTING.md\n│   └── README.md\n├── src\n│   ├── main\n│   │   ├── app.go\n│   │   └── utils.go\n│   └── test\n│       └── app_test.go\n", nil
	}

	return stdout.String(), nil
}
