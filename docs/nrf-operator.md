# NRF Operator Implementation

This document describes the implementation of the Network Repository Function (NRF) operator in the SDCore operator.

## Overview

The NRF (Network Repository Function) is a critical component in the 5G core network architecture. It serves as a central registry for all network functions, allowing them to discover and communicate with each other. The NRF operator manages the deployment and lifecycle of the NRF component in a Kubernetes environment.

## Implementation Details

The NRF operator is implemented as a controller that watches for `NFDeployment` custom resources with the provider `nrf.sdcore.io`. When such a resource is created, updated, or deleted, the controller reconciles the state of the NRF deployment to match the desired state specified in the custom resource.

The implementation includes:

1. A reconciler that handles the creation and management of the NRF deployment
2. Support for configuration parameters specific to NRF
3. Status updates to reflect the current state of the NRF deployment

## Docker Image

The NRF operator uses the following Docker image:

```
gcr.io/rt-hub-0015/free5gc-nrf:0.0.1
```

## Deployment

To deploy the NRF operator, you can use the provided script:

```bash
./scripts/deploy-nrf.sh
```

This script will:

1. Build the SDCore operator image
2. Deploy the operator to the Kubernetes cluster
3. Deploy a test NRF deployment

### Options

The deployment script supports the following options:

- `--image IMAGE`: Set the image name (default: nephio/sdcore-operator:latest)
- `--no-build`: Skip building the image
- `--push`: Push the image to the registry
- `--registry REGISTRY`: Set the registry to push to
- `--undeploy`: Undeploy the operator
- `--no-test`: Skip deploying test NRF deployment
- `--no-skip-code-gen`: Do not skip code generation
- `--full-build`: Do a full build instead of a simplified build

## Test Deployment

The test deployment creates an NRF instance with the following configuration:

```yaml
apiVersion: nf.nephio.org/v1alpha1
kind: NFDeployment
metadata:
  name: example-nrf
  namespace: default
spec:
  provider: nrf.sdcore.io
  parameterValues:
    - name: capacity
      value: small
    - name: dns
      value: 8.8.8.8
```

## Development

When developing the NRF operator, you can use the following workflow:

1. Make changes to the NRF reconciler code in `controllers/nf/nrf/reconciler.go`
2. Build and deploy the operator using the deployment script
3. Create or update an NRF deployment to test your changes
4. Check the logs of the operator to verify that your changes are working as expected

## Troubleshooting

If you encounter issues with the NRF operator, you can check the logs of the operator pod:

```bash
kubectl logs -l app.kubernetes.io/name=sdcore-operator -n sdcore-system
```

You can also check the status of the NRF deployment:

```bash
kubectl get nfdeployments example-nrf -o yaml
``` 