# Research: quality-hook

## 1. Feature Context

### Description
A configurable command-line utility that serves as Claude Code hooks to enforce code quality for LLM coding agents. The tool wraps project-specific commands (format, lint, typecheck, test) and intelligently filters their output to provide relevant error information to the LLM. Supports monorepos with multiple technologies.

### Scope
- **Layer**: CLI Tool/Backend
- **Components**: Command wrapper, configuration system, output filter, project detector, monorepo handler

### Architecture Reference
*Note: No architecture.md exists yet. This is a greenfield project.*

## 2. Similar Existing Features

### Feature: Claude Code Hooks System
- **Location**: Claude Code built-in functionality
- **What it does**: Executes user-defined shell commands at lifecycle points
- **What we can reuse**:
  - Exit code 2 pattern for blocking errors
  - JSON output format for advanced communication
  - PostToolUse hook type for quality checks
- **What to improve**:
  - Add intelligent output filtering
  - Provide configuration-driven behavior

### Feature: Multi-language build tools (e.g., Bazel, Make)
- **Location**: Industry standard tools
- **Patterns to follow**: Configuration-driven command execution
- **Lessons learned**: Need flexible configuration for diverse project types

### Feature: Monorepo tools (e.g., Nx, Lerna, Rush)
- **Location**: Monorepo management tools
- **What we can reuse**:
  - Project detection patterns
  - Per-directory configuration
  - Workspace-aware command execution
- **Lessons learned**: Need path-based configuration for different parts of monorepo

## 3. Affected Components

### Direct Impact
| Component | Location | Purpose | Changes Needed |
|:----------|:---------|:--------|:---------------|
| CLI Application | `qualhook` | Main executable | Create from scratch |
| Configuration System | `~/.qualhook/config.yaml` | Store project configs | Design flexible schema with path-based rules |
| Project Detector | Internal module | Auto-detect project type | Implement heuristics for monorepos |
| Output Filter | Internal module | Extract relevant errors | Pattern-based filtering |
| Monorepo Handler | Internal module | Manage multiple project types | Path-based configuration mapping |

### Integration Points
| System | Type | Current State | Integration Needs |
|:-------|:-----|:--------------|:------------------|
| Claude Code | Shell Hook | Receives JSON input | Parse input, return appropriate exit codes |
| Project Tools | CLI Commands | Various (npm, go, cargo, etc.) | Spawn subprocesses, capture output |
| File System | Config Files | N/A | Read/write YAML/JSON configs |

## 4. Technical Considerations

### New Dependencies
- [x] New dependencies required:
  - **CLI Framework**: (e.g., Cobra for Go, Clap for Rust) - command parsing
  - **YAML/JSON Parser**: Configuration file handling
  - **Regex Engine**: Pattern matching for output filtering
  - **Process Spawning**: Execute external commands

### Performance Impact
- **Expected Load**: Lightweight wrapper, minimal overhead
- **Performance Concerns**: Output parsing on large outputs
- **Optimization Needs**: Stream processing for large outputs

### Security Considerations
- **Authentication**: Not required
- **Authorization**: File system permissions for config
- **Data Sensitivity**: May process source code errors
- **Vulnerabilities**: Command injection risks in config

## 5. Implementation Constraints

### Must Follow (from Claude Code hooks)
- Exit code 2 for blocking errors with stderr feedback
- JSON input parsing from Claude Code
- PostToolUse hook pattern for quality checks

### Cannot Change
- Claude Code hook interface
- Project-specific tool behaviors
- Exit code conventions

### Technical Debt in Area
- N/A (new project)

## 6. Recommendations

### Architecture Approach
Based on the analysis:
- **Pattern**: Configuration-driven plugin architecture
- **Structure**: Core engine + project type configs
- **Integration**: Subprocess execution with output capture

### Implementation Strategy
1. **Start with**: Core CLI framework and config schema
2. **Build on**: Basic command execution and output capture
3. **Reuse**: Standard patterns for CLI tools
4. **Avoid**: Hardcoding project-specific logic

### Testing Approach
- **Unit Tests**: Mock subprocess execution
- **Integration Tests**: Test with real project tools
- **Test Data**: Sample outputs from various linters/tools

## 7. Risks and Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|:-----|:------------|:-------|:-----------|
| Complex regex patterns | High | Medium | Provide good defaults, allow customization |
| Tool output changes | Medium | High | Version-aware configurations |
| Performance on large outputs | Low | Medium | Stream processing, output limits |
| Monorepo complexity | High | High | Path-based configuration, clear precedence rules |

### Integration Risks
- **Breaking Changes**: Tool output format changes
- **Data Migration**: Config schema evolution
- **Rollback Plan**: Version configs, support legacy formats

## 8. Next Steps

### Requirements Considerations
Based on this research, the requirements should:
- Include specific hook behavior for each command type
- Consider project detection heuristics for monorepos
- Account for extensibility via configuration
- Define path-based configuration precedence for monorepos

### Design Considerations
The design phase should focus on:
- Configuration schema flexibility
- Output filtering algorithm
- Interactive configuration wizard

---

## Research Summary

**Key Findings**:
1. Claude Code uses exit code 2 to block actions and feed stderr back to LLM
2. Configuration-driven architecture enables support for any project type
3. Output filtering is critical to avoid overwhelming the LLM with verbose output
4. Monorepo support requires path-based configuration with clear precedence rules

**Recommended Approach**: Build a configuration-driven CLI wrapper that delegates to project tools and intelligently filters output for LLM consumption, with path-based rules for monorepo support.

**Proceed to Requirements?** Yes