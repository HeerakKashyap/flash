package performance

import (
	"encoding/json"
	"time"
)

// BenchmarkResult represents a single benchmark measurement
type BenchmarkResult struct {
	Name           string  `json:"name"`
	NsPerOp        float64 `json:"ns_per_op"`
	BytesPerOp     int64   `json:"bytes_per_op"`
	AllocsPerOp    int64   `json:"allocs_per_op"`
	MBPerSec       float64 `json:"mb_per_sec,omitempty"`
	RequestsPerSec float64 `json:"requests_per_sec,omitempty"`
	AvgLatencyNs   float64 `json:"avg_latency_ns,omitempty"`
	Workers        int     `json:"workers,omitempty"`
	CPUs           int     `json:"cpus"`
	Iterations     int     `json:"iterations"`
}

// PerformanceReport represents a complete performance test run
type PerformanceReport struct {
	Timestamp time.Time         `json:"timestamp"`
	GitCommit string            `json:"git_commit,omitempty"`
	GoVersion string            `json:"go_version"`
	OS        string            `json:"os"`
	Arch      string            `json:"arch"`
	CPUModel  string            `json:"cpu_model,omitempty"`
	NumCPU    int               `json:"num_cpu"`
	Results   []BenchmarkResult `json:"results"`
	Summary   Summary           `json:"summary"`
}

// Summary provides aggregate performance metrics
type Summary struct {
	TotalTests       int     `json:"total_tests"`
	SimpleHandlerNs  float64 `json:"simple_handler_ns"`
	JSONResponseNs   float64 `json:"json_response_ns"`
	HighPressureBest float64 `json:"high_pressure_best_ns"`
	MemoryPerRequest int64   `json:"memory_per_request"`
	AllocsPerRequest int64   `json:"allocs_per_request"`
}

// ComparisonResult represents the comparison between two performance reports
type ComparisonResult struct {
	Current       *PerformanceReport `json:"current"`
	Previous      *PerformanceReport `json:"previous"`
	Improvements  []Comparison       `json:"improvements"`
	Regressions   []Comparison       `json:"regressions"`
	OverallStatus string             `json:"overall_status"` // "improved", "regressed", "stable"
	Summary       ComparisonSummary  `json:"summary"`
}

// Comparison represents a single benchmark comparison
type Comparison struct {
	TestName         string  `json:"test_name"`
	CurrentNs        float64 `json:"current_ns"`
	PreviousNs       float64 `json:"previous_ns"`
	PercentChange    float64 `json:"percent_change"`
	AbsoluteChangeNs float64 `json:"absolute_change_ns"`
	Significance     string  `json:"significance"` // "major", "minor", "negligible"
}

// ComparisonSummary provides aggregate comparison metrics
type ComparisonSummary struct {
	TotalImprovedTests  int     `json:"total_improved_tests"`
	TotalRegressedTests int     `json:"total_regressed_tests"`
	TotalStableTests    int     `json:"total_stable_tests"`
	AvgImprovement      float64 `json:"avg_improvement_percent"`
	AvgRegression       float64 `json:"avg_regression_percent"`
	MajorChanges        int     `json:"major_changes"`
}

// Constants for comparison thresholds
const (
	MajorChangeThreshold      = 10.0 // 10% change is considered major
	MinorChangeThreshold      = 3.0  // 3% change is considered minor
	NegligibleChangeThreshold = 1.0  // < 1% change is negligible
)

// ToJSON converts the report to JSON string
func (pr *PerformanceReport) ToJSON() (string, error) {
	data, err := json.MarshalIndent(pr, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON creates a PerformanceReport from JSON string
func FromJSON(jsonStr string) (*PerformanceReport, error) {
	var report PerformanceReport
	err := json.Unmarshal([]byte(jsonStr), &report)
	if err != nil {
		return nil, err
	}
	return &report, nil
}
