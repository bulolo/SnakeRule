#!/usr/bin/env python3
"""Repeatable concurrent load probe for the SnakeRule evaluation API."""

import argparse
import concurrent.futures
import json
import os
import platform
import statistics
import subprocess
import time
import urllib.error
import urllib.request


def percentile(values, quantile):
    if not values:
        return None
    ordered = sorted(values)
    index = max(0, min(len(ordered) - 1, int((len(ordered) - 1) * quantile + 0.999999)))
    return round(ordered[index] * 1000, 3)


def run_request(url, payload, api_key):
    request = urllib.request.Request(
        url,
        data=payload,
        headers={"Content-Type": "application/json", "X-API-Key": api_key},
        method="POST",
    )
    started = time.perf_counter()
    try:
        with urllib.request.urlopen(request, timeout=15) as response:
            response.read()
            return response.status, time.perf_counter() - started, ""
    except urllib.error.HTTPError as error:
        error.read()
        return error.code, time.perf_counter() - started, f"HTTP {error.code}"
    except Exception as error:  # Include connection and timeout failures in the report.
        return 0, time.perf_counter() - started, type(error).__name__


def scrape_metrics(base_url):
    try:
        with urllib.request.urlopen(base_url.rstrip("/") + "/metrics", timeout=2) as response:
            lines = response.read().decode().splitlines()
        return {line.split(" ", 1)[0]: line.split(" ", 1)[1] for line in lines if line.startswith("snakerule_database_") and " " in line}
    except Exception as error:
        return {"unavailable": type(error).__name__}


def docker_stats(container):
    if not container:
        return None
    try:
        result = subprocess.run(["docker", "stats", "--no-stream", "--format", "{{.CPUPerc}} {{.MemUsage}} {{.MemPerc}}", container], check=True, capture_output=True, text=True, timeout=10)
        return result.stdout.strip()
    except Exception as error:
        return f"unavailable: {type(error).__name__}"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base-url", default="http://localhost:8081")
    parser.add_argument("--ruleset-id", required=True)
    parser.add_argument("--api-key", default=os.environ.get("SNAKERULE_API_KEY", ""))
    parser.add_argument("--requests", type=int, default=1000)
    parser.add_argument("--concurrency", type=int, default=20)
    parser.add_argument("--facts-file", help="JSON facts input; defaults to the ecommerce fixture")
    parser.add_argument("--container", help="optional backend container name for docker stats")
    args = parser.parse_args()
    if not args.api_key:
        parser.error("pass --api-key or set SNAKERULE_API_KEY")
    if args.requests < 1 or args.concurrency < 1:
        parser.error("requests and concurrency must be positive")

    facts = {"order": {"amount": 12800}, "customer": {"isNew": True}}
    if args.facts_file:
        with open(args.facts_file, encoding="utf-8") as source:
            facts = json.load(source)
    payload = json.dumps({"ruleset_id": args.ruleset_id, "facts": facts}).encode()
    url = args.base_url.rstrip("/") + "/api/v1/engine/evaluate"
    metrics_before = scrape_metrics(args.base_url)
    container_before = docker_stats(args.container)
    started = time.perf_counter()
    with concurrent.futures.ThreadPoolExecutor(max_workers=args.concurrency) as pool:
        results = list(pool.map(lambda _: run_request(url, payload, args.api_key), range(args.requests)))
    elapsed = time.perf_counter() - started
    metrics_after = scrape_metrics(args.base_url)
    container_after = docker_stats(args.container)
    latencies = [row[1] for row in results]
    statuses = {}
    errors = 0
    for status, _, error in results:
        statuses[str(status)] = statuses.get(str(status), 0) + 1
        errors += status < 200 or status >= 300

    report = {
        "endpoint": url,
        "requests": args.requests,
        "concurrency": args.concurrency,
        "elapsed_seconds": round(elapsed, 3),
        "throughput_per_second": round(args.requests / elapsed, 2) if elapsed else None,
        "latency_ms": {
            "min": round(min(latencies) * 1000, 3),
            "mean": round(statistics.mean(latencies) * 1000, 3),
            "p50": percentile(latencies, 0.50),
            "p95": percentile(latencies, 0.95),
            "p99": percentile(latencies, 0.99),
            "max": round(max(latencies) * 1000, 3),
        },
        "errors": errors,
        "error_rate": round(errors / args.requests, 6),
        "status_codes": statuses,
        "environment": {"platform": platform.platform(), "cpu_count": os.cpu_count()},
        "database_metrics_before": metrics_before,
        "database_metrics_after": metrics_after,
        "backend_container_stats_before": container_before,
        "backend_container_stats_after": container_after,
        "note": "Repeat with fixed rules, facts, resources, and concurrency; do not publish API keys with this report.",
    }
    print(json.dumps(report, indent=2))


if __name__ == "__main__":
    main()
