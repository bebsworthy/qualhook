#!/usr/bin/env python3
"""
Visualize test quality trends over time
"""

import json
import sys
import os
import matplotlib.pyplot as plt
import matplotlib.dates as mdates
from datetime import datetime
from pathlib import Path

def visualize_trends(trend_file, output_dir):
    """Generate trend visualizations"""
    
    # Create output directory
    Path(output_dir).mkdir(parents=True, exist_ok=True)
    
    # Load trend data
    with open(trend_file, 'r') as f:
        trends = json.load(f)
    
    if not trends:
        print("No trend data available")
        return
    
    # Extract data
    timestamps = [datetime.fromisoformat(t['timestamp'].replace('Z', '+00:00')) for t in trends]
    coverage = [t.get('coverage', 0) for t in trends]
    quality_scores = [t.get('quality_score', 0) for t in trends]
    test_counts = [t.get('test_count', 0) for t in trends]
    flaky_counts = [t.get('flaky_count', 0) for t in trends]
    
    # Set up the plot style
    plt.style.use('seaborn-v0_8-darkgrid')
    fig, ((ax1, ax2), (ax3, ax4)) = plt.subplots(2, 2, figsize=(15, 10))
    fig.suptitle('Qualhook Test Quality Trends', fontsize=16, fontweight='bold')
    
    # 1. Coverage Trend
    ax1.plot(timestamps, coverage, 'b-', linewidth=2, marker='o', markersize=6)
    ax1.axhline(y=70, color='r', linestyle='--', alpha=0.5, label='Minimum Threshold')
    ax1.set_title('Test Coverage Over Time', fontsize=14)
    ax1.set_ylabel('Coverage %', fontsize=12)
    ax1.set_ylim(0, 100)
    ax1.legend()
    ax1.grid(True, alpha=0.3)
    
    # Color the background based on coverage levels
    ax1.axhspan(80, 100, alpha=0.1, color='green')
    ax1.axhspan(60, 80, alpha=0.1, color='yellow')
    ax1.axhspan(0, 60, alpha=0.1, color='red')
    
    # 2. Quality Score Trend
    ax2.plot(timestamps, quality_scores, 'g-', linewidth=2, marker='s', markersize=6)
    ax2.axhline(y=60, color='r', linestyle='--', alpha=0.5, label='Minimum Threshold')
    ax2.set_title('Quality Score Over Time', fontsize=14)
    ax2.set_ylabel('Quality Score', fontsize=12)
    ax2.set_ylim(0, 100)
    ax2.legend()
    ax2.grid(True, alpha=0.3)
    
    # 3. Test Count Trend
    ax3.bar(timestamps, test_counts, width=0.8, color='purple', alpha=0.7)
    ax3.set_title('Total Test Files Over Time', fontsize=14)
    ax3.set_ylabel('Number of Test Files', fontsize=12)
    ax3.grid(True, alpha=0.3, axis='y')
    
    # 4. Flaky Tests Trend
    ax4.bar(timestamps, flaky_counts, width=0.8, color='orange', alpha=0.7)
    ax4.axhline(y=5, color='r', linestyle='--', alpha=0.5, label='Maximum Threshold')
    ax4.set_title('Flaky Tests Over Time', fontsize=14)
    ax4.set_ylabel('Number of Flaky Tests', fontsize=12)
    ax4.legend()
    ax4.grid(True, alpha=0.3, axis='y')
    
    # Format x-axis for all subplots
    for ax in [ax1, ax2, ax3, ax4]:
        ax.xaxis.set_major_formatter(mdates.DateFormatter('%m/%d'))
        ax.xaxis.set_major_locator(mdates.DayLocator(interval=1))
        plt.setp(ax.xaxis.get_majorticklabels(), rotation=45, ha='right')
    
    # Adjust layout
    plt.tight_layout()
    
    # Save the plot
    output_file = os.path.join(output_dir, 'test_quality_trends.png')
    plt.savefig(output_file, dpi=300, bbox_inches='tight')
    print(f"Trend visualization saved to: {output_file}")
    
    # Generate summary statistics
    generate_summary_stats(trends, output_dir)

def generate_summary_stats(trends, output_dir):
    """Generate summary statistics"""
    
    if len(trends) < 2:
        return
    
    # Calculate statistics
    recent = trends[-7:]  # Last 7 data points
    
    avg_coverage = sum(t.get('coverage', 0) for t in recent) / len(recent)
    avg_quality = sum(t.get('quality_score', 0) for t in recent) / len(recent)
    max_flaky = max(t.get('flaky_count', 0) for t in recent)
    
    # Calculate trends
    coverage_trend = trends[-1].get('coverage', 0) - trends[-2].get('coverage', 0)
    quality_trend = trends[-1].get('quality_score', 0) - trends[-2].get('quality_score', 0)
    
    # Generate summary report
    summary = f"""# Test Quality Summary Report

Generated: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}

## 7-Day Averages
- **Average Coverage**: {avg_coverage:.1f}%
- **Average Quality Score**: {avg_quality:.1f}/100
- **Maximum Flaky Tests**: {max_flaky}

## Recent Trends
- **Coverage Change**: {coverage_trend:+.1f}%
- **Quality Score Change**: {quality_trend:+.1f}

## Current Status
- **Latest Coverage**: {trends[-1].get('coverage', 0):.1f}%
- **Latest Quality Score**: {trends[-1].get('quality_score', 0):.1f}/100
- **Latest Flaky Tests**: {trends[-1].get('flaky_count', 0)}

## Recommendations
"""
    
    if avg_coverage < 70:
        summary += "- ⚠️ Coverage is below recommended threshold (70%)\n"
    
    if avg_quality < 60:
        summary += "- ⚠️ Quality score is below recommended threshold (60)\n"
    
    if max_flaky > 0:
        summary += f"- ⚠️ Fix {max_flaky} flaky tests\n"
    
    if coverage_trend < -5:
        summary += "- 📉 Coverage is declining rapidly\n"
    
    if quality_trend < -10:
        summary += "- 📉 Quality score is declining rapidly\n"
    
    # Save summary
    summary_file = os.path.join(output_dir, 'test_quality_summary.md')
    with open(summary_file, 'w') as f:
        f.write(summary)
    
    print(f"Summary report saved to: {summary_file}")

if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("Usage: visualize_test_trends.py <trend_file> [output_dir]")
        sys.exit(1)
    
    trend_file = sys.argv[1]
    output_dir = sys.argv[2] if len(sys.argv) > 2 else 'test_metrics/visualizations'
    
    # Check if matplotlib is available
    try:
        visualize_trends(trend_file, output_dir)
    except ImportError:
        print("Warning: matplotlib not installed. Install with: pip install matplotlib")
        print("Generating text summary only...")
        
        # Load trend data for summary
        with open(trend_file, 'r') as f:
            trends = json.load(f)
        
        if trends:
            generate_summary_stats(trends, output_dir)