# GoFlash Performance Testing Makefile

.PHONY: help perf perf-baseline perf-compare perf-interactive example-server

# Default target
help: ## Show performance testing help
	@echo "GoFlash Performance Testing Commands"
	@echo "═══════════════════════════════════════"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Performance Testing
perf: ## Run all performance tests
	go run ../../cmd/perf/main.go -all

perf-baseline: ## Run baseline performance tests only (fast)
	go run ../../cmd/perf/main.go -baseline

perf-compare: ## Compare latest performance results with previous
	go run ../../cmd/perf/main.go -compare

perf-interactive: ## Start interactive performance testing session
	go run ../../cmd/perf/main.go -interactive

# Traditional Go benchmarks
bench: ## Run Go benchmarks directly
	go test -bench=. -benchmem .

bench-cpu: ## Run benchmarks with CPU profiling
	go test -bench=. -benchmem -cpuprofile=cpu.prof .
	@echo "CPU profile saved to cpu.prof"
	@echo "View with: go tool pprof cpu.prof"

bench-mem: ## Run benchmarks with memory profiling
	go test -bench=. -benchmem -memprofile=mem.prof .
	@echo "Memory profile saved to mem.prof"
	@echo "View with: go tool pprof mem.prof"

# Example server
example-server: ## Run example server for testing
	@echo "Starting example server on :8080..."
	@echo "Visit: http://localhost:8080/ping"
	go run examples/basic/main.go

# Cleanup
clean: ## Clean up performance test results and profiles
	rm -rf results/
	rm -f cpu.prof mem.prof
	rm -f *.test

# Monitoring
monitor: ## Set up performance monitoring (runs tests every hour)
	@echo "Setting up performance monitoring..."
	@echo "This will run performance tests every hour"
	@echo "Results will be saved to results/"
	@while true; do \
		echo "Running scheduled performance test..."; \
		$(MAKE) perf-baseline; \
		sleep 3600; \
	done
