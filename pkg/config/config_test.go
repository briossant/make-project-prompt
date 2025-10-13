package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandAlias(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Simple options",
			input:    "-i *.go -e tests",
			expected: []string{"-i", "*.go", "-e", "tests"},
		},
		{
			name:     "Options with double quotes",
			input:    `--role-message "You are a JS expert" -i *.js`,
			expected: []string{"--role-message", "You are a JS expert", "-i", "*.js"},
		},
		{
			name:     "Options with single quotes",
			input:    `--role-message 'You are a Go expert' -i *.go`,
			expected: []string{"--role-message", "You are a Go expert", "-i", "*.go"},
		},
		{
			name:     "Multiple spaces",
			input:    "-i   *.go    -e  tests",
			expected: []string{"-i", "*.go", "-e", "tests"},
		},
		{
			name:     "Empty string",
			input:    "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandAlias(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d args, got %d: %v", len(tt.expected), len(result), result)
				return
			}
			for i, arg := range result {
				if arg != tt.expected[i] {
					t.Errorf("Arg %d: expected %q, got %q", i, tt.expected[i], arg)
				}
			}
		})
	}
}

func TestParseConfigFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".mpp.txt")

	configContent := `# This is a comment
js dev: --role-message "You are a JS expert" -i **.js
go dev: -i *.go -e tests
python: -i *.py --role-message "Python expert"

# Another comment
invalid line without colon
: empty name
empty_options:
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	aliases, err := parseConfigFile(configPath)
	if err != nil {
		t.Fatalf("Failed to parse config file: %v", err)
	}

	// Should have 4 valid aliases (js dev, go dev, python, empty_options)
	expectedCount := 4
	if len(aliases) != expectedCount {
		t.Errorf("Expected %d aliases, got %d", expectedCount, len(aliases))
	}

	// Check specific aliases
	aliasMap := make(map[string]Alias)
	for _, alias := range aliases {
		aliasMap[alias.Name] = alias
	}

	if alias, exists := aliasMap["js dev"]; !exists {
		t.Error("Expected 'js dev' alias to exist")
	} else if alias.Options != `--role-message "You are a JS expert" -i **.js` {
		t.Errorf("Wrong options for 'js dev': %q", alias.Options)
	}

	if alias, exists := aliasMap["go dev"]; !exists {
		t.Error("Expected 'go dev' alias to exist")
	} else if alias.Options != "-i *.go -e tests" {
		t.Errorf("Wrong options for 'go dev': %q", alias.Options)
	}

	if alias, exists := aliasMap["empty_options"]; !exists {
		t.Error("Expected 'empty_options' alias to exist")
	} else if alias.Options != "" {
		t.Errorf("Expected empty options, got: %q", alias.Options)
	}
}

func TestLoadAliases(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "project", "src")
	err := os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory structure: %v", err)
	}

	// Create config file in root
	rootConfig := filepath.Join(tmpDir, ".mpp.txt")
	rootContent := `root_alias: -i *.txt
common: --role-message "From root"
`
	err = os.WriteFile(rootConfig, []byte(rootContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write root config: %v", err)
	}

	// Create config file in project dir
	projectConfig := filepath.Join(tmpDir, "project", ".mpp.txt")
	projectContent := `project_alias: -i *.go
common: --role-message "From project"
`
	err = os.WriteFile(projectConfig, []byte(projectContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write project config: %v", err)
	}

	// Change to subDir to test loading
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	err = os.Chdir(subDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	config, err := LoadAliases()
	if err != nil {
		t.Fatalf("Failed to load aliases: %v", err)
	}

	// Should have 3 aliases: root_alias, project_alias, and common (first one wins)
	if len(config.Aliases) != 3 {
		t.Errorf("Expected 3 aliases, got %d", len(config.Aliases))
	}

	// Check that common alias is from project (closer config wins)
	if alias, exists := config.GetAlias("common"); !exists {
		t.Error("Expected 'common' alias to exist")
	} else if alias.Options != `--role-message "From project"` {
		t.Errorf("Expected 'common' from project, got: %q", alias.Options)
	}

	// Check other aliases exist
	if _, exists := config.GetAlias("root_alias"); !exists {
		t.Error("Expected 'root_alias' to exist")
	}
	if _, exists := config.GetAlias("project_alias"); !exists {
		t.Error("Expected 'project_alias' to exist")
	}
}
