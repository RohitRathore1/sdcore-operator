# SDCore Operator

A Kubernetes operator for SDCore network functions based on the Nephio NFDeployment API.

## Description

SDCore Operator manages deployments of SDCore's network functions by reconciling Nephio's
`NFDeployment` custom resources. The operator creates and maintains the necessary Kubernetes resources
(Deployments, ConfigMaps, Services) according to the specifications in the NFDeployment objects.

Currently supports:

- UPF (User Plane Function) with `provider: upf.sdcore.io`
  - Based on BESS-UPF implementation from the sdcore-helm-charts
  - Multi-container architecture with BESS dataplane and PFCP control plane
- SMF (Session Management Function) with `provider: sdcore`
  - Based on Free5GC SMF implementation
  - Handles session management and communicates with UPF via PFCP
- AMF (Access and Mobility Management Function) with `provider: sdcore`
  - Based on Free5GC AMF implementation
  - Handles connection and mobility management for UEs (User Equipment)

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

#### 4. Deploy a Test Network Function

Apply the example NFDeployment for the desired network function:

```sh
# To deploy a UPF
kubectl apply -f test/upf.yaml

# To deploy an SMF
kubectl apply -f test/smf.yaml

# To deploy an AMF
kubectl apply -f test/amf.yaml
```

#### 5. Verify the Deployment

Check that the resources were created:

```sh
# For UPF
kubectl get nfdeployment test-upf
kubectl get configmap,deployment,service -l app=test-upf-upf

# For SMF
kubectl get nfdeployment test-smf
kubectl get configmap,deployment,service -l app=test-smf-smf

# For AMF
kubectl get nfdeployment test-amf
kubectl get configmap,deployment,service -l app=test-amf-amf
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

### SMF Deployment

Here's an example NFDeployment for SMF:

```yaml
apiVersion: workload.nephio.org/v1alpha1
kind: NFDeployment
metadata:
  name: test-smf
spec:
  provider: sdcore
  interfaces:
  - name: n4
    ipv4:
      address: 192.168.250.4/24
      gateway: 192.168.250.1
  networkInstances:
  - name: smf-network
    interfaces:
    - n4
```

### AMF Deployment

Here's an example NFDeployment for AMF:

```yaml
apiVersion: workload.nephio.org/v1alpha1
kind: NFDeployment
metadata:
  name: test-amf
spec:
  provider: sdcore
  interfaces:
  - name: n2
    ipv4:
      address: 192.168.251.5/24
      gateway: 192.168.251.1
  networkInstances:
  - name: amf-network
    interfaces:
    - n2
```

## Architecture

### Components

The SDCore operator consists of:

1. **Main Controller** - Routes NFDeployment resources to specific network function reconcilers based on the `provider` field
2. **UPF Reconciler** - Handles UPF deployments using a BESS-based implementation:
   - Creates ConfigMap with UPF configuration
   - Creates Deployment with multiple containers (BESS dataplane, PFCP agent, etc.)
   - Creates Service to expose PFCP and management interfaces
3. **SMF Reconciler** - Handles SMF deployments:
   - Creates ConfigMap with SMF configuration files (`smfcfg.yaml`, `uerouting.yaml`, and startup script)
   - Creates Deployment with SMF container
   - Creates Service to expose SBI (Service-Based Interface) and PFCP endpoints
4. **AMF Reconciler** - Handles AMF deployments:
   - Creates ConfigMap with AMF configuration files (`amfcfg.yaml` and startup script)
   - Creates Deployment with AMF container
   - Creates Service to expose NGAP (N2) and SBI endpoints
   - Creates Headless Service for internal discovery

### UPF Implementation

The UPF is implemented using a multi-container setup based on the OMEC BESS-UPF architecture:

- **BESS Dataplane (`bessd`)** - High-performance software dataplane using Berkeley Extensible Software Switch
- **PFCP Agent (`pfcp-agent`)** - Control plane component that handles PFCP signaling with SMF
- **Route Controller (`routectl`)** - Manages network routes for the UPF
- **Web Interface (`web`)** - Provides a web dashboard for BESS monitoring

### SMF Implementation

The SMF is implemented as a single container deployment:

- Based on Free5GC/SDCore SMF implementation
- Handles session establishment, modification, and termination
- Communicates with UPF via PFCP protocol on the N4 interface
- Exposes Service-Based Interface (SBI) for communication with other network functions
- Configurable via `smfcfg.yaml` for core settings and `uerouting.yaml` for UE routing policies

### AMF Implementation

The AMF is implemented as a single container deployment:

- Based on Free5GC/SDCore AMF implementation
- Handles UE registration, connection management, and mobility
- Communicates with gNBs via NGAP protocol on the N2 interface
- Exposes Service-Based Interface (SBI) for communication with other network functions
- Provides SCTP load balancing capabilities for RAN connections
- Configurable via `amfcfg.yaml` with PLMN, TAI, and security settings

## Network Repository Function (NRF) Controller

The NRF controller has been implemented to manage the Network Repository Function component of the 5G core network. The NRF is a critical component that provides service discovery functionality, allowing other components to register themselves and discover other network functions.

### Implementation Details

The NRF controller:

1. Creates and manages a ConfigMap with NRF configuration
2. Deploys the NRF container using the image `omecproject/5gc-nrf:rel-1.6.3`
3. Creates a Service for other components to access the NRF
4. Updates the status of the NFDeployment based on the readiness of the NRF deployment

### Testing

To test the NRF controller, apply the test NFDeployment:

```bash
kubectl apply -f test/nrf.yaml
```

This will create an NFDeployment that the operator will reconcile by creating the necessary ConfigMap, Deployment, and Service for the NRF component.

### Access

Other components in the 5G core network can access the NRF via:

```
http://test-nrf-nrf-service:8080
```

The NRF exposes the following service interfaces:
- nnrf-nfm (NF management)
- nnrf-disc (NF discovery)

### Role in 5G Core Architecture

The Network Repository Function (NRF) plays a central role in the 5G Core network:

1. **Service Registration**: All Network Functions (NFs) like AMF, SMF, and UPF register their capabilities and services with the NRF.
2. **Service Discovery**: When an NF needs to communicate with another, it queries the NRF to discover the available instances and their capabilities.
3. **Load Balancing**: The NRF can assist in load balancing by directing NFs to appropriate service instances.
4. **NFProfile Management**: Maintains profiles of all registered NFs, including their services, addresses, and capabilities.

### Integration with Other Components

In the sdcore-operator implementation:

1. **AMF Integration**: The AMF configuration points to the NRF service for registration and discovering other required services.
2. **SMF Integration**: The SMF uses the NRF to register itself and discover the UPF instances it needs to manage.
3. **Centralized Configuration**: The NRF provides a single point of configuration for service endpoints, eliminating the need to hardcode service addresses in each component.

When deploying a complete 5G core with the sdcore-operator, you should deploy the NRF first, followed by other components to ensure proper service registration and discovery.

## Troubleshooting

### Common Issues

1. **Pods stuck in `Init:CrashLoopBackOff`** - Check network configuration and ensure Multus is properly configured
2. **PFCP connection failures** - Verify that the SMF can reach the UPF's N4 interface
3. **Image pull failures** - Ensure the container registry is accessible from your cluster
4. **SMF-UPF connectivity issues** - Check that the N4 interface addresses are configured correctly in both SMF and UPF
5. **NGAP connection failures** - Verify that AMF is accessible from gNBs on the N2 interface
6. **AMF-NRF communication failures** - Check that the NRF URI is correctly configured in the AMF configuration

### Debugging

To debug the operator:

```sh
# Run with increased verbosity
make run ARGS="--zap-log-level=debug"

# Check operator logs
kubectl logs -n sdcore-operator-system deployment/sdcore-operator-controller-manager

# Check SMF logs
kubectl logs -l app=test-smf-smf

# Check AMF logs
kubectl logs -l app=test-amf-amf
```

## Development

### Project Structure

```
├── controllers/          # Controller implementations
│   ├── nf/               # Network function reconcilers
│   │   ├── upf/          # UPF reconciler
│   │   ├── smf/          # SMF reconciler
│   │   └── amf/          # AMF reconciler
├── test/                 # Example custom resources for testing
└── main.go               # Main entry point
```
