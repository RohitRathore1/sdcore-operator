#!/bin/bash
set -e

# Check if we're running in kind
if kubectl get nodes | grep -q kind; then
  echo "Detected kind cluster, loading image into kind..."
  kind load docker-image sdcore-operator:local
fi

# Create the namespace if it doesn't exist
kubectl create namespace sdcore-system --dry-run=client -o yaml | kubectl apply -f -

# Apply the CRD
kubectl apply -f config/crd/nfdeployment.yaml

# Create the service account
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: controller-manager
  namespace: sdcore-system
EOF

# Create the cluster role
cat <<EOF | kubectl apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: sdcore-operator-role
rules:
- apiGroups:
  - ""
  resources:
  - configmaps
  - services
  - pods
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - apps
  resources:
  - deployments
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - workload.nephio.org
  resources:
  - nfdeployments
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
EOF

# Create the cluster role binding
cat <<EOF | kubectl apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: sdcore-operator-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: sdcore-operator-role
subjects:
- kind: ServiceAccount
  name: controller-manager
  namespace: sdcore-system
EOF

# Delete any existing deployment
kubectl delete deployment controller-manager -n sdcore-system --ignore-not-found

# Deploy the operator
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: controller-manager
  namespace: sdcore-system
  labels:
    control-plane: controller-manager
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
  replicas: 1
  template:
    metadata:
      annotations:
        kubectl.kubernetes.io/default-container: manager
      labels:
        control-plane: controller-manager
    spec:
      securityContext:
        runAsNonRoot: true
      containers:
      - command:
        - /manager
        args:
        - --leader-elect=false
        image: sdcore-operator:local
        imagePullPolicy: Never
        name: manager
        securityContext:
          allowPrivilegeEscalation: false
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8081
          initialDelaySeconds: 15
          periodSeconds: 20
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8081
          initialDelaySeconds: 5
          periodSeconds: 10
        resources:
          limits:
            cpu: 500m
            memory: 128Mi
          requests:
            cpu: 10m
            memory: 64Mi
      serviceAccountName: controller-manager
      terminationGracePeriodSeconds: 10
EOF

echo "SDCore Operator deployed successfully!" 