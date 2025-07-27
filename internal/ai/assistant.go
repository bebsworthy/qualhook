// Package ai provides AI-powered configuration generation for qualhook.
package ai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/AlecAivazis/survey/v2"
	intconfig "github.com/bebsworthy/qualhook/internal/config"
	"github.com/bebsworthy/qualhook/internal/debug"
	"github.com/bebsworthy/qualhook/internal/executor"
	"github.com/bebsworthy/qualhook/pkg/config"
)

// assistantImpl implements the Assistant interface
type assistantImpl struct {
	detector          ToolDetector
	promptGen         PromptGenerator
	parser            ResponseParser
	executor          commandExecutor
	progress          ProgressIndicator
	testRunner        TestRunner
	ui                *InteractiveUI
	selectedTool      string // Cache for tool selection
	toolSelectionTime time.Time
	responseCache     map[string]*cachedResponse // Cache for AI responses
	cacheMutex        sync.RWMutex
}

// cachedResponse holds a cached AI response
type cachedResponse struct {
	response  string
	timestamp time.Time
	ttl       time.Duration
}

// NewAssistant creates a new AI assistant for configuration generation
func NewAssistant(executor commandExecutor) Assistant {
	return &assistantImpl{
		detector:      NewToolDetector(executor),
		promptGen:     NewPromptGenerator(),
		parser:        NewResponseParser(intconfig.NewValidator()),
		executor:      executor,
		progress:      NewProgressIndicator(),
		testRunner:    NewTestRunner(executor),
		ui:            NewInteractiveUI(),
		responseCache: make(map[string]*cachedResponse),
	}
}

// GenerateConfig generates a complete configuration using AI
func (a *assistantImpl) GenerateConfig(ctx context.Context, options AIOptions) (*config.Config, error) {
	debug.Log("Starting AI config generation with options: %+v", options)

	// Phase 1: Tool Detection
	tool, err := a.selectTool(ctx, options)
	if err != nil {
		return nil, err
	}

	// Phase 2: Generate Prompt
	prompt := a.promptGen.GenerateConfigPrompt(options.WorkingDir)
	debug.Log("Generated AI prompt with length: %d", len(prompt))

	// Phase 3: Execute AI Tool
	response, err := a.executeAITool(ctx, tool, prompt, options)
	if err != nil {
		return nil, err
	}

	// Phase 4: Parse Response
	cfg, err := a.parser.ParseConfigResponse(response)
	if err != nil {
		debug.Log("Failed to parse AI response: %v", err)
		// Try to extract partial information if possible
		return a.handlePartialResponse(response, err)
	}

	// Phase 5: Test Commands (if requested)
	if options.TestCommands && options.Interactive {
		err = a.testAndRefineConfig(ctx, cfg)
		if err != nil && !errors.Is(err, context.Canceled) {
			return nil, err
		}
	}

	debug.Log("AI config generation completed successfully")
	return cfg, nil
}

// SuggestCommand suggests a command for a specific purpose
func (a *assistantImpl) SuggestCommand(ctx context.Context, commandType string, projectInfo ProjectContext) (*CommandSuggestion, error) {
	debug.Log("Starting AI command suggestion for type: %s", commandType)

	// Use cached tool selection if available and recent
	var tool Tool
	if a.selectedTool != "" && time.Since(a.toolSelectionTime) < 5*time.Minute {
		tools, err := a.detector.DetectTools()
		if err == nil {
			for _, t := range tools {
				if t.Name == a.selectedTool && t.Available {
					tool = t
					break
				}
			}
		}
	}

	// If no cached tool or it's not available, select one
	if tool.Name == "" {
		options := AIOptions{Interactive: true}
		selectedTool, err := a.selectTool(ctx, options)
		if err != nil {
			return nil, err
		}
		tool = selectedTool
	}

	// Generate command-specific prompt
	prompt := a.promptGen.GenerateCommandPrompt(commandType, projectInfo)

	// Execute AI tool with shorter timeout for single command
	// Use current directory if no working directory specified
	workingDir := "."
	response, err := a.executeAITool(ctx, tool, prompt, AIOptions{
		WorkingDir:  workingDir,
		Interactive: true,
		Timeout:     30 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	// Parse the command suggestion
	suggestion, err := a.parser.ParseCommandResponse(response)
	if err != nil {
		debug.Log("Failed to parse command suggestion: %v", err)
		return nil, NewErrorWithRecovery(
			ErrTypeResponseInvalid,
			"Failed to parse AI command suggestion",
			err,
			[]string{
				"Try rephrasing your request",
				"Use 'qualhook config' to configure this command manually",
				"Check debug output for more details",
			},
			nil,
		)
	}

	debug.Log("AI command suggestion completed: %s", suggestion.Command)
	return suggestion, nil
}

// selectTool selects an AI tool based on options and availability
func (a *assistantImpl) selectTool(_ context.Context, options AIOptions) (Tool, error) {
	// Detect available tools
	tools, err := a.detector.DetectTools()
	if err != nil {
		return Tool{}, NewErrorWithRecovery(
			ErrTypeExecutionFailed,
			"Failed to detect AI tools",
			err,
			[]string{
				"Check if AI CLI tools are in your PATH",
				"Try running 'claude --version' or 'gemini --version' manually",
				"Use 'qualhook config' for manual configuration",
			},
			nil,
		)
	}

	// Filter available tools
	var availableTools []Tool
	for _, tool := range tools {
		if tool.Available {
			availableTools = append(availableTools, tool)
		}
	}

	if len(availableTools) == 0 {
		a.showInstallationInstructions()
		return Tool{}, NewAIError(ErrTypeNoTools, "No AI tools available. Please install Claude or Gemini CLI.", nil)
	}

	// If tool is specified, find it
	if options.Tool != "" {
		for _, tool := range availableTools {
			if tool.Name == options.Tool {
				debug.Log("Using specified AI tool: %s", tool.Name)
				a.cacheToolSelection(tool.Name)
				return tool, nil
			}
		}
		return Tool{}, NewAIError(ErrTypeToolNotFound, fmt.Sprintf("Specified tool '%s' not found", options.Tool), nil)
	}

	// If only one tool available, use it
	if len(availableTools) == 1 {
		tool := availableTools[0]
		debug.Log("Using only available AI tool: %s", tool.Name)
		a.cacheToolSelection(tool.Name)
		return tool, nil
	}

	// Interactive tool selection
	if options.Interactive {
		selectedToolName, err := a.ui.SelectTool(availableTools)
		if err != nil {
			return Tool{}, NewAIError(ErrTypeUserCanceled, "Tool selection canceled", err)
		}
		a.cacheToolSelection(selectedToolName)
		// Find the selected tool
		for _, tool := range availableTools {
			if tool.Name == selectedToolName {
				return tool, nil
			}
		}
		return Tool{}, NewAIError(ErrTypeToolNotFound, "Selected tool not found", nil)
	}

	// Non-interactive: use first available tool
	tool := availableTools[0]
	debug.Log("Using first available AI tool (non-interactive): %s", tool.Name)
	a.cacheToolSelection(tool.Name)
	return tool, nil
}

// executeAITool executes the AI tool with proper progress handling
func (a *assistantImpl) executeAITool(ctx context.Context, tool Tool, prompt string, options AIOptions) (string, error) {
	debug.Log("Executing AI tool %s in directory: %s", tool.Name, options.WorkingDir)

	// Generate cache key from prompt and tool
	cacheKey := a.generateCacheKey(tool.Name, prompt)

	// Check cache for recent responses
	if cached := a.getCachedResponse(cacheKey); cached != nil {
		debug.Log("Using cached AI response for key: %s", cacheKey[:8])
		return cached.response, nil
	}

	// Start progress indicator if interactive
	if options.Interactive {
		a.progress.Start(fmt.Sprintf("Analyzing project with %s...", tool.Name))
		defer a.progress.Stop()
	}

	// Create a cancellable context
	execCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Apply timeout if specified
	if options.Timeout > 0 {
		timeoutCtx, timeoutCancel := context.WithTimeout(execCtx, options.Timeout)
		defer timeoutCancel()
		execCtx = timeoutCtx
	}

	// Set up cancellation handling if interactive
	if options.Interactive {
		go func() {
			select {
			case <-a.progress.WaitForCancellation(ctx):
				debug.Log("User requested cancellation")
				cancel()
			case <-ctx.Done():
				// Context canceled by other means
			}
		}()
	}

	// Execute the AI tool
	execOptions := executor.ExecOptions{
		WorkingDir: options.WorkingDir,
		InheritEnv: true,
		Timeout:    0, // We handle timeout via context
	}

	// Build command args
	args := buildAIToolArgs(tool.Name, prompt)

	// Execute with proper context handling
	resultChan := make(chan *executor.ExecResult, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := a.executor.Execute(tool.Command, args, execOptions)
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- result
	}()

	// Wait for execution or cancellation
	select {
	case <-execCtx.Done():
		if errors.Is(execCtx.Err(), context.Canceled) {
			return "", NewAIError(ErrTypeUserCanceled, "AI analysis canceled by user", execCtx.Err())
		}
		return "", NewAIError(ErrTypeTimeout, "AI analysis timed out", execCtx.Err())

	case err := <-errChan:
		return "", NewAIError(ErrTypeExecutionFailed, fmt.Sprintf("Failed to execute %s", tool.Name), err)

	case result := <-resultChan:
		if result.Error != nil {
			return "", NewAIError(ErrTypeExecutionFailed, fmt.Sprintf("%s execution failed", tool.Name), result.Error)
		}
		if result.ExitCode != 0 {
			debug.Log("AI tool returned non-zero exit code %d: %s", result.ExitCode, result.Stderr)
			return "", NewAIError(ErrTypeExecutionFailed, fmt.Sprintf("%s failed with exit code %d: %s", tool.Name, result.ExitCode, result.Stderr), nil)
		}

		// Cache successful response
		a.cacheResponse(cacheKey, result.Stdout, 10*time.Minute)

		return result.Stdout, nil
	}
}

// testAndRefineConfig tests the generated configuration and allows refinement
func (a *assistantImpl) testAndRefineConfig(ctx context.Context, cfg *config.Config) error {
	debug.Log("Starting command testing phase")

	// Show configuration summary
	if err := a.ui.ReviewConfiguration(cfg); err != nil {
		return err
	}

	// Ask user if they want to test commands
	confirm := false
	prompt := &survey.Confirm{
		Message: "Would you like to test the commands now?",
		Default: true,
	}
	if err := survey.AskOne(prompt, &confirm); err != nil {
		return err
	}
	if !confirm {
		debug.Log("User skipped command testing")
		return nil
	}

	// Test commands
	commands := cfg.Commands
	results, err := a.testRunner.TestCommands(ctx, commands)
	if err != nil {
		return err
	}

	// Update configuration with any modifications
	for name, result := range results {
		if result.Modified && result.FinalCommand != nil {
			cfg.Commands[name] = result.FinalCommand
			debug.Log("Updated command after testing: %s", name)
		}
	}

	return nil
}

// handlePartialResponse attempts to extract useful information from a partial response
func (a *assistantImpl) handlePartialResponse(response string, parseErr error) (*config.Config, error) {
	debug.Log("Attempting to handle partial AI response")

	// Try to extract any valid JSON sections
	cfg := &config.Config{
		Commands: make(map[string]*config.CommandConfig),
	}

	// Look for command patterns in the response
	// This is a best-effort approach to salvage something useful
	commandTypes := []string{"format", "lint", "typecheck", "test"}
	for _, cmdType := range commandTypes {
		if cmd := extractCommandFromResponse(response, cmdType); cmd != nil {
			cfg.Commands[cmdType] = cmd
			debug.Log("Extracted partial command: %s", cmdType)
		}
	}

	// Use the new extraction helper
	partialCommands, recoveryHint := ExtractPartialConfig(response, parseErr)

	if len(cfg.Commands) > 0 {
		debug.Log("Extracted partial configuration from AI response with %d commands", len(cfg.Commands))
		// If we have some commands, return them with a recovery error
		if recoveryHint != "" {
			// Show user what we recovered
			fmt.Println(recoveryHint)
		}
		return cfg, nil
	}

	// Check if we detected any commands even if we couldn't parse them
	if len(partialCommands) > 0 {
		return nil, NewErrorWithRecovery(
			ErrTypeResponseInvalid,
			msgResponseInvalid,
			parseErr,
			GetRecoverySuggestions(ErrTypeResponseInvalid),
			partialCommands,
		)
	}

	// No partial data could be extracted
	return nil, NewErrorWithRecovery(
		ErrTypeResponseInvalid,
		"Failed to parse AI response and no partial data could be extracted",
		parseErr,
		GetRecoverySuggestions(ErrTypeResponseInvalid),
		nil,
	)
}

// showInstallationInstructions displays installation instructions for AI tools
func (a *assistantImpl) showInstallationInstructions() {
	// Use the centralized installation instructions
	fmt.Println("\n" + msgNoToolsAvailable)
}

// cacheToolSelection caches the selected tool for the session
func (a *assistantImpl) cacheToolSelection(toolName string) {
	a.selectedTool = toolName
	a.toolSelectionTime = time.Now()
}

// buildAIToolArgs builds command arguments for the AI tool
func buildAIToolArgs(toolName string, prompt string) []string {
	switch toolName {
	case "claude":
		// Claude CLI expects the prompt as a direct argument
		return []string{prompt}
	case "gemini":
		// Gemini CLI might need different args
		return []string{"--prompt", prompt}
	default:
		// Default to Claude-style
		return []string{prompt}
	}
}

// extractCommandFromResponse attempts to extract a command from raw response text
func extractCommandFromResponse(response string, cmdType string) *config.CommandConfig {
	// This is a simple heuristic approach
	// Look for common patterns in AI responses

	// Convert response to lowercase for case-insensitive matching
	lowerResponse := strings.ToLower(response)

	// Look for the command type mentioned with a command
	patterns := []struct {
		prefix string
		suffix string
	}{
		{cmdType + ":", ""},
		{cmdType + " command:", ""},
		{"\"" + cmdType + "\":", ""},
		{cmdType + " ->", ""},
	}

	for _, pattern := range patterns {
		idx := strings.Index(lowerResponse, pattern.prefix)
		if idx >= 0 {
			// Extract the line containing the command
			start := idx + len(pattern.prefix)
			end := strings.IndexAny(response[start:], "\n,}")
			if end == -1 {
				end = len(response) - start
			}

			cmdLine := strings.TrimSpace(response[start : start+end])

			// Remove quotes if present
			cmdLine = strings.Trim(cmdLine, "\"'")

			// Try to parse the command
			parts := strings.Fields(cmdLine)
			if len(parts) > 0 {
				return &config.CommandConfig{
					Command: parts[0],
					Args:    parts[1:],
				}
			}
		}
	}

	return nil
}

// generateCacheKey creates a cache key from tool name and prompt
func (a *assistantImpl) generateCacheKey(toolName, prompt string) string {
	h := sha256.New()
	h.Write([]byte(toolName))
	h.Write([]byte(prompt))
	return hex.EncodeToString(h.Sum(nil))
}

// getCachedResponse retrieves a cached response if it's still valid
func (a *assistantImpl) getCachedResponse(key string) *cachedResponse {
	a.cacheMutex.RLock()
	defer a.cacheMutex.RUnlock()

	if cached, exists := a.responseCache[key]; exists {
		if time.Since(cached.timestamp) < cached.ttl {
			return cached
		}
	}
	return nil
}

// cacheResponse stores a response in the cache
func (a *assistantImpl) cacheResponse(key, response string, ttl time.Duration) {
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()

	a.responseCache[key] = &cachedResponse{
		response:  response,
		timestamp: time.Now(),
		ttl:       ttl,
	}

	// Clean up old entries
	a.cleanupCache()
}

// cleanupCache removes expired entries from the cache
func (a *assistantImpl) cleanupCache() {
	now := time.Now()
	for key, cached := range a.responseCache {
		if now.Sub(cached.timestamp) > cached.ttl {
			delete(a.responseCache, key)
		}
	}
}
