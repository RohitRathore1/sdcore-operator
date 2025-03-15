package amf

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
	// AMF container names
	amfContainerName = "amf"

	// AMF image names
	amfImageName = "registry.opennetworking.org/docker.io/omecproject/5gc-amf:rel-1.6.4"

	// AMF service names
	amfServiceName = "amf-service"

	// AMF port names
	amfNgappPortName    = "ngapp"
	amfSbiPortName      = "sbi"
	amfSctpGrpcPortName = "sctp-grpc"
	amfPromPortName     = "prometheus"

	// AMF port numbers
	amfNgappPort    = 38412
	amfSbiPort      = 8080
	amfSctpGrpcPort = 9000
	amfPromPort     = 9089
)

// reconcileConfigMap reconciles the ConfigMap for the AMF
func reconcileConfigMap(ctx context.Context, c client.Client, nfDeployment *nephiov1alpha1.NFDeployment) (bool, error) {
	log := log.FromContext(ctx).WithValues("AMFConfigMap", nfDeployment.Name)

	configMapName := controllers.GetNamespacedName(nfDeployment, "amf-config")
	configMap := &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: nfDeployment.Namespace,
			Labels: map[string]string{
				"app": controllers.GetNamespacedName(nfDeployment, "amf"),
			},
		},
		Data: map[string]string{
			"amf-run.sh":  generateAMFRunScript(),
			"amfcfg.yaml": generateAMFConfig(nfDeployment),
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

// reconcileDeployment reconciles the Deployment for the AMF
func reconcileDeployment(ctx context.Context, c client.Client, scheme *runtime.Scheme, nfDeployment *nephiov1alpha1.NFDeployment) (bool, error) {
	log := log.FromContext(ctx).WithValues("AMFDeployment", nfDeployment.Name)

	deploymentName := controllers.GetNamespacedName(nfDeployment, "amf")
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
							Name:  amfContainerName,
							Image: amfImageName,
							Ports: []apiv1.ContainerPort{
								{
									Name:          amfNgappPortName,
									ContainerPort: amfNgappPort,
									Protocol:      apiv1.ProtocolSCTP,
								},
								{
									Name:          amfSbiPortName,
									ContainerPort: amfSbiPort,
									Protocol:      apiv1.ProtocolTCP,
								},
								{
									Name:          amfSctpGrpcPortName,
									ContainerPort: amfSctpGrpcPort,
									Protocol:      apiv1.ProtocolTCP,
								},
								{
									Name:          amfPromPortName,
									ContainerPort: amfPromPort,
									Protocol:      apiv1.ProtocolTCP,
								},
							},
							Command: []string{
								"/opt/amf-run.sh",
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      "amf-config",
									MountPath: "/opt",
								},
							},
							Env: []apiv1.EnvVar{
								{
									Name:  "GRPC_GO_LOG_VERBOSITY_LEVEL",
									Value: "99",
								},
								{
									Name:  "GRPC_GO_LOG_SEVERITY_LEVEL",
									Value: "info",
								},
								{
									Name:  "GRPC_TRACE",
									Value: "all",
								},
								{
									Name:  "GRPC_VERBOSITY",
									Value: "DEBUG",
								},
								{
									Name: "POD_IP",
									ValueFrom: &apiv1.EnvVarSource{
										FieldRef: &apiv1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
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
							Name: "amf-config",
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
										Name: controllers.GetNamespacedName(nfDeployment, "amf-config"),
									},
									DefaultMode: int32Ptr(0755), // Executable permission for scripts
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

// reconcileService reconciles the Service for the AMF
func reconcileService(ctx context.Context, c client.Client, scheme *runtime.Scheme, nfDeployment *nephiov1alpha1.NFDeployment) (bool, error) {
	log := log.FromContext(ctx).WithValues("AMFService", nfDeployment.Name)

	serviceName := controllers.GetNamespacedName(nfDeployment, "amf-service")
	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: nfDeployment.Namespace,
			Labels: map[string]string{
				"app": controllers.GetNamespacedName(nfDeployment, "amf"),
			},
		},
		Spec: apiv1.ServiceSpec{
			Selector: map[string]string{
				"app": controllers.GetNamespacedName(nfDeployment, "amf"),
			},
			Ports: []apiv1.ServicePort{
				{
					Name:       amfNgappPortName,
					Port:       amfNgappPort,
					TargetPort: intstr.FromInt(amfNgappPort),
					Protocol:   apiv1.ProtocolSCTP,
				},
				{
					Name:       amfSbiPortName,
					Port:       amfSbiPort,
					TargetPort: intstr.FromInt(amfSbiPort),
					Protocol:   apiv1.ProtocolTCP,
				},
				{
					Name:       amfSctpGrpcPortName,
					Port:       amfSctpGrpcPort,
					TargetPort: intstr.FromInt(amfSctpGrpcPort),
					Protocol:   apiv1.ProtocolTCP,
				},
				{
					Name:       amfPromPortName,
					Port:       amfPromPort,
					TargetPort: intstr.FromInt(amfPromPort),
					Protocol:   apiv1.ProtocolTCP,
				},
			},
		},
	}

	// Create AMF headless service for service discovery
	headlessServiceName := controllers.GetNamespacedName(nfDeployment, "amf-headless")
	headlessService := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      headlessServiceName,
			Namespace: nfDeployment.Namespace,
			Labels: map[string]string{
				"app": controllers.GetNamespacedName(nfDeployment, "amf"),
			},
		},
		Spec: apiv1.ServiceSpec{
			ClusterIP: "None",
			Selector: map[string]string{
				"app": controllers.GetNamespacedName(nfDeployment, "amf"),
			},
			Ports: []apiv1.ServicePort{
				{
					Name:     "grpc",
					Port:     9000,
					Protocol: apiv1.ProtocolTCP,
				},
			},
		},
	}

	// Set the owner reference for main service
	if err := controllerutil.SetControllerReference(nfDeployment, service, scheme); err != nil {
		log.Error(err, "Failed to set owner reference on Service")
		return false, err
	}

	// Set the owner reference for headless service
	if err := controllerutil.SetControllerReference(nfDeployment, headlessService, scheme); err != nil {
		log.Error(err, "Failed to set owner reference on Headless Service")
		return false, err
	}

	// Check if the main Service already exists
	existingService := &apiv1.Service{}
	err := c.Get(ctx, client.ObjectKey{Namespace: nfDeployment.Namespace, Name: serviceName}, existingService)
	servicesChanged := false
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Create the Service
			log.Info("Creating Service", "Service.Name", serviceName)
			if err := c.Create(ctx, service); err != nil {
				log.Error(err, "Failed to create Service")
				return false, err
			}
			servicesChanged = true
		} else {
			log.Error(err, "Failed to get Service")
			return false, err
		}
	} else {
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
			servicesChanged = true
		}
	}

	// Check if the headless Service already exists
	existingHeadlessService := &apiv1.Service{}
	err = c.Get(ctx, client.ObjectKey{Namespace: nfDeployment.Namespace, Name: headlessServiceName}, existingHeadlessService)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Create the headless Service
			log.Info("Creating Headless Service", "Service.Name", headlessServiceName)
			if err := c.Create(ctx, headlessService); err != nil {
				log.Error(err, "Failed to create Headless Service")
				return false, err
			}
			servicesChanged = true
		} else {
			log.Error(err, "Failed to get Headless Service")
			return false, err
		}
	} else {
		// Update the headless Service if needed
		if !serviceEqual(existingHeadlessService, headlessService) {
			log.Info("Updating Headless Service", "Service.Name", headlessServiceName)
			existingHeadlessService.Spec = headlessService.Spec
			if err := c.Update(ctx, existingHeadlessService); err != nil {
				log.Error(err, "Failed to update Headless Service")
				return false, err
			}
			servicesChanged = true
		}
	}

	return servicesChanged, nil
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

// generateAMFRunScript generates the AMF run script
func generateAMFRunScript() string {
	return `#!/bin/bash
cd /free5gc
./bin/amf -c /opt/amfcfg.yaml
`
}

// generateAMFConfig generates the AMF configuration
func generateAMFConfig(nfDeployment *nephiov1alpha1.NFDeployment) string {
	// Get the N2 address from the NFDeployment
	var n2Address string
	for _, iface := range nfDeployment.Spec.Interfaces {
		if iface.Name == "n2" && iface.IPv4 != nil {
			n2Address = iface.IPv4.Address
			break
		}
	}

	// Extract the IP address without the CIDR
	if n2Address != "" {
		n2Address = n2Address[:len(n2Address)-3]
	} else {
		// Default address if not specified
		n2Address = "192.168.251.5"
	}

	return fmt.Sprintf(`info:
  version: 1.0.0
  description: AMF initial configuration

configuration:
  amfName: AMF
  ngapIpList:
    - %s
  sbi:
    scheme: http
    registerIPv4: %s
    bindingIPv4: 0.0.0.0
    port: 8080
  serviceNameList:
    - namf-comm
    - namf-evts
    - namf-mt
    - namf-loc
    - namf-oam
  servedGuamiList:
    - plmnId:
        mcc: 208
        mnc: 93
      amfId: cafe00
  supportTaiList:
    - plmnId:
        mcc: 208
        mnc: 93
      tac: 1
  plmnSupportList:
    - plmnId:
        mcc: 208
        mnc: 93
      snssaiList:
        - sst: 1
          sd: 010203
        - sst: 1
          sd: 112233
  supportDnnList:
    - internet
  nrfUri: http://nrf-service:8000
  security:
    integrityOrder:
      - NIA2
    cipheringOrder:
      - NEA0
  networkName:
    full: free5GC
    short: free
  ngapPort: 38412
  sctpGrpcPort: 9000
  enableSctpLb: false
  t3502: 720
  t3512: 3600
  non3gppDeregistrationTimer: 3240
`, n2Address, n2Address)
}
