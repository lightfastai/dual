package service

import (
	"errors"
	"fmt"
	"sort"

	"github.com/lightfastai/dual/internal/config"
	"github.com/lightfastai/dual/internal/logger"
	"github.com/lightfastai/dual/internal/registry"
)

// ErrContextNotFound is returned when the context is not found in the registry
var ErrContextNotFound = fmt.Errorf("context not found in registry")

// ErrServiceNotFound is returned when the service is not found in the config
var ErrServiceNotFound = fmt.Errorf("service not found in config")

// Calculator handles port calculation logic
type Calculator struct{}

// NewCalculator creates a new Calculator
func NewCalculator() *Calculator {
	return &Calculator{}
}

// CalculatePort calculates the port for a given service in a context
// Formula: port = basePort + serviceIndex + 1
// where serviceIndex is the 0-based index of the service in config.Services (ordered alphabetically)
func (c *Calculator) CalculatePort(cfg *config.Config, reg *registry.Registry, projectPath, contextName, serviceName string) (int, error) {
	// Get the context from the registry
	ctx, err := reg.GetContext(projectPath, contextName)
	if err != nil {
		if errors.Is(err, registry.ErrContextNotFound) || errors.Is(err, registry.ErrProjectNotFound) {
			return 0, ErrContextNotFound
		}
		return 0, fmt.Errorf("failed to get context: %w", err)
	}

	logger.Debug("Base port: %d", ctx.BasePort)

	// Check if the service exists in the config
	if _, exists := cfg.Services[serviceName]; !exists {
		return 0, ErrServiceNotFound
	}

	// Get the service index (services are ordered alphabetically for determinism)
	serviceIndex := c.getServiceIndex(cfg, serviceName)
	logger.Debug("Service index: %d", serviceIndex)

	// Calculate port: basePort + serviceIndex + 1
	port := ctx.BasePort + serviceIndex + 1
	logger.Debug("Formula: %d + %d + 1 = %d", ctx.BasePort, serviceIndex, port)
	logger.Success("Port: %d", port)

	return port, nil
}

// getServiceIndex returns the 0-based index of a service in the config
// Services are sorted alphabetically to ensure deterministic ordering
func (c *Calculator) getServiceIndex(cfg *config.Config, serviceName string) int {
	// Get all service names and sort them alphabetically
	serviceNames := make([]string, 0, len(cfg.Services))
	for name := range cfg.Services {
		serviceNames = append(serviceNames, name)
	}
	sort.Strings(serviceNames)

	// Find the index of the target service
	for i, name := range serviceNames {
		if name == serviceName {
			return i
		}
	}

	// Should never reach here if service exists in config
	return -1
}

// GetServiceIndex is a helper function that returns the service index without needing a Calculator instance
func GetServiceIndex(cfg *config.Config, serviceName string) int {
	calc := NewCalculator()
	return calc.getServiceIndex(cfg, serviceName)
}

// CalculatePort is a convenience function that creates a calculator and calculates the port
func CalculatePort(cfg *config.Config, reg *registry.Registry, projectPath, contextName, serviceName string) (int, error) {
	calc := NewCalculator()
	return calc.CalculatePort(cfg, reg, projectPath, contextName, serviceName)
}

// CalculateAllPorts returns a map of all service names to their calculated ports for a given context
func CalculateAllPorts(cfg *config.Config, reg *registry.Registry, projectPath, contextName string) (map[string]int, error) {
	// Get the context to ensure it exists
	ctx, err := reg.GetContext(projectPath, contextName)
	if err != nil {
		if errors.Is(err, registry.ErrContextNotFound) || errors.Is(err, registry.ErrProjectNotFound) {
			return nil, ErrContextNotFound
		}
		return nil, fmt.Errorf("failed to get context: %w", err)
	}

	// Get sorted service names
	serviceNames := make([]string, 0, len(cfg.Services))
	for name := range cfg.Services {
		serviceNames = append(serviceNames, name)
	}
	sort.Strings(serviceNames)

	// Calculate port for each service
	ports := make(map[string]int)
	for i, name := range serviceNames {
		ports[name] = ctx.BasePort + i + 1
	}

	return ports, nil
}
