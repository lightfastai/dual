package env

import (
	"bufio"
	"fmt"
	"os"
	"strings"
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

	// Read file
	content, err := l.readFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read env file: %w", err)
	}

	// Parse content
	return l.parseEnvContent(string(content))
}

// parseEnvContent parses environment variable content into a map
// Format: KEY=value (one per line)
// Supports:
// - Comments starting with #
// - Empty lines
// - Quoted values
// - Export prefix (export KEY=value)
func (l *Loader) parseEnvContent(content string) (map[string]string, error) {
	envMap := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Trim whitespace
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Remove 'export ' prefix if present
		line = strings.TrimPrefix(line, "export ")
		line = strings.TrimSpace(line)

		// Find the first = sign
		idx := strings.Index(line, "=")
		if idx == -1 {
			// Line doesn't contain =, skip it
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		// Validate key
		if key == "" {
			return nil, fmt.Errorf("invalid env format at line %d: empty key", lineNum)
		}

		// Handle quoted values
		value = unquoteValue(value)

		envMap[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning env file: %w", err)
	}

	return envMap, nil
}

// unquoteValue removes surrounding quotes from a value
func unquoteValue(value string) string {
	if len(value) < 2 {
		return value
	}

	// Handle double quotes
	if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		return value[1 : len(value)-1]
	}

	// Handle single quotes
	if strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`) {
		return value[1 : len(value)-1]
	}

	return value
}

// LoadEnvFile is a convenience function that creates a loader and loads a file
func LoadEnvFile(path string) (map[string]string, error) {
	loader := NewLoader()
	return loader.LoadEnvFile(path)
}
