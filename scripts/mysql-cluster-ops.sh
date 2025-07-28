#!/bin/bash
# MySQL InnoDB Cluster operations script

MYSQL_PWD=$(./scripts/get-mysql-password.sh)

function cluster_status() {
    echo "🔍 Checking InnoDB Cluster status..."
    kubectl exec -n db mysql-0 -c mysql-shell -- mysqlsh --uri clusteradmin:ClusterAdmin123!@localhost:3306 --js -e "
    try {
        var cluster = dba.getCluster();
        print(JSON.stringify(cluster.status(), null, 2));
    } catch(e) {
        print('Error: ' + e.message);
    }"
}

function switch_primary() {
    local target_instance=$1
    if [ -z "$target_instance" ]; then
        echo "Usage: $0 switch_primary <instance-number>"
        echo "Example: $0 switch_primary 1"
        return 1
    fi
    
    echo "🔄 Switching primary to mysql-$target_instance..."
    kubectl exec -n db mysql-0 -c mysql-shell -- mysqlsh --uri clusteradmin:ClusterAdmin123!@localhost:3306 --js -e "
    try {
        var cluster = dba.getCluster();
        cluster.setPrimaryInstance('mysql-$target_instance.mysql-headless.db.svc.cluster.local:3306');
        print('Primary switched successfully');
        print(JSON.stringify(cluster.status(), null, 2));
    } catch(e) {
        print('Error: ' + e.message);
    }"
}

function rejoin_instance() {
    local instance=$1
    if [ -z "$instance" ]; then
        echo "Usage: $0 rejoin_instance <instance-number>"
        return 1
    fi
    
    echo "➕ Rejoining mysql-$instance to cluster..."
    kubectl exec -n db mysql-0 -c mysql-shell -- mysqlsh --uri clusteradmin:ClusterAdmin123!@localhost:3306 --js -e "
    try {
        var cluster = dba.getCluster();
        cluster.rejoinInstance('mysql-$instance.mysql-headless.db.svc.cluster.local:3306', {
            password: 'ClusterAdmin123!'
        });
        print('Instance rejoined successfully');
    } catch(e) {
        print('Error: ' + e.message);
    }"
}

function remove_instance() {
    local instance=$1
    if [ -z "$instance" ]; then
        echo "Usage: $0 remove_instance <instance-number>"
        return 1
    fi
    
    echo "➖ Removing mysql-$instance from cluster..."
    kubectl exec -n db mysql-0 -c mysql-shell -- mysqlsh --uri clusteradmin:ClusterAdmin123!@localhost:3306 --js -e "
    try {
        var cluster = dba.getCluster();
        cluster.removeInstance('mysql-$instance.mysql-headless.db.svc.cluster.local:3306', {
            force: true
        });
        print('Instance removed successfully');
    } catch(e) {
        print('Error: ' + e.message);
    }"
}

function show_replicas() {
    echo "📊 Current replication status:"
    kubectl exec -n db mysql-0 -- mysql -u root -p$MYSQL_PWD -e "
    SELECT 
        MEMBER_HOST,
        MEMBER_PORT,
        MEMBER_STATE,
        IF(MEMBER_STATE = 'ONLINE', 
           IF(MEMBER_ROLE = 'PRIMARY', '🟢 PRIMARY', '🔵 SECONDARY'), 
           '🔴 ' || MEMBER_STATE) as STATUS,
        MEMBER_VERSION
    FROM performance_schema.replication_group_members
    ORDER BY MEMBER_ROLE DESC, MEMBER_HOST;"
}

function test_failover() {
    echo "🧪 Testing automatic failover..."
    echo "Current status:"
    show_replicas
    
    # Get current primary
    PRIMARY=$(kubectl exec -n db mysql-0 -- mysql -u root -p$MYSQL_PWD -Nse "
    SELECT SUBSTRING_INDEX(MEMBER_HOST, '.', 1) 
    FROM performance_schema.replication_group_members 
    WHERE MEMBER_ROLE='PRIMARY'")
    
    echo ""
    echo "Current primary: $PRIMARY"
    echo "Killing primary pod to trigger failover..."
    kubectl delete pod -n db $PRIMARY --force --grace-period=0
    
    echo ""
    echo "Waiting for failover (30 seconds)..."
    sleep 30
    
    echo ""
    echo "New status after failover:"
    show_replicas
}

# Parse command
case "$1" in
    status)
        cluster_status
        ;;
    switch_primary)
        switch_primary $2
        ;;
    rejoin)
        rejoin_instance $2
        ;;
    remove)
        remove_instance $2
        ;;
    replicas)
        show_replicas
        ;;
    test_failover)
        test_failover
        ;;
    *)
        echo "MySQL InnoDB Cluster Operations"
        echo ""
        echo "Usage: $0 <command> [options]"
        echo ""
        echo "Commands:"
        echo "  status          - Show cluster status"
        echo "  replicas        - Show replication members"
        echo "  switch_primary  - Switch primary to specific instance"
        echo "  rejoin          - Rejoin instance to cluster"
        echo "  remove          - Remove instance from cluster"
        echo "  test_failover   - Test automatic failover"
        echo ""
        echo "Examples:"
        echo "  $0 status"
        echo "  $0 switch_primary 1"
        echo "  $0 rejoin 2"
        ;;
esac