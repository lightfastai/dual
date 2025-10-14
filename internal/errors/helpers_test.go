package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestConfigNotFound(t *testing.T) {
	err := ConfigNotFound()

	if err.Type != ErrConfigNotFound {
		t.Errorf("Type = %v, want %v", err.Type, ErrConfigNotFound)
	}

	if !strings.Contains(err.Message, "Configuration file not found") {
		t.Errorf("Message should contain 'Configuration file not found', got %q", err.Message)
	}

	if len(err.Fixes) == 0 {
		t.Error("Should have fix suggestions")
	}
}

func TestConfigInvalid(t *testing.T) {
	cause := errors.New("yaml parse error")
	err := ConfigInvalid("invalid YAML syntax", cause)

	if err.Type != ErrConfigInvalid {
		t.Errorf("Type = %v, want %v", err.Type, ErrConfigInvalid)
	}

	if err.Context["Reason"] != "invalid YAML syntax" {
		t.Errorf("Context[Reason] = %q, want %q", err.Context["Reason"], "invalid YAML syntax")
	}

	if err.Cause != cause {
		t.Error("Cause should be set")
	}
}

func TestConfigExists(t *testing.T) {
	err := ConfigExists("dual.config.yml")

	if err.Type != ErrConfigExists {
		t.Errorf("Type = %v, want %v", err.Type, ErrConfigExists)
	}

	if err.Context["File"] != "dual.config.yml" {
		t.Errorf("Context[File] = %q, want %q", err.Context["File"], "dual.config.yml")
	}

	formatted := err.Format()
	if !strings.Contains(formatted, "--force") {
		t.Error("Should suggest --force flag")
	}
}

func TestContextNotFound(t *testing.T) {
	tests := []struct {
		name        string
		contextName string
		wantFixes   []string
	}{
		{
			name:        "default context",
			contextName: "default",
			wantFixes:   []string{"dual context create default"},
		},
		{
			name:        "feature branch context",
			contextName: "feature-auth",
			wantFixes:   []string{"dual context create feature-auth", "dual context list"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ContextNotFound(tt.contextName, "/home/user/project")

			if err.Type != ErrContextNotFound {
				t.Errorf("Type = %v, want %v", err.Type, ErrContextNotFound)
			}

			if err.Context["Context"] != tt.contextName {
				t.Errorf("Context[Context] = %q, want %q", err.Context["Context"], tt.contextName)
			}

			formatted := err.Format()
			for _, want := range tt.wantFixes {
				if !strings.Contains(formatted, want) {
					t.Errorf("Format() should contain %q", want)
				}
			}
		})
	}
}

func TestServiceNotFoundInConfig(t *testing.T) {
	tests := []struct {
		name               string
		serviceName        string
		availableServices  []string
		shouldSuggestMatch bool
	}{
		{
			name:               "exact match available",
			serviceName:        "api",
			availableServices:  []string{"www", "api-gateway", "auth"},
			shouldSuggestMatch: true,
		},
		{
			name:               "partial match",
			serviceName:        "web",
			availableServices:  []string{"www", "deus", "auth"},
			shouldSuggestMatch: true,
		},
		{
			name:               "no match",
			serviceName:        "unknown",
			availableServices:  []string{"www", "deus", "auth"},
			shouldSuggestMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ServiceNotFoundInConfig(tt.serviceName, tt.availableServices)

			if err.Type != ErrServiceNotFound {
				t.Errorf("Type = %v, want %v", err.Type, ErrServiceNotFound)
			}

			formatted := err.Format()

			// Should contain available services
			if len(tt.availableServices) > 0 {
				if !strings.Contains(formatted, "Available services") {
					t.Error("Should list available services")
				}
			}

			// Should suggest adding the service
			if !strings.Contains(formatted, "dual service add") {
				t.Error("Should suggest adding the service")
			}
		})
	}
}

func TestServiceNotDetected(t *testing.T) {
	serviceNames := []string{"www", "deus", "auth"}
	servicePaths := []string{"apps/www", "apps/deus", "apps/auth"}

	err := ServiceNotDetected("/home/user/project/docs", serviceNames, servicePaths)

	if err.Type != ErrServiceNotDetected {
		t.Errorf("Type = %v, want %v", err.Type, ErrServiceNotDetected)
	}

	if err.Context["Current directory"] != "/home/user/project/docs" {
		t.Error("Should include current directory in context")
	}

	formatted := err.Format()

	// Should show available services
	if !strings.Contains(formatted, "www (apps/www)") {
		t.Error("Should list services with paths")
	}

	// Should suggest fixes
	if !strings.Contains(formatted, "cd into a service directory") {
		t.Error("Should suggest cd into service directory")
	}

	if !strings.Contains(formatted, "--service") {
		t.Error("Should suggest --service flag")
	}
}

func TestPortConflict(t *testing.T) {
	tests := []struct {
		name        string
		port        int
		processInfo string
	}{
		{
			name:        "with process info",
			port:        4201,
			processInfo: "node (PID 12345)",
		},
		{
			name:        "without process info",
			port:        4201,
			processInfo: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PortConflict(tt.port, tt.processInfo)

			if err.Type != ErrPortConflict {
				t.Errorf("Type = %v, want %v", err.Type, ErrPortConflict)
			}

			if err.Context["Port"] != "4201" {
				t.Error("Should include port in context")
			}

			if tt.processInfo != "" {
				if err.Context["Process"] != tt.processInfo {
					t.Error("Should include process info when available")
				}
			}

			formatted := err.Format()
			if !strings.Contains(formatted, "4201") {
				t.Error("Should mention port number")
			}
		})
	}
}

func TestCommandFailed(t *testing.T) {
	err := CommandFailed("pnpm dev", 1, "Error: Cannot find module")

	if err.Type != ErrCommandFailed {
		t.Errorf("Type = %v, want %v", err.Type, ErrCommandFailed)
	}

	if err.Context["Command"] != "pnpm dev" {
		t.Error("Should include command in context")
	}

	if err.Context["Exit code"] != "1" {
		t.Error("Should include exit code in context")
	}

	if !strings.Contains(err.Context["Error output"], "Cannot find module") {
		t.Error("Should include stderr in context")
	}
}

func TestCommandFailed_LongStderr(t *testing.T) {
	longStderr := strings.Repeat("error line\n", 100)
	err := CommandFailed("pnpm dev", 1, longStderr)

	// Should truncate long stderr
	if len(err.Context["Error output"]) > 600 {
		t.Error("Should truncate long stderr output")
	}

	if !strings.Contains(err.Context["Error output"], "...") {
		t.Error("Should indicate truncation with ...")
	}
}

func TestEnvNotFound(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		isWorktree bool
		wantFix    string
	}{
		{
			name:       "in worktree",
			path:       "/main/repo/.env",
			isWorktree: true,
			wantFix:    "main repository",
		},
		{
			name:       "not in worktree",
			path:       "/project/.env",
			isWorktree: false,
			wantFix:    "Create the environment file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EnvNotFound(tt.path, tt.isWorktree)

			if err.Type != ErrEnvNotFound {
				t.Errorf("Type = %v, want %v", err.Type, ErrEnvNotFound)
			}

			if err.Context["Expected path"] != tt.path {
				t.Error("Should include path in context")
			}

			formatted := err.Format()
			if !strings.Contains(formatted, tt.wantFix) {
				t.Errorf("Should contain fix mentioning %q", tt.wantFix)
			}
		})
	}
}

func TestEnvParseFailed(t *testing.T) {
	cause := errors.New("invalid syntax")
	err := EnvParseFailed("/project/.env", 42, cause)

	if err.Type != ErrEnvParseFailed {
		t.Errorf("Type = %v, want %v", err.Type, ErrEnvParseFailed)
	}

	if err.Context["File"] != "/project/.env" {
		t.Error("Should include file path in context")
	}

	if err.Context["Line"] != "42" {
		t.Error("Should include line number in context")
	}

	if err.Cause != cause {
		t.Error("Should include cause")
	}
}

func TestProjectRootNotFound(t *testing.T) {
	err := ProjectRootNotFound("/home/user/somewhere")

	if err.Type != ErrProjectRootNotFound {
		t.Errorf("Type = %v, want %v", err.Type, ErrProjectRootNotFound)
	}

	if err.Context["Current directory"] != "/home/user/somewhere" {
		t.Error("Should include current directory in context")
	}

	formatted := err.Format()
	if !strings.Contains(formatted, "dual init") {
		t.Error("Should suggest running dual init")
	}
}

func TestWorktreeDetectionFailed(t *testing.T) {
	cause := errors.New("git command failed")
	err := WorktreeDetectionFailed(cause)

	if err.Type != ErrWorktreeDetectionFailed {
		t.Errorf("Type = %v, want %v", err.Type, ErrWorktreeDetectionFailed)
	}

	if err.Cause != cause {
		t.Error("Should include cause")
	}
}

func TestPermissionDenied(t *testing.T) {
	cause := errors.New("permission denied")
	err := PermissionDenied("/var/dual/registry.json", "write file", cause)

	if err.Type != ErrPermissionDenied {
		t.Errorf("Type = %v, want %v", err.Type, ErrPermissionDenied)
	}

	if err.Context["Path"] != "/var/dual/registry.json" {
		t.Error("Should include path in context")
	}

	if !strings.Contains(err.Message, "write file") {
		t.Error("Should include operation in message")
	}

	if err.Cause != cause {
		t.Error("Should include cause")
	}
}
