// Package security provides resource limiting functionality
package security

import (
	"fmt"
	"io"
	"sync/atomic"
	"time"
)

// ResourceLimits defines limits for resource consumption
type ResourceLimits struct {
	// MaxOutputSize is the maximum allowed output size in bytes
	MaxOutputSize int64
	// MaxMemory is the maximum allowed memory usage in bytes
	MaxMemory int64
	// MaxCPUTime is the maximum allowed CPU time
	MaxCPUTime time.Duration
	// MaxFileHandles is the maximum number of open file handles
	MaxFileHandles int
}

// DefaultLimits returns default resource limits
func DefaultLimits() *ResourceLimits {
	return &ResourceLimits{
		MaxOutputSize:  10 * 1024 * 1024,   // 10MB
		MaxMemory:      1024 * 1024 * 1024, // 1GB
		MaxCPUTime:     5 * time.Minute,
		MaxFileHandles: 100,
	}
}

// StrictLimits returns strict resource limits
func StrictLimits() *ResourceLimits {
	return &ResourceLimits{
		MaxOutputSize:  1 * 1024 * 1024,   // 1MB
		MaxMemory:      256 * 1024 * 1024, // 256MB
		MaxCPUTime:     1 * time.Minute,
		MaxFileHandles: 10,
	}
}

// LimitedWriter wraps an io.Writer to enforce output size limits
type LimitedWriter struct {
	writer   io.Writer
	limit    int64
	written  int64
	exceeded atomic.Bool
}

// NewLimitedWriter creates a new limited writer
func NewLimitedWriter(w io.Writer, limit int64) *LimitedWriter {
	return &LimitedWriter{
		writer: w,
		limit:  limit,
	}
}

// Write implements io.Writer with size limiting
func (lw *LimitedWriter) Write(p []byte) (int, error) {
	// Check if already exceeded
	if lw.exceeded.Load() {
		return 0, fmt.Errorf("output size limit exceeded (%d bytes)", lw.limit)
	}

	// Check if this write would exceed the limit
	newTotal := atomic.AddInt64(&lw.written, int64(len(p)))
	if newTotal > lw.limit {
		lw.exceeded.Store(true)
		// Calculate how much we can write
		canWrite := lw.limit - (newTotal - int64(len(p)))
		if canWrite <= 0 {
			return 0, fmt.Errorf("output size limit exceeded (%d bytes)", lw.limit)
		}
		// Write partial data up to the limit
		n, err := lw.writer.Write(p[:canWrite])
		if err != nil {
			return n, err
		}
		return n, fmt.Errorf("output size limit exceeded (%d bytes)", lw.limit)
	}

	// Write the full data
	return lw.writer.Write(p)
}

// Written returns the number of bytes written
func (lw *LimitedWriter) Written() int64 {
	return atomic.LoadInt64(&lw.written)
}

// Exceeded returns whether the limit has been exceeded
func (lw *LimitedWriter) Exceeded() bool {
	return lw.exceeded.Load()
}

// RateLimiter provides rate limiting for operations
type RateLimiter struct {
	// Maximum operations per second
	maxOpsPerSecond int
	// Time window for rate limiting
	window time.Duration
	// Current operation count
	operations int64
	// Window start time
	windowStart time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxOpsPerSecond int) *RateLimiter {
	return &RateLimiter{
		maxOpsPerSecond: maxOpsPerSecond,
		window:          time.Second,
		windowStart:     time.Now(),
	}
}

// Allow checks if an operation is allowed under the rate limit
func (rl *RateLimiter) Allow() bool {
	now := time.Now()

	// Check if we need to reset the window
	if now.Sub(rl.windowStart) >= rl.window {
		atomic.StoreInt64(&rl.operations, 0)
		rl.windowStart = now
	}

	// Check if we're under the limit
	ops := atomic.AddInt64(&rl.operations, 1)
	return ops <= int64(rl.maxOpsPerSecond)
}

// Wait waits until an operation is allowed
func (rl *RateLimiter) Wait() {
	for !rl.Allow() {
		// Calculate remaining time in window
		remaining := rl.window - time.Since(rl.windowStart)
		if remaining > 0 {
			time.Sleep(remaining)
		}
	}
}

// CommandRateLimiter limits the rate of command execution
type CommandRateLimiter struct {
	// Rate limiters per command
	limiters map[string]*RateLimiter
	// Default rate limit
	defaultLimit int
}

// NewCommandRateLimiter creates a new command rate limiter
func NewCommandRateLimiter(defaultLimit int) *CommandRateLimiter {
	return &CommandRateLimiter{
		limiters:     make(map[string]*RateLimiter),
		defaultLimit: defaultLimit,
	}
}

// Allow checks if a command execution is allowed
func (crl *CommandRateLimiter) Allow(command string) bool {
	limiter, exists := crl.limiters[command]
	if !exists {
		limiter = NewRateLimiter(crl.defaultLimit)
		crl.limiters[command] = limiter
	}
	return limiter.Allow()
}

// Wait waits until a command execution is allowed
func (crl *CommandRateLimiter) Wait(command string) {
	limiter, exists := crl.limiters[command]
	if !exists {
		limiter = NewRateLimiter(crl.defaultLimit)
		crl.limiters[command] = limiter
	}
	limiter.Wait()
}

// SetLimit sets the rate limit for a specific command
func (crl *CommandRateLimiter) SetLimit(command string, maxOpsPerSecond int) {
	crl.limiters[command] = NewRateLimiter(maxOpsPerSecond)
}
