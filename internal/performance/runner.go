package performance

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// Runner orchestrates performance test execution
type Runner struct {
	tracker *PerformanceTracker
}

// NewRunner creates a new performance test runner
func NewRunner() *Runner {
	return &Runner{
		tracker: NewTracker(),
	}
}

// RunAllTests executes all performance tests and saves results
func (r *Runner) RunAllTests() error {
	fmt.Println("🚀 Starting comprehensive performance tests...")
	fmt.Println("═══════════════════════════════════════════════")

	report := r.tracker.CreateReport()

	// Run baseline tests
	fmt.Println("📊 Running baseline performance tests...")
	if err := r.runBaselineTests(report); err != nil {
		return fmt.Errorf("baseline tests failed: %w", err)
	}

	// Run high-pressure tests
	fmt.Println("🔥 Running high-pressure performance tests...")
	if err := r.runHighPressureTests(report); err != nil {
		return fmt.Errorf("high-pressure tests failed: %w", err)
	}

	// Finalize summary
	r.tracker.FinalizeSummary(report)

	// Save results
	if err := r.tracker.SaveResults(report); err != nil {
		return fmt.Errorf("failed to save results: %w", err)
	}

	// Print summary
	r.tracker.PrintSummary(report)

	// Compare with previous results if available
	fmt.Println("\n🔍 Comparing with previous results...")
	comparison, err := r.tracker.CompareWithPrevious(report)
	if err != nil {
		fmt.Printf("⚠️  No previous results for comparison: %v\n", err)
	} else {
		r.tracker.PrintComparison(comparison)
	}

	fmt.Println("\n✅ Performance testing completed!")
	return nil
}

// RunBaselineOnly runs only baseline tests (faster for development)
func (r *Runner) RunBaselineOnly() error {
	fmt.Println("📊 Running baseline performance tests only...")

	report := r.tracker.CreateReport()

	if err := r.runBaselineTests(report); err != nil {
		return fmt.Errorf("baseline tests failed: %w", err)
	}

	r.tracker.FinalizeSummary(report)

	// Save results
	if err := r.tracker.SaveResults(report); err != nil {
		return fmt.Errorf("failed to save results: %w", err)
	}

	// Print summary
	r.tracker.PrintSummary(report)

	// Compare with previous results if available
	fmt.Println("\n🔍 Comparing with previous results...")
	comparison, err := r.tracker.CompareWithPrevious(report)
	if err != nil {
		fmt.Printf("⚠️  No previous results for comparison: %v\n", err)
	} else {
		r.tracker.PrintComparison(comparison)
	}

	return nil
}

// runBaselineTests executes baseline performance benchmarks
func (r *Runner) runBaselineTests(report *PerformanceReport) error {
	baselineTests := []string{
		"BenchmarkBaseline_SimpleHandler",
		"BenchmarkBaseline_JSONResponse",
		"BenchmarkBaseline_PathParams",
		"BenchmarkBaseline_QueryParams",
		"BenchmarkBaseline_JSONBinding",
		"BenchmarkBaseline_Middleware",
		"BenchmarkBaseline_ErrorHandling",
		"BenchmarkBaseline_LargeJSON",
	}

	for _, test := range baselineTests {
		fmt.Printf("  Running %s...\n", test)
		result, err := r.runSingleBenchmark(test)
		if err != nil {
			fmt.Printf("    ❌ Failed: %v\n", err)
			continue
		}
		r.tracker.AddBenchmarkResult(report, result)
		fmt.Printf("    ✅ %8.1f ns/op, %6d B/op, %4d allocs/op\n",
			result.NsPerOp, result.BytesPerOp, result.AllocsPerOp)
	}

	return nil
}

// runHighPressureTests executes high-pressure performance benchmarks
func (r *Runner) runHighPressureTests(report *PerformanceReport) error {
	highPressureTests := []string{
		"BenchmarkHighPressure_SimpleHandler",
		"BenchmarkHighPressure_JSONResponse",
		"BenchmarkHighPressure_MixedWorkload",
		"BenchmarkRPS_PureLoad",
		"BenchmarkRPS_StressTest",
		"BenchmarkMemoryPressure",
		"BenchmarkConcurrentRoutes",
	}

	// Run tests with different CPU configurations
	cpuCounts := []int{1, 2, 4}
	if runtime.NumCPU() >= 8 {
		cpuCounts = append(cpuCounts, 8)
	}

	for _, test := range highPressureTests {
		for _, cpus := range cpuCounts {
			testName := fmt.Sprintf("%s-cpu%d", test, cpus)
			fmt.Printf("  Running %s...\n", testName)

			result, err := r.runSingleBenchmarkWithCPU(test, cpus)
			if err != nil {
				fmt.Printf("    ❌ Failed: %v\n", err)
				continue
			}

			result.Name = testName
			result.CPUs = cpus
			r.tracker.AddBenchmarkResult(report, result)

			fmt.Printf("    ✅ %8.1f ns/op, %6d B/op, %4d allocs/op",
				result.NsPerOp, result.BytesPerOp, result.AllocsPerOp)
			if result.RequestsPerSec > 0 {
				fmt.Printf(", %.0f req/s", result.RequestsPerSec)
			}
			fmt.Printf("\n")
		}
	}

	return nil
}

// runSingleBenchmark executes a single benchmark test
func (r *Runner) runSingleBenchmark(testName string) (BenchmarkResult, error) {
	return r.runSingleBenchmarkWithCPU(testName, runtime.NumCPU())
}

// runSingleBenchmarkWithCPU executes a benchmark with specific CPU count
func (r *Runner) runSingleBenchmarkWithCPU(testName string, cpus int) (BenchmarkResult, error) {
	cmd := exec.Command("go", "test", "-bench=^"+testName+"$", "-benchmem", "-count=1",
		fmt.Sprintf("-cpu=%d", cpus), ".")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return BenchmarkResult{}, fmt.Errorf("benchmark failed: %w\nOutput: %s", err, output)
	}

	return r.parseBenchmarkOutput(string(output), testName)
}

// parseBenchmarkOutput parses go test benchmark output
func (r *Runner) parseBenchmarkOutput(output, testName string) (BenchmarkResult, error) {
	// More flexible regex to handle different benchmark output formats
	// Standard format: BenchmarkTest-11    2000000    500.0 ns/op    1000 B/op    10 allocs/op
	// Custom format: BenchmarkTest    2000000    500.0 ns/op    16845 avg-latency-ns    557428 req/sec    88.00 workers    6465 B/op    23 allocs/op

	// Look for the main benchmark line
	benchLineRegex := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(testName) + `[^\s]*\s+(\d+)\s+([0-9.]+)\s+ns/op.*?(\d+)\s+B/op\s+(\d+)\s+allocs/op`)

	// Also look for custom metrics anywhere in the line
	reqSecRegex := regexp.MustCompile(`([0-9.]+)\s+req/sec`)
	latencyRegex := regexp.MustCompile(`([0-9.]+)\s+avg-latency-ns`)
	workersRegex := regexp.MustCompile(`([0-9.]+)\s+workers`)

	matches := benchLineRegex.FindStringSubmatch(output)
	if len(matches) < 5 {
		// Try a simpler regex for cases where B/op and allocs/op might be in different positions
		simpleRegex := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(testName) + `[^\s]*\s+(\d+)\s+([0-9.]+)\s+ns/op`)
		simpleMatches := simpleRegex.FindStringSubmatch(output)
		if len(simpleMatches) < 3 {
			return BenchmarkResult{}, fmt.Errorf("could not parse benchmark output: %s", output)
		}

		// Extract what we can and set defaults for missing values
		iterations, _ := strconv.Atoi(simpleMatches[1])
		nsPerOp, _ := strconv.ParseFloat(simpleMatches[2], 64)

		// Try to find B/op and allocs/op separately
		var bytesPerOp, allocsPerOp int64
		if bopMatch := regexp.MustCompile(`(\d+)\s+B/op`).FindStringSubmatch(output); len(bopMatch) >= 2 {
			bytesPerOp, _ = strconv.ParseInt(bopMatch[1], 10, 64)
		}
		if aopMatch := regexp.MustCompile(`(\d+)\s+allocs/op`).FindStringSubmatch(output); len(aopMatch) >= 2 {
			allocsPerOp, _ = strconv.ParseInt(aopMatch[1], 10, 64)
		}

		result := BenchmarkResult{
			Name:        testName,
			NsPerOp:     nsPerOp,
			BytesPerOp:  bytesPerOp,
			AllocsPerOp: allocsPerOp,
			Iterations:  iterations,
			CPUs:        runtime.NumCPU(),
		}

		// Parse custom metrics
		if matches := reqSecRegex.FindStringSubmatch(output); len(matches) >= 2 {
			result.RequestsPerSec, _ = strconv.ParseFloat(matches[1], 64)
		}
		if matches := latencyRegex.FindStringSubmatch(output); len(matches) >= 2 {
			result.AvgLatencyNs, _ = strconv.ParseFloat(matches[1], 64)
		}
		if matches := workersRegex.FindStringSubmatch(output); len(matches) >= 2 {
			workers, _ := strconv.ParseFloat(matches[1], 64)
			result.Workers = int(workers)
		}

		return result, nil
	}

	iterations, _ := strconv.Atoi(matches[1])
	nsPerOp, _ := strconv.ParseFloat(matches[2], 64)
	bytesPerOp, _ := strconv.ParseInt(matches[3], 10, 64)
	allocsPerOp, _ := strconv.ParseInt(matches[4], 10, 64)

	result := BenchmarkResult{
		Name:        testName,
		NsPerOp:     nsPerOp,
		BytesPerOp:  bytesPerOp,
		AllocsPerOp: allocsPerOp,
		Iterations:  iterations,
		CPUs:        runtime.NumCPU(),
	}

	// Parse custom metrics if present
	if matches := reqSecRegex.FindStringSubmatch(output); len(matches) >= 2 {
		result.RequestsPerSec, _ = strconv.ParseFloat(matches[1], 64)
	}
	if matches := latencyRegex.FindStringSubmatch(output); len(matches) >= 2 {
		result.AvgLatencyNs, _ = strconv.ParseFloat(matches[1], 64)
	}
	if matches := workersRegex.FindStringSubmatch(output); len(matches) >= 2 {
		workers, _ := strconv.ParseFloat(matches[1], 64)
		result.Workers = int(workers)
	}

	return result, nil
}

// Interactive mode functions

// RunInteractive starts an interactive performance testing session
func (r *Runner) RunInteractive() error {
	fmt.Println("🎯 Interactive Performance Testing")
	fmt.Println("═══════════════════════════════════")
	fmt.Println("Commands:")
	fmt.Println("  all     - Run all performance tests")
	fmt.Println("  baseline - Run baseline tests only")
	fmt.Println("  pressure - Run high-pressure tests only")
	fmt.Println("  compare  - Compare latest results with previous")
	fmt.Println("  history  - Show test history")
	fmt.Println("  help     - Show this help")
	fmt.Println("  exit     - Exit")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("perf> ")
		if !scanner.Scan() {
			break
		}

		command := strings.TrimSpace(scanner.Text())
		if command == "" {
			continue
		}

		switch command {
		case "all":
			if err := r.RunAllTests(); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}
		case "baseline":
			if err := r.RunBaselineOnly(); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}
		case "pressure":
			if err := r.runHighPressureOnly(); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}
		case "compare":
			if err := r.compareLatest(); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}
		case "history":
			r.showHistory()
		case "help":
			fmt.Println("Available commands: all, baseline, pressure, compare, history, help, exit")
		case "exit", "quit", "q":
			fmt.Println("👋 Goodbye!")
			return nil
		default:
			fmt.Printf("Unknown command: %s (type 'help' for available commands)\n", command)
		}
		fmt.Println()
	}

	return nil
}

func (r *Runner) runHighPressureOnly() error {
	fmt.Println("🔥 Running high-pressure tests only...")

	report := r.tracker.CreateReport()

	if err := r.runHighPressureTests(report); err != nil {
		return fmt.Errorf("high-pressure tests failed: %w", err)
	}

	r.tracker.FinalizeSummary(report)
	r.tracker.PrintSummary(report)

	return nil
}

func (r *Runner) compareLatest() error {
	latest, err := r.tracker.LoadLatest()
	if err != nil {
		return fmt.Errorf("failed to load latest results: %w", err)
	}

	comparison, err := r.tracker.CompareWithPrevious(latest)
	if err != nil {
		return fmt.Errorf("failed to compare results: %w", err)
	}

	r.tracker.PrintComparison(comparison)
	return nil
}

func (r *Runner) showHistory() {
	fmt.Println("📚 Performance Test History")
	fmt.Println("═══════════════════════════")

	files, err := r.tracker.getResultFiles()
	if err != nil {
		fmt.Printf("❌ Error reading history: %v\n", err)
		return
	}

	if len(files) == 0 {
		fmt.Println("No performance test history found.")
		return
	}

	fmt.Printf("Found %d test results:\n", len(files))
	for i, file := range files {
		// Extract timestamp from filename
		parts := strings.Split(file, "_")
		if len(parts) >= 2 {
			timestamp := strings.TrimSuffix(parts[1], ".json")
			timestamp = strings.ReplaceAll(timestamp, "-", ":")
			timestamp = strings.ReplaceAll(timestamp, "_", " ")
			fmt.Printf("  %d. %s\n", i+1, timestamp)
		}
	}
}
