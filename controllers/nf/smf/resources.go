package smf

import (
	"context"
	"fmt"

	"github.com/RohitRathore1/sdcore-operator/controllers"
	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// SMF container names
	smfContainerName = "smf"

	// SMF image names
	smfImageName = "registry.opennetworking.org/docker.io/omecproject/5gc-smf:master-latest"

	// SMF service names
	smfServiceName = "smf-service"

	// SMF port names
	smfPfcpPortName = "pfcp"
	smfSbiPortName  = "sbi"

	// SMF port numbers
	smfPfcpPort = 8805
	smfSbiPort  = 8080
)

// reconcileConfigMap reconciles the ConfigMap for the SMF
func reconcileConfigMap(ctx context.Context, c client.Client, nfDeployment *nephiov1alpha1.NFDeployment) (bool, error) {
	log := log.FromContext(ctx).WithValues("SMFConfigMap", nfDeployment.Name)

	configMapName := controllers.GetNamespacedName(nfDeployment, "smf-config")
	configMap := &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: nfDeployment.Namespace,
			Labels: map[string]string{
				"app": controllers.GetNamespacedName(nfDeployment, "smf"),
			},
		},
		Data: map[string]string{
			"smf-run.sh":     generateSMFRunScript(),
			"smfcfg.yaml":    generateSMFConfig(nfDeployment),
			"uerouting.yaml": generateUERoutingConfig(),
		},
	}

	// Set the owner reference
	if err := controllerutil.SetControllerReference(nfDeployment, configMap, c.Scheme()); err != nil {
		log.Error(err, "Failed to set owner reference on ConfigMap")
		return false, err
	}

	// Check if the ConfigMap already exists
	existingConfigMap := &apiv1.ConfigMap{}
	err := c.Get(ctx, client.ObjectKey{Namespace: nfDeployment.Namespace, Name: configMapName}, existingConfigMap)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Create the ConfigMap
			log.Info("Creating ConfigMap", "ConfigMap.Name", configMapName)
			if err := c.Create(ctx, configMap); err != nil {
				log.Error(err, "Failed to create ConfigMap")
				return false, err
			}
			return true, nil
		}
		log.Error(err, "Failed to get ConfigMap")
		return false, err
	}

	// Update the ConfigMap if needed
	if !mapsEqual(existingConfigMap.Data, configMap.Data) {
		log.Info("Updating ConfigMap", "ConfigMap.Name", configMapName)
		existingConfigMap.Data = configMap.Data
		if err := c.Update(ctx, existingConfigMap); err != nil {
			log.Error(err, "Failed to update ConfigMap")
			return false, err
		}
		return true, nil
	}

	return false, nil
}

// reconcileDeployment reconciles the Deployment for the SMF
func reconcileDeployment(ctx context.Context, c client.Client, scheme *runtime.Scheme, nfDeployment *nephiov1alpha1.NFDeployment) (bool, error) {
	log := log.FromContext(ctx).WithValues("SMFDeployment", nfDeployment.Name)

	deploymentName := controllers.GetNamespacedName(nfDeployment, "smf")
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: nfDeployment.Namespace,
			Labels: map[string]string{
				"app": deploymentName,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": deploymentName,
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": deploymentName,
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  smfContainerName,
							Image: smfImageName,
							Ports: []apiv1.ContainerPort{
								{
									Name:          smfPfcpPortName,
									ContainerPort: smfPfcpPort,
									Protocol:      apiv1.ProtocolUDP,
								},
								{
									Name:          smfSbiPortName,
									ContainerPort: smfSbiPort,
									Protocol:      apiv1.ProtocolTCP,
								},
							},
							Command: []string{
								"/bin/bash",
								"/config/smf-run.sh",
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      "smf-config",
									MountPath: "/config",
								},
							},
							Env: []apiv1.EnvVar{
								{
									Name:  "PFCP_PORT",
									Value: fmt.Sprintf("%d", smfPfcpPort),
								},
								{
									Name:  "LOG_LEVEL",
									Value: "info",
								},
							},
							Resources: apiv1.ResourceRequirements{
								Requests: createResourceList("500m", "512Mi"),
								Limits:   createResourceList("500m", "512Mi"),
							},
						},
					},
					Volumes: []apiv1.Volume{
						{
							Name: "smf-config",
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
										Name: controllers.GetNamespacedName(nfDeployment, "smf-config"),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Set the owner reference
	if err := controllerutil.SetControllerReference(nfDeployment, deployment, scheme); err != nil {
		log.Error(err, "Failed to set owner reference on Deployment")
		return false, err
	}

	// Check if the Deployment already exists
	existingDeployment := &appsv1.Deployment{}
	err := c.Get(ctx, client.ObjectKey{Namespace: nfDeployment.Namespace, Name: deploymentName}, existingDeployment)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Create the Deployment
			log.Info("Creating Deployment", "Deployment.Name", deploymentName)
			if err := c.Create(ctx, deployment); err != nil {
				log.Error(err, "Failed to create Deployment")
				return false, err
			}
			return true, nil
		}
		log.Error(err, "Failed to get Deployment")
		return false, err
	}

	// Update the Deployment if needed
	if !deploymentEqual(existingDeployment, deployment) {
		log.Info("Updating Deployment", "Deployment.Name", deploymentName)
		existingDeployment.Spec = deployment.Spec
		if err := c.Update(ctx, existingDeployment); err != nil {
			log.Error(err, "Failed to update Deployment")
			return false, err
		}
		return true, nil
	}

	return false, nil
}

// reconcileService reconciles the Service for the SMF
func reconcileService(ctx context.Context, c client.Client, scheme *runtime.Scheme, nfDeployment *nephiov1alpha1.NFDeployment) (bool, error) {
	log := log.FromContext(ctx).WithValues("SMFService", nfDeployment.Name)

	serviceName := controllers.GetNamespacedName(nfDeployment, "smf-service")
	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: nfDeployment.Namespace,
			Labels: map[string]string{
				"app": controllers.GetNamespacedName(nfDeployment, "smf"),
			},
		},
		Spec: apiv1.ServiceSpec{
			Selector: map[string]string{
				"app": controllers.GetNamespacedName(nfDeployment, "smf"),
			},
			Ports: []apiv1.ServicePort{
				{
					Name:       smfPfcpPortName,
					Port:       smfPfcpPort,
					TargetPort: intstr.FromInt(smfPfcpPort),
					Protocol:   apiv1.ProtocolUDP,
				},
				{
					Name:       smfSbiPortName,
					Port:       smfSbiPort,
					TargetPort: intstr.FromInt(smfSbiPort),
					Protocol:   apiv1.ProtocolTCP,
				},
			},
		},
	}

	// Set the owner reference
	if err := controllerutil.SetControllerReference(nfDeployment, service, scheme); err != nil {
		log.Error(err, "Failed to set owner reference on Service")
		return false, err
	}

	// Check if the Service already exists
	existingService := &apiv1.Service{}
	err := c.Get(ctx, client.ObjectKey{Namespace: nfDeployment.Namespace, Name: serviceName}, existingService)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Create the Service
			log.Info("Creating Service", "Service.Name", serviceName)
			if err := c.Create(ctx, service); err != nil {
				log.Error(err, "Failed to create Service")
				return false, err
			}
			return true, nil
		}
		log.Error(err, "Failed to get Service")
		return false, err
	}

	// Update the Service if needed
	if !serviceEqual(existingService, service) {
		log.Info("Updating Service", "Service.Name", serviceName)
		// Preserve the ClusterIP
		service.Spec.ClusterIP = existingService.Spec.ClusterIP
		existingService.Spec = service.Spec
		if err := c.Update(ctx, existingService); err != nil {
			log.Error(err, "Failed to update Service")
			return false, err
		}
		return true, nil
	}

	return false, nil
}

// Helper functions

// int32Ptr returns a pointer to an int32
func int32Ptr(i int32) *int32 {
	return &i
}

// createResourceList creates a resource list from CPU and memory values
func createResourceList(cpu, memory string) apiv1.ResourceList {
	return apiv1.ResourceList{
		apiv1.ResourceCPU:    resource.MustParse(cpu),
		apiv1.ResourceMemory: resource.MustParse(memory),
	}
}

// mapsEqual compares two maps for equality
func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}

// deploymentEqual compares two deployments for equality
func deploymentEqual(a, b *appsv1.Deployment) bool {
	// Simple check for now
	return a.Spec.Template.Spec.Containers[0].Image == b.Spec.Template.Spec.Containers[0].Image
}

// serviceEqual compares two services for equality
func serviceEqual(a, b *apiv1.Service) bool {
	// Simple check for now
	return len(a.Spec.Ports) == len(b.Spec.Ports)
}

// generateSMFRunScript generates the SMF run script
func generateSMFRunScript() string {
	return `#!/bin/bash
cd /free5gc
./bin/smf -c /config/smfcfg.yaml -u /config/uerouting.yaml
`
}

// generateSMFConfig generates the SMF configuration
func generateSMFConfig(nfDeployment *nephiov1alpha1.NFDeployment) string {
	// Get the N4 address from the NFDeployment
	var n4Address string
	for _, iface := range nfDeployment.Spec.Interfaces {
		if iface.Name == "n4" && iface.IPv4 != nil {
			n4Address = iface.IPv4.Address
			break
		}
	}

	// Extract the IP address without the CIDR
	if n4Address != "" {
		n4Address = n4Address[:len(n4Address)-3]
	} else {
		// Default address if not specified
		n4Address = "192.168.250.4"
	}

	return fmt.Sprintf(`info:
  version: 1.0.0
  description: SMF initial configuration

configuration:
  smfName: SMF
  sbi:
    scheme: http
    registerIPv4: %s
    bindingIPv4: 0.0.0.0
    port: 8080
  serviceNameList:
    - nsmf-pdusession
    - nsmf-event-exposure
    - nsmf-oam
  snssaiInfos:
    - sNssai:
        sst: 1
        sd: 010203
      dnnInfos:
        - dnn: internet
          dns:
            ipv4: 8.8.8.8
            ipv6: 2001:4860:4860::8888
  pfcp:
    addr: %s
    nodeID: %s
    retransTimeout: 1
    maxRetrans: 3
  userplane_information:
    up_nodes:
      gNB1:
        type: AN
        an_ip: 192.168.250.1
      UPF:
        type: UPF
        node_id: 192.168.250.3
        up_resource_ip: 192.168.252.3
    links:
      - A: gNB1
        B: UPF
  nrfUri: http://nrf-service:8000
  urrPeriod: 10
  ulcl: false
`, n4Address, n4Address, n4Address)
}

// generateUERoutingConfig generates the UE routing configuration
func generateUERoutingConfig() string {
	return `info:
  version: 1.0.0
  description: Routing information for UE

ueRoutingInfo:
  - SUPI: imsi-2089300007487
    AN: 192.168.250.1
    PathList:
      - DestinationIP: 10.60.0.0/16
        UPF: !!seq
          - BranchingUPF
          - AnchorUPF1
      - DestinationIP: 10.61.0.0/16
        UPF: !!seq
          - BranchingUPF
          - AnchorUPF2

routeProfile:
  - RouteProfileID: internet
    ForwardingPolicyID: 10

pfdDataForApp:
  - applicationId: edge
    pfds:
      - pfdID: pfd1
        flowDescriptions:
          - permit out ip from 10.60.0.0/16 8080 to any
`
}
