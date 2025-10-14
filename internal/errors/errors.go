package errors

import (
	"fmt"
	"strings"
)

// ErrorType represents the category of error
type ErrorType int

const (
	// Config errors
	ErrConfigNotFound ErrorType = iota
	ErrConfigInvalid
	ErrConfigExists

	// Registry errors
	ErrRegistryCorrupted
	ErrContextNotFound
	ErrServiceNotFound
	ErrPortConflict

	// Runtime errors
	ErrServiceNotDetected
	ErrContextDetectionFailed
	ErrPortCalculationFailed
	ErrCommandFailed

	// Environment errors
	ErrEnvNotFound
	ErrEnvParseFailed
	ErrEnvConflict

	// General errors
	ErrPermissionDenied
	ErrProjectRootNotFound
	ErrWorktreeDetectionFailed
)

// Error represents a structured error with context and helpful messages
type Error struct {
	Type    ErrorType
	Message string
	Context map[string]string
	Cause   error
	Fixes   []string
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Message == "" {
		return "unknown error"
	}
	return e.Message
}

// Unwrap implements error unwrapping for errors.Is and errors.As
func (e *Error) Unwrap() error {
	return e.Cause
}

// Format returns a formatted, human-readable error message with colors and context
func (e *Error) Format() string {
	var buf strings.Builder

	// Error header with red color
	buf.WriteString("\033[31mError:\033[0m ")
	buf.WriteString(e.Message)
	buf.WriteString("\n")

	// Context information (if any)
	if len(e.Context) > 0 {
		buf.WriteString("\n")
		for k, v := range e.Context {
			buf.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}

	// Cause (if any)
	if e.Cause != nil {
		buf.WriteString("\n")
		buf.WriteString("Cause: ")
		buf.WriteString(e.Cause.Error())
		buf.WriteString("\n")
	}

	// Fix suggestions
	if len(e.Fixes) > 0 {
		buf.WriteString("\n")
		buf.WriteString("\033[33mHow to fix:\033[0m\n")
		for _, fix := range e.Fixes {
			buf.WriteString("  • ")
			buf.WriteString(fix)
			buf.WriteString("\n")
		}
	}

	return buf.String()
}

// New creates a new Error with the given type and message
func New(errType ErrorType, message string) *Error {
	return &Error{
		Type:    errType,
		Message: message,
		Context: make(map[string]string),
		Fixes:   []string{},
	}
}

// WithContext adds context key-value pairs to the error
func (e *Error) WithContext(key, value string) *Error {
	if e.Context == nil {
		e.Context = make(map[string]string)
	}
	e.Context[key] = value
	return e
}

// WithCause adds a cause error
func (e *Error) WithCause(cause error) *Error {
	e.Cause = cause
	return e
}

// WithFix adds a fix suggestion
func (e *Error) WithFix(fix string) *Error {
	e.Fixes = append(e.Fixes, fix)
	return e
}

// WithFixes adds multiple fix suggestions
func (e *Error) WithFixes(fixes ...string) *Error {
	e.Fixes = append(e.Fixes, fixes...)
	return e
}
