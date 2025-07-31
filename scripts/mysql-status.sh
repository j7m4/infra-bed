#!/bin/bash

# MySQL Status Check Script
# Shows hostname and uptime for MySQL instance

set -e

# Get MySQL root password
MYSQL_PASSWORD=$(./scripts/get-mysql-password.sh)

# Execute MySQL status commands
kubectl exec -n db mysql-0 -- mysql -u root -p"${MYSQL_PASSWORD}" -e "SHOW VARIABLES LIKE 'hostname'; SHOW STATUS LIKE 'Uptime';"