# OpenTelemetry Profiling with Grafana Pyroscope

> ⚠️ **PROOF OF CONCEPT** ⚠️
> 
> This project is a **proof-of-concept demonstration only** and is **NOT intended for production use**.
> 
> - No security hardening has been applied
> - No performance optimizations for production workloads
> - Configuration may contain insecure defaults
> - Limited error handling and monitoring
> - Not tested at scale
> 
> Use this code for learning, experimentation, and as a starting point only.

## Summary

This project contains various spikes for infrastructure testing. See each of the **Spikes** in `go-spikes/README.md`
for details.

## Prerequisites

- [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
- [Tilt](https://docs.tilt.dev/install.html)
- kubectl

## Quick Start

1. Run the setup script to create the Kind cluster:
```bash
./setup.sh
```

2. (Optional) Test your setup:
```bash
./test.sh
```

3. Start the development environment:
```bash
tilt up
```

4. Access Grafana at http://localhost:3000 (admin/admin)

5. When finished, clean up resources:
```bash
./teardown.sh
```

The environment includes:
   - Kind cluster
   - Grafana LGTM stack (Loki, Grafana, Tempo, Mimir, Pyroscope)
   - Grafana Alloy for scraping pprof endpoints
   - Sample Go application with pprof and otel-profiling-go integration

## Architecture

- **Go Spikes**: Go application that contains spike code, along with with pprof endpoints for trace correlation
- **Grafana Alloy**: Scrapes pprof endpoints and forwards to Pyroscope
- **Grafana LGTM**: Stores and visualizes profiles, logs, metrics, and traces
- **Pyroscope**: Integrated in LGTM for continuous profiling

## Infrastructure and Tests

Included are various infrastructure options against which spikes are run. 
The management of the infrastructure and the running of tests is mostly 
handled through tilt. Descriptions of the options are listed below:

* [Spikes](go-spikes/README.md) describes each spike and all maybe be executed through
  the [Tilt UI](http://localhost:10350).
* [Mysql](docs/mysql.md) - MySQL database operations
* [Kafka](docs/kafka.md) - Kafka message broker operations

## Features

- Continuous profiling with pprof (CPU, heap, goroutines)
- Trace-to-profile correlation using otel-profiling-go
- Grafana dashboards for profile visualization
- OpenTelemetry integration for distributed tracing