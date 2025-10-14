package env

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestLoadEnvFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expected    map[string]string
		expectError bool
	}{
		{
			name: "simple key-value pairs",
			content: `KEY1=value1
KEY2=value2
KEY3=value3`,
			expected: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
				"KEY3": "value3",
			},
			expectError: false,
		},
		{
			name: "with comments and empty lines",
			content: `# This is a comment
KEY1=value1

# Another comment
KEY2=value2

KEY3=value3`,
			expected: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
				"KEY3": "value3",
			},
			expectError: false,
		},
		{
			name: "with export prefix",
			content: `export KEY1=value1
export KEY2=value2
KEY3=value3`,
			expected: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
				"KEY3": "value3",
			},
			expectError: false,
		},
		{
			name: "with quoted values",
			content: `KEY1="value with spaces"
KEY2='single quoted'
KEY3=unquoted`,
			expected: map[string]string{
				"KEY1": "value with spaces",
				"KEY2": "single quoted",
				"KEY3": "unquoted",
			},
			expectError: false,
		},
		{
			name: "with whitespace",
			content: `  KEY1  =  value1
KEY2=value2
  KEY3=value3  `,
			expected: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
				"KEY3": "value3",
			},
			expectError: false,
		},
		{
			name:        "empty file",
			content:     ``,
			expected:    map[string]string{},
			expectError: false,
		},
		{
			name: "only comments",
			content: `# Comment 1
# Comment 2`,
			expected:    map[string]string{},
			expectError: false,
		},
		{
			name: "with empty values",
			content: `KEY1=
KEY2=value2`,
			expected: map[string]string{
				"KEY1": "",
				"KEY2": "value2",
			},
			expectError: false,
		},
		{
			name:        "invalid format - empty key",
			content:     `=value`,
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := &Loader{
				readFile: func(path string) ([]byte, error) {
					return []byte(tt.content), nil
				},
				stat: func(path string) (os.FileInfo, error) {
					return nil, nil // File exists
				},
			}

			result, err := loader.LoadEnvFile("test.env")

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d entries, got %d", len(tt.expected), len(result))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, ok := result[key]; !ok {
					t.Errorf("missing key %q", key)
				} else if actualValue != expectedValue {
					t.Errorf("key %q: expected %q, got %q", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestLoadEnvFile_FileNotFound(t *testing.T) {
	loader := &Loader{
		readFile: func(path string) ([]byte, error) {
			return nil, errors.New("should not be called")
		},
		stat: func(path string) (os.FileInfo, error) {
			return nil, os.ErrNotExist
		},
	}

	result, err := loader.LoadEnvFile("nonexistent.env")
	if err != nil {
		t.Errorf("expected no error for nonexistent file, got: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty map for nonexistent file, got %d entries", len(result))
	}
}

func TestLoadEnvFile_ReadError(t *testing.T) {
	loader := &Loader{
		readFile: func(path string) ([]byte, error) {
			return nil, errors.New("permission denied")
		},
		stat: func(path string) (os.FileInfo, error) {
			return nil, nil // File exists
		},
	}

	_, err := loader.LoadEnvFile("test.env")

	if err == nil {
		t.Error("expected error for read failure, got nil")
	}

	if !strings.Contains(err.Error(), "failed to read env file") {
		t.Errorf("expected 'failed to read env file' in error, got: %v", err)
	}
}

func TestUnquoteValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: `"quoted"`, expected: "quoted"},
		{input: `'quoted'`, expected: "quoted"},
		{input: `unquoted`, expected: "unquoted"},
		{input: `"with spaces"`, expected: "with spaces"},
		{input: `"`, expected: `"`},
		{input: `""`, expected: ""},
		{input: ``, expected: ""},
		{input: `"mismatched`, expected: `"mismatched`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := unquoteValue(tt.input)
			if result != tt.expected {
				t.Errorf("unquoteValue(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNewLoader(t *testing.T) {
	loader := NewLoader()

	if loader == nil {
		t.Fatal("NewLoader() returned nil")
	}

	if loader.readFile == nil {
		t.Error("readFile should not be nil")
	}

	if loader.stat == nil {
		t.Error("stat should not be nil")
	}
}

func TestLoadEnvFile_ConvenienceFunction(t *testing.T) {
	// This is more of an integration test
	// Just verify the function exists and doesn't panic
	_, _ = LoadEnvFile("/nonexistent/path")
}
