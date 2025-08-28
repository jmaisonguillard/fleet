package validation

import (
	"fmt"
	"strconv"
	"strings"
)

// PortValidator checks for port conflicts and validates port usage
type PortValidator struct {
	portMap      map[int][]string // port -> list of services using it
	reservedPorts map[int]string   // port -> reason for reservation
}

// NewPortValidator creates a new port validator
func NewPortValidator() *PortValidator {
	pv := &PortValidator{
		portMap:      make(map[int][]string),
		reservedPorts: make(map[int]string),
	}
	
	// Add commonly reserved ports
	pv.AddReservedPort(22, "SSH")
	pv.AddReservedPort(53, "DNS")
	pv.AddReservedPort(443, "HTTPS")
	pv.AddReservedPort(3306, "MySQL default")
	pv.AddReservedPort(5432, "PostgreSQL default")
	pv.AddReservedPort(6379, "Redis default")
	pv.AddReservedPort(27017, "MongoDB default")
	
	return pv
}

// AddReservedPort marks a port as reserved
func (pv *PortValidator) AddReservedPort(port int, reason string) {
	pv.reservedPorts[port] = reason
}

// RegisterPort registers a port usage by a service
func (pv *PortValidator) RegisterPort(port int, serviceName string) error {
	if err := ValidatePort(port); err != nil {
		return err
	}
	
	if reason, reserved := pv.reservedPorts[port]; reserved {
		return fmt.Errorf("port %d is reserved for %s", port, reason)
	}
	
	pv.portMap[port] = append(pv.portMap[port], serviceName)
	return nil
}

// RegisterPortRange registers a port range (e.g., "8080:80")
func (pv *PortValidator) RegisterPortRange(portSpec string, serviceName string) error {
	// Parse port specification
	parts := strings.Split(portSpec, ":")
	if len(parts) == 0 {
		return fmt.Errorf("invalid port specification: %s", portSpec)
	}
	
	// Get the host port (first part)
	hostPort, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return fmt.Errorf("invalid host port in '%s': %v", portSpec, err)
	}
	
	return pv.RegisterPort(hostPort, serviceName)
}

// CheckConflicts returns all port conflicts
func (pv *PortValidator) CheckConflicts() map[int][]string {
	conflicts := make(map[int][]string)
	
	for port, services := range pv.portMap {
		if len(services) > 1 {
			conflicts[port] = services
		}
	}
	
	return conflicts
}

// HasConflicts returns true if there are port conflicts
func (pv *PortValidator) HasConflicts() bool {
	for _, services := range pv.portMap {
		if len(services) > 1 {
			return true
		}
	}
	return false
}

// GetConflictReport returns a formatted conflict report
func (pv *PortValidator) GetConflictReport() string {
	conflicts := pv.CheckConflicts()
	if len(conflicts) == 0 {
		return ""
	}
	
	var report []string
	report = append(report, "Port conflicts detected:")
	
	for port, services := range conflicts {
		report = append(report, fmt.Sprintf("  Port %d is used by: %s", 
			port, strings.Join(services, ", ")))
	}
	
	return strings.Join(report, "\n")
}

// Validate performs validation and returns results
func (pv *PortValidator) Validate() *Validator {
	validator := NewValidator()
	
	// Check for conflicts
	if pv.HasConflicts() {
		validator.AddError(fmt.Errorf(pv.GetConflictReport()))
	}
	
	// Check for use of reserved ports
	for port, services := range pv.portMap {
		if reason, reserved := pv.reservedPorts[port]; reserved {
			for _, service := range services {
				validator.AddWarning(fmt.Sprintf(
					"Service '%s' uses reserved port %d (%s)", 
					service, port, reason))
			}
		}
	}
	
	return validator
}

// Clear resets the port validator
func (pv *PortValidator) Clear() {
	pv.portMap = make(map[int][]string)
}

// GetUsedPorts returns all ports in use
func (pv *PortValidator) GetUsedPorts() []int {
	ports := make([]int, 0, len(pv.portMap))
	for port := range pv.portMap {
		ports = append(ports, port)
	}
	return ports
}

// GetServicesOnPort returns services using a specific port
func (pv *PortValidator) GetServicesOnPort(port int) []string {
	return pv.portMap[port]
}

// IsPortAvailable checks if a port is available
func (pv *PortValidator) IsPortAvailable(port int) bool {
	_, used := pv.portMap[port]
	_, reserved := pv.reservedPorts[port]
	return !used && !reserved
}

// SuggestAvailablePort suggests an available port near the requested one
func (pv *PortValidator) SuggestAvailablePort(preferredPort int) int {
	// Check preferred port first
	if pv.IsPortAvailable(preferredPort) {
		return preferredPort
	}
	
	// Try ports in range +/- 100
	for offset := 1; offset <= 100; offset++ {
		// Try higher
		port := preferredPort + offset
		if port <= 65535 && pv.IsPortAvailable(port) {
			return port
		}
		
		// Try lower
		port = preferredPort - offset
		if port > 0 && pv.IsPortAvailable(port) {
			return port
		}
	}
	
	// Fallback to finding any available port in common ranges
	ranges := []struct{ start, end int }{
		{8000, 9000},  // Common development ports
		{3000, 4000},  // Node.js apps
		{5000, 6000},  // Python apps
		{9000, 10000}, // Various services
	}
	
	for _, r := range ranges {
		for port := r.start; port <= r.end; port++ {
			if pv.IsPortAvailable(port) {
				return port
			}
		}
	}
	
	// Last resort: find any available port
	for port := 1024; port <= 65535; port++ {
		if pv.IsPortAvailable(port) {
			return port
		}
	}
	
	return 0 // No available port found
}