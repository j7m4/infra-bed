# Tiltfile for OpenTelemetry eBPF Profile Integration POC

# Load extensions
load('ext://namespace', 'namespace_create')

# Check if Kind cluster exists and use tacops-dev as fallback
cluster_exists = str(local('kind get clusters | grep -q go-infra-spikes && echo "exists" || echo "not found"', quiet=True)).strip()
if cluster_exists == "not found":
    print("⚠️  Kind cluster 'go-infra-spikes' not found. Using tacops-dev cluster instead.")
    cluster_name = 'tacops-dev'
else:
    cluster_name = 'go-infra-spikes'

# Set kubectl context
local('kubectl config use-context kind-' + cluster_name, quiet=True)

# Create namespace
namespace_create('observability')
namespace_create('db')

# Deploy Grafana LGTM stack
k8s_yaml('k8s/lgtm/deployment.yaml', allow_duplicates=True)
k8s_resource('lgtm', 
  port_forwards=[
    '3000:3000',  # Grafana UI
    '3200:3200',  # Tempo HTTP
    '4317:4317',  # OTLP gRPC
  ],
  labels=['o11y']
)

# Deploy Pyroscope
k8s_yaml('k8s/pyroscope/deployment.yaml')
k8s_resource('pyroscope',
  port_forwards=['4040:4040'],
  labels=['o11y']
)

# Deploy Grafana Alloy
k8s_yaml([
  'k8s/alloy/configmap.yaml',
  'k8s/alloy/deployment.yaml'
])
k8s_resource('alloy',
  labels=['o11y'],
  resource_deps=['lgtm', 'pyroscope']
)

# Helper commands
local_resource(
  'grafana-login',
  cmd='echo "Grafana URL: http://localhost:3000\\nUsername: admin\\nPassword: admin"',
  labels=['helpers']
)


# Print cluster info
print("""
🚀 OpenTelemetry Profiling with Grafana Pyroscope

Access points:
- Grafana UI: http://localhost:3000 (admin/admin)
- Tempo: http://localhost:3200
- Pyroscope: http://localhost:4040
- OTLP endpoint: localhost:4317

Next steps:
1. Wait for all resources to be ready
2. Access Grafana to verify LGTM stack is working
3. Sample app is running with pprof enabled on port 6060
4. View profiles in Grafana Pyroscope (integrated in LGTM stack)
""")

# Development tips
local_resource('cluster-info-cmd',
  cmd='kubectl cluster-info',
  labels=['helpers']
)

local_resource('get-pods-cmd', 
  cmd='kubectl get pods -n observability',
  labels=['helpers']
)

# Build and deploy sample app
docker_build(
  'go-spikes:dev',
  './go-spikes',
  dockerfile='./go-spikes/Dockerfile',
  live_update=[
    sync('./go-spikes/', '/app/'),
    run('cd /app && go build -o /root/go-spikes ./cmd/main.go', trigger=['**/*.go'])
  ]
)

k8s_yaml('go-spikes/k8s/deployment.yaml')
k8s_resource('go-spikes',
  port_forwards=['8080:8080', '6060:6060'],
  labels=['apps'],
  resource_deps=['alloy']
)

# Spike commands
local_resource('fibonacci-spike',
  cmd='curl http://localhost:8080/cpu/fibonacci/40',
  labels=['spikes'],
  resource_deps=['go-spikes']
)

# Deploy MySQL using Helm
# Ensure MySQL operator is installed/updated
local_resource('mysql-operator-deploy',
  cmd='helm upgrade --install mysql-operator mysql-operator/mysql-operator --namespace db --create-namespace --values k8s/mysql-operator/values.yaml',
  labels=['data']
)

# Deploy/Update MySQL InnoDB Cluster via Helm
local_resource('mysql-cluster-deploy',
  cmd='helm upgrade --install my-mysql-cluster mysql-operator/mysql-innodbcluster --namespace db --values k8s/mysql-operator/mysql-cluster-values.yaml',
  labels=['data'],
  resource_deps=['mysql-operator-deploy']
)

# MySQL Cluster resources - port forward in background
local_resource('mysql-port-forward',
    serve_cmd=['kubectl', 'port-forward', '-n', 'db', 'svc/my-mysql-cluster', '3306:3306', '6446:6446', '6447:6447'],
    labels=['data'],
    resource_deps=['mysql-cluster-deploy'],
    readiness_probe=probe(
      exec=exec_action(['sh', '-c', 'nc -z localhost 3306']),
      period_secs=5,
      failure_threshold=3
    )
  )

# MySQL helper commands
local_resource('mysql-status',
  cmd='./scripts/mysql-status.sh',
  labels=['mysql-ops'],
  resource_deps=['mysql-cluster-deploy']
)

local_resource('mysql-show-instances',
  cmd='./scripts/mysql-show-instances.sh',
  labels=['mysql-ops'],
  resource_deps=['mysql-cluster-deploy']
)

# MySQL cluster testing commands
local_resource('mysql-cluster-status',
  cmd='kubectl get innodbclusters -n db',
  labels=['mysql-ops'],
  resource_deps=['mysql-cluster-deploy']
)

local_resource('mysql-test-connection',
  cmd='./scripts/mysql-test-connection.sh',
  labels=['mysql-ops'],
  resource_deps=['mysql-cluster-deploy']
)

local_resource('mysql-kill-primary',
  cmd='./scripts/mysql-kill-primary.sh',
  labels=['mysql-ops'],
  resource_deps=['mysql-cluster-deploy']
)

# MySQL connection helper
local_resource('mysql-connect',
  cmd='./scripts/mysql-connect.sh',
  labels=['mysql-ops']
)

# Deploy PostgreSQL
# Option 1: Use Official PostgreSQL (simplest, manual failover)
k8s_yaml([
  'k8s/postgres/services.yaml',
  'k8s/postgres/statefulset-official.yaml'
])

# Option 2: Use Bitnami PostgreSQL with Repmgr (automatic failover)
# k8s_yaml([
#   'k8s/postgres/services.yaml',
#   'k8s/postgres/statefulset-simple.yaml',
#   'k8s/postgres/label-updater-repmgr.yaml'
# ])

# Option 3: Use Patroni (uncomment below and comment above)
# k8s_yaml([
#   'k8s/postgres/etcd.yaml',
#   'k8s/postgres/configmap.yaml',
#   'k8s/postgres/services.yaml',
#   'k8s/postgres/statefulset-patroni-official.yaml',
#   'k8s/postgres/label-updater.yaml'
# ])

# Only needed if using Patroni with etcd
# k8s_resource('postgres-etcd',
#   labels=['data'],
#   resource_deps=['lgtm']
# )

k8s_resource('postgres',
  port_forwards=['5432:5432'],
  labels=['data'],
  resource_deps=['lgtm']
)

# Only needed if using repmgr or patroni
# k8s_resource('postgres-label-updater',
#   labels=['data'],
#   resource_deps=['postgres']
# )

# PostgreSQL helper commands
local_resource('postgres-status',
  cmd='./scripts/postgres-status.sh',
  labels=['postgres-ops'],
  resource_deps=['postgres']
)

local_resource('postgres-list-dbs',
  cmd='kubectl exec -n db postgres-0 -- psql -U postgres -c "\\l"',
  labels=['postgres-ops'],
  resource_deps=['postgres']
)

# Failover testing commands (manual for official PostgreSQL)
local_resource('postgres-kill-pod',
  cmd='''
    echo "Killing postgres-0 pod..."
    kubectl delete pod -n db postgres-0 --grace-period=0 --force
    echo "Pod killed. Note: With official PostgreSQL, failover is manual."
  ''',
  labels=['postgres-failover'],
  resource_deps=['postgres']
)

local_resource('postgres-manual-failover-info',
  cmd='echo "Official PostgreSQL requires manual failover configuration. Consider using Option 2 (Bitnami with repmgr) or Option 3 (Patroni) for automatic failover."',
  labels=['postgres-failover']
)

local_resource('postgres-test-write',
  cmd='''kubectl exec -n db postgres-0 -- psql -U postgres -d postgres -c "
    CREATE TABLE IF NOT EXISTS test_table (id SERIAL PRIMARY KEY, data TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);
    INSERT INTO test_table (data) VALUES (\'Test write at $(date)\');
    SELECT * FROM test_table ORDER BY id DESC LIMIT 5;"''',
  labels=['postgres-failover'],
  resource_deps=['postgres']
)

local_resource('postgres-test-read',
  cmd='kubectl exec -n db postgres-1 -- psql -U postgres -d postgres -c "SELECT * FROM test_table ORDER BY id DESC LIMIT 5;"',
  labels=['postgres-failover'],
  resource_deps=['postgres']
)

# PostgreSQL connection helper
local_resource('postgres-connect',
  cmd='echo "PostgreSQL connection string: psql -h localhost -p 5432 -U app -d postgres\\nPassword: app_password"',
  labels=['postgres-ops']
)

