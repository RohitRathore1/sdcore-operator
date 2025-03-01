# SDCore Operator

A Kubernetes operator for managing SD-Core 5G network functions. This operator enables declarative deployment and management of 5G network functions including UPF, AMF, SMF, NRF, and more.

## Overview

The SDCore Operator watches for NFDeployment custom resources and creates the necessary Kubernetes resources (Deployments, ConfigMaps, Services) to run SD-Core network functions. It's designed to be lightweight and efficient, using direct Kubernetes API calls rather than complex controller-runtime abstractions.

## Supported Network Functions

The operator supports the following SD-Core network functions:

- UPF (User Plane Function)
- AMF (Access and Mobility Management Function)
- SMF (Session Management Function)
- NRF (Network Repository Function)
- PCF (Policy Control Function)
- UDM (Unified Data Management)
- UDR (Unified Data Repository)
- AUSF (Authentication Server Function)
- NSSF (Network Slice Selection Function)

## Prerequisites

- Kubernetes cluster (v1.20+)
- `kubectl` command-line tool
- Docker (for building and loading images)
- Kind cluster (for local testing)

## Installation

### Using the provided script

```bash
chmod +x build-and-deploy.sh
./build-and-deploy.sh
```

This script will:
1. Build the operator Docker image
2. Load it into your Kind cluster
3. Apply the CRD
4. Create the operator namespace
5. Deploy the operator

### Manual installation

1. Build the operator image:
   ```bash
   docker build -t sdcore-operator:latest -f Dockerfile.simple .
   ```

2. Load the image into your Kind cluster:
   ```bash
   kind load docker-image sdcore-operator:latest
   ```

3. Apply the CRD:
   ```bash
   kubectl apply -f config/crd/nfdeployment.yaml
   ```

4. Create the operator namespace:
   ```bash
   kubectl create namespace sdcore-system
   ```

5. Deploy the operator:
   ```bash
   kubectl apply -f config/deployment/operator.yaml
   ```

## Usage

### Create Network Function Deployments

After installing the operator, you can create network function deployments using the provided sample YAML files:

```bash
# Deploy the NRF (should be deployed first as other NFs depend on it)
kubectl apply -f test/nrf_deployment.yaml

# Deploy the AMF
kubectl apply -f test/amf_deployment.yaml

# Deploy the UPF
kubectl apply -f test/upf_deployment.yaml
```

### Customizing Network Function Deployments

You can customize the network function deployments by editing the YAML files or creating new ones. Each NFDeployment resource supports the following fields:

- `provider`: Specifies the network function provider (e.g., `upf.sdcore.io`, `amf.sdcore.io`)
- `interfaces`: Network interfaces configuration
- `parameterValues`: Configuration parameters for the network function
- `replicas`: Number of replicas for the deployment
- `monitoringEnabled`: Whether monitoring is enabled
- `upstreamNFs`: List of upstream network functions that this NF depends on

Example:

```yaml
apiVersion: workload.nephio.org/v1alpha1
kind: NFDeployment
metadata:
  name: example-upf
  namespace: default
spec:
  provider: upf.sdcore.io
  interfaces:
    - name: n3
      ipv4:
        address: 172.16.10.2/24
        gateway: 172.16.10.1
  parameterValues:
    - name: capacity
      value: small
    - name: dns
      value: 8.8.8.8
  replicas: 1
  monitoringEnabled: true
```

## Development

### Architecture

The operator uses a simple reconciliation loop to watch for NFDeployment resources and create the necessary Kubernetes resources. It directly uses the Kubernetes client-go libraries to interact with the API server, rather than using the more complex controller-runtime libraries.

The main components are:
- A dynamic client that watches for NFDeployment resources
- Switch statement to handle different network function types
- Resource creation functions for each network function

### Adding a new Network Function

To add support for a new network function:

1. Add a new constant for the image in `simple-operator.go`
2. Add a new case in the switch statement in the main loop
3. Implement the processing function for the new network function
4. Create helper functions for ConfigMap, Deployment, and Service creation

### Building and Testing

```bash
# Build the operator
go build -o sdcore-operator simple-operator.go

# Run locally (for development)
./sdcore-operator

# Build the Docker image
docker build -t sdcore-operator:latest -f Dockerfile.simple .
```

## License

MIT
