#!/bin/bash

set -e

echo "==========================================="
echo "Deploying Metrics Exporters for Databases"
echo "==========================================="

# Check if databases are running
echo "Checking database deployments..."

# Check Kafka
if kubectl get kafka persistent-cluster -n streaming &>/dev/null; then
    echo "✓ Kafka cluster found"
    KAFKA_READY=true
else
    echo "✗ Kafka cluster not found - skipping Kafka exporter"
    KAFKA_READY=false
fi

# Check MySQL
if kubectl get pods -n db -l component=mysqld &>/dev/null && [ $(kubectl get pods -n db -l component=mysqld -o json | jq '.items | length') -gt 0 ]; then
    echo "✓ MySQL cluster found"
    MYSQL_READY=true
else
    echo "✗ MySQL cluster not found - skipping MySQL exporter"
    MYSQL_READY=false
fi

# Check PostgreSQL
if kubectl get cluster postgres-cluster -n db &>/dev/null; then
    echo "✓ PostgreSQL cluster found"
    POSTGRES_READY=true
else
    echo "✗ PostgreSQL cluster not found - skipping PostgreSQL exporter"
    POSTGRES_READY=false
fi

echo ""
echo "Deploying exporters..."

# Deploy Kafka exporter
if [ "$KAFKA_READY" = true ]; then
    echo "Deploying Kafka exporter..."
    kubectl apply -f k8s/exporters/kafka-exporter.yaml
    echo "✓ Kafka exporter deployed"
fi

# Deploy MySQL exporter with user creation
if [ "$MYSQL_READY" = true ]; then
    echo "Creating MySQL exporter user..."
    # Get MySQL root password
    MYSQL_ROOT_PASSWORD=$(kubectl get secret my-mysql-cluster-cluster-secret -n db -o jsonpath='{.data.rootPassword}' | base64 -d)
    
    # Create exporter user
    kubectl exec -n db my-mysql-cluster-0 -- mysql -uroot -p${MYSQL_ROOT_PASSWORD} -e "
    CREATE USER IF NOT EXISTS 'exporter'@'%' IDENTIFIED BY 'exporter';
    GRANT PROCESS, REPLICATION CLIENT, SELECT ON *.* TO 'exporter'@'%';
    GRANT SELECT ON performance_schema.* TO 'exporter'@'%';
    FLUSH PRIVILEGES;
    " 2>/dev/null || echo "Note: MySQL exporter user might already exist"
    
    echo "Deploying MySQL exporter..."
    kubectl apply -f k8s/exporters/mysql-exporter.yaml
    echo "✓ MySQL exporter deployed"
fi

# Deploy PostgreSQL exporter
if [ "$POSTGRES_READY" = true ]; then
    echo "Deploying PostgreSQL exporter..."
    kubectl apply -f k8s/exporters/postgres-exporter.yaml
    echo "✓ PostgreSQL exporter deployed"
fi

echo ""
echo "Deploying Grafana dashboards..."
kubectl apply -f k8s/grafana-dashboards/kafka-dashboard.yaml 2>/dev/null || true
kubectl apply -f k8s/grafana-dashboards/mysql-dashboard.yaml 2>/dev/null || true
kubectl apply -f k8s/grafana-dashboards/postgres-dashboard.yaml 2>/dev/null || true
echo "✓ Grafana dashboards deployed"

echo ""
echo "Restarting Alloy to pick up new configuration..."
kubectl rollout restart deployment/alloy -n observability
kubectl rollout status deployment/alloy -n observability --timeout=60s

echo ""
echo "==========================================="
echo "Deployment Complete!"
echo "==========================================="
echo ""
echo "Next steps:"
echo "1. Access Grafana at http://localhost:3000 (admin/admin)"
echo "2. Navigate to Dashboards to see:"
echo "   - Kafka Cluster Metrics"
echo "   - MySQL Database Metrics"
echo "   - PostgreSQL Database Metrics"
echo ""
echo "To verify metrics are being scraped:"
echo "  kubectl logs -n observability deployment/alloy | grep -E 'kafka|mysql|postgres'"
echo ""
echo "To check exporter status:"
if [ "$KAFKA_READY" = true ]; then
    echo "  curl http://localhost:9308/metrics  # Kafka metrics (need port-forward)"
fi
if [ "$MYSQL_READY" = true ]; then
    echo "  curl http://localhost:9104/metrics  # MySQL metrics (need port-forward)"
fi
if [ "$POSTGRES_READY" = true ]; then
    echo "  curl http://localhost:9187/metrics  # PostgreSQL metrics (need port-forward)"
fi