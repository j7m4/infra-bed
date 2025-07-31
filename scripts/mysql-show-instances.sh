#!/bin/bash

for i in 0 1 2; do 
  echo "=== my-mysql-cluster-$i ==="; 
  kubectl exec -n db -c mysql my-mysql-cluster-$i -- \
          mysql -u root -pmysql-root-password \
                -e "SELECT @@hostname as instance" 2>/dev/null || echo "Not ready"; 
done