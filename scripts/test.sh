#!/bin/bash

set -e

echo "Testing SDCore Operator..."

# Check if the simple operator is running
echo "Checking if simple operator is running..."
kubectl wait --for=condition=available --timeout=60s deployment/simple-operator -n sdcore-system || {
    echo "Simple operator is not running. Please run ./scripts/build.sh first."
    exit 1
}

# Apply network function deployments
echo "Deploying UPF..."
kubectl apply -f test/upf_deployment.yaml

echo "Deploying NRF..."
kubectl apply -f test/nrf_deployment.yaml

echo "Deploying AMF..."
kubectl apply -f test/amf_deployment.yaml

echo "Waiting for pods to be created..."
sleep 10

# Check the status of all deployments
echo "Checking deployment status..."
kubectl get deployments -A | grep example

# Check the status of all services
echo "Checking service status..."
kubectl get services -A | grep example

# Check the status of the pods
echo "Checking pod status..."
kubectl get pods -A | grep example

echo "Test completed!" 