#!/bin/bash

set -e

echo "Building simplified SDCore Operator..."

# Create a temporary Dockerfile
cat > Dockerfile.simple << EOF
FROM golang:1.20-alpine

WORKDIR /app
COPY . .

# Install required tools
RUN apk add --no-cache curl bash

# Entry point
ENTRYPOINT ["sleep", "infinity"]
EOF

# Build the Docker image with a local tag
DOCKER_IMAGE="sdcore-operator:latest"

echo "Building simplified Docker image ${DOCKER_IMAGE}..."
docker build -t ${DOCKER_IMAGE} -f Dockerfile.simple .

# Check if running in Kind
if command -v kind &> /dev/null && kind get clusters | grep -q kind; then
    echo "Detected Kind cluster, loading image directly..."
    kind load docker-image ${DOCKER_IMAGE}
fi

# Create namespace if it doesn't exist
kubectl create namespace sdcore-system --dry-run=client -o yaml | kubectl apply -f -

# Apply CRDs
echo "Applying CRDs..."
kubectl apply -f config/crd/bases/

# Update deployment to use the simplified container
cat > config/deploy/simple-operator.yaml << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: simple-operator
  namespace: sdcore-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: simple-operator
  template:
    metadata:
      labels:
        app: simple-operator
    spec:
      serviceAccountName: default
      containers:
      - name: operator
        image: ${DOCKER_IMAGE}
        imagePullPolicy: IfNotPresent
        command: ["sleep", "infinity"]
        resources:
          limits:
            cpu: 100m
            memory: 100Mi
          requests:
            cpu: 100m
            memory: 60Mi
EOF

# Deploy the simple operator
echo "Deploying simple operator..."
kubectl apply -f config/deploy/simple-operator.yaml

echo "Simple SDCore Operator built and deployed successfully!" 