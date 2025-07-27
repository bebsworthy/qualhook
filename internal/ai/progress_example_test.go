package ai_test

import (
	"context"
	"fmt"
	"time"

	"github.com/bebsworthy/qualhook/internal/ai"
)

// ExampleProgressIndicator demonstrates how to use the progress indicator
func ExampleProgressIndicator() {
	// Create a new progress indicator
	progress := ai.NewProgressIndicator()

	// Start showing progress
	progress.Start("Analyzing project")

	// Set up cancellation monitoring
	ctx := context.Background()
	cancelChan := progress.WaitForCancellation(ctx)

	// Simulate work with the ability to cancel
	done := make(chan bool)
	go func() {
		// Simulate different phases of work
		time.Sleep(500 * time.Millisecond)
		progress.Update("Detecting project type")

		time.Sleep(500 * time.Millisecond)
		progress.Update("Generating configuration")

		time.Sleep(500 * time.Millisecond)
		done <- true
	}()

	// Wait for either completion or cancellation
	select {
	case <-done:
		fmt.Println("Work completed")
	case <-cancelChan:
		fmt.Println("Operation canceled by user")
	}

	// Always stop the progress indicator
	progress.Stop()

	// Output:
	// Work completed
}
