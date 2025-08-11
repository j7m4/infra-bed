# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This is an infrastructure testbed repository designed for testing and experimenting with various infrastructure components and Go application spikes in a Kubernetes environment. The project uses Kind for local Kubernetes clusters and Tilt for development workflow orchestration.

## Key Technologies

- **Container Orchestration**: Kubernetes (Kind cluster)
- **Development Workflow**: Tilt
- **Language**: Go 1.24
- **Observability Stack**: Grafana LGTM (Loki, Grafana, Tempo, Mimir), Pyroscope, Alloy
- **Database Operators**: MySQL Operator, CloudNativePG (PostgreSQL)
- **Streaming**: Apache Kafka with Strimzi Operator
- **Tracing**: OpenTelemetry with trace-to-profile correlation

## Common Development Commands

### Environment Setup
```bash
./setup.sh                    # Create Kind cluster and setup environment
./setup.sh --no-preload       # Setup without preloading Docker images
./test.sh                     # Test the setup and validate configuration
./teardown.sh                 # Destroy Kind cluster and clean up resources
```

### Development Workflow
```bash
tilt up                       # Start the development environment
tilt down                     # Stop all Tilt resources
```

### Testing Spikes via Tilt UI (http://localhost:10350)
- `run-fibonacci` - Test CPU-intensive Fibonacci calculation
- `run-produce-kafka` - Test Kafka message production
- `run-consume-kafka` - Test Kafka message consumption

### Database Operations (via Tilt triggers)
#### MySQL:
- `deploy-operator-mysql` - Deploy MySQL operator
- `install-cluster-mysql` - Create MySQL cluster
- `mysql-status` - Check cluster status
- `mysql-test-connection` - Test database connectivity
- `mysql-kill-primary` - Test failover

#### PostgreSQL:
- `deploy-operator-postgres` - Deploy CloudNativePG operator
- `install-cluster-postgres` - Create PostgreSQL cluster
- `postgres-status` - Check cluster status
- `postgres-test-connection` - Test database connectivity
- `postgres-kill-primary` - Test failover

### Kafka Operations (via Tilt triggers)
- `operator-install-kafka` - Install Strimzi Kafka operator
- `install-persistent-cluster-kafka` - Create persistent Kafka cluster
- `uninstall-persistent-cluster-kafka` - Remove Kafka cluster

## Architecture & Code Structure

### Infrastructure Components
The system deploys a complete observability and database infrastructure stack in Kubernetes:

1. **Observability Layer** (`k8s/` directory):
   - LGTM stack for logs, metrics, traces, and profiles
   - Pyroscope for continuous profiling
   - Alloy for pprof endpoint scraping
   - OpenTelemetry collector for trace processing

2. **Database Layer**:
   - MySQL Operator manages InnoDB clusters with automatic failover
   - CloudNativePG manages PostgreSQL clusters with built-in PgBouncer pooling
   - Both support high availability with 3-node configurations

3. **Streaming Layer**:
   - Strimzi Operator manages Kafka clusters
   - Supports persistent storage configurations

### Go Application Structure (`go-spikes/`)
The Go application serves as a testing harness for infrastructure components:

- **Entry Point**: `cmd/main.go` - HTTP server with spike endpoints
- **Handlers**: `cmd/handler/` - Request handlers for different spikes
- **Business Logic**: `pkg/` - Core functionality packages
  - `fibonacci/` - CPU-intensive computation spike
  - `kafka/` - Kafka producer/consumer implementations
  - `logger/` - Zerolog-based structured logging

### Adding New Spikes
To add a new spike:
1. Create endpoint in `cmd/main.go` main() function
2. Add handler function in `cmd/handler/`
3. Implement logic in new package under `pkg/`
4. Add Tilt resource in `Tiltfile` to trigger the spike

## Service Endpoints

- **Grafana UI**: http://localhost:3000 (admin/admin)
- **Tilt UI**: http://localhost:10350
- **Go Application**: http://localhost:8080
- **pprof Profiling**: http://localhost:6060/debug/pprof/
- **Tempo**: http://localhost:3200
- **Pyroscope**: http://localhost:4040
- **OTLP gRPC**: localhost:4317

## Database Connection Details

### MySQL
- **Port**: 3306 (direct), 6446/6447 (router)
- **Database**: test_db
- **User**: root (password from secret)

### PostgreSQL
- **Port**: 5432 (direct), 5433 (pooled)
- **Database**: myapp
- **User**: app / app_password
- **Superuser**: postgres / postgres-root-password

## Development Notes

- The project uses Kind cluster named "infra-bed" (falls back to "tacops-dev" if not found)
- All infrastructure is deployed via Tilt with manual triggers for database/Kafka operations
- The Go application uses live reload in Tilt for rapid development
- Observability stack is automatically deployed on `tilt up`
- Database and Kafka clusters require manual triggering in Tilt UI