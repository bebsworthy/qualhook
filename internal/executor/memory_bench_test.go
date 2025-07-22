// Package executor provides command execution functionality for qualhook.
package executor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// createLargeOutputScript creates a script that generates large output
func createLargeOutputScript(b *testing.B, lines int, errorRate float64) string {
	tmpDir := b.TempDir()
	scriptPath := filepath.Join(tmpDir, "generate_output.sh")
	
	errorInterval := int(1.0 / errorRate)
	if errorRate == 0 {
		errorInterval = lines + 1
	}
	
	script := fmt.Sprintf(`#!/bin/bash
for i in $(seq 1 %d); do
    if [ $((i %% %d)) -eq 0 ]; then
        echo "ERROR: Failed at line $i" >&2
    else
        echo "INFO: Processing line $i"
    fi
done
`, lines, errorInterval)
	
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	if err != nil {
		b.Fatal(err)
	}
	
	return scriptPath
}

// BenchmarkLargeOutputMemory measures memory usage with large command outputs
func BenchmarkLargeOutputMemory(b *testing.B) {
	exec := NewCommandExecutor(30 * time.Second)
	
	testCases := []struct {
		name      string
		lines     int
		errorRate float64
	}{
		{"SmallOutput", 100, 0.1},
		{"MediumOutput", 1000, 0.1},
		{"LargeOutput", 10000, 0.01},
		{"VeryLargeOutput", 50000, 0.001},
		{"HugeOutput", 100000, 0.0001},
	}
	
	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			scriptPath := createLargeOutputScript(b, tc.lines, tc.errorRate)
			
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				opts := ExecOptions{
					Timeout: 30 * time.Second,
				}
				_, _ = exec.Execute("/bin/bash", []string{scriptPath}, opts)
			}
		})
	}
}

// BenchmarkStreamingMemory measures memory usage with streaming vs buffering
func BenchmarkStreamingMemory(b *testing.B) {
	exec := NewCommandExecutor(10 * time.Second)
	
	// Create a script that generates continuous output
	tmpDir := b.TempDir()
	streamScript := filepath.Join(tmpDir, "stream.sh")
	script := `#!/bin/bash
for i in {1..1000}; do
    echo "Line $i: $(head -c 100 /dev/urandom | base64)"
    if [ $((i % 100)) -eq 0 ]; then
        echo "ERROR: Checkpoint at line $i" >&2
    fi
done
`
	os.WriteFile(streamScript, []byte(script), 0755)
	
	b.Run("BufferedExecution", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			opts := ExecOptions{
				Timeout: 10 * time.Second,
			}
			_, _ = exec.Execute("/bin/bash", []string{streamScript}, opts)
		}
	})
	
	b.Run("StreamingExecution", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			opts := ExecOptions{
				Timeout: 10 * time.Second,
			}
			var stdoutBuf, stderrBuf strings.Builder
			result, _ := exec.ExecuteWithStreaming("/bin/bash", []string{streamScript}, opts, &stdoutBuf, &stderrBuf)
			// Consume the output
			_ = result
		}
	})
}

// BenchmarkParallelMemoryUsage measures memory usage under parallel execution
func BenchmarkParallelMemoryUsage(b *testing.B) {
	cmdExec := NewCommandExecutor(30 * time.Second)
	pe := NewParallelExecutor(cmdExec, 4) // 4 concurrent workers
	
	// Create multiple scripts with different outputs
	scripts := make([]string, 4)
	for i := range scripts {
		scripts[i] = createLargeOutputScript(b, 1000*(i+1), 0.1)
	}
	
	b.Run("ParallelSmallOutputs", func(b *testing.B) {
		commands := make([]ParallelCommand, 4)
		for i := range commands {
			commands[i] = ParallelCommand{
				ID:      fmt.Sprintf("cmd_%d", i),
				Command: "echo",
				Args:    []string{fmt.Sprintf("Output from command %d", i)},
				Options: ExecOptions{
					Timeout: 5 * time.Second,
				},
			}
		}
		
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = pe.Execute(context.Background(), commands, nil)
		}
	})
	
	b.Run("ParallelLargeOutputs", func(b *testing.B) {
		commands := make([]ParallelCommand, len(scripts))
		for i, script := range scripts {
			commands[i] = ParallelCommand{
				ID:      fmt.Sprintf("script_%d", i),
				Command: "/bin/bash",
				Args:    []string{script},
				Options: ExecOptions{
					Timeout: 10 * time.Second,
				},
			}
		}
		
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = pe.Execute(context.Background(), commands, nil)
		}
	})
}

// BenchmarkOutputBuffering measures different buffer sizes' impact on memory
func BenchmarkOutputBuffering(b *testing.B) {
	// Generate a large string to simulate command output
	largeOutput := strings.Repeat("This is a line of output that simulates real command output.\n", 10000)
	
	testCases := []struct {
		name       string
		bufferSize int
	}{
		{"SmallBuffer_1KB", 1024},
		{"MediumBuffer_64KB", 64 * 1024},
		{"LargeBuffer_1MB", 1024 * 1024},
		{"VeryLargeBuffer_10MB", 10 * 1024 * 1024},
	}
	
	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Simulate buffering the output
				buffer := make([]byte, 0, tc.bufferSize)
				for j := 0; j < len(largeOutput); j += tc.bufferSize {
					end := j + tc.bufferSize
					if end > len(largeOutput) {
						end = len(largeOutput)
					}
					buffer = append(buffer[:0], largeOutput[j:end]...)
				}
			}
		})
	}
}

// BenchmarkMemoryCleanup measures GC pressure from command execution
func BenchmarkMemoryCleanup(b *testing.B) {
	exec := NewCommandExecutor(5 * time.Second)
	
	b.Run("ManySmallCommands", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Execute many small commands
			for j := 0; j < 100; j++ {
				opts := ExecOptions{
					Timeout: 1 * time.Second,
				}
				_, _ = exec.Execute("echo", []string{fmt.Sprintf("test %d", j)}, opts)
			}
		}
	})
	
	b.Run("FewLargeCommands", func(b *testing.B) {
		scriptPath := createLargeOutputScript(b, 10000, 0.01)
		
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Execute few commands with large output
			for j := 0; j < 5; j++ {
				opts := ExecOptions{
					Timeout: 10 * time.Second,
				}
				_, _ = exec.Execute("/bin/bash", []string{scriptPath}, opts)
			}
		}
	})
}

// BenchmarkWorstCaseMemory tests memory usage in worst-case scenarios
func BenchmarkWorstCaseMemory(b *testing.B) {
	exec := NewCommandExecutor(30 * time.Second)
	
	b.Run("AllErrorOutput", func(b *testing.B) {
		// Script that outputs everything to stderr
		tmpDir := b.TempDir()
		scriptPath := filepath.Join(tmpDir, "all_errors.sh")
		script := `#!/bin/bash
for i in {1..1000}; do
    echo "ERROR: Failed operation at line $i with a very long error message that contains lots of details" >&2
done
`
		os.WriteFile(scriptPath, []byte(script), 0755)
		
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			opts := ExecOptions{
				Timeout: 10 * time.Second,
			}
			_, _ = exec.Execute("/bin/bash", []string{scriptPath}, opts)
		}
	})
	
	b.Run("RapidSmallOutputs", func(b *testing.B) {
		// Script that generates many small outputs rapidly
		tmpDir := b.TempDir()
		scriptPath := filepath.Join(tmpDir, "rapid_output.sh")
		script := `#!/bin/bash
for i in {1..10000}; do
    echo "$i"
done
`
		os.WriteFile(scriptPath, []byte(script), 0755)
		
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			opts := ExecOptions{
				Timeout: 10 * time.Second,
			}
			_, _ = exec.Execute("/bin/bash", []string{scriptPath}, opts)
		}
	})
}