#!/bin/bash

echo "=== PostgreSQL Cluster Status ==="
kubectl get clusters -n db postgres-cluster

echo -e "\n=== PostgreSQL Pods ==="
kubectl get pods -n db -l cnpg.io/cluster=postgres-cluster

echo -e "\n=== Instance Status ==="
for i in 1 2 3; do 
  POD="postgres-cluster-$i"
  echo "=== $POD ==="
  kubectl exec -n db $POD -c postgres -- \
    psql -U postgres -c "SELECT pg_is_in_recovery(), inet_server_addr(), current_setting('cluster_name');" 2>/dev/null || echo "Not ready"
done

echo -e "\n=== Replication Status ==="
PRIMARY=$(kubectl get pods -n db -l cnpg.io/cluster=postgres-cluster,cnpg.io/instanceRole=primary -o jsonpath='{.items[0].metadata.name}')
if [ ! -z "$PRIMARY" ]; then
  echo "Primary: $PRIMARY"
  kubectl exec -n db $PRIMARY -c postgres -- \
    psql -U postgres -c "SELECT client_addr, state, sync_state FROM pg_stat_replication;" 2>/dev/null
fi