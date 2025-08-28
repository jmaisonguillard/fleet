package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// VolumeManager manages Docker volumes for services
type VolumeManager struct {
	namedVolumes  map[string]VolumeConfig // Named volumes
	bindMounts    map[string]BindMount    // Bind mounts
	volumeUsage   map[string][]string     // volume -> services using it
}

// VolumeConfig represents a named volume configuration
type VolumeConfig struct {
	Name       string
	Driver     string
	DriverOpts map[string]string
	Labels     map[string]string
}

// BindMount represents a bind mount configuration
type BindMount struct {
	Source   string
	Target   string
	ReadOnly bool
	Service  string
}

// NewVolumeManager creates a new volume manager
func NewVolumeManager() *VolumeManager {
	return &VolumeManager{
		namedVolumes: make(map[string]VolumeConfig),
		bindMounts:   make(map[string]BindMount),
		volumeUsage:  make(map[string][]string),
	}
}

// ParseVolumeSpec parses a volume specification string
func (vm *VolumeManager) ParseVolumeSpec(spec string, serviceName string) (*VolumeSpec, error) {
	vs := &VolumeSpec{}
	
	// Split by colons
	parts := strings.Split(spec, ":")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid volume specification: %s", spec)
	}
	
	vs.Source = parts[0]
	vs.Target = parts[1]
	
	// Check for read-only flag
	if len(parts) > 2 {
		if parts[2] == "ro" || parts[2] == "readonly" {
			vs.ReadOnly = true
		}
	}
	
	// Determine volume type
	if vm.isNamedVolume(vs.Source) {
		vs.Type = VolumeTypeNamed
	} else if vm.isBindMount(vs.Source) {
		vs.Type = VolumeTypeBind
	} else {
		vs.Type = VolumeTypeUnknown
	}
	
	return vs, nil
}

// VolumeSpec represents a parsed volume specification
type VolumeSpec struct {
	Type     VolumeType
	Source   string
	Target   string
	ReadOnly bool
}

// VolumeType represents the type of volume
type VolumeType int

const (
	VolumeTypeUnknown VolumeType = iota
	VolumeTypeNamed
	VolumeTypeBind
)

// isNamedVolume checks if a source is a named volume
func (vm *VolumeManager) isNamedVolume(source string) bool {
	// Named volumes don't contain path separators or start with . or /
	return !strings.Contains(source, "/") && 
		!strings.HasPrefix(source, ".") && 
		!strings.HasPrefix(source, "~")
}

// isBindMount checks if a source is a bind mount
func (vm *VolumeManager) isBindMount(source string) bool {
	// Bind mounts contain path separators or start with . or / or ~
	return strings.Contains(source, "/") || 
		strings.HasPrefix(source, ".") || 
		strings.HasPrefix(source, "/") || 
		strings.HasPrefix(source, "~")
}

// AddVolume adds a volume for a service
func (vm *VolumeManager) AddVolume(spec string, serviceName string) error {
	vs, err := vm.ParseVolumeSpec(spec, serviceName)
	if err != nil {
		return err
	}
	
	switch vs.Type {
	case VolumeTypeNamed:
		vm.AddNamedVolume(vs.Source, serviceName)
	case VolumeTypeBind:
		vm.AddBindMount(vs.Source, vs.Target, vs.ReadOnly, serviceName)
	default:
		return fmt.Errorf("unknown volume type for: %s", spec)
	}
	
	return nil
}

// AddNamedVolume adds a named volume
func (vm *VolumeManager) AddNamedVolume(name string, serviceName string) {
	if _, exists := vm.namedVolumes[name]; !exists {
		vm.namedVolumes[name] = VolumeConfig{
			Name:   name,
			Driver: "local",
		}
	}
	
	// Track usage
	vm.volumeUsage[name] = append(vm.volumeUsage[name], serviceName)
}

// AddBindMount adds a bind mount
func (vm *VolumeManager) AddBindMount(source, target string, readOnly bool, serviceName string) {
	key := fmt.Sprintf("%s:%s", serviceName, target)
	vm.bindMounts[key] = BindMount{
		Source:   source,
		Target:   target,
		ReadOnly: readOnly,
		Service:  serviceName,
	}
}

// GetNamedVolumes returns all named volumes
func (vm *VolumeManager) GetNamedVolumes() map[string]VolumeConfig {
	return vm.namedVolumes
}

// GetBindMounts returns all bind mounts
func (vm *VolumeManager) GetBindMounts() map[string]BindMount {
	return vm.bindMounts
}

// GetVolumeUsage returns services using a specific volume
func (vm *VolumeManager) GetVolumeUsage(volumeName string) []string {
	return vm.volumeUsage[volumeName]
}

// GetServiceVolumes returns all volumes for a specific service
func (vm *VolumeManager) GetServiceVolumes(serviceName string) []string {
	var volumes []string
	
	// Check named volumes
	for name, services := range vm.volumeUsage {
		for _, svc := range services {
			if svc == serviceName {
				volumes = append(volumes, name)
				break
			}
		}
	}
	
	// Check bind mounts
	for _, mount := range vm.bindMounts {
		if mount.Service == serviceName {
			spec := fmt.Sprintf("%s:%s", mount.Source, mount.Target)
			if mount.ReadOnly {
				spec += ":ro"
			}
			volumes = append(volumes, spec)
		}
	}
	
	return volumes
}

// ValidateVolumes validates all volumes
func (vm *VolumeManager) ValidateVolumes() error {
	// Check for conflicts
	for key, mount := range vm.bindMounts {
		// Check if source path exists (for relative paths)
		if !filepath.IsAbs(mount.Source) && !strings.HasPrefix(mount.Source, "..") {
			// Could check if path exists
		}
		
		// Check for duplicate mount targets in the same service
		parts := strings.Split(key, ":")
		if len(parts) != 2 {
			continue
		}
		serviceName := parts[0]
		targetPath := parts[1]
		
		// Count how many mounts to the same target
		count := 0
		for _, m := range vm.bindMounts {
			if m.Service == serviceName && m.Target == targetPath {
				count++
				if count > 1 {
					return fmt.Errorf("service '%s' has multiple mounts to '%s'", 
						serviceName, targetPath)
				}
			}
		}
	}
	
	return nil
}

// GenerateDockerVolumes generates Docker volume definitions
func (vm *VolumeManager) GenerateDockerVolumes() map[string]DockerVolume {
	volumes := make(map[string]DockerVolume)
	
	for name, config := range vm.namedVolumes {
		volume := DockerVolume{
			Driver: config.Driver,
		}
		
		// Add driver options if present
		if len(config.DriverOpts) > 0 {
			// Note: DockerVolume struct would need to be extended to support driver_opts
		}
		
		// Add labels if present
		if len(config.Labels) > 0 {
			// Note: DockerVolume struct would need to be extended to support labels
		}
		
		volumes[name] = volume
	}
	
	return volumes
}

// GetVolumeSpecs returns volume specifications for a service
func (vm *VolumeManager) GetVolumeSpecs(serviceName string) []string {
	var specs []string
	
	// Add named volumes
	for volumeName, services := range vm.volumeUsage {
		for _, svc := range services {
			if svc == serviceName {
				// For named volumes, we need to know the mount point
				// This would typically come from service configuration
				// For now, use a standard mount point
				specs = append(specs, fmt.Sprintf("%s:/data", volumeName))
				break
			}
		}
	}
	
	// Add bind mounts
	for _, mount := range vm.bindMounts {
		if mount.Service == serviceName {
			spec := fmt.Sprintf("%s:%s", mount.Source, mount.Target)
			if mount.ReadOnly {
				spec += ":ro"
			}
			specs = append(specs, spec)
		}
	}
	
	return specs
}

// Clear resets the volume manager
func (vm *VolumeManager) Clear() {
	vm.namedVolumes = make(map[string]VolumeConfig)
	vm.bindMounts = make(map[string]BindMount)
	vm.volumeUsage = make(map[string][]string)
}

// HasNamedVolume checks if a named volume exists
func (vm *VolumeManager) HasNamedVolume(name string) bool {
	_, exists := vm.namedVolumes[name]
	return exists
}

// RemoveVolume removes a volume from tracking
func (vm *VolumeManager) RemoveVolume(volumeName string, serviceName string) {
	// Remove from usage tracking
	if services, exists := vm.volumeUsage[volumeName]; exists {
		var updated []string
		for _, svc := range services {
			if svc != serviceName {
				updated = append(updated, svc)
			}
		}
		
		if len(updated) == 0 {
			// No more services using this volume
			delete(vm.volumeUsage, volumeName)
			delete(vm.namedVolumes, volumeName)
		} else {
			vm.volumeUsage[volumeName] = updated
		}
	}
	
	// Remove bind mounts for the service
	for key, mount := range vm.bindMounts {
		if mount.Service == serviceName {
			delete(vm.bindMounts, key)
		}
	}
}

// GetUnusedVolumes returns volumes not used by any service
func (vm *VolumeManager) GetUnusedVolumes() []string {
	var unused []string
	
	for name := range vm.namedVolumes {
		if len(vm.volumeUsage[name]) == 0 {
			unused = append(unused, name)
		}
	}
	
	return unused
}

// OptimizeVolumes removes unused volumes
func (vm *VolumeManager) OptimizeVolumes() []string {
	removed := vm.GetUnusedVolumes()
	
	for _, name := range removed {
		delete(vm.namedVolumes, name)
		delete(vm.volumeUsage, name)
	}
	
	return removed
}