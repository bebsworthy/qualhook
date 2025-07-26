#!/usr/bin/env python3
"""
Generate HTML dashboard for test quality metrics
"""

import json
import sys
import os
from datetime import datetime
from pathlib import Path

def generate_dashboard(metrics_file, trend_file, output_file):
    """Generate HTML dashboard from metrics data"""
    
    # Load data
    with open(metrics_file, 'r') as f:
        metrics = json.load(f)
    
    with open(trend_file, 'r') as f:
        trends = json.load(f)
    
    # Extract key metrics
    coverage = metrics['metrics']['coverage']['average']
    quality_score = metrics['metrics']['quality_score']['overall']
    test_count = metrics['metrics']['test_types']['total_test_files']
    flaky_count = metrics['metrics']['flakiness']['flaky_test_count']
    
    # Generate trend data for charts
    trend_labels = [t['timestamp'][:10] for t in trends[-10:]]
    coverage_trend = [t.get('coverage', 0) for t in trends[-10:]]
    quality_trend = [t.get('quality_score', 0) for t in trends[-10:]]
    
    # Generate HTML
    html = f"""
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Qualhook Test Quality Dashboard</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body {{
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }}
        .container {{
            max-width: 1200px;
            margin: 0 auto;
        }}
        h1 {{
            color: #333;
            text-align: center;
            margin-bottom: 30px;
        }}
        .metrics-grid {{
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 40px;
        }}
        .metric-card {{
            background: white;
            border-radius: 8px;
            padding: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }}
        .metric-value {{
            font-size: 36px;
            font-weight: bold;
            margin: 10px 0;
        }}
        .metric-label {{
            color: #666;
            font-size: 14px;
        }}
        .quality-score {{
            color: {get_score_color(quality_score)};
        }}
        .coverage {{
            color: {get_score_color(coverage)};
        }}
        .charts-grid {{
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
            gap: 20px;
            margin-bottom: 40px;
        }}
        .chart-container {{
            background: white;
            border-radius: 8px;
            padding: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }}
        .details {{
            background: white;
            border-radius: 8px;
            padding: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 20px;
        }}
        table {{
            width: 100%;
            border-collapse: collapse;
        }}
        th, td {{
            text-align: left;
            padding: 12px;
            border-bottom: 1px solid #eee;
        }}
        th {{
            background-color: #f8f9fa;
            font-weight: 600;
        }}
        .timestamp {{
            text-align: center;
            color: #666;
            font-size: 14px;
            margin-top: 40px;
        }}
        .warning {{
            color: #ff9800;
        }}
        .error {{
            color: #f44336;
        }}
        .success {{
            color: #4caf50;
        }}
    </style>
</head>
<body>
    <div class="container">
        <h1>Qualhook Test Quality Dashboard</h1>
        
        <div class="metrics-grid">
            <div class="metric-card">
                <div class="metric-label">Quality Score</div>
                <div class="metric-value quality-score">{quality_score:.1f}/100</div>
            </div>
            <div class="metric-card">
                <div class="metric-label">Test Coverage</div>
                <div class="metric-value coverage">{coverage:.1f}%</div>
            </div>
            <div class="metric-card">
                <div class="metric-label">Total Tests</div>
                <div class="metric-value">{test_count}</div>
            </div>
            <div class="metric-card">
                <div class="metric-label">Flaky Tests</div>
                <div class="metric-value {get_flaky_class(flaky_count)}">{flaky_count}</div>
            </div>
        </div>
        
        <div class="charts-grid">
            <div class="chart-container">
                <h3>Coverage Trend</h3>
                <canvas id="coverageTrend"></canvas>
            </div>
            <div class="chart-container">
                <h3>Quality Score Trend</h3>
                <canvas id="qualityTrend"></canvas>
            </div>
        </div>
        
        <div class="details">
            <h3>Package Coverage</h3>
            <table>
                <thead>
                    <tr>
                        <th>Package</th>
                        <th>Coverage</th>
                        <th>Test Files</th>
                    </tr>
                </thead>
                <tbody>
                    {generate_package_rows(metrics['metrics']['coverage']['by_package'])}
                </tbody>
            </table>
        </div>
        
        <div class="details">
            <h3>Test Type Distribution</h3>
            <table>
                <thead>
                    <tr>
                        <th>Test Type</th>
                        <th>Count</th>
                    </tr>
                </thead>
                <tbody>
                    {generate_test_type_rows(metrics['metrics']['test_types'])}
                </tbody>
            </table>
        </div>
        
        {generate_flaky_tests_section(metrics['metrics']['flakiness'])}
        
        {generate_slow_tests_section(metrics['metrics']['execution_time'])}
        
        <div class="timestamp">
            Generated at: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}
        </div>
    </div>
    
    <script>
        // Coverage Trend Chart
        const coverageCtx = document.getElementById('coverageTrend').getContext('2d');
        new Chart(coverageCtx, {{
            type: 'line',
            data: {{
                labels: {json.dumps(trend_labels)},
                datasets: [{{
                    label: 'Coverage %',
                    data: {json.dumps(coverage_trend)},
                    borderColor: 'rgb(75, 192, 192)',
                    backgroundColor: 'rgba(75, 192, 192, 0.2)',
                    tension: 0.1
                }}]
            }},
            options: {{
                responsive: true,
                scales: {{
                    y: {{
                        beginAtZero: true,
                        max: 100
                    }}
                }}
            }}
        }});
        
        // Quality Score Trend Chart
        const qualityCtx = document.getElementById('qualityTrend').getContext('2d');
        new Chart(qualityCtx, {{
            type: 'line',
            data: {{
                labels: {json.dumps(trend_labels)},
                datasets: [{{
                    label: 'Quality Score',
                    data: {json.dumps(quality_trend)},
                    borderColor: 'rgb(54, 162, 235)',
                    backgroundColor: 'rgba(54, 162, 235, 0.2)',
                    tension: 0.1
                }}]
            }},
            options: {{
                responsive: true,
                scales: {{
                    y: {{
                        beginAtZero: true,
                        max: 100
                    }}
                }}
            }}
        }});
    </script>
</body>
</html>
"""
    
    # Write dashboard
    with open(output_file, 'w') as f:
        f.write(html)
    
    print(f"Dashboard generated: {output_file}")

def get_score_color(score):
    """Get color based on score"""
    if score >= 80:
        return '#4caf50'  # green
    elif score >= 60:
        return '#ff9800'  # orange
    else:
        return '#f44336'  # red

def get_flaky_class(count):
    """Get CSS class for flaky test count"""
    if count == 0:
        return 'success'
    elif count <= 2:
        return 'warning'
    else:
        return 'error'

def generate_package_rows(packages):
    """Generate HTML rows for package coverage"""
    rows = []
    for pkg in sorted(packages, key=lambda x: x['coverage'], reverse=True)[:10]:
        coverage = pkg['coverage']
        color = get_score_color(coverage)
        error_indicator = ' ⚠️' if pkg.get('error') else ''
        rows.append(f"""
            <tr>
                <td>{pkg['package'].replace('github.com/bebsworthy/qualhook/', '')}</td>
                <td style="color: {color}">{coverage:.1f}%{error_indicator}</td>
                <td>{pkg['test_files']}</td>
            </tr>
        """)
    return ''.join(rows)

def generate_test_type_rows(test_types):
    """Generate HTML rows for test type distribution"""
    rows = []
    types = [
        ('Unit Tests', test_types['unit_test_files']),
        ('Integration Tests', test_types['integration_test_files']),
        ('E2E Tests', test_types['e2e_test_files']),
        ('Benchmarks', test_types['benchmark_files']),
        ('Examples', test_types['example_files'])
    ]
    for name, count in types:
        rows.append(f"""
            <tr>
                <td>{name}</td>
                <td>{count}</td>
            </tr>
        """)
    return ''.join(rows)

def generate_flaky_tests_section(flakiness):
    """Generate flaky tests section"""
    if flakiness['flaky_test_count'] == 0:
        return ''
    
    rows = []
    for test in flakiness['flaky_tests'][:10]:
        rows.append(f"""
            <tr>
                <td>{test['test']}</td>
                <td>{test['package'].replace('github.com/bebsworthy/qualhook/', '')}</td>
            </tr>
        """)
    
    return f"""
        <div class="details">
            <h3>Flaky Tests</h3>
            <table>
                <thead>
                    <tr>
                        <th>Test</th>
                        <th>Package</th>
                    </tr>
                </thead>
                <tbody>
                    {''.join(rows)}
                </tbody>
            </table>
        </div>
    """

def generate_slow_tests_section(execution_time):
    """Generate slow tests section"""
    slowest = execution_time.get('slowest_tests', [])
    if not slowest:
        return ''
    
    rows = []
    for test in slowest[:10]:
        duration = test['duration']
        color = '#f44336' if duration > 1.0 else '#ff9800' if duration > 0.5 else '#4caf50'
        rows.append(f"""
            <tr>
                <td>{test['name']}</td>
                <td>{test['package'].replace('github.com/bebsworthy/qualhook/', '')}</td>
                <td style="color: {color}">{duration:.3f}s</td>
            </tr>
        """)
    
    return f"""
        <div class="details">
            <h3>Slowest Tests</h3>
            <table>
                <thead>
                    <tr>
                        <th>Test</th>
                        <th>Package</th>
                        <th>Duration</th>
                    </tr>
                </thead>
                <tbody>
                    {''.join(rows)}
                </tbody>
            </table>
        </div>
    """

if __name__ == '__main__':
    if len(sys.argv) != 4:
        print("Usage: generate_test_dashboard.py <metrics_json> <trend_json> <output_html>")
        sys.exit(1)
    
    generate_dashboard(sys.argv[1], sys.argv[2], sys.argv[3])