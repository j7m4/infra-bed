# Load extensions
load('ext://namespace', 'namespace_create')

# Check if Kind cluster exists and use tacops-dev as fallback
cluster_exists = str(local('kind get clusters | grep -q infra-bed && echo "exists" || echo "not found"', quiet=True)).strip()
if cluster_exists == "not found":
    print("⚠️  Kind cluster 'infra-bed' not found. Using tacops-dev cluster instead.")
    cluster_name = 'tacops-dev'
else:
    cluster_name = 'infra-bed'

# Set kubectl context
local('kubectl config use-context kind-' + cluster_name, quiet=True)

# Create namespace
namespace_create('observability')
namespace_create('db')
namespace_create('streaming')

#########################################################
# OBSERVABILITY

k8s_yaml('k8s/lgtm/deployment.yaml', allow_duplicates=True)
k8s_resource('lgtm', 
    port_forwards=[
        '3000:3000',  # Grafana UI
        '3200:3200',  # Tempo HTTP
        '4317:4317',  # OTLP gRPC
    ],
    labels=['o11y']
)

k8s_yaml('k8s/pyroscope/deployment.yaml')
k8s_resource('pyroscope',
    port_forwards=['4040:4040'],
    labels=['o11y']
)

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

##################################################
# HELPERS

local_resource('cluster-info-cmd',
    cmd='kubectl cluster-info',
    labels=['helpers']
)

local_resource('get-pods-cmd', 
    cmd='kubectl get pods -n observability',
    labels=['helpers']
)

##################################################
# GO SPIKES

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
    labels=['spikes'],
    resource_deps=['alloy']
)

# Spike commands
local_resource('run-fibonacci',
    cmd='curl http://localhost:8080/cpu/fibonacci/40',
    labels=['spikes'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False
)

local_resource('run-produce-kafka',
    cmd='curl http://localhost:8080/kafka/produce',
    labels=['spikes'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False
)

local_resource('run-consume-kafka',
    cmd='curl http://localhost:8080/kafka/consume',
    labels=['spikes'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False
)


##############################################
# MYSQL

local_resource('deploy-operator-mysql',
    cmd="""
    helm upgrade --install mysql-operator mysql-operator/mysql-operator \
      --namespace db --create-namespace --values k8s/mysql-operator/values.yaml
    echo "MySQL Operator deployed successfully."
    """,
    labels=['mysql']
)

local_resource('install-cluster-mysql',
    cmd='helm upgrade --install my-mysql-cluster mysql-operator/mysql-innodbcluster --namespace db --values k8s/mysql-operator/mysql-cluster-values.yaml',
    labels=['mysql'],
    resource_deps=['deploy-operator-mysql'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False
)

local_resource('port-forward-mysql',
    serve_cmd=['kubectl', 'port-forward', '-n', 'db', 'svc/my-mysql-cluster', '3306:3306', '6446:6446', '6447:6447'],
    labels=['mysql-ops'],
    resource_deps=['install-cluster-mysql'],
    readiness_probe=probe(
      exec=exec_action(['sh', '-c', 'nc -z localhost 3306']),
      period_secs=5,
      failure_threshold=3
    ),
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False
  )

# MySQL helper commands
local_resource('mysql-status',
    cmd='./scripts/mysql-status.sh',
    labels=['mysql-ops'],
    resource_deps=['install-cluster-mysql'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False
)

local_resource('show-instances-mysql',
    cmd='./scripts/show-instances-mysql.sh',
    labels=['mysql-ops'],
    resource_deps=['install-cluster-mysql'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False
)

# MySQL cluster testing commands
local_resource('cluster-status-mysql',
    cmd='kubectl get innodbclusters -n db',
    labels=['mysql-ops'],
    resource_deps=['install-cluster-mysql'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False
)

local_resource('test-connection-mysql',
    cmd='./scripts/test-connection-mysql.sh',
    labels=['mysql-ops'],
    resource_deps=['install-cluster-mysql'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False
)

local_resource('mysql-kill-primary',
    cmd='./scripts/mysql-kill-primary.sh',
    labels=['mysql-ops'],
    resource_deps=['install-cluster-mysql'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False
)

# MySQL connection helper
local_resource('mysql-connect',
    cmd='./scripts/mysql-connect.sh',
    labels=['mysql-ops'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False
)

##################################################
# POSTGRES

local_resource('deploy-operator-postgres',
    cmd="""
    helm repo add cnpg https://cloudnative-pg.github.io/charts
    helm repo update
    helm upgrade --install cnpg cnpg/cloudnative-pg \
      --namespace db --create-namespace --values k8s/postgres-operator/values.yaml
    echo "CloudNativePG Operator deployed successfully."
    """,
    labels=['postgres']
)

local_resource('install-cluster-postgres',
    cmd='kubectl apply -f k8s/postgres-operator/postgres-cluster.yaml',
    labels=['postgres'],
    resource_deps=['deploy-operator-postgres'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False
)

local_resource('port-forward-postgres',
    serve_cmd=['kubectl', 'port-forward', '-n', 'db', 'svc/postgres-cluster-rw', '5432:5432'],
    labels=['postgres-ops'],
    resource_deps=['install-cluster-postgres'],
    readiness_probe=probe(
      exec=exec_action(['sh', '-c', 'nc -z localhost 5432']),
      period_secs=5,
      failure_threshold=3
    ),
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False
)

local_resource('port-forward-postgres-pooler',
    serve_cmd=['kubectl', 'port-forward', '-n', 'db', 'svc/postgres-cluster-pooler-rw', '5433:5432'],
    labels=['postgres-ops'],
    resource_deps=['install-cluster-postgres'],
    readiness_probe=probe(
      exec=exec_action(['sh', '-c', 'nc -z localhost 5433']),
      period_secs=5,
      failure_threshold=3
    ),
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False
)

# PostgreSQL helper commands
local_resource('postgres-status',
    cmd='./scripts/postgres-status.sh',
    labels=['postgres-ops'],
    resource_deps=['install-cluster-postgres'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False
)

local_resource('show-instances-postgres',
    cmd='./scripts/postgres-show-instances.sh',
    labels=['postgres-ops'],
    resource_deps=['install-cluster-postgres'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False
)

# PostgreSQL cluster testing commands
local_resource('cluster-status-postgres',
    cmd='kubectl get clusters -n db',
    labels=['postgres-ops'],
    resource_deps=['install-cluster-postgres'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False
)

local_resource('test-connection-postgres',
    cmd='./scripts/postgres-test-connection.sh',
    labels=['postgres-ops'],
    resource_deps=['install-cluster-postgres'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False
)

local_resource('postgres-kill-primary',
    cmd='./scripts/postgres-kill-primary.sh',
    labels=['postgres-ops'],
    resource_deps=['install-cluster-postgres'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False
)

# PostgreSQL connection helper
local_resource('postgres-connect',
    cmd='./scripts/postgres-connect.sh',
    labels=['postgres-ops'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False
)

##################################################
# KAFKA

local_resource('operator-install-kafka',
    cmd="""
    kubectl create namespace streaming --dry-run=client -o yaml | kubectl apply -f -
    kubectl create -f https://strimzi.io/install/latest?namespace=streaming || true
    echo "Waiting for Kafka operator to be ready..."
    sleep 10
    kubectl wait --for=condition=Available deployment/strimzi-cluster-operator -n streaming --timeout=300s || true
    echo "Kafka operator installed successfully."
    """,
    labels=['kafka'],
    resource_deps=['lgtm']
)

local_resource('install-persistent-cluster-kafka',
    cmd="""
    kubectl apply -f k8s/kafka/persistent-cluster.yaml -n streaming
    kubectl wait kafka/persistent-cluster --for=condition=Ready --timeout=300s -n streaming
    echo "Kafka persistent cluster created successfully."
    """,
    labels=['kafka'],
    resource_deps=['operator-install-kafka'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False
)

local_resource('uninstall-persistent-cluster-kafka',
    cmd="""
    kubectl delete -f k8s/kafka/persistent-cluster.yaml -n streaming
    echo "Kafka persistent cluster removed."
    """,
    labels=['kafka'],
    resource_deps=['operator-install-kafka'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False
)

