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

	"github.com/gobwas/glob"
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
	IncludePatterns     []string
	ExcludePatterns     []string
	ForceIncludePatterns []string
}

// ListGitFiles returns a list of files tracked by Git, filtered by the provided patterns
func ListGitFiles(config Config) ([]FileInfo, error) {
	// Build git ls-files command
	args := []string{"ls-files", "-co", "--exclude-standard"}

	// Add -- separator and include patterns
	args = append(args, "--")
	if len(config.IncludePatterns) > 0 || len(config.ForceIncludePatterns) > 0 {
		// If we have include patterns, use them
		allIncludePatterns := append([]string{}, config.IncludePatterns...)
		allIncludePatterns = append(allIncludePatterns, config.ForceIncludePatterns...)
		args = append(args, allIncludePatterns...)
	} else {
		// Default to all files if no include patterns specified
		args = append(args, "*")
	}

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
	if output == "" {
		return []FileInfo{}, nil
	}

	fileList := strings.Split(output, "\n")

	// Apply filtering
	return filterFiles(fileList, config)
}

// filterFiles applies include, exclude, and force include patterns to the file list
func filterFiles(files []string, config Config) ([]FileInfo, error) {
	var result []FileInfo

	// Compile glob patterns for better performance
	includeGlobs, err := compileGlobs(config.IncludePatterns)
	if err != nil {
		return nil, fmt.Errorf("invalid include pattern: %w", err)
	}

	excludeGlobs, err := compileGlobs(config.ExcludePatterns)
	if err != nil {
		return nil, fmt.Errorf("invalid exclude pattern: %w", err)
	}

	forceIncludeGlobs, err := compileGlobs(config.ForceIncludePatterns)
	if err != nil {
		return nil, fmt.Errorf("invalid force include pattern: %w", err)
	}

	for _, file := range files {
		// Check if file should be included
		included := len(includeGlobs) == 0 // If no include patterns, include all files
		for _, g := range includeGlobs {
			if g.Match(file) {
				included = true
				break
			}
		}

		// Check if file should be force included
		forced := false
		for _, g := range forceIncludeGlobs {
			if g.Match(file) {
				included = true
				forced = true
				break
			}
		}

		// If not included or forced, skip this file
		if !included {
			continue
		}

		// Check if file should be excluded
		excluded := false
		for _, g := range excludeGlobs {
			if g.Match(file) {
				excluded = true
				break
			}
		}

		// If excluded, skip this file
		if excluded {
			continue
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
			IsForced:  forced,
			Size:      fileInfo.Size(),
			IsRegular: fileInfo.Mode().IsRegular(),
		}

		// Only check if it's a text file if it's not force included
		if !forced {
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

// compileGlobs compiles a list of glob patterns for efficient matching
func compileGlobs(patterns []string) ([]glob.Glob, error) {
	var globs []glob.Glob
	for _, pattern := range patterns {
		g, err := glob.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern '%s': %w", pattern, err)
		}
		globs = append(globs, g)
	}
	return globs, nil
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
		// Check if 'file' command is available
		_, err := exec.LookPath("file")
		if err == nil {
			cmd := exec.Command("file", "-b", "--mime-type", filePath)
			var out bytes.Buffer
			cmd.Stdout = &out
			err := cmd.Run()
			if err == nil {
				mimeType = strings.TrimSpace(out.String())
			}
		} else {
			// If 'file' command is not available, make a best guess based on extension
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
				defer f.Close()

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
	// Directories to ignore in tree output
	ignorePattern := ".git|node_modules|vendor|dist|build"

	cmd := exec.Command("tree", "-I", ignorePattern)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("failed to run tree command: %s: %w", strings.TrimSpace(stderr.String()), err)
		}
		return "", fmt.Errorf("failed to run tree command: %w", err)
	}

	return stdout.String(), nil
}
