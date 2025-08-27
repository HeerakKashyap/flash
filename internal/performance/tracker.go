package performance

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

const (
	ResultsDir     = "results"
	LatestFile     = "latest.json"
	ComparisonFile = "comparison.json"
)

// PerformanceTracker handles saving and comparing performance results
type PerformanceTracker struct {
	resultsDir string
}

// NewTracker creates a new performance tracker
func NewTracker() *PerformanceTracker {
	return &PerformanceTracker{
		resultsDir: ResultsDir,
	}
}

// SaveResults saves the performance report with timestamp
func (pt *PerformanceTracker) SaveResults(report *PerformanceReport) error {
	// Ensure results directory exists
	if err := os.MkdirAll(pt.resultsDir, 0755); err != nil {
		return fmt.Errorf("failed to create results directory: %w", err)
	}

	// Save with timestamp filename
	timestamp := report.Timestamp.Format("2006-01-02_15-04-05")
	timestampFile := filepath.Join(pt.resultsDir, fmt.Sprintf("perf_%s.json", timestamp))

	if err := pt.saveToFile(report, timestampFile); err != nil {
		return fmt.Errorf("failed to save timestamped results: %w", err)
	}

	// Also save as latest.json for easy comparison
	latestFile := filepath.Join(pt.resultsDir, LatestFile)
	if err := pt.saveToFile(report, latestFile); err != nil {
		return fmt.Errorf("failed to save latest results: %w", err)
	}

	fmt.Printf("✅ Performance results saved:\n")
	fmt.Printf("   📁 %s\n", timestampFile)
	fmt.Printf("   📁 %s\n", latestFile)

	return nil
}

// LoadLatest loads the latest performance results
func (pt *PerformanceTracker) LoadLatest() (*PerformanceReport, error) {
	latestFile := filepath.Join(pt.resultsDir, LatestFile)
	return pt.loadFromFile(latestFile)
}

// LoadPrevious loads the previous performance results (second latest)
func (pt *PerformanceTracker) LoadPrevious() (*PerformanceReport, error) {
	files, err := pt.getResultFiles()
	if err != nil {
		return nil, err
	}

	if len(files) < 2 {
		return nil, fmt.Errorf("no previous results found")
	}

	// Get second latest file
	previousFile := filepath.Join(pt.resultsDir, files[1])
	return pt.loadFromFile(previousFile)
}

// CompareWithPrevious compares current results with previous results
func (pt *PerformanceTracker) CompareWithPrevious(current *PerformanceReport) (*ComparisonResult, error) {
	previous, err := pt.LoadPrevious()
	if err != nil {
		return nil, fmt.Errorf("failed to load previous results: %w", err)
	}

	comparison := pt.compareReports(current, previous)

	// Save comparison results
	comparisonFile := filepath.Join(pt.resultsDir, ComparisonFile)
	if err := pt.saveComparisonToFile(comparison, comparisonFile); err != nil {
		return nil, fmt.Errorf("failed to save comparison: %w", err)
	}

	return comparison, nil
}

// CreateReport creates a performance report from system info
func (pt *PerformanceTracker) CreateReport() *PerformanceReport {
	return &PerformanceReport{
		Timestamp: time.Now(),
		GitCommit: pt.getGitCommit(),
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		CPUModel:  pt.getCPUModel(),
		NumCPU:    runtime.NumCPU(),
		Results:   []BenchmarkResult{},
	}
}

// AddBenchmarkResult adds a benchmark result to the report
func (pt *PerformanceTracker) AddBenchmarkResult(report *PerformanceReport, result BenchmarkResult) {
	report.Results = append(report.Results, result)
}

// FinalizeSummary calculates and sets the summary for the report
func (pt *PerformanceTracker) FinalizeSummary(report *PerformanceReport) {
	summary := Summary{
		TotalTests: len(report.Results),
	}

	// Find key metrics
	for _, result := range report.Results {
		switch {
		case strings.Contains(result.Name, "SimpleHandler") && !strings.Contains(result.Name, "HighPressure"):
			summary.SimpleHandlerNs = result.NsPerOp
			summary.MemoryPerRequest = result.BytesPerOp
			summary.AllocsPerRequest = result.AllocsPerOp
		case strings.Contains(result.Name, "JSONResponse") && !strings.Contains(result.Name, "HighPressure"):
			summary.JSONResponseNs = result.NsPerOp
		case strings.Contains(result.Name, "HighPressure") || strings.Contains(result.Name, "RPS"):
			if summary.HighPressureBest == 0 || result.NsPerOp < summary.HighPressureBest {
				summary.HighPressureBest = result.NsPerOp
			}
		}
	}

	report.Summary = summary
}

// PrintSummary prints a formatted summary of the performance report
func (pt *PerformanceTracker) PrintSummary(report *PerformanceReport) {
	fmt.Printf("\n🚀 Performance Test Summary\n")
	fmt.Printf("═══════════════════════════════════════\n")
	fmt.Printf("📅 Timestamp: %s\n", report.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("🔧 Go Version: %s\n", report.GoVersion)
	fmt.Printf("💻 Platform: %s/%s (%d CPUs)\n", report.OS, report.Arch, report.NumCPU)
	if report.GitCommit != "" {
		fmt.Printf("📝 Git Commit: %s\n", report.GitCommit[:8])
	}
	fmt.Printf("\n📊 Key Metrics:\n")
	fmt.Printf("   Simple Handler:    %8.1f ns/op\n", report.Summary.SimpleHandlerNs)
	fmt.Printf("   JSON Response:     %8.1f ns/op\n", report.Summary.JSONResponseNs)
	fmt.Printf("   High Pressure Best:%8.1f ns/op\n", report.Summary.HighPressureBest)
	fmt.Printf("   Memory per Request:%8d B/op\n", report.Summary.MemoryPerRequest)
	fmt.Printf("   Allocs per Request:%8d allocs/op\n", report.Summary.AllocsPerRequest)
	fmt.Printf("   Total Tests:       %8d\n", report.Summary.TotalTests)
}

// PrintComparison prints a formatted comparison result
func (pt *PerformanceTracker) PrintComparison(comparison *ComparisonResult) {
	fmt.Printf("\n📈 Performance Comparison\n")
	fmt.Printf("═══════════════════════════════════════\n")
	fmt.Printf("🎯 Overall Status: %s\n", strings.ToUpper(comparison.OverallStatus))

	summary := comparison.Summary
	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("   ✅ Improved:  %d tests (avg: %+.1f%%)\n",
		summary.TotalImprovedTests, summary.AvgImprovement)
	fmt.Printf("   ❌ Regressed: %d tests (avg: %+.1f%%)\n",
		summary.TotalRegressedTests, summary.AvgRegression)
	fmt.Printf("   ⚪ Stable:    %d tests\n", summary.TotalStableTests)
	fmt.Printf("   🔥 Major Changes: %d\n", summary.MajorChanges)

	// Show significant improvements
	if len(comparison.Improvements) > 0 {
		fmt.Printf("\n✅ Improvements:\n")
		for _, imp := range comparison.Improvements {
			if imp.Significance == "major" || imp.Significance == "minor" {
				fmt.Printf("   %s: %+.1f%% (%+.1f ns)\n",
					imp.TestName, imp.PercentChange, imp.AbsoluteChangeNs)
			}
		}
	}

	// Show significant regressions
	if len(comparison.Regressions) > 0 {
		fmt.Printf("\n❌ Regressions:\n")
		for _, reg := range comparison.Regressions {
			if reg.Significance == "major" || reg.Significance == "minor" {
				fmt.Printf("   %s: %+.1f%% (%+.1f ns)\n",
					reg.TestName, reg.PercentChange, reg.AbsoluteChangeNs)
			}
		}
	}
}

// Helper methods

func (pt *PerformanceTracker) saveToFile(report *PerformanceReport, filename string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0644)
}

func (pt *PerformanceTracker) loadFromFile(filename string) (*PerformanceReport, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var report PerformanceReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}

	return &report, nil
}

func (pt *PerformanceTracker) saveComparisonToFile(comparison *ComparisonResult, filename string) error {
	data, err := json.MarshalIndent(comparison, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0644)
}

func (pt *PerformanceTracker) getResultFiles() ([]string, error) {
	files, err := ioutil.ReadDir(pt.resultsDir)
	if err != nil {
		return nil, err
	}

	var resultFiles []string
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "perf_") && strings.HasSuffix(file.Name(), ".json") {
			resultFiles = append(resultFiles, file.Name())
		}
	}

	// Sort by name (which includes timestamp) in descending order
	sort.Sort(sort.Reverse(sort.StringSlice(resultFiles)))
	return resultFiles, nil
}

func (pt *PerformanceTracker) compareReports(current, previous *PerformanceReport) *ComparisonResult {
	comparison := &ComparisonResult{
		Current:  current,
		Previous: previous,
	}

	// Create maps for easy lookup
	currentMap := make(map[string]BenchmarkResult)
	for _, result := range current.Results {
		currentMap[result.Name] = result
	}

	previousMap := make(map[string]BenchmarkResult)
	for _, result := range previous.Results {
		previousMap[result.Name] = result
	}

	// Compare matching tests
	var totalImprovement, totalRegression float64
	var improvedCount, regressedCount, stableCount, majorChanges int

	for name, currentResult := range currentMap {
		if previousResult, exists := previousMap[name]; exists {
			change := ((currentResult.NsPerOp - previousResult.NsPerOp) / previousResult.NsPerOp) * 100
			absChange := currentResult.NsPerOp - previousResult.NsPerOp

			comp := Comparison{
				TestName:         name,
				CurrentNs:        currentResult.NsPerOp,
				PreviousNs:       previousResult.NsPerOp,
				PercentChange:    change,
				AbsoluteChangeNs: absChange,
			}

			// Classify significance
			absChangePercent := change
			if absChangePercent < 0 {
				absChangePercent = -absChangePercent
			}

			if absChangePercent >= MajorChangeThreshold {
				comp.Significance = "major"
				majorChanges++
			} else if absChangePercent >= MinorChangeThreshold {
				comp.Significance = "minor"
			} else {
				comp.Significance = "negligible"
			}

			// Classify as improvement or regression
			if change < -NegligibleChangeThreshold {
				comparison.Improvements = append(comparison.Improvements, comp)
				totalImprovement += -change
				improvedCount++
			} else if change > NegligibleChangeThreshold {
				comparison.Regressions = append(comparison.Regressions, comp)
				totalRegression += change
				regressedCount++
			} else {
				stableCount++
			}
		}
	}

	// Calculate averages
	var avgImprovement, avgRegression float64
	if improvedCount > 0 {
		avgImprovement = totalImprovement / float64(improvedCount)
	}
	if regressedCount > 0 {
		avgRegression = totalRegression / float64(regressedCount)
	}

	// Determine overall status
	overallStatus := "stable"
	if majorChanges > 0 {
		if improvedCount > regressedCount {
			overallStatus = "improved"
		} else if regressedCount > improvedCount {
			overallStatus = "regressed"
		}
	} else if improvedCount > regressedCount*2 {
		overallStatus = "improved"
	} else if regressedCount > improvedCount*2 {
		overallStatus = "regressed"
	}

	comparison.OverallStatus = overallStatus
	comparison.Summary = ComparisonSummary{
		TotalImprovedTests:  improvedCount,
		TotalRegressedTests: regressedCount,
		TotalStableTests:    stableCount,
		AvgImprovement:      avgImprovement,
		AvgRegression:       avgRegression,
		MajorChanges:        majorChanges,
	}

	return comparison
}

func (pt *PerformanceTracker) getGitCommit() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func (pt *PerformanceTracker) getCPUModel() string {
	// This is a simple implementation - could be enhanced for different platforms
	if runtime.GOOS == "darwin" {
		cmd := exec.Command("sysctl", "-n", "machdep.cpu.brand_string")
		output, err := cmd.Output()
		if err == nil {
			return strings.TrimSpace(string(output))
		}
	}
	return ""
}
