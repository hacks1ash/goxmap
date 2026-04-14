#!/bin/bash
#
# run_benchmarks.sh - Run model-mapper benchmark suite and produce reports.
#
# Usage:
#   ./benchmarks/run_benchmarks.sh          # from project root
#   cd benchmarks && ./run_benchmarks.sh    # from benchmarks dir
#
# Prerequisites:
#   go test (standard toolchain)
#   benchstat (optional): go install golang.org/x/perf/cmd/benchstat@latest

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
RESULTS_FILE="${SCRIPT_DIR}/results.txt"
COUNT=6

echo "=== model-mapper benchmark suite ==="
echo "Running benchmarks (count=${COUNT}) ..."
echo ""

cd "${PROJECT_ROOT}"

go test -bench=. -benchmem -count="${COUNT}" -timeout=10m ./benchmarks/ | tee "${RESULTS_FILE}"

echo ""
echo "Raw results saved to: ${RESULTS_FILE}"
echo ""

# ---------------------------------------------------------------------------
# benchstat summary (if available)
# ---------------------------------------------------------------------------
if command -v benchstat &>/dev/null; then
    echo "=== benchstat summary ==="
    echo ""
    benchstat "${RESULTS_FILE}"
else
    echo "benchstat not found. Install it for statistical analysis:"
    echo "  go install golang.org/x/perf/cmd/benchstat@latest"
fi

echo ""

# ---------------------------------------------------------------------------
# Markdown table from raw output
# ---------------------------------------------------------------------------
echo "=== Markdown table ==="
echo ""
echo "| Benchmark | ns/op | B/op | allocs/op |"
echo "|-----------|------:|-----:|----------:|"

# Parse only the final run of each benchmark (last occurrence per name).
# When count>1, benchstat is the proper tool; this table gives a quick glance.
grep -E '^Benchmark' "${RESULTS_FILE}" | \
    awk '{
        name=$1;
        for (i=2; i<=NF; i++) {
            if ($i == "ns/op")       nsop = $(i-1);
            if ($i == "B/op")        bop  = $(i-1);
            if ($i == "allocs/op")   aop  = $(i-1);
        }
        data[name] = nsop "|" bop "|" aop;
    }
    END {
        for (name in data) {
            split(data[name], v, "|");
            printf "| %-40s | %10s | %8s | %9s |\n", name, v[1], v[2], v[3];
        }
    }' | sort

echo ""
echo "Done."
