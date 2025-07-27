// Package ai provides AI-powered configuration generation for qualhook.
package ai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bebsworthy/qualhook/internal/config"
	pkgconfig "github.com/bebsworthy/qualhook/pkg/config"
)

// aiResponse represents the expected JSON structure from AI tools
type aiResponse struct {
	Version        string                `json:"version"`
	ProjectType    string                `json:"projectType"`
	Monorepo       *monorepoInfo         `json:"monorepo,omitempty"`
	Commands       map[string]*aiCommand `json:"commands"`
	Paths          []aiPathConfig        `json:"paths,omitempty"`
	CustomCommands map[string]*aiCommand `json:"customCommands,omitempty"`
}

// monorepoInfo contains monorepo detection information
type monorepoInfo struct {
	Detected   bool     `json:"detected"`
	Type       string   `json:"type"`
	Workspaces []string `json:"workspaces"`
}

// aiCommand represents a command configuration from AI
type aiCommand struct {
	Command       string           `json:"command"`
	Args          []string         `json:"args,omitempty"`
	ErrorPatterns []aiRegexPattern `json:"errorPatterns,omitempty"`
	ExitCodes     []int            `json:"exitCodes,omitempty"`
	Explanation   string           `json:"explanation,omitempty"`
}

// aiRegexPattern represents a regex pattern from AI
type aiRegexPattern struct {
	Pattern string `json:"pattern"`
	Flags   string `json:"flags,omitempty"`
}

// aiPathConfig represents path-specific configuration from AI
type aiPathConfig struct {
	Path     string                `json:"path"`
	Commands map[string]*aiCommand `json:"commands"`
}

// ResponseParserImpl implements the ResponseParser interface
type ResponseParserImpl struct {
	validator *config.Validator
}

// NewResponseParser creates a new response parser
func NewResponseParser(validator *config.Validator) ResponseParser {
	return &ResponseParserImpl{
		validator: validator,
	}
}

// ParseConfigResponse parses a full configuration response from AI
func (p *ResponseParserImpl) ParseConfigResponse(response string) (*pkgconfig.Config, error) {
	// Try to extract JSON from the response
	jsonStr := p.extractJSON(response)
	if jsonStr == "" {
		return nil, &AIError{
			Type:    ErrTypeResponseInvalid,
			Message: "no valid JSON found in AI response",
		}
	}

	// Parse the JSON
	var aiResp aiResponse
	if err := json.Unmarshal([]byte(jsonStr), &aiResp); err != nil {
		// Try to recover partial response
		partial, partialErr := p.recoverPartialResponse(jsonStr)
		if partialErr != nil {
			return nil, &AIError{
				Type:    ErrTypeResponseInvalid,
				Message: "failed to parse AI response",
				Cause:   err,
			}
		}
		aiResp = *partial
	}

	// Convert to qualhook config
	cfg := p.convertToConfig(&aiResp)

	// Validate the configuration
	if err := p.validator.Validate(cfg); err != nil {
		// Try to fix common issues
		fixedCfg := p.attemptAutoFix(cfg, err)
		if fixedCfg != nil {
			cfg = fixedCfg
		} else {
			return nil, &AIError{
				Type:    ErrTypeValidationFailed,
				Message: "configuration validation failed",
				Cause:   err,
			}
		}
	}

	return cfg, nil
}

// ParseCommandResponse parses a single command suggestion from AI
func (p *ResponseParserImpl) ParseCommandResponse(response string) (*CommandSuggestion, error) {
	// Try to extract JSON from the response
	jsonStr := p.extractJSON(response)
	if jsonStr == "" {
		// Try to parse as a simple command suggestion
		return p.parseSimpleCommand(response)
	}

	// Parse the JSON
	var cmd aiCommand
	if err := json.Unmarshal([]byte(jsonStr), &cmd); err != nil {
		return nil, &AIError{
			Type:    ErrTypeResponseInvalid,
			Message: "failed to parse command response",
			Cause:   err,
		}
	}

	// Convert to CommandSuggestion
	suggestion := &CommandSuggestion{
		Command:     cmd.Command,
		Args:        cmd.Args,
		ExitCodes:   cmd.ExitCodes,
		Explanation: cmd.Explanation,
	}

	// Convert error patterns
	for _, pattern := range cmd.ErrorPatterns {
		regexPattern := &pkgconfig.RegexPattern{
			Pattern: pattern.Pattern,
			Flags:   pattern.Flags,
		}

		// Validate the pattern
		if err := regexPattern.Validate(); err != nil {
			// Skip invalid patterns but log them
			continue
		}

		suggestion.ErrorPatterns = append(suggestion.ErrorPatterns, *regexPattern)
	}

	// Validate the command
	cmdConfig := &pkgconfig.CommandConfig{
		Command:       suggestion.Command,
		Args:          suggestion.Args,
		ExitCodes:     suggestion.ExitCodes,
		ErrorPatterns: p.convertRegexPatterns(suggestion.ErrorPatterns),
	}

	if err := p.validator.ValidateCommand(cmdConfig); err != nil {
		return nil, &AIError{
			Type:    ErrTypeValidationFailed,
			Message: "command validation failed",
			Cause:   err,
		}
	}

	return suggestion, nil
}

// extractJSON attempts to extract JSON from AI response text
func (p *ResponseParserImpl) extractJSON(response string) string {
	// Try code block extraction first
	if json := p.extractFromCodeBlock(response); json != "" {
		return json
	}

	// Try raw JSON extraction
	if json := p.extractRawJSON(response); json != "" {
		return json
	}

	return ""
}

// extractFromCodeBlock extracts JSON from markdown code blocks
func (p *ResponseParserImpl) extractFromCodeBlock(response string) string {
	// Look for JSON code blocks first
	if json := p.extractMarkedCodeBlock(response, "```json", 7); json != "" {
		return json
	}

	// Look for any code blocks that contain JSON
	if json := p.extractGenericCodeBlock(response); json != "" {
		return json
	}

	return ""
}

// extractMarkedCodeBlock extracts content from a specific code block type
func (p *ResponseParserImpl) extractMarkedCodeBlock(response, marker string, offset int) string {
	if !strings.Contains(response, marker) {
		return ""
	}

	start := strings.Index(response, marker)
	end := strings.Index(response[start+offset:], "```")
	if end > 0 {
		return strings.TrimSpace(response[start+offset : start+offset+end])
	}
	return ""
}

// extractGenericCodeBlock extracts JSON from generic code blocks
func (p *ResponseParserImpl) extractGenericCodeBlock(response string) string {
	if !strings.Contains(response, "```") {
		return ""
	}

	start := strings.Index(response, "```")
	end := strings.Index(response[start+3:], "```")
	if end > 0 {
		content := strings.TrimSpace(response[start+3 : start+3+end])
		// Check if it looks like JSON
		if strings.HasPrefix(content, "{") || strings.HasPrefix(content, "[") {
			return content
		}
	}
	return ""
}

// extractRawJSON extracts JSON that appears directly in the text
func (p *ResponseParserImpl) extractRawJSON(response string) string {
	// First check if the whole response is JSON
	trimmed := strings.TrimSpace(response)
	if strings.HasPrefix(trimmed, "{") {
		return trimmed
	}

	// Then try to find JSON that starts at the beginning of a line
	return p.extractJSONFromLines(response)
}

// extractJSONFromLines searches for JSON objects in response lines
func (p *ResponseParserImpl) extractJSONFromLines(response string) string {
	lines := strings.Split(response, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "{") {
			jsonStr := strings.Join(lines[i:], "\n")
			if extracted := p.extractCompleteJSON(jsonStr); extracted != "" {
				return extracted
			}
		}
	}
	return ""
}

// extractCompleteJSON extracts a complete JSON object from a string
func (p *ResponseParserImpl) extractCompleteJSON(jsonStr string) string {
	depth := 0
	lastBrace := -1
	inString := false
	escape := false

	for j, ch := range jsonStr {
		if escape {
			escape = false
			continue
		}

		if ch == '\\' {
			escape = true
			continue
		}

		if ch == '"' && !escape {
			inString = !inString
			continue
		}

		if !inString {
			if ch == '{' {
				depth++
			} else if ch == '}' {
				depth--
				if depth == 0 {
					lastBrace = j
					break
				}
			}
		}
	}

	if lastBrace > 0 {
		return strings.TrimSpace(jsonStr[:lastBrace+1])
	} else if depth > 0 {
		// Incomplete JSON, return what we have for recovery
		return jsonStr
	}

	return ""
}

// recoverPartialResponse attempts to extract valid parts from partially invalid JSON
func (p *ResponseParserImpl) recoverPartialResponse(jsonStr string) (*aiResponse, error) {
	// Try to fix common JSON issues
	fixed := p.fixCommonJSONIssues(jsonStr)

	var partial aiResponse
	if err := json.Unmarshal([]byte(fixed), &partial); err == nil {
		// Set default version if not provided
		if partial.Version == "" {
			partial.Version = "1.0"
		}
		return &partial, nil
	}

	// Try to extract just the commands section
	if commands := p.extractCommandsSection(jsonStr); commands != "" {
		partial.Commands = make(map[string]*aiCommand)
		if err := json.Unmarshal([]byte(commands), &partial.Commands); err == nil {
			// Set default version
			partial.Version = "1.0"
			return &partial, nil
		}
	}

	return nil, fmt.Errorf("unable to recover any valid configuration from response")
}

// fixCommonJSONIssues attempts to fix common JSON formatting issues
func (p *ResponseParserImpl) fixCommonJSONIssues(jsonStr string) string {
	// Remove trailing commas
	fixed := strings.ReplaceAll(jsonStr, ",]", "]")
	fixed = strings.ReplaceAll(fixed, ",}", "}")

	// Fix trailing commas before newlines
	fixed = strings.ReplaceAll(fixed, ",\n}", "\n}")
	fixed = strings.ReplaceAll(fixed, ",\n]", "\n]")

	// Fix unescaped quotes in strings (basic attempt)
	// This is a simplified fix and may not handle all cases

	// Add missing closing braces/brackets if needed
	openBraces := strings.Count(fixed, "{") - strings.Count(fixed, "}")
	for i := 0; i < openBraces; i++ {
		fixed += "}"
	}

	openBrackets := strings.Count(fixed, "[") - strings.Count(fixed, "]")
	for i := 0; i < openBrackets; i++ {
		fixed += "]"
	}

	return fixed
}

// extractCommandsSection tries to extract just the commands object from JSON
func (p *ResponseParserImpl) extractCommandsSection(jsonStr string) string {
	// Look for "commands": { ... }
	start := strings.Index(jsonStr, `"commands"`)
	if start < 0 {
		return ""
	}

	// Find the opening brace
	openIdx := strings.Index(jsonStr[start:], "{")
	if openIdx < 0 {
		return ""
	}

	// Find the matching closing brace
	depth := 0
	closeIdx := -1
	for i := start + openIdx; i < len(jsonStr); i++ {
		if jsonStr[i] == '{' {
			depth++
		} else if jsonStr[i] == '}' {
			depth--
			if depth == 0 {
				closeIdx = i
				break
			}
		}
	}

	if closeIdx > 0 {
		return jsonStr[start+openIdx : closeIdx+1]
	}

	return ""
}

// parseSimpleCommand attempts to parse a simple command from plain text
func (p *ResponseParserImpl) parseSimpleCommand(response string) (*CommandSuggestion, error) {
	// Look for command-like patterns in the response
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and obvious non-commands
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		// Skip lines that are clearly not commands but don't skip lines ending with :
		if strings.HasPrefix(strings.ToLower(line), "to ") ||
			strings.HasPrefix(strings.ToLower(line), "run ") ||
			(strings.Contains(line, ":") && !strings.HasSuffix(line, ":")) ||
			(strings.HasSuffix(line, ".") && !strings.Contains(line, " ")) {
			continue
		}

		// Look for lines that might be commands
		parts := strings.Fields(line)
		if len(parts) > 0 && p.looksLikeCommand(parts[0]) {
			return &CommandSuggestion{
				Command:     parts[0],
				Args:        parts[1:],
				Explanation: "Extracted from AI response",
			}, nil
		}
	}

	return nil, &AIError{
		Type:    ErrTypeResponseInvalid,
		Message: "no valid command found in response",
	}
}

// looksLikeCommand checks if a string looks like a command
func (p *ResponseParserImpl) looksLikeCommand(s string) bool {
	// Common command names
	commonCommands := []string{
		"npm", "yarn", "pnpm", "go", "cargo", "python", "pip",
		"jest", "mocha", "pytest", "rspec", "gradle", "maven",
		"eslint", "prettier", "black", "rustfmt", "gofmt",
		"tsc", "mypy", "pylint", "rubocop",
	}

	s = strings.ToLower(s)
	for _, cmd := range commonCommands {
		if strings.HasPrefix(s, cmd) {
			return true
		}
	}

	return false
}

// convertToConfig converts AI response to qualhook configuration
func (p *ResponseParserImpl) convertToConfig(aiResp *aiResponse) *pkgconfig.Config {
	cfg := &pkgconfig.Config{
		Version:     "1.0",
		ProjectType: aiResp.ProjectType,
		Commands:    make(map[string]*pkgconfig.CommandConfig),
	}

	// Set version from response if provided
	if aiResp.Version != "" {
		cfg.Version = aiResp.Version
	}

	// Convert standard commands
	for name, cmd := range aiResp.Commands {
		cmdConfig, err := p.convertCommand(cmd)
		if err != nil {
			// Skip invalid commands but continue
			continue
		}
		cfg.Commands[name] = cmdConfig
	}

	// Convert custom commands
	for name, cmd := range aiResp.CustomCommands {
		cmdConfig, err := p.convertCommand(cmd)
		if err != nil {
			// Skip invalid commands but continue
			continue
		}
		cfg.Commands[name] = cmdConfig
	}

	// Convert path configurations for monorepo
	for _, pathCfg := range aiResp.Paths {
		path := &pkgconfig.PathConfig{
			Path:     pathCfg.Path,
			Commands: make(map[string]*pkgconfig.CommandConfig),
		}

		for name, cmd := range pathCfg.Commands {
			cmdConfig, err := p.convertCommand(cmd)
			if err != nil {
				continue
			}
			path.Commands[name] = cmdConfig
		}

		cfg.Paths = append(cfg.Paths, path)
	}

	return cfg
}

// convertCommand converts an AI command to a CommandConfig
func (p *ResponseParserImpl) convertCommand(cmd *aiCommand) (*pkgconfig.CommandConfig, error) {
	if cmd.Command == "" {
		return nil, fmt.Errorf("command cannot be empty")
	}

	cmdConfig := &pkgconfig.CommandConfig{
		Command:   cmd.Command,
		Args:      cmd.Args,
		ExitCodes: cmd.ExitCodes,
	}

	// Convert error patterns
	for _, pattern := range cmd.ErrorPatterns {
		regexPattern := &pkgconfig.RegexPattern{
			Pattern: pattern.Pattern,
			Flags:   pattern.Flags,
		}

		// Validate the pattern
		if err := regexPattern.Validate(); err != nil {
			// Skip invalid patterns
			continue
		}

		cmdConfig.ErrorPatterns = append(cmdConfig.ErrorPatterns, regexPattern)
	}

	// Set reasonable defaults if not provided
	if len(cmdConfig.ExitCodes) == 0 {
		// Default to treating any non-zero exit code as failure
		cmdConfig.ExitCodes = []int{1}
	}

	return cmdConfig, nil
}

// convertRegexPatterns converts regex patterns to the expected format
func (p *ResponseParserImpl) convertRegexPatterns(patterns []pkgconfig.RegexPattern) []*pkgconfig.RegexPattern {
	result := make([]*pkgconfig.RegexPattern, len(patterns))
	for i := range patterns {
		result[i] = &patterns[i]
	}
	return result
}

// attemptAutoFix tries to fix common validation errors
func (p *ResponseParserImpl) attemptAutoFix(cfg *pkgconfig.Config, err error) *pkgconfig.Config {
	errStr := err.Error()

	// Try to fix timeout issues
	if strings.Contains(errStr, "timeout") {
		for _, cmd := range cfg.Commands {
			if cmd.Timeout < 0 {
				cmd.Timeout = 0
			} else if cmd.Timeout > 3600000 {
				cmd.Timeout = 3600000 // 1 hour max
			}
		}

		// Re-validate
		if err := p.validator.Validate(cfg); err == nil {
			return cfg
		}
	}

	// Try to fix regex pattern issues
	if strings.Contains(errStr, "regex") || strings.Contains(errStr, "pattern") {
		for _, cmd := range cfg.Commands {
			// Remove invalid patterns
			validPatterns := []*pkgconfig.RegexPattern{}
			for _, pattern := range cmd.ErrorPatterns {
				if err := pattern.Validate(); err == nil {
					validPatterns = append(validPatterns, pattern)
				}
			}
			cmd.ErrorPatterns = validPatterns

			// Do the same for include patterns
			validInclude := []*pkgconfig.RegexPattern{}
			for _, pattern := range cmd.IncludePatterns {
				if err := pattern.Validate(); err == nil {
					validInclude = append(validInclude, pattern)
				}
			}
			cmd.IncludePatterns = validInclude
		}

		// Re-validate
		if err := p.validator.Validate(cfg); err == nil {
			return cfg
		}
	}

	return nil
}
