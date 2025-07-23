#!/bin/bash

echo "🧪 Testing OpenTelemetry Profiling setup..."

# Check if cluster exists
if ! kind get clusters | grep -q otel-profiling-cluster; then
    echo "❌ Kind cluster not found. Run './setup.sh' first."
    exit 1
fi

# Check kubectl context
if ! kubectl config current-context | grep -q kind-otel-profiling-cluster; then
    echo "⚠️  Setting kubectl context..."
    kubectl config use-context kind-otel-profiling-cluster
fi

# Check Tiltfile syntax
echo "🔍 Checking Tiltfile syntax..."
if tilt ci --file Tiltfile --dry-run; then
    echo "✅ Tiltfile syntax is valid"
else
    echo "❌ Tiltfile has syntax errors"
    exit 1
fi

echo "✅ All checks passed! You can now run 'tilt up'"