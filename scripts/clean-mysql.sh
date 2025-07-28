#!/bin/bash
# Script to clean up MySQL deployment and start fresh

echo "🧹 Cleaning up MySQL deployment..."

# Delete the statefulset
echo "Deleting MySQL statefulset..."
kubectl delete statefulset mysql -n db --force --grace-period=0 2>/dev/null || true

# Delete all MySQL pods
echo "Deleting MySQL pods..."
kubectl delete pods -n db -l app=mysql --force --grace-period=0 2>/dev/null || true

# Delete PVCs
echo "Deleting MySQL PVCs..."
kubectl delete pvc -n db -l app=mysql 2>/dev/null || true
for i in 0 1 2; do
  kubectl delete pvc data-mysql-$i -n db 2>/dev/null || true
done

# Delete the init job if it exists
echo "Deleting init job..."
kubectl delete job mysql-innodb-cluster-init -n db 2>/dev/null || true

# Wait a bit for cleanup
echo "Waiting for cleanup to complete..."
sleep 5

echo "✅ MySQL cleanup complete. You can now redeploy with:"
echo "   kubectl apply -f k8s/mysql/innodb-cluster-statefulset.yaml"