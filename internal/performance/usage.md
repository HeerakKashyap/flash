# GoFlash Performance Testing

Simple performance monitoring to track GoFlash performance over time and prevent regressions.

## Quick Start

### Get Initial Results

```bash
# Navigate to performance directory
cd internal/performance

# Run all performance tests (creates baseline)
make perf

# Or run just the core tests (faster)
make perf-baseline
```

This creates your first performance snapshot in `results/`.

### Compare with Previous

```bash
# Run tests and automatically compare with previous results
make perf-baseline

# Or just compare existing results without running new tests
make perf-compare
```

The system automatically compares your latest run with the previous one and shows:

- ✅ **Improvements**: Tests that got faster
- ❌ **Regressions**: Tests that got slower  
- ⚪ **Stable**: Tests with minimal change

## What Gets Tested

### Core Tests (make perf-baseline)

- **SimpleHandler**: Basic `/ping` response (~435ns/op)
- **JSONResponse**: JSON serialization (~1000ns/op)
- **PathParams**: URL parameter extraction  
- **QueryParams**: Query string parsing
- **JSONBinding**: Request body parsing
- **Middleware**: Middleware chain overhead
- **ErrorHandling**: Error response performance
- **LargeJSON**: Large payload handling

### High-Pressure Tests (make perf)

- **Concurrent Load**: Performance under high concurrency
- **RPS Testing**: Maximum requests per second
- **Memory Pressure**: Performance with memory allocation
- **Mixed Workloads**: Realistic traffic patterns

Tests run with 1, 2, 4, and 8 CPUs to measure scalability.

## Example Output

```
📊 Running baseline performance tests only...
  Running BenchmarkBaseline_SimpleHandler...
    ✅    435.4 ns/op,   1040 B/op,   10 allocs/op

🚀 Performance Test Summary
═══════════════════════════════════════
📅 Timestamp: 2025-08-26 03:27:00
📊 Key Metrics:
   Simple Handler:       435.4 ns/op
   JSON Response:       1007.0 ns/op
   Memory per Request:    1040 B/op
   Total Tests:              8

📈 Performance Comparison
═══════════════════════════════════════
🎯 Overall Status: IMPROVED

📊 Summary:
   ✅ Improved:  5 tests (avg: +8.2%)
   ❌ Regressed: 2 tests (avg: -3.1%)
   ⚪ Stable:    1 tests
```

## Understanding Results

### Performance Status

- **IMPROVED**: More tests got faster than slower
- **REGRESSED**: More tests got slower than faster  
- **STABLE**: Minimal overall change

### Change Significance

- **Major**: >10% performance change
- **Minor**: >3% performance change
- **Negligible**: <1% performance change

## Files Created

```
internal/performance/results/
├── perf_2025-08-26_03-27-00.json  # Timestamped results
├── latest.json                     # Latest test results
└── comparison.json                 # Latest comparison
```

## Development Workflow

```bash
# Navigate to performance directory
cd internal/performance

# During development - quick check
make perf-baseline

# Before releases - comprehensive testing
make perf

# Set up git hook to run tests before commits (from root directory)
cd ../../ && make install-hooks
```

## Troubleshooting

**"No previous results for comparison"**

- Normal for first run
- Run tests twice to see comparisons

**Inconsistent results**

- System load affects performance
- Run on quiet system for consistent results

**Need help?**

```bash
make help                         # Show performance testing commands
go run ../../cmd/perf/main.go -help  # Show performance tool options
```
