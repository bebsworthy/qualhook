# Implementation Plan: Quality Hook

## Overview
This implementation plan breaks down the Quality Hook CLI tool into atomic, executable tasks. Tasks are organized to maximize parallel execution opportunities, with clear dependencies marked.

## Parallel Execution Strategy

Tasks are grouped into tracks that can be developed concurrently:
- **Track A**: Core Types & Configuration (Tasks 2, 4-6)
- **Track B**: CLI Framework (Tasks 9-10, 12)
- **Track C**: Project Detection (Tasks 7-8)
- **Track D**: Execution Engine (Tasks 13-15)
- **Track E**: Filtering & Output (Tasks 19-21)
- **Track F**: Testing & Documentation (Tasks 26-28, 31-32)

## Tasks

### Phase 1: Project Setup and Core Types

- [x] 1. **Initialize Go project structure**
  - Create directory structure as per design
  - Initialize go.mod with module path
  - Add .gitignore for Go projects
  - Create README.md with project overview
  - _Requirements: General setup_

- [x] 2. **Define core configuration types**
  - Create `pkg/types/config.go` with all configuration structs
  - Define Config, CommandConfig, PathConfig, FilterConfig types
  - Add JSON tags for serialization
  - Include validation methods on types
  - _Requirements: 3.1, 3.2_

- [x] 3. **Set up development tooling**
  - Add Makefile with build, test, lint targets
  - Configure golangci-lint for code quality
  - Set up GitHub Actions for CI (optional)
  - Add pre-commit hooks for formatting
  - _Requirements: General quality_

### Phase 2: Configuration System

- [x] 4. **Implement configuration loader**
  - Create `internal/config/loader.go`
  - Implement JSON file loading from standard paths
  - Support environment variable for config path
  - Add configuration merging for monorepo paths
  - Write unit tests for loader
  - _Requirements: 3.1, 3.3, 3.4_

- [x] 5. **Implement configuration validator**
  - Create `internal/config/validator.go`
  - Validate required fields in configuration
  - Check regex pattern validity
  - Validate command existence in PATH
  - Write unit tests for validation logic
  - _Requirements: 3.4, 11.2_

- [x] 6. **Create default configuration templates**
  - Create `internal/config/defaults.go`
  - Define default configs for Node.js, Go, Python, Rust
  - Include common error patterns for each language
  - Store as embedded resources
  - _Requirements: 3.2, 4.2_

### Phase 3: Project Detection

- [x] 7. **Implement project detector**
  - Create `internal/detector/project.go`
  - Scan for marker files (package.json, go.mod, etc.)
  - Calculate confidence scores based on files found
  - Support detecting multiple project types
  - Write unit tests with sample project structures
  - _Requirements: 4.1, 4.3, 4.5_

- [x] 8. **Add monorepo detection logic**
  - Extend detector to identify monorepo patterns
  - Detect workspace files (lerna.json, nx.json, etc.)
  - Map subdirectories to project types
  - Handle nested project structures
  - _Requirements: 5.4_

### Phase 4: CLI Foundation

- [x] 9. **Set up Cobra CLI framework**
  - Create `cmd/qualhook/main.go`
  - Initialize Cobra application
  - Define root command with global flags (--debug, --config)
  - Set up command hierarchy
  - _Requirements: 1.1-1.5_

- [x] 10. **Implement core quality commands**
  - Add format, lint, typecheck, test subcommands
  - Create command handler structure
  - Support custom commands from config
  - Add --help documentation for each command
  - _Requirements: 1.1-1.5_

- [x] 11. **Implement config command**
  - Create `qualhook config` subcommand
  - Add interactive configuration wizard
  - Implement project detection integration
  - Allow manual configuration option
  - Write config file to appropriate location
  - _Requirements: 8.1, 4.2, 4.4_

- [x] 12. **Add config validation command**
  - Implement `qualhook config --validate`
  - Check current configuration validity
  - Display detailed validation errors
  - Suggest fixes for common issues
  - _Requirements: 8.2_

### Phase 5: Command Execution

- [x] 13. **Implement command executor**
  - Create `internal/executor/command.go`
  - Implement safe subprocess execution using exec.Command
  - Handle environment variables and working directory
  - Capture stdout/stderr streams
  - Add timeout support with context
  - _Requirements: 1.6, 10.4, 11.1_

- [x] 14. **Add execution error handling**
  - Distinguish between different error types
  - Handle command not found errors
  - Manage permission denied scenarios
  - Implement timeout killing and cleanup
  - Write unit tests with mocked commands
  - _Requirements: 9.2, 9.4_

- [x] 15. **Implement parallel execution for monorepos**
  - Create `internal/executor/parallel.go`
  - Use goroutines for concurrent execution
  - Implement result aggregation
  - Add progress reporting for multiple commands
  - Handle partial failures gracefully
  - _Requirements: 5.2, 10.5_

### Phase 6: File-Aware Execution

- [x] 16. **Parse Claude Code hook input**
  - Create `internal/hook/parser.go`
  - Parse JSON input from Claude Code
  - Extract edited file paths from tool_use
  - Handle different tool types (Edit, Write, etc.)
  - _Requirements: 2.4, 6.3_

- [x] 17. **Implement file-to-component mapping**
  - Create `internal/watcher/mapper.go`
  - Map file paths to configuration paths
  - Apply glob pattern matching
  - Implement precedence rules for overlapping paths
  - Group files by component
  - _Requirements: 6.1, 6.2, 6.3_

- [x] 18. **Add file-aware command execution**
  - Integrate file mapping with command execution
  - Execute appropriate commands per component
  - Skip components with no changed files
  - Report which checks ran for which files
  - _Requirements: 6.1, 6.5_

### Phase 7: Output Filtering

- [x] 19. **Implement output filter engine**
  - Create `internal/filter/output.go`
  - Apply regex patterns to command output
  - Extract matching lines with context
  - Support both stdout and stderr filtering
  - Handle large outputs with streaming
  - _Requirements: 7.1, 7.2, 10.2_

- [x] 20. **Add intelligent truncation**
  - Implement output size limits
  - Preserve error information during truncation
  - Add priority-based filtering (errors > warnings)
  - Include truncation indicators
  - _Requirements: 7.3, 7.4_

- [x] 21. **Create pattern management system**
  - Create `internal/filter/patterns.go`
  - Compile and cache regex patterns
  - Provide pattern validation
  - Add performance optimizations
  - Support pattern testing mode
  - _Requirements: 7.1, 7.4_

### Phase 8: Error Reporting

- [x] 22. **Implement error reporter**
  - Create `internal/reporter/error.go`
  - Format errors for LLM consumption
  - Add configurable prompt prefixes
  - Determine appropriate exit codes
  - Aggregate errors from multiple components
  - _Requirements: 2.1, 2.2, 2.5_

- [x] 23. **Add debug mode output**
  - Implement verbose logging for --debug flag
  - Show full command execution details
  - Display pattern matching process
  - Include timing information
  - _Requirements: 9.3_

### Phase 9: Configuration Management

- [x] 24. **Build configuration wizard**
  - Create interactive prompts using survey library
  - Guide through project type selection
  - Allow customization of default configs
  - Validate user inputs
  - Save configuration with proper formatting
  - _Requirements: 8.1, 3.3_

- [x] 25. **Add import/export functionality**
  - Implement config template export
  - Support importing shared configurations
  - Add config merging capabilities
  - Version configuration schemas
  - _Requirements: 8.3_

### Phase 10: Testing and Integration

- [ ] 26. **Create comprehensive test suite**
  - Write unit tests for all components
  - Add integration tests for end-to-end flows
  - Create test fixtures for different project types
  - Mock external command execution
  - Achieve >80% code coverage
  - _Requirements: General quality_

- [ ] 27. **Add performance benchmarks**
  - Benchmark regex pattern matching
  - Test startup time overhead
  - Measure memory usage with large outputs
  - Optimize hot paths
  - _Requirements: 10.1, 10.2_

- [ ] 28. **Create example configurations**
  - Build examples for common project types
  - Add monorepo configuration examples
  - Include custom command examples
  - Document configuration options
  - _Requirements: Documentation_

### Phase 11: Security Hardening

- [ ] 29. **Implement security validations**
  - Add command whitelisting
  - Prevent path traversal attacks
  - Validate all file paths
  - Sanitize regex patterns
  - Limit resource consumption
  - _Requirements: 11.1-11.5_

- [ ] 30. **Add security tests**
  - Test command injection prevention
  - Verify path traversal protection
  - Check environment variable filtering
  - Test configuration validation
  - _Requirements: 11.1-11.5_

### Phase 12: Documentation and Polish

- [ ] 31. **Write comprehensive documentation**
  - Create user guide with examples
  - Document configuration schema
  - Add troubleshooting guide
  - Include Claude Code integration guide
  - _Requirements: Documentation_

- [ ] 32. **Add CLI help and examples**
  - Write detailed help for each command
  - Include usage examples in help text
  - Add man page generation
  - Create shell completion scripts
  - _Requirements: Usability_

---

## Task Dependencies & Parallel Execution

### Prerequisites (Must Complete First)
- Task 1: Project initialization (blocks all)
- Task 2: Core types (blocks 4-6, 9-10, 13-15, 19-21)
- Task 3: Development tooling (optional, but recommended early)

### Parallel Execution Groups

**After Prerequisites, these can run in parallel:**

**Group 1 (Independent Tracks)**
- Track A: Tasks 4-6 (Configuration System)
- Track B: Tasks 7-8 (Project Detection)  
- Track C: Tasks 9-10 (CLI Foundation)
- Track D: Tasks 19-21 (Output Filtering)

**Group 2 (Depends on Group 1)**
- Track E: Tasks 11-12 (Config Commands) - needs 4-6, 7-8
- Track F: Tasks 13-15 (Command Execution) - needs 9-10
- Track G: Task 22 (Error Reporter) - needs 19-21
- Track H: Task 16 (Hook Parser) - independent

**Group 3 (Integration)**
- Tasks 17-18 (File-Aware) - needs 16 and 13-15
- Task 23 (Debug Mode) - needs 22
- Task 24-25 (Config Management) - needs 11-12

**Group 4 (Quality & Polish)**
- Tasks 26-28 (Testing) - can start anytime after Group 1
- Tasks 29-30 (Security) - can start anytime after Group 2
- Tasks 31-32 (Documentation) - can start anytime

### Critical Path
The longest dependency chain is:
1 → 2 → 9-10 → 13-15 → 17-18

This means the minimum time with perfect parallelization is about 6-7 days instead of 15 sequential days.

## Parallel Execution Visualization

```
Day 1:  [Task 1] → [Task 2, Task 3]
Day 2:  [Task 4-6] | [Task 7-8] | [Task 9-10] | [Task 19-21]
Day 3:  [Continue above] | [Task 16] | [Task 26-28 start]
Day 4:  [Task 11-12] | [Task 13-15] | [Task 22] | [Task 31-32 start]
Day 5:  [Task 17-18] | [Task 23] | [Task 24-25] | [Task 29-30]
Day 6:  [Integration & Testing]
Day 7:  [Final Polish & Documentation]
```

## Estimated Timeline

### Sequential Approach: ~15 days
### Parallel Approach (Team of 4): ~6-7 days
### Parallel Approach (Team of 2): ~8-10 days

### Optimal Team Assignment:
- **Developer 1**: Core types, Configuration system, Config commands
- **Developer 2**: CLI framework, Command execution, Integration
- **Developer 3**: Project detection, Output filtering, Error reporting
- **Developer 4**: Testing, Security, Documentation (can start early)

---

Do the tasks look good?