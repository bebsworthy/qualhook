# Linting Issue Resolution Tasks

## Overview
This document outlines the tasks required to resolve all 78 linting issues identified in the qualhook codebase. The issues span multiple categories including code duplication, error handling, security, complexity, and code quality.

## Issue Summary
- **Total Issues**: 78
- **Duplication**: 4 issues (command files with repeated code)
- **Error Handling**: 29 issues (unchecked errors)
- **Security**: 11 issues (file permissions, path validation)
- **Complexity**: 8 issues (functions exceeding cyclomatic complexity of 15)
- **Constants**: 6 issues (repeated strings)
- **Misspellings**: 5 issues ("cancelled" vs "canceled")
- **Other**: 15 issues (static checks, unused code, etc.)

## Parallel Execution Strategy

Tasks are organized into tracks that can be executed concurrently:
- **Track A**: Quick Fixes (Tasks 1-5) - Simple replacements and deletions
- **Track B**: Security Fixes (Tasks 6-8) - Permission and validation fixes
- **Track C**: Error Handling (Tasks 9-13) - Adding error checks
- **Track D**: Refactoring (Tasks 14-17) - Reducing duplication and complexity
- **Track E**: Code Quality (Tasks 18-21) - Constants and API updates

## Tasks

### Phase 1: Quick Fixes (Low Risk, High Impact)

- [ ] 1. **Fix spelling errors**
  - Replace "cancelled" with "canceled" in 5 locations
  - Files: `internal/executor/command.go` (lines 129, 239)
  - Files: `internal/executor/parallel_test.go` (lines 275, 284)
  - Files: `internal/wizard/config.go` (line 60)
  - _Effort: 5 minutes_

- [ ] 2. **Fix package comment format**
  - Update package comment in `internal/executor/test_helpers.go`
  - Change from `// test_helpers.go` to `// Package executor provides test helpers...`
  - _Effort: 2 minutes_

- [ ] 3. **Remove unused code**
  - Delete unused `progressChan` field in `internal/executor/parallel.go:46`
  - Delete unused `progressUpdate` type in `internal/executor/parallel.go:49`
  - _Effort: 5 minutes_

- [ ] 4. **Fix ineffectual assignment**
  - Fix or remove unused error assignment in `cmd/qualhook/main.go:196`
  - Either handle the error or remove the assignment
  - _Effort: 5 minutes_

- [ ] 5. **Fix unparam issue**
  - Update `(*ProjectDetector).scanWorkspaces` in `internal/detector/project.go:320`
  - Either return meaningful errors or change return type
  - _Effort: 10 minutes_

### Phase 2: Security Fixes (Critical Priority)

- [ ] 6. **Fix directory permissions (G301)**
  - Change from 0755 to 0750 in 3 locations:
  - `cmd/qualhook/man.go:52`
  - `internal/config/templates.go:66`
  - `internal/security/config.go:161`
  - _Requirement: Security compliance_

- [ ] 7. **Fix file permissions (G306)**
  - Change from 0644 to 0600 in 3 locations:
  - `cmd/qualhook/template.go:209`
  - `internal/config/templates.go:77`
  - `internal/wizard/config.go:198`
  - _Requirement: Security compliance_

- [ ] 8. **Add path validation for file operations (G304)**
  - Add validation or #nosec comments for 5 locations:
  - `internal/config/loader.go:121, 253`
  - `internal/config/templates.go:101, 149`
  - `internal/detector/project.go:310`
  - Consider creating a common path validation function
  - _Requirement: Prevent directory traversal attacks_

### Phase 3: Error Handling (Reliability)

- [ ] 9. **Add error checks for UI operations**
  - Handle errors from `fmt.Fprintln`, `fmt.Fprintf` operations
  - Files: `cmd/qualhook/execute.go:236`, `cmd/qualhook/template.go:252,253,272`
  - Can use `_ = fmt.Fprintln(...)` if errors can be safely ignored
  - _Effort: 15 minutes_

- [ ] 10. **Add error checks for completion generation**
  - Handle errors in `cmd/qualhook/completion.go` lines 62, 64, 66, 68
  - Add error returns or log errors appropriately
  - _Effort: 10 minutes_

- [ ] 11. **Add error checks for file operations**
  - Add deferred error checks for `file.Close()` operations
  - Files: `internal/config/loader.go:126, 257`
  - Use pattern: `defer func() { _ = file.Close() }()`
  - _Effort: 10 minutes_

- [ ] 12. **Fix command error handling**
  - Add error checks in `internal/executor/command.go:134, 244`
  - Add error check for `cmd.Process.Wait()` in `internal/executor/errors.go:174`
  - _Effort: 15 minutes_

- [ ] 13. **Fix remaining error checks**
  - Add missing error checks in test helpers and other locations
  - Total of ~10 remaining errcheck issues
  - _Effort: 20 minutes_

### Phase 4: Code Refactoring (Maintainability)

- [ ] 14. **Extract common command logic**
  - Reduce duplication in command files (format.go, lint.go, test.go, typecheck.go)
  - Create base command helper function or struct
  - Extract common initialization and execution logic
  - _Effort: 2 hours_
  - _Blocked by: None_

- [ ] 15. **Reduce executeCommand complexity**
  - Break down `executeCommand` in `cmd/qualhook/execute.go` (complexity: 26)
  - Extract sub-functions for different command types
  - Simplify control flow with early returns
  - _Effort: 1 hour_

- [ ] 16. **Reduce ConfigWizard.Run complexity**
  - Refactor `(*ConfigWizard).Run` in `internal/wizard/config.go` (complexity: 33)
  - Extract configuration steps into separate methods
  - Simplify the main flow
  - _Effort: 1.5 hours_

- [ ] 17. **Reduce other high complexity functions**
  - Refactor 5 remaining functions with complexity > 15
  - Files: `internal/config/validator.go`, `internal/executor/file_aware.go`, 
    `internal/security/validator.go` (2 functions), `internal/wizard/config.go`
  - _Effort: 2 hours total_

### Phase 5: Code Quality Improvements

- [ ] 18. **Extract string constants**
  - Create constants for repeated strings (6 goconst issues)
  - Common error messages, configuration keys, etc.
  - Add to appropriate const blocks
  - _Effort: 30 minutes_

- [ ] 19. **Update deprecated APIs**
  - Replace `cobra.ExactValidArgs` with `MatchAll(ExactArgs(n), OnlyValidArgs)`
  - File: `cmd/qualhook/completion.go:58`
  - _Effort: 10 minutes_

- [ ] 20. **Fix static analysis issues**
  - Fix nil pointer dereference in `cmd/qualhook/commands_test.go:124`
  - Fix empty branch in `internal/config/loader.go:168`
  - Fix memory allocation issue in `internal/filter/optimizations.go:224`
  - Fix unused append in `internal/filter/optimizations_bench_test.go:185`
  - _Effort: 30 minutes_

- [ ] 21. **Fix remaining code quality issues**
  - Address remaining gocritic suggestions
  - Fix `fmt.Sscanf` error handling in `internal/wizard/config.go:319`
  - Fix `w.Flush()` error handling in `cmd/qualhook/template.go:275`
  - Update `exportCmd.MarkFlagRequired` error handling
  - _Effort: 20 minutes_

## Validation

After completing all tasks:
1. Run `make lint` to ensure all issues are resolved
2. Run `make test` to ensure no regressions
3. Update CI to fail on lint errors to prevent future issues
4. Consider adding pre-commit hooks for automated checking

## Dependencies

- No external dependencies required
- All fixes can be done with standard Go tooling
- Consider using `gofumpt` for consistent formatting

## Success Criteria

- [ ] All 78 linting issues resolved
- [ ] CI pipeline passes all checks
- [ ] No new issues introduced
- [ ] Code maintainability improved
- [ ] Security posture enhanced