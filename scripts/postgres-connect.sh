#!/bin/bash

echo "=== PostgreSQL Connection Helper ==="
echo ""

echo "1. Connecting to primary (read-write)..."
kubectl run postgres-client-test-rw --rm -i --restart=Never --image=postgres:16 -n db -- \
  psql postgresql://app:app_password@postgres-cluster-rw:5432/myapp -c "SELECT 1;"

echo "2. Connecting to read-only replica..."
kubectl run postgres-client-test-ro --rm -i --restart=Never --image=postgres:16 -n db -- \
  psql postgresql://app:app_password@postgres-cluster-ro:5432/myapp -c "SELECT 1;"

echo "3. Connecting to 'r' read-balanced replica..."
kubectl run postgres-client-test-r --rm -i --restart=Never --image=postgres:16 -n db -- \
  psql postgresql://app:app_password@postgres-cluster-r:5432/myapp -c "SELECT 1;"

echo "4. Connecting through PgBouncer pooler..."
kubectl run postgres-client-test-pooler --rm -i --restart=Never --image=postgres:16 -n db -- \
  psql postgresql://app:app_password@postgres-cluster-pooler:5432/myapp -c "SELECT 1;"

echo "5. Connecting as superuser..."
PRIMARY=$(kubectl get pods -n db -l cnpg.io/cluster=postgres-cluster,cnpg.io/instanceRole=primary -o jsonpath='{.items[0].metadata.name}')
kubectl exec -i -n db $PRIMARY -c postgres -- psql -U postgres -c "SELECT 1;"

echo ""
echo "All connection methods tested."