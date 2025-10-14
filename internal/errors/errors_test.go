package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected string
	}{
		{
			name:     "simple message",
			err:      New(ErrConfigNotFound, "config not found"),
			expected: "config not found",
		},
		{
			name:     "empty message",
			err:      &Error{Type: ErrConfigNotFound},
			expected: "unknown error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := New(ErrConfigInvalid, "config invalid").WithCause(cause)

	if got := err.Unwrap(); got != cause {
		t.Errorf("Unwrap() = %v, want %v", got, cause)
	}

	// Test errors.Is
	if !errors.Is(err, cause) {
		t.Error("errors.Is() should return true for wrapped error")
	}
}

func TestError_WithContext(t *testing.T) {
	err := New(ErrConfigNotFound, "config not found").
		WithContext("File", "dual.config.yml").
		WithContext("Directory", "/home/user/project")

	if len(err.Context) != 2 {
		t.Errorf("expected 2 context entries, got %d", len(err.Context))
	}

	if err.Context["File"] != "dual.config.yml" {
		t.Errorf("Context[File] = %q, want %q", err.Context["File"], "dual.config.yml")
	}

	if err.Context["Directory"] != "/home/user/project" {
		t.Errorf("Context[Directory] = %q, want %q", err.Context["Directory"], "/home/user/project")
	}
}

func TestError_WithFixes(t *testing.T) {
	err := New(ErrConfigNotFound, "config not found").
		WithFix("Run 'dual init'").
		WithFix("Check directory").
		WithFixes("Fix 3", "Fix 4")

	if len(err.Fixes) != 4 {
		t.Errorf("expected 4 fixes, got %d", len(err.Fixes))
	}

	expected := []string{"Run 'dual init'", "Check directory", "Fix 3", "Fix 4"}
	for i, fix := range expected {
		if err.Fixes[i] != fix {
			t.Errorf("Fixes[%d] = %q, want %q", i, err.Fixes[i], fix)
		}
	}
}

func TestError_Format(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		contains []string
	}{
		{
			name: "basic error",
			err:  New(ErrConfigNotFound, "Configuration file not found"),
			contains: []string{
				"Error:",
				"Configuration file not found",
			},
		},
		{
			name: "error with context",
			err: New(ErrContextNotFound, "Context not found").
				WithContext("Context", "feature-auth").
				WithContext("Project", "/home/user/project"),
			contains: []string{
				"Error:",
				"Context not found",
				"Context: feature-auth",
				"Project: /home/user/project",
			},
		},
		{
			name: "error with cause",
			err: New(ErrConfigInvalid, "Invalid config").
				WithCause(errors.New("yaml parse error")),
			contains: []string{
				"Error:",
				"Invalid config",
				"Cause:",
				"yaml parse error",
			},
		},
		{
			name: "error with fixes",
			err: New(ErrConfigNotFound, "Config not found").
				WithFix("Run 'dual init'").
				WithFix("Check directory"),
			contains: []string{
				"Error:",
				"Config not found",
				"How to fix:",
				"Run 'dual init'",
				"Check directory",
			},
		},
		{
			name: "complete error",
			err: New(ErrServiceNotDetected, "Service not detected").
				WithContext("Directory", "/home/user/project").
				WithCause(errors.New("no match found")).
				WithFix("cd into a service directory").
				WithFix("Or specify --service flag"),
			contains: []string{
				"Error:",
				"Service not detected",
				"Directory: /home/user/project",
				"Cause:",
				"no match found",
				"How to fix:",
				"cd into a service directory",
				"Or specify --service flag",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := tt.err.Format()

			for _, want := range tt.contains {
				if !strings.Contains(formatted, want) {
					t.Errorf("Format() does not contain %q\nGot:\n%s", want, formatted)
				}
			}
		})
	}
}

func TestError_Format_Colors(t *testing.T) {
	err := New(ErrConfigNotFound, "Config not found").
		WithFix("Run dual init")

	formatted := err.Format()

	// Check for ANSI color codes
	if !strings.Contains(formatted, "\033[31m") { // Red for "Error:"
		t.Error("Format() should contain red color code for error")
	}

	if !strings.Contains(formatted, "\033[33m") { // Yellow for "How to fix:"
		t.Error("Format() should contain yellow color code for fixes")
	}

	if !strings.Contains(formatted, "\033[0m") { // Reset
		t.Error("Format() should contain color reset code")
	}
}

func TestNew(t *testing.T) {
	err := New(ErrConfigNotFound, "test message")

	if err.Type != ErrConfigNotFound {
		t.Errorf("Type = %v, want %v", err.Type, ErrConfigNotFound)
	}

	if err.Message != "test message" {
		t.Errorf("Message = %q, want %q", err.Message, "test message")
	}

	if err.Context == nil {
		t.Error("Context should be initialized")
	}

	if err.Fixes == nil {
		t.Error("Fixes should be initialized")
	}
}

func TestErrorType_Constants(t *testing.T) {
	// Ensure all error types are defined and unique
	types := []ErrorType{
		ErrConfigNotFound,
		ErrConfigInvalid,
		ErrConfigExists,
		ErrRegistryCorrupted,
		ErrContextNotFound,
		ErrServiceNotFound,
		ErrPortConflict,
		ErrServiceNotDetected,
		ErrContextDetectionFailed,
		ErrPortCalculationFailed,
		ErrCommandFailed,
		ErrEnvNotFound,
		ErrEnvParseFailed,
		ErrEnvConflict,
		ErrPermissionDenied,
		ErrProjectRootNotFound,
		ErrWorktreeDetectionFailed,
	}

	seen := make(map[ErrorType]bool)
	for _, errType := range types {
		if seen[errType] {
			t.Errorf("Duplicate error type: %v", errType)
		}
		seen[errType] = true
	}
}
