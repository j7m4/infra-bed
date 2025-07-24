# Sample Go Application

A RESTful Go application with various spikes for testing Golang against Infrastructure.

## Endpoints

### Health Check
- `GET /health` - Simple health check

### Spikes
- `GET /cpu/fibonacci/{n}` - Calculate Fibonacci number (n: 1-45) -- sample spike

#### Adding new spikes

Adding a new spike requires:
- a new endpoint in `cmd/main.go` in the `main()` 
- a new handler function in `cmd/main.go`
- a new package in `pkg` that contains the function called by the handler function.
- add an entry in the parent folder's `Tiltfile` to call the new endpoint.

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