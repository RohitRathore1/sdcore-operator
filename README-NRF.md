# NRF Operator

This is a simplified operator that watches for ConfigMap resources with the label `app=nrf` and logs when they are created, updated, or deleted. This operator is a proof-of-concept for a more comprehensive NRF (Network Repository Function) operator that would manage NRF deployments in a 5G core network.

## Overview

The NRF operator is built as a standalone Kubernetes operator that:

1. Watches for ConfigMap resources with the label `app=nrf`
2. Logs when these ConfigMaps are reconciled
3. Provides a foundation for implementing more complex NRF management logic

## Building and Deploying

### Prerequisites

- Docker
- Kubernetes cluster (e.g., Kind, Minikube)
- kubectl

### Building the Operator

The operator can be built using the provided build script:

```bash
./scripts/build-nrf-operator.sh
```

This script:
- Creates a temporary build directory
- Generates a simplified Go application that watches ConfigMaps
- Builds a Docker image for the operator
- Optionally pushes the image to a registry
- Optionally deploys the operator to the Kubernetes cluster

### Command-line Options

The build script supports the following options:

- `--image`: Specify the image name (default: nephio/nrf-operator:latest)
- `--push`: Push the image to the registry
- `--registry`: Specify the registry to push to
- `--no-deploy`: Skip deploying the operator
- `--help`: Show help message

### Deployment

When deployed, the operator:
1. Creates a namespace called `nrf-system`
2. Sets up necessary RBAC permissions
3. Deploys the operator pod
4. Watches for ConfigMaps named "nrf-config" across all namespaces

## Testing

To test the operator, create a ConfigMap with the label `app=nrf`:

```yaml
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
```

Apply this ConfigMap:

```bash
kubectl apply -f nrf-config.yaml
```

You can also create ConfigMaps with different names but the same label:

```bash
kubectl create configmap nrf-config-test --from-literal=config.yaml="capacity: medium\ndns: \"1.1.1.1\"" -n default --dry-run=client -o yaml | kubectl label --local -f - app=nrf -o yaml | kubectl apply -f -
```

Check the operator logs to see that it detected the ConfigMap:

```bash
kubectl logs -n nrf-system -l app=nrf-operator
```

You should see log entries indicating that the operator reconciled the ConfigMap, such as:

```
INFO    Found NRF ConfigMap     {"controller": "configmap", "name": "nrf-config-test", "namespace": "default", "data": 1}
```

## Future Enhancements

This simplified operator could be extended to:

1. Create actual NRF deployments based on the ConfigMap content
2. Manage NRF service configurations
3. Handle updates to NRF configurations
4. Integrate with other 5G core network functions
5. Implement the NFDeployment CRD for more comprehensive management

## Troubleshooting

If the operator is not functioning as expected:

1. Check the operator logs:
   ```bash
   kubectl logs -n nrf-system -l app=nrf-operator
   ```

2. Verify the ConfigMap exists:
   ```bash
   kubectl get configmap nrf-config -n default
   ```

3. Check that the operator has the necessary permissions:
   ```bash
   kubectl get clusterrole nrf-operator-role -o yaml
   kubectl get clusterrolebinding nrf-operator-rolebinding -o yaml
   ```

## Summary of Implementation

This implementation demonstrates a simplified approach to building a Kubernetes operator for the NRF component of a 5G core network. Key accomplishments include:

1. **Simplified Operator Architecture**: Created a standalone operator that focuses solely on the NRF component, rather than a monolithic operator that manages multiple network functions.

2. **ConfigMap-based Configuration**: Implemented a mechanism to detect and process ConfigMaps with the `app=nrf` label, which can be used as a configuration source for NRF deployments.

3. **Build and Deployment Automation**: Developed a comprehensive build script (`build-nrf-operator.sh`) that handles the entire process from code generation to Kubernetes deployment.

4. **Kubernetes Integration**: Set up proper RBAC permissions and deployment configurations to ensure the operator can function within a Kubernetes cluster.

5. **Testing Framework**: Established a testing approach that verifies the operator's ability to detect and process NRF configurations.

This implementation serves as a foundation for more advanced NRF management capabilities, such as deploying actual NRF instances, managing their lifecycle, and integrating with other 5G core network functions. 