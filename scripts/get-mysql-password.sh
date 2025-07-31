#!/bin/bash
# Helper script to retrieve MySQL root password from Kubernetes secret

# Function to get MySQL root password from Kubernetes secret
get_mysql_password() {
    local namespace="${1:-db}"
    # For MySQL operator deployment, the secret name is different
    kubectl get secret my-mysql-cluster-cluster-secret -n "$namespace" -o jsonpath='{.data.rootPassword}' 2>/dev/null | base64 -d
}

# If script is run directly, print the password
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    get_mysql_password "$@"
fi