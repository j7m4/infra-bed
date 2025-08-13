# Configuration Strategy for go-spikes

## Overview

The go-spikes application uses a ConfigMap-based configuration strategy with hot reload capabilities. This allows for dynamic configuration updates without requiring pod restarts.

## Features

- **Hot Reload**: Configuration changes are automatically detected and applied without restarting the application
- **Type-Safe Configuration**: Strongly typed configuration structs with validation
- **Thread-Safe Access**: Configuration access is protected with read-write mutexes
- **Change Callbacks**: Handlers can register callbacks to be notified when configuration changes
- **Default Values**: Sensible defaults for all configuration options

## Configuration Structure

```yaml
server:
  port: 8080
  readTimeout: 30s
  writeTimeout: 30s
  idleTimeout: 120s

kafka:
  brokers:
    - kafka-cluster-kafka-bootstrap.kafka:9092
  topic: test-topic
  consumerGroup: go-spikes-consumer
  producer:
    batchSize: 100
    batchTimeout: 1s
    compressionType: snappy
    maxRetries: 3
  consumer:
    sessionTimeout: 10s
    heartbeatInterval: 3s
    maxPollRecords: 500
    autoOffsetReset: latest

database:
  mysql:
    enabled: false
    host: mycluster-router.default
    port: 6446
    database: test_db
    user: root
    maxConnections: 25
    maxIdleConns: 5
    connMaxLifetime: 5m
  postgres:
    enabled: false
    host: postgres-cluster-rw.default
    port: 5432
    database: myapp
    user: app
    sslMode: disable
    maxConnections: 25
    maxIdleConns: 5
    connMaxLifetime: 5m

features:
  enableProfiling: true
  enableTracing: true
  enableMetrics: true
  enableDebugLogging: false
  experimental:
    asyncProcessing: false
    cacheWarmup: false
    advancedMetrics: true

metrics:
  scrapeInterval: 10s
  histogramBuckets:
    - 0.001
    - 0.01
    - 0.1
    - 0.5
    - 1
    - 2.5
    - 5
    - 10
  labels:
    environment: development
    team: infrastructure
    version: v1.0.0
```

## Kubernetes Integration

### ConfigMap

The configuration is stored in a Kubernetes ConfigMap (`go-spikes-config`) that is mounted as a volume in the pod at `/etc/config/config.yaml`.

### Updating Configuration

To update the configuration in a running cluster:

```bash
# Edit the ConfigMap directly
kubectl edit configmap go-spikes-config

# Or apply an updated ConfigMap manifest
kubectl apply -f go-spikes/k8s/configmap.yaml
```

The application will automatically detect and apply the changes within a few seconds.

## API Endpoints

### Get Current Configuration
```bash
curl http://localhost:8080/config
```

### Check Feature Flag
```bash
# Check built-in features
curl http://localhost:8080/config/feature/profiling
curl http://localhost:8080/config/feature/tracing
curl http://localhost:8080/config/feature/metrics
curl http://localhost:8080/config/feature/debug

# Check experimental features
curl http://localhost:8080/config/feature/asyncProcessing
curl http://localhost:8080/config/feature/cacheWarmup
```

## Code Usage

### Accessing Configuration in Handlers

```go
// Get the full configuration
cfg := configManager.Get()

// Get specific sections
serverCfg := configManager.GetServer()
kafkaCfg := configManager.GetKafka()
dbCfg := configManager.GetDatabase()
features := configManager.GetFeatures()
metrics := configManager.GetMetrics()

// Check if a feature is enabled
if configManager.IsFeatureEnabled("asyncProcessing") {
    // Feature is enabled
}
```

### Registering Change Callbacks

```go
configManager.OnChange(func(cfg *config.Config) {
    // React to configuration changes
    log.Info().Msg("Configuration updated")
    
    // Update application behavior based on new config
    if cfg.Features.EnableDebugLogging {
        logger.SetDebugLevel()
    } else {
        logger.SetInfoLevel()
    }
})
```

## Implementation Details

### File Watching

The configuration system uses `fsnotify` to watch for changes to the configuration file. When Kubernetes updates the ConfigMap, it creates a new file and atomically swaps it, which triggers the file watcher.

### Thread Safety

All configuration access is protected by read-write mutexes to ensure thread-safe operation during configuration reloads.

### Error Handling

- If the configuration file is not found, the application uses default values
- If a configuration reload fails (e.g., due to invalid YAML), the application keeps the previous valid configuration
- All configuration changes are logged for debugging purposes

## Testing Configuration Changes

1. Deploy the application with Tilt:
   ```bash
   tilt up
   ```

2. Check current configuration:
   ```bash
   curl http://localhost:8080/config | jq .
   ```

3. Update the ConfigMap:
   ```bash
   kubectl edit configmap go-spikes-config
   # Change enableDebugLogging to true
   ```

4. Watch the application logs to see the configuration reload:
   ```bash
   kubectl logs -f deployment/go-spikes
   ```

5. Verify the change was applied:
   ```bash
   curl http://localhost:8080/config/feature/debug
   ```

## Benefits

1. **Zero Downtime Updates**: Configuration changes without pod restarts
2. **Centralized Configuration**: All configuration in one ConfigMap
3. **Environment-Specific Settings**: Easy to maintain different configs for different environments
4. **Feature Toggles**: Enable/disable features without code changes
5. **Dynamic Behavior**: Application behavior can be tuned in real-time

## Future Enhancements

- Support for secrets management (database passwords, API keys)
- Configuration validation with JSON Schema
- Configuration versioning and rollback
- Metrics for configuration changes
- Support for multiple configuration sources (ConfigMaps, Secrets, CRDs)