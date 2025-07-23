# Tiltfile for OpenTelemetry eBPF Profile Integration POC

# Load extensions
load('ext://namespace', 'namespace_create')

# Check if Kind cluster exists and use tacops-dev as fallback
cluster_exists = str(local('kind get clusters | grep -q otel-profiling-cluster && echo "exists" || echo "not found"', quiet=True)).strip()
if cluster_exists == "not found":
    print("⚠️  Kind cluster 'otel-profiling-cluster' not found. Using tacops-dev cluster instead.")
    cluster_name = 'tacops-dev'
else:
    cluster_name = 'otel-profiling-cluster'

# Set kubectl context
local('kubectl config use-context kind-' + cluster_name, quiet=True)

# Create namespace
namespace_create('observability')

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
  'sample-app:dev',
  './sample-app',
  dockerfile='./sample-app/Dockerfile',
  live_update=[
    sync('./sample-app/', '/app/'),
    run('cd /app && go build -o /root/sample-app ./cmd/main.go', trigger=['**/*.go'])
  ]
)

k8s_yaml('sample-app/k8s/deployment.yaml')
k8s_resource('sample-app',
  port_forwards=['8080:8080', '6060:6060'],
  labels=['apps'],
  resource_deps=['alloy']
)

# Load testing commands
local_resource('test-fibonacci',
  cmd='curl http://localhost:8080/cpu/fibonacci/40',
  labels=['testing'],
  resource_deps=['sample-app']
)

local_resource('test-prime',
  cmd='curl http://localhost:8080/cpu/prime/100000',
  labels=['testing'],
  resource_deps=['sample-app']
)

local_resource('test-hash',
  cmd='curl http://localhost:8080/cpu/hash/100000',
  labels=['testing'],
  resource_deps=['sample-app']
)

local_resource('test-mixed',
  cmd='curl http://localhost:8080/workload/mixed',
  labels=['testing'],
  resource_deps=['sample-app']
)