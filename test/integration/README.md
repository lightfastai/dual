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
  - Project-local registry support

- **full_workflow_test.go**: Tests for the complete dual workflow
  - `TestFullWorkflow`: End-to-end test of init → service add → context create → context query
  - `TestFullWorkflowWithEnvFile`: Workflow with env file configuration
  - `TestInitForceFlag`: Testing the --force flag for dual init
  - `TestContextAutoDetection`: Automatic context name detection from git branches

- **worktree_test.go**: Tests for git worktree scenarios
  - `TestMultiWorktreeSetup`: Full worktree workflow with multiple branches
  - `TestWorktreeContextIsolation`: Context isolation across different worktrees
  - `TestWorktreeWithDualContextFile`: Using .dual-context file override in worktrees
  - `TestWorktreeServiceDetection`: Service detection in nested worktree directories

- **lifecycle_hooks_test.go**: Tests for environment remapping and lifecycle features
  - `TestEnvRemappingWithDualCreate`: Full worktree creation with env file generation
  - `TestEnvRemappingRegeneration`: Env file regeneration on dual env set/unset
  - `TestEnvRemapCommand`: Manual regeneration with dual env remap
  - `TestEnvRemappingCleanup`: Cleanup of .dual/.local/ on worktree deletion
  - `TestEnvRemappingWithHooks`: Hooks working alongside built-in remapping
  - `TestEnvRemappingEmptyOverrides`: Sparse env file creation (no overrides = no files)
  - `TestEnvRemappingServiceSpecificOnly`: Service-specific overrides without globals
  - `TestEnvRemappingQuotedValues`: Special character handling in env values
  - `TestEnvRemappingWithPORT`: PORT treated as a normal environment variable

- **context_crud_test.go**: Context lifecycle management tests
  - `TestContextList`: Listing contexts with current context highlighting
  - `TestContextListJSON`: JSON output format for context listing
  - `TestContextListNoContexts`: Empty state handling
  - `TestContextDelete`: Deleting contexts with --force flag
  - `TestContextDeleteCurrent`: Prevention of current context deletion
  - `TestContextDeleteNonExistent`: Error handling for missing contexts
  - `TestContextListAll`: Listing contexts across all projects
  - `TestContextDeleteShowsInfo`: Context information display before deletion
  - `TestContextListSorting`: Alphabetical ordering of contexts

- **service_crud_test.go**: Service management tests
  - Service addition, removal, and listing
  - Service path validation
  - Env file configuration
  - Service name validation

- **doctor_test.go**: Health check and diagnostics tests
  - `TestDoctorCommand`: Complete health check in initialized project
  - `TestDoctorWithJSON`: JSON output format for diagnostics
  - `TestDoctorWithoutConfig`: Error detection for missing config
  - `TestDoctorWithInvalidServicePaths`: Path validation errors
  - `TestDoctorWithFix`: Orphaned context cleanup with --fix
  - `TestDoctorWithVerbose`: Detailed diagnostic output
  - `TestDoctorExitCodes`: Exit code behavior (0=pass, 1=warnings, 2=errors)
  - `TestDoctorWorktreeValidation`: Worktree health checks
  - `TestDoctorEnvironmentFiles`: Env file validation
  - `TestDoctorServiceDetection`: Service detection verification

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
- ✓ Complete init → service add → context create → context query workflow
- ✓ Context lifecycle management (create, list, delete)
- ✓ Context information display
- ✓ Worktree lifecycle management with dual create/delete

### Worktree Management
- ✓ Creating worktrees with dual create
- ✓ Deleting worktrees with dual delete
- ✓ Worktree naming patterns
- ✓ Registry integration with worktree operations
- ✓ Hook execution during worktree lifecycle

### Environment Remapping
- ✓ Sparse .env file generation (.dual/.local/service/<service>/.env)
- ✓ Global environment overrides (all services)
- ✓ Service-specific environment overrides
- ✓ Automatic regeneration on dual env set/unset
- ✓ Manual regeneration with dual env remap
- ✓ Cleanup on worktree deletion
- ✓ Quoted value handling for special characters
- ✓ PORT as normal environment variable
- ✓ Header comments in generated files

### Git Integration
- ✓ Git branch-based context detection
- ✓ Multiple git worktrees with isolated contexts
- ✓ Manual context override with .dual-context file
- ✓ Service detection across worktrees
- ✓ Project-local registry shared across worktrees

### Service Management
- ✓ Service addition, removal, and listing
- ✓ Service path validation
- ✓ Env file configuration
- ✓ Service name validation (special characters)
- ✓ Longest-match algorithm for nested services
- ✓ Symbolic link handling

### Configuration Validation
- ✓ Version validation
- ✓ Path validation (relative vs absolute)
- ✓ Service existence validation
- ✓ Duplicate service detection
- ✓ Env file path validation
- ✓ YAML parsing error handling
- ✓ Config file search in parent directories
- ✓ Worktree configuration (path, naming)
- ✓ Hook configuration validation

### Health Checks (dual doctor)
- ✓ Git repository validation
- ✓ Configuration file validation
- ✓ Registry validation
- ✓ Service path checks
- ✓ Environment file checks
- ✓ Worktree validation
- ✓ Orphaned context detection and cleanup
- ✓ JSON output format
- ✓ Verbose output mode
- ✓ Exit code semantics (0=pass, 1=warnings, 2=errors)

### Error Handling
- ✓ Missing config file
- ✓ Unregistered context
- ✓ Non-existent service
- ✓ Service detection failure
- ✓ Duplicate context prevention
- ✓ Current context deletion prevention
- ✓ Orphaned worktree detection

## Test Utilities

### TestHelper Methods

**Binary Management:**
- `NewTestHelper(t)`: Creates isolated test environment with temp directories
- `RunDual(args...)`: Executes dual command in project directory
- `RunDualInDir(dir, args...)`: Executes dual command in specific directory
- `SetTestHome()`: Sets HOME to test-specific directory for isolation
- `RestoreHome()`: Restores original HOME environment variable

**Git Operations:**
- `InitGitRepo()`: Initializes a git repository
- `CreateGitBranch(name)`: Creates and checks out a git branch
- `CreateGitWorktree(branch, path)`: Creates a git worktree at path
- `RunGitCommand(args...)`: Executes git commands in project directory

**File System:**
- `WriteFile(path, content)`: Creates a file with content (relative to project)
- `ReadFile(path)`: Reads file content (relative to project)
- `ReadFileInDir(dir, path)`: Reads file from specific directory
- `FileExists(path)`: Checks if file exists (relative to project)
- `FileExistsInDir(dir, path)`: Checks if file exists in specific directory
- `CreateDirectory(path)`: Creates a directory (relative to project)

**Assertions:**
- `AssertExitCode(got, want, output)`: Verifies exit code
- `AssertOutputContains(output, expected)`: Checks for substring in output
- `AssertOutputNotContains(output, unexpected)`: Checks substring is absent
- `AssertFileContains(path, expected)`: Verifies file content
- `AssertFileExists(path)`: Verifies file exists
- `AssertFileNotExists(path)`: Verifies file does not exist

**Registry:**
- `ReadRegistryJSON()`: Reads project-local registry content
- `RegistryExists()`: Checks if project-local registry file exists

## CI Integration

Integration tests run automatically on every push and pull request via GitHub Actions:

- **Workflow**: `.github/workflows/test.yml`
- **Timeout**: 10 minutes
- **Environment**: Ubuntu latest with Go 1.25.2
- **Coverage**: Uploaded to Codecov with `integration` flag

## Test Isolation

Each test is completely isolated:

1. **Temporary Directories**: Each test gets its own temp directory via `t.TempDir()`
2. **Registry Isolation**: Project-local registry at `$PROJECT_ROOT/.dual/registry.json` (not global)
3. **HOME Isolation**: HOME environment variable set to test-specific directory
4. **Git Repositories**: Fresh git repos created for each test
5. **Binary Building**: Dual binary built once per test (cached by Go)
6. **Cleanup**: Automatic cleanup via Go's testing framework

### Project-Local Registry

Tests in v0.3.0 use project-local registries:
- Registry location: `$PROJECT_ROOT/.dual/registry.json`
- Each test project has its own isolated registry
- Worktrees share their parent repo's registry via path normalization
- No global state across tests

## Test Execution Time

Approximate execution times:
- Full test suite: ~35-40 seconds
- Individual test: ~0.7-1.0 seconds
- Large service tests (50+ services): ~1.0-1.2 seconds

## Known Test Behaviors

### Worktree Requirements
Git worktree tests require service directories to be committed before creating worktrees, so tests include `.gitkeep` files and commit them.

### Context Detection Priority
1. Git branch name (if in git repository)
2. `.dual-context` file content
3. "default" (fallback)

### Environment File Generation
- `.dual/.local/service/<service>/.env` files are only created when overrides exist (sparse pattern)
- Files are automatically regenerated on `dual env set` and `dual env unset`
- Files include warning headers about being auto-generated
- Service-specific overrides take precedence over global overrides

## Contributing

When adding new integration tests:

1. **Use TestHelper**: Use `TestHelper` for all test utilities to ensure consistency
2. **Ensure Isolation**: No shared state between tests - each test gets its own temp directory and registry
3. **Naming Convention**: Follow the pattern `Test<Feature><Scenario>` (e.g., `TestEnvRemappingWithHooks`)
4. **Test Both Paths**: Include both success and failure cases
5. **Document Setup**: Document any special setup requirements (e.g., git commits for worktrees)
6. **Keep Tests Fast**: Aim for < 2 seconds per test when possible
7. **Use RestoreHome**: Call `defer h.RestoreHome()` to restore HOME after tests
8. **Test Project-Local Registry**: Verify registry operations use `$PROJECT_ROOT/.dual/registry.json`
9. **Test Worktree Isolation**: When testing worktrees, verify context sharing via parent repo registry
10. **Hook Testing**: When testing hooks, ensure scripts are made executable with `os.Chmod(path, 0o755)`

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
- [ ] Hook failure handling and rollback scenarios
- [ ] Hook output parsing for environment variables
- [ ] Multiple hooks executing in sequence
- [ ] Concurrent dual command execution with file locking
- [ ] Very large projects (100+ services)
- [ ] Complex worktree configurations with custom naming patterns
- [ ] Environment variable precedence edge cases
- [ ] Performance benchmarks for large registries
- [ ] Upgrade/migration scenarios from v0.2.x to v0.3.0
- [ ] Symlink handling in worktree paths
- [ ] Registry corruption recovery
- [ ] Hook script error messages and debugging
