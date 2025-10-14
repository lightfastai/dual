package ports

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/lightfastai/dual/internal/config"
	"github.com/lightfastai/dual/internal/registry"
	"github.com/lightfastai/dual/internal/service"
)

// ProcessInfo contains information about a process using a port
type ProcessInfo struct {
	PID     int
	Name    string
	Command string
	User    string
}

// BasePortConflict represents a conflict where the same base port is assigned to multiple contexts
type BasePortConflict struct {
	BasePort int
	Contexts []ContextInfo
}

// ContextInfo contains information about a context
type ContextInfo struct {
	ProjectPath string
	ContextName string
	BasePort    int
}

// PortRangeOverlap represents overlapping port ranges between contexts
type PortRangeOverlap struct {
	Context1    ContextInfo
	Context2    ContextInfo
	StartPort1  int
	EndPort1    int
	StartPort2  int
	EndPort2    int
	OverlapPort int
}

// FindDuplicateBasePorts checks the registry for duplicate base ports across contexts
// Returns a map of base port to list of contexts using that port
func FindDuplicateBasePorts(reg *registry.Registry) []BasePortConflict {
	basePortMap := make(map[int][]ContextInfo)

	// Collect all base ports and their contexts
	for projectPath, project := range reg.Projects {
		for contextName, ctx := range project.Contexts {
			info := ContextInfo{
				ProjectPath: projectPath,
				ContextName: contextName,
				BasePort:    ctx.BasePort,
			}
			basePortMap[ctx.BasePort] = append(basePortMap[ctx.BasePort], info)
		}
	}

	// Find conflicts (base ports used by more than one context)
	var conflicts []BasePortConflict
	for basePort, contexts := range basePortMap {
		if len(contexts) > 1 {
			conflicts = append(conflicts, BasePortConflict{
				BasePort: basePort,
				Contexts: contexts,
			})
		}
	}

	return conflicts
}

// IsPortInUse checks if a port is currently in use by attempting to bind to it
// Returns true if the port is in use, false if available
func IsPortInUse(port int) bool {
	// Try to listen on the port
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		// Port is in use or cannot be bound
		return true
	}

	// Port is available, close the listener
	_ = listener.Close()
	return false
}

// GetProcessUsingPort attempts to get information about the process using a port
// This is platform-specific and may not work on all systems
func GetProcessUsingPort(port int) (*ProcessInfo, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin", "linux":
		// Use lsof on macOS and Linux
		// #nosec G204 - port is validated as an integer
		cmd = exec.Command("lsof", "-i", fmt.Sprintf(":%d", port), "-P", "-n")
	case "windows":
		// Use netstat on Windows
		// #nosec G204 - port is validated as an integer
		cmd = exec.Command("netstat", "-ano", "-p", "tcp")
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Command failed, likely means no process is using the port
		return nil, fmt.Errorf("failed to get process info: %w", err)
	}

	// Parse output based on platform
	switch runtime.GOOS {
	case "darwin", "linux":
		return parseLsofOutput(string(output), port)
	case "windows":
		return parseNetstatOutput(string(output), port)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// parseLsofOutput parses lsof output to extract process information
func parseLsofOutput(output string, port int) (*ProcessInfo, error) {
	lines := strings.Split(output, "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("no process found using port %d", port)
	}

	// Skip the header line and parse the first data line
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		// lsof output format: COMMAND PID USER FD TYPE DEVICE SIZE/OFF NODE NAME
		pid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}

		return &ProcessInfo{
			PID:     pid,
			Name:    fields[0],
			User:    fields[2],
			Command: strings.Join(fields, " "),
		}, nil
	}

	return nil, fmt.Errorf("no process found using port %d", port)
}

// parseNetstatOutput parses netstat output on Windows to extract process information
func parseNetstatOutput(output string, port int) (*ProcessInfo, error) {
	lines := strings.Split(output, "\n")
	portStr := fmt.Sprintf(":%d", port)

	for _, line := range lines {
		if !strings.Contains(line, portStr) || !strings.Contains(line, "LISTENING") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		// netstat output format: Proto Local Address Foreign Address State PID
		pid, err := strconv.Atoi(fields[4])
		if err != nil {
			continue
		}

		return &ProcessInfo{
			PID:     pid,
			Name:    "unknown",
			User:    "unknown",
			Command: line,
		}, nil
	}

	return nil, fmt.Errorf("no process found using port %d", port)
}

// CheckPortRangeOverlap detects overlapping port ranges between contexts
// For each context, the port range is basePort+1 to basePort+numServices
func CheckPortRangeOverlap(reg *registry.Registry, cfg *config.Config, projectPath string) ([]PortRangeOverlap, error) {
	var overlaps []PortRangeOverlap

	// Get all contexts for the project
	contexts, err := reg.ListContexts(projectPath)
	if err != nil {
		// If project doesn't exist, no overlaps
		return overlaps, nil
	}

	// Calculate port ranges for each context
	numServices := len(cfg.Services)
	if numServices == 0 {
		return overlaps, nil
	}

	// Convert contexts map to slice for easier iteration
	type contextWithName struct {
		name string
		ctx  registry.Context
	}
	var contextList []contextWithName
	for name, ctx := range contexts {
		contextList = append(contextList, contextWithName{name: name, ctx: ctx})
	}

	// Check each pair of contexts for overlaps
	for i := 0; i < len(contextList); i++ {
		for j := i + 1; j < len(contextList); j++ {
			ctx1 := contextList[i]
			ctx2 := contextList[j]

			startPort1 := ctx1.ctx.BasePort + 1
			endPort1 := ctx1.ctx.BasePort + numServices
			startPort2 := ctx2.ctx.BasePort + 1
			endPort2 := ctx2.ctx.BasePort + numServices

			// Check for overlap
			if startPort1 <= endPort2 && startPort2 <= endPort1 {
				// Ranges overlap - find the overlapping port
				overlapStart := max(startPort1, startPort2)

				overlaps = append(overlaps, PortRangeOverlap{
					Context1: ContextInfo{
						ProjectPath: projectPath,
						ContextName: ctx1.name,
						BasePort:    ctx1.ctx.BasePort,
					},
					Context2: ContextInfo{
						ProjectPath: projectPath,
						ContextName: ctx2.name,
						BasePort:    ctx2.ctx.BasePort,
					},
					StartPort1:  startPort1,
					EndPort1:    endPort1,
					StartPort2:  startPort2,
					EndPort2:    endPort2,
					OverlapPort: overlapStart,
				})
			}
		}
	}

	return overlaps, nil
}

// FindNextAvailableBasePort suggests the next free base port
// It checks the registry for used base ports and also verifies that the suggested
// port range doesn't overlap with any existing context
func FindNextAvailableBasePort(reg *registry.Registry, cfg *config.Config, projectPath string) int {
	usedPorts := make(map[int]bool)

	// Collect all used base ports globally
	for _, project := range reg.Projects {
		for _, context := range project.Contexts {
			usedPorts[context.BasePort] = true
		}
	}

	// Calculate the port range we need for this project
	numServices := len(cfg.Services)
	if numServices == 0 {
		numServices = 1 // At least need space for 1 port
	}

	// Find next available port starting from DefaultBasePort
	nextPort := registry.DefaultBasePort
	for {
		// Check if base port is used
		if usedPorts[nextPort] {
			nextPort += registry.PortIncrement
			continue
		}

		// Check if the port range [nextPort+1, nextPort+numServices] is clear
		rangeIsClear := true
		for _, project := range reg.Projects {
			for _, context := range project.Contexts {
				startExisting := context.BasePort + 1
				endExisting := context.BasePort + numServices // Conservative: assume same number of services
				startNew := nextPort + 1
				endNew := nextPort + numServices

				// Check for overlap
				if startNew <= endExisting && startExisting <= endNew {
					rangeIsClear = false
					break
				}
			}
			if !rangeIsClear {
				break
			}
		}

		if rangeIsClear {
			return nextPort
		}

		nextPort += registry.PortIncrement
	}
}

// CheckContextPortConflict checks if creating a context with the given base port
// would conflict with existing contexts
func CheckContextPortConflict(reg *registry.Registry, cfg *config.Config, projectPath string, basePort int) error {
	// Check if base port is already used
	for _, project := range reg.Projects {
		for contextName, context := range project.Contexts {
			if context.BasePort == basePort {
				return fmt.Errorf("base port %d already assigned to context '%s'", basePort, contextName)
			}
		}
	}

	// Check for port range overlaps
	numServices := len(cfg.Services)
	if numServices == 0 {
		return nil // No services, no overlap possible
	}

	startNew := basePort + 1
	endNew := basePort + numServices

	// Check against existing contexts in the same project
	contexts, err := reg.ListContexts(projectPath)
	if err != nil {
		// Project doesn't exist yet, no conflicts
		return nil
	}

	for contextName, context := range contexts {
		startExisting := context.BasePort + 1
		endExisting := context.BasePort + numServices

		// Check for overlap
		if startNew <= endExisting && startExisting <= endNew {
			return fmt.Errorf("port range [%d-%d] overlaps with context '%s' range [%d-%d]",
				startNew, endNew, contextName, startExisting, endExisting)
		}
	}

	return nil
}

// CheckServicePortInUse checks if the calculated port for a service is currently in use
// Returns the port number and process info if in use
func CheckServicePortInUse(cfg *config.Config, reg *registry.Registry, projectPath, contextName, serviceName string) (int, *ProcessInfo, error) {
	// Calculate the port for this service
	port, err := service.CalculatePort(cfg, reg, projectPath, contextName, serviceName)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to calculate port: %w", err)
	}

	// Check if port is in use
	if !IsPortInUse(port) {
		return port, nil, nil
	}

	// Port is in use, try to get process info
	processInfo, _ := GetProcessUsingPort(port)
	return port, processInfo, nil
}

// Helper function for max (Go 1.21+)
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
