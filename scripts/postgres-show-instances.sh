#!/bin/bash

echo "=== PostgreSQL Cluster Instances ==="
echo ""

# Get primary pod
PRIMARY=$(kubectl get pods -n db -l cnpg.io/cluster=postgres-cluster,cnpg.io/instanceRole=primary -o jsonpath='{.items[0].metadata.name}')
echo "PRIMARY: $PRIMARY"

# Get replica pods
echo "REPLICAS:"
kubectl get pods -n db -l cnpg.io/cluster=postgres-cluster,cnpg.io/instanceRole=replica -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}'

echo ""
echo "=== Detailed Instance Information ==="
kubectl get pods -n db -l cnpg.io/cluster=postgres-cluster \
  -o custom-columns=NAME:.metadata.name,ROLE:.metadata.labels.cnpg\\.io/instanceRole,STATUS:.status.phase,NODE:.spec.nodeName,AGE:.metadata.creationTimestamp

echo ""
echo "=== CloudNativePG Cluster Description ==="
kubectl describe cluster postgres-cluster -n db | grep -A 10 "Status:"