// Package executor provides command execution functionality for qualhook.
package executor

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ParallelResult represents the result of a parallel execution
type ParallelResult struct {
	// Individual results keyed by identifier
	Results map[string]*ExecResult
	// Order of execution (for maintaining order in output)
	Order []string
	// Total execution time
	TotalTime time.Duration
	// Whether any command failed
	HasFailures bool
	// Count of successful executions
	SuccessCount int
	// Count of failed executions
	FailureCount int
}

// ParallelCommand represents a command to be executed in parallel
type ParallelCommand struct {
	// Unique identifier for this command
	ID string
	// Command to execute
	Command string
	// Command arguments
	Args []string
	// Execution options
	Options ExecOptions
}

// ProgressCallback is called to report progress during parallel execution
type ProgressCallback func(completed int, total int, currentID string)

// ParallelExecutor executes multiple commands concurrently
type ParallelExecutor struct {
	executor    *CommandExecutor
	maxParallel int
}

// NewParallelExecutor creates a new parallel executor
func NewParallelExecutor(executor *CommandExecutor, maxParallel int) *ParallelExecutor {
	if maxParallel <= 0 {
		maxParallel = 4 // Default parallelism
	}
	return &ParallelExecutor{
		executor:    executor,
		maxParallel: maxParallel,
	}
}

// Execute runs multiple commands in parallel
func (pe *ParallelExecutor) Execute(ctx context.Context, commands []ParallelCommand, progress ProgressCallback) (*ParallelResult, error) {
	if len(commands) == 0 {
		return &ParallelResult{
			Results: make(map[string]*ExecResult),
			Order:   []string{},
		}, nil
	}

	startTime := time.Now()
	
	// Initialize result
	result := &ParallelResult{
		Results: make(map[string]*ExecResult),
		Order:   make([]string, 0, len(commands)),
	}

	// Track order
	for _, cmd := range commands {
		result.Order = append(result.Order, cmd.ID)
	}

	// Create semaphore for limiting parallelism
	semaphore := make(chan struct{}, pe.maxParallel)
	
	// Create wait group for synchronization
	var wg sync.WaitGroup
	
	// Mutex for result map access
	var resultMutex sync.Mutex
	
	// Progress tracking
	var progressMutex sync.Mutex
	completed := 0

	// Execute commands
	for _, cmd := range commands {
		wg.Add(1)
		
		go func(pc ParallelCommand) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			// Check context cancellation
			select {
			case <-ctx.Done():
				resultMutex.Lock()
				result.Results[pc.ID] = &ExecResult{
					ExitCode: -1,
					Error:    ctx.Err(),
				}
				resultMutex.Unlock()
				return
			default:
			}
			
			// Execute command
			execResult, err := pe.executor.Execute(pc.Command, pc.Args, pc.Options)
			if err != nil {
				execResult = &ExecResult{
					ExitCode: -1,
					Error:    err,
				}
			}
			
			// Store result
			resultMutex.Lock()
			result.Results[pc.ID] = execResult
			resultMutex.Unlock()
			
			// Update progress
			if progress != nil {
				progressMutex.Lock()
				completed++
				currentCompleted := completed
				progressMutex.Unlock()
				
				progress(currentCompleted, len(commands), pc.ID)
			}
		}(cmd)
	}
	
	// Wait for all commands to complete
	wg.Wait()
	
	// Calculate statistics
	result.TotalTime = time.Since(startTime)
	for _, execResult := range result.Results {
		if execResult.Error != nil || execResult.ExitCode != 0 || execResult.TimedOut {
			result.FailureCount++
			result.HasFailures = true
		} else {
			result.SuccessCount++
		}
	}
	
	return result, nil
}

// ExecuteWithAggregation runs commands and aggregates output
func (pe *ParallelExecutor) ExecuteWithAggregation(ctx context.Context, commands []ParallelCommand, progress ProgressCallback) (*AggregatedResult, error) {
	// Execute commands in parallel
	parallelResult, err := pe.Execute(ctx, commands, progress)
	if err != nil {
		return nil, err
	}
	
	// Aggregate results
	aggregated := &AggregatedResult{
		ParallelResult: parallelResult,
		CombinedStdout: make([]string, 0),
		CombinedStderr: make([]string, 0),
		FailedCommands: make([]string, 0),
	}
	
	// Collect output in order
	for _, id := range parallelResult.Order {
		execResult, ok := parallelResult.Results[id]
		if !ok {
			continue
		}
		
		// Add stdout if present
		if execResult.Stdout != "" {
			aggregated.CombinedStdout = append(aggregated.CombinedStdout, 
				fmt.Sprintf("=== %s ===\n%s", id, execResult.Stdout))
		}
		
		// Add stderr if present
		if execResult.Stderr != "" {
			aggregated.CombinedStderr = append(aggregated.CombinedStderr,
				fmt.Sprintf("=== %s ===\n%s", id, execResult.Stderr))
		}
		
		// Track failures
		if execResult.Error != nil || execResult.ExitCode != 0 || execResult.TimedOut {
			aggregated.FailedCommands = append(aggregated.FailedCommands, id)
		}
	}
	
	return aggregated, nil
}

// AggregatedResult contains aggregated results from parallel execution
type AggregatedResult struct {
	*ParallelResult
	// Combined stdout from all commands (in order)
	CombinedStdout []string
	// Combined stderr from all commands (in order)
	CombinedStderr []string
	// List of command IDs that failed
	FailedCommands []string
}

// GetFailureSummary returns a summary of failures
func (ar *AggregatedResult) GetFailureSummary() string {
	if !ar.HasFailures {
		return ""
	}
	
	summary := fmt.Sprintf("Failed commands (%d/%d):\n", ar.FailureCount, len(ar.Order))
	for _, id := range ar.FailedCommands {
		if result, ok := ar.Results[id]; ok {
			if result.TimedOut {
				summary += fmt.Sprintf("  - %s: timed out\n", id)
			} else if result.Error != nil {
				summary += fmt.Sprintf("  - %s: %v\n", id, result.Error)
			} else {
				summary += fmt.Sprintf("  - %s: exit code %d\n", id, result.ExitCode)
			}
		}
	}
	
	return summary
}

// ParallelExecutorPool manages a pool of parallel executors
type ParallelExecutorPool struct {
	executors []*ParallelExecutor
	current   int
	mu        sync.Mutex
}

// NewParallelExecutorPool creates a pool of parallel executors
func NewParallelExecutorPool(size int, commandExecutor *CommandExecutor, maxParallel int) *ParallelExecutorPool {
	if size <= 0 {
		size = 1
	}
	
	pool := &ParallelExecutorPool{
		executors: make([]*ParallelExecutor, size),
	}
	
	for i := 0; i < size; i++ {
		pool.executors[i] = NewParallelExecutor(commandExecutor, maxParallel)
	}
	
	return pool
}

// Get returns the next available executor
func (p *ParallelExecutorPool) Get() *ParallelExecutor {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	executor := p.executors[p.current]
	p.current = (p.current + 1) % len(p.executors)
	
	return executor
}