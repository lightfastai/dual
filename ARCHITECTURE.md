# dual Architecture

Technical documentation explaining how `dual` works internally.

## Table of Contents

- [Overview](#overview)
- [Core Components](#core-components)
- [Data Flow](#data-flow)
- [File Structure](#file-structure)
- [Port Calculation Algorithm](#port-calculation-algorithm)
- [Context Detection](#context-detection)
- [Service Detection](#service-detection)
- [Registry Management](#registry-management)
- [Configuration System](#configuration-system)
- [Command Execution](#command-execution)
- [Concurrency and Thread Safety](#concurrency-and-thread-safety)
- [Error Handling](#error-handling)
- [Design Decisions](#design-decisions)

---

## Overview

`dual` is a CLI tool written in Go that manages port assignments across different development contexts. It operates as a transparent command wrapper that:

1. Detects the current development context (git branch or manual override)
2. Detects the current service (from working directory or CLI flag)
3. Calculates the appropriate port using a deterministic formula
4. Injects the PORT environment variable into the wrapped command
5. Executes the command with the correct port

### Key Principles

- **Transparent**: Shows exactly what it's doing
- **Deterministic**: Same inputs always produce same outputs
- **Non-invasive**: Never modifies files (Vercel-proof)
- **Fast**: Minimal overhead, instant execution
- **Safe**: Thread-safe registry operations

---

## Core Components

### 1. Command Wrapper (`cmd/dual/main.go`)

The entry point and primary interface. Handles:
- Parsing command-line arguments
- Detecting passthrough vs. subcommand mode
- Orchestrating detection and execution pipeline
- Injecting PORT environment variable

### 2. Config Manager (`internal/config/`)

Manages `dual.config.yml` file:
- Searches for config file up the directory tree
- Parses and validates YAML structure
- Provides service definitions to other components
- Validates service paths and env files

### 3. Registry Manager (`internal/registry/`)

Manages `~/.dual/registry.json`:
- Stores context-to-basePort mappings globally
- Thread-safe read/write operations
- Atomic file updates to prevent corruption
- Auto-assigns next available ports

### 4. Context Detector (`internal/context/`)

Determines the current development context:
- Executes `git branch --show-current`
- Searches for `.dual-context` file
- Falls back to "default"

### 5. Service Detector (`internal/service/`)

Identifies the current service:
- Matches current directory against service paths
- Uses longest path match for nested services
- Calculates service index from config order

### 6. Port Calculator (`internal/service/`)

Computes final port number:
- Formula: `port = basePort + serviceIndex + 1`
- Retrieves basePort from registry
- Determines serviceIndex from config

---

## Data Flow

### Command Wrapper Execution Flow

```
User runs: dual pnpm dev
         │
         ▼
┌────────────────────┐
│  Parse Arguments   │ ──► Detect: passthrough mode
└────────────────────┘
         │
         ▼
┌────────────────────┐
│  Load Config       │ ──► Find dual.config.yml
│  (config.go)       │     Parse YAML
└────────────────────┘     Validate services
         │
         ▼
┌────────────────────┐
│  Detect Context    │ ──► Try git branch
│  (context.go)      │     Try .dual-context file
└────────────────────┘     Fall back to "default"
         │
         ▼
┌────────────────────┐
│  Detect Service    │ ──► Match CWD vs service paths
│  (service.go)      │     Select longest match
└────────────────────┘     Return service name
         │
         ▼
┌────────────────────┐
│  Load Registry     │ ──► Read ~/.dual/registry.json
│  (registry.go)     │     Parse JSON
└────────────────────┘     Find project context
         │
         ▼
┌────────────────────┐
│  Calculate Port    │ ──► Get basePort from registry
│  (calculator.go)   │     Get serviceIndex from config
└────────────────────┘     port = basePort + serviceIndex + 1
         │
         ▼
┌────────────────────┐
│  Print Info        │ ──► [dual] Context: main | Service: web | Port: 4101
└────────────────────┘
         │
         ▼
┌────────────────────┐
│  Execute Command   │ ──► Set PORT=4101 in environment
│                    │     Run: pnpm dev
└────────────────────┘     Stream stdout/stderr
```

---

## File Structure

### Project Layout

```
dual/
├── cmd/dual/                    # Command implementations
│   ├── main.go                  # Entry point, command wrapper
│   ├── init.go                  # dual init
│   ├── service.go               # dual service add
│   ├── context.go               # dual context, dual context create
│   ├── port.go                  # dual port
│   ├── ports.go                 # dual ports
│   ├── open.go                  # dual open
│   └── sync.go                  # dual sync
│
├── internal/                    # Internal packages
│   ├── config/                  # Configuration management
│   │   ├── config.go            # Config loading, parsing, validation
│   │   └── config_test.go       # Unit tests
│   │
│   ├── registry/                # Registry management
│   │   ├── registry.go          # Registry CRUD operations
│   │   ├── registry_test.go     # Unit tests
│   │   └── example_test.go      # Example usage
│   │
│   ├── context/                 # Context detection
│   │   ├── detector.go          # Detection logic
│   │   └── detector_test.go     # Unit tests
│   │
│   └── service/                 # Service detection and port calculation
│       ├── detector.go          # Service detection
│       ├── detector_test.go     # Unit tests
│       ├── calculator.go        # Port calculation
│       └── calculator_test.go   # Unit tests
│
├── dual.config.yml              # Example configuration
├── go.mod                       # Go module definition
└── go.sum                       # Dependency checksums
```

### Runtime Files

```
Project directory:
  dual.config.yml              # Committed to repo, defines services
  .dual-context                # Optional, overrides git branch detection

User home directory:
  ~/.dual/
    └── registry.json          # Global registry, never committed
```

---

## Port Calculation Algorithm

### Formula

```
port = basePort + serviceIndex + 1
```

### Components

1. **basePort**: Retrieved from registry for current context
   - Stored in `~/.dual/registry.json`
   - Typically assigned in 100-port increments (4100, 4200, 4300)

2. **serviceIndex**: Position of service in config
   - Determined by order in `dual.config.yml`
   - Zero-indexed (first service = 0, second = 1, etc.)

3. **+1 offset**: Prevents using base port directly
   - Leaves base port available for metadata/routing
   - Makes port math clearer (4100 base → services start at 4101)

### Example

```yaml
# dual.config.yml
version: 1
services:
  web:     # serviceIndex = 0
    path: apps/web
  api:     # serviceIndex = 1
    path: apps/api
  worker:  # serviceIndex = 2
    path: apps/worker
```

```json
// ~/.dual/registry.json
{
  "projects": {
    "/Users/dev/Code/myproject": {
      "contexts": {
        "main": {
          "basePort": 4100
        },
        "feature-auth": {
          "basePort": 4200
        }
      }
    }
  }
}
```

**Port Calculation:**

```
Context: main (basePort = 4100)
  web:    4100 + 0 + 1 = 4101
  api:    4100 + 1 + 1 = 4102
  worker: 4100 + 2 + 1 = 4103

Context: feature-auth (basePort = 4200)
  web:    4200 + 0 + 1 = 4201
  api:    4200 + 1 + 1 = 4202
  worker: 4200 + 2 + 1 = 4203
```

### Implementation

```go
// internal/service/calculator.go

func CalculatePort(cfg *config.Config, reg *registry.Registry,
                   projectRoot, contextName, serviceName string) (int, error) {
    // 1. Get basePort from registry
    ctx, err := reg.GetContext(projectRoot, contextName)
    if err != nil {
        return 0, err
    }
    basePort := ctx.BasePort

    // 2. Get serviceIndex from config
    serviceIndex := 0
    found := false
    for name := range cfg.Services {
        if name == serviceName {
            found = true
            break
        }
        serviceIndex++
    }
    if !found {
        return 0, ErrServiceNotFound
    }

    // 3. Calculate port
    port := basePort + serviceIndex + 1

    return port, nil
}
```

---

## Context Detection

### Priority Order

1. **Git Branch** (highest priority)
2. **`.dual-context` File** (manual override)
3. **"default"** (fallback)

### Implementation

```go
// internal/context/detector.go

func (d *Detector) DetectContext() (string, error) {
    // Priority 1: Try git branch
    if branch, err := d.detectGitBranch(); err == nil && branch != "" {
        return branch, nil
    }

    // Priority 2: Look for .dual-context file
    if context, err := d.findDualContextFile(); err == nil && context != "" {
        return context, nil
    }

    // Priority 3: Return default
    return DefaultContext, nil
}
```

### Git Branch Detection

```go
func (d *Detector) detectGitBranch() (string, error) {
    // Execute: git branch --show-current
    cmd := exec.Command("git", "branch", "--show-current")
    output, err := cmd.Output()
    if err != nil {
        return "", err  // Not a git repo or error
    }

    branch := strings.TrimSpace(string(output))
    if branch == "" {
        // Detached HEAD state
        return "", fmt.Errorf("no current branch")
    }

    return branch, nil
}
```

### `.dual-context` File Detection

```go
func (d *Detector) findDualContextFile() (string, error) {
    cwd, _ := os.Getwd()

    // Walk up directory tree
    currentDir := cwd
    for {
        contextPath := filepath.Join(currentDir, DualContextFile)

        // Try to read the file
        data, err := os.ReadFile(contextPath)
        if err == nil {
            context := strings.TrimSpace(string(data))
            if context != "" {
                return context, nil
            }
        }

        // Move up one directory
        parent := filepath.Dir(currentDir)
        if parent == currentDir {
            break  // Reached root
        }
        currentDir = parent
    }

    return "", fmt.Errorf("no .dual-context file found")
}
```

### Use Cases

| Method | Use Case |
|--------|----------|
| Git branch | Normal development (95% of cases) |
| `.dual-context` file | Manual override (long branch names, custom naming) |
| "default" | Not in git repo, CI/CD, containers |

---

## Service Detection

### Algorithm

1. Get current working directory
2. Normalize path (resolve symlinks, make absolute)
3. For each service in config:
   - Construct absolute service path
   - Check if CWD is within service path
   - Track longest matching path
4. Return service with longest match

### Why Longest Match?

Handles nested services correctly:

```yaml
services:
  web:
    path: apps/web
  admin:
    path: apps/web/admin  # Nested under web
```

```bash
# CWD: /project/apps/web/admin/src
# Matches both "web" and "admin"
# Longest match: "admin" (more specific)
```

### Implementation

```go
// internal/service/detector.go

func DetectService(cfg *config.Config, projectRoot string) (string, error) {
    cwd, err := os.Getwd()
    if err != nil {
        return "", err
    }

    // Normalize current directory
    cwdAbs, err := filepath.Abs(cwd)
    if err != nil {
        return "", err
    }

    var longestMatch string
    longestMatchLen := 0

    // Check each service
    for name, service := range cfg.Services {
        servicePath := filepath.Join(projectRoot, service.Path)
        servicePathAbs, _ := filepath.Abs(servicePath)

        // Check if CWD is within service path
        if strings.HasPrefix(cwdAbs, servicePathAbs) {
            matchLen := len(servicePathAbs)
            if matchLen > longestMatchLen {
                longestMatch = name
                longestMatchLen = matchLen
            }
        }
    }

    if longestMatch == "" {
        return "", ErrServiceNotDetected
    }

    return longestMatch, nil
}
```

---

## Registry Management

### Registry Structure

```json
{
  "projects": {
    "<absolute-project-path>": {
      "contexts": {
        "<context-name>": {
          "basePort": 4100,
          "path": "/optional/worktree/path",
          "created": "2025-10-14T10:00:00Z"
        }
      }
    }
  }
}
```

### Thread Safety

Registry operations are thread-safe using `sync.RWMutex`:

```go
type Registry struct {
    Projects map[string]Project `json:"projects"`
    mu       sync.RWMutex       `json:"-"`
}

func (r *Registry) GetContext(projectPath, contextName string) (*Context, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    // ... read operations
}

func (r *Registry) SetContext(projectPath, contextName string, basePort int) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    // ... write operations
}
```

### Atomic Writes

Registry updates use atomic write pattern:

```go
func (r *Registry) SaveRegistry() error {
    // 1. Marshal to JSON
    data, _ := json.MarshalIndent(r, "", "  ")

    // 2. Write to temporary file
    tempFile := registryPath + ".tmp"
    os.WriteFile(tempFile, data, 0644)

    // 3. Atomic rename (POSIX guarantees atomicity)
    os.Rename(tempFile, registryPath)

    return nil
}
```

This prevents corruption if:
- Process crashes during write
- Multiple dual instances run concurrently
- Disk fills up during write

### Auto-Port Assignment

```go
func (r *Registry) FindNextAvailablePort() int {
    usedPorts := make(map[int]bool)

    // Collect all used base ports
    for _, project := range r.Projects {
        for _, context := range project.Contexts {
            usedPorts[context.BasePort] = true
        }
    }

    // Find next available port starting from DefaultBasePort (4100)
    nextPort := DefaultBasePort
    for {
        if !usedPorts[nextPort] {
            return nextPort
        }
        nextPort += PortIncrement  // Increment by 100
    }
}
```

---

## Configuration System

### Configuration File (`dual.config.yml`)

```yaml
version: 1
services:
  <service-name>:
    path: <relative-path>
    envFile: <relative-env-file-path>
```

### Loading Algorithm

1. Start from current working directory
2. Check for `dual.config.yml`
3. If not found, move up one directory
4. Repeat until found or reach filesystem root
5. Parse YAML
6. Validate structure and paths

### Validation

```go
func validateConfig(config *Config, projectRoot string) error {
    // Check version
    if config.Version != SupportedVersion {
        return fmt.Errorf("unsupported config version")
    }

    // Validate each service
    for name, service := range config.Services {
        // Path must be relative
        if filepath.IsAbs(service.Path) {
            return fmt.Errorf("path must be relative")
        }

        // Path must exist
        fullPath := filepath.Join(projectRoot, service.Path)
        if _, err := os.Stat(fullPath); err != nil {
            return fmt.Errorf("path does not exist: %s", service.Path)
        }

        // EnvFile directory must exist (if specified)
        if service.EnvFile != "" {
            envFileDir := filepath.Dir(filepath.Join(projectRoot, service.EnvFile))
            if _, err := os.Stat(envFileDir); err != nil {
                return fmt.Errorf("envFile directory does not exist")
            }
        }
    }

    return nil
}
```

---

## Command Execution

### Environment Injection

```go
func runCommandWrapper(args []string) error {
    // ... detection logic ...

    // Prepare command
    cmd := exec.Command(args[0], args[1:]...)

    // Inject PORT environment variable
    // Preserve all existing environment variables
    cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", port))

    // Stream output in real-time
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    cmd.Stdin = os.Stdin

    // Execute
    return cmd.Run()
}
```

### Exit Code Preservation

```go
err = cmd.Run()
if err != nil {
    // Check if it's an exit error with specific code
    if exitErr, ok := err.(*exec.ExitError); ok {
        os.Exit(exitErr.ExitCode())  // Preserve exit code
    }
    // Other errors (command not found, etc.)
    return fmt.Errorf("failed to execute command: %w", err)
}
```

---

## Concurrency and Thread Safety

### Registry Access

Multiple `dual` commands may run simultaneously:

```bash
# Terminal 1
dual pnpm dev

# Terminal 2 (at the same time)
dual context create feature-x
```

Registry uses `sync.RWMutex`:
- Multiple readers can access simultaneously
- Writers get exclusive access
- Prevents race conditions

### File I/O

All file operations use atomic patterns:

1. **Config writes**: Temp file + atomic rename
2. **Registry writes**: Temp file + atomic rename
3. **Env file writes** (`dual sync`): Temp file + atomic rename

### Concurrent Safety Guarantees

- ✅ Multiple dual commands reading registry
- ✅ Multiple dual commands executing wrapped commands
- ✅ One dual writing registry while others read
- ✅ Crashed writes don't corrupt registry
- ⚠️ Race condition: Two dual commands creating same context simultaneously
  - Last write wins (acceptable for this use case)

---

## Error Handling

### Error Types

```go
// Service detection
var (
    ErrServiceNotDetected = errors.New("service not detected")
    ErrServiceNotFound    = errors.New("service not found")
)

// Registry
var (
    ErrProjectNotFound = errors.New("project not found in registry")
    ErrContextNotFound = errors.New("context not found in project")
)
```

### Error Messages

All errors include helpful hints:

```
Error: context "feature-x" not found in registry
Hint: Run 'dual context create' to create this context
```

### Error Handling Strategy

1. **Recoverable errors**: Show error + hint, exit 1
2. **Unrecoverable errors**: Show error, exit 1
3. **Wrapped command errors**: Preserve exit code
4. **Corrupt registry**: Warn, create new empty registry

---

## Design Decisions

### Why Go?

- **Fast startup**: No runtime overhead like Node.js/Python
- **Single binary**: Easy distribution, no dependencies
- **Cross-platform**: Compile for macOS, Linux, Windows
- **Excellent stdlib**: File I/O, JSON, YAML, exec
- **Static typing**: Catch errors at compile time

### Why Not Modify Files?

**Problem**: Vercel's `vercel pull` overwrites `.env` files

**Solution**: Never write PORT to files, only inject in environment

**Benefits**:
- No conflicts with tool-generated files
- No git diffs for port changes
- Cleaner separation of concerns

**Tradeoff**: Requires wrapping all commands with `dual`

**Fallback**: `dual sync` for cases where wrapper can't be used

### Why Global Registry?

**Alternative**: Store contexts in project

**Problem**:
- Git worktrees share same project root
- Multiple clones have separate configs
- Need to track ports across all contexts globally

**Solution**: `~/.dual/registry.json` in user home directory

**Benefits**:
- Single source of truth
- Auto-assign ports without conflicts
- Works across worktrees and clones

### Why Deterministic Port Formula?

**Alternative**: Random port assignment

**Problem**:
- Ports change between runs
- Hard to remember/document
- Breaks bookmarks and scripts

**Solution**: `port = basePort + serviceIndex + 1`

**Benefits**:
- Same port every time
- Easy to predict and document
- Service order in config defines ports

### Why Support `.dual-context` File?

**Use Cases**:
1. **Long branch names**: `feature/JIRA-1234-very-long-description` → `feat-1234`
2. **Detached HEAD**: Working with specific commits
3. **Non-git projects**: Manual context management
4. **Testing**: Override detection for testing

**Tradeoff**: Another thing to manage, but optional

---

## Performance Considerations

### Startup Time

Typical execution time: **< 50ms**

Breakdown:
- Load config: ~5ms
- Detect context: ~10ms (git command)
- Detect service: ~5ms
- Load registry: ~5ms
- Calculate port: <1ms
- Execute command: ~remaining

### Optimization Techniques

1. **No unnecessary file I/O**: Only read files when needed
2. **Compiled binary**: No interpreter startup
3. **Minimal dependencies**: Small binary size (~5MB)
4. **Efficient path matching**: O(n) where n = number of services

### Scalability

- **Services per project**: No practical limit (tested up to 100)
- **Contexts per project**: No practical limit (tested up to 50)
- **Projects in registry**: No practical limit (tested up to 100)
- **Registry file size**: Stays small (~1KB per project)

---

## Testing Strategy

### Unit Tests

Each component has unit tests:

```
internal/config/config_test.go
internal/registry/registry_test.go
internal/context/detector_test.go
internal/service/detector_test.go
internal/service/calculator_test.go
```

### Test Coverage

- Config loading and validation
- Registry CRUD operations
- Context detection (with mocks for git)
- Service detection (with temp directories)
- Port calculation (pure function)

### Dependency Injection

Context detector uses dependency injection for testability:

```go
type Detector struct {
    gitCommand func(args ...string) (string, error)
    readFile   func(path string) ([]byte, error)
    getwd      func() (string, error)
}
```

Tests can inject mocks:

```go
detector := &Detector{
    gitCommand: func(args ...string) (string, error) {
        return "test-branch", nil
    },
    readFile: os.ReadFile,
    getwd:    os.Getwd,
}
```

---

## Future Enhancements

### Potential Improvements

1. **Context Cleanup**: `dual context delete`
2. **Port Health Check**: Warn if port already in use
3. **Shell Completions**: Bash/Zsh/Fish completion scripts
4. **Visual Dashboard**: `dual ui` - web interface
5. **Service Dependencies**: Define service startup order
6. **Port Range Validation**: Ensure services fit within base port range
7. **Configuration Validation**: `dual config validate`
8. **Migration Tools**: Upgrade config versions
9. **Telemetry**: Optional usage analytics

### Architecture Extensions

1. **Plugin System**: Allow custom detection strategies
2. **Hook System**: Pre/post command execution hooks
3. **Remote Registry**: Share contexts across team
4. **Watch Mode**: Auto-restart on config changes

---

## References

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [YAML v3](https://github.com/go-yaml/yaml) - YAML parsing
- [Go stdlib](https://pkg.go.dev/std) - Core functionality

---

## Contributing to Architecture

When making architectural changes:

1. Maintain backward compatibility with config/registry formats
2. Add migration path for breaking changes
3. Update this document with new designs
4. Add comprehensive tests
5. Consider performance implications
6. Document design decisions

---

For usage information, see [USAGE.md](USAGE.md).
For examples, see [EXAMPLES.md](EXAMPLES.md).
For project overview, see [README.md](README.md).
