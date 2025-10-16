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
		// Note: godotenv creates an empty key for =value
		{
			name:        "invalid format - empty key handled by godotenv",
			content:     `=value`,
			expected:    map[string]string{"": "value"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temp file for testing
			tmpfile, err := os.CreateTemp("", "test*.env")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tt.content)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			loader := NewLoader()
			result, err := loader.LoadEnvFile(tmpfile.Name())

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
	// Create a temp file with no read permissions
	tmpfile, err := os.CreateTemp("", "test*.env")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	// Remove read permissions
	if err := os.Chmod(tmpfile.Name(), 0o000); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader()
	_, err = loader.LoadEnvFile(tmpfile.Name())

	if err == nil {
		t.Error("expected error for read failure, got nil")
	}

	if !strings.Contains(err.Error(), "failed to parse env file") {
		t.Errorf("expected 'failed to parse env file' in error, got: %v", err)
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

// Test new dotenv features provided by godotenv

func TestLoadEnvFile_MultilineValues(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected map[string]string
	}{
		{
			name: "multiline value with double quotes",
			content: `MULTILINE="line 1
line 2
line 3"`,
			expected: map[string]string{
				"MULTILINE": "line 1\nline 2\nline 3",
			},
		},
		{
			name: "multiline certificate",
			content: `SSL_CERT="-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAKLdQVPy90WjMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
-----END CERTIFICATE-----"`,
			expected: map[string]string{
				"SSL_CERT": "-----BEGIN CERTIFICATE-----\nMIIDXTCCAkWgAwIBAgIJAKLdQVPy90WjMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV\n-----END CERTIFICATE-----",
			},
		},
		{
			name: "mixed quotes in multiline",
			content: `JSON='{"key": "value",
"nested": "data"}'`,
			expected: map[string]string{
				"JSON": `{"key": "value",
"nested": "data"}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", "test*.env")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tt.content)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			loader := NewLoader()
			result, err := loader.LoadEnvFile(tmpfile.Name())
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			for key, expectedValue := range tt.expected {
				if actualValue, ok := result[key]; !ok {
					t.Errorf("missing key %q", key)
				} else if actualValue != expectedValue {
					t.Errorf("key %q:\nexpected: %q\ngot:      %q", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestLoadEnvFile_VariableExpansion(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected map[string]string
	}{
		{
			name: "basic variable expansion",
			content: `BASE_URL=http://localhost:3000
API_URL=${BASE_URL}/api
WS_URL=$BASE_URL/ws`,
			expected: map[string]string{
				"BASE_URL": "http://localhost:3000",
				"API_URL":  "http://localhost:3000/api",
				"WS_URL":   "http://localhost:3000/ws",
			},
		},
		// Note: godotenv partially processes ${VAR:-default} syntax
		{
			name: "expansion with default value (partial support)",
			content: `PORT=${UNDEFINED_PORT}
HOST=$UNDEFINED_HOST
DEFAULT_SYNTAX=${NOT_SET:-default}`,
			expected: map[string]string{
				"PORT":           "",
				"HOST":           "",
				"DEFAULT_SYNTAX": ":-default}",
			},
		},
		{
			name: "nested expansion",
			content: `A=hello
B=$A world
C=${B}!`,
			expected: map[string]string{
				"A": "hello",
				"B": "hello world",
				"C": "hello world!",
			},
		},
		{
			name: "expansion in quotes",
			content: `NAME=World
GREETING="Hello, ${NAME}!"`,
			expected: map[string]string{
				"NAME":     "World",
				"GREETING": "Hello, World!",
			},
		},
		{
			name: "no expansion in single quotes",
			content: `BASE=value
LITERAL='${BASE}'`,
			expected: map[string]string{
				"BASE":    "value",
				"LITERAL": "${BASE}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", "test*.env")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tt.content)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			loader := NewLoader()
			result, err := loader.LoadEnvFile(tmpfile.Name())
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
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

func TestLoadEnvFile_EscapeSequences(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected map[string]string
	}{
		{
			name:    "newline escape",
			content: `MESSAGE="Line 1\nLine 2\nLine 3"`,
			expected: map[string]string{
				"MESSAGE": "Line 1\nLine 2\nLine 3",
			},
		},
		// Note: godotenv doesn't process \t as tab, it removes the backslash
		{
			name:    "tab escape (godotenv behavior)",
			content: `TABS="Column1\tColumn2\tColumn3"`,
			expected: map[string]string{
				"TABS": "Column1tColumn2tColumn3",
			},
		},
		{
			name:    "backslash escape",
			content: `PATH="C:\\Program Files\\MyApp"`,
			expected: map[string]string{
				"PATH": "C:\\Program Files\\MyApp",
			},
		},
		{
			name:    "quote escape",
			content: `QUOTED="She said \"Hello\" to me"`,
			expected: map[string]string{
				"QUOTED": `She said "Hello" to me`,
			},
		},
		{
			name:    "no escape in single quotes",
			content: `LITERAL='Line 1\nLine 2'`,
			expected: map[string]string{
				"LITERAL": `Line 1\nLine 2`,
			},
		},
		// Note: godotenv handles \n and \" but not \t
		{
			name:    "mixed escapes",
			content: `MIXED="Tab:\t\"Quoted\"\nNewline"`,
			expected: map[string]string{
				"MIXED": "Tab:t\"Quoted\"\nNewline",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", "test*.env")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tt.content)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			loader := NewLoader()
			result, err := loader.LoadEnvFile(tmpfile.Name())
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			for key, expectedValue := range tt.expected {
				if actualValue, ok := result[key]; !ok {
					t.Errorf("missing key %q", key)
				} else if actualValue != expectedValue {
					t.Errorf("key %q:\nexpected: %q\ngot:      %q", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestLoadEnvFile_InlineComments(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected map[string]string
	}{
		{
			name: "inline comment after value",
			content: `PORT=3000 # API port
HOST=localhost # Development host`,
			expected: map[string]string{
				"PORT": "3000",
				"HOST": "localhost",
			},
		},
		{
			name: "no comment inside quotes",
			content: `URL="http://example.com#anchor"
MESSAGE='This is not # a comment'`,
			expected: map[string]string{
				"URL":     "http://example.com#anchor",
				"MESSAGE": "This is not # a comment",
			},
		},
		{
			name: "hash in value without spaces",
			content: `COLOR=#FF00FF
CHANNEL=#general`,
			expected: map[string]string{
				"COLOR":   "#FF00FF",
				"CHANNEL": "#general",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", "test*.env")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tt.content)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			loader := NewLoader()
			result, err := loader.LoadEnvFile(tmpfile.Name())
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
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
