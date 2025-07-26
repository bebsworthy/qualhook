//go:build test || unit || integration || e2e

package executor

// Test command constants
const (
	// Operating system constants
	osWindows = "windows"

	// Command constants
	echoCommand = "echo"
	cmdCommand  = "cmd"
	shCommand   = "sh"

	// Command arguments
	cmdArgC = "/C"
	shArgC  = "-c"
)
