package registry_test

import (
	"fmt"
	"log"
	"os"

	"github.com/lightfastai/dual/internal/registry"
)

// Example demonstrates basic registry usage
func Example() {
	// Set up temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "dual-example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Override home directory for this example
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Load or create registry
	reg, err := registry.LoadRegistry()
	if err != nil {
		log.Fatal(err)
	}

	// Create a new context
	projectPath := "/home/user/myproject"
	contextName := "feature-branch"
	basePort := reg.FindNextAvailablePort()

	err = reg.SetContext(projectPath, contextName, basePort, "/home/user/myproject/worktree")
	if err != nil {
		log.Fatal(err)
	}

	// Save the registry
	err = reg.SaveRegistry()
	if err != nil {
		log.Fatal(err)
	}

	// Retrieve the context
	ctx, err := reg.GetContext(projectPath, contextName)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Context: %s\n", contextName)
	fmt.Printf("Base Port: %d\n", ctx.BasePort)
	fmt.Printf("Path: %s\n", ctx.Path)

	// Find next available port
	nextPort := reg.FindNextAvailablePort()
	fmt.Printf("Next available port: %d\n", nextPort)

	// Output:
	// Context: feature-branch
	// Base Port: 4100
	// Path: /home/user/myproject/worktree
	// Next available port: 4200
}

// ExampleRegistry_FindNextAvailablePort demonstrates port allocation
func ExampleRegistry_FindNextAvailablePort() {
	// Set up temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "dual-example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	reg, _ := registry.LoadRegistry()

	// First port (empty registry)
	port1 := reg.FindNextAvailablePort()
	fmt.Printf("First port: %d\n", port1)

	// Create a context
	reg.SetContext("/project1", "main", port1, "")

	// Second port
	port2 := reg.FindNextAvailablePort()
	fmt.Printf("Second port: %d\n", port2)

	// Create another context
	reg.SetContext("/project2", "main", port2, "")

	// Third port
	port3 := reg.FindNextAvailablePort()
	fmt.Printf("Third port: %d\n", port3)

	// Output:
	// First port: 4100
	// Second port: 4200
	// Third port: 4300
}

// ExampleRegistry_ListContexts demonstrates listing all contexts for a project
func ExampleRegistry_ListContexts() {
	// Set up temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "dual-example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	reg, _ := registry.LoadRegistry()

	// Add multiple contexts
	projectPath := "/home/user/myproject"
	reg.SetContext(projectPath, "main", 4100, "")
	reg.SetContext(projectPath, "feature-1", 4200, "")
	reg.SetContext(projectPath, "feature-2", 4300, "")

	// List all contexts
	contexts, err := reg.ListContexts(projectPath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d contexts\n", len(contexts))
	// Note: We can't print the actual contexts as map iteration order is not guaranteed

	// Output:
	// Found 3 contexts
}
