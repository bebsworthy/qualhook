package testutil

import (
	"bytes"
	"io"
	"os"
	"sync"
)

// CaptureOutput captures stdout and stderr output from a function.
// This is a simple implementation suitable for most test cases.
func CaptureOutput(fn func()) (stdout, stderr string, err error) {
	// Save current stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// Create pipes
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return "", "", err
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		return "", "", err
	}

	// Redirect stdout and stderr to our pipes
	os.Stdout = stdoutW
	os.Stderr = stderrW

	// Capture output in goroutines
	stdoutChan := make(chan string)
	stderrChan := make(chan string)

	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, stdoutR) //nolint:errcheck
		stdoutChan <- buf.String()
	}()

	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, stderrR) //nolint:errcheck
		stderrChan <- buf.String()
	}()

	// Run the function
	fn()

	// Close write ends
	_ = stdoutW.Close() //nolint:errcheck
	_ = stderrW.Close() //nolint:errcheck

	// Restore original stdout and stderr
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	// Get captured output
	stdout = <-stdoutChan
	stderr = <-stderrChan

	return stdout, stderr, nil
}

// CaptureStdout is a convenience function that captures only stdout.
func CaptureStdout(fn func()) (string, error) {
	stdout, _, err := CaptureOutput(fn)
	return stdout, err
}

// CaptureStderr is a convenience function that captures only stderr.
func CaptureStderr(fn func()) (string, error) {
	_, stderr, err := CaptureOutput(fn)
	return stderr, err
}

// TestWriter provides a simple io.Writer for tests.
type TestWriter struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

// NewTestWriter creates a new TestWriter.
func NewTestWriter() *TestWriter {
	return &TestWriter{}
}

// Write implements io.Writer.
func (tw *TestWriter) Write(p []byte) (n int, err error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	return tw.buf.Write(p)
}

// String returns the written content.
func (tw *TestWriter) String() string {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	return tw.buf.String()
}

// Reset clears the buffer.
func (tw *TestWriter) Reset() {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.buf.Reset()
}

// Bytes returns the written content as bytes.
func (tw *TestWriter) Bytes() []byte {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	return tw.buf.Bytes()
}