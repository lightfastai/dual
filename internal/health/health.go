package health

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/fatih/color"
)

// Status represents the health check result status
type Status string

const (
	// StatusPass indicates the check passed
	StatusPass Status = "pass"
	// StatusWarn indicates the check passed with warnings
	StatusWarn Status = "warn"
	// StatusError indicates the check failed
	StatusError Status = "error"
)

// Severity returns the numeric severity of a status (higher is worse)
func (s Status) Severity() int {
	switch s {
	case StatusPass:
		return 0
	case StatusWarn:
		return 1
	case StatusError:
		return 2
	default:
		return 0
	}
}

// Check represents a single health check
type Check struct {
	Name        string   `json:"name"`
	Status      Status   `json:"status"`
	Message     string   `json:"message"`
	FixAction   string   `json:"fixAction,omitempty"`
	FixApplied  bool     `json:"fixApplied,omitempty"`
	Details     []string `json:"details,omitempty"`
	ErrorString string   `json:"error,omitempty"`
}

// Result represents the aggregated health check results
type Result struct {
	Checks      []Check `json:"checks"`
	TotalChecks int     `json:"totalChecks"`
	Passed      int     `json:"passed"`
	Warnings    int     `json:"warnings"`
	Errors      int     `json:"errors"`
	ExitCode    int     `json:"exitCode"`
}

// AddCheck adds a health check result to the result set
func (r *Result) AddCheck(check Check) {
	r.Checks = append(r.Checks, check)
	r.TotalChecks++

	switch check.Status {
	case StatusPass:
		r.Passed++
	case StatusWarn:
		r.Warnings++
	case StatusError:
		r.Errors++
	}
}

// DetermineExitCode calculates the exit code based on check results
// 0 = all pass, 1 = warnings, 2 = errors
func (r *Result) DetermineExitCode() int {
	if r.Errors > 0 {
		return 2
	}
	if r.Warnings > 0 {
		return 1
	}
	return 0
}

// Format formats the result as human-readable output with colors
func (r *Result) Format(verbose bool) string {
	var sb strings.Builder

	// Header
	sb.WriteString("\n")
	sb.WriteString(color.New(color.Bold).Sprint("Dual Health Check Results"))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("=", 50))
	sb.WriteString("\n\n")

	// Summary
	statusColor := color.GreenString
	statusIcon := "✓"
	statusText := "HEALTHY"

	if r.Errors > 0 {
		statusColor = color.RedString
		statusIcon = "✗"
		statusText = "UNHEALTHY"
	} else if r.Warnings > 0 {
		statusColor = color.YellowString
		statusIcon = "⚠"
		statusText = "WARNINGS"
	}

	sb.WriteString(fmt.Sprintf("%s Overall Status: %s\n\n", statusIcon, statusColor(statusText)))

	// Stats
	sb.WriteString(fmt.Sprintf("Total Checks: %d\n", r.TotalChecks))
	sb.WriteString(fmt.Sprintf("%s Passed: %d\n", color.GreenString("✓"), r.Passed))
	if r.Warnings > 0 {
		sb.WriteString(fmt.Sprintf("%s Warnings: %d\n", color.YellowString("⚠"), r.Warnings))
	}
	if r.Errors > 0 {
		sb.WriteString(fmt.Sprintf("%s Errors: %d\n", color.RedString("✗"), r.Errors))
	}
	sb.WriteString("\n")

	// Sort checks: errors first, then warnings, then passed
	sortedChecks := make([]Check, len(r.Checks))
	copy(sortedChecks, r.Checks)
	sort.Slice(sortedChecks, func(i, j int) bool {
		return sortedChecks[i].Status.Severity() > sortedChecks[j].Status.Severity()
	})

	// Detailed results
	sb.WriteString(color.New(color.Bold).Sprint("Check Details"))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("-", 50))
	sb.WriteString("\n\n")

	for _, check := range sortedChecks {
		// Status icon and name
		icon, statusColor := getStatusDisplay(check.Status)
		sb.WriteString(fmt.Sprintf("%s %s: %s\n",
			icon,
			color.New(color.Bold).Sprint(check.Name),
			statusColor("%s", string(check.Status))))

		// Message
		if check.Message != "" {
			sb.WriteString(fmt.Sprintf("  %s\n", check.Message))
		}

		// Details (only in verbose mode or for errors/warnings)
		if (verbose || check.Status != StatusPass) && len(check.Details) > 0 {
			for _, detail := range check.Details {
				sb.WriteString(fmt.Sprintf("  - %s\n", detail))
			}
		}

		// Fix action
		if check.FixAction != "" {
			if check.FixApplied {
				sb.WriteString(fmt.Sprintf("  %s Applied fix: %s\n",
					color.GreenString("✓"),
					check.FixAction))
			} else {
				sb.WriteString(fmt.Sprintf("  %s Fix available: %s\n",
					color.CyanString("ℹ"),
					check.FixAction))
			}
		}

		// Error details
		if check.ErrorString != "" && verbose {
			sb.WriteString(fmt.Sprintf("  Error: %s\n", color.RedString(check.ErrorString)))
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatJSON formats the result as JSON
func (r *Result) FormatJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(data), nil
}

// getStatusDisplay returns the icon and color function for a status
func getStatusDisplay(status Status) (string, func(format string, a ...interface{}) string) {
	switch status {
	case StatusPass:
		return "✓", color.GreenString
	case StatusWarn:
		return "⚠", color.YellowString
	case StatusError:
		return "✗", color.RedString
	default:
		return "?", color.WhiteString
	}
}

// NewCheck creates a new health check result
func NewCheck(name string, status Status, message string) Check {
	return Check{
		Name:    name,
		Status:  status,
		Message: message,
	}
}

// WithDetails adds details to a check
func (c Check) WithDetails(details ...string) Check {
	c.Details = append(c.Details, details...)
	return c
}

// WithFixAction adds a fix action to a check
func (c Check) WithFixAction(action string) Check {
	c.FixAction = action
	return c
}

// WithFixApplied marks a fix as applied
func (c Check) WithFixApplied() Check {
	c.FixApplied = true
	return c
}

// WithError adds an error string to a check
func (c Check) WithError(err error) Check {
	if err != nil {
		c.ErrorString = err.Error()
	}
	return c
}

// NewResult creates a new health check result
func NewResult() *Result {
	return &Result{
		Checks: make([]Check, 0),
	}
}
