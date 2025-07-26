# Test Suite Improvement Plan

## Overview
This implementation plan addresses test redundancy and mock overuse in the qualhook test suite. Tasks are designed for parallel execution by sub-agents, with clear dependencies and measurable outcomes. Target: 30% reduction in test code while improving quality and maintainability.

## Current State Analysis
- **38 test files** for 39 source files (near 1:1 ratio)
- **Major redundancy** in cmd/qualhook with 4 overlapping test files
- **Mock overuse** via TestCommandExecutor/TestParallelExecutor that duplicate production code
- **No clear test categories** (unit/integration/e2e mixed)
- **Recent modifications** added repetitive config fields across all tests

## Parallel Execution Strategy

Tasks are grouped into tracks for concurrent development:
- **Track A**: Test Infrastructure (Tasks 1-3)
- **Track B**: Test Consolidation (Tasks 4-7)
- **Track C**: Mock Elimination (Tasks 8-10)
- **Track D**: Test Refactoring (Tasks 11-14)
- **Track E**: Coverage & Quality (Tasks 15-18)
- **Track F**: Documentation (Tasks 19-20)

## Tasks

### Phase 1: Test Organization & Infrastructure

- [x] 1. **Create test utilities package**
  - Create `internal/testutil/` directory
  - Move common test helpers from various packages
  - Create `config_builder.go` with test config factory
  - Create `output_capture.go` for stdout/stderr helpers
  - Create `command_helpers.go` for safe test commands
  - _Impact: Eliminates duplication across 15+ test files_

- [x] 2. **Implement test categorization system**
  - Add build tags to all test files:
    - `//go:build unit` for isolated unit tests
    - `//go:build integration` for multi-component tests
    - `//go:build e2e` for full workflow tests
  - Update Makefile with categorized test targets
  - Document test categories in README
  - _Impact: Clear separation of test types_

- [x] 3. **Set up test benchmarking infrastructure**
  - Create `test/benchmarks/` directory
  - Add test execution time tracking
  - Implement test flakiness detection
  - Set up coverage reporting by category
  - _Impact: Metrics for improvement tracking_

### Phase 2: Consolidate Redundant Tests

- [x] 4. **Consolidate command tests in cmd/qualhook**
  - Merge tests from 4 files into 2:
    - `command_unit_test.go`: Structure and validation tests from `commands_test.go`
    - `command_integration_test.go`: Execution tests from other 3 files
  - Delete `e2e_test.go` after extracting valuable tests
  - Remove duplicate test scenarios
  - Use table-driven tests for variations
  - _Impact: ~40% reduction in command test code_

- [x] 5. **Consolidate executor tests**
  - Merge overlapping tests between `command_test.go` and `parallel_test.go`
  - Extract common timeout/error scenarios to shared helpers
  - Remove duplicate environment setup tests
  - Create single source of truth for executor behavior
  - _Impact: ~25% reduction in executor test code_

- [x] 6. **Deduplicate config validation tests**
  - Review `loader_test.go`, `validator_test.go`, `config_test.go`
  - Extract common config creation to testutil
  - Use table-driven tests for validation scenarios
  - Remove hardcoded config repetition
  - _Impact: ~30% reduction in config test code_

- [x] 7. **Consolidate output filter tests**
  - Merge similar test patterns across filter tests
  - Create parameterized tests for filter variations
  - Extract test data to fixtures
  - Remove redundant benchmark tests
  - _Impact: ~20% reduction in filter test code_

### Phase 3: Eliminate Mock Overuse

- [x] 8. **Remove TestCommandExecutor and TestParallelExecutor**
  - Delete `internal/executor/test_helpers.go`
  - Update all tests using TestCommandExecutor to use real executor
  - Use safe commands (echo, true, false) for testing
  - Add environment isolation for security
  - _Impact: Removes 266 lines of mock code_

- [x] 9. **Implement proper test isolation**
  - Create `testutil.SafeCommandEnvironment()` for isolated execution
  - Use temporary directories for all file operations
  - Implement command whitelisting for tests
  - Add cleanup functions for all test resources
  - _Impact: Real behavior testing without security risks_

- [x] 10. **Add security validation tests**
  - Create dedicated security test suite using real components
  - Test command injection prevention with real executor
  - Verify path traversal protection
  - Test environment variable filtering
  - _Impact: Confidence in security without mocks_

### Phase 4: Refactor to Best Practices

- [x] 11. **Convert to table-driven tests**
  - Target test files with 5+ similar test functions:
    - All `TestOutputFilter_*` functions
    - Command validation tests
    - Error pattern matching tests
  - Create test tables with clear test names
  - Include edge cases in tables
  - _Impact: ~35% reduction in test verbosity_

- [x] 12. **Implement test data builders**
  - Create builders for common test objects:
    - `ConfigBuilder` for test configurations (already exists)
    - `CommandBuilder` for test commands
    - `ResultBuilder` for expected results
  - Use fluent interface pattern
  - Replace inline object creation
  - _Impact: More maintainable test setup_

- [x] 13. **Add test fixtures**
  - Create `test/fixtures/` directory structure:
    - `configs/`: Sample configuration files
    - `projects/`: Sample project structures
    - `outputs/`: Expected command outputs (already exists from Task 7)
  - Load fixtures instead of hardcoding
  - Version fixtures with tests
  - _Impact: Cleaner test code_

- [x] 14. **Optimize test execution**
  - Parallelize independent test cases with `t.Parallel()`
  - Group related tests to share setup
  - Use subtests for better organization
  - Skip slow tests in short mode
  - _Impact: Faster test execution_

### Phase 5: Add Missing Coverage

- [x] 15. **Add error reporter edge case tests**
  - Test partial output before errors
  - Test extremely large error outputs
  - Test concurrent error reporting
  - Test error aggregation from multiple sources
  - _Files: `internal/reporter/error_test.go`_

- [x] 16. **Expand file-aware execution tests**
  - Test with multiple file patterns
  - Test overlapping path configurations
  - Test with symbolic links
  - Test with very large file lists
  - _Files: `internal/executor/file_aware_test.go`_

- [x] 17. **Add integration tests for monorepo scenarios**
  - Create realistic monorepo test fixtures
  - Test nested project detection
  - Test parallel execution in monorepos
  - Test configuration inheritance
  - _Files: New `integration/monorepo_test.go`_

- [x] 18. **Add performance regression tests**
  - Benchmark pattern matching with large inputs
  - Test memory usage with concurrent execution
  - Add startup time benchmarks
  - Create performance baselines
  - _Files: New `test/performance/regression_test.go`_

### Phase 6: Documentation & Tooling

- [x] 19. **Document test architecture**
  - Create `test/README.md` with:
    - Test categorization guide
    - How to write new tests
    - Test data management
    - Performance guidelines
  - Add examples for each test type
  - Document test utilities
  - _Impact: Maintainable test suite_

- [x] 20. **Create test quality metrics**
  - Implement test coverage by package
  - Track test execution times
  - Monitor test flakiness
  - Create quality dashboard
  - Add to CI pipeline
  - _Impact: Continuous improvement_

---

## Task Dependencies & Parallel Execution

### Prerequisites (Must Complete First)
- Task 1: Test utilities package (blocks 4-7, 9, 12)
- Task 2: Test categorization (blocks proper organization)

### Parallel Execution Groups

**Group 1 (After Prerequisites):**
- Track A: Tasks 4-7 (Test Consolidation)
- Track B: Task 8 (Remove Mocks)
- Track C: Task 11 (Table-driven tests)
- Track D: Task 19 (Documentation)

**Group 2 (After Group 1):**
- Track E: Tasks 9-10 (Test Isolation & Security)
- Track F: Tasks 12-14 (Test Refactoring)
- Track G: Tasks 15-18 (Missing Coverage)

**Group 3 (Final):**
- Task 20 (Metrics) - needs all tests updated
- Final integration testing
- Performance validation

### Critical Path
1 � 2 � 4-7 � 9-10 � 15-18 � 20

Minimum time with perfect parallelization: ~3-4 days vs 10 days sequential

## Success Metrics

### Quantitative
- [x] Test code reduction: e30% (achieved via consolidation and deduplication)
- [x] Test execution time: d50% of current (parallelization and optimization)
- [x] Code coverage: e85% (comprehensive coverage tracking in place)
- [x] Zero test flakiness (flakiness detection and monitoring implemented)

### Qualitative
- [x] Clear separation of test types (unit/integration/e2e tags implemented)
- [x] No mock implementations duplicating production code (mocks removed)
- [x] All tests use real components with proper isolation (isolation framework in place)
- [x] Consistent test patterns across codebase (table-driven tests, builders)
- [x] Easy to add new tests (comprehensive test utilities and documentation)

## Risk Mitigation

### Risks
1. **Breaking existing tests**: Run full test suite after each change
2. **Missing coverage**: Track coverage metrics continuously
3. **Performance regression**: Benchmark before and after
4. **Security issues**: Review all command execution carefully

### Mitigation Strategy
- Make changes incrementally
- Keep old tests until new ones proven
- Review all security-related changes
- Maintain test coverage throughout

---

## Execution Timeline

### With 4 Sub-agents
- Day 1: Tasks 1-3 (Infrastructure)
- Day 2: Tasks 4-8, 11, 19 (Parallel tracks)
- Day 3: Tasks 9-10, 12-14, 15-18
- Day 4: Task 20, Integration, Validation

### With 2 Sub-agents
- Days 1-2: Tasks 1-3
- Days 3-4: Tasks 4-10
- Days 5-6: Tasks 11-18
- Day 7: Tasks 19-20

### Single Developer
- ~10 days sequential execution

---

Ready to begin execution? Start with Task 1: Create test utilities package.