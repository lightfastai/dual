package health

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusSeverity(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected int
	}{
		{"Pass", StatusPass, 0},
		{"Warn", StatusWarn, 1},
		{"Error", StatusError, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.Severity())
		})
	}
}

func TestNewCheck(t *testing.T) {
	check := NewCheck("Test Check", StatusPass, "All good")

	assert.Equal(t, "Test Check", check.Name)
	assert.Equal(t, StatusPass, check.Status)
	assert.Equal(t, "All good", check.Message)
	assert.Empty(t, check.Details)
	assert.Empty(t, check.FixAction)
	assert.False(t, check.FixApplied)
}

func TestCheckWithDetails(t *testing.T) {
	check := NewCheck("Test", StatusPass, "message").
		WithDetails("detail1", "detail2", "detail3")

	assert.Len(t, check.Details, 3)
	assert.Equal(t, "detail1", check.Details[0])
	assert.Equal(t, "detail3", check.Details[2])
}

func TestCheckWithFixAction(t *testing.T) {
	check := NewCheck("Test", StatusWarn, "warning").
		WithFixAction("Run 'dual fix'")

	assert.Equal(t, "Run 'dual fix'", check.FixAction)
	assert.False(t, check.FixApplied)
}

func TestCheckWithFixApplied(t *testing.T) {
	check := NewCheck("Test", StatusWarn, "warning").
		WithFixAction("Run 'dual fix'").
		WithFixApplied()

	assert.True(t, check.FixApplied)
}

func TestCheckWithError(t *testing.T) {
	err := assert.AnError
	check := NewCheck("Test", StatusError, "failed").
		WithError(err)

	assert.Equal(t, err.Error(), check.ErrorString)
}

func TestCheckWithNilError(t *testing.T) {
	check := NewCheck("Test", StatusError, "failed").
		WithError(nil)

	assert.Empty(t, check.ErrorString)
}

func TestNewResult(t *testing.T) {
	result := NewResult()

	assert.NotNil(t, result)
	assert.Empty(t, result.Checks)
	assert.Equal(t, 0, result.TotalChecks)
	assert.Equal(t, 0, result.Passed)
	assert.Equal(t, 0, result.Warnings)
	assert.Equal(t, 0, result.Errors)
}

func TestResultAddCheck(t *testing.T) {
	result := NewResult()

	result.AddCheck(NewCheck("Check1", StatusPass, "ok"))
	result.AddCheck(NewCheck("Check2", StatusWarn, "warning"))
	result.AddCheck(NewCheck("Check3", StatusError, "error"))
	result.AddCheck(NewCheck("Check4", StatusPass, "ok"))

	assert.Equal(t, 4, result.TotalChecks)
	assert.Equal(t, 2, result.Passed)
	assert.Equal(t, 1, result.Warnings)
	assert.Equal(t, 1, result.Errors)
}

func TestResultDetermineExitCode(t *testing.T) {
	tests := []struct {
		name     string
		checks   []Check
		expected int
	}{
		{
			name: "All pass",
			checks: []Check{
				NewCheck("C1", StatusPass, "ok"),
				NewCheck("C2", StatusPass, "ok"),
			},
			expected: 0,
		},
		{
			name: "With warnings",
			checks: []Check{
				NewCheck("C1", StatusPass, "ok"),
				NewCheck("C2", StatusWarn, "warning"),
			},
			expected: 1,
		},
		{
			name: "With errors",
			checks: []Check{
				NewCheck("C1", StatusPass, "ok"),
				NewCheck("C2", StatusWarn, "warning"),
				NewCheck("C3", StatusError, "error"),
			},
			expected: 2,
		},
		{
			name: "Errors take precedence",
			checks: []Check{
				NewCheck("C1", StatusError, "error"),
				NewCheck("C2", StatusWarn, "warning"),
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewResult()
			for _, check := range tt.checks {
				result.AddCheck(check)
			}
			assert.Equal(t, tt.expected, result.DetermineExitCode())
		})
	}
}

func TestResultFormat(t *testing.T) {
	result := NewResult()
	result.AddCheck(NewCheck("Check1", StatusPass, "Everything is fine"))
	result.AddCheck(NewCheck("Check2", StatusWarn, "Minor issue").
		WithDetails("Detail 1", "Detail 2").
		WithFixAction("Run fix command"))
	result.AddCheck(NewCheck("Check3", StatusError, "Critical error").
		WithError(assert.AnError))

	output := result.Format(false)

	// Check that output contains expected elements
	assert.Contains(t, output, "Dual Health Check Results")
	assert.Contains(t, output, "Check1")
	assert.Contains(t, output, "Check2")
	assert.Contains(t, output, "Check3")
	assert.Contains(t, output, "pass")
	assert.Contains(t, output, "warn")
	assert.Contains(t, output, "error")
}

func TestResultFormatVerbose(t *testing.T) {
	result := NewResult()
	result.AddCheck(NewCheck("Check1", StatusPass, "ok").
		WithDetails("Hidden detail"))

	outputNormal := result.Format(false)
	outputVerbose := result.Format(true)

	// Verbose should show details for passing checks
	assert.NotContains(t, outputNormal, "Hidden detail")
	assert.Contains(t, outputVerbose, "Hidden detail")
}

func TestResultFormatJSON(t *testing.T) {
	result := NewResult()
	result.AddCheck(NewCheck("Check1", StatusPass, "ok"))
	result.AddCheck(NewCheck("Check2", StatusWarn, "warning"))
	result.ExitCode = result.DetermineExitCode()

	jsonOutput, err := result.FormatJSON()
	require.NoError(t, err)

	// Verify it's valid JSON
	var parsed Result
	err = json.Unmarshal([]byte(jsonOutput), &parsed)
	require.NoError(t, err)

	assert.Equal(t, 2, parsed.TotalChecks)
	assert.Equal(t, 1, parsed.Passed)
	assert.Equal(t, 1, parsed.Warnings)
	assert.Equal(t, 0, parsed.Errors)
	assert.Equal(t, 1, parsed.ExitCode)
}

func TestCheckSortingByStatus(t *testing.T) {
	result := NewResult()
	result.AddCheck(NewCheck("Pass1", StatusPass, "ok"))
	result.AddCheck(NewCheck("Error1", StatusError, "error"))
	result.AddCheck(NewCheck("Warn1", StatusWarn, "warn"))
	result.AddCheck(NewCheck("Pass2", StatusPass, "ok"))
	result.AddCheck(NewCheck("Error2", StatusError, "error"))

	output := result.Format(false)

	// Errors should appear before warnings, warnings before passes
	errorPos := strings.Index(output, "Error1")
	warnPos := strings.Index(output, "Warn1")
	passPos := strings.Index(output, "Pass1")

	assert.True(t, errorPos < warnPos, "Errors should appear before warnings")
	assert.True(t, warnPos < passPos, "Warnings should appear before passes")
}

func TestCheckMethodChaining(t *testing.T) {
	check := NewCheck("Test", StatusPass, "message").
		WithDetails("detail1").
		WithFixAction("fix").
		WithFixApplied().
		WithError(assert.AnError)

	assert.Equal(t, "Test", check.Name)
	assert.Equal(t, StatusPass, check.Status)
	assert.Len(t, check.Details, 1)
	assert.Equal(t, "fix", check.FixAction)
	assert.True(t, check.FixApplied)
	assert.NotEmpty(t, check.ErrorString)
}
