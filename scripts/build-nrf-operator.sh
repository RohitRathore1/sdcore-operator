#!/bin/bash

# Script to build and deploy a simplified NRF-only operator

set -e

# Default values
IMAGE="nephio/nrf-operator:latest"
PUSH=false
REGISTRY=""
DEPLOY=true

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --image)
      IMAGE="$2"
      shift 2
      ;;
    --push)
      PUSH=true
      shift
      ;;
    --registry)
      REGISTRY="$2"
      shift 2
      ;;
    --no-deploy)
      DEPLOY=false
      shift
      ;;
    --help)
      echo "Usage: $0 [options]"
      echo "Options:"
      echo "  --image IMAGE     Specify the image name (default: nephio/nrf-operator:latest)"
      echo "  --push            Push the image to the registry"
      echo "  --registry REG    Specify the registry"
      echo "  --no-deploy       Skip deployment to Kubernetes"
      echo "  --help            Show this help message"
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

echo "Building NRF operator image: $IMAGE"

# Create a temporary directory
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

# Create a simple main.go file
cat > "$TMP_DIR/main.go" << 'EOL'
package main

import (
	"context"
	"flag"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

// NRFController watches for ConfigMaps with a specific label
type NRFController struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile handles ConfigMap events
func (r *NRFController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling ConfigMap", "name", req.NamespacedName)

	// Fetch the ConfigMap
	configMap := &corev1.ConfigMap{}
	if err := r.Get(ctx, req.NamespacedName, configMap); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if this is an NRF ConfigMap
	if configMap.Labels["app"] == "nrf" {
		logger.Info("Found NRF ConfigMap", 
			"name", configMap.Name,
			"namespace", configMap.Namespace,
			"data", len(configMap.Data))
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *NRFController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		Complete(r)
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "nrf-operator.nephio.org",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Create and register the NRF controller
	if err = (&NRFController{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NRF")
		os.Exit(1)
	}

	setupLog.Info("Starting NRF operator")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
EOL

# Create a go.mod file
cat > "$TMP_DIR/go.mod" << 'EOL'
module github.com/RohitRathore1/nrf-operator

go 1.20

require (
	k8s.io/api v0.25.0
	k8s.io/apimachinery v0.25.0
	k8s.io/client-go v0.25.0
	sigs.k8s.io/controller-runtime v0.13.0
)
EOL

# Create Dockerfile
cat > "$TMP_DIR/Dockerfile" << 'EOL'
FROM golang:1.20-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod tidy
RUN go mod download
# Build the binary
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o manager .

FROM alpine:3.18
WORKDIR /
COPY --from=builder /app/manager .
ENTRYPOINT ["/manager"]
EOL

# Build the Docker image
echo "Building the Docker image..."
(cd "$TMP_DIR" && docker build -t "$IMAGE" .)

# Push the image if requested
if [ "$PUSH" = true ]; then
  if [ -n "$REGISTRY" ]; then
    REMOTE_IMAGE="$REGISTRY/$IMAGE"
    docker tag "$IMAGE" "$REMOTE_IMAGE"
    docker push "$REMOTE_IMAGE"
  else
    docker push "$IMAGE"
  fi
fi

# Load the image into Kind if running in a Kind cluster
if kubectl get nodes | grep -q "kind-control-plane"; then
  echo "Detected Kind cluster, loading image..."
  kind load docker-image "$IMAGE"
fi

# Deploy the operator to Kubernetes
if [ "$DEPLOY" = true ]; then
  echo "Deploying NRF operator..."
  
  # Create namespace
  kubectl create namespace nrf-system 2>/dev/null || true
  
  # Create service account
  kubectl apply -f - << EOL
apiVersion: v1
kind: ServiceAccount
metadata:
  name: nrf-operator
  namespace: nrf-system
EOL

  # Create role
  kubectl apply -f - << EOL
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: nrf-operator-role
rules:
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "list", "watch"]
EOL

  # Create role binding
  kubectl apply -f - << EOL
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: nrf-operator-rolebinding
subjects:
- kind: ServiceAccount
  name: nrf-operator
  namespace: nrf-system
roleRef:
  kind: ClusterRole
  name: nrf-operator-role
  apiGroup: rbac.authorization.k8s.io
EOL

  # Deploy the operator
  kubectl apply -f - << EOL
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nrf-operator
  namespace: nrf-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nrf-operator
  template:
    metadata:
      labels:
        app: nrf-operator
    spec:
      serviceAccountName: nrf-operator
      containers:
      - name: operator
        image: $IMAGE
        imagePullPolicy: IfNotPresent
        resources:
          limits:
            cpu: 200m
            memory: 256Mi
          requests:
            cpu: 100m
            memory: 128Mi
EOL

  # Wait for the operator to be ready
  echo "Waiting for the NRF operator to be ready..."
  kubectl -n nrf-system wait --for=condition=available --timeout=60s deployment/nrf-operator || true

  # Create a test ConfigMap with the NRF label
  kubectl apply -f - << EOL
apiVersion: v1
kind: ConfigMap
metadata:
  name: nrf-config
  namespace: default
  labels:
    app: nrf
data:
  config.yaml: |
    capacity: small
    dns: "8.8.8.8"
EOL

  echo "NRF operator and test ConfigMap deployed successfully!"
fi

echo "Done!" 