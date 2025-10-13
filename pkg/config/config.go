// Package config provides functionality for loading and parsing .mpp.txt configuration files.
// It handles alias definitions and recursive file search up the directory tree.
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Alias represents a named alias with its associated options
type Alias struct {
	Name    string
	Options string
	Source  string // Path to the config file where this alias was defined
}

// Config holds all loaded aliases
type Config struct {
	Aliases map[string]Alias // Key is the alias name
}

// NewConfig creates a new empty config
func NewConfig() *Config {
	return &Config{
		Aliases: make(map[string]Alias),
	}
}

// LoadAliases loads aliases from .mpp.txt files, searching recursively up the directory tree
func LoadAliases() (*Config, error) {
	config := NewConfig()
	seenAliases := make(map[string]string) // Track where each alias was first seen

	// Start from current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Walk up the directory tree
	for {
		configPath := filepath.Join(currentDir, ".mpp.txt")

		// Check if config file exists
		if _, err := os.Stat(configPath); err == nil {
			// Load aliases from this file
			aliases, err := parseConfigFile(configPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to parse config file %s: %v\n", configPath, err)
			} else {
				// Add aliases, checking for duplicates
				for _, alias := range aliases {
					if existingSource, exists := seenAliases[alias.Name]; exists {
						// Alias already exists - first one wins
						fmt.Fprintf(os.Stderr, "Warning: alias [%s] is duplicated (first defined in %s, also in %s)\n",
							alias.Name, existingSource, configPath)
					} else {
						// Add the alias
						config.Aliases[alias.Name] = alias
						seenAliases[alias.Name] = configPath
					}
				}
			}
		}

		// Move to parent directory
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			// Reached root
			break
		}
		currentDir = parent
	}

	return config, nil
}

// parseConfigFile parses a single .mpp.txt config file
func parseConfigFile(path string) ([]Alias, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var aliases []Alias
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse alias definition: "alias_name: options"
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "Warning: Invalid alias definition at %s:%d (expected format 'name: options')\n", path, lineNum)
			continue
		}

		name := strings.TrimSpace(parts[0])
		options := strings.TrimSpace(parts[1])

		if name == "" {
			fmt.Fprintf(os.Stderr, "Warning: Empty alias name at %s:%d\n", path, lineNum)
			continue
		}

		aliases = append(aliases, Alias{
			Name:    name,
			Options: options,
			Source:  path,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return aliases, nil
}

// GetAlias retrieves an alias by name
func (c *Config) GetAlias(name string) (Alias, bool) {
	alias, exists := c.Aliases[name]
	return alias, exists
}

// ListAliases returns all aliases sorted by name
func (c *Config) ListAliases() []Alias {
	aliases := make([]Alias, 0, len(c.Aliases))
	for _, alias := range c.Aliases {
		aliases = append(aliases, alias)
	}
	return aliases
}

// ExpandAlias takes an alias and returns the expanded options as a slice of arguments
func ExpandAlias(options string) []string {
	// Simple shell-like parsing that respects quotes
	var args []string
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0)

	for _, ch := range options {
		if inQuotes {
			if ch == quoteChar {
				inQuotes = false
				quoteChar = 0
			} else {
				current.WriteRune(ch)
			}
		} else {
			if ch == '"' || ch == '\'' {
				inQuotes = true
				quoteChar = ch
			} else if ch == ' ' || ch == '\t' {
				if current.Len() > 0 {
					args = append(args, current.String())
					current.Reset()
				}
			} else {
				current.WriteRune(ch)
			}
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}
