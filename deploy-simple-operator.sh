#!/bin/bash
set -e

echo "Building and deploying simple SDCore Operator..."

# Check if we're running in a kind cluster
if kubectl cluster-info | grep -q kind; then
  echo "Detected kind cluster"
  KIND_CLUSTER=true
else
  KIND_CLUSTER=false
fi

# Build the operator image
echo "Building simple operator image..."
docker build -t sdcore-simple-operator:local -f Dockerfile.simple .

# Load the image into kind if needed
if [ "$KIND_CLUSTER" = true ]; then
  echo "Loading image into kind cluster..."
  kind load docker-image sdcore-simple-operator:local
fi

# Create namespace if it doesn't exist
kubectl create namespace sdcore-system --dry-run=client -o yaml | kubectl apply -f -

# Apply the CRD
echo "Configuring NFDeployment CRD..."
kubectl apply -f config/crd/nfdeployment.yaml

# Create a service account for the operator
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: simple-operator
  namespace: sdcore-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: simple-operator-role
rules:
- apiGroups: [""]
  resources: ["pods", "services", "configmaps"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["apps"]
  resources: ["deployments"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["workload.nephio.org"]
  resources: ["nfdeployments"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: simple-operator-rolebinding
subjects:
- kind: ServiceAccount
  name: simple-operator
  namespace: sdcore-system
roleRef:
  kind: ClusterRole
  name: simple-operator-role
  apiGroup: rbac.authorization.k8s.io
EOF

# Deploy the operator
echo "Deploying simple operator..."
cat <<EOF | kubectl apply -f -
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
      serviceAccountName: simple-operator
      containers:
      - name: operator
        image: sdcore-simple-operator:local
        imagePullPolicy: IfNotPresent
        resources:
          limits:
            cpu: 500m
            memory: 512Mi
          requests:
            cpu: 100m
            memory: 128Mi
EOF

echo "Simple SDCore Operator deployed successfully!"
echo "You can now create NFDeployment resources to be processed by the operator." 