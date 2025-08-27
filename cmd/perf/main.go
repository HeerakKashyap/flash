package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/goflash/flash/v2/internal/performance"
)

func main() {
	var (
		all         = flag.Bool("all", false, "Run all performance tests")
		baseline    = flag.Bool("baseline", false, "Run baseline tests only")
		compare     = flag.Bool("compare", false, "Compare latest results with previous")
		interactive = flag.Bool("interactive", false, "Start interactive mode")
		help        = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	if *help {
		printHelp()
		return
	}

	runner := performance.NewRunner()

	switch {
	case *interactive:
		if err := runner.RunInteractive(); err != nil {
			fmt.Fprintf(os.Stderr, "Interactive mode failed: %v\n", err)
			os.Exit(1)
		}
	case *all:
		if err := runner.RunAllTests(); err != nil {
			fmt.Fprintf(os.Stderr, "Performance tests failed: %v\n", err)
			os.Exit(1)
		}
	case *baseline:
		if err := runner.RunBaselineOnly(); err != nil {
			fmt.Fprintf(os.Stderr, "Baseline tests failed: %v\n", err)
			os.Exit(1)
		}
	case *compare:
		// Load latest and compare
		tracker := performance.NewTracker()
		latest, err := tracker.LoadLatest()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load latest results: %v\n", err)
			os.Exit(1)
		}
		comparison, err := tracker.CompareWithPrevious(latest)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to compare results: %v\n", err)
			os.Exit(1)
		}
		tracker.PrintComparison(comparison)
	default:
		// Default to running all tests
		if err := runner.RunAllTests(); err != nil {
			fmt.Fprintf(os.Stderr, "Performance tests failed: %v\n", err)
			os.Exit(1)
		}
	}
}

func printHelp() {
	fmt.Println("GoFlash Performance Testing Tool")
	fmt.Println("═══════════════════════════════════")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go run cmd/perf/main.go [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -all           Run all performance tests (default)")
	fmt.Println("  -baseline      Run baseline tests only (faster)")
	fmt.Println("  -pressure      Run high-pressure tests only")
	fmt.Println("  -compare       Compare latest results with previous")
	fmt.Println("  -interactive   Start interactive mode")
	fmt.Println("  -help          Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run cmd/perf/main.go                    # Run all tests")
	fmt.Println("  go run cmd/perf/main.go -baseline          # Quick baseline check")
	fmt.Println("  go run cmd/perf/main.go -compare           # Compare with previous")
	fmt.Println("  go run cmd/perf/main.go -interactive       # Interactive mode")
	fmt.Println()
	fmt.Println("Results are saved in:")
	fmt.Println("  internal/performance/results/")
}
