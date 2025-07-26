# Implementation Plan

## Phase 1: Core AI Infrastructure

- [x] 1. Create AI package structure and interfaces
  - Create `internal/ai/` directory structure
  - Define core interfaces in `internal/ai/types.go`
  - Create `AIOptions`, `Tool`, and `CommandSuggestion` types
  - _Requirements: 2.1_

- [x] 2. Implement Tool Detector
  - Create `internal/ai/detector.go`
  - Implement `DetectTools()` to check for claude/gemini CLI availability
  - Add version detection using `--version` flags
  - Write unit tests in `internal/ai/detector_test.go`
  - _Requirements: 2.1, 2.2_

- [x] 3. Create Prompt Generator
  - Create `internal/ai/prompt.go`
  - Implement prompt templates for full config generation
  - Add template for individual command suggestions
  - Include monorepo detection instructions in prompts
  - Write unit tests with various project scenarios
  - _Requirements: 2.3_

- [x] 4. Build Response Parser
  - Create `internal/ai/parser.go`
  - Implement JSON parsing with validation
  - Add partial response recovery logic
  - Handle monorepo path configurations
  - Write comprehensive parsing tests
  - _Requirements: 2.5, 2.6_

## Phase 2: Interactive Components

- [x] 5. Implement Progress Indicator
  - Create `internal/ai/progress.go`
  - Add spinner with elapsed time display
  - Implement ESC key cancellation handler
  - Show "Press ESC to cancel" message
  - Test cancellation behavior
  - _Requirements: 2.4, 5.5_

- [x] 6. Create Test Runner
  - Create `internal/ai/tester.go`
  - Implement command test execution with user approval
  - Add output capture and display
  - Handle test failures with modification options
  - Write tests for various command scenarios
  - _Requirements: 2.7_

- [x] 7. Build Interactive UI helpers
  - Create `internal/ai/ui.go`
  - Implement tool selection prompt
  - Add command review interface
  - Create test approval dialogs
  - _Requirements: 3.5, 6.5_

## Phase 3: AI Assistant Service

- [x] 8. Implement core Assistant service
  - Create `internal/ai/assistant.go`
  - Wire together detector, prompt generator, and parser
  - Add `GenerateConfig()` method with full flow
  - Implement `SuggestCommand()` for individual commands
  - Handle timeouts and cancellations
  - _Requirements: 2.1-2.8_

- [x] 9. Add error handling and recovery
  - Implement all error types from design
  - Add fallback mechanisms for each error scenario
  - Create helpful error messages with next steps
  - Test error recovery paths
  - _Requirements: 5.1-5.5_

- [x] 10. Integrate security validation
  - Connect to existing `SecurityValidator`
  - Validate AI-suggested commands before testing
  - Add logging with privacy protection
  - Write security-focused tests
  - _Requirements: 4.3, 4.4_

## Phase 4: Wizard Enhancement

- [ ] 11. Create wizard AI integration
  - Create `internal/wizard/ai_integration.go`
  - Implement `ReviewCommands()` with AI options
  - Add AI assistance option to command configuration
  - Maintain backward compatibility
  - _Requirements: 1.1-1.7_

- [ ] 12. Modify existing wizard flow
  - Update `internal/wizard/config.go`
  - Add mandatory command review for all types
  - Include custom command addition support
  - Integrate AI tool selection when needed
  - Test enhanced wizard flow
  - _Requirements: 1.1, 1.2, 1.6_

## Phase 5: CLI Command Implementation

- [x] 13. Create ai-config command
  - Create `cmd/qualhook/ai-config.go`
  - Implement command with --tool flag support
  - Add proper help documentation
  - Handle existing config merge/overwrite
  - _Requirements: 3.1-3.5_

- [ ] 14. Register and test CLI command
  - Add command to `cmd/qualhook/main.go`
  - Write integration tests for ai-config command
  - Test with mock AI tools
  - Verify help output
  - _Requirements: 3.1_

## Phase 6: Testing Infrastructure

- [x] 15. Create mock AI tool for testing
  - Create `internal/testutil/mock_ai.go`
  - Implement configurable responses
  - Add delay simulation
  - Support various test scenarios
  - _Requirements: Testing Strategy_

- [ ] 16. Write comprehensive integration tests
  - Create `internal/ai/integration_test.go`
  - Test end-to-end ai-config flow
  - Test wizard AI enhancement
  - Cover monorepo scenarios
  - Test cancellation and error cases
  - _Requirements: All_

## Phase 7: Documentation and Polish

- [x] 17. Add installation instructions
  - Create platform-specific install guides
  - Add to error messages when tools not found
  - Include in help documentation
  - _Requirements: 2.2, 5.2_

- [ ] 18. Optimize performance
  - Implement concurrent tool detection
  - Add response caching for retries
  - Clean up resources properly
  - Profile and optimize critical paths
  - _Requirements: Performance Requirements_

## Testing Checklist

- [ ] 19. Verify all acceptance criteria
  - Test each requirement's acceptance criteria
  - Ensure proper user consent flow
  - Verify command testing works correctly
  - Check monorepo support
  - Validate security measures
  - _Requirements: All_

## Notes

- Each task should be completed with tests before moving to the next
- Use existing patterns from the codebase (e.g., `CommandExecutor`, `survey` prompts)
- Maintain backward compatibility with existing configurations
- Focus on user control and clear feedback throughout