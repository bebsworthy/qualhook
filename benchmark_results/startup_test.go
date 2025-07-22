package main

import (
    "fmt"
    "os/exec"
    "time"
)

func main() {
    // Test help command startup
    times := make([]time.Duration, 10)
    for i := 0; i < 10; i++ {
        start := time.Now()
        cmd := exec.Command("./qualhook", "--help")
        _ = cmd.Run()
        times[i] = time.Since(start)
    }
    
    // Calculate average
    var total time.Duration
    for _, t := range times {
        total += t
    }
    avg := total / 10
    
    fmt.Printf("Average startup time (--help): %v\n", avg)
    fmt.Printf("Average startup time (ms): %.2f\n", float64(avg.Nanoseconds())/1e6)
}
