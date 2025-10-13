package detector

import (
	"os"
	"os/exec"
	"strings"
)

// DetectContext detects the current development context
func DetectContext() (string, error) {
	// Check for .dual-context file
	if data, err := os.ReadFile(".dual-context"); err == nil {
		return strings.TrimSpace(string(data)), nil
	}

	// Try git branch
	cmd := exec.Command("git", "branch", "--show-current")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		return strings.TrimSpace(string(output)), nil
	}

	// Default context
	return "default", nil
}
