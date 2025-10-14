package logger

import (
	"fmt"
	"os"
)

var (
	// VerboseEnabled controls whether Verbose messages are displayed
	VerboseEnabled bool
	// DebugEnabled controls whether Debug messages are displayed (also enables Verbose)
	DebugEnabled bool
)

// Init initializes the logger based on flags and environment variables
func Init(verbose, debug bool) {
	VerboseEnabled = verbose || debug
	DebugEnabled = debug

	// Support DUAL_DEBUG environment variable
	if os.Getenv("DUAL_DEBUG") == "1" {
		DebugEnabled = true
		VerboseEnabled = true
	}
}

// Verbose prints verbose messages to stderr (shown when --verbose or --debug is enabled)
func Verbose(format string, args ...interface{}) {
	if VerboseEnabled {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}

// Debug prints debug messages to stderr (shown only when --debug is enabled)
func Debug(format string, args ...interface{}) {
	if DebugEnabled {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}

// Info prints informational messages to stderr (always shown)
func Info(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// Success prints success messages with a checkmark to stderr (always shown)
func Success(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "\u2713 "+format+"\n", args...)
}

// Error prints error messages to stderr (always shown)
func Error(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}
