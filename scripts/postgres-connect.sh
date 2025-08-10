#!/bin/bash

echo "=== PostgreSQL Connection Helper ==="
echo ""
echo "Choose connection method:"
echo "1. Connect to primary (read-write)"
echo "2. Connect to read-only replica"
echo "3. Connect through PgBouncer pooler"
echo "4. Connect as superuser (postgres)"
echo ""
read -p "Enter choice (1-4): " choice

case $choice in
  1)
    echo "Connecting to primary (read-write)..."
    kubectl run postgres-client --rm -it --restart=Never --image=postgres:16 -n db -- \
      psql postgresql://app:app_password@postgres-cluster-rw:5432/myapp
    ;;
  2)
    echo "Connecting to read-only replica..."
    kubectl run postgres-client --rm -it --restart=Never --image=postgres:16 -n db -- \
      psql postgresql://app:app_password@postgres-cluster-ro:5432/myapp
    ;;
  3)
    echo "Connecting through PgBouncer pooler..."
    kubectl run postgres-client --rm -it --restart=Never --image=postgres:16 -n db -- \
      psql postgresql://app:app_password@postgres-cluster-pooler-rw:5432/myapp
    ;;
  4)
    echo "Connecting as superuser..."
    PRIMARY=$(kubectl get pods -n db -l cnpg.io/cluster=postgres-cluster,cnpg.io/instanceRole=primary -o jsonpath='{.items[0].metadata.name}')
    kubectl exec -it -n db $PRIMARY -c postgres -- psql -U postgres
    ;;
  *)
    echo "Invalid choice"
    exit 1
    ;;
esac