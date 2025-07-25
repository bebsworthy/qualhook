# Implementation Plan: Configuration Simplification

## Overview
This implementation plan breaks down the configuration structure simplification into atomic, executable tasks. The goal is to remove dead code (`errorDetection.patterns`) and flatten the configuration structure by eliminating unnecessary nesting.

## Parallel Execution Strategy

Tasks are grouped into tracks that can be developed concurrently:
- **Track A**: Schema Updates (Tasks 1-3)
- **Track B**: Code Refactoring (Tasks 4-7)
- **Track C**: Configuration Updates (Tasks 8-10)
- **Track D**: Testing & Validation (Tasks 11-13)

## Tasks

### Phase 1: Configuration Schema Updates

- [x] 1. **Update core configuration types**
  - Remove `ErrorDetection` struct from `pkg/config/config.go`
  - Remove `FilterConfig` struct from `pkg/config/config.go`
  - Add fields directly to `CommandConfig`:
    - `ExitCodes []int`
    - `ErrorPatterns []*RegexPattern`
    - `ContextLines int`
    - `MaxOutput int`
    - `IncludePatterns []*RegexPattern`
  - Update JSON tags for new structure
  - _Breaking change: No migration needed (no current users)_

- [x] 2. **Update configuration validation**
  - Modify `CommandConfig.Validate()` in `pkg/config/config.go`
  - Remove `ErrorDetection.Validate()` method
  - Remove `FilterConfig.Validate()` method
  - Update validation to check fields directly on CommandConfig
  - Ensure pattern validation still works
  - _Maintains same validation rules, just different structure_

- [x] 3. **Update configuration cloning methods**
  - Update `CommandConfig.Clone()` in `pkg/config/config.go`
  - Remove `ErrorDetection.Clone()` method
  - Remove `FilterConfig.Clone()` method
  - Ensure all new fields are properly cloned
  - _Required for configuration merging in monorepos_

### Phase 2: Code Refactoring

- [x] 4. **Update error detection logic**
  - Modify `hasErrors()` in `internal/reporter/error.go`
  - Change from `result.CommandConfig.ErrorDetection.ExitCodes`
    to `result.CommandConfig.ExitCodes`
  - Remove checks for nil ErrorDetection
  - Simplify the error checking logic
  - _Core functionality remains the same_

- [x] 5. **Update output filtering logic**
  - Modify `applyOutputFilter()` in `cmd/qualhook/execute.go`
  - Create FilterRules from CommandConfig fields directly
  - Remove references to `cmdConfig.OutputFilter`
  - Update to use:
    - `cmdConfig.ErrorPatterns`
    - `cmdConfig.IncludePatterns`
    - `cmdConfig.MaxOutput`
    - `cmdConfig.ContextLines`
  - _No change in filtering behavior_

- [x] 6. **Update configuration validator**
  - Modify validation logic in `internal/config/validator.go`
  - Remove validation for nested ErrorDetection
  - Remove validation for nested FilterConfig
  - Add direct validation of patterns on CommandConfig
  - Update error messages for new structure
  - _Same validation rules, flatter structure_

- [x] 7. **Update template cloning**
  - Modify `cloneCommandConfig()` in `internal/config/templates.go`
  - Remove ErrorDetection cloning logic
  - Remove FilterConfig cloning logic
  - Clone fields directly on CommandConfig
  - _Required for template system_

### Phase 3: Configuration File Updates

- [x] 8. **Update default configuration templates**
  - Modify all templates in `internal/config/defaults.go`
  - Convert from nested to flat structure for:
    - Go project defaults
    - Node.js project defaults
    - Python project defaults
    - Rust project defaults
  - Remove `errorDetection` and `outputFilter` nesting
  - Move all fields up one level
  - _Affects all new project configurations_

- [x] 9. **Update example configuration**
  - Modify `.qualhook.json` to use new flat structure
  - Update all command configurations:
    - format
    - lint
    - test
    - typecheck
    - vet
  - Ensure examples are clear and well-documented
  - _Primary reference for users_

- [x] 10. **Update test fixtures**
  - Find all test configuration files
  - Update to new flat structure
  - Remove nested errorDetection/outputFilter
  - Ensure test data is consistent
  - _Required for tests to pass_

### Phase 4: Testing and Validation

- [x] 11. **Update unit tests**
  - Fix tests in `internal/config/validator_test.go`
  - Fix tests in `internal/reporter/error_test.go`
  - Fix tests in `cmd/qualhook/execute_test.go`
  - Update any test that references old structure
  - Add tests for flat structure validation
  - _Ensure no regression_

- [x] 12. **Update integration tests**
  - Fix integration tests that use configuration
  - Update test expectations for new structure
  - Verify end-to-end functionality unchanged
  - Test with various project types
  - _Validate complete system works_

- [ ] 13. **Manual testing and verification**
  - Test `qualhook config` wizard with new structure
  - Verify all commands work with updated config
  - Test monorepo configurations
  - Check error messages are still clear
  - Validate performance is unchanged
  - _Final quality check_

### Phase 5: Documentation Updates

- [x] 14. **Update design documentation**
  - Modify `documentation/features/quality-hook/design.md`
  - Update Configuration Schema section (lines 254-295)
  - Remove `ErrorDetection` interface definition
  - Remove `FilterConfig` interface definition
  - Update `CommandConfig` interface to show flat structure
  - Update all example configurations in the document
  - _Critical for understanding the new structure_

- [x] 15. **Update API documentation**
  - Search for any API docs that reference the old structure
  - Update interface definitions
  - Update example requests/responses
  - Ensure consistency across all docs
  - _Required for proper usage_

- [x] 16. **Update README and getting started guides**
  - Update main README.md configuration examples
  - Update any quickstart guides
  - Update configuration reference documentation
  - Add migration notes for the breaking change
  - _First thing users will see_

---

## Task Dependencies & Parallel Execution

### Prerequisites (Must Complete First)
- Task 1: Update core types (blocks all other tasks)

### Parallel Execution Groups

**After Task 1, these can run in parallel:**

**Group 1 (Schema completion)**
- Task 2-3: Validation and cloning updates

**Group 2 (Code updates - can start after Task 1)**
- Track A: Tasks 4-7 (Code refactoring)
- Track B: Tasks 8-10 (Configuration updates)
- Track C: Tasks 14-16 (Documentation updates)

**Group 3 (Testing - after Groups 1 & 2)**
- Tasks 11-13 (Testing and validation)

### Critical Path
The longest dependency chain is:
Task 1 → Tasks 2-7 (parallel) → Tasks 11-13

### Execution Timeline

**Day 1:**
- Morning: Task 1 (Update core types)
- Afternoon: Start Tasks 2-3, 4-7, and 14-16 in parallel

**Day 2:**
- Continue Tasks 4-7
- Start Tasks 8-10
- Complete documentation updates (14-16)
- Begin updating tests (Task 11)

**Day 3:**
- Complete all code changes
- Focus on testing (Tasks 11-13)
- Final validation and cleanup

## Estimated Timeline

### Sequential Approach: ~6-7 days
### Parallel Approach (Team of 4): ~2-3 days
### Single Developer: ~4 days

### Optimal Team Assignment:
- **Developer 1**: Core schema updates (Tasks 1-3), then help with testing
- **Developer 2**: Code refactoring (Tasks 4-7)
- **Developer 3**: Configuration updates (Tasks 8-10), then testing
- **Developer 4**: Documentation updates (Tasks 14-16), then assist with testing

## Success Criteria

1. All tests pass with new structure
2. Configuration files are simpler and clearer
3. No change in runtime behavior
4. Dead code completely removed
5. Documentation updated to reflect changes

## Risk Mitigation

1. **Breaking Change**: Clearly document in release notes
2. **Test Coverage**: Ensure all paths tested before merging
3. **Rollback Plan**: Tag version before changes
4. **Review Process**: Multiple reviewers for schema changes