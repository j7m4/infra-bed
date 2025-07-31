#!/bin/bash

USER="root"
PASSWORD="mysql-root-password"
DATABASE="mysql"

if command -v mysql >/dev/null 2>&1; then
  PRIMARY=$(mysql --protocol=tcp -h localhost -P 6446 -u $USER -p$PASSWORD -D $DATABASE \
        -e "SELECT @@hostname" 2>&1 | grep -E "my-mysql-cluster-[0-9]|ERROR" || echo "")
  if [ -z "$PRIMARY" ]; then
    echo "Primary (RW) host not found"
  else
    echo "Primary (RW) - port 6446 - from host: $PRIMARY"
    echo "!!!!! WARNING !!!!! PRIMARY POD WILL BE DELETED !!!!!!"
    echo "5 seconds to cancel..."
    sleep 5
    echo "Deleting primary pod..."
    kubectl delete pod -n db $PRIMARY &
    sleep 2
  fi
  echo ""
else
  echo "mysql command not found on the host"
fi