#!/bin/bash

echo "=== PostgreSQL Failover Test ==="
echo ""

# Get current primary
PRIMARY=$(kubectl get pods -n db -l cnpg.io/cluster=postgres-cluster,cnpg.io/instanceRole=primary -o jsonpath='{.items[0].metadata.name}')

if [ -z "$PRIMARY" ]; then
  echo "Error: Could not find primary instance"
  exit 1
fi

echo "Current primary: $PRIMARY"
echo ""

# Show current cluster state
echo "=== Current Cluster State ==="
kubectl get pods -n db -l cnpg.io/cluster=postgres-cluster \
  -o custom-columns=NAME:.metadata.name,ROLE:.metadata.labels.cnpg\\.io/instanceRole,STATUS:.status.phase

echo ""
echo "⚠️  WARNING: This will delete the primary PostgreSQL pod to trigger failover!"
echo "Press Ctrl+C to cancel, or wait 5 seconds to continue..."
sleep 5

# Delete primary pod
echo ""
echo "Deleting primary pod: $PRIMARY"
kubectl delete pod $PRIMARY -n db --force --grace-period=0

echo ""
echo "Waiting for failover to complete..."
sleep 10

# Show new cluster state
echo ""
echo "=== New Cluster State After Failover ==="
kubectl get pods -n db -l cnpg.io/cluster=postgres-cluster \
  -o custom-columns=NAME:.metadata.name,ROLE:.metadata.labels.cnpg\\.io/instanceRole,STATUS:.status.phase

# Get new primary
NEW_PRIMARY=$(kubectl get pods -n db -l cnpg.io/cluster=postgres-cluster,cnpg.io/instanceRole=primary -o jsonpath='{.items[0].metadata.name}')
echo ""
echo "New primary: $NEW_PRIMARY"

# Wait for cluster to stabilize
echo ""
echo "Waiting for cluster to stabilize..."
kubectl wait --for=condition=Ready cluster/postgres-cluster -n db --timeout=60s

# Test connection after failover
echo ""
echo "=== Testing Connection After Failover ==="
kubectl run postgres-test-failover --rm -i --restart=Never --image=postgres:16 -n db -- \
  psql postgresql://app:app_password@postgres-cluster-rw:5432/myapp -c "SELECT version(), pg_is_in_recovery();" 2>/dev/null || echo "Connection failed"

echo ""
echo "Failover test completed!"