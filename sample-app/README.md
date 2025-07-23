# Sample Go Application

A RESTful Go application with various CPU and memory-intensive endpoints for testing continuous profiling with pprof and OpenTelemetry integration.

## Endpoints

### Health Check
- `GET /health` - Simple health check

### CPU-Intensive Endpoints
- `GET /cpu/fibonacci/{n}` - Calculate Fibonacci number (n: 1-45)
- `GET /cpu/prime/{n}` - Find prime numbers up to n using Sieve of Eratosthenes (n: 1-1000000)
- `GET /cpu/hash/{iterations}` - Perform SHA256 hashing iterations (1-1000000)
- `GET /cpu/sort/{size}` - Sort an array of given size (1-1000000)
- `GET /cpu/matrix/{size}` - Matrix multiplication of size x size (1-500)

### Memory-Intensive Endpoints
- `GET /memory/allocate/{mb}` - Allocate memory in MB (1-100)

### Mixed Workload
- `GET /workload/mixed` - Combined CPU and memory workload

## Features

- **pprof endpoints** exposed on port 6060
- **OpenTelemetry integration** for distributed tracing
- **otel-profiling-go** for linking profiles to traces
- **Continuous profiling** via Grafana Alloy scraping

## Testing

The application exposes pprof endpoints:
- `http://localhost:6060/debug/pprof/profile` - CPU profile
- `http://localhost:6060/debug/pprof/heap` - Heap profile
- `http://localhost:6060/debug/pprof/goroutine` - Goroutine profile

Use the Tilt UI buttons or curl commands to generate load and observe profiles in Grafana Pyroscope.