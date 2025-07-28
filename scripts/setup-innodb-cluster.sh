#!/bin/bash
# Setup InnoDB Cluster manually

set -e

MYSQL_PWD=$(./scripts/get-mysql-password.sh)

echo "🔧 Setting up InnoDB Cluster..."

# Step 1: Create cluster admin user on all instances
echo "Creating cluster admin user..."
for i in 0 1 2; do
  kubectl exec -n db mysql-$i -c mysql -- mysql -u root -p$MYSQL_PWD -e "
  CREATE USER IF NOT EXISTS 'clusteradmin'@'%' IDENTIFIED BY 'ClusterAdmin123!';
  GRANT ALL PRIVILEGES ON *.* TO 'clusteradmin'@'%' WITH GRANT OPTION;
  GRANT PERSIST_RO_VARIABLES_ADMIN ON *.* TO 'clusteradmin'@'%';
  GRANT SYSTEM_VARIABLES_ADMIN ON *.* TO 'clusteradmin'@'%';
  GRANT CLONE_ADMIN ON *.* TO 'clusteradmin'@'%';
  GRANT BACKUP_ADMIN ON *.* TO 'clusteradmin'@'%';
  GRANT GROUP_REPLICATION_STREAM ON *.* TO 'clusteradmin'@'%';
  FLUSH PRIVILEGES;" 2>/dev/null || echo "User may already exist on mysql-$i"
done

# Step 2: Configure instances for InnoDB Cluster
echo "Configuring instances..."
for i in 0 1 2; do
  echo "Configuring mysql-$i..."
  kubectl exec -n db mysql-$i -c mysql-shell -- mysqlsh --py -e "
shell.connect('clusteradmin:ClusterAdmin123!@localhost:3306')
try:
    dba.configure_instance()
    print('Instance configured')
except Exception as e:
    print(f'Configuration note: {e}')
    if 'already valid' in str(e):
        print('Instance already configured for InnoDB Cluster')
" || echo "Note: mysql-$i configuration check complete"
done

# Step 3: Create the cluster on mysql-0
echo "Creating cluster on mysql-0..."
kubectl exec -n db mysql-0 -c mysql-shell -- mysqlsh --py -e "
shell.connect('clusteradmin:ClusterAdmin123!@localhost:3306')
try:
    cluster = dba.get_cluster()
    print('Cluster already exists')
except:
    print('Creating new cluster...')
    cluster = dba.create_cluster('mycluster', {
        'exitStateAction': 'OFFLINE_MODE',
        'autoRejoinTries': 3,
        'ipAllowlist': '10.0.0.0/8,172.16.0.0/12,192.168.0.0/16'
    })
    print('Cluster created')

print(cluster.status())
"

# Step 4: Add other instances to cluster
echo "Adding instances to cluster..."
for i in 1 2; do
  echo "Adding mysql-$i..."
  kubectl exec -n db mysql-0 -c mysql-shell -- mysqlsh --py -e "
shell.connect('clusteradmin:ClusterAdmin123!@localhost:3306')
cluster = dba.get_cluster()
try:
    cluster.add_instance('clusteradmin:ClusterAdmin123!@mysql-$i.mysql-headless.db.svc.cluster.local:3306', {
        'recoveryMethod': 'clone',
        'waitRecovery': 2
    })
    print('Instance added')
except Exception as e:
    print(f'Error adding instance: {e}')
    if 'already' in str(e).lower():
        print('Instance already in cluster')
"
done

# Step 5: Create router user
echo "Creating router user..."
kubectl exec -n db mysql-0 -c mysql-shell -- mysqlsh --py -e "
shell.connect('clusteradmin:ClusterAdmin123!@localhost:3306')
cluster = dba.get_cluster()
try:
    cluster.setup_router_account('mysqlrouter', {'password': 'RouterPass123!'})
    print('Router account created')
except Exception as e:
    print(f'Router account error: {e}')
"

# Step 6: Create app user
echo "Creating app user..."
kubectl exec -n db mysql-0 -c mysql -- mysql -u clusteradmin -pClusterAdmin123! -e "
CREATE USER IF NOT EXISTS 'app'@'%' IDENTIFIED BY 'app_password';
GRANT ALL PRIVILEGES ON *.* TO 'app'@'%';
CREATE DATABASE IF NOT EXISTS test;
FLUSH PRIVILEGES;" 2>/dev/null

# Show final status
echo ""
echo "✅ Cluster setup complete!"
echo ""
./scripts/mysql-cluster-ops.sh status