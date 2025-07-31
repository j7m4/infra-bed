#!/bin/bash

echo "MySQL connection strings:"
echo "Primary (read/write): mysql -h localhost -P 6446 -u app -papp_password"
echo "Read-only: mysql -h localhost -P 6447 -u app -papp_password"
echo "Root access: mysql -h localhost -P 3306 -u root -pmysql-root-password"