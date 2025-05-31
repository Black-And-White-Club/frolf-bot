package testutils

import (
	"log"
	"sync"
	"time"
)

type TestMetrics struct {
	TestName      string
	SetupTime     time.Duration
	ExecutionTime time.Duration
	CleanupTime   time.Duration
	ContainerOps  int
}

var (
	testMetrics  []TestMetrics
	metricsMutex sync.Mutex
)

// TrackTestPerformance records test performance metrics
func TrackTestPerformance(testName string, setupTime, execTime, cleanupTime time.Duration, containerOps int) {
	metricsMutex.Lock()
	defer metricsMutex.Unlock()

	testMetrics = append(testMetrics, TestMetrics{
		TestName:      testName,
		SetupTime:     setupTime,
		ExecutionTime: execTime,
		CleanupTime:   cleanupTime,
		ContainerOps:  containerOps,
	})
}

// PrintTestMetrics outputs performance statistics
func PrintTestMetrics() {
	metricsMutex.Lock()
	defer metricsMutex.Unlock()

	if len(testMetrics) == 0 {
		return
	}

	log.Println("=== TEST PERFORMANCE METRICS ===")
	totalSetup := time.Duration(0)
	totalExec := time.Duration(0)
	totalCleanup := time.Duration(0)
	totalContainerOps := 0

	for _, metric := range testMetrics {
		log.Printf("Test: %s | Setup: %v | Exec: %v | Cleanup: %v | ContainerOps: %d",
			metric.TestName, metric.SetupTime, metric.ExecutionTime, metric.CleanupTime, metric.ContainerOps)

		totalSetup += metric.SetupTime
		totalExec += metric.ExecutionTime
		totalCleanup += metric.CleanupTime
		totalContainerOps += metric.ContainerOps
	}

	log.Printf("TOTALS | Setup: %v | Exec: %v | Cleanup: %v | ContainerOps: %d",
		totalSetup, totalExec, totalCleanup, totalContainerOps)
}

func TrackTestPhases(testName string, setupFunc, execFunc, cleanupFunc func() error) error {
	startTime := time.Now()

	// Setup phase
	setupStart := time.Now()
	if err := setupFunc(); err != nil {
		return err
	}
	setupTime := time.Since(setupStart)

	// Execution phase
	execStart := time.Now()
	if err := execFunc(); err != nil {
		return err
	}
	execTime := time.Since(execStart)

	// Cleanup phase
	cleanupStart := time.Now()
	if err := cleanupFunc(); err != nil {
		return err
	}
	cleanupTime := time.Since(cleanupStart)

	TrackTestPerformance(testName, setupTime, execTime, cleanupTime, 1)

	totalTime := time.Since(startTime)
	if totalTime > 5*time.Second {
		log.Printf("SLOW TEST: %s took %v", testName, totalTime)
	}

	return nil
}

// Add cleanup function to print metrics at the end
func init() {
	// Print metrics when the package is done
	// This will run when tests finish
	go func() {
		time.Sleep(1 * time.Second)
		PrintTestMetrics()
	}()
}
