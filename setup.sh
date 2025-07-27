#!/bin/bash
set -e

# Parse command line arguments
PRELOAD_IMAGES=true
while [[ $# -gt 0 ]]; do
  case $1 in
    --no-preload)
      PRELOAD_IMAGES=false
      shift
      ;;
    -h|--help)
      echo "Usage: $0 [OPTIONS]"
      echo "Options:"
      echo "  --no-preload    Skip preloading Docker images into Kind cluster"
      echo "  -h, --help      Show this help message"
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      echo "Use '$0 --help' for usage information"
      exit 1
      ;;
  esac
done

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

# Preload images if not disabled
if [ "$PRELOAD_IMAGES" = true ]; then
    echo ""
    echo "📥 Preloading Docker images into Kind cluster..."
    echo "(Use --no-preload to skip this step)"
    ./scripts/preload-images.sh "$CLUSTER_NAME"
else
    echo ""
    echo "⏭️  Skipping image preloading (--no-preload flag used)"
fi

echo ""
echo "✅ Setup complete! Run 'tilt up' to start the development environment."