package ai

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockProgressIndicator is a test implementation that doesn't use terminal I/O
type mockProgressIndicator struct {
	mu         sync.Mutex
	message    string
	running    bool
	startTime  time.Time
	output     *bytes.Buffer
	cancelChan chan bool
}

func newMockProgressIndicator() *mockProgressIndicator {
	return &mockProgressIndicator{
		output:     &bytes.Buffer{},
		cancelChan: make(chan bool, 1),
	}
}

func (m *mockProgressIndicator) Start(message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.message = message
	m.running = true
	m.startTime = time.Now()
}

func (m *mockProgressIndicator) Update(message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.message = message
}

func (m *mockProgressIndicator) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.running = false
}

func (m *mockProgressIndicator) WaitForCancellation(ctx context.Context) <-chan bool {
	return m.cancelChan
}

func (m *mockProgressIndicator) isRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

func (m *mockProgressIndicator) getMessage() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.message
}

func TestProgressIndicator_StartStop(t *testing.T) {
	p := newMockProgressIndicator()

	// Test initial state
	if p.isRunning() {
		t.Error("Progress indicator should not be running initially")
	}

	// Test Start
	p.Start("Test message")
	if !p.isRunning() {
		t.Error("Progress indicator should be running after Start")
	}
	if p.getMessage() != "Test message" {
		t.Errorf("Expected message 'Test message', got '%s'", p.getMessage())
	}

	// Test Stop
	p.Stop()
	if p.isRunning() {
		t.Error("Progress indicator should not be running after Stop")
	}
}

func TestProgressIndicator_Update(t *testing.T) {
	p := newMockProgressIndicator()

	p.Start("Initial message")
	if p.getMessage() != "Initial message" {
		t.Errorf("Expected message 'Initial message', got '%s'", p.getMessage())
	}

	p.Update("Updated message")
	if p.getMessage() != "Updated message" {
		t.Errorf("Expected message 'Updated message', got '%s'", p.getMessage())
	}
}

func TestProgressIndicator_MultipleStartStop(t *testing.T) {
	p := newMockProgressIndicator()

	// First cycle
	p.Start("First")
	if !p.isRunning() {
		t.Error("Should be running after first Start")
	}
	p.Stop()
	if p.isRunning() {
		t.Error("Should not be running after first Stop")
	}

	// Second cycle
	p.Start("Second")
	if !p.isRunning() {
		t.Error("Should be running after second Start")
	}
	if p.getMessage() != "Second" {
		t.Errorf("Expected message 'Second', got '%s'", p.getMessage())
	}
	p.Stop()
	if p.isRunning() {
		t.Error("Should not be running after second Stop")
	}
}

func TestProgressIndicator_WaitForCancellation(t *testing.T) {
	p := newMockProgressIndicator()
	ctx := context.Background()

	cancelChan := p.WaitForCancellation(ctx)

	// Test that we can receive from the channel
	select {
	case p.cancelChan <- true:
		// Successfully sent
	default:
		t.Error("Cancel channel should be buffered")
	}

	// Test that we can receive the cancellation
	select {
	case canceled := <-cancelChan:
		if !canceled {
			t.Error("Expected true from cancel channel")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for cancellation")
	}
}

func TestProgressIndicator_ConcurrentAccess(t *testing.T) {
	p := newMockProgressIndicator()

	var wg sync.WaitGroup

	// Start multiple goroutines that interact with the progress indicator
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()

			// Each goroutine performs multiple operations
			for j := 0; j < 5; j++ {
				p.Start("Concurrent start")
				time.Sleep(time.Millisecond)
				p.Update("Concurrent update")
				time.Sleep(time.Millisecond)
				p.Stop()
			}
		}(i)
	}

	wg.Wait()

	// Final state should be stopped
	if p.isRunning() {
		t.Error("Progress indicator should be stopped after concurrent access")
	}
}

func TestFormatElapsed(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "seconds only",
			duration: 45 * time.Second,
			expected: "00:45",
		},
		{
			name:     "minutes and seconds",
			duration: 2*time.Minute + 30*time.Second,
			expected: "02:30",
		},
		{
			name:     "exactly one hour",
			duration: 1 * time.Hour,
			expected: "01:00:00",
		},
		{
			name:     "hours minutes seconds",
			duration: 2*time.Hour + 15*time.Minute + 45*time.Second,
			expected: "02:15:45",
		},
		{
			name:     "zero duration",
			duration: 0,
			expected: "00:00",
		},
		{
			name:     "sub-second duration",
			duration: 500 * time.Millisecond,
			expected: "00:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatElapsed(tt.duration)
			if result != tt.expected {
				t.Errorf("formatElapsed(%v) = %s, want %s", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestProgressIndicator_RealImplementation(t *testing.T) {
	// Test the real implementation with a mock writer
	var output bytes.Buffer
	p := &progress{
		writer:     &output,
		cancelChan: make(chan bool, 1),
		stopChan:   make(chan struct{}),
		doneChan:   make(chan struct{}),
	}

	// Test that Start initializes properly
	p.Start("Testing real implementation")

	// Give it enough time for the display loop to update
	// The display loop updates every 100ms
	time.Sleep(150 * time.Millisecond)

	// Check that running is set
	p.mu.Lock()
	running := p.running
	p.mu.Unlock()

	if !running {
		t.Error("Real implementation should be running after Start")
	}

	// Test Update
	p.Update("Updated message")

	// Give time for another update cycle
	time.Sleep(150 * time.Millisecond)

	// Stop and verify cleanup
	p.Stop()

	p.mu.Lock()
	finalRunning := p.running
	p.mu.Unlock()

	if finalRunning {
		t.Error("Real implementation should not be running after Stop")
	}

	// Check that some output was generated
	outputStr := output.String()
	if !strings.Contains(outputStr, "Press ESC to cancel") {
		t.Errorf("Output should contain 'Press ESC to cancel', got: %q", outputStr)
	}
}

func TestSpinnerFrames(t *testing.T) {
	// Verify spinner frames are defined
	if len(spinnerFrames) == 0 {
		t.Error("Spinner frames should not be empty")
	}

	// Verify all frames are non-empty
	for i, frame := range spinnerFrames {
		if frame == "" {
			t.Errorf("Spinner frame %d should not be empty", i)
		}
	}
}
