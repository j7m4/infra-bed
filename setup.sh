#!/bin/bash
set -e

# Load common configuration
source "$(dirname "$0")/config.env"

echo "🚀 Setting up ${PROJECT_NAME} environment..."

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

# Create Kind cluster if it doesn't exist
if ! kind get clusters | grep -q "$CLUSTER_NAME"; then
    echo "📦 Creating Kind cluster: $CLUSTER_NAME..."
    kind create cluster --name="$CLUSTER_NAME" --config="$KIND_CONFIG"
else
    echo "✅ Kind cluster '$CLUSTER_NAME' already exists"
fi

# Set kubectl context
echo "🔧 Setting kubectl context..."
kubectl config use-context "$KUBECTL_CONTEXT"

# Verify cluster is ready
echo "🔍 Verifying cluster..."
kubectl cluster-info --context "$KUBECTL_CONTEXT"

echo "✅ Setup complete! Run 'tilt up' to start the development environment."