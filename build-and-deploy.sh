#!/bin/bash
set -e

echo "Building SDCore Operator container image..."
docker build -t sdcore-operator:latest -f Dockerfile.simple .

echo "Loading image into Kind cluster..."
kind load docker-image sdcore-operator:latest

echo "Applying CRD..."
kubectl apply -f config/crd/nfdeployment.yaml

echo "Checking for existing SDCore operator namespace..."
if ! kubectl get namespace sdcore-system &> /dev/null; then
  echo "Creating SDCore operator namespace..."
  kubectl create namespace sdcore-system
fi

echo "Deploying operator..."
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sdcore-operator
  namespace: sdcore-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: sdcore-operator
  template:
    metadata:
      labels:
        app: sdcore-operator
    spec:
      serviceAccountName: default
      containers:
      - name: operator
        image: sdcore-operator:latest
        imagePullPolicy: IfNotPresent
        resources:
          limits:
            cpu: 500m
            memory: 512Mi
          requests:
            cpu: 100m
            memory: 128Mi
EOF

echo "Waiting for operator to be ready..."
kubectl -n sdcore-system rollout status deployment/sdcore-operator

echo "SDCore Operator successfully deployed!"
echo "You can now create network function deployments using the sample YAML files in the test directory:"
echo "  - kubectl apply -f test/upf_deployment.yaml"
echo "  - kubectl apply -f test/amf_deployment.yaml"
echo "  - kubectl apply -f test/nrf_deployment.yaml" 