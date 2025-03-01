# Deployment Configuration

This directory contains deployment manifests for the SDCore operator:

- `rbac.yaml`: Contains all RBAC resources (ServiceAccount, ClusterRole, ClusterRoleBinding) needed for the operator
- `operator.yaml`: Deployment manifest for the SDCore operator
- `kustomization.yaml`: Kustomize configuration to include all deployment resources

## Deployment Instructions

To deploy the SDCore operator:

1. Build the operator image:
   ```
   make docker-build IMG=sdcore-operator:latest
   ```

2. Push the image to your registry (if needed):
   ```
   make docker-push IMG=your-registry/sdcore-operator:latest
   ```

3. Deploy the operator:
   ```
   kubectl apply -k config/deploy
   ```

4. Verify the deployment:
   ```
   kubectl get pods -n sdcore-system
   ```

## Configuration

The operator deployment uses the following configuration:
- Namespace: `sdcore-system`
- ServiceAccount: `sdcore-operator`
- Resource limits: 500m CPU, 512Mi memory
- Resource requests: 100m CPU, 128Mi memory

You can modify these settings by editing the respective YAML files before deployment. 