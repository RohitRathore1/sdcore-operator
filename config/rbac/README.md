# RBAC Configuration

This directory contains Role-Based Access Control (RBAC) configurations for the SDCore operator:

- `role.yaml`: Defines the ClusterRole with permissions needed by the operator to manage resources
- `role_binding.yaml`: Binds the ClusterRole to the ServiceAccount
- `service_account.yaml`: Defines the ServiceAccount used by the operator
- `kustomization.yaml`: Kustomize configuration to include all RBAC resources

These RBAC configurations ensure that the SDCore operator has the necessary permissions to:
- Create, read, update, and delete ConfigMaps, Pods, Services, and Deployments
- Manage NFDeployment custom resources
- Access events and other resources needed for proper operation

The operator uses these permissions to reconcile the desired state of SDCore network functions with the actual state in the Kubernetes cluster. 