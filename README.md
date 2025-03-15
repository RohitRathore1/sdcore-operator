# SDCore Operator

A Kubernetes operator for bess-upf based on the Nephio NFDeployment API.

## Description

SDCore Operator manages deployments of SDCore's network functions by reconciling Nephio's
`NFDeployment` custom resources. The operator creates and maintains the necessary Kubernetes resources
(Deployments, ConfigMaps, Services) according to the specifications in the NFDeployment objects.

Currently supports:

- UPF (User Plane Function) with `provider: upf.sdcore.io`
  - Based on BESS-UPF implementation from the sdcore-helm-charts
  - Multi-container architecture with BESS dataplane and PFCP control plane

## Getting Started

### Prerequisites

- Kubernetes cluster (v1.23+)
- kubectl CLI tool
- Multus CNI plugin for multi-interface support (for production deployments)
- Nephio NFDeployment CRD installed

### Quick Start Guide

#### 1. Install Nephio CRDs

First, install the required Nephio Custom Resource Definitions:

```sh
kubectl apply -f https://raw.githubusercontent.com/nephio-project/api/main/config/crd/bases/workload.nephio.org_nfdeployments.yaml
kubectl apply -f https://raw.githubusercontent.com/nephio-project/api/main/config/crd/bases/workload.nephio.org_nfconfigs.yaml
kubectl apply -f https://raw.githubusercontent.com/nephio-project/api/main/config/crd/bases/ref.nephio.org_configs.yaml
```

#### 2. Clone and Navigate to the Repository

```sh
git clone https://github.com/RohitRathore1/sdcore-operator.git
cd sdcore-operator
```

#### 3. Run the Operator Locally (Development Mode)

For testing and development, you can run the operator locally:

```sh
make run
```

This will run the operator on your local machine, connecting to the cluster configured in your current kubeconfig.

#### 4. Deploy a Test UPF

Apply the example UPF NFDeployment:

```sh
kubectl apply -f test/upf.yaml
```

#### 5. Verify the Deployment

Check that the resources were created:

```sh
kubectl get nfdeployment test-upf
kubectl get configmap,deployment,service | grep upf
kubectl get pods -l app=test-upf-upf
```

### Production Deployment

For production use, build and deploy the operator as a container:

#### 1. Build and Push the Operator Image

```sh
make docker-build docker-push IMG=<your-registry>/sdcore-operator:v0.1.0
```

#### 2. Deploy the Operator to the Cluster

```sh
make deploy IMG=<your-registry>/sdcore-operator:v0.1.0
```

## NFDeployment Examples

### UPF Deployment

Here's an example NFDeployment for UPF:

```yaml
apiVersion: workload.nephio.org/v1alpha1
kind: NFDeployment
metadata:
  name: example-upf
spec:
  provider: upf.sdcore.io
  interfaces:
  - name: n3
    ipv4:
      address: 192.168.252.3/24
      gateway: 192.168.252.1
  - name: n4
    ipv4:
      address: 192.168.250.3/24
      gateway: 192.168.250.1
  - name: n6
    ipv4:
      address: 192.168.249.3/24
      gateway: 192.168.249.1
  networkInstances:
  - name: data-network
    interfaces:
    - n6
    dataNetworks:
    - name: internet
      pool:
      - prefix: 172.250.0.0/16
```

## Architecture

### Components

The SDCore operator consists of:

1. **Main Controller** - Routes NFDeployment resources to specific network function reconcilers based on the `provider` field
2. **UPF Reconciler** - Handles UPF deployments using a BESS-based implementation:
   - Creates ConfigMap with UPF configuration
   - Creates Deployment with multiple containers (BESS dataplane, PFCP agent, etc.)
   - Creates Service to expose PFCP and management interfaces

### UPF Implementation

The UPF is implemented using a multi-container setup based on the OMEC BESS-UPF architecture:

- **BESS Dataplane (`bessd`)** - High-performance software dataplane using Berkeley Extensible Software Switch
- **PFCP Agent (`pfcp-agent`)** - Control plane component that handles PFCP signaling with SMF
- **Route Controller (`routectl`)** - Manages network routes for the UPF
- **Web Interface (`web`)** - Provides a web dashboard for BESS monitoring

## Troubleshooting

### Common Issues

1. **Pods stuck in `Init:CrashLoopBackOff`** - Check network configuration and ensure Multus is properly configured
2. **PFCP connection failures** - Verify that the SMF can reach the UPF's N4 interface
3. **Image pull failures** - Ensure the container registry is accessible from your cluster

### Debugging

To debug the operator:

```sh
# Run with increased verbosity
make run ARGS="--zap-log-level=debug"

# Check operator logs
kubectl logs -n sdcore-operator-system deployment/sdcore-operator-controller-manager
```

## Development

### Project Structure

```
├── controllers/          # Controller implementations
│   ├── nf/               # Network function reconcilers
│   │   └── upf/          # UPF reconciler
├── test/                 # Example custom resources for testing
└── main.go               # Main entry point
```
