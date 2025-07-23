#!/bin/bash
set -e

# Load common configuration
source "$(dirname "$0")/config.env"

echo "🧪 Testing ${PROJECT_NAME} setup..."

# Check if cluster exists
if ! kind get clusters | grep -q "$CLUSTER_NAME"; then
    echo "❌ Kind cluster '$CLUSTER_NAME' not found. Run './setup.sh' first."
    exit 1
fi

# Check kubectl context
if ! kubectl config current-context | grep -q "$KUBECTL_CONTEXT"; then
    echo "⚠️  Setting kubectl context..."
    kubectl config use-context "$KUBECTL_CONTEXT"
fi

# Check if Tiltfile exists and has basic syntax
echo "🔍 Checking Tiltfile..."
if [ ! -f "Tiltfile" ]; then
    echo "❌ Tiltfile not found"
    exit 1
fi

# Check if Tilt is already running
echo "🔍 Checking if Tilt is running..."
TILT_RUNNING=false
if lsof -ti:10350 > /dev/null 2>&1; then
    TILT_RUNNING=true
    echo "ℹ️  Tilt is already running on port 10350"
elif curl -s http://localhost:10350 > /dev/null 2>&1; then
    TILT_RUNNING=true
    echo "ℹ️  Tilt is already running on port 10350"
else
    echo "✅ Tilt is not running - ready for 'tilt up'"
fi

# Validate Tiltfile syntax using tilt ci with short timeout (only if Tilt is not running)
if [ "$TILT_RUNNING" = false ]; then
    echo "🔍 Validating Tiltfile syntax..."
    if tilt ci --timeout=2s > /dev/null 2>&1; then
        echo "✅ Tiltfile syntax is valid"
    elif tilt ci --timeout=2s 2>&1 | grep -q "Loading Tiltfile"; then
        echo "✅ Tiltfile syntax is valid (loaded successfully)"
    else
        echo "❌ Tiltfile has syntax errors. Run 'tilt ci --timeout=5s' for details."
        exit 1
    fi
else
    echo "⏭️  Skipping Tiltfile validation (Tilt already running)"
fi

# Check if required tools are available
echo "🔍 Checking required tools..."
for tool in kind kubectl tilt; do
    if command -v "$tool" &> /dev/null; then
        echo "✅ $tool is installed"
    else
        echo "❌ $tool is not installed"
        exit 1
    fi
done

# Provide appropriate next steps based on Tilt status
echo ""
if [ "$TILT_RUNNING" = true ]; then
    echo "✅ All checks passed! Tilt is already running."
    echo "🌐 Access the Tilt UI at: http://localhost:10350"
    echo "📊 Access Grafana at: http://localhost:3000 (admin/admin)"
    echo ""
    echo "💡 Next steps:"
    echo "   - Open http://localhost:10350 to see Tilt status"
    echo "   - If you need to restart Tilt, run 'tilt down' first, then 'tilt up'"
else
    echo "✅ All checks passed! You can now run 'tilt up'"
fi