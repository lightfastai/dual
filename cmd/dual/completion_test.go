package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lightfastai/dual/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompletionCommand(t *testing.T) {
	tests := []struct {
		name          string
		shell         string
		expectErr     bool
		expectContain string
	}{
		{
			name:          "bash completion",
			shell:         "bash",
			expectErr:     false,
			expectContain: "# bash completion",
		},
		{
			name:          "zsh completion",
			shell:         "zsh",
			expectErr:     false,
			expectContain: "# zsh completion",
		},
		{
			name:          "fish completion",
			shell:         "fish",
			expectErr:     false,
			expectContain: "# fish completion",
		},
		{
			name:      "invalid shell",
			shell:     "invalid",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh command for this test
			cmd := &cobra.Command{
				Use:   "completion",
				Args:  cobra.ExactArgs(1),
				ValidArgs: []string{"bash", "zsh", "fish"},
				RunE: func(cmd *cobra.Command, args []string) error {
					switch args[0] {
					case "bash":
						return rootCmd.GenBashCompletion(cmd.OutOrStdout())
					case "zsh":
						return rootCmd.GenZshCompletion(cmd.OutOrStdout())
					case "fish":
						return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
					default:
						return fmt.Errorf("unsupported shell type %q", args[0])
					}
				},
			}

			// Capture output
			var stdout bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stdout)
			cmd.SetArgs([]string{tt.shell})

			// Execute command
			err := cmd.Execute()

			// Check error expectation
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				output := stdout.String()
				assert.NotEmpty(t, output, "completion output should not be empty")
				assert.Contains(t, output, tt.expectContain, "completion output should contain expected string")
			}
		})
	}
}

func TestServiceCompletion(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()

	// Create a test config
	testConfig := &config.Config{
		Version: 1,
		Services: map[string]config.Service{
			"api": {
				Path: ".",
			},
			"web": {
				Path: ".",
			},
			"worker": {
				Path: ".",
			},
		},
	}

	// Save config to temp directory
	configPath := filepath.Join(tempDir, "dual.config.yml")
	err := config.SaveConfig(testConfig, configPath)
	require.NoError(t, err)

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		require.NoError(t, err)
	}()
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test service completion
	completions, directive := serviceCompletion(nil, []string{}, "")

	// Verify completions
	assert.Equal(t, 3, len(completions), "should return 3 services")
	assert.Contains(t, completions, "api")
	assert.Contains(t, completions, "web")
	assert.Contains(t, completions, "worker")

	// Verify no file completion
	assert.Equal(t, directive, cobra.ShellCompDirectiveNoFileComp)

	// Verify alphabetical order
	assert.Equal(t, []string{"api", "web", "worker"}, completions)
}

func TestServiceCompletionNoConfig(t *testing.T) {
	// Create a temporary directory without config
	tempDir := t.TempDir()

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		require.NoError(t, err)
	}()
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test service completion
	completions, directive := serviceCompletion(nil, []string{}, "")

	// Should return empty list when config doesn't exist
	assert.Empty(t, completions, "should return empty list when config not found")
	assert.Equal(t, directive, cobra.ShellCompDirectiveNoFileComp)
}

func TestContextCompletion(t *testing.T) {
	// This test verifies that context completion returns empty gracefully when registry doesn't exist
	// Rather than testing the full integration (which would require mocking the registry),
	// we just verify the function doesn't panic and returns the expected directive

	// Create a temporary directory for test
	tempDir := t.TempDir()

	// Create a test config
	testConfig := &config.Config{
		Version: 1,
		Services: map[string]config.Service{
			"api": {
				Path: ".",
			},
		},
	}

	// Save config to temp directory
	configPath := filepath.Join(tempDir, "dual.config.yml")
	err := config.SaveConfig(testConfig, configPath)
	require.NoError(t, err)

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		require.NoError(t, err)
	}()
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test context completion - should not panic even with no contexts
	completions, directive := contextCompletion(nil, []string{}, "")

	// Should return empty or some completions, but not panic
	assert.NotNil(t, completions)

	// Verify no file completion directive is set
	assert.Equal(t, directive, cobra.ShellCompDirectiveNoFileComp)
}

func TestContextCompletionNoRegistry(t *testing.T) {
	// Create a temporary directory with config but no contexts
	tempDir := t.TempDir()

	// Create a test config
	testConfig := &config.Config{
		Version: 1,
		Services: map[string]config.Service{
			"api": {
				Path: ".",
			},
		},
	}

	// Save config to temp directory
	configPath := filepath.Join(tempDir, "dual.config.yml")
	err := config.SaveConfig(testConfig, configPath)
	require.NoError(t, err)

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		require.NoError(t, err)
	}()
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test context completion (no contexts exist)
	completions, directive := contextCompletion(nil, []string{}, "")

	// Should return empty list when no contexts exist
	assert.Empty(t, completions, "should return empty list when no contexts found")
	assert.Equal(t, directive, cobra.ShellCompDirectiveNoFileComp)
}

func TestCompletionHelp(t *testing.T) {
	// Test that completion command has proper help text
	assert.NotEmpty(t, completionCmd.Long)
	assert.Contains(t, completionCmd.Long, "bash")
	assert.Contains(t, completionCmd.Long, "zsh")
	assert.Contains(t, completionCmd.Long, "fish")
	assert.Contains(t, strings.ToLower(completionCmd.Long), "completion")
}
