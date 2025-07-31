#!/bin/bash

# USER="app"
# PASSWORD="app_password"
# DATABASE="myapp"
USER="root"
PASSWORD="mysql-root-password"
DATABASE="mysql"

echo "Testing MySQL connections through router..."
echo "Primary (RW) - port 6446 - from cluster:"
kubectl run mysql-test-rw --rm -it --restart=Never --image=mysql:8.0 -n db -- \
  mysql -h my-mysql-cluster -P 6446 -u $USER -p$PASSWORD -D $DATABASE \
        -e "SELECT @@hostname" 2>&1 | grep -E "my-mysql-cluster-[0-9]|ERROR" || echo "Connection test failed"
echo ""
echo "Read-only - port 6447 - from cluster:"
kubectl run mysql-test-ro --rm -it --restart=Never --image=mysql:8.0 -n db -- \
  mysql -h my-mysql-cluster -P 6447 -u $USER -p$PASSWORD -D $DATABASE \
        -e "SELECT @@hostname" 2>&1 | grep -E "my-mysql-cluster-[0-9]|ERROR" || echo "Connection test failed"
echo ""

if command -v mysql >/dev/null 2>&1; then
  echo "Primary (RW) - port 6446 - from host:"
  mysql --protocol=tcp -h localhost -P 6446 -u $USER -p$PASSWORD -D $DATABASE \
        -e "SELECT @@hostname" 2>&1 | grep -E "my-mysql-cluster-[0-9]|ERROR" || echo "Connection test failed"
echo ""
else
  echo "mysql command not found on the host"
fi

if command -v mysql >/dev/null 2>&1; then
  echo "Read-only - port 6447 - from host:"
  mysql --protocol=tcp -h localhost -P 6447 -u $USER -p$PASSWORD -D $DATABASE \
        -e "SELECT @@hostname" 2>&1 | grep -E "my-mysql-cluster-[0-9]|ERROR" || echo "Connection test failed"
echo ""
else
  echo "mysql command not found on the host"
fi