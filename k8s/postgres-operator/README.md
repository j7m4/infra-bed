# PostgreSQL with CloudNativePG Operator

This setup uses CloudNativePG operator to create a highly available PostgreSQL cluster with automatic failover.

## Features

- **High Availability**: 3-node cluster (1 primary + 2 replicas) with automatic failover
- **Connection Pooling**: Built-in PgBouncer for connection pooling
- **Monitoring**: Prometheus metrics enabled
- **Automatic Backups**: Can be configured with S3/GCS/Azure storage
- **Point-in-Time Recovery**: WAL archiving support
- **Rolling Updates**: Zero-downtime PostgreSQL updates

## Architecture

```
┌─────────────────────────────────────────┐
│           CloudNativePG Operator        │
└─────────────────────────────────────────┘
                    │
    ┌───────────────┼───────────────┐
    ▼               ▼               ▼
┌─────────┐   ┌─────────┐   ┌─────────┐
│Primary  │   │Replica 1│   │Replica 2│
│  Pod    │◄──│   Pod   │◄──│   Pod   │
└─────────┘   └─────────┘   └─────────┘
    │               │               │
    └───────────────┼───────────────┘
                    ▼
    ┌──────────────────────────────┐
    │         PgBouncer            │
    │    (Connection Pooler)       │
    └──────────────────────────────┘
```

## Services

- `postgres-cluster-rw`: Read-write service (points to primary)
- `postgres-cluster-ro`: Read-only service (load balanced across replicas)
- `postgres-cluster-pooler-rw`: PgBouncer pooled connection to primary
- `postgres-cluster-pooler-ro`: PgBouncer pooled connection to replicas

## Quick Start

1. Deploy the operator:
   ```bash
   tilt trigger deploy-operator-postgres
   ```

2. Create the PostgreSQL cluster:
   ```bash
   tilt trigger install-cluster-postgres
   ```

3. Port forward for local access:
   ```bash
   tilt trigger port-forward-postgres        # Direct connection on :5432
   tilt trigger port-forward-postgres-pooler # Pooled connection on :5433
   ```

4. Test the connection:
   ```bash
   tilt trigger test-connection-postgres
   ```

## Connection Details

- **Database**: myapp
- **Username**: app
- **Password**: app_password
- **Superuser**: postgres / postgres-root-password

### Local Connection Examples

Direct connection:
```bash
psql postgresql://app:app_password@localhost:5432/myapp
```

Through pooler:
```bash
psql postgresql://app:app_password@localhost:5433/myapp
```

## Failover Testing

To test automatic failover:
```bash
tilt trigger postgres-kill-primary
```

This will:
1. Delete the current primary pod
2. CloudNativePG automatically promotes a replica to primary
3. Creates a new replica to maintain the desired instance count
4. Updates service endpoints automatically

## Monitoring

Check cluster status:
```bash
tilt trigger postgres-status
tilt trigger show-instances-postgres
tilt trigger cluster-status-postgres
```

## Helper Scripts

- `postgres-status.sh`: Shows cluster and replication status
- `postgres-show-instances.sh`: Displays detailed instance information
- `postgres-test-connection.sh`: Tests all connection endpoints
- `postgres-kill-primary.sh`: Triggers failover for testing
- `postgres-connect.sh`: Interactive connection helper

## Configuration

The cluster is configured in `postgres-cluster.yaml` with:
- 3 instances for HA
- 10Gi storage per instance
- Connection pooling with PgBouncer
- Optimized PostgreSQL parameters
- Resource limits and requests

## CloudNativePG vs MySQL Operator

| Feature | CloudNativePG | MySQL Operator |
|---------|--------------|----------------|
| Failover | Automatic, < 30s | Automatic |
| Connection Pooling | Built-in PgBouncer | MySQL Router |
| Backup/Restore | S3/GCS/Azure support | mysqldump |
| Point-in-Time Recovery | Yes | Limited |
| Rolling Updates | Yes | Yes |
| Monitoring | Prometheus native | Metrics exporter |