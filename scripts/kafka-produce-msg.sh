#!/usr/bin/env bash

export CLUSTER=$1
export TOPIC="test-topic"

if [ -z "$CLUSTER" ]; then
  echo "Usage: $0 <cluster-name>"
  echo "Example: $0 persistent-cluster"
  exit 1
fi

kubectl -n streaming run kafka-test-producer -ti \
  --image=quay.io/strimzi/kafka:0.47.0-kafka-4.0.0 --rm=true --restart=Never \
-- bin/kafka-console-producer.sh --topic "$TOPIC" \
  --bootstrap-server "${CLUSTER}-kafka-bootstrap:9092"


sleep 3

echo ""
echo "Cleanup pod, just in case it hangs"
kubectl -n streaming delete pod kafka-test-producer