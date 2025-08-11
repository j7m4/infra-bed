#!/bin/bash

echo "=== Testing PostgreSQL Connection ==="

# Test direct connection to primary
echo -e "\n1. Testing connection to primary service (postgres-cluster-rw)..."
kubectl run postgres-test-rw --rm -i --restart=Never --image=postgres:16 -n db -- \
  psql postgresql://app:app_password@postgres-cluster-rw:5432/myapp -c "SELECT version();" 2>/dev/null || echo "Connection failed"

# Test connection to read-only service
echo -e "\n2. Testing connection to read-only service (postgres-cluster-ro)..."
kubectl run postgres-test-ro --rm -i --restart=Never --image=postgres:16 -n db -- \
  psql postgresql://app:app_password@postgres-cluster-ro:5432/myapp -c "SELECT version();" 2>/dev/null || echo "Connection failed"

# Test connection through pooler
echo -e "\n3. Testing connection through PgBouncer pooler (postgres-cluster-pooler-rw)..."
kubectl run postgres-test-pooler --rm -i --restart=Never --image=postgres:16 -n db -- \
  psql postgresql://app:app_password@postgres-cluster-pooler-rw:5432/myapp -c "SELECT version();" 2>/dev/null || echo "Connection failed"

# Create a test table and insert data
echo -e "\n4. Creating test table and inserting data..."
kubectl run postgres-test-write --rm -i --restart=Never --image=postgres:16 -n db -- \
  psql postgresql://app:app_password@postgres-cluster-rw:5432/myapp -c "
    CREATE TABLE IF NOT EXISTS test_table (
      id SERIAL PRIMARY KEY,
      data TEXT,
      created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    INSERT INTO test_table (data) VALUES ('Test data from $(date)');
    SELECT COUNT(*) as record_count FROM test_table;
  " 2>/dev/null || echo "Write operation failed"

# Verify replication
echo -e "\n5. Verifying data replication to read replicas..."
sleep 2
kubectl run postgres-test-read --rm -i --restart=Never --image=postgres:16 -n db -- \
  psql postgresql://app:app_password@postgres-cluster-ro:5432/myapp -c "
    SELECT COUNT(*) as record_count FROM test_table;
    SELECT * FROM test_table ORDER BY created_at DESC LIMIT 5;
  " 2>/dev/null || echo "Read operation failed"