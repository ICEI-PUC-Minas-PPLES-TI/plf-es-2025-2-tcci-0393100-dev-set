# Estimation Engine Benchmarks

This directory contains benchmark tools and performance analysis for the SET CLI Estimation Engine v2.0.

## Contents

### Scripts

- **`run_benchmark.py`** - Python script to run automated benchmarks
  - Executes multiple estimation tests
  - Collects performance metrics
  - Generates detailed reports
  - Exports CSV for analysis

### Configuration

- **`benchmark_estimation.json`** - Test case definitions
  - 12 diverse test cases
  - Categories: authentication, bug fixes, features, documentation, etc.
  - Expected outcomes for validation

### Reports

- **`benchmark_report_YYYYMMDD_HHMMSS.md`** - Auto-generated benchmark reports
  - Executive summary
  - Detailed results by test case
  - Performance analysis by category
  - Consistency and confidence metrics

- **`benchmark_results_YYYYMMDD_HHMMSS.csv`** - CSV export of results
  - For spreadsheet analysis
  - Includes all metrics and statistics

### Analysis

- **`PERFORMANCE_ANALYSIS.md`** - Comprehensive performance analysis
  - Executive summary and key findings
  - Response time analysis
  - Confidence score analysis
  - Estimate accuracy analysis
  - v1.1 vs v2.0 comparison
  - Optimization recommendations
  - Production readiness assessment

## Running Benchmarks

### Prerequisites

```bash
# Python 3.7+
python --version

# Build the CLI
go build -o bin/set.exe .
```

### Run Benchmark

```bash
cd benchmarks

# Run with 1 iteration per test (fast)
python run_benchmark.py

# Or edit the script to run 3 iterations per test (more reliable)
# Change: runs_per_test=1 to runs_per_test=3
```

### Output

The script will:
1. Run all test cases from `benchmark_estimation.json`
2. Generate a timestamped markdown report
3. Export results to CSV
4. Print summary to console

## Benchmark Results Summary

**Last Run:** 2025-10-19 16:29:54

### Key Metrics

| Metric | Value |
|--------|-------|
| Success Rate | 100% (12/12) |
| Avg Response Time | 11.62s |
| Avg Confidence | 62.0% |
| Avg Similar Tasks | 10.0 |
| Size Accuracy | 75% (9/12) |

### Performance by Category

| Category | Avg Time | Avg Confidence |
|----------|----------|----------------|
| Generic | 8.02s | 75% ⭐ Best |
| Data Migration | 9.26s | 62% |
| Refactoring | 10.15s | 62% |
| UI | 10.97s | 63% |
| Testing | 11.04s | 55% |
| Bug Fix | 11.79s | 60% |
| Authentication | 12.36s | 63% |
| Infrastructure | 12.52s | 65% |
| Security | 12.58s | 60% |
| Performance | 12.72s | 58% |
| Documentation | 13.61s | 63% |
| Feature | 14.37s | 58% |

## Adding New Test Cases

Edit `benchmark_estimation.json` and add a new test case:

```json
{
  "id": "TC013",
  "category": "your_category",
  "task_title": "Your task title",
  "task_description": "Detailed description (optional)",
  "expected_characteristics": {
    "should_find_similar": true,
    "expected_size_range": ["M", "L"],
    "notes": "What you expect from this test"
  }
}
```

## Customizing Benchmarks

### Adjust Test Runs

Edit `run_benchmark.py` line ~401:

```python
# Change from 1 to 3 for more thorough testing
results = benchmark.run_benchmark(runs_per_test=3)
```

### Adjust Timeout

Edit `run_benchmark.py` line ~45:

```python
# Change timeout from 60 to 120 seconds
result = subprocess.run(cmd, capture_output=True, text=True, timeout=120)
```

### Change CLI Path

Edit `run_benchmark.py` line ~9:

```python
# Update if your CLI is in a different location
def __init__(self, cli_path: str = "./bin/set.exe", ...):
```

## Interpreting Results

### Response Time
- **<10s:** Excellent
- **10-15s:** Good (current range)
- **>15s:** Consider optimization

### Confidence Score
- **≥70%:** High confidence
- **50-69%:** Medium confidence (expected range)
- **<50%:** Low confidence (ambiguous tasks)

### Size Accuracy
- **≥80%:** Excellent calibration
- **70-79%:** Good (current: 75%)
- **<70%:** Needs calibration

### Consistency
- **100%:** Perfect (all runs agree)
- **90-99%:** Very good
- **<90%:** May need temperature adjustment

## Troubleshooting

### "AI provider not available"
- Ensure `OPENAI_API_KEY` is set in environment or `.set.yaml`
- Check API key is valid: `./bin/set.exe configure --validate`

### "Timeout exceeded"
- Increase timeout in script (line ~45)
- Check network connection
- Verify OpenAI API status

### "Failed to parse JSON output"
- Check CLI is built correctly: `go build -o bin/set.exe .`
- Run single estimate manually to debug: `./bin/set.exe estimate "Test" --output json`

## Future Improvements

### Planned Features
- [ ] Multi-model comparison (GPT-4 vs GPT-5-nano vs GPT-4-turbo)
- [ ] Historical accuracy tracking (estimate vs actual)
- [ ] Performance regression detection
- [ ] Automated CI/CD integration
- [ ] Visual charts and graphs
- [ ] Confidence calibration tuning

### Contributing
To add benchmark features:
1. Fork the repository
2. Add your test cases to `benchmark_estimation.json`
3. Modify `run_benchmark.py` as needed
4. Submit a pull request with your improvements

## Related Documentation

- [PERFORMANCE_ANALYSIS.md](./PERFORMANCE_ANALYSIS.md) - Detailed analysis
- [../docs/ESTIMATION_GUIDE.md](../docs/ESTIMATION_GUIDE.md) - Estimation engine documentation
- [../CURRENT_STATE.md](../CURRENT_STATE.md) - Project status

---

**Benchmark Tool Version:** 1.0
**Last Updated:** 2025-10-19
**Maintainer:** Development Team
