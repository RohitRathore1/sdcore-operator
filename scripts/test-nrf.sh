#!/bin/bash

# Script to test the NRF operator implementation

set -e

echo "Testing NRF operator implementation..."

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "Error: kubectl is not installed or not in PATH"
    exit 1
fi

# Deploy the NRF test resource
echo "Deploying NRF test resource..."
./scripts/deploy.sh --test --nrf

# Wait for the NRF deployment to be ready
echo "Waiting for NRF deployment to be ready..."
kubectl wait --for=condition=available --timeout=300s deployment/example-nrf

# Check if the NRF service is created
echo "Checking NRF service..."
kubectl get service example-nrf

# Check if the NRF ConfigMap is created
echo "Checking NRF ConfigMap..."
kubectl get configmap example-nrf

# Get the NRF pod logs
echo "NRF pod logs:"
NRF_POD=$(kubectl get pods -l app=nrf,nf-name=example-nrf -o jsonpath='{.items[0].metadata.name}')
kubectl logs $NRF_POD

# Test NRF connectivity
echo "Testing NRF connectivity..."
kubectl run -i --rm --restart=Never curl-test --image=curlimages/curl -- \
  curl -s http://example-nrf:8000/nnrf-nfm/v1/nf-instances || true

echo "NRF test completed." 