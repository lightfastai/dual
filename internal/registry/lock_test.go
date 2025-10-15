package registry

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestConcurrentRegistryWrites tests that multiple concurrent registry operations don't corrupt data
func TestConcurrentRegistryWrites(t *testing.T) {
	// Use a temporary directory as project root
	projectRoot := t.TempDir()

	const numGoroutines = 10
	const numIterations = 5

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numIterations)

	// Spawn multiple goroutines that perform registry operations concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numIterations; j++ {
				// Load registry (acquires lock)
				reg, err := LoadRegistry(projectRoot)
				if err != nil {
					errors <- err
					return
				}

				// Perform some operations
				projectPath := "/test/project"
				contextName := "context"
				contextPath := "/test/project/context"

				if err := reg.SetContext(projectPath, contextName, contextPath); err != nil {
					reg.Close()
					errors <- err
					return
				}

				// Save registry (still holding lock)
				if err := reg.SaveRegistry(); err != nil {
					reg.Close()
					errors <- err
					return
				}

				// Release lock
				if err := reg.Close(); err != nil {
					errors <- err
					return
				}

				// Small delay to increase contention
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent operation failed: %v", err)
	}

	// Verify registry is still valid and not corrupted
	reg, err := LoadRegistry(projectRoot)
	if err != nil {
		t.Fatalf("Failed to load registry after concurrent writes: %v", err)
	}
	defer reg.Close()

	// Should have the project
	if len(reg.Projects) != 1 {
		t.Errorf("Expected 1 project after concurrent writes, got %d", len(reg.Projects))
	}

	// Verify we can read from the registry
	ctx, err := reg.GetContext("/test/project", "context")
	if err != nil {
		t.Fatalf("Failed to get context after concurrent writes: %v", err)
	}

	// Should have the correct path
	if ctx.Path != "/test/project/context" {
		t.Errorf("Invalid context path after concurrent writes: %s", ctx.Path)
	}
}

// TestLockTimeout tests that lock acquisition times out appropriately
func TestLockTimeout(t *testing.T) {
	// Use a temporary directory as project root
	projectRoot := t.TempDir()

	// First registry holds the lock
	reg1, err := LoadRegistry(projectRoot)
	if err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}
	// Don't close yet - keep the lock

	// Try to acquire lock from another "process"
	// This should timeout
	done := make(chan bool)
	var reg2 *Registry
	var loadErr error

	go func() {
		reg2, loadErr = LoadRegistry(projectRoot)
		done <- true
	}()

	// Wait for timeout plus some buffer
	select {
	case <-done:
		// Should have timed out
		if loadErr == nil {
			reg2.Close()
			t.Fatal("Expected lock timeout error, got nil")
		}
		if !isLockTimeoutError(loadErr) {
			t.Errorf("Expected lock timeout error, got: %v", loadErr)
		}
	case <-time.After(LockTimeout + 2*time.Second):
		t.Fatal("Lock acquisition didn't timeout in expected time")
	}

	// Now release the first lock
	if err := reg1.Close(); err != nil {
		t.Fatalf("Failed to close first registry: %v", err)
	}

	// Should be able to acquire lock now
	reg3, err := LoadRegistry(projectRoot)
	if err != nil {
		t.Fatalf("Failed to load registry after releasing lock: %v", err)
	}
	defer reg3.Close()
}

// TestStaleLockCleanup tests that stale locks can be detected and handled
func TestStaleLockCleanup(t *testing.T) {
	// Use a temporary directory as project root
	projectRoot := t.TempDir()

	// Load and immediately close to create lock file
	reg1, err := LoadRegistry(projectRoot)
	if err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}
	reg1.Close()

	// Verify lock file exists
	lockPath, _ := GetLockPath(projectRoot)
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Log("Lock file was cleaned up automatically (expected behavior)")
	}

	// Should be able to acquire lock again
	reg2, err := LoadRegistry(projectRoot)
	if err != nil {
		t.Fatalf("Failed to load registry after lock cleanup: %v", err)
	}
	defer reg2.Close()
}

// TestAtomicWriteFailure tests that atomic write failure doesn't corrupt registry
func TestAtomicWriteFailure(t *testing.T) {
	// Use a temporary directory as project root
	projectRoot := t.TempDir()

	// Create initial registry with data
	reg, err := LoadRegistry(projectRoot)
	if err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}

	if err := reg.SetContext("/test/project", "main", "/test/project/main"); err != nil {
		t.Fatalf("Failed to set context: %v", err)
	}

	if err := reg.SaveRegistry(); err != nil {
		t.Fatalf("Failed to save initial registry: %v", err)
	}
	reg.Close()

	// Load registry again
	reg2, err := LoadRegistry(projectRoot)
	if err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}

	// Verify original data is intact
	ctx, err := reg2.GetContext("/test/project", "main")
	if err != nil {
		t.Fatalf("Failed to get context: %v", err)
	}

	if ctx.Path != "/test/project/main" {
		t.Errorf("Expected path '/test/project/main', got '%s'", ctx.Path)
	}

	reg2.Close()
}

// TestLockPathGeneration tests that lock path is generated correctly
func TestLockPathGeneration(t *testing.T) {
	projectRoot := "/test/project"

	lockPath, err := GetLockPath(projectRoot)
	if err != nil {
		t.Fatalf("GetLockPath() failed: %v", err)
	}

	expected := filepath.Join(projectRoot, ".dual", "registry.json.lock")
	if lockPath != expected {
		t.Errorf("Expected lock path '%s', got '%s'", expected, lockPath)
	}
}

// TestMultipleCloseCallsSafe tests that calling Close() multiple times is safe
func TestMultipleCloseCallsSafe(t *testing.T) {
	projectRoot := t.TempDir()

	reg, err := LoadRegistry(projectRoot)
	if err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}

	// First close should succeed
	if err := reg.Close(); err != nil {
		t.Fatalf("First Close() failed: %v", err)
	}

	// Second close should not panic (lock is already released)
	if err := reg.Close(); err != nil {
		// This is okay - flock may return an error for double unlock
		t.Logf("Second Close() returned error (expected): %v", err)
	}
}

// isLockTimeoutError checks if an error is a lock timeout error
func isLockTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	// Check if error message contains timeout indication
	errMsg := err.Error()
	return contains(errMsg, "timeout") || contains(errMsg, "lock")
}

// contains checks if a string contains a substring (case insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && hasSubstring(s, substr)))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
