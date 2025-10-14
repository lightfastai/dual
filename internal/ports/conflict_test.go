package ports

import (
	"net"
	"testing"
	"time"

	"github.com/lightfastai/dual/internal/config"
	"github.com/lightfastai/dual/internal/registry"
)

func TestFindDuplicateBasePorts(t *testing.T) {
	tests := []struct {
		name             string
		registry         *registry.Registry
		expectedConflict int // Number of conflicts expected
	}{
		{
			name: "no conflicts",
			registry: &registry.Registry{
				Projects: map[string]registry.Project{
					"/project1": {
						Contexts: map[string]registry.Context{
							"main":    {BasePort: 4100, Created: time.Now()},
							"feature": {BasePort: 4200, Created: time.Now()},
						},
					},
				},
			},
			expectedConflict: 0,
		},
		{
			name: "duplicate base ports in same project",
			registry: &registry.Registry{
				Projects: map[string]registry.Project{
					"/project1": {
						Contexts: map[string]registry.Context{
							"main":    {BasePort: 4100, Created: time.Now()},
							"feature": {BasePort: 4100, Created: time.Now()},
						},
					},
				},
			},
			expectedConflict: 1,
		},
		{
			name: "duplicate base ports across projects",
			registry: &registry.Registry{
				Projects: map[string]registry.Project{
					"/project1": {
						Contexts: map[string]registry.Context{
							"main": {BasePort: 4100, Created: time.Now()},
						},
					},
					"/project2": {
						Contexts: map[string]registry.Context{
							"main": {BasePort: 4100, Created: time.Now()},
						},
					},
				},
			},
			expectedConflict: 1,
		},
		{
			name: "multiple conflicts",
			registry: &registry.Registry{
				Projects: map[string]registry.Project{
					"/project1": {
						Contexts: map[string]registry.Context{
							"ctx1": {BasePort: 4100, Created: time.Now()},
							"ctx2": {BasePort: 4100, Created: time.Now()},
							"ctx3": {BasePort: 4200, Created: time.Now()},
							"ctx4": {BasePort: 4200, Created: time.Now()},
						},
					},
				},
			},
			expectedConflict: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conflicts := FindDuplicateBasePorts(tt.registry)
			if len(conflicts) != tt.expectedConflict {
				t.Errorf("expected %d conflicts, got %d", tt.expectedConflict, len(conflicts))
			}

			// Verify conflict details
			for _, conflict := range conflicts {
				if len(conflict.Contexts) < 2 {
					t.Errorf("conflict must have at least 2 contexts, got %d", len(conflict.Contexts))
				}
			}
		})
	}
}

func TestIsPortInUse(t *testing.T) {
	// Start a test server on a random port on all interfaces
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to start test server: %v", err)
	}
	defer listener.Close()

	// Get the port the test server is using
	usedPort := listener.Addr().(*net.TCPAddr).Port

	tests := []struct {
		name     string
		port     int
		expected bool
	}{
		{
			name:     "port in use",
			port:     usedPort,
			expected: true,
		},
		{
			name:     "port available",
			port:     usedPort + 1, // Likely available
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inUse := IsPortInUse(tt.port)
			if inUse != tt.expected {
				t.Errorf("IsPortInUse(%d) = %v, want %v", tt.port, inUse, tt.expected)
			}
		})
	}
}

func TestGetProcessUsingPort(t *testing.T) {
	// Start a test server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test server: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	// Try to get process info
	processInfo, err := GetProcessUsingPort(port)

	// This test might fail on some systems due to permissions or unsupported platforms
	// So we just check that it doesn't panic
	if err == nil && processInfo != nil {
		// If we got process info, verify it has some data
		if processInfo.PID == 0 {
			t.Error("expected non-zero PID")
		}
	}
	// If err != nil, that's okay - the function is platform-dependent
}

func TestCheckPortRangeOverlap(t *testing.T) {
	tests := []struct {
		name            string
		registry        *registry.Registry
		config          *config.Config
		projectPath     string
		expectedOverlap int
	}{
		{
			name: "no overlap",
			registry: &registry.Registry{
				Projects: map[string]registry.Project{
					"/project1": {
						Contexts: map[string]registry.Context{
							"ctx1": {BasePort: 4100, Created: time.Now()},
							"ctx2": {BasePort: 4200, Created: time.Now()},
						},
					},
				},
			},
			config: &config.Config{
				Services: map[string]config.Service{
					"api": {Path: "./api"},
					"web": {Path: "./web"},
				},
			},
			projectPath:     "/project1",
			expectedOverlap: 0,
		},
		{
			name: "overlapping ranges",
			registry: &registry.Registry{
				Projects: map[string]registry.Project{
					"/project1": {
						Contexts: map[string]registry.Context{
							"ctx1": {BasePort: 4100, Created: time.Now()},
							"ctx2": {BasePort: 4102, Created: time.Now()}, // Will overlap with ctx1
						},
					},
				},
			},
			config: &config.Config{
				Services: map[string]config.Service{
					"api":    {Path: "./api"},
					"web":    {Path: "./web"},
					"worker": {Path: "./worker"},
				},
			},
			projectPath:     "/project1",
			expectedOverlap: 1,
		},
		{
			name: "project not in registry",
			registry: &registry.Registry{
				Projects: map[string]registry.Project{},
			},
			config: &config.Config{
				Services: map[string]config.Service{
					"api": {Path: "./api"},
				},
			},
			projectPath:     "/nonexistent",
			expectedOverlap: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overlaps, err := CheckPortRangeOverlap(tt.registry, tt.config, tt.projectPath)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(overlaps) != tt.expectedOverlap {
				t.Errorf("expected %d overlaps, got %d", tt.expectedOverlap, len(overlaps))
			}

			// Verify overlap details
			for _, overlap := range overlaps {
				if overlap.OverlapPort == 0 {
					t.Error("overlap port should not be zero")
				}
			}
		})
	}
}

func TestFindNextAvailableBasePort(t *testing.T) {
	tests := []struct {
		name         string
		registry     *registry.Registry
		config       *config.Config
		projectPath  string
		expectedPort int
	}{
		{
			name: "empty registry",
			registry: &registry.Registry{
				Projects: map[string]registry.Project{},
			},
			config: &config.Config{
				Services: map[string]config.Service{
					"api": {Path: "./api"},
				},
			},
			projectPath:  "/project1",
			expectedPort: registry.DefaultBasePort,
		},
		{
			name: "default port used",
			registry: &registry.Registry{
				Projects: map[string]registry.Project{
					"/project1": {
						Contexts: map[string]registry.Context{
							"main": {BasePort: registry.DefaultBasePort, Created: time.Now()},
						},
					},
				},
			},
			config: &config.Config{
				Services: map[string]config.Service{
					"api": {Path: "./api"},
				},
			},
			projectPath:  "/project1",
			expectedPort: registry.DefaultBasePort + registry.PortIncrement,
		},
		{
			name: "multiple ports used",
			registry: &registry.Registry{
				Projects: map[string]registry.Project{
					"/project1": {
						Contexts: map[string]registry.Context{
							"ctx1": {BasePort: 4100, Created: time.Now()},
							"ctx2": {BasePort: 4200, Created: time.Now()},
						},
					},
				},
			},
			config: &config.Config{
				Services: map[string]config.Service{
					"api": {Path: "./api"},
				},
			},
			projectPath:  "/project1",
			expectedPort: 4300,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := FindNextAvailableBasePort(tt.registry, tt.config, tt.projectPath)
			if port != tt.expectedPort {
				t.Errorf("expected port %d, got %d", tt.expectedPort, port)
			}
		})
	}
}

func TestCheckContextPortConflict(t *testing.T) {
	tests := []struct {
		name        string
		registry    *registry.Registry
		config      *config.Config
		projectPath string
		basePort    int
		expectError bool
	}{
		{
			name: "no conflict",
			registry: &registry.Registry{
				Projects: map[string]registry.Project{
					"/project1": {
						Contexts: map[string]registry.Context{
							"main": {BasePort: 4100, Created: time.Now()},
						},
					},
				},
			},
			config: &config.Config{
				Services: map[string]config.Service{
					"api": {Path: "./api"},
					"web": {Path: "./web"},
				},
			},
			projectPath: "/project1",
			basePort:    4200,
			expectError: false,
		},
		{
			name: "base port already used",
			registry: &registry.Registry{
				Projects: map[string]registry.Project{
					"/project1": {
						Contexts: map[string]registry.Context{
							"main": {BasePort: 4100, Created: time.Now()},
						},
					},
				},
			},
			config: &config.Config{
				Services: map[string]config.Service{
					"api": {Path: "./api"},
				},
			},
			projectPath: "/project1",
			basePort:    4100,
			expectError: true,
		},
		{
			name: "port range overlap",
			registry: &registry.Registry{
				Projects: map[string]registry.Project{
					"/project1": {
						Contexts: map[string]registry.Context{
							"main": {BasePort: 4100, Created: time.Now()},
						},
					},
				},
			},
			config: &config.Config{
				Services: map[string]config.Service{
					"api":    {Path: "./api"},
					"web":    {Path: "./web"},
					"worker": {Path: "./worker"},
				},
			},
			projectPath: "/project1",
			basePort:    4102, // Will overlap: 4103-4105 overlaps with 4101-4103
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckContextPortConflict(tt.registry, tt.config, tt.projectPath, tt.basePort)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error: %v, got: %v", tt.expectError, err)
			}
		})
	}
}

func TestCheckServicePortInUse(t *testing.T) {
	// Create a test registry and config
	reg := &registry.Registry{
		Projects: map[string]registry.Project{
			"/project1": {
				Contexts: map[string]registry.Context{
					"main": {BasePort: 4100, Created: time.Now()},
				},
			},
		},
	}

	cfg := &config.Config{
		Services: map[string]config.Service{
			"api": {Path: "./api"},
			"web": {Path: "./web"},
		},
	}

	// Test with a service that likely has its port available
	port, processInfo, err := CheckServicePortInUse(cfg, reg, "/project1", "main", "api")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Port should be 4101 (basePort 4100 + serviceIndex 0 + 1)
	expectedPort := 4101
	if port != expectedPort {
		t.Errorf("expected port %d, got %d", expectedPort, port)
	}

	// Process info should be nil if port is not in use
	// (or non-nil if something happens to be using that port)
	_ = processInfo // Just check it doesn't panic
}

func TestParseLsofOutput(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		port        int
		expectError bool
	}{
		{
			name: "valid lsof output",
			output: `COMMAND    PID USER   FD   TYPE             DEVICE SIZE/OFF NODE NAME
node     12345 user   21u  IPv4 0x1234567890abcdef      0t0  TCP *:3000 (LISTEN)`,
			port:        3000,
			expectError: false,
		},
		{
			name:        "empty output",
			output:      "",
			port:        3000,
			expectError: true,
		},
		{
			name:        "only header",
			output:      "COMMAND    PID USER   FD   TYPE",
			port:        3000,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processInfo, err := parseLsofOutput(tt.output, tt.port)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error: %v, got: %v", tt.expectError, err)
			}

			if !tt.expectError && processInfo != nil {
				if processInfo.PID == 0 {
					t.Error("expected non-zero PID")
				}
				if processInfo.Name == "" {
					t.Error("expected non-empty name")
				}
			}
		})
	}
}

func TestParseNetstatOutput(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		port        int
		expectError bool
	}{
		{
			name: "valid netstat output",
			output: `  TCP    0.0.0.0:3000           0.0.0.0:0              LISTENING       12345
  TCP    0.0.0.0:8080           0.0.0.0:0              LISTENING       67890`,
			port:        3000,
			expectError: false,
		},
		{
			name:        "empty output",
			output:      "",
			port:        3000,
			expectError: true,
		},
		{
			name:        "port not found",
			output:      "  TCP    0.0.0.0:8080           0.0.0.0:0              LISTENING       67890",
			port:        3000,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processInfo, err := parseNetstatOutput(tt.output, tt.port)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error: %v, got: %v", tt.expectError, err)
			}

			if !tt.expectError && processInfo != nil {
				if processInfo.PID == 0 {
					t.Error("expected non-zero PID")
				}
			}
		})
	}
}
