#!/bin/bash
set -e

echo "🚀 Setting up OpenTelemetry Profiling environment..."

# Check if kind is installed
if ! command -v kind &> /dev/null; then
    echo "❌ kind is not installed. Please install it first."
    exit 1
fi

# Check if tilt is installed
if ! command -v tilt &> /dev/null; then
    echo "❌ tilt is not installed. Please install it first."
    exit 1
fi


KIND_CONFIG="kind-config.yaml"

# Create Kind cluster if it doesn't exist
if ! kind get clusters | grep -q otel-profiling-cluster; then
    echo "📦 Creating Kind cluster..."
    kind create cluster --config=$KIND_CONFIG
else
    echo "✅ Kind cluster already exists"
fi

# Set kubectl context
echo "🔧 Setting kubectl context..."
kubectl config use-context kind-otel-profiling-cluster

# Verify cluster is ready
echo "🔍 Verifying cluster..."
kubectl cluster-info --context kind-otel-profiling-cluster

echo "✅ Setup complete! Run 'tilt up' to start the development environment."