package registry_test

import (
	"fmt"
	"log"
	"os"

	"github.com/lightfastai/dual/internal/registry"
)

// Example demonstrates basic registry usage
func Example() {
	// Set up temporary directory as project root
	projectRoot, err := os.MkdirTemp("", "dual-example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(projectRoot)

	// Load or create registry
	reg, err := registry.LoadRegistry(projectRoot)
	if err != nil {
		log.Fatal(err)
	}
	defer reg.Close()

	// Create a new context
	projectPath := "/home/user/myproject"
	contextName := "feature-branch"
	contextPath := "/home/user/myproject/worktree"

	err = reg.SetContext(projectPath, contextName, contextPath)
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
	fmt.Printf("Path: %s\n", ctx.Path)

	// Output:
	// Context: feature-branch
	// Path: /home/user/myproject/worktree
}

// ExampleRegistry_SetContext demonstrates creating contexts
func ExampleRegistry_SetContext() {
	// Set up temporary directory as project root
	projectRoot, err := os.MkdirTemp("", "dual-example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(projectRoot)

	reg, _ := registry.LoadRegistry(projectRoot)
	defer reg.Close()

	// Create contexts for different projects
	reg.SetContext("/project1", "main", "/project1/main")
	reg.SetContext("/project2", "main", "/project2/main")
	reg.SetContext("/project3", "feature", "/project3/feature")

	// List projects
	projects := reg.GetAllProjects()
	fmt.Printf("Total projects: %d\n", len(projects))

	// Output:
	// Total projects: 3
}

// ExampleRegistry_ListContexts demonstrates listing all contexts for a project
func ExampleRegistry_ListContexts() {
	// Set up temporary directory as project root
	projectRoot, err := os.MkdirTemp("", "dual-example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(projectRoot)

	reg, _ := registry.LoadRegistry(projectRoot)
	defer reg.Close()

	// Add multiple contexts
	projectPath := "/home/user/myproject"
	reg.SetContext(projectPath, "main", "")
	reg.SetContext(projectPath, "feature-1", "")
	reg.SetContext(projectPath, "feature-2", "")

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
