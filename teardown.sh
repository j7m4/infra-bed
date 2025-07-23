#!/bin/bash
set -e

# Load common configuration
source "$(dirname "$0")/config.env"

echo "🔥 Tearing down ${PROJECT_NAME} environment..."

# Check if kind is installed
if ! command -v kind &> /dev/null; then
    echo "❌ kind is not installed. Cannot proceed with teardown."
    exit 1
fi

# Check if cluster exists
if kind get clusters | grep -q "$CLUSTER_NAME"; then
    echo "🗑️  Deleting Kind cluster: $CLUSTER_NAME..."
    kind delete cluster --name="$CLUSTER_NAME"
    echo "✅ Kind cluster deleted successfully"
else
    echo "ℹ️  Kind cluster '$CLUSTER_NAME' does not exist or is already deleted"
fi

# Clean up kubectl context (if it exists)
if kubectl config get-contexts | grep -q "$KUBECTL_CONTEXT"; then
    echo "🧹 Removing kubectl context: $KUBECTL_CONTEXT..."
    kubectl config delete-context "$KUBECTL_CONTEXT" 2>/dev/null || true
    echo "✅ kubectl context removed"
else
    echo "ℹ️  kubectl context '$KUBECTL_CONTEXT' does not exist"
fi

echo "✅ Teardown complete! All resources have been cleaned up." 