# Integration Tests for Dual CLI

This directory contains comprehensive integration tests for the `dual` CLI tool. These tests verify end-to-end functionality by running the actual `dual` binary against real file systems and git repositories.

## Test Structure

### Test Files

- **helpers_test.go**: Common test utilities and the `TestHelper` struct that provides:
  - Temporary directory management
  - Dual binary building and execution
  - Git repository initialization and manipulation
  - File system operations
  - Assertion helpers

- **full_workflow_test.go**: Tests for the complete dual workflow
  - `TestFullWorkflow`: End-to-end test of init → service add → context create → port query → command wrapper
  - `TestFullWorkflowWithEnvFile`: Workflow with env file configuration
  - `TestInitForceFlag`: Testing the --force flag for dual init
  - `TestContextAutoDetection`: Automatic context name detection from git branches
  - `TestContextAutoPortAssignment`: Automatic base port assignment

- **worktree_test.go**: Tests for git worktree scenarios
  - `TestMultiWorktreeSetup`: Full worktree workflow with multiple branches
  - `TestWorktreeContextIsolation`: Port isolation across different worktrees
  - `TestWorktreeWithDualContextFile`: Using .dual-context file override in worktrees
  - `TestWorktreeServiceDetection`: Service detection in nested worktree directories

- **service_detection_test.go**: Service auto-detection tests
  - `TestServiceAutoDetection`: Detection from various directory levels
  - `TestServiceDetectionLongestMatch`: Longest-match algorithm for nested services
  - `TestServiceDetectionWithSymlinks`: Detection with symbolic links
  - `TestServiceDetectionMultipleServices`: Detection with many services (7+ services)
  - `TestServiceDetectionErrorMessages`: Error message quality
  - `TestServiceDetectionWithCommandWrapper`: Detection in command wrapper mode

- **port_assignment_test.go**: Port calculation and conflict tests
  - `TestPortConflictHandling`: Deterministic port assignment without conflicts
  - `TestPortCalculationDeterminism`: Alphabetical service ordering
  - `TestAutoPortAssignment`: Automatic port assignment with 100-port increments
  - `TestPortAssignmentWithGaps`: Gap-filling in port assignments
  - `TestPortBoundaryValidation`: Port range validation (1024-65535)
  - `TestContextDuplicatePrevention`: Preventing duplicate contexts
  - `TestPortCalculationWithManyServices`: Testing with 50+ services
  - `TestPortStability`: Documenting port assignment behavior when services change
  - `TestMultiProjectPortIsolation`: Independent port spaces for different projects

- **config_validation_test.go**: Configuration validation tests
  - `TestConfigValidationInvalidVersion`: Invalid config version handling
  - `TestConfigValidationMissingVersion`: Missing version field
  - `TestConfigValidationAbsolutePath`: Rejection of absolute service paths
  - `TestConfigValidationNonExistentPath`: Non-existent path detection
  - `TestConfigValidationFileNotDirectory`: File vs directory validation
  - `TestConfigValidationEnvFileAbsolutePath`: Env file path validation
  - `TestConfigValidationEnvFileNonExistentDirectory`: Env file directory validation
  - `TestConfigValidationEmptyServiceName`: Empty service name handling
  - `TestConfigValidationDuplicateService`: Duplicate service detection
  - `TestConfigValidationEmptyServices`: Empty services config (valid case)
  - `TestConfigValidationMalformedYAML`: Malformed YAML handling
  - `TestConfigNotFound`: Config file not found error
  - `TestConfigSearchUpDirectory`: Config search in parent directories
  - `TestConfigValidationRelativePathNormalization`: Path normalization
  - `TestConfigValidationServicePathOverlap`: Overlapping service paths
  - `TestContextValidationInvalidNames`: Context name validation
  - `TestContextNotRegistered`: Unregistered context error handling
  - `TestServiceNotInConfig`: Non-existent service error handling
  - `TestConfigWithSpecialCharacters`: Service names with hyphens/underscores

## Running Tests

### Run All Integration Tests

```bash
go test -v -timeout=10m ./test/integration/...
```

### Run Specific Test File

```bash
go test -v -timeout=5m ./test/integration/... -run TestFullWorkflow
```

### Run Tests with Coverage

```bash
go test -v -timeout=10m -coverprofile=coverage-integration.out ./test/integration/...
go tool cover -html=coverage-integration.out
```

## Test Coverage

The integration tests cover:

### Core Workflows
- ✓ Complete init → service add → context create → run workflow
- ✓ Command wrapper with PORT environment variable injection
- ✓ Port queries (individual and all services)
- ✓ Context information display

### Git Integration
- ✓ Git branch-based context detection
- ✓ Multiple git worktrees with isolated contexts
- ✓ Manual context override with .dual-context file
- ✓ Service detection across worktrees

### Service Management
- ✓ Auto-detection from current directory
- ✓ Longest-match algorithm for nested services
- ✓ Service path validation
- ✓ Symbolic link handling
- ✓ Multiple services (tested with 50+ services)
- ✓ --service flag override

### Port Assignment
- ✓ Deterministic port calculation (basePort + serviceIndex + 1)
- ✓ Alphabetical service ordering
- ✓ Auto-assignment with 100-port increments
- ✓ Gap-filling in port ranges
- ✓ Port boundary validation (1024-65535)
- ✓ Multi-project port isolation

### Configuration Validation
- ✓ Version validation
- ✓ Path validation (relative vs absolute)
- ✓ Service existence validation
- ✓ Duplicate service detection
- ✓ Env file path validation
- ✓ YAML parsing error handling
- ✓ Config file search in parent directories

### Error Handling
- ✓ Missing config file
- ✓ Unregistered context
- ✓ Non-existent service
- ✓ Service detection failure
- ✓ Invalid port ranges
- ✓ Duplicate context prevention

## Test Utilities

### TestHelper Methods

**Binary Management:**
- `NewTestHelper(t)`: Creates isolated test environment with temp directories
- `RunDual(args...)`: Executes dual command in project directory
- `RunDualInDir(dir, args...)`: Executes dual command in specific directory

**Git Operations:**
- `InitGitRepo()`: Initializes a git repository
- `CreateGitBranch(name)`: Creates and checks out a git branch
- `CreateGitWorktree(branch, path)`: Creates a git worktree
- `RunGitCommand(args...)`: Executes git commands

**File System:**
- `WriteFile(path, content)`: Creates a file with content
- `ReadFile(path)`: Reads file content
- `FileExists(path)`: Checks if file exists
- `CreateDirectory(path)`: Creates a directory

**Assertions:**
- `AssertExitCode(got, want, output)`: Verifies exit code
- `AssertOutputContains(output, expected)`: Checks for substring in output
- `AssertOutputNotContains(output, unexpected)`: Checks substring is absent
- `AssertFileContains(path, expected)`: Verifies file content

**Registry:**
- `ReadRegistryJSON()`: Reads registry content
- `RegistryExists()`: Checks if registry file exists

## CI Integration

Integration tests run automatically on every push and pull request via GitHub Actions:

- **Workflow**: `.github/workflows/test.yml`
- **Timeout**: 10 minutes
- **Environment**: Ubuntu latest with Go 1.25.2
- **Coverage**: Uploaded to Codecov with `integration` flag

## Test Isolation

Each test is completely isolated:

1. **Temporary Directories**: Each test gets its own temp directory via `t.TempDir()`
2. **Registry Isolation**: HOME environment variable set to test-specific directory
3. **Git Repositories**: Fresh git repos created for each test
4. **Binary Building**: Dual binary built once per test (cached by Go)
5. **Cleanup**: Automatic cleanup via Go's testing framework

## Test Execution Time

Approximate execution times:
- Full test suite: ~35-40 seconds
- Individual test: ~0.7-1.0 seconds
- Large service tests (50+ services): ~1.0-1.2 seconds

## Known Test Behaviors

### Port Assignment Order
Services are always ordered alphabetically for deterministic port calculation:
- `admin`, `api`, `auth`, `web`, `worker` → ports 4101, 4102, 4103, 4104, 4105

### Worktree Requirements
Git worktree tests require service directories to be committed before creating worktrees, so tests include `.gitkeep` files and commit them.

### Context Detection Priority
1. Git branch name (if in git repository)
2. `.dual-context` file content
3. "default" (fallback)

## Contributing

When adding new integration tests:

1. Use the `TestHelper` for consistency
2. Ensure test isolation (no shared state)
3. Add descriptive test names following the pattern `Test<Feature><Scenario>`
4. Include both success and failure cases
5. Document any special setup requirements
6. Keep tests fast (< 2 seconds per test when possible)
7. Clean up resources (though temp dirs auto-cleanup)

## Debugging Tests

To debug a failing test:

```bash
# Run with verbose output
go test -v ./test/integration/... -run TestSpecificTest

# Add t.Log() statements in test code
t.Logf("Debug info: %v", value)

# Check temp directory contents (add this before test completes)
t.Logf("Temp dir: %s", h.TempDir)
time.Sleep(60 * time.Second) // Pause to inspect
```

## Future Test Coverage

Potential areas for additional tests:
- [ ] `dual open` command
- [ ] `dual sync` command with env file writing
- [ ] Concurrent dual command execution
- [ ] Very large projects (100+ services)
- [ ] Network port availability checking
- [ ] Shell integration tests
- [ ] Performance benchmarks
- [ ] Upgrade/migration scenarios
