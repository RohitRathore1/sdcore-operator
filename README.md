# SDCore Operator

Kubernetes operator for managing 5G Core Network Functions using Nephio-based NFDeployment CRDs.

## Overview

The SDCore Operator manages the deployment of SDCore network functions using NFDeployment custom resources. It supports the following network functions:

- UPF (User Plane Function)
- AMF (Access and Mobility Function)
- SMF (Session Management Function)
- NRF (Network Repository Function)
- AUSF (Authentication Server Function)
- NSSF (Network Slice Selection Function)
- PCF (Policy Control Function)
- UDM (Unified Data Management)
- UDR (Unified Data Repository)
- NEF (Network Exposure Function)

## Prerequisites

- Kubernetes 1.23+
- kubectl
- Go 1.20+
- Docker

## Installation

### Using Scripts

The operator can be built and deployed using the provided scripts:

```bash
# Build and deploy the operator
./scripts/build.sh

# Test the operator by deploying sample network functions
./scripts/test.sh
```

### Manual Installation

1. Apply the Custom Resource Definition (CRD):

```bash
kubectl apply -f config/crd/bases/workload.nephio.org_nfdeployments.yaml
```

2. Build and push the Docker image:

```bash
docker build -t docker.io/nephio/sdcore-operator:latest .
```

3. Deploy the operator:

```bash
kubectl create namespace sdcore-system
kubectl apply -f config/deploy/deployment.yaml
```

## Usage

To deploy network functions, create NFDeployment custom resources. Examples are provided in the `test` directory.

### Deploying UPF

```bash
kubectl apply -f test/upf_deployment.yaml
```

### Deploying AMF

```bash
kubectl apply -f test/amf_deployment.yaml
```

### Deploying NRF

```bash
kubectl apply -f test/nrf_deployment.yaml
```

## Known Issues

1. Code generation with controller-gen may fail with nil pointer dereference. Use the provided scripts to build and deploy instead.
2. The SetupWithManager method may be missing in some controllers. This is addressed in the script-based deployment.

## Troubleshooting

### Checking Operator Status

```bash
kubectl get pods -n sdcore-system
```

### Checking Network Function Deployments

```bash
kubectl get nfdeployments
kubectl get deployments
kubectl get pods | grep example
```

## Development

### Build

```bash
go build -o bin/sdcore-operator main.go
```

### Test

```bash
go test ./... -v
```

License
-------

Copyright 2024 The Nephio Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
