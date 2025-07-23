// Package debug provides debug logging functionality for qualhook.
package debug

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Logger provides debug logging capabilities
type Logger struct {
	enabled bool
	writer  io.Writer
	start   time.Time
}

// Global debug logger instance
var globalLogger = &Logger{
	enabled: false,
	writer:  os.Stderr,
}

// Enable enables debug logging
func Enable() {
	globalLogger.enabled = true
	globalLogger.start = time.Now()
}

// IsEnabled returns whether debug logging is enabled
func IsEnabled() bool {
	return globalLogger.enabled
}

// SetWriter sets the output writer for debug logs
func SetWriter(w io.Writer) {
	globalLogger.writer = w
}

// Log writes a debug message if debugging is enabled
func Log(format string, args ...interface{}) {
	if !globalLogger.enabled {
		return
	}
	
	elapsed := time.Since(globalLogger.start)
	prefix := fmt.Sprintf("[DEBUG %s] ", formatDuration(elapsed))
	message := fmt.Sprintf(format, args...)
	
	// Ensure message ends with newline
	if !strings.HasSuffix(message, "\n") {
		message += "\n"
	}
	
	_, _ = fmt.Fprint(globalLogger.writer, prefix+message)
}

// LogSection writes a section header for better organization
func LogSection(title string) {
	if !globalLogger.enabled {
		return
	}
	
	Log("=== %s ===", title)
}

// LogCommand logs command execution details
func LogCommand(command string, args []string, workingDir string) {
	if !globalLogger.enabled {
		return
	}
	
	LogSection("Command Execution")
	Log("Command: %s", command)
	if len(args) > 0 {
		Log("Arguments: %v", args)
	}
	if workingDir != "" {
		Log("Working Directory: %s", workingDir)
	}
}

// LogTiming logs timing information
func LogTiming(operation string, duration time.Duration) {
	if !globalLogger.enabled {
		return
	}
	
	Log("Timing: %s took %s", operation, formatDuration(duration))
}

// LogPatternMatch logs pattern matching details
func LogPatternMatch(pattern, input string, matched bool) {
	if !globalLogger.enabled {
		return
	}
	
	status := "no match"
	if matched {
		status = "matched"
	}
	
	Log("Pattern: %q against %q - %s", pattern, truncate(input, 80), status)
}

// LogFilterProcess logs the filtering process
func LogFilterProcess(totalLines, matchedLines, outputLines int) {
	if !globalLogger.enabled {
		return
	}
	
	Log("Filter: %d total lines -> %d matched -> %d output", totalLines, matchedLines, outputLines)
}

// LogError logs error details
func LogError(err error, context string) {
	if !globalLogger.enabled {
		return
	}
	
	Log("Error in %s: %v", context, err)
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// truncate truncates a string to the specified length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}