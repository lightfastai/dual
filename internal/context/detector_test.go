package context

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Test helper functions

// mockGitCommand creates a mock git command function
func mockGitCommand(output string, err error) func(args ...string) (string, error) {
	return func(args ...string) (string, error) {
		return output, err
	}
}

// mockReadFile creates a mock readFile function with a map of paths to contents
func mockReadFile(files map[string]string) func(path string) ([]byte, error) {
	return func(path string) ([]byte, error) {
		if content, ok := files[path]; ok {
			return []byte(content), nil
		}
		return nil, os.ErrNotExist
	}
}

// mockGetwd creates a mock getwd function
func mockGetwd(dir string, err error) func() (string, error) {
	return func() (string, error) {
		return dir, err
	}
}

// TestDetectContext_GitBranch tests that git branch is detected with highest priority
func TestDetectContext_GitBranch(t *testing.T) {
	tests := []struct { //nolint:govet // Test struct optimization not critical
		name           string
		gitOutput      string
		gitError       error
		expectedResult string
	}{
		{
			name:           "valid branch name",
			gitOutput:      "main\n",
			gitError:       nil,
			expectedResult: "main",
		},
		{
			name:           "feature branch",
			gitOutput:      "feature/new-feature",
			gitError:       nil,
			expectedResult: "feature/new-feature",
		},
		{
			name:           "branch with spaces trimmed",
			gitOutput:      "  develop  \n",
			gitError:       nil,
			expectedResult: "develop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &Detector{
				gitCommand: mockGitCommand(tt.gitOutput, tt.gitError),
				readFile:   mockReadFile(map[string]string{}),
				getwd:      mockGetwd("/test/dir", nil),
			}

			result, err := detector.DetectContext()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expectedResult {
				t.Errorf("expected %q, got %q", tt.expectedResult, result)
			}
		})
	}
}

// TestDetectContext_DualContextFile tests .dual-context file detection
func TestDetectContext_DualContextFile(t *testing.T) {
	tests := []struct { //nolint:govet // Test struct optimization not critical
		name           string
		workingDir     string
		files          map[string]string
		gitError       error
		expectedResult string
	}{
		{
			name:       "context file in current directory",
			workingDir: "/project/subdir",
			files: map[string]string{
				"/project/subdir/.dual-context": "custom-context",
			},
			gitError:       fmt.Errorf("not a git repo"),
			expectedResult: "custom-context",
		},
		{
			name:       "context file in parent directory",
			workingDir: "/project/subdir/nested",
			files: map[string]string{
				"/project/.dual-context": "parent-context",
			},
			gitError:       fmt.Errorf("not a git repo"),
			expectedResult: "parent-context",
		},
		{
			name:       "context file walks up tree",
			workingDir: "/project/a/b/c",
			files: map[string]string{
				"/project/a/.dual-context": "found-context",
			},
			gitError:       fmt.Errorf("not a git repo"),
			expectedResult: "found-context",
		},
		{
			name:       "context file with whitespace trimmed",
			workingDir: "/project",
			files: map[string]string{
				"/project/.dual-context": "  trimmed-context  \n",
			},
			gitError:       fmt.Errorf("not a git repo"),
			expectedResult: "trimmed-context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &Detector{
				gitCommand: mockGitCommand("", tt.gitError),
				readFile:   mockReadFile(tt.files),
				getwd:      mockGetwd(tt.workingDir, nil),
			}

			result, err := detector.DetectContext()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expectedResult {
				t.Errorf("expected %q, got %q", tt.expectedResult, result)
			}
		})
	}
}

// TestDetectContext_DefaultFallback tests fallback to "default"
func TestDetectContext_DefaultFallback(t *testing.T) {
	tests := []struct { //nolint:govet // Test struct optimization not critical
		name           string
		workingDir     string
		gitError       error
		expectedResult string
	}{
		{
			name:           "no git and no context file",
			workingDir:     "/some/directory",
			gitError:       fmt.Errorf("not a git repo"),
			expectedResult: "default",
		},
		{
			name:           "empty git output",
			workingDir:     "/repo/detached",
			gitError:       nil,
			expectedResult: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &Detector{
				gitCommand: mockGitCommand("", tt.gitError),
				readFile:   mockReadFile(map[string]string{}),
				getwd:      mockGetwd(tt.workingDir, nil),
			}

			result, err := detector.DetectContext()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expectedResult {
				t.Errorf("expected %q, got %q", tt.expectedResult, result)
			}
		})
	}
}

// TestDetectContext_Priority tests that git branch takes priority over .dual-context
func TestDetectContext_Priority(t *testing.T) {
	detector := &Detector{
		gitCommand: mockGitCommand("git-branch\n", nil),
		readFile: mockReadFile(map[string]string{
			"/project/.dual-context": "file-context",
		}),
		getwd: mockGetwd("/project", nil),
	}

	result, err := detector.DetectContext()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "git-branch"
	if result != expected {
		t.Errorf("expected git branch %q to take priority, got %q", expected, result)
	}
}

// TestDetectContext_DetachedHEAD tests behavior in detached HEAD state
func TestDetectContext_DetachedHEAD(t *testing.T) {
	detector := &Detector{
		gitCommand: mockGitCommand("", nil), // Empty output simulates detached HEAD
		readFile: mockReadFile(map[string]string{
			"/repo/.dual-context": "context-from-file",
		}),
		getwd: mockGetwd("/repo", nil),
	}

	result, err := detector.DetectContext()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "context-from-file"
	if result != expected {
		t.Errorf("expected to fall back to .dual-context file, got %q", result)
	}
}

// TestDetectContext_EmptyContextFile tests error handling for empty .dual-context
func TestDetectContext_EmptyContextFile(t *testing.T) {
	detector := &Detector{
		gitCommand: mockGitCommand("", fmt.Errorf("not a git repo")),
		readFile: mockReadFile(map[string]string{
			"/project/.dual-context": "   \n",
		}),
		getwd: mockGetwd("/project", nil),
	}

	result, err := detector.DetectContext()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should fall back to default when .dual-context is empty
	expected := "default"
	if result != expected {
		t.Errorf("expected %q for empty context file, got %q", expected, result)
	}
}

// TestDetectContext_GetwdError tests error handling when getwd fails
func TestDetectContext_GetwdError(t *testing.T) {
	detector := &Detector{
		gitCommand: mockGitCommand("", fmt.Errorf("not a git repo")),
		readFile:   mockReadFile(map[string]string{}),
		getwd:      mockGetwd("", fmt.Errorf("permission denied")),
	}

	result, err := detector.DetectContext()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should fall back to default when getwd fails
	expected := "default"
	if result != expected {
		t.Errorf("expected %q when getwd fails, got %q", expected, result)
	}
}

// TestDetectContext_RootDirectory tests behavior at filesystem root
func TestDetectContext_RootDirectory(t *testing.T) {
	detector := &Detector{
		gitCommand: mockGitCommand("", fmt.Errorf("not a git repo")),
		readFile:   mockReadFile(map[string]string{}),
		getwd:      mockGetwd("/", nil),
	}

	result, err := detector.DetectContext()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "default"
	if result != expected {
		t.Errorf("expected %q at root directory, got %q", expected, result)
	}
}

// TestDetectContext_ConvenienceFunction tests the package-level DetectContext function
func TestDetectContext_ConvenienceFunction(t *testing.T) {
	// This is more of an integration test that ensures the convenience function works
	// It will use real git/filesystem, so we just check it doesn't panic
	result, err := DetectContext()
	if err != nil {
		t.Fatalf("unexpected error from convenience function: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result from convenience function")
	}
}

// TestFindDualContextFile_WalkUpTree tests the directory tree walking logic
func TestFindDualContextFile_WalkUpTree(t *testing.T) {
	tests := []struct { //nolint:govet // Test struct optimization not critical
		name        string
		workingDir  string
		files       map[string]string
		shouldFind  bool
		expectedCtx string
	}{
		{
			name:       "finds in current directory",
			workingDir: "/a/b/c",
			files: map[string]string{
				"/a/b/c/.dual-context": "context-c",
			},
			shouldFind:  true,
			expectedCtx: "context-c",
		},
		{
			name:       "finds in parent directory",
			workingDir: "/a/b/c",
			files: map[string]string{
				"/a/b/.dual-context": "context-b",
			},
			shouldFind:  true,
			expectedCtx: "context-b",
		},
		{
			name:       "finds in grandparent directory",
			workingDir: "/a/b/c",
			files: map[string]string{
				"/a/.dual-context": "context-a",
			},
			shouldFind:  true,
			expectedCtx: "context-a",
		},
		{
			name:       "prefers closest file",
			workingDir: "/a/b/c",
			files: map[string]string{
				"/a/.dual-context":     "context-a",
				"/a/b/.dual-context":   "context-b",
				"/a/b/c/.dual-context": "context-c",
			},
			shouldFind:  true,
			expectedCtx: "context-c",
		},
		{
			name:       "no file found",
			workingDir: "/a/b/c",
			files:      map[string]string{},
			shouldFind: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &Detector{
				readFile: mockReadFile(tt.files),
				getwd:    mockGetwd(tt.workingDir, nil),
			}

			result, err := detector.findDualContextFile()

			if tt.shouldFind {
				if err != nil {
					t.Fatalf("expected to find file, got error: %v", err)
				}
				if result != tt.expectedCtx {
					t.Errorf("expected context %q, got %q", tt.expectedCtx, result)
				}
			} else {
				if err == nil {
					t.Errorf("expected error when file not found, got result: %q", result)
				}
			}
		})
	}
}

// TestDetectGitBranch tests the git branch detection logic specifically
func TestDetectGitBranch(t *testing.T) {
	tests := []struct { //nolint:govet // Test struct optimization not critical
		name           string
		gitOutput      string
		gitError       error
		shouldSucceed  bool
		expectedBranch string
	}{
		{
			name:           "normal branch",
			gitOutput:      "main",
			gitError:       nil,
			shouldSucceed:  true,
			expectedBranch: "main",
		},
		{
			name:          "detached HEAD (empty output)",
			gitOutput:     "",
			gitError:      nil,
			shouldSucceed: false,
		},
		{
			name:          "git command error",
			gitOutput:     "",
			gitError:      fmt.Errorf("not a git repository"),
			shouldSucceed: false,
		},
		{
			name:           "branch with trailing newline",
			gitOutput:      "feature/test\n",
			gitError:       nil,
			shouldSucceed:  true,
			expectedBranch: "feature/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &Detector{
				gitCommand: mockGitCommand(tt.gitOutput, tt.gitError),
			}

			result, err := detector.detectGitBranch()

			if tt.shouldSucceed {
				if err != nil {
					t.Fatalf("expected success, got error: %v", err)
				}
				if result != tt.expectedBranch {
					t.Errorf("expected branch %q, got %q", tt.expectedBranch, result)
				}
			} else {
				if err == nil {
					t.Errorf("expected error, got success with result: %q", result)
				}
			}
		})
	}
}

// Integration test with real filesystem
func TestDetectContext_Integration(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()

	// Create subdirectories
	subDir := filepath.Join(tmpDir, "sub", "nested")
	err := os.MkdirAll(subDir, 0o755)
	if err != nil {
		t.Fatalf("failed to create test directories: %v", err)
	}

	// Create a .dual-context file in the temp directory
	contextFile := filepath.Join(tmpDir, ".dual-context")
	err = os.WriteFile(contextFile, []byte("test-context"), 0o644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Create a detector that will look in the nested directory
	detector := &Detector{
		gitCommand: mockGitCommand("", fmt.Errorf("not a git repo")),
		readFile:   os.ReadFile,
		getwd:      mockGetwd(subDir, nil),
	}

	result, err := detector.DetectContext()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "test-context"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
