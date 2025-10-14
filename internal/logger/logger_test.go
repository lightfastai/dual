package logger

import (
	"bytes"
	"os"
	"testing"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name        string
		verbose     bool
		debug       bool
		envVar      string
		wantVerbose bool
		wantDebug   bool
	}{
		{
			name:        "no flags",
			verbose:     false,
			debug:       false,
			wantVerbose: false,
			wantDebug:   false,
		},
		{
			name:        "verbose flag",
			verbose:     true,
			debug:       false,
			wantVerbose: true,
			wantDebug:   false,
		},
		{
			name:        "debug flag",
			verbose:     false,
			debug:       true,
			wantVerbose: true,
			wantDebug:   true,
		},
		{
			name:        "both flags",
			verbose:     true,
			debug:       true,
			wantVerbose: true,
			wantDebug:   true,
		},
		{
			name:        "env var",
			verbose:     false,
			debug:       false,
			envVar:      "1",
			wantVerbose: true,
			wantDebug:   true,
		},
		{
			name:        "env var with verbose flag",
			verbose:     true,
			debug:       false,
			envVar:      "1",
			wantVerbose: true,
			wantDebug:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.envVar != "" {
				os.Setenv("DUAL_DEBUG", tt.envVar)
				defer os.Unsetenv("DUAL_DEBUG")
			} else {
				os.Unsetenv("DUAL_DEBUG")
			}

			// Reset state
			VerboseEnabled = false
			DebugEnabled = false

			// Execute
			Init(tt.verbose, tt.debug)

			// Verify
			if VerboseEnabled != tt.wantVerbose {
				t.Errorf("VerboseEnabled = %v, want %v", VerboseEnabled, tt.wantVerbose)
			}
			if DebugEnabled != tt.wantDebug {
				t.Errorf("DebugEnabled = %v, want %v", DebugEnabled, tt.wantDebug)
			}
		})
	}
}

func TestVerbose(t *testing.T) {
	tests := []struct {
		name        string
		verbose     bool
		debug       bool
		format      string
		args        []interface{}
		wantOutput  bool
		wantContent string
	}{
		{
			name:        "verbose enabled",
			verbose:     true,
			debug:       false,
			format:      "test message",
			wantOutput:  true,
			wantContent: "test message\n",
		},
		{
			name:        "debug enabled",
			verbose:     false,
			debug:       true,
			format:      "test message",
			wantOutput:  true,
			wantContent: "test message\n",
		},
		{
			name:        "neither enabled",
			verbose:     false,
			debug:       false,
			format:      "test message",
			wantOutput:  false,
			wantContent: "",
		},
		{
			name:        "with args",
			verbose:     true,
			debug:       false,
			format:      "test %s %d",
			args:        []interface{}{"message", 42},
			wantOutput:  true,
			wantContent: "test message 42\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			// Setup
			VerboseEnabled = tt.verbose || tt.debug
			DebugEnabled = tt.debug

			// Execute
			if tt.args != nil {
				Verbose(tt.format, tt.args...)
			} else {
				Verbose(tt.format)
			}

			// Restore stderr and read output
			w.Close()
			os.Stderr = oldStderr
			var buf bytes.Buffer
			buf.ReadFrom(r)

			// Verify
			got := buf.String()
			if tt.wantOutput && got != tt.wantContent {
				t.Errorf("Verbose() output = %q, want %q", got, tt.wantContent)
			}
			if !tt.wantOutput && got != "" {
				t.Errorf("Verbose() should not output, got %q", got)
			}
		})
	}
}

func TestDebug(t *testing.T) {
	tests := []struct {
		name        string
		debug       bool
		format      string
		args        []interface{}
		wantOutput  bool
		wantContent string
	}{
		{
			name:        "debug enabled",
			debug:       true,
			format:      "debug message",
			wantOutput:  true,
			wantContent: "debug message\n",
		},
		{
			name:        "debug disabled",
			debug:       false,
			format:      "debug message",
			wantOutput:  false,
			wantContent: "",
		},
		{
			name:        "with args",
			debug:       true,
			format:      "debug %s %d",
			args:        []interface{}{"value", 123},
			wantOutput:  true,
			wantContent: "debug value 123\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			// Setup
			DebugEnabled = tt.debug

			// Execute
			if tt.args != nil {
				Debug(tt.format, tt.args...)
			} else {
				Debug(tt.format)
			}

			// Restore stderr and read output
			w.Close()
			os.Stderr = oldStderr
			var buf bytes.Buffer
			buf.ReadFrom(r)

			// Verify
			got := buf.String()
			if tt.wantOutput && got != tt.wantContent {
				t.Errorf("Debug() output = %q, want %q", got, tt.wantContent)
			}
			if !tt.wantOutput && got != "" {
				t.Errorf("Debug() should not output, got %q", got)
			}
		})
	}
}

func TestInfo(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Execute
	Info("info message")

	// Restore stderr and read output
	w.Close()
	os.Stderr = oldStderr
	var buf bytes.Buffer
	buf.ReadFrom(r)

	// Verify
	want := "info message\n"
	if got := buf.String(); got != want {
		t.Errorf("Info() output = %q, want %q", got, want)
	}
}

func TestSuccess(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Execute
	Success("success message")

	// Restore stderr and read output
	w.Close()
	os.Stderr = oldStderr
	var buf bytes.Buffer
	buf.ReadFrom(r)

	// Verify
	want := "\u2713 success message\n"
	if got := buf.String(); got != want {
		t.Errorf("Success() output = %q, want %q", got, want)
	}
}

func TestError(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Execute
	Error("error message")

	// Restore stderr and read output
	w.Close()
	os.Stderr = oldStderr
	var buf bytes.Buffer
	buf.ReadFrom(r)

	// Verify
	want := "Error: error message\n"
	if got := buf.String(); got != want {
		t.Errorf("Error() output = %q, want %q", got, want)
	}
}
