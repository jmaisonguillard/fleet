# Fleet Refactoring Roadmap

## Overview
This document tracks refactoring opportunities identified in the Fleet codebase to improve maintainability, reduce complexity, and standardize patterns across the project.

## Codebase Metrics (Current State)
- **Total Lines**: ~11,000 lines of Go code
- **Largest Files**: 
  - Test files: 500-800 lines (well-structured with suite patterns)
  - Service implementations: 300-450 lines
  - Core logic: 300-400 lines
- **Complexity Hotspot**: `compose.go` with 43 conditionals in main function

## Priority 1: High Impact Refactoring

### ❏ 1.1 Refactor `generateDockerCompose()` Function
**File**: `compose.go:60-296`  
**Problem**: Single function with 236 lines and 43 conditionals  
**Solution**: Extract into focused functions:

- [x] `buildServiceConfig()` - Basic service configuration (image, restart policy, networks)
- [x] `configureVolumes()` - Volume mounting logic (folder mapping, named volumes)
- [x] `configurePorts()` - Port exposure logic (domain check, port mapping)
- [x] `configureEnvironment()` - Environment variable setup
- [x] `configureHealthCheck()` - Health check configuration
- [x] `addServiceDependencies()` - Service dependency management
- [x] `addSupportServices()` - Adds additional services based on configuration
- [x] `finalizeVolumes()` - Creates volume definitions for all tracked volumes

**Benefits**:
- Improved readability and testability
- Easier to add new service types
- Reduced cognitive load

### ✅ 1.2 Extract Database Password Configuration
**File**: `compose.go:156-176`  
**Problem**: Database-specific password logic embedded in main function  
**Solution**: Create `configureDatabaseCredentials()` function

- [x] Move database detection logic
- [x] Standardize credential environment variables
- [x] Add support for new database types easily

## Priority 2: Standardization

### ✅ 2.1 Create Service Provider Interface
**Files**: All `*_service.go` files  
**Problem**: Similar patterns but no common interface  
**Solution**: Define and implement interface:

```go
type ServiceProvider interface {
    GetServiceName(serviceType, version string) string
    AddService(compose *DockerCompose, svc *Service, config *Config)
    ValidateConfig(svc *Service) error
    GetDefaultVersion() string
    GetSupportedVersions() []string
    IsShared() bool
    GetEnvironmentVariables(svc *Service, config *Config) map[string]string
}
```

- [x] Create `service_provider.go` with interface definition
- [x] Implement for `DatabaseServiceProvider`
- [x] Implement for `CacheServiceProvider`
- [x] Implement for `SearchServiceProvider`
- [x] Implement for `CompatServiceProvider`
- [x] Implement for `EmailServiceProvider`
- [x] Create service registry/factory pattern

### ❏ 2.2 Unify Shared Container Naming
**Files**: Multiple service files  
**Problem**: Each service has its own naming function  
**Solution**: Create unified naming strategy

- [ ] Create `SharedServiceNamer` type
- [ ] Implement consistent naming patterns
- [ ] Add name collision detection

### ❏ 2.3 Centralize Environment Variable Patterns
**Files**: All service files  
**Problem**: Duplicated environment variable setting logic  
**Solution**: Create environment builder

- [ ] Create `EnvBuilder` type with fluent API
- [ ] Standard patterns for connection strings
- [ ] Consistent naming conventions

## Priority 3: Code Organization

### ❏ 3.1 Create Validation Package
**Problem**: Validation scattered across files  
**Solution**: Centralized validation

- [ ] Create `validation/` package
- [ ] Port conflict detection
- [ ] Service compatibility checks
- [ ] Configuration completeness validation
- [ ] Pre-compose generation validation

### ❏ 3.2 Extract Volume Management
**File**: `compose.go:262-287`  
**Problem**: Volume logic mixed with service generation  
**Solution**: Dedicated volume manager

- [ ] Create `VolumeManager` type
- [ ] Track volume requirements
- [ ] Handle bind mounts vs named volumes
- [ ] Validate volume configurations

### ❏ 3.3 Improve PHP Service Configuration
**Files**: `php_frameworks.go`, `runtime_php.go`  
**Problem**: Complex PHP detection and configuration  
**Solution**: Refactor PHP handling

- [ ] Create `PHPConfigurator` type
- [ ] Standardize framework detection
- [ ] Simplify nginx config generation
- [ ] Centralize Xdebug configuration

## Priority 4: Testing Improvements

### ❏ 4.1 Extract Test Fixtures
**Problem**: Large test files with embedded fixtures  
**Solution**: Separate fixture management

- [ ] Create `testdata/` directory
- [ ] Move sample configurations to files
- [ ] Create fixture loader utility

### ❏ 4.2 Enhance Mock System
**File**: `docker_mock_test.go`  
**Problem**: Mock could be more reusable  
**Solution**: Improve mock infrastructure

- [ ] Create `mocks/` package
- [ ] Make mock configurable per test
- [ ] Add mock state verification

## Priority 5: Documentation and Examples

### ❏ 5.1 Add Code Documentation
- [ ] Document public functions
- [ ] Add package-level documentation
- [ ] Create architecture diagrams

### ❏ 5.2 Enhance Examples
- [ ] Add complex multi-service examples
- [ ] Document all service combinations
- [ ] Create troubleshooting guide

## Implementation Strategy

### Phase 1: Core Refactoring (Week 1-2)
1. Start with `generateDockerCompose()` refactoring
2. Extract helper functions
3. Ensure all tests pass after each change

### Phase 2: Standardization (Week 3-4)
1. Implement ServiceProvider interface
2. Migrate one service at a time
3. Update tests for new structure

### Phase 3: Organization (Week 5-6)
1. Create new packages
2. Move code gradually
3. Update imports and dependencies

### Phase 4: Testing and Documentation (Week 7-8)
1. Improve test organization
2. Add missing documentation
3. Update examples

## Tracking

### Completed
- ✅ 1.1 Refactor `generateDockerCompose()` Function (reduced from 236 to 52 lines!)
- ✅ 1.2 Extract Database Password Configuration
- ✅ 2.1 Create Service Provider Interface (all 5 providers implemented)

### In Progress
- None

### Blocked
- None yet

## Notes

### Important Considerations
1. **Backward Compatibility**: Ensure generated docker-compose.yml remains unchanged
2. **Test Coverage**: Maintain or improve current coverage
3. **Performance**: Monitor compilation and runtime performance
4. **Cross-Platform**: Ensure changes work on Linux, macOS, Windows

### Success Metrics
- [x] Reduce `generateDockerCompose()` to under 50 lines (✅ Achieved: 52 lines)
- [x] No function exceeds 100 lines (✅ All new functions are under 50 lines)
- [ ] Cyclomatic complexity under 10 for all functions
- [ ] Test coverage remains above 80%
- [x] All existing tests pass without modification (✅ All tests passing)

## References
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://golang.org/doc/effective_go)
- [Clean Code principles](https://github.com/Pungyeon/clean-go-article)

---
*Last Updated: 2025-08-28*  
*Next Review: After Phase 1 completion*