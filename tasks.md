# Linting Issue Resolution Tasks - Comprehensive Plan

## Current Status
- **Initial Issues**: 78
- **Fixed Issues**: 52 (Phases 1-4, 5-7, 9 completed; Phase 8 in progress)
- **Remaining Issues**: 26
- **Last Updated**: 2025-07-23

## Issue Breakdown

### Summary by Category
1. **Code Duplication** (dupl): 4 issues - Command files with identical structure
2. **Error Checking** (errcheck): 26 issues - Mix of false positives and genuine unchecked errors
3. **Constants** (goconst): 6 issues - Repeated string literals
4. **Code Style** (gocritic): 4 issues - If-else chains that should be switches
5. **Complexity** (gocyclo): 8 issues - Functions exceeding cyclomatic complexity of 15
6. **Static Analysis** (staticcheck): 6 issues - Deprecated APIs, nil checks, performance

## Completed Phases

### ✅ Phase 1: Quick Fixes (Completed)
- [x] 1. Fixed spelling errors (5 instances of "cancelled" → "canceled")
- [x] 2. Fixed package comment format in test_helpers.go
- [x] 3. Removed unused progressChan field and progressUpdate type
- [x] 4. Fixed ineffectual assignment in main.go
- [x] 5. Fixed unparam issue in scanWorkspaces function

### ✅ Phase 2: Security Fixes (Completed)
- [x] 6. Fixed directory permissions (0755 → 0750) in 3 locations
- [x] 7. Fixed file permissions (0644 → 0600) in 3 locations
- [x] 8. Added #nosec comments for validated file paths (5 locations)

### ✅ Phase 3: Error Handling (Completed)
- [x] 9. Added error checks for UI operations
- [x] 10. Added error checks for completion generation
- [x] 11. Added error checks for file operations
- [x] 12. Fixed command error handling
- [x] 13. Fixed remaining error checks

### ✅ Phase 4: Code Duplication Refactoring (Completed)

- [x] 14. **Create base command structure**
  - Extracted common command logic into `cmd/qualhook/base_command.go`
  - Created `createQualityCommand` factory function
  - Implemented `createRunFunc` for common execution logic
  - _Successfully reduced ~300 lines of duplicate code_

- [x] 15. **Refactor format, lint, test, typecheck commands**
  - Updated all four commands to use the base structure
  - Maintained command-specific descriptions and examples
  - All commands tested and working correctly
  - _All duplication issues resolved_

### Phase 5: Error Checking Refinement (26 issues)

- [x] 16. **Analyze errcheck false positives**
  - Review why errcheck still reports issues for `_, _ = fmt.Fprintln`
  - Consider adding `//nolint:errcheck` directives where appropriate
  - Or configure errcheck to ignore specific patterns
  - _Effort: 30 minutes_

- [x] 17. **Fix genuine unchecked errors**
  - `internal/filter/optimizations.go:218` - Check bufferPool.Get() type assertion
  - `internal/filter/output.go:395` - Handle NewPatternCache error
  - `internal/filter/patterns.go:138,303` - Handle NewPatternCache errors
  - `internal/security/validator.go:330` - Check regexp.MatchString error
  - _Effort: 30 minutes_

- [x] 18. **Configure golangci-lint for error handling**
  - Add errcheck exclusions for UI output functions
  - Configure to ignore `_, _ =` pattern for specific functions
  - Update `.golangci.yml` with appropriate rules
  - _Effort: 20 minutes_

### Phase 6: Extract Constants (6 issues)

- [x] 19. **Create constants for test strings**
  - Extract "modified" (7 occurrences) to `testModifiedValue`
  - Extract "windows" (25 occurrences) to `osWindows`
  - Extract "echo" (7 occurrences) to `echoCommand`
  - Extract "cmd" (17 occurrences) to `cmdCommand`
  - Create appropriate const blocks in relevant packages
  - _Effort: 45 minutes_

### Phase 7: Code Style Improvements (4 issues)

- [x] 20. **Convert if-else chains to switch statements**
  - `internal/config/validator.go:347` - Error type detection
  - `internal/executor/errors.go:142` - Error classification
  - `internal/executor/parallel.go:225` - Result type handling
  - `internal/watcher/mapper_test.go:301` - Comparison logic
  - _Effort: 30 minutes_

### Phase 8: Reduce Cyclomatic Complexity (8 issues)

- [x] 21. **Refactor executeCommand (complexity: 26)**
  - File: `cmd/qualhook/execute.go:23`
  - Extracted `parseHookInput()`, `extractEditedFiles()`, `executeFileAwareCommand()`
  - Extracted `executeSingleCommand()`, `executeWithOptions()`, `applyOutputFilter()`
  - Extracted `reportAndOutputResults()` for result processing
  - _Successfully reduced complexity to <15_

- [x] 22. **Refactor ConfigWizard.Run (complexity: 33)**
  - File: `internal/wizard/config.go:37`
  - Extracted `detectProject()`, `displayDetectionResults()`, `createConfiguration()`
  - Extracted `handleMonorepoConfig()`, `validateAndSave()`, `printSuccess()`
  - Split validation and writing logic into separate methods
  - _Successfully reduced complexity to <15_

- [x] 23. **Refactor FileAwareExecutor.ExecuteForEditedFiles (complexity: 21)**
  - File: `internal/executor/file_aware.go:58`
  - Extracted `executeForRootComponent()` and `executeForComponents()`
  - Created debug logging helper methods
  - Simplified error aggregation
  - _Successfully reduced complexity to <15_

- [ ] 24. **Refactor SecurityValidator.ValidatePath (complexity: 21)**
  - File: `internal/security/validator.go:83`
  - Extract path checks into separate validation functions
  - Create path validation pipeline
  - _Effort: 1 hour_

- [ ] 25. **Refactor remaining complex functions**
  - `internal/config/validator.go:70` (complexity: 16)
  - `internal/executor/errors.go:98` (complexity: 16)
  - `internal/security/validator.go:254` (complexity: 19)
  - `internal/wizard/config.go:233` (complexity: 19)
  - _Effort: 2 hours total_

### Phase 9: Static Analysis Fixes (6 issues)

- [x] 26. **Fix nil pointer dereference**
  - File: `cmd/qualhook/commands_test.go:121-124`
  - Add proper nil check before accessing validateFlag
  - _Effort: 10 minutes_

- [x] 27. **Update deprecated cobra API**
  - File: `cmd/qualhook/completion.go:58`
  - Replace `cobra.ExactValidArgs(1)` with `cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs)`
  - _Effort: 10 minutes_

- [x] 28. **Remove empty branch**
  - File: `internal/config/loader.go:169`
  - Either implement the branch logic or remove the condition
  - _Effort: 15 minutes_

- [x] 29. **Fix memory allocation issue**
  - File: `internal/filter/optimizations.go:224`
  - Change buffer pool to use pointer type
  - Update Get/Put operations accordingly
  - _Effort: 30 minutes_

- [x] 30. **Fix unused append result**
  - File: `internal/filter/optimizations_bench_test.go:185`
  - Either use the append result or remove the operation
  - _Effort: 10 minutes_

## Implementation Strategy

### Priority Order
1. **Phase 9** (Static Analysis) - Quick fixes for potential bugs
2. **Phase 6** (Constants) - Easy wins for code quality
3. **Phase 7** (Code Style) - Simple refactoring
4. **Phase 5** (Error Checking) - Configure linter properly
5. **Phase 4** (Duplication) - Significant refactoring
6. **Phase 8** (Complexity) - Most time-consuming

### Parallel Execution Opportunities
- **Track A**: Phase 9 + Phase 6 (can be done immediately)
- **Track B**: Phase 7 + Phase 5 (independent changes)
- **Track C**: Phase 4 (requires careful design)
- **Track D**: Phase 8 (requires deep understanding)

### Testing Strategy
1. Run `make lint` after each phase completion
2. Run `make test` to ensure no regressions
3. Test individual commands manually
4. Verify Claude Code integration still works

## Success Metrics
- [ ] All 54 remaining issues resolved
- [ ] `make lint` passes with no errors
- [ ] No regression in functionality
- [ ] Code coverage maintained or improved
- [ ] Cyclomatic complexity reduced below 15 for all functions

## Long-term Improvements
1. **Pre-commit hooks**: Add linting to prevent future issues
2. **CI enforcement**: Make lint checks required for PR merges
3. **Code review guidelines**: Document patterns to avoid
4. **Refactoring guidelines**: Create standards for function complexity

## Notes
- Some errcheck issues appear to be false positives from the linter
- Consider upgrading golangci-lint version if false positives persist
- The duplication in command files is the most significant technical debt
- Complexity reduction will improve maintainability significantly