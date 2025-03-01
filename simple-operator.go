package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

// NFDeployment resource definition
var nfDeploymentGVR = schema.GroupVersionResource{
	Group:    "workload.nephio.org",
	Version:  "v1alpha1",
	Resource: "nfdeployments",
}

// SDCore container images
const (
	// Common images
	InitImage     = "omecproject/pod-init:rel-1.1.2"
	SctplbImage   = "omecproject/sctplb:rel-1.6.1"
	MetricImage   = "omecproject/metricfunc:rel-1.6.1"
	UpfAdapterImage = "omecproject/upfadapter:rel-2.0.2"
	
	// 5G Core images
	AMFImage      = "omecproject/5gc-amf:rel-1.6.4"
	NRFImage      = "omecproject/5gc-nrf:rel-1.6.3"
	SMFImage      = "omecproject/5gc-smf:rel-2.0.3"
	AUSFImage     = "omecproject/5gc-ausf:rel-1.6.2"
	NSSFImage     = "omecproject/5gc-nssf:rel-1.6.2"
	PCFImage      = "omecproject/5gc-pcf:rel-1.6.2"
	UDRImage      = "omecproject/5gc-udr:rel-1.6.3"
	UDMImage      = "omecproject/5gc-udm:rel-1.6.2"
	WebUIImage    = "omecproject/5gc-webui:rel-1.8.3"
	
	// Default UPF image (may be overridden by newer versions)
	UPFImage      = "registry.aetherproject.org/omecproject/upf-epc:1.5.0"
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	// Get Kubernetes config
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fallback to kubeconfig
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = os.Getenv("HOME") + "/.kube/config"
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			klog.Fatalf("Error building kubeconfig: %s", err.Error())
		}
	}

	// Create Kubernetes clients
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Error creating Kubernetes client: %s", err.Error())
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Error creating dynamic client: %s", err.Error())
	}

	// Start the controller loop
	klog.Info("Starting SDCore Operator")
	for {
		// List all NFDeployment resources
		nfDeployments, err := dynamicClient.Resource(nfDeploymentGVR).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			klog.Errorf("Error listing NFDeployments: %s", err.Error())
			time.Sleep(10 * time.Second)
			continue
		}

		// Process each NFDeployment
		for _, nfDeployment := range nfDeployments.Items {
			provider, found, err := unstructured.NestedString(nfDeployment.Object, "spec", "provider")
			if err != nil || !found {
				klog.Errorf("Error getting provider from NFDeployment %s: %v", nfDeployment.GetName(), err)
				continue
			}

			// Process based on provider
			switch {
			case provider == "upf.sdcore.io":
				if err := processUPF(clientset, &nfDeployment); err != nil {
					klog.Errorf("Error processing UPF: %v", err)
				}
			case provider == "amf.sdcore.io":
				if err := processAMF(clientset, &nfDeployment); err != nil {
					klog.Errorf("Error processing AMF: %v", err)
				}
			case provider == "smf.sdcore.io":
				if err := processSMF(clientset, &nfDeployment); err != nil {
					klog.Errorf("Error processing SMF: %v", err)
				}
			case provider == "nrf.sdcore.io":
				if err := processNRF(clientset, &nfDeployment); err != nil {
					klog.Errorf("Error processing NRF: %v", err)
				}
			case provider == "ausf.sdcore.io":
				if err := processAUSF(clientset, &nfDeployment); err != nil {
					klog.Errorf("Error processing AUSF: %v", err)
				}
			case provider == "nssf.sdcore.io":
				if err := processNSSF(clientset, &nfDeployment); err != nil {
					klog.Errorf("Error processing NSSF: %v", err)
				}
			case provider == "pcf.sdcore.io":
				if err := processPCF(clientset, &nfDeployment); err != nil {
					klog.Errorf("Error processing PCF: %v", err)
				}
			case provider == "udr.sdcore.io":
				if err := processUDR(clientset, &nfDeployment); err != nil {
					klog.Errorf("Error processing UDR: %v", err)
				}
			case provider == "udm.sdcore.io":
				if err := processUDM(clientset, &nfDeployment); err != nil {
					klog.Errorf("Error processing UDM: %v", err)
				}
			default:
				klog.Warningf("Unsupported provider type: %s for NFDeployment: %s", 
					provider, nfDeployment.GetName())
			}
		}

		// Sleep before next reconciliation
		time.Sleep(30 * time.Second)
	}
}

// processUPF handles UPF related NFDeployments
func processUPF(clientset *kubernetes.Clientset, nfDeployment *unstructured.Unstructured) error {
	klog.Infof("Processing UPF NFDeployment: %s", nfDeployment.GetName())

	// Extract parameters
	capacity := "small" // default
	dns := "8.8.8.8"    // default

	paramValues, found, err := unstructured.NestedSlice(nfDeployment.Object, "spec", "parameterValues")
	if err == nil && found {
		for _, param := range paramValues {
			paramMap, ok := param.(map[string]interface{})
			if !ok {
				continue
			}

			name, ok := paramMap["name"].(string)
			if !ok {
				continue
			}

			value, ok := paramMap["value"].(string)
			if !ok {
				continue
			}

			if name == "capacity" {
				capacity = value
			} else if name == "dns" {
				dns = value
			}
		}
	}

	klog.Infof("Using capacity: %s, DNS: %s", capacity, dns)

	// Create or update ConfigMap
	err = createOrUpdateConfigMap(clientset, nfDeployment.GetNamespace(), nfDeployment.GetName(), capacity, dns)
	if err != nil {
		klog.Errorf("Error creating/updating ConfigMap: %s", err.Error())
		return err
	}

	// Create or update Deployment
	err = createOrUpdateUPFDeployment(clientset, nfDeployment.GetNamespace(), nfDeployment.GetName(), capacity)
	if err != nil {
		klog.Errorf("Error creating/updating Deployment: %s", err.Error())
		return err
	}

	// Create or update Service
	err = createOrUpdateService(clientset, nfDeployment.GetNamespace(), nfDeployment.GetName())
	if err != nil {
		klog.Errorf("Error creating/updating Service: %s", err.Error())
		return err
	}

	klog.Infof("Successfully processed UPF NFDeployment: %s", nfDeployment.GetName())
	return nil
}

// processAMF handles AMF related NFDeployments
func processAMF(clientset *kubernetes.Clientset, nfDeployment *unstructured.Unstructured) error {
	klog.Infof("Processing AMF NFDeployment: %s", nfDeployment.GetName())
	
	// Create or update ConfigMap with AMF-specific configuration
	if err := createOrUpdateAMFConfigMap(clientset, nfDeployment); err != nil {
		return err
	}
	
	// Create or update AMF deployment
	if err := createOrUpdateAMFDeployment(clientset, nfDeployment); err != nil {
		return err
	}
	
	// Create or update AMF service
	if err := createOrUpdateAMFService(clientset, nfDeployment); err != nil {
		return err
	}
	
	klog.Infof("Successfully processed AMF NFDeployment: %s", nfDeployment.GetName())
	return nil
}

// processSMF handles SMF related NFDeployments
func processSMF(clientset *kubernetes.Clientset, nfDeployment *unstructured.Unstructured) error {
	klog.Infof("Processing SMF NFDeployment: %s", nfDeployment.GetName())
	
	// Implementation for SMF deployment
	// For brevity, we're not implementing all functions now
	
	klog.Infof("Successfully processed SMF NFDeployment: %s", nfDeployment.GetName())
	return nil
}

// processNRF handles NRF related NFDeployments
func processNRF(clientset *kubernetes.Clientset, nfDeployment *unstructured.Unstructured) error {
	klog.Infof("Processing NRF NFDeployment: %s", nfDeployment.GetName())
	
	// Implementation for NRF deployment
	
	klog.Infof("Successfully processed NRF NFDeployment: %s", nfDeployment.GetName())
	return nil
}

// processAUSF handles AUSF related NFDeployments
func processAUSF(clientset *kubernetes.Clientset, nfDeployment *unstructured.Unstructured) error {
	klog.Infof("Processing AUSF NFDeployment: %s", nfDeployment.GetName())
	
	// Implementation for AUSF deployment
	
	klog.Infof("Successfully processed AUSF NFDeployment: %s", nfDeployment.GetName())
	return nil
}

// processNSSF handles NSSF related NFDeployments
func processNSSF(clientset *kubernetes.Clientset, nfDeployment *unstructured.Unstructured) error {
	klog.Infof("Processing NSSF NFDeployment: %s", nfDeployment.GetName())
	
	// Implementation for NSSF deployment
	
	klog.Infof("Successfully processed NSSF NFDeployment: %s", nfDeployment.GetName())
	return nil
}

// processPCF handles PCF related NFDeployments
func processPCF(clientset *kubernetes.Clientset, nfDeployment *unstructured.Unstructured) error {
	klog.Infof("Processing PCF NFDeployment: %s", nfDeployment.GetName())
	
	// Implementation for PCF deployment
	
	klog.Infof("Successfully processed PCF NFDeployment: %s", nfDeployment.GetName())
	return nil
}

// processUDR handles UDR related NFDeployments
func processUDR(clientset *kubernetes.Clientset, nfDeployment *unstructured.Unstructured) error {
	klog.Infof("Processing UDR NFDeployment: %s", nfDeployment.GetName())
	
	// Implementation for UDR deployment
	
	klog.Infof("Successfully processed UDR NFDeployment: %s", nfDeployment.GetName())
	return nil
}

// processUDM handles UDM related NFDeployments
func processUDM(clientset *kubernetes.Clientset, nfDeployment *unstructured.Unstructured) error {
	klog.Infof("Processing UDM NFDeployment: %s", nfDeployment.GetName())
	
	// Implementation for UDM deployment
	
	klog.Infof("Successfully processed UDM NFDeployment: %s", nfDeployment.GetName())
	return nil
}

// AMF specific functions
func createOrUpdateAMFConfigMap(clientset *kubernetes.Clientset, nfDeployment *unstructured.Unstructured) error {
	namespace := nfDeployment.GetNamespace()
	name := nfDeployment.GetName()
	
	configMapData := map[string]string{
		"amf.yaml": `
info:
  version: 1.0.0
  description: AMF initial configuration
configuration:
  amfName: AMF
  serviceNameList:
    - namf-comm
    - namf-evts
    - namf-mt
    - namf-loc
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
  networkName:
    full: 5GS-SDCore
    short: 5GS
  nrfUri: http://nrf:8000
  security:
    integrityOrder:
      - NIA2
    cipheringOrder:
      - NEA0
  networkFeatureSupport5GS:
    enable: true
  # The IP address used for 5G NF services
  sbi:
    scheme: http
    ipv4Addr: 0.0.0.0
    port: 8000
  ngap:
    port: 38412
  `,
	}
	
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: configMapData,
	}
	
	_, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create ConfigMap
			_, err = clientset.CoreV1().ConfigMaps(namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create AMF ConfigMap: %w", err)
			}
			klog.Infof("Created AMF ConfigMap %s/%s", namespace, name)
			return nil
		}
		return fmt.Errorf("failed to get AMF ConfigMap: %w", err)
	}
	
	// Update ConfigMap
	_, err = clientset.CoreV1().ConfigMaps(namespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update AMF ConfigMap: %w", err)
	}
	klog.Infof("Updated AMF ConfigMap %s/%s", namespace, name)
	return nil
}

func createOrUpdateAMFDeployment(clientset *kubernetes.Clientset, nfDeployment *unstructured.Unstructured) error {
	namespace := nfDeployment.GetNamespace()
	name := nfDeployment.GetName()
	
	var replicas int32 = 1
	
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "amf",
							Image:           AMFImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 8000,
								},
								{
									Name:          "ngap",
									Protocol:      corev1.ProtocolSCTP,
									ContainerPort: 38412,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config-volume",
									MountPath: "/free5gc/config/",
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: name,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	
	_, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create Deployment
			_, err = clientset.AppsV1().Deployments(namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create AMF Deployment: %w", err)
			}
			klog.Infof("Created AMF Deployment %s/%s", namespace, name)
			return nil
		}
		return fmt.Errorf("failed to get AMF Deployment: %w", err)
	}
	
	// Update Deployment
	_, err = clientset.AppsV1().Deployments(namespace).Update(context.TODO(), deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update AMF Deployment: %w", err)
	}
	klog.Infof("Updated AMF Deployment %s/%s", namespace, name)
	return nil
}

func createOrUpdateAMFService(clientset *kubernetes.Clientset, nfDeployment *unstructured.Unstructured) error {
	namespace := nfDeployment.GetNamespace()
	name := nfDeployment.GetName()
	
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Protocol: corev1.ProtocolTCP,
					Port:     8000,
				},
				{
					Name:     "ngap",
					Protocol: corev1.ProtocolSCTP,
					Port:     38412,
				},
			},
		},
	}
	
	_, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create Service
			_, err = clientset.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create AMF Service: %w", err)
			}
			klog.Infof("Created AMF Service %s/%s", namespace, name)
			return nil
		}
		return fmt.Errorf("failed to get AMF Service: %w", err)
	}
	
	// Update Service
	_, err = clientset.CoreV1().Services(namespace).Update(context.TODO(), service, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update AMF Service: %w", err)
	}
	klog.Infof("Updated AMF Service %s/%s", namespace, name)
	return nil
}

// UPF specific functions
func createOrUpdateConfigMap(clientset *kubernetes.Clientset, namespace, name, capacity, dns string) error {
	configMapData := map[string]string{
		"upf.yaml": fmt.Sprintf(`
version: 1.0
description: UPF Configuration
capacity: %s
dns: %s
# Additional UPF specific configuration would go here
`, capacity, dns),
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: configMapData,
	}

	_, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create ConfigMap
			_, err = clientset.CoreV1().ConfigMaps(namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create ConfigMap: %w", err)
			}
			klog.Infof("Created ConfigMap %s/%s", namespace, name)
			return nil
		}
		return fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	// Update ConfigMap
	_, err = clientset.CoreV1().ConfigMaps(namespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update ConfigMap: %w", err)
	}
	klog.Infof("Updated ConfigMap %s/%s", namespace, name)
	return nil
}

func createOrUpdateUPFDeployment(clientset *kubernetes.Clientset, namespace, name, capacity string) error {
	var replicas int32 = 1
	var wrapperScriptMode int32 = 0777

	// Set resource requirements based on capacity
	cpuRequest := "500m"
	memoryRequest := "512Mi"
	cpuLimit := "1000m"
	memoryLimit := "1Gi"

	switch capacity {
	case "medium":
		cpuRequest = "1000m"
		memoryRequest = "1Gi"
		cpuLimit = "2000m"
		memoryLimit = "2Gi"
	case "large":
		cpuRequest = "2000m"
		memoryRequest = "2Gi"
		cpuLimit = "4000m"
		memoryLimit = "4Gi"
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"name": name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "upf",
							Image:           UPFImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{"NET_ADMIN"},
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "n4",
									Protocol:      corev1.ProtocolUDP,
									ContainerPort: 8805,
								},
							},
							Command: []string{
								"/bin/bash", "/config/wrapper.sh",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/config/",
									Name:      "upf-volume",
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(cpuRequest),
									corev1.ResourceMemory: resource.MustParse(memoryRequest),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(cpuLimit),
									corev1.ResourceMemory: resource.MustParse(memoryLimit),
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "upf-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: name,
									},
									DefaultMode: &wrapperScriptMode,
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create Deployment
			_, err = clientset.AppsV1().Deployments(namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create UPF Deployment: %w", err)
			}
			klog.Infof("Created Deployment %s/%s", namespace, name)
			return nil
		}
		return fmt.Errorf("failed to get Deployment: %w", err)
	}

	// Update Deployment
	_, err = clientset.AppsV1().Deployments(namespace).Update(context.TODO(), deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update Deployment: %w", err)
	}
	klog.Infof("Updated Deployment %s/%s", namespace, name)
	return nil
}

func createOrUpdateService(clientset *kubernetes.Clientset, namespace, name string) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"name": name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:     "n4",
					Protocol: corev1.ProtocolUDP,
					Port:     8805,
				},
			},
		},
	}

	_, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create Service
			_, err = clientset.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create Service: %w", err)
			}
			klog.Infof("Created Service %s/%s", namespace, name)
			return nil
		}
		return fmt.Errorf("failed to get Service: %w", err)
	}

	// Update Service
	_, err = clientset.CoreV1().Services(namespace).Update(context.TODO(), service, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update Service: %w", err)
	}
	klog.Infof("Updated Service %s/%s", namespace, name)
	return nil
} 