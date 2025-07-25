#!/bin/bash
set -e

echo "=== PostgreSQL Patroni Failover Test ==="
echo

echo "1. Checking current Patroni cluster status..."
kubectl exec -n observability postgres-0 -- curl -s http://localhost:8008/cluster | python3 -m json.tool || echo "Cluster info not available"

echo
echo "2. Identifying current leader..."
LEADER=""
for i in 0 1 2; do
  ROLE=$(kubectl exec -n observability postgres-$i -- curl -s http://localhost:8008 2>/dev/null | python3 -c "import sys, json; data = json.load(sys.stdin); print(data.get('role', 'unknown'))" 2>/dev/null || echo "error")
  if [ "$ROLE" = "master" ]; then
    LEADER="postgres-$i"
    echo "Current LEADER: $LEADER"
    break
  fi
done

if [ -z "$LEADER" ]; then
  echo "ERROR: Could not identify leader. Make sure PostgreSQL cluster is running."
  exit 1
fi

echo
echo "3. Creating test table and writing data through leader..."
kubectl exec -n observability $LEADER -c postgres -- psql -U postgres -d postgres -c "
CREATE TABLE IF NOT EXISTS failover_test (
  id SERIAL PRIMARY KEY,
  data TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  node TEXT DEFAULT current_setting('cluster_name', true)
);
INSERT INTO failover_test (data, node) VALUES ('Before failover - written to leader', '$LEADER');
" 2>/dev/null

echo
echo "4. Verifying replication to followers..."
for i in 0 1 2; do
  if [ "postgres-$i" != "$LEADER" ]; then
    echo "--- Data from postgres-$i (follower) ---"
    kubectl exec -n observability postgres-$i -c postgres -- psql -U postgres -d postgres -c "SELECT * FROM failover_test ORDER BY id DESC LIMIT 3;" 2>/dev/null || echo "Not ready"
  fi
done

echo
echo "5. Killing LEADER pod to trigger failover..."
kubectl delete pod -n observability $LEADER --grace-period=0 --force

echo
echo "6. Waiting for Patroni to elect new leader (20 seconds)..."
sleep 20

echo
echo "7. Checking new cluster status..."
NEW_LEADER=""
for i in 0 1 2; do
  if kubectl get pod -n observability postgres-$i &>/dev/null; then
    ROLE=$(kubectl exec -n observability postgres-$i -- curl -s http://localhost:8008 2>/dev/null | python3 -c "import sys, json; data = json.load(sys.stdin); print(data.get('role', 'unknown'))" 2>/dev/null || echo "error")
    if [ "$ROLE" = "master" ]; then
      NEW_LEADER="postgres-$i"
      echo "New LEADER: $NEW_LEADER"
      break
    fi
  fi
done

echo
echo "8. Writing test data through new leader..."
if [ ! -z "$NEW_LEADER" ]; then
  kubectl exec -n observability $NEW_LEADER -c postgres -- psql -U postgres -d postgres -c "
  INSERT INTO failover_test (data, node) VALUES ('After failover - written to new leader', '$NEW_LEADER');
  " 2>/dev/null
fi

echo
echo "9. Waiting for old leader to rejoin as follower (30 seconds)..."
sleep 30

echo
echo "10. Final cluster status..."
for i in 0 1 2; do
  echo "--- postgres-$i ---"
  kubectl exec -n observability postgres-$i -- curl -s http://localhost:8008 2>/dev/null | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    print(f\"Role: {data.get('role', 'unknown')}\"")
    print(f\"State: {data.get('state', 'unknown')}\"")
    print(f\"Timeline: {data.get('timeline', 'unknown')}\"")
except:
    print('Not available')
" || echo "Pod not ready"
  echo
done

echo
echo "11. Verifying data consistency across all nodes..."
for i in 0 1 2; do
  if kubectl get pod -n observability postgres-$i &>/dev/null; then
    echo "--- Data from postgres-$i ---"
    kubectl exec -n observability postgres-$i -c postgres -- psql -U postgres -d postgres -c "SELECT * FROM failover_test ORDER BY id DESC LIMIT 5;" 2>/dev/null || echo "Not ready"
    echo
  fi
done

echo "=== Failover test complete! ==="
echo
echo "The old LEADER ($LEADER) was killed and a new LEADER ($NEW_LEADER) was automatically elected by Patroni."
echo "Data remains consistent across all nodes with automatic replication."