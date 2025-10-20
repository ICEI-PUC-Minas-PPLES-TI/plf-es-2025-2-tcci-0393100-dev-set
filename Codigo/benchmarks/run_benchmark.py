#!/usr/bin/env python3
"""
Benchmark script for SET CLI Estimation Engine v2.0
Tests performance, accuracy, and consistency of AI-powered estimations
"""

import json
import subprocess
import time
import statistics
from datetime import datetime
from typing import Dict, List, Any
import csv

class EstimationBenchmark:
    def __init__(self, cli_path: str = "./bin/set.exe", test_cases_file: str = "benchmark_estimation.json"):
        self.cli_path = cli_path
        self.test_cases_file = test_cases_file
        self.results = []

    def load_test_cases(self) -> List[Dict]:
        """Load test cases from JSON file"""
        with open(self.test_cases_file, 'r', encoding='utf-8') as f:
            data = json.load(f)
        return data['test_cases']

    def run_estimation(self, task_title: str, task_description: str = "") -> Dict[str, Any]:
        """Run a single estimation and collect metrics"""
        cmd = [self.cli_path, "estimate", task_title, "--output", "json"]
        if task_description:
            cmd.extend(["--description", task_description])

        start_time = time.time()

        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=60,
                encoding='utf-8'
            )

            elapsed_time = time.time() - start_time

            if result.returncode != 0:
                return {
                    'success': False,
                    'error': result.stderr,
                    'elapsed_time': elapsed_time
                }

            # Parse JSON output
            # Output format includes both the pretty-printed output and JSON at the end
            output_lines = result.stdout.strip().split('\n')

            # Find the JSON output (last line or lines starting with {)
            json_output = None
            for line in reversed(output_lines):
                if line.strip().startswith('{'):
                    try:
                        json_output = json.loads(line)
                        break
                    except:
                        continue

            if not json_output:
                # Try parsing the entire output
                try:
                    json_output = json.loads(result.stdout)
                except:
                    return {
                        'success': False,
                        'error': 'Failed to parse JSON output',
                        'elapsed_time': elapsed_time,
                        'raw_output': result.stdout[:500]
                    }

            return {
                'success': True,
                'elapsed_time': elapsed_time,
                'estimation': json_output.get('estimation', {}),
                'method': json_output.get('method', 'unknown'),
                'similar_tasks_count': len(json_output.get('similar_tasks', [])),
                'raw_output': result.stdout
            }

        except subprocess.TimeoutExpired:
            return {
                'success': False,
                'error': 'Timeout (>60s)',
                'elapsed_time': 60.0
            }
        except Exception as e:
            return {
                'success': False,
                'error': str(e),
                'elapsed_time': time.time() - start_time
            }

    def run_benchmark(self, runs_per_test: int = 3) -> List[Dict]:
        """Run all test cases multiple times and collect results"""
        test_cases = self.load_test_cases()
        all_results = []

        print(f"\n{'='*80}")
        print(f"  ESTIMATION ENGINE BENCHMARK - v2.0")
        print(f"{'='*80}")
        print(f"Test cases: {len(test_cases)}")
        print(f"Runs per test: {runs_per_test}")
        print(f"Total estimations: {len(test_cases) * runs_per_test}")
        print(f"{'='*80}\n")

        for idx, test_case in enumerate(test_cases, 1):
            tc_id = test_case['id']
            title = test_case['task_title']
            description = test_case.get('task_description', '')
            category = test_case['category']

            print(f"[{idx}/{len(test_cases)}] Running {tc_id}: {title}")
            print(f"  Category: {category}")

            test_results = []

            for run in range(runs_per_test):
                print(f"  Run {run + 1}/{runs_per_test}...", end='', flush=True)

                result = self.run_estimation(title, description)

                if result['success']:
                    est = result['estimation']
                    print(f" OK {est.get('estimated_hours', 'N/A')}h, "
                          f"{est.get('estimated_size', 'N/A')}, "
                          f"{est.get('confidence_score', 0)*100:.0f}% conf, "
                          f"{result['elapsed_time']:.1f}s")
                else:
                    print(f" FAIL: {result.get('error', 'Unknown error')}")

                test_results.append(result)

                # Small delay between runs to avoid rate limiting
                if run < runs_per_test - 1:
                    time.sleep(1)

            # Aggregate results for this test case
            successful_runs = [r for r in test_results if r['success']]

            if successful_runs:
                hours = [r['estimation'].get('estimated_hours', 0) for r in successful_runs]
                times = [r['elapsed_time'] for r in successful_runs]
                confidences = [r['estimation'].get('confidence_score', 0) for r in successful_runs]
                sizes = [r['estimation'].get('estimated_size', 'N/A') for r in successful_runs]
                similar_counts = [r['similar_tasks_count'] for r in successful_runs]

                aggregated = {
                    'test_case_id': tc_id,
                    'category': category,
                    'task_title': title,
                    'task_description': description,
                    'runs': runs_per_test,
                    'successful_runs': len(successful_runs),
                    'failed_runs': runs_per_test - len(successful_runs),

                    # Hours statistics
                    'avg_hours': statistics.mean(hours) if hours else 0,
                    'median_hours': statistics.median(hours) if hours else 0,
                    'stdev_hours': statistics.stdev(hours) if len(hours) > 1 else 0,
                    'min_hours': min(hours) if hours else 0,
                    'max_hours': max(hours) if hours else 0,

                    # Size consistency
                    'most_common_size': max(set(sizes), key=sizes.count) if sizes else 'N/A',
                    'size_consistency': sizes.count(max(set(sizes), key=sizes.count)) / len(sizes) * 100 if sizes else 0,

                    # Confidence statistics
                    'avg_confidence': statistics.mean(confidences) * 100 if confidences else 0,
                    'min_confidence': min(confidences) * 100 if confidences else 0,
                    'max_confidence': max(confidences) * 100 if confidences else 0,

                    # Performance statistics
                    'avg_time': statistics.mean(times) if times else 0,
                    'min_time': min(times) if times else 0,
                    'max_time': max(times) if times else 0,

                    # Similar tasks
                    'avg_similar_tasks': statistics.mean(similar_counts) if similar_counts else 0,

                    # Expected characteristics
                    'expected_size_range': test_case['expected_characteristics']['expected_size_range'],
                    'expected_similarity': test_case['expected_characteristics']['should_find_similar'],

                    # Raw results for detailed analysis
                    'all_runs': test_results
                }

                all_results.append(aggregated)
            else:
                print(f"  ⚠ All runs failed for {tc_id}")

            print()

        return all_results

    def generate_report(self, results: List[Dict]) -> str:
        """Generate a detailed markdown report"""
        timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")

        report = f"""# Estimation Engine Benchmark Report

**Date:** {timestamp}
**Version:** 2.0 - AI-Only Mode with Enhanced Dataset
**Test Cases:** {len(results)}

## Executive Summary

"""

        # Calculate overall statistics
        successful_tests = [r for r in results if r['successful_runs'] > 0]
        total_runs = sum(r['runs'] for r in results)
        total_successful = sum(r['successful_runs'] for r in results)

        avg_time = statistics.mean([r['avg_time'] for r in successful_tests]) if successful_tests else 0
        avg_confidence = statistics.mean([r['avg_confidence'] for r in successful_tests]) if successful_tests else 0

        report += f"""### Overall Performance

- **Success Rate:** {total_successful}/{total_runs} ({total_successful/total_runs*100:.1f}%)
- **Average Response Time:** {avg_time:.2f}s
- **Average Confidence:** {avg_confidence:.1f}%
- **Test Categories:** {len(set(r['category'] for r in results))}

### Key Metrics

| Metric | Value |
|--------|-------|
| Total Test Cases | {len(results)} |
| Successful Tests | {len(successful_tests)} |
| Failed Tests | {len(results) - len(successful_tests)} |
| Avg Response Time | {avg_time:.2f}s |
| Min Response Time | {min(r['min_time'] for r in successful_tests):.2f}s |
| Max Response Time | {max(r['max_time'] for r in successful_tests):.2f}s |
| Avg Confidence | {avg_confidence:.1f}% |
| Avg Similar Tasks Found | {statistics.mean([r['avg_similar_tasks'] for r in successful_tests]):.1f} |

---

## Detailed Results by Test Case

"""

        # Group by category
        by_category = {}
        for result in results:
            cat = result['category']
            if cat not in by_category:
                by_category[cat] = []
            by_category[cat].append(result)

        for category, cat_results in sorted(by_category.items()):
            report += f"\n### Category: {category.replace('_', ' ').title()}\n\n"

            for result in cat_results:
                tc_id = result['test_case_id']
                title = result['task_title']

                # Determine if estimate matches expected range
                actual_size = result['most_common_size']
                expected_range = result['expected_size_range']
                size_match = "✓" if actual_size in expected_range else "⚠"

                report += f"""#### {tc_id}: {title}

**Success:** {result['successful_runs']}/{result['runs']} runs

| Metric | Value | Notes |
|--------|-------|-------|
| **Estimate** | {result['avg_hours']:.1f}h (±{result['stdev_hours']:.1f}h) | Range: {result['min_hours']:.1f}h - {result['max_hours']:.1f}h |
| **Size** | {actual_size} {size_match} | Expected: {', '.join(expected_range)} |
| **Confidence** | {result['avg_confidence']:.1f}% | Range: {result['min_confidence']:.1f}% - {result['max_confidence']:.1f}% |
| **Performance** | {result['avg_time']:.2f}s | Range: {result['min_time']:.2f}s - {result['max_time']:.2f}s |
| **Similar Tasks** | {result['avg_similar_tasks']:.1f} | Should find similar: {result['expected_similarity']} |
| **Consistency** | {result['size_consistency']:.0f}% | Size agreement across runs |

"""

                if result['task_description']:
                    report += f"**Description:** {result['task_description']}\n\n"

        # Analysis section
        report += "\n---\n\n## Analysis\n\n"

        report += "### Consistency Analysis\n\n"
        high_consistency = [r for r in successful_tests if r['size_consistency'] >= 100]
        low_consistency = [r for r in successful_tests if r['size_consistency'] < 100]

        report += f"- **Perfect Consistency (100%):** {len(high_consistency)}/{len(successful_tests)} tests\n"
        report += f"- **Variable Results:** {len(low_consistency)}/{len(successful_tests)} tests\n\n"

        if low_consistency:
            report += "**Tests with Variable Results:**\n\n"
            for r in low_consistency:
                report += f"- {r['test_case_id']}: {r['task_title']} ({r['size_consistency']:.0f}% consistency)\n"
            report += "\n"

        report += "### Performance by Category\n\n"
        report += "| Category | Avg Time | Avg Confidence | Avg Similar Tasks |\n"
        report += "|----------|----------|----------------|-------------------|\n"

        for category, cat_results in sorted(by_category.items()):
            cat_successful = [r for r in cat_results if r['successful_runs'] > 0]
            if cat_successful:
                avg_time = statistics.mean([r['avg_time'] for r in cat_successful])
                avg_conf = statistics.mean([r['avg_confidence'] for r in cat_successful])
                avg_similar = statistics.mean([r['avg_similar_tasks'] for r in cat_successful])
                report += f"| {category} | {avg_time:.2f}s | {avg_conf:.1f}% | {avg_similar:.1f} |\n"

        report += "\n### Confidence Distribution\n\n"
        high_conf = [r for r in successful_tests if r['avg_confidence'] >= 70]
        med_conf = [r for r in successful_tests if 40 <= r['avg_confidence'] < 70]
        low_conf = [r for r in successful_tests if r['avg_confidence'] < 40]

        report += f"- **High Confidence (≥70%):** {len(high_conf)} tests\n"
        report += f"- **Medium Confidence (40-69%):** {len(med_conf)} tests\n"
        report += f"- **Low Confidence (<40%):** {len(low_conf)} tests\n\n"

        # Recommendations
        report += "\n---\n\n## Recommendations\n\n"

        if avg_time > 10:
            report += "⚠ **Performance:** Average response time exceeds 10 seconds. Consider:\n"
            report += "  - Optimizing AI prompt length\n"
            report += "  - Reducing dataset size sent to AI\n\n"

        if avg_confidence < 60:
            report += "⚠ **Confidence:** Average confidence is below 60%. Consider:\n"
            report += "  - Improving historical data quality\n"
            report += "  - Adding more similar tasks to dataset\n\n"

        if len(low_consistency) > len(results) * 0.3:
            report += "⚠ **Consistency:** More than 30% of tests show variable results. Consider:\n"
            report += "  - Adjusting AI temperature for more deterministic outputs\n"
            report += "  - Improving task descriptions for clarity\n\n"

        report += "\n---\n\n*Report generated by Estimation Engine Benchmark v2.0*\n"

        return report

    def export_csv(self, results: List[Dict], filename: str = "benchmark_results.csv"):
        """Export results to CSV for spreadsheet analysis"""
        with open(filename, 'w', newline='', encoding='utf-8') as f:
            writer = csv.writer(f)

            # Header
            writer.writerow([
                'Test ID', 'Category', 'Task Title', 'Task Description',
                'Runs', 'Successful', 'Failed',
                'Avg Hours', 'Median Hours', 'Stdev Hours', 'Min Hours', 'Max Hours',
                'Most Common Size', 'Size Consistency %',
                'Avg Confidence %', 'Min Confidence %', 'Max Confidence %',
                'Avg Time (s)', 'Min Time (s)', 'Max Time (s)',
                'Avg Similar Tasks', 'Expected Size Range', 'Expected Similarity'
            ])

            # Data rows
            for r in results:
                writer.writerow([
                    r['test_case_id'],
                    r['category'],
                    r['task_title'],
                    r['task_description'],
                    r['runs'],
                    r['successful_runs'],
                    r['failed_runs'],
                    f"{r['avg_hours']:.2f}",
                    f"{r['median_hours']:.2f}",
                    f"{r['stdev_hours']:.2f}",
                    f"{r['min_hours']:.2f}",
                    f"{r['max_hours']:.2f}",
                    r['most_common_size'],
                    f"{r['size_consistency']:.1f}",
                    f"{r['avg_confidence']:.1f}",
                    f"{r['min_confidence']:.1f}",
                    f"{r['max_confidence']:.1f}",
                    f"{r['avg_time']:.2f}",
                    f"{r['min_time']:.2f}",
                    f"{r['max_time']:.2f}",
                    f"{r['avg_similar_tasks']:.1f}",
                    ', '.join(r['expected_size_range']),
                    r['expected_similarity']
                ])

        print(f"[OK] CSV exported to {filename}")


def main():
    """Main benchmark execution"""
    benchmark = EstimationBenchmark()

    # Run benchmark with 1 run per test (change to 3 for full benchmark)
    results = benchmark.run_benchmark(runs_per_test=1)

    # Generate report
    report = benchmark.generate_report(results)

    # Save report
    report_file = f"benchmark_report_{datetime.now().strftime('%Y%m%d_%H%M%S')}.md"
    with open(report_file, 'w', encoding='utf-8') as f:
        f.write(report)

    print(f"\n{'='*80}")
    print(f"[OK] Benchmark complete!")
    print(f"[OK] Report saved to: {report_file}")

    # Export CSV
    csv_file = f"benchmark_results_{datetime.now().strftime('%Y%m%d_%H%M%S')}.csv"
    benchmark.export_csv(results, csv_file)

    print(f"{'='*80}\n")

    # Print summary
    print(report.split('## Detailed Results')[0])


if __name__ == "__main__":
    main()
