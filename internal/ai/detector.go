// Package ai provides AI-powered configuration generation for qualhook.
package ai

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/bebsworthy/qualhook/internal/executor"
)

// commandExecutor is an interface for executing commands
type commandExecutor interface {
	Execute(command string, args []string, options executor.ExecOptions) (*executor.ExecResult, error)
}

// toolDetector implements the ToolDetector interface
type toolDetector struct {
	executor      commandExecutor
	detectedTools []Tool
	lastDetection time.Time
	cacheDuration time.Duration
}

// NewToolDetector creates a new tool detector
func NewToolDetector(executor commandExecutor) ToolDetector {
	return &toolDetector{
		executor:      executor,
		cacheDuration: 5 * time.Minute, // Cache results for 5 minutes
	}
}

// DetectTools returns all available AI tools
func (d *toolDetector) DetectTools() ([]Tool, error) {
	// Return cached results if still valid
	if d.isCacheValid() {
		return d.detectedTools, nil
	}

	// Use concurrent detection for better performance
	tools := make([]Tool, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	// Check for Claude CLI concurrently
	go func() {
		defer wg.Done()
		tools[0] = d.detectClaude()
	}()

	// Check for Gemini CLI concurrently
	go func() {
		defer wg.Done()
		tools[1] = d.detectGemini()
	}()

	wg.Wait()

	// Update cache
	d.detectedTools = tools
	d.lastDetection = time.Now()

	return tools, nil
}

// IsToolAvailable checks if a specific tool is available
func (d *toolDetector) IsToolAvailable(toolName string) (bool, error) {
	tools, err := d.DetectTools()
	if err != nil {
		return false, err
	}

	toolNameLower := strings.ToLower(toolName)
	for _, tool := range tools {
		if strings.ToLower(tool.Name) == toolNameLower && tool.Available {
			return true, nil
		}
	}

	return false, nil
}

// isCacheValid checks if the cache is still valid
func (d *toolDetector) isCacheValid() bool {
	if d.detectedTools == nil {
		return false
	}
	return time.Since(d.lastDetection) < d.cacheDuration
}

// detectClaude detects the Claude CLI tool
func (d *toolDetector) detectClaude() Tool {
	return d.detectTool("claude", "claude")
}

// detectGemini detects the Gemini CLI tool
func (d *toolDetector) detectGemini() Tool {
	return d.detectTool("gemini", "gemini")
}

// detectTool is a generic tool detector for CLI tools
func (d *toolDetector) detectTool(name, command string) Tool {
	tool := Tool{
		Name:    name,
		Command: command,
	}

	// Try to get version
	options := executor.ExecOptions{
		Timeout:    5 * time.Second,
		InheritEnv: true,
	}

	// Try command --version
	result, err := d.executor.Execute(command, []string{"--version"}, options)
	if err == nil && result.ExitCode == 0 {
		tool.Available = true
		tool.Version = extractVersion(result.Stdout)
		return tool
	}

	// Try command version (without --)
	result, err = d.executor.Execute(command, []string{"version"}, options)
	if err == nil && result.ExitCode == 0 {
		tool.Available = true
		tool.Version = extractVersion(result.Stdout)
		return tool
	}

	// If both version commands fail, check if we can at least run the command
	// This handles the case where the tool exists but doesn't have a version flag
	result, err = d.executor.Execute(command, []string{"--help"}, options)
	if err == nil && result.ExitCode == 0 {
		tool.Available = true
		return tool
	}

	// Command not available
	tool.Available = false
	return tool
}

// extractVersion extracts version number from version output
func extractVersion(output string) string {
	// Look for semantic version patterns
	versionPattern := regexp.MustCompile(`v?(\d+\.\d+\.\d+(?:-[a-zA-Z0-9.-]+)?(?:\+[a-zA-Z0-9.-]+)?)`)
	matches := versionPattern.FindStringSubmatch(output)
	if len(matches) > 1 {
		return matches[1]
	}

	// Look for simple version patterns (e.g., "1.2")
	simplePattern := regexp.MustCompile(`v?(\d+\.\d+)`)
	matches = simplePattern.FindStringSubmatch(output)
	if len(matches) > 1 {
		return matches[1]
	}

	// Clean up output to use as version if no pattern matches
	output = strings.TrimSpace(output)
	lines := strings.Split(output, "\n")
	if len(lines) > 0 {
		// Take first line and remove common prefixes
		firstLine := lines[0]
		for _, prefix := range []string{"claude ", "gemini ", "version ", "v"} {
			firstLine = strings.TrimPrefix(strings.ToLower(firstLine), prefix)
		}
		firstLine = strings.TrimSpace(firstLine)
		if firstLine != "" && len(firstLine) < 20 { // Reasonable version string length
			return firstLine
		}
	}

	return ""
}

// GetAvailableTools returns only the tools that are available
func GetAvailableTools(tools []Tool) []Tool {
	available := []Tool{}
	for _, tool := range tools {
		if tool.Available {
			available = append(available, tool)
		}
	}
	return available
}

// FormatToolsStatus formats the status of detected tools for display
func FormatToolsStatus(tools []Tool) string {
	var status strings.Builder

	status.WriteString("AI Tool Detection Results:\n")
	for _, tool := range tools {
		status.WriteString(fmt.Sprintf("\n%s:\n", tool.Name))
		if tool.Available {
			status.WriteString("  Status: ✓ Available\n")
			if tool.Version != "" {
				status.WriteString(fmt.Sprintf("  Version: %s\n", tool.Version))
			}
			status.WriteString(fmt.Sprintf("  Command: %s\n", tool.Command))
		} else {
			status.WriteString("  Status: ✗ Not found\n")
			status.WriteString("  Install: Run 'qualhook ai-config' for installation instructions\n")
		}
	}

	return status.String()
}
