# GEMINI.md

This file provides guidance to Gemini when working with code in this repository.

## Cross-Cutting Concerns Documentation

When documenting cross-cutting concerns in code and manifest files, use the following comment format to mark sections:

### Comment Format
- **Start marker**: `CROSS-CUTTING START OF <service1> CONFIGURATION FOR <service2>`
- **End marker**: `CROSS-CUTTING END OF <service1> CONFIGURATION FOR <service2>`

Where:
- `<service1>` is the service being configured
- `<service2>` is the service that the configuration is for

### Examples
- `CROSS-CUTTING START OF oltp-metrics CONFIGURATION FOR mysql`
- `CROSS-CUTTING START OF loki-logs CONFIGURATION FOR go-spikes`
- `CROSS-CUTTING START OF pyroscope CONFIGURATION FOR go-spikes`

These markers should be placed as inline comments appropriate to the file type:
- Go files: `// CROSS-CUTTING START OF <service1> CONFIGURATION FOR <service2>`
- YAML files: `# CROSS-CUTTING START OF <service1> CONFIGURATION FOR <service2>`
- JavaScript/TypeScript: `// CROSS-CUTTING START OF <service1> CONFIGURATION FOR <service2>`
- Shell scripts: `# CROSS-CUTTING START OF <service1> CONFIGURATION FOR <service2>`

### Services
The following services can be used as either service1 or service2 in cross-cutting documentation. Services may include descriptive notes in parentheses:
- `mysql`
- `postgres`
- `kafka`
- `alloy`
- `pyroscope`
- `go-spikes`
- `mimir`
- `tempo`
- `otel-metrics` (opentelemetry configurations related to metrics)
- `otel-tracing` (opentelemetry configurations related to tracing)
- `otel-logging` (opentelemetry configurations related to logging)

### Excluded Service Pairs
The following service1/service2 pairs should NOT be documented with cross-cutting comments:
- (Excluded pairs will be added as specified)

### Documented Cross-Cutting Topics
Only cross-cutting topics explicitly requested should be documented. Currently documented topics:
- `otel-tracing` / `go-spikes` - OpenTelemetry tracing configuration in main.go
- `otel-tracing` / `kafka` - OpenTelemetry tracing in Kafka consumer and producer
- `otel-tracing` / `tempo` - OTLP exporter configuration for Tempo
- `otel-logging` / `go-spikes` - Loki logging integration in main.go
- `otel-metrics` / `mysql` - MySQL exporter for Prometheus metrics
- `otel-metrics` / `postgres` - PostgreSQL exporter for Prometheus metrics
- `otel-metrics` / `kafka` - Kafka exporter for Prometheus metrics
- `otel-metrics` / `mimir` - OTLP metrics export to Mimir via LGTM
- `pyroscope` / `go-spikes` - pprof server and Pyroscope integration
- `alloy` / `go-spikes` - Alloy scraping pprof endpoints from go-spikes
- `alloy` / `mysql` - Alloy scraping MySQL exporter metrics
- `alloy` / `postgres` - Alloy scraping PostgreSQL exporter metrics
- `alloy` / `kafka` - Alloy scraping Kafka exporter metrics

## Commenting Style

Avoid adding obvious comments that simply restate what the code does as in-line comments. Function- or class-level comments that explain the 'why' are perfectly fine.
