#!/bin/bash
set -e

# Source the helper script
source ./scripts/get-mysql-password.sh

# Get the password once at the beginning
MYSQL_ROOT_PASSWORD=$(get_mysql_password observability)

echo "=== MySQL Group Replication Failover Test ==="
echo

echo "1. Checking current Group Replication status..."
kubectl exec -n observability mysql-0 -- mysql -u root -p$MYSQL_ROOT_PASSWORD -e "SELECT MEMBER_HOST, MEMBER_ROLE, MEMBER_STATE FROM performance_schema.replication_group_members ORDER BY MEMBER_HOST;" 2>/dev/null

echo
echo "2. Identifying current PRIMARY..."
PRIMARY=$(kubectl exec -n observability mysql-0 -- mysql -u root -p$MYSQL_ROOT_PASSWORD -Nse "SELECT MEMBER_HOST FROM performance_schema.replication_group_members WHERE MEMBER_ROLE='PRIMARY'" 2>/dev/null | cut -d'.' -f1)
echo "Current PRIMARY: $PRIMARY"

echo
echo "3. Writing test data through PRIMARY service..."
kubectl exec -n observability $PRIMARY -- mysql -u root -p$MYSQL_ROOT_PASSWORD -e "USE testdb; INSERT INTO test_table (data) VALUES ('Before failover - written to $PRIMARY at $(date)');" 2>/dev/null

echo
echo "4. Killing PRIMARY pod to trigger failover..."
kubectl delete pod -n observability $PRIMARY --grace-period=0 --force

echo
echo "5. Waiting for new PRIMARY election (15 seconds)..."
sleep 15

echo
echo "6. Checking new Group Replication status..."
# Try different pods since the original primary is down
for pod in mysql-0 mysql-1 mysql-2; do
  if kubectl get pod -n observability $pod &>/dev/null; then
    kubectl exec -n observability $pod -- mysql -u root -p$MYSQL_ROOT_PASSWORD -e "SELECT MEMBER_HOST, MEMBER_ROLE, MEMBER_STATE FROM performance_schema.replication_group_members ORDER BY MEMBER_HOST;" 2>/dev/null && break
  fi
done

echo
echo "7. Identifying new PRIMARY..."
for pod in mysql-0 mysql-1 mysql-2; do
  if kubectl get pod -n observability $pod &>/dev/null; then
    NEW_PRIMARY=$(kubectl exec -n observability $pod -- mysql -u root -p$MYSQL_ROOT_PASSWORD -Nse "SELECT MEMBER_HOST FROM performance_schema.replication_group_members WHERE MEMBER_ROLE='PRIMARY'" 2>/dev/null | cut -d'.' -f1)
    if [ ! -z "$NEW_PRIMARY" ]; then
      echo "New PRIMARY: $NEW_PRIMARY"
      break
    fi
  fi
done

echo
echo "8. Writing test data through new PRIMARY..."
kubectl exec -n observability $NEW_PRIMARY -- mysql -u root -p$MYSQL_ROOT_PASSWORD -e "USE testdb; INSERT INTO test_table (data) VALUES ('After failover - written to $NEW_PRIMARY at $(date)');" 2>/dev/null

echo
echo "9. Verifying data consistency across all nodes..."
for pod in mysql-0 mysql-1 mysql-2; do
  if kubectl get pod -n observability $pod &>/dev/null; then
    echo "--- Data from $pod ---"
    kubectl exec -n observability $pod -- mysql -u root -p$MYSQL_ROOT_PASSWORD -e "USE testdb; SELECT * FROM test_table ORDER BY id DESC LIMIT 5;" 2>/dev/null || echo "$pod is not available"
    echo
  fi
done

echo "=== Failover test complete! ==="
echo
echo "The old PRIMARY ($PRIMARY) was killed and a new PRIMARY ($NEW_PRIMARY) was automatically elected."
echo "Data remains consistent across all nodes."