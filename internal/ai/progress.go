// Package ai provides AI-powered configuration generation for qualhook.
package ai

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

// progress implements the ProgressIndicator interface with spinner animation
// and ESC key cancellation support.
type progress struct {
	mu          sync.Mutex
	writer      io.Writer
	message     string
	startTime   time.Time
	running     bool
	cancelChan  chan bool
	stopChan    chan struct{}
	doneChan    chan struct{}
	rawMode     *term.State
	spinnerIdx  int
	lastLineLen int
}

// spinnerFrames defines the animation frames for the spinner
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// NewProgressIndicator creates a new progress indicator
func NewProgressIndicator() ProgressIndicator {
	return &progress{
		writer:     os.Stderr,
		cancelChan: make(chan bool, 1),
		stopChan:   make(chan struct{}),
		doneChan:   make(chan struct{}),
	}
}

// Start begins showing progress with the given message
func (p *progress) Start(message string) {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return
	}
	p.running = true
	p.message = message
	p.startTime = time.Now()
	p.spinnerIdx = 0
	p.lastLineLen = 0
	p.mu.Unlock()

	// Start the display loop in a goroutine
	go p.displayLoop()
}

// Update updates the progress message
func (p *progress) Update(message string) {
	p.mu.Lock()
	p.message = message
	p.mu.Unlock()
}

// Stop stops the progress indicator
func (p *progress) Stop() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.running = false
	p.mu.Unlock()

	// Signal stop and wait for display loop to finish
	close(p.stopChan)
	<-p.doneChan

	// Clear the last line
	p.clearLine()
}

// WaitForCancellation returns a channel that receives true when user cancels
func (p *progress) WaitForCancellation(ctx context.Context) <-chan bool {
	go p.watchForESC(ctx)
	return p.cancelChan
}

// displayLoop runs the spinner animation and updates elapsed time
func (p *progress) displayLoop() {
	defer close(p.doneChan)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.updateDisplay()
		}
	}
}

// updateDisplay updates the progress display with spinner and elapsed time
func (p *progress) updateDisplay() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}

	elapsed := time.Since(p.startTime)
	spinner := spinnerFrames[p.spinnerIdx]
	p.spinnerIdx = (p.spinnerIdx + 1) % len(spinnerFrames)

	// Format elapsed time
	elapsedStr := formatElapsed(elapsed)

	// Build the output line
	line := fmt.Sprintf("\r%s %s [%s] (Press ESC to cancel)", spinner, p.message, elapsedStr)

	// Clear previous line if it was longer
	if len(line) < p.lastLineLen {
		p.clearLineLocked()
	}
	p.lastLineLen = len(line)

	// Write the new line
	fmt.Fprint(p.writer, line) //nolint:errcheck // Display error is non-critical
	p.mu.Unlock()
}

// clearLine clears the current line
func (p *progress) clearLine() {
	p.mu.Lock()
	p.clearLineLocked()
	p.mu.Unlock()
}

// clearLineLocked clears the current line (must be called with lock held)
func (p *progress) clearLineLocked() {
	fmt.Fprintf(p.writer, "\r%s\r", strings.Repeat(" ", p.lastLineLen)) //nolint:errcheck // Display error is non-critical
}

// watchForESC monitors for ESC key presses
func (p *progress) watchForESC(ctx context.Context) {
	// Only watch for ESC if stdin is a terminal
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return
	}

	// Set terminal to raw mode to capture ESC key
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return
	}
	defer func() {
		term.Restore(int(os.Stdin.Fd()), oldState) //nolint:errcheck // Best effort restore
	}()

	p.mu.Lock()
	p.rawMode = oldState
	p.mu.Unlock()

	// Create a buffer for reading input
	buf := make([]byte, 1)

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		default:
			// Set a short timeout for non-blocking read
			if err := os.Stdin.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
				continue
			}

			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				continue
			}

			// Check for ESC key (ASCII 27)
			if buf[0] == 27 {
				select {
				case p.cancelChan <- true:
				default:
					// Channel already has a value, don't block
				}
				return
			}
		}
	}
}

// formatElapsed formats a duration as MM:SS or HH:MM:SS
func formatElapsed(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}
