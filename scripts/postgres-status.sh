#!/bin/bash

for i in 0 1 2; do 
  echo "=== postgres-$i ==="; 
  kubectl exec -n db postgres-$i -- pg_isready -U postgres && echo "Ready" || echo "Not ready"; 
done