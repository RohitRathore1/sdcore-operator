#!/bin/bash

# Script to deploy the SDCore operator and test network functions

set -e

# Default values
IMAGE="nephio/sdcore-operator:latest"
ACTION="deploy"
BUILD=true
PUSH=false
REGISTRY=""
TEST=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
    --image)
      IMAGE="$2"
      shift
      shift
      ;;
    --no-build)
      BUILD=false
      shift
      ;;
    --push)
      PUSH=true
      shift
      ;;
    --registry)
      REGISTRY="$2"
      shift
      shift
      ;;
    --undeploy)
      ACTION="undeploy"
      shift
      ;;
    --test)
      TEST=true
      shift
      ;;
    --help)
      echo "Usage: $0 [options]"
      echo "Options:"
      echo "  --image IMAGE     Set the image name (default: nephio/sdcore-operator:latest)"
      echo "  --no-build        Skip building the image"
      echo "  --push            Push the image to registry"
      echo "  --registry REG    Set the registry to push to"
      echo "  --undeploy        Undeploy the operator instead of deploying"
      echo "  --test            Deploy test network functions after deploying the operator"
      echo "  --help            Show this help message"
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

# Set the full image name if registry is provided
if [ -n "$REGISTRY" ]; then
  IMAGE="${REGISTRY}/${IMAGE}"
fi

# Build the image if requested
if [ "$BUILD" = true ] && [ "$ACTION" = "deploy" ]; then
  echo "Building image: $IMAGE"
  make docker-build IMG=$IMAGE
fi

# Push the image if requested
if [ "$PUSH" = true ] && [ "$ACTION" = "deploy" ]; then
  echo "Pushing image: $IMAGE"
  make docker-push IMG=$IMAGE
fi

# Deploy or undeploy the operator
if [ "$ACTION" = "deploy" ]; then
  echo "Deploying SDCore operator with image: $IMAGE"
  make deploy-simple IMG=$IMAGE
  
  # Wait for the operator to be ready
  echo "Waiting for operator to be ready..."
  kubectl wait --for=condition=available --timeout=60s deployment/sdcore-operator -n sdcore-system
  
  # Deploy test network functions if requested
  if [ "$TEST" = true ]; then
    echo "Deploying test network functions..."
    kubectl apply -f test/upf_deployment.yaml
    kubectl apply -f test/nrf_deployment.yaml
    kubectl apply -f test/ausf_deployment.yaml
    kubectl apply -f test/udm_deployment.yaml
    kubectl apply -f test/udr_deployment.yaml
    kubectl apply -f test/pcf_deployment.yaml
    
    echo "Waiting for network functions to be deployed..."
    sleep 10
    kubectl get pods -A
  fi
else
  echo "Undeploying SDCore operator"
  make undeploy-simple
fi

echo "Done!" 