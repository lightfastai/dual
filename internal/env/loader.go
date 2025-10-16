package env

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Loader handles loading environment variables from files
type Loader struct {
	// readFile allows for dependency injection in tests
	readFile func(path string) ([]byte, error)
	// stat allows for dependency injection in tests
	stat func(path string) (os.FileInfo, error)
}

// NewLoader creates a new Loader with default implementations
func NewLoader() *Loader {
	return &Loader{
		readFile: os.ReadFile,
		stat:     os.Stat,
	}
}

// LoadEnvFile loads environment variables from a file into a map
// Returns an empty map if the file doesn't exist (non-fatal)
// Returns an error only for read failures or parse errors
// Now supports full dotenv spec including:
// - Multiline values
// - Variable expansion (${VAR}, $VAR)
// - Escape sequences in double quotes (\n, \t, \\, \")
// - Inline comments
// - Complex quoting
func (l *Loader) LoadEnvFile(path string) (map[string]string, error) {
	// Check if file exists
	if _, err := l.stat(path); err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist - this is OK, return empty map
			return make(map[string]string), nil
		}
		// Other stat error (permissions, etc)
		return nil, fmt.Errorf("failed to stat env file: %w", err)
	}

	// Use godotenv.Read which returns map[string]string
	// This provides full dotenv compatibility including:
	// - Multiline values with proper quote handling
	// - Variable expansion with ${VAR} and $VAR syntax
	// - Escape sequence processing in double-quoted strings
	// - Inline comment support
	env, err := godotenv.Read(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse env file: %w", err)
	}

	return env, nil
}

// LoadEnvFile is a convenience function that creates a loader and loads a file
func LoadEnvFile(path string) (map[string]string, error) {
	loader := NewLoader()
	return loader.LoadEnvFile(path)
}
