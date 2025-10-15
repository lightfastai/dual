package hooks

// HookEvent represents a lifecycle event that can trigger hooks
type HookEvent string

const (
	// PostWorktreeCreate is triggered after a worktree is created
	PostWorktreeCreate HookEvent = "postWorktreeCreate"

	// PreWorktreeDelete is triggered before a worktree is deleted
	PreWorktreeDelete HookEvent = "preWorktreeDelete"

	// PostWorktreeDelete is triggered after a worktree is deleted
	PostWorktreeDelete HookEvent = "postWorktreeDelete"

	// PostEnvChange is triggered after environment variables are changed
	PostEnvChange HookEvent = "postEnvChange"
)

// String returns the string representation of a HookEvent
func (e HookEvent) String() string {
	return string(e)
}

// IsValid checks if a HookEvent is one of the recognized events
func (e HookEvent) IsValid() bool {
	switch e {
	case PostWorktreeCreate, PreWorktreeDelete, PostWorktreeDelete, PostEnvChange:
		return true
	default:
		return false
	}
}

// HookContext contains all the context information passed to a hook
type HookContext struct {
	// Event is the lifecycle event that triggered this hook
	Event HookEvent

	// ContextName is the name of the dual context (usually branch name)
	ContextName string

	// ContextPath is the absolute path to the context (worktree path)
	ContextPath string

	// ProjectRoot is the absolute path to the main project repository
	ProjectRoot string
}

// HookResult contains the result of executing a hook
type HookResult struct {
	// Script is the name of the hook script that was executed
	Script string

	// Success indicates whether the hook executed successfully
	Success bool

	// Error is the error if the hook failed
	Error error

	// Output contains stdout/stderr from the hook (optional, for debugging)
	Output string
}
