package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLoadRegistry_EmptyFile tests loading when no registry exists
func TestLoadRegistry_EmptyFile(t *testing.T) {
	// Use a temporary directory as project root
	projectRoot := t.TempDir()

	registry, err := LoadRegistry(projectRoot)
	if err != nil {
		t.Fatalf("LoadRegistry() failed: %v", err)
	}
	defer registry.Close()

	if registry == nil {
		t.Fatal("LoadRegistry() returned nil registry")
	}

	if registry.Projects == nil {
		t.Fatal("LoadRegistry() returned registry with nil Projects map")
	}

	if len(registry.Projects) != 0 {
		t.Errorf("Expected empty registry, got %d projects", len(registry.Projects))
	}
}

// TestLoadRegistry_CorruptFile tests handling of corrupt registry files
func TestLoadRegistry_CorruptFile(t *testing.T) {
	// Use a temporary directory as project root
	projectRoot := t.TempDir()

	// Create registry directory
	registryDir := filepath.Join(projectRoot, ".dual")
	if err := os.MkdirAll(registryDir, 0o755); err != nil {
		t.Fatalf("Failed to create registry directory: %v", err)
	}

	// Write corrupt JSON
	registryPath := filepath.Join(registryDir, "registry.json")
	if err := os.WriteFile(registryPath, []byte("{corrupt json"), 0o644); err != nil {
		t.Fatalf("Failed to write corrupt registry: %v", err)
	}

	// Should return empty registry without error
	registry, err := LoadRegistry(projectRoot)
	if err != nil {
		t.Fatalf("LoadRegistry() failed on corrupt file: %v", err)
	}
	defer registry.Close()

	if len(registry.Projects) != 0 {
		t.Errorf("Expected empty registry after corruption, got %d projects", len(registry.Projects))
	}
}

// TestLoadRegistry_ValidFile tests loading a valid registry
func TestLoadRegistry_ValidFile(t *testing.T) {
	// Use a temporary directory as project root
	projectRoot := t.TempDir()

	// Create registry directory
	registryDir := filepath.Join(projectRoot, ".dual")
	if err := os.MkdirAll(registryDir, 0o755); err != nil {
		t.Fatalf("Failed to create registry directory: %v", err)
	}

	// Create valid registry
	testRegistry := Registry{
		Projects: map[string]Project{
			"/test/project": {
				Contexts: map[string]Context{
					"main": {
						Created:  time.Now(),
						BasePort: 4100,
						Path:     "/test/project",
					},
				},
			},
		},
	}

	data, err := json.MarshalIndent(testRegistry, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test registry: %v", err)
	}

	registryPath := filepath.Join(registryDir, "registry.json")
	if err := os.WriteFile(registryPath, data, 0o644); err != nil {
		t.Fatalf("Failed to write test registry: %v", err)
	}

	// Load and verify
	registry, err := LoadRegistry(projectRoot)
	if err != nil {
		t.Fatalf("LoadRegistry() failed: %v", err)
	}
	defer registry.Close()

	if len(registry.Projects) != 1 {
		t.Errorf("Expected 1 project, got %d", len(registry.Projects))
	}

	project, exists := registry.Projects["/test/project"]
	if !exists {
		t.Fatal("Expected project '/test/project' not found")
	}

	if len(project.Contexts) != 1 {
		t.Errorf("Expected 1 context, got %d", len(project.Contexts))
	}

	context, exists := project.Contexts["main"]
	if !exists {
		t.Fatal("Expected context 'main' not found")
	}

	if context.BasePort != 4100 {
		t.Errorf("Expected base port 4100, got %d", context.BasePort)
	}
}

// TestSaveRegistry tests saving the registry atomically
func TestSaveRegistry(t *testing.T) {
	// Use a temporary directory as project root
	projectRoot := t.TempDir()

	registry := &Registry{
		Projects: map[string]Project{
			"/test/project": {
				Contexts: map[string]Context{
					"feature": {
						Created:  time.Now(),
						BasePort: 4200,
						Path:     "/test/project/feature",
					},
				},
			},
		},
		projectRoot: projectRoot,
	}

	// Save registry
	if err := registry.SaveRegistry(); err != nil {
		t.Fatalf("SaveRegistry() failed: %v", err)
	}

	// Verify file exists
	registryPath := filepath.Join(projectRoot, ".dual", "registry.json")
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		t.Fatal("Registry file was not created")
	}

	// Verify temp file was removed
	tempFile := registryPath + ".tmp"
	if _, err := os.Stat(tempFile); !os.IsNotExist(err) {
		t.Error("Temporary file was not cleaned up")
	}

	// Load and verify
	loadedRegistry, err := LoadRegistry(projectRoot)
	if err != nil {
		t.Fatalf("Failed to load saved registry: %v", err)
	}
	defer loadedRegistry.Close()

	if len(loadedRegistry.Projects) != 1 {
		t.Errorf("Expected 1 project, got %d", len(loadedRegistry.Projects))
	}

	project := loadedRegistry.Projects["/test/project"]
	context := project.Contexts["feature"]
	if context.BasePort != 4200 {
		t.Errorf("Expected base port 4200, got %d", context.BasePort)
	}
}

// TestGetContext tests retrieving a context
func TestGetContext(t *testing.T) {
	registry := &Registry{
		Projects: map[string]Project{
			"/test/project": {
				Contexts: map[string]Context{
					"main": {
						Created:  time.Now(),
						BasePort: 4100,
						Path:     "/test/project",
					},
				},
			},
		},
	}

	// Test existing context
	context, err := registry.GetContext("/test/project", "main")
	if err != nil {
		t.Fatalf("GetContext() failed: %v", err)
	}

	if context.BasePort != 4100 {
		t.Errorf("Expected base port 4100, got %d", context.BasePort)
	}

	// Test non-existent project
	_, err = registry.GetContext("/nonexistent", "main")
	if err != ErrProjectNotFound {
		t.Errorf("Expected ErrProjectNotFound, got %v", err)
	}

	// Test non-existent context
	_, err = registry.GetContext("/test/project", "nonexistent")
	if err != ErrContextNotFound {
		t.Errorf("Expected ErrContextNotFound, got %v", err)
	}
}

// TestSetContext tests creating and updating contexts
func TestSetContext(t *testing.T) {
	registry := &Registry{
		Projects: make(map[string]Project),
	}

	// Create new context
	err := registry.SetContext("/test/project", "feature", 4200, "/test/project/feature")
	if err != nil {
		t.Fatalf("SetContext() failed: %v", err)
	}

	// Verify context was created
	context, err := registry.GetContext("/test/project", "feature")
	if err != nil {
		t.Fatalf("GetContext() failed after SetContext: %v", err)
	}

	if context.BasePort != 4200 {
		t.Errorf("Expected base port 4200, got %d", context.BasePort)
	}

	if context.Path != "/test/project/feature" {
		t.Errorf("Expected path '/test/project/feature', got '%s'", context.Path)
	}

	// Update existing context
	err = registry.SetContext("/test/project", "feature", 4300, "/test/project/feature2")
	if err != nil {
		t.Fatalf("SetContext() failed on update: %v", err)
	}

	// Verify update
	context, err = registry.GetContext("/test/project", "feature")
	if err != nil {
		t.Fatalf("GetContext() failed after update: %v", err)
	}

	if context.BasePort != 4300 {
		t.Errorf("Expected updated base port 4300, got %d", context.BasePort)
	}
}

// TestDeleteContext tests removing contexts
func TestDeleteContext(t *testing.T) {
	registry := &Registry{
		Projects: map[string]Project{
			"/test/project": {
				Contexts: map[string]Context{
					"feature1": {BasePort: 4100},
					"feature2": {BasePort: 4200},
				},
			},
		},
	}

	// Delete one context
	err := registry.DeleteContext("/test/project", "feature1")
	if err != nil {
		t.Fatalf("DeleteContext() failed: %v", err)
	}

	// Verify deletion
	_, err = registry.GetContext("/test/project", "feature1")
	if err != ErrContextNotFound {
		t.Errorf("Expected ErrContextNotFound after deletion, got %v", err)
	}

	// Verify other context still exists
	_, err = registry.GetContext("/test/project", "feature2")
	if err != nil {
		t.Errorf("Other context should still exist: %v", err)
	}

	// Delete last context (should remove project)
	err = registry.DeleteContext("/test/project", "feature2")
	if err != nil {
		t.Fatalf("DeleteContext() failed on last context: %v", err)
	}

	// Verify project was removed
	if _, exists := registry.Projects["/test/project"]; exists {
		t.Error("Project should be removed when last context is deleted")
	}

	// Test deleting from non-existent project
	err = registry.DeleteContext("/nonexistent", "feature")
	if err != ErrProjectNotFound {
		t.Errorf("Expected ErrProjectNotFound, got %v", err)
	}
}

// TestListContexts tests listing all contexts for a project
func TestListContexts(t *testing.T) {
	registry := &Registry{
		Projects: map[string]Project{
			"/test/project": {
				Contexts: map[string]Context{
					"main":     {BasePort: 4100},
					"feature1": {BasePort: 4200},
					"feature2": {BasePort: 4300},
				},
			},
		},
	}

	contexts, err := registry.ListContexts("/test/project")
	if err != nil {
		t.Fatalf("ListContexts() failed: %v", err)
	}

	if len(contexts) != 3 {
		t.Errorf("Expected 3 contexts, got %d", len(contexts))
	}

	// Verify all contexts are present
	expectedContexts := []string{"main", "feature1", "feature2"}
	for _, name := range expectedContexts {
		if _, exists := contexts[name]; !exists {
			t.Errorf("Expected context '%s' not found", name)
		}
	}

	// Test non-existent project
	_, err = registry.ListContexts("/nonexistent")
	if err != ErrProjectNotFound {
		t.Errorf("Expected ErrProjectNotFound, got %v", err)
	}
}

// TestFindNextAvailablePort tests finding the next available port
func TestFindNextAvailablePort(t *testing.T) {
	tests := []struct { //nolint:govet // Test struct optimization not critical
		name         string
		registry     *Registry
		expectedPort int
	}{
		{
			name: "Empty registry",
			registry: &Registry{
				Projects: make(map[string]Project),
			},
			expectedPort: 4100,
		},
		{
			name: "One port used",
			registry: &Registry{
				Projects: map[string]Project{
					"/test/project": {
						Contexts: map[string]Context{
							"main": {BasePort: 4100},
						},
					},
				},
			},
			expectedPort: 4200,
		},
		{
			name: "Multiple ports used",
			registry: &Registry{
				Projects: map[string]Project{
					"/test/project1": {
						Contexts: map[string]Context{
							"main": {BasePort: 4100},
							"dev":  {BasePort: 4200},
						},
					},
					"/test/project2": {
						Contexts: map[string]Context{
							"main": {BasePort: 4300},
						},
					},
				},
			},
			expectedPort: 4400,
		},
		{
			name: "Gap in ports",
			registry: &Registry{
				Projects: map[string]Project{
					"/test/project": {
						Contexts: map[string]Context{
							"main": {BasePort: 4100},
							"dev":  {BasePort: 4300},
						},
					},
				},
			},
			expectedPort: 4200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := tt.registry.FindNextAvailablePort()
			if port != tt.expectedPort {
				t.Errorf("Expected port %d, got %d", tt.expectedPort, port)
			}
		})
	}
}

// TestGetAllProjects tests listing all projects
func TestGetAllProjects(t *testing.T) {
	registry := &Registry{
		Projects: map[string]Project{
			"/test/project3": {Contexts: map[string]Context{"main": {BasePort: 4100}}},
			"/test/project1": {Contexts: map[string]Context{"main": {BasePort: 4200}}},
			"/test/project2": {Contexts: map[string]Context{"main": {BasePort: 4300}}},
		},
	}

	projects := registry.GetAllProjects()

	if len(projects) != 3 {
		t.Errorf("Expected 3 projects, got %d", len(projects))
	}

	// Verify sorted order
	expected := []string{"/test/project1", "/test/project2", "/test/project3"}
	for i, project := range projects {
		if project != expected[i] {
			t.Errorf("Expected project[%d] = '%s', got '%s'", i, expected[i], project)
		}
	}
}

// TestContextExists tests checking context existence
func TestContextExists(t *testing.T) {
	registry := &Registry{
		Projects: map[string]Project{
			"/test/project": {
				Contexts: map[string]Context{
					"main": {BasePort: 4100},
				},
			},
		},
	}

	// Test existing context
	if !registry.ContextExists("/test/project", "main") {
		t.Error("Expected context to exist")
	}

	// Test non-existent context
	if registry.ContextExists("/test/project", "nonexistent") {
		t.Error("Expected context to not exist")
	}

	// Test non-existent project
	if registry.ContextExists("/nonexistent", "main") {
		t.Error("Expected context to not exist in non-existent project")
	}
}

// TestConcurrentAccess tests thread safety
func TestConcurrentAccess(t *testing.T) {
	registry := &Registry{
		Projects: make(map[string]Project),
	}

	// Spawn multiple goroutines to set contexts concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			projectPath := "/test/project"
			contextName := "feature"
			basePort := 4100 + (idx * 100)
			err := registry.SetContext(projectPath, contextName, basePort, "")
			if err != nil {
				t.Errorf("Concurrent SetContext failed: %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify registry is still valid
	context, err := registry.GetContext("/test/project", "feature")
	if err != nil {
		t.Fatalf("GetContext failed after concurrent access: %v", err)
	}

	// Should have one of the base ports (last write wins)
	if context.BasePort < 4100 || context.BasePort > 5000 {
		t.Errorf("Unexpected base port after concurrent access: %d", context.BasePort)
	}
}

// TestGetRegistryPath tests registry path generation
func TestGetRegistryPath(t *testing.T) {
	projectRoot := "/test/project"

	path, err := GetRegistryPath(projectRoot)
	if err != nil {
		t.Fatalf("GetRegistryPath() failed: %v", err)
	}

	expected := filepath.Join(projectRoot, ".dual", "registry.json")
	if path != expected {
		t.Errorf("Expected path '%s', got '%s'", expected, path)
	}
}

// TestRegistryJSONFormat validates the JSON format matches the expected schema
func TestRegistryJSONFormat(t *testing.T) {
	projectRoot := t.TempDir()

	registry := &Registry{
		Projects: map[string]Project{
			"/absolute/project/path": {
				Contexts: map[string]Context{
					"main": {
						Created:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
						BasePort: 4100,
						Path:     "/absolute/context/path",
					},
					"feature": {
						Created:  time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
						BasePort: 4200,
					},
				},
			},
		},
		projectRoot: projectRoot,
	}

	// Save registry
	if err := registry.SaveRegistry(); err != nil {
		t.Fatalf("SaveRegistry() failed: %v", err)
	}

	// Read the raw JSON
	registryPath, _ := GetRegistryPath(projectRoot)
	data, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("Failed to read registry file: %v", err)
	}

	// Verify JSON structure
	var rawRegistry map[string]interface{}
	if err := json.Unmarshal(data, &rawRegistry); err != nil {
		t.Fatalf("Failed to parse registry JSON: %v", err)
	}

	// Verify top-level structure
	projects, ok := rawRegistry["projects"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'projects' field at top level")
	}

	// Verify project structure
	project, ok := projects["/absolute/project/path"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected project at '/absolute/project/path'")
	}

	contexts, ok := project["contexts"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'contexts' field in project")
	}

	// Verify context with path
	mainContext, ok := contexts["main"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'main' context")
	}

	if mainContext["basePort"].(float64) != 4100 {
		t.Errorf("Expected basePort 4100, got %v", mainContext["basePort"])
	}

	if mainContext["path"].(string) != "/absolute/context/path" {
		t.Errorf("Expected path '/absolute/context/path', got %v", mainContext["path"])
	}

	if mainContext["created"] == nil {
		t.Error("Expected 'created' field")
	}

	// Verify context without path (should omit path field)
	featureContext, ok := contexts["feature"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'feature' context")
	}

	if featureContext["basePort"].(float64) != 4200 {
		t.Errorf("Expected basePort 4200, got %v", featureContext["basePort"])
	}

	// Path should be omitted when empty (omitempty tag)
	if _, exists := featureContext["path"]; exists {
		pathVal := featureContext["path"]
		// Path should either not exist or be empty string
		if pathStr, ok := pathVal.(string); ok && pathStr != "" {
			t.Errorf("Expected path to be omitted or empty, got %v", pathVal)
		}
	}
}
