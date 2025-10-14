package service

import (
	"testing"
	"time"

	"github.com/lightfastai/dual/internal/config"
	"github.com/lightfastai/dual/internal/registry"
)

// createTestRegistry creates a registry with test data
func createTestRegistry() *registry.Registry {
	reg := &registry.Registry{
		Projects: map[string]registry.Project{
			"/project": {
				Contexts: map[string]registry.Context{
					"main": {
						BasePort: 4100,
						Created:  time.Now(),
					},
					"feature": {
						BasePort: 4200,
						Created:  time.Now(),
					},
					"develop": {
						BasePort: 4300,
						Created:  time.Now(),
					},
				},
			},
			"/other-project": {
				Contexts: map[string]registry.Context{
					"main": {
						BasePort: 5000,
						Created:  time.Now(),
					},
				},
			},
		},
	}
	return reg
}

// createTestConfig creates a config with test services
func createTestConfig() *config.Config {
	return &config.Config{
		Version: 1,
		Services: map[string]config.Service{
			"web": {
				Path:    "apps/web",
				EnvFile: ".env.local",
			},
			"api": {
				Path:    "apps/api",
				EnvFile: ".env.local",
			},
			"worker": {
				Path: "apps/worker",
			},
		},
	}
}

// TestCalculatePort_BasicCalculation tests basic port calculation
func TestCalculatePort_BasicCalculation(t *testing.T) {
	cfg := createTestConfig()
	reg := createTestRegistry()
	calc := NewCalculator()

	tests := []struct { //nolint:govet // Test struct optimization not critical
		name        string
		projectPath string
		contextName string
		serviceName string
		expected    int
		wantErr     bool
	}{
		{
			name:        "web service - main context",
			projectPath: "/project",
			contextName: "main",
			serviceName: "web",
			expected:    4102, // basePort 4100 + index 1 (alphabetically: api=0, web=1, worker=2) + 1
			wantErr:     false,
		},
		{
			name:        "api service - main context",
			projectPath: "/project",
			contextName: "main",
			serviceName: "api",
			expected:    4101, // basePort 4100 + index 0 + 1
			wantErr:     false,
		},
		{
			name:        "worker service - main context",
			projectPath: "/project",
			contextName: "main",
			serviceName: "worker",
			expected:    4103, // basePort 4100 + index 2 + 1
			wantErr:     false,
		},
		{
			name:        "web service - feature context",
			projectPath: "/project",
			contextName: "feature",
			serviceName: "web",
			expected:    4202, // basePort 4200 + index 1 + 1
			wantErr:     false,
		},
		{
			name:        "api service - develop context",
			projectPath: "/project",
			contextName: "develop",
			serviceName: "api",
			expected:    4301, // basePort 4300 + index 0 + 1
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port, err := calc.CalculatePort(cfg, reg, tt.projectPath, tt.contextName, tt.serviceName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got port: %d", port)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if port != tt.expected {
					t.Errorf("expected port %d, got %d", tt.expected, port)
				}
			}
		})
	}
}

// TestCalculatePort_ErrorHandling tests error cases
func TestCalculatePort_ErrorHandling(t *testing.T) {
	cfg := createTestConfig()
	reg := createTestRegistry()
	calc := NewCalculator()

	tests := []struct { //nolint:govet // Test struct optimization not critical
		name        string
		projectPath string
		contextName string
		serviceName string
		expectedErr error
	}{
		{
			name:        "context not found",
			projectPath: "/project",
			contextName: "nonexistent",
			serviceName: "web",
			expectedErr: ErrContextNotFound,
		},
		{
			name:        "project not found",
			projectPath: "/nonexistent",
			contextName: "main",
			serviceName: "web",
			expectedErr: ErrContextNotFound,
		},
		{
			name:        "service not found",
			projectPath: "/project",
			contextName: "main",
			serviceName: "nonexistent",
			expectedErr: ErrServiceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := calc.CalculatePort(cfg, reg, tt.projectPath, tt.contextName, tt.serviceName)

			if err != tt.expectedErr {
				t.Errorf("expected error %v, got %v", tt.expectedErr, err)
			}
		})
	}
}

// TestCalculatePort_DifferentProjects tests calculation across different projects
func TestCalculatePort_DifferentProjects(t *testing.T) {
	cfg := createTestConfig()
	reg := createTestRegistry()
	calc := NewCalculator()

	// Same service, different projects should have different ports
	port1, err := calc.CalculatePort(cfg, reg, "/project", "main", "api")
	if err != nil {
		t.Fatalf("unexpected error for project 1: %v", err)
	}

	port2, err := calc.CalculatePort(cfg, reg, "/other-project", "main", "api")
	if err != nil {
		t.Fatalf("unexpected error for project 2: %v", err)
	}

	if port1 == port2 {
		t.Errorf("expected different ports for different projects, both got %d", port1)
	}

	// Verify the actual values
	if port1 != 4101 {
		t.Errorf("expected port 4101 for /project, got %d", port1)
	}
	if port2 != 5001 {
		t.Errorf("expected port 5001 for /other-project, got %d", port2)
	}
}

// TestGetServiceIndex tests the service index calculation
func TestGetServiceIndex(t *testing.T) {
	cfg := createTestConfig()
	calc := NewCalculator()

	tests := []struct { //nolint:govet // Test struct optimization not critical
		serviceName string
		expected    int
	}{
		{"api", 0},    // First alphabetically
		{"web", 1},    // Second alphabetically
		{"worker", 2}, // Third alphabetically
	}

	for _, tt := range tests {
		t.Run(tt.serviceName, func(t *testing.T) {
			index := calc.getServiceIndex(cfg, tt.serviceName)
			if index != tt.expected {
				t.Errorf("expected index %d for %s, got %d", tt.expected, tt.serviceName, index)
			}
		})
	}
}

// TestGetServiceIndex_Nonexistent tests index for nonexistent service
func TestGetServiceIndex_Nonexistent(t *testing.T) {
	cfg := createTestConfig()
	calc := NewCalculator()

	index := calc.getServiceIndex(cfg, "nonexistent")
	if index != -1 {
		t.Errorf("expected -1 for nonexistent service, got %d", index)
	}
}

// TestGetServiceIndex_AlphabeticalOrdering tests that services are ordered alphabetically
func TestGetServiceIndex_AlphabeticalOrdering(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		Services: map[string]config.Service{
			"zebra":   {Path: "zebra"},
			"alpha":   {Path: "alpha"},
			"charlie": {Path: "charlie"},
			"bravo":   {Path: "bravo"},
		},
	}

	calc := NewCalculator()

	expected := map[string]int{
		"alpha":   0,
		"bravo":   1,
		"charlie": 2,
		"zebra":   3,
	}

	for serviceName, expectedIndex := range expected {
		index := calc.getServiceIndex(cfg, serviceName)
		if index != expectedIndex {
			t.Errorf("expected index %d for %s, got %d", expectedIndex, serviceName, index)
		}
	}
}

// TestCalculatePort_PortFormula tests that the formula port = basePort + serviceIndex + 1 is correct
func TestCalculatePort_PortFormula(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		Services: map[string]config.Service{
			"service-a": {Path: "a"},
			"service-b": {Path: "b"},
			"service-c": {Path: "c"},
		},
	}

	reg := &registry.Registry{
		Projects: map[string]registry.Project{
			"/test": {
				Contexts: map[string]registry.Context{
					"test": {
						BasePort: 3000,
						Created:  time.Now(),
					},
				},
			},
		},
	}

	calc := NewCalculator()

	tests := []struct { //nolint:govet // Test struct optimization not critical
		serviceName  string
		expectedPort int
	}{
		// Alphabetically: service-a (index 0), service-b (index 1), service-c (index 2)
		{"service-a", 3001}, // 3000 + 0 + 1
		{"service-b", 3002}, // 3000 + 1 + 1
		{"service-c", 3003}, // 3000 + 2 + 1
	}

	for _, tt := range tests {
		t.Run(tt.serviceName, func(t *testing.T) {
			port, err := calc.CalculatePort(cfg, reg, "/test", "test", tt.serviceName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if port != tt.expectedPort {
				t.Errorf("expected port %d, got %d", tt.expectedPort, port)
			}
		})
	}
}

// TestCalculatePort_ConvenienceFunction tests the package-level convenience function
func TestCalculatePort_ConvenienceFunction(t *testing.T) {
	cfg := createTestConfig()
	reg := createTestRegistry()

	port, err := CalculatePort(cfg, reg, "/project", "main", "api")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := 4101
	if port != expected {
		t.Errorf("expected port %d, got %d", expected, port)
	}
}

// TestGetServiceIndex_HelperFunction tests the package-level helper function
func TestGetServiceIndex_HelperFunction(t *testing.T) {
	cfg := createTestConfig()

	index := GetServiceIndex(cfg, "api")
	if index != 0 {
		t.Errorf("expected index 0, got %d", index)
	}
}

// TestCalculateAllPorts tests calculating ports for all services
func TestCalculateAllPorts(t *testing.T) {
	cfg := createTestConfig()
	reg := createTestRegistry()

	ports, err := CalculateAllPorts(cfg, reg, "/project", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := map[string]int{
		"api":    4101, // basePort 4100 + 0 + 1
		"web":    4102, // basePort 4100 + 1 + 1
		"worker": 4103, // basePort 4100 + 2 + 1
	}

	if len(ports) != len(expected) {
		t.Errorf("expected %d ports, got %d", len(expected), len(ports))
	}

	for serviceName, expectedPort := range expected {
		actualPort, exists := ports[serviceName]
		if !exists {
			t.Errorf("expected port for service %s, not found", serviceName)
			continue
		}
		if actualPort != expectedPort {
			t.Errorf("service %s: expected port %d, got %d", serviceName, expectedPort, actualPort)
		}
	}
}

// TestCalculateAllPorts_ErrorHandling tests error handling for CalculateAllPorts
func TestCalculateAllPorts_ErrorHandling(t *testing.T) {
	cfg := createTestConfig()
	reg := createTestRegistry()

	tests := []struct { //nolint:govet // Test struct optimization not critical
		name        string
		projectPath string
		contextName string
		expectedErr error
	}{
		{
			name:        "context not found",
			projectPath: "/project",
			contextName: "nonexistent",
			expectedErr: ErrContextNotFound,
		},
		{
			name:        "project not found",
			projectPath: "/nonexistent",
			contextName: "main",
			expectedErr: ErrContextNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CalculateAllPorts(cfg, reg, tt.projectPath, tt.contextName)
			if err != tt.expectedErr {
				t.Errorf("expected error %v, got %v", tt.expectedErr, err)
			}
		})
	}
}

// TestCalculateAllPorts_DifferentContexts tests that different contexts produce different ports
func TestCalculateAllPorts_DifferentContexts(t *testing.T) {
	cfg := createTestConfig()
	reg := createTestRegistry()

	mainPorts, err := CalculateAllPorts(cfg, reg, "/project", "main")
	if err != nil {
		t.Fatalf("unexpected error for main context: %v", err)
	}

	featurePorts, err := CalculateAllPorts(cfg, reg, "/project", "feature")
	if err != nil {
		t.Fatalf("unexpected error for feature context: %v", err)
	}

	// Check that all services have different ports between contexts
	for serviceName, mainPort := range mainPorts {
		featurePort, exists := featurePorts[serviceName]
		if !exists {
			t.Errorf("service %s not found in feature context", serviceName)
			continue
		}
		if mainPort == featurePort {
			t.Errorf("service %s has same port %d in both contexts", serviceName, mainPort)
		}
	}
}

// TestCalculatePort_Determinism tests that port calculation is deterministic
func TestCalculatePort_Determinism(t *testing.T) {
	cfg := createTestConfig()
	reg := createTestRegistry()
	calc := NewCalculator()

	// Calculate the same port multiple times
	const iterations = 10
	var ports []int

	for i := 0; i < iterations; i++ {
		port, err := calc.CalculatePort(cfg, reg, "/project", "main", "web")
		if err != nil {
			t.Fatalf("unexpected error on iteration %d: %v", i, err)
		}
		ports = append(ports, port)
	}

	// All ports should be identical
	firstPort := ports[0]
	for i, port := range ports {
		if port != firstPort {
			t.Errorf("iteration %d: expected port %d, got %d (determinism failed)", i, firstPort, port)
		}
	}
}

// TestCalculatePort_SingleService tests calculation with only one service
func TestCalculatePort_SingleService(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		Services: map[string]config.Service{
			"app": {Path: "app"},
		},
	}

	reg := &registry.Registry{
		Projects: map[string]registry.Project{
			"/single": {
				Contexts: map[string]registry.Context{
					"main": {
						BasePort: 8000,
						Created:  time.Now(),
					},
				},
			},
		},
	}

	calc := NewCalculator()
	port, err := calc.CalculatePort(cfg, reg, "/single", "main", "app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := 8001 // 8000 + 0 + 1
	if port != expected {
		t.Errorf("expected port %d, got %d", expected, port)
	}
}

// TestCalculatePort_ManyServices tests calculation with many services
func TestCalculatePort_ManyServices(t *testing.T) {
	// Create config with 10 services
	services := make(map[string]config.Service)
	for i := 0; i < 10; i++ {
		name := string(rune('a' + i))
		services[name] = config.Service{Path: name}
	}

	cfg := &config.Config{
		Version:  1,
		Services: services,
	}

	reg := &registry.Registry{
		Projects: map[string]registry.Project{
			"/many": {
				Contexts: map[string]registry.Context{
					"main": {
						BasePort: 6000,
						Created:  time.Now(),
					},
				},
			},
		},
	}

	calc := NewCalculator()

	// Verify each service gets the correct port
	expectedPorts := map[string]int{
		"a": 6001, "b": 6002, "c": 6003, "d": 6004, "e": 6005,
		"f": 6006, "g": 6007, "h": 6008, "i": 6009, "j": 6010,
	}

	for serviceName, expectedPort := range expectedPorts {
		port, err := calc.CalculatePort(cfg, reg, "/many", "main", serviceName)
		if err != nil {
			t.Fatalf("unexpected error for service %s: %v", serviceName, err)
		}
		if port != expectedPort {
			t.Errorf("service %s: expected port %d, got %d", serviceName, expectedPort, port)
		}
	}
}

// TestCalculateAllPorts_EmptyServices tests behavior with no services (edge case)
func TestCalculateAllPorts_EmptyServices(t *testing.T) {
	cfg := &config.Config{
		Version:  1,
		Services: map[string]config.Service{},
	}

	reg := &registry.Registry{
		Projects: map[string]registry.Project{
			"/empty": {
				Contexts: map[string]registry.Context{
					"main": {
						BasePort: 7000,
						Created:  time.Now(),
					},
				},
			},
		},
	}

	ports, err := CalculateAllPorts(cfg, reg, "/empty", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(ports) != 0 {
		t.Errorf("expected empty ports map, got %d entries", len(ports))
	}
}
