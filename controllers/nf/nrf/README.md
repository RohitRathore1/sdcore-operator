# NRF Operator

The NRF (Network Repository Function) operator is responsible for managing the deployment and lifecycle of the NRF component in the SD-Core 5G network.

## Overview

The NRF is a critical component in the 5G Core network that:
- Maintains a registry of all Network Functions (NFs) in the network
- Handles service discovery for other NFs
- Manages NF profiles and their capabilities
- Facilitates communication between different NFs

## Implementation

The NRF operator consists of:
- A reconciler that implements the NFReconciler interface
- Resource creation functions for ConfigMap, Deployment, and Service
- Helper functions for parameter extraction and configuration

## Configuration Parameters

The NRF operator supports the following parameters in the NFDeployment custom resource:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `capacity` | Capacity setting for the NRF | `small` |
| `dns` | DNS server IP address | `8.8.8.8` |

## Example Deployment

```yaml
apiVersion: workload.nephio.org/v1alpha1
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

## Deployment

You can deploy the NRF using the deployment script:

```bash
./scripts/deploy.sh --test --nrf
```

This will deploy only the NRF test resource. To deploy all network functions including the NRF, use:

```bash
./scripts/deploy.sh --test
```

## Resources Created

The NRF operator creates the following Kubernetes resources:
- ConfigMap: Contains the NRF configuration
- Deployment: Runs the NRF container
- Service: Exposes the NRF SBI interface

## Container Image

The NRF uses the container image specified in the constants file:
- Image: `omecproject/5gc-nrf:rel-1.6.3` 