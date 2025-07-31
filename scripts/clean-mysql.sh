#!/bin/bash
# Script to clean up MySQL Operator deployment and start fresh

echo "🧹 Cleaning up MySQL deployment..."

# Uninstall MySQL cluster
echo "Uninstalling MySQL cluster..."
helm uninstall my-mysql-cluster -n db 2>/dev/null || true

# Delete any remaining MySQL resources
echo "Deleting MySQL resources..."
kubectl delete innodbclusters --all -n db --force --grace-period=0 2>/dev/null || true
kubectl delete pods -n db -l mysql.oracle.com/cluster=my-mysql-cluster --force --grace-period=0 2>/dev/null || true

# Delete PVCs
echo "Deleting MySQL PVCs..."
kubectl delete pvc -n db -l mysql.oracle.com/cluster=my-mysql-cluster 2>/dev/null || true
for i in 0 1 2; do
  kubectl delete pvc datadir-my-mysql-cluster-$i -n db 2>/dev/null || true
done

# Optional: Uninstall MySQL operator
read -p "Do you want to uninstall the MySQL operator as well? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
  echo "Uninstalling MySQL operator..."
  helm uninstall mysql-operator -n db 2>/dev/null || true
fi

# Wait a bit for cleanup
echo "Waiting for cleanup to complete..."
sleep 5

echo "✅ MySQL cleanup complete. You can now redeploy with:"
echo "   helm upgrade --install my-mysql-cluster mysql-operator/mysql-innodbcluster --namespace db --values k8s/mysql-operator/mysql-cluster-values.yaml"