# dual Architecture

Technical documentation explaining how `dual` works internally.

## Table of Contents

- [Overview](#overview)
- [Core Components](#core-components)
- [Data Flow](#data-flow)
- [File Structure](#file-structure)
- [Worktree Management](#worktree-management)
- [Hook System](#hook-system)
- [Context Detection](#context-detection)
- [Service Detection](#service-detection)
- [Registry Management](#registry-management)
- [Configuration System](#configuration-system)
- [Concurrency and Thread Safety](#concurrency-and-thread-safety)
- [Error Handling](#error-handling)
- [Design Decisions](#design-decisions)

---

## Overview

`dual` is a CLI tool written in Go that manages git worktree lifecycle (creation, deletion) with environment remapping via hooks. It operates in **Management Mode** with direct subcommands:

- `dual create <branch>` - Create a new worktree with lifecycle hooks
- `dual delete <context>` - Delete a worktree with cleanup hooks
- `dual init` - Initialize dual configuration
- `dual service add/list/remove` - Manage service definitions
- `dual context list` - List contexts (deprecated in favor of create/delete)
- `dual doctor` - Diagnose configuration and registry health

### Key Principles

- **Hook-Based Customization**: Core tool manages worktree lifecycle; users implement custom logic (ports, databases, env) in hooks
- **Transparent**: Always show user what operations are being performed
- **Fail-safe**: Corrupt registry or missing config produce helpful errors, not crashes
- **Isolated Contexts**: Each worktree is an independent environment
- **Project-Local State**: Registry and hooks are project-specific, not global

---

## Core Components

### 1. Command Entry Point (`cmd/dual/main.go`)

The entry point and command router. Handles:
- Parsing command-line arguments via cobra
- Routing to appropriate subcommands
- Global flags (--verbose, --debug)
- Version information

### 2. Config Manager (`internal/config/`)

Manages `dual.config.yml` file:
- Searches for config file up the directory tree
- Parses and validates YAML structure
- Provides service definitions to other components
- Validates service paths exist
- Supports optional worktrees and hooks configuration
- Thread-safe file I/O with atomic writes

### 3. Registry Manager (`internal/registry/`)

Manages `$PROJECT_ROOT/.dual/.local/registry.json`:
- Project-local state (not global like v0.2.x)
- Structure: projects → contexts (name → path, created timestamp)
- All worktrees of a repository share the parent repo's registry via `GetProjectIdentifier()` normalization
- Thread-safe read/write operations with `sync.RWMutex`
- File locking with flock to prevent concurrent modifications
- Atomic writes via temp file + rename pattern
- Auto-recovers from corruption (returns empty registry)

### 4. Context Detector (`internal/context/`)

Determines the current development context:
- Priority: git branch → `.dual-context` file → "default"
- Executes `git branch --show-current`
- Searches for `.dual-context` file
- Falls back to "default"

### 5. Service Detector (`internal/service/detector.go`)

Identifies the current service:
- Matches current working directory against service paths from config
- Resolves symlinks for consistency across worktrees
- Uses longest path match for nested service structures
- Returns `ErrServiceNotDetected` if no match found

### 6. Hook System (`internal/hooks/`)

Executes lifecycle hooks at key worktree management points:
- Hook events: `postWorktreeCreate`, `preWorktreeDelete`, `postWorktreeDelete`
- Scripts located in `$PROJECT_ROOT/.dual/hooks/` and configured in `dual.config.yml`
- Receives context via environment variables: `DUAL_EVENT`, `DUAL_CONTEXT_NAME`, `DUAL_CONTEXT_PATH`, `DUAL_PROJECT_ROOT`
- Non-zero exit codes fail the operation and halt execution
- Scripts run in sequence (not parallel)
- stdout/stderr are streamed to the user in real-time

### 7. Worktree Manager (`internal/worktree/`)

Handles git worktree operations:
- Creates git worktrees at configured location
- Names directories using configured pattern (defaults to `{branch}`)
- Registers context in project-local registry
- Integrates with hook system for automation
- Validates worktree paths and branch names
- Removes worktrees with cleanup

### 8. Logger Manager (`internal/logger/`)

Provides structured logging with multiple verbosity levels:
- **Verbose mode** (`--verbose`): Shows detailed operational info
- **Debug mode** (`--debug`): Shows internal state and decisions
- **Environment variable**: `DUAL_DEBUG=1` enables debug mode
- All output goes to stderr to keep stdout clean
- Functions: `Info()`, `Success()`, `Error()`, `Verbose()`, `Debug()`

---

## Data Flow

### Worktree Creation Flow

```
User runs: dual create feature-x
         │
         ▼
┌────────────────────┐
│  Parse Arguments   │ ──► Extract branch name: feature-x
└────────────────────┘
         │
         ▼
┌────────────────────┐
│  Load Config       │ ──► Find dual.config.yml
│  (config.go)       │     Parse YAML
└────────────────────┘     Validate worktrees.path
         │
         ▼
┌────────────────────┐
│  Validate Path     │ ──► Check if worktree path exists
│                    │     Ensure not in worktree already
└────────────────────┘
         │
         ▼
┌────────────────────┐
│  Create Worktree   │ ──► git worktree add <path> -b <branch>
│  (worktree.go)     │     Register in registry
└────────────────────┘
         │
         ▼
┌────────────────────┐
│  Execute Hooks     │ ──► Run postWorktreeCreate scripts
│  (hooks.go)        │     Pass context via env vars
└────────────────────┘     Stream output to user
         │
         ▼
┌────────────────────┐
│  Success Message   │ ──► [dual] Created worktree for: feature-x
└────────────────────┘     Path: /path/to/worktrees/feature-x
```

### Worktree Deletion Flow

```
User runs: dual delete feature-x
         │
         ▼
┌────────────────────┐
│  Parse Arguments   │ ──► Extract context name: feature-x
└────────────────────┘
         │
         ▼
┌────────────────────┐
│  Load Registry     │ ──► Read project-local registry
│  (registry.go)     │     Find context entry
└────────────────────┘     Validate it's a worktree
         │
         ▼
┌────────────────────┐
│  Pre-Delete Hooks  │ ──► Run preWorktreeDelete scripts
│  (hooks.go)        │     While files still exist
└────────────────────┘
         │
         ▼
┌────────────────────┐
│  Remove Worktree   │ ──► git worktree remove <path>
│  (worktree.go)     │     Remove from registry
└────────────────────┘
         │
         ▼
┌────────────────────┐
│  Post-Delete Hooks │ ──► Run postWorktreeDelete scripts
│  (hooks.go)        │     After worktree removed
└────────────────────┘
         │
         ▼
┌────────────────────┐
│  Success Message   │ ──► [dual] Deleted worktree: feature-x
└────────────────────┘
```

---

## File Structure

### Project Layout

```
dual/
├── cmd/dual/                    # Command implementations
│   ├── main.go                  # Entry point, command routing
│   ├── init.go                  # dual init
│   ├── service.go               # dual service add/list/remove
│   ├── context.go               # dual context list
│   ├── create.go                # dual create <branch>
│   ├── delete.go                # dual delete <context>
│   ├── doctor.go                # dual doctor
│   └── completion.go            # Shell completions
│
├── internal/                    # Internal packages
│   ├── config/                  # Configuration management
│   │   ├── config.go            # Config loading, parsing, validation
│   │   └── config_test.go       # Unit tests
│   │
│   ├── registry/                # Registry management
│   │   ├── registry.go          # Registry CRUD operations (with file locking)
│   │   ├── registry_test.go     # Unit tests
│   │   └── example_test.go      # Example usage
│   │
│   ├── context/                 # Context detection
│   │   ├── detector.go          # Detection logic
│   │   └── detector_test.go     # Unit tests
│   │
│   ├── service/                 # Service detection
│   │   ├── detector.go          # Service detection
│   │   └── detector_test.go     # Unit tests
│   │
│   ├── hooks/                   # Hook system
│   │   ├── executor.go          # Hook execution logic
│   │   └── executor_test.go     # Unit tests
│   │
│   ├── worktree/                # Worktree operations
│   │   ├── manager.go           # Worktree creation/deletion
│   │   └── manager_test.go      # Unit tests
│   │
│   └── logger/                  # Logging system
│       ├── logger.go            # Structured logging with verbosity levels
│       └── logger_test.go       # Unit tests
│
├── test/integration/            # Integration tests
│   ├── helpers_test.go          # Test utilities
│   ├── worktree_lifecycle_test.go  # Worktree creation/deletion tests
│   └── hooks_test.go            # Hook execution tests
│
├── dual.config.yml              # Example configuration
├── go.mod                       # Go module definition
└── go.sum                       # Dependency checksums
```

### Runtime Files

```
Project directory:
  dual.config.yml              # Committed to repo, defines services/worktrees/hooks
  .dual/
    ├── registry.json          # Project-local state, NOT committed (add to .gitignore)
    ├── registry.json.lock     # Temporary file during operations
    └── hooks/                 # Hook scripts, can be committed
        ├── setup-database.sh
        ├── setup-environment.sh
        └── cleanup-database.sh

Worktree directory:
  .dual-context                # Optional, overrides git branch detection
```

---

## Worktree Management

### Creation (`dual create <branch>`)

**Steps:**

1. **Validation**
   - Check if `worktrees.path` is configured in `dual.config.yml`
   - Validate branch name is valid
   - Check if already in a worktree (only create from main repo)
   - Resolve worktree target path based on naming pattern

2. **Git Worktree Creation**
   ```bash
   git worktree add <target-path> -b <branch>
   ```

3. **Registry Update**
   - Add context entry to project-local registry
   - Store worktree path, creation timestamp
   - Use file locking to prevent concurrent modifications

4. **Hook Execution**
   - Execute `postWorktreeCreate` hooks in sequence
   - Set environment variables: `DUAL_EVENT`, `DUAL_CONTEXT_NAME`, `DUAL_CONTEXT_PATH`, `DUAL_PROJECT_ROOT`
   - Stream stdout/stderr to user
   - Halt on non-zero exit code

**Example:**
```bash
dual create feature-auth
# Creates: ../worktrees/feature-auth
# Registers: feature-auth → /abs/path/to/worktrees/feature-auth
# Executes: .dual/hooks/setup-database.sh, .dual/hooks/setup-environment.sh
```

### Deletion (`dual delete <context>`)

**Steps:**

1. **Validation**
   - Check if context exists in registry
   - Verify it has a worktree path (not main repo)
   - Validate worktree directory exists

2. **Pre-Delete Hooks**
   - Execute `preWorktreeDelete` hooks in sequence
   - Files still exist at this point
   - Typical use: backup data, cleanup databases

3. **Git Worktree Removal**
   ```bash
   git worktree remove <path>
   ```

4. **Registry Update**
   - Remove context from registry
   - Use file locking to prevent concurrent modifications

5. **Post-Delete Hooks**
   - Execute `postWorktreeDelete` hooks in sequence
   - Worktree files are gone at this point
   - Typical use: notify team, cleanup external resources

**Example:**
```bash
dual delete feature-auth
# Executes: .dual/hooks/backup-data.sh (pre-delete)
# Removes: git worktree at /abs/path/to/worktrees/feature-auth
# Unregisters: feature-auth from registry
# Executes: .dual/hooks/notify-team.sh (post-delete)
```

### Worktree Path Resolution

**Naming pattern** (from `worktrees.naming` in config):
- Default: `{branch}` - uses branch name as directory name
- Example: `feature/auth` → `feature-auth` (slashes replaced with dashes)

**Target path calculation:**
```go
func ResolveWorktreePath(projectRoot, worktreesPath, branch string, pattern string) string {
    // 1. Resolve worktreesPath relative to projectRoot
    absWorktreesPath := filepath.Join(projectRoot, worktreesPath)

    // 2. Apply naming pattern
    dirName := strings.ReplaceAll(pattern, "{branch}", branch)
    dirName = strings.ReplaceAll(dirName, "/", "-")

    // 3. Construct final path
    return filepath.Join(absWorktreesPath, dirName)
}
```

---

## Hook System

### Hook Events

- **`postWorktreeCreate`**: Runs after creating a git worktree and registering the context
- **`preWorktreeDelete`**: Runs before deleting a worktree, while files still exist
- **`postWorktreeDelete`**: Runs after deleting a worktree and removing from registry

### Hook Configuration

Hook scripts are stored in `$PROJECT_ROOT/.dual/hooks/` and configured in `dual.config.yml`:

```yaml
version: 1

services:
  web:
    path: ./apps/web
  api:
    path: ./apps/api

worktrees:
  path: ../worktrees
  naming: "{branch}"

hooks:
  postWorktreeCreate:
    - setup-database.sh
    - setup-environment.sh
    - install-dependencies.sh
  preWorktreeDelete:
    - backup-data.sh
    - cleanup-database.sh
  postWorktreeDelete:
    - notify-team.sh
```

### Hook Environment Variables

Hook scripts receive the following environment variables:

- **`DUAL_EVENT`**: The hook event name (e.g., `postWorktreeCreate`)
- **`DUAL_CONTEXT_NAME`**: Context name (usually the branch name)
- **`DUAL_CONTEXT_PATH`**: Absolute path to the worktree directory
- **`DUAL_PROJECT_ROOT`**: Absolute path to the main repository

### Hook Execution Rules

- Scripts must be executable (`chmod +x`)
- Scripts run in sequence (not parallel)
- Non-zero exit code halts execution and fails the operation
- stdout/stderr are streamed to the user in real-time
- Scripts run with the worktree directory as working directory (except `postWorktreeDelete`)
- Hook failure during `dual create` leaves the worktree in place but may be partially configured
- Hook failure during `dual delete` halts deletion - worktree and registry entry remain

### Hook Script Examples

**Example 1: Port Assignment Hook**
```bash
#!/bin/bash
# .dual/hooks/setup-environment.sh

set -e

echo "Setting up environment for: $DUAL_CONTEXT_NAME"

# Calculate port based on context name hash
BASE_PORT=4000
CONTEXT_HASH=$(echo -n "$DUAL_CONTEXT_NAME" | md5sum | cut -c1-4)
PORT=$((BASE_PORT + 0x$CONTEXT_HASH % 1000))

# Write to .env file
cat > "$DUAL_CONTEXT_PATH/.env.local" <<EOF
PORT=$PORT
DATABASE_URL=postgresql://localhost/myapp_${DUAL_CONTEXT_NAME}
EOF

echo "Assigned port: $PORT"
```

**Example 2: Database Branch Setup**
```bash
#!/bin/bash
# .dual/hooks/setup-database.sh

set -e

echo "Creating database branch for: $DUAL_CONTEXT_NAME"

# Create PlanetScale branch
pscale branch create myapp "$DUAL_CONTEXT_NAME" --from main

# Get connection string
CONNECTION_URL=$(pscale connect myapp "$DUAL_CONTEXT_NAME" --format url)

# Update .env file
echo "DATABASE_URL=$CONNECTION_URL" >> "$DUAL_CONTEXT_PATH/.env.local"

echo "Database branch created"
```

**Example 3: Dependency Installation**
```bash
#!/bin/bash
# .dual/hooks/install-dependencies.sh

set -e

echo "Installing dependencies for: $DUAL_CONTEXT_NAME"

cd "$DUAL_CONTEXT_PATH"
pnpm install

echo "Dependencies installed"
```

**Example 4: Pre-Delete Backup**
```bash
#!/bin/bash
# .dual/hooks/backup-data.sh

set -e

echo "Backing up data for: $DUAL_CONTEXT_NAME"

# Backup database
pg_dump myapp_${DUAL_CONTEXT_NAME} > ~/backups/${DUAL_CONTEXT_NAME}.sql

echo "Data backed up to ~/backups/${DUAL_CONTEXT_NAME}.sql"
```

**Example 5: Post-Delete Cleanup**
```bash
#!/bin/bash
# .dual/hooks/cleanup-database.sh

set -e

echo "Cleaning up database for: $DUAL_CONTEXT_NAME"

# Delete PlanetScale branch
pscale branch delete myapp "$DUAL_CONTEXT_NAME" --force

echo "Database branch deleted"
```

### Hook Implementation

```go
// internal/hooks/executor.go

type Executor struct {
    ProjectRoot string
    HooksDir    string
    Verbose     bool
}

func (e *Executor) ExecuteHooks(event HookEvent, contextName, contextPath string, scripts []string) error {
    for _, script := range scripts {
        scriptPath := filepath.Join(e.HooksDir, script)

        // Check if script exists and is executable
        info, err := os.Stat(scriptPath)
        if err != nil {
            return fmt.Errorf("hook script not found: %s", script)
        }
        if info.Mode()&0111 == 0 {
            return fmt.Errorf("hook script not executable: %s", script)
        }

        // Prepare command
        cmd := exec.Command(scriptPath)
        cmd.Dir = contextPath  // Run in worktree directory
        cmd.Env = append(os.Environ(),
            fmt.Sprintf("DUAL_EVENT=%s", event),
            fmt.Sprintf("DUAL_CONTEXT_NAME=%s", contextName),
            fmt.Sprintf("DUAL_CONTEXT_PATH=%s", contextPath),
            fmt.Sprintf("DUAL_PROJECT_ROOT=%s", e.ProjectRoot),
        )

        // Stream output to user
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr

        // Execute
        if err := cmd.Run(); err != nil {
            return fmt.Errorf("hook script failed: %s (exit code %d)", script, cmd.ProcessState.ExitCode())
        }
    }

    return nil
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

    // Resolve symlinks
    cwdResolved, err := filepath.EvalSymlinks(cwdAbs)
    if err != nil {
        cwdResolved = cwdAbs
    }

    var longestMatch string
    longestMatchLen := 0

    // Check each service
    for name, service := range cfg.Services {
        servicePath := filepath.Join(projectRoot, service.Path)
        servicePathAbs, _ := filepath.Abs(servicePath)
        servicePathResolved, _ := filepath.EvalSymlinks(servicePathAbs)

        // Check if CWD is within service path
        if strings.HasPrefix(cwdResolved, servicePathResolved) {
            matchLen := len(servicePathResolved)
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

The registry is **project-local** at `$PROJECT_ROOT/.dual/.local/registry.json`:

```json
{
  "projects": {
    "<absolute-project-path>": {
      "contexts": {
        "<context-name>": {
          "path": "/optional/worktree/path",
          "created": "2025-10-14T10:00:00Z"
        }
      }
    }
  }
}
```

**Key points:**
- Each project has its own registry file
- All worktrees of a repository share the parent repo's registry (normalized via `GetProjectIdentifier()`)
- The registry should be added to `.gitignore` to avoid committing context mappings
- File locking ensures concurrent dual operations don't corrupt the registry

### File Locking

Registry operations use file locking (`gofrs/flock`) to prevent concurrent access issues:

```go
type Registry struct {
    Projects map[string]Project `json:"projects"`
    mu       sync.RWMutex       `json:"-"`
    flock    *flock.Flock       `json:"-"`  // File lock
}

func LoadRegistry(registryPath string) (*Registry, error) {
    // Create file lock
    lockPath := registryPath + ".lock"
    fileLock := flock.New(lockPath)

    // Try to acquire lock with timeout (5 seconds)
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    locked, err := fileLock.TryLockContext(ctx, 100*time.Millisecond)
    if !locked {
        return nil, ErrLockTimeout
    }

    // Load registry data...
    // Lock is held until Close() is called

    return registry, nil
}

func (r *Registry) Close() error {
    if r.flock != nil {
        return r.flock.Unlock()
    }
    return nil
}
```

**Lock behavior**:
- Lock acquired in `LoadRegistry()` and held until `Close()`
- Timeout: 5 seconds (prevents deadlocks)
- Lock file: `$PROJECT_ROOT/.dual/.local/registry.json.lock`
- Must call `Close()` to release lock (use `defer reg.Close()`)

### Thread Safety

Registry operations are thread-safe using both file locking and in-memory mutex:

```go
func (r *Registry) GetContext(projectPath, contextName string) (*Context, error) {
    r.mu.RLock()  // In-memory read lock
    defer r.mu.RUnlock()
    // ... read operations
}

func (r *Registry) SetContext(projectPath, contextName string, ctx *Context) error {
    r.mu.Lock()  // In-memory write lock
    defer r.mu.Unlock()
    // ... write operations
}
```

**Two-level locking strategy**:
1. **File lock** (`flock`): Prevents concurrent access across processes
2. **In-memory mutex** (`sync.RWMutex`): Prevents concurrent access within a process

### Atomic Writes

Registry updates use atomic write pattern:

```go
func (r *Registry) SaveRegistry(registryPath string) error {
    r.mu.Lock()  // Acquire in-memory lock
    defer r.mu.Unlock()

    // 1. Marshal to JSON
    data, _ := json.MarshalIndent(r, "", "  ")

    // 2. Write to temporary file
    tempFile := registryPath + ".tmp"
    os.WriteFile(tempFile, data, 0600)

    // 3. Atomic rename (POSIX guarantees atomicity)
    os.Rename(tempFile, registryPath)

    return nil
}
```

This prevents corruption if:
- Process crashes during write
- Multiple dual instances run concurrently (prevented by file lock)
- Disk fills up during write
- SIGKILL during save operation

---

## Configuration System

### Configuration File (`dual.config.yml`)

```yaml
version: 1

services:
  <service-name>:
    path: <relative-path>
    envFile: <relative-env-file-path>  # Optional

worktrees:
  path: <relative-path>          # Optional, required for dual create/delete
  naming: "{branch}"             # Optional, defaults to {branch}

hooks:
  postWorktreeCreate:            # Optional
    - script1.sh
    - script2.sh
  preWorktreeDelete:             # Optional
    - script3.sh
  postWorktreeDelete:            # Optional
    - script4.sh
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
            return fmt.Errorf("service %q: path must be relative", name)
        }

        // Path must exist
        fullPath := filepath.Join(projectRoot, service.Path)
        if _, err := os.Stat(fullPath); err != nil {
            return fmt.Errorf("service %q: path does not exist: %s", name, service.Path)
        }

        // EnvFile directory must exist (if specified)
        if service.EnvFile != "" {
            envFileDir := filepath.Dir(filepath.Join(projectRoot, service.EnvFile))
            if _, err := os.Stat(envFileDir); err != nil {
                return fmt.Errorf("service %q: envFile directory does not exist", name)
            }
        }
    }

    // Validate worktrees configuration (if present)
    if config.Worktrees != nil {
        if filepath.IsAbs(config.Worktrees.Path) {
            return fmt.Errorf("worktrees.path must be relative")
        }
        // Note: path doesn't need to exist yet, will be created
    }

    // Validate hooks configuration (if present)
    if config.Hooks != nil {
        hooksDir := filepath.Join(projectRoot, ".dual", "hooks")
        for event, scripts := range config.Hooks {
            for _, script := range scripts {
                scriptPath := filepath.Join(hooksDir, script)
                if _, err := os.Stat(scriptPath); err != nil {
                    return fmt.Errorf("hook script not found: %s (event: %s)", script, event)
                }
            }
        }
    }

    return nil
}
```

---

## Concurrency and Thread Safety

### Registry Access

Multiple `dual` commands may run simultaneously:

```bash
# Terminal 1
dual create feature-1

# Terminal 2 (at the same time)
dual create feature-2

# Terminal 3 (at the same time)
dual context list
```

Registry uses **two-level locking** for maximum safety:

**Level 1: File Lock** (`gofrs/flock`)
- Prevents concurrent access across processes
- Lock acquired in `LoadRegistry()`, released in `Close()`
- Timeout: 5 seconds (returns `ErrLockTimeout` if exceeded)
- Lock file: `$PROJECT_ROOT/.dual/.local/registry.json.lock`

**Level 2: In-Memory Mutex** (`sync.RWMutex`)
- Prevents concurrent access within a process
- Multiple readers can access simultaneously
- Writers get exclusive access
- Prevents race conditions in goroutines

```go
// Example usage pattern
reg, err := registry.LoadRegistry(registryPath)  // Acquires file lock
if err != nil {
    return err
}
defer reg.Close()  // MUST release lock

// Operations are safe - both file lock and in-memory mutex protect data
ctx, err := reg.GetContext(projectPath, contextName)
// ...
err = reg.SaveRegistry(registryPath)
```

### File I/O

All file operations use atomic patterns:

1. **Config writes**: Temp file + atomic rename
2. **Registry writes**: Temp file + atomic rename + file locking
3. **Hook script execution**: Sequential, streaming output

### Lock Timeout Behavior

If a lock cannot be acquired within 5 seconds:

```go
Error: timeout waiting for registry lock (waited 5s)
Hint: Another dual command may be running. Wait for it to complete or check for hung processes.
```

This prevents:
- Deadlocks from crashed processes
- Indefinite waiting when something goes wrong
- Silent failures that corrupt data

### Concurrent Safety Guarantees

- ✅ Multiple dual commands reading registry (blocked by file lock, executed serially)
- ✅ Multiple dual commands creating worktrees (file lock serializes access)
- ✅ One dual writing registry while others wait (file lock serializes access)
- ✅ Crashed writes don't corrupt registry (atomic writes + temp files)
- ✅ Lock timeout prevents deadlocks (5 second timeout)
- ✅ Two dual commands creating same context simultaneously (file lock prevents race)
- ✅ Registry operations across processes are serialized (file lock)
- ⚠️ Lock timeout may cause "busy" errors under extreme load (acceptable tradeoff)

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

// Worktree
var (
    ErrNotInMainRepo    = errors.New("must be in main repository to create worktree")
    ErrWorktreeExists   = errors.New("worktree already exists")
    ErrNotAWorktree     = errors.New("context is not a worktree")
)

// Hooks
var (
    ErrHookScriptNotFound     = errors.New("hook script not found")
    ErrHookScriptNotExecutable = errors.New("hook script not executable")
    ErrHookExecutionFailed    = errors.New("hook execution failed")
)
```

### Error Messages

All errors include helpful hints:

```
Error: context "feature-x" not found in registry
Hint: Run 'dual create feature-x' to create this context

Error: worktrees.path not configured in dual.config.yml
Hint: Add worktrees configuration to use 'dual create'

Error: hook script not executable: setup-database.sh
Hint: Run 'chmod +x .dual/hooks/setup-database.sh'
```

### Error Handling Strategy

1. **Recoverable errors**: Show error + hint, exit 1
2. **Unrecoverable errors**: Show error, exit 1
3. **Hook execution errors**: Show error, preserve hook exit code
4. **Corrupt registry**: Warn, create new empty registry

---

## Design Decisions

### Why Hook-Based Architecture?

**Problem**: Different teams need different automation (ports, databases, dependencies, etc.)

**Solution**: Core tool manages worktree lifecycle; users implement custom logic in hooks

**Benefits**:
- Maximum flexibility - implement any workflow
- Zero assumptions about project structure
- Easy to customize per-project
- Users control complexity

**Tradeoff**: Requires writing hook scripts, but examples are provided

### Why Project-Local Registry?

**Alternative**: Store registry in user home directory (`~/.dual/.local/registry.json`)

**Problem**:
- Git worktrees share same project root
- Multiple clones have separate configs
- Team collaboration difficult

**Solution**: `$PROJECT_ROOT/.dual/.local/registry.json` in project directory

**Benefits**:
- Single source of truth per project
- Works across worktrees and clones
- Each project is isolated
- Can be shared (but typically .gitignored)

### Why Not Modify Files?

**Problem**: Tools like `vercel pull` overwrite `.env` files

**Solution**: Encourage using hooks to write to files instead of direct file modification

**Benefits**:
- No conflicts with tool-generated files
- No git diffs for environment changes
- Cleaner separation of concerns
- Users control what gets written

**Tradeoff**: Requires hook setup, but provides flexibility

### Why Support `.dual-context` File?

**Use Cases**:
1. **Long branch names**: `feature/JIRA-1234-very-long-description` → `feat-1234`
2. **Detached HEAD**: Working with specific commits
3. **Non-git projects**: Manual context management
4. **Testing**: Override detection for testing

**Tradeoff**: Another thing to manage, but optional

### Why Restrict `dual create` to Project Root?

**Problem**: Running `dual create` from within a worktree can cause confusion

**Solution**: Enforce that `dual create` only runs from the main repository

**Benefits**:
- Clear mental model: main repo = worktree factory
- Prevents nested worktrees
- Avoids registry confusion

**Implementation**: Check if current directory is a worktree before creating

---

## Performance Considerations

### Startup Time

Typical execution time: **< 100ms**

Breakdown:
- Load config: ~10ms
- Detect context: ~15ms (git command)
- Load registry: ~10ms
- Hook execution: variable (depends on hook scripts)

### Optimization Techniques

1. **No unnecessary file I/O**: Only read files when needed
2. **Compiled binary**: No interpreter startup
3. **Minimal dependencies**: Small binary size (~5MB)
4. **Efficient path matching**: O(n) where n = number of services

### Scalability

- **Services per project**: No practical limit (tested up to 100)
- **Contexts per project**: No practical limit (tested up to 50)
- **Projects in registry**: No practical limit
- **Registry file size**: Stays small (~1KB per project)
- **Hook execution**: Sequential, scales linearly with number of hooks

---

## Testing Strategy

### Unit Tests

Each component has unit tests:

```
internal/config/config_test.go
internal/registry/registry_test.go
internal/context/detector_test.go
internal/service/detector_test.go
internal/hooks/executor_test.go
internal/worktree/manager_test.go
```

### Integration Tests

End-to-end workflow tests:

```
test/integration/worktree_lifecycle_test.go
test/integration/hooks_test.go
test/integration/config_validation_test.go
```

### Test Coverage

- Config loading and validation
- Registry CRUD operations
- Context detection (with mocks for git)
- Service detection (with temp directories)
- Hook execution (with mock scripts)
- Worktree creation and deletion

### Dependency Injection

Components use dependency injection for testability:

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

1. **Worktree Templates**: Pre-configured hook sets for common workflows
2. **Hook Output Parsing**: Parse hook output for structured data
3. **Parallel Hook Execution**: Run independent hooks in parallel
4. **Hook Timeouts**: Prevent hung hook scripts
5. **Visual Dashboard**: `dual ui` - web interface showing all worktrees
6. **Hook Marketplace**: Share hook scripts with community
7. **Configuration Validation**: `dual doctor --fix` to repair issues
8. **Migration Tools**: Upgrade config versions automatically

### Architecture Extensions

1. **Plugin System**: Allow custom detection strategies
2. **Remote Registry**: Share contexts across team
3. **Watch Mode**: Auto-detect worktree changes
4. **Hook Library**: Standard library of reusable hooks

---

## References

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [YAML v3](https://github.com/go-yaml/yaml) - YAML parsing
- [gofrs/flock](https://github.com/gofrs/flock) - File locking
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
