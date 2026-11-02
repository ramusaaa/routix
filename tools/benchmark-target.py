import argparse
import json
import math
from pathlib import Path
import re
import statistics
import sys

WORKLOADS = ("Noop204", "StaticText", "PathParameter", "Wildcard", "JSONStdlib", "JSONNative")
ROUTE_COUNTS = (1, 10, 100, 1000)
MODES = (("HTTP", 1), ("HTTPParallel", 8))
PATTERN = re.compile(
    r"^Benchmark(HTTPParallel|HTTP)/([^/\s]+)/routes=(\d+)/([^/\s]+?)"
    r"(?:-(\d+))?\s+\d+\s+([0-9]+(?:\.[0-9]*)?(?:[eE][+-]?[0-9]+)?)\s+ns/op(?:\s|$)"
)


def load_samples(paths):
    samples = {}
    for path in paths:
        for line in path.read_text().splitlines():
            if re.match(r"^(?:FAIL(?:\s|$)|--- FAIL:)", line.strip()):
                raise ValueError("input contains a failed Go test or benchmark")
            match = PATTERN.match(line)
            if match is None:
                if line.startswith(("BenchmarkHTTP/", "BenchmarkHTTPParallel/")):
                    raise ValueError("input contains an incomplete or unsupported HTTP benchmark line")
                continue
            mode, workload, count, framework, cpu, timing = match.groups()
            value = float(timing)
            if not math.isfinite(value) or value <= 0:
                raise ValueError("benchmark timings must be finite and positive")
            key = (mode, int(cpu or 1), workload, int(count), framework)
            samples.setdefault(key, []).append(value)
    return samples


def evaluate(samples, speedup, minimum_samples):
    rows = []
    for mode, cpu in MODES:
        for workload in WORKLOADS:
            for count in ROUTE_COUNTS:
                routix = samples.get((mode, cpu, workload, count, "Routix"), [])
                gin = samples.get((mode, cpu, workload, count, "Gin"), [])
                own_ns = statistics.median(routix) if routix else None
                gin_ns = statistics.median(gin) if gin else None
                ratio = gin_ns / own_ns if own_ns is not None and gin_ns is not None else None
                if not routix or not gin:
                    status = "missing"
                elif min(len(routix), len(gin)) < minimum_samples:
                    status = "insufficient_samples"
                elif ratio >= speedup:
                    status = "pass"
                else:
                    status = "below_target"
                rows.append({
                    "mode": mode,
                    "cpu": cpu,
                    "workload": workload,
                    "routes": count,
                    "routix_samples": len(routix),
                    "gin_samples": len(gin),
                    "routix_median_ns": own_ns,
                    "gin_median_ns": gin_ns,
                    "routix_min_ns": min(routix) if routix else None,
                    "routix_max_ns": max(routix) if routix else None,
                    "gin_min_ns": min(gin) if gin else None,
                    "gin_max_ns": max(gin) if gin else None,
                    "observed_speedup": ratio,
                    "status": status,
                })
    return {
        "target_speedup": speedup,
        "maximum_routix_ns_fraction_of_gin": 1 / speedup,
        "minimum_samples_per_framework": minimum_samples,
        "required_cells": len(rows),
        "passed_cells": sum(row["status"] == "pass" for row in rows),
        "target_met": all(row["status"] == "pass" for row in rows),
        "interpretation": "Observed median ratios; this is not a statistical significance test or a production latency guarantee.",
        "cells": rows,
    }


def main():
    parser = argparse.ArgumentParser(description="Check the complete Routix/Gin benchmark matrix against a throughput target.")
    parser.add_argument("files", nargs="+", type=Path)
    parser.add_argument("--speedup", type=float, default=1.5)
    parser.add_argument("--min-samples", type=int, default=6)
    args = parser.parse_args()
    if not math.isfinite(args.speedup) or args.speedup <= 1:
        parser.error("--speedup must be finite and greater than 1")
    if args.min_samples < 1:
        parser.error("--min-samples must be positive")
    paths = [path.resolve() for path in args.files]
    if len(set(paths)) != len(paths):
        parser.error("each input file must be provided only once")
    try:
        samples = load_samples(paths)
    except (OSError, UnicodeError, ValueError) as error:
        parser.error(str(error))
    if not samples:
        parser.error("no supported HTTP benchmark samples found")
    report = evaluate(samples, args.speedup, args.min_samples)
    print(json.dumps(report, indent=2, allow_nan=False))
    return 0 if report["target_met"] else 1


if __name__ == "__main__":
    sys.exit(main())
