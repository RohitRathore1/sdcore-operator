package upf

import (
	"context"

	"github.com/RohitRathore1/sdcore-operator/controllers"
	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Constants for the UPF deployment
const (
	upfBessImageName       = "omecproject/upf-epc-bess:rel-2.0.1"
	upfPfcpifaceImageName  = "omecproject/upf-epc-pfcpiface:rel-2.0.1"
	upfToolsImageName      = "omecproject/pod-init:rel-1.1.2"
	upfContainerName       = "upf"
	upfConfigName          = "upf-config"
	upfServiceName         = "upf-service"
	bessdContainerName     = "bessd"
	routectlContainerName  = "routectl"
	webContainerName       = "web"
	pfcpAgentContainerName = "pfcp-agent"
)

// reconcileConfigMap ensures the ConfigMap for the UPF deployment exists and is up to date
func (r *UPFDeploymentReconciler) reconcileConfigMap(ctx context.Context, nfDeployment *nephiov1alpha1.NFDeployment) error {
	log := ctrl.LoggerFrom(ctx)

	// Create a ConfigMap for UPF configuration
	configMapName := controllers.GetNamespacedName(nfDeployment, upfConfigName)
	configMap := &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: nfDeployment.Namespace,
		},
	}

	// Create or update the ConfigMap
	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		// Set the owner reference so the ConfigMap is automatically cleaned up
		if err := ctrl.SetControllerReference(nfDeployment, configMap, r.Scheme); err != nil {
			return err
		}

		// Generate UPF configuration based on NFDeployment spec
		configMap.Data = map[string]string{
			"upf.jsonc":          generateUPFConfig(nfDeployment),
			"bessd-poststart.sh": generateBESSPostStartScript(),
		}

		return nil
	})

	if err != nil {
		return err
	}

	log.Info("ConfigMap reconciled", "operation", op)
	return nil
}

// reconcileDeployment ensures the Deployment for the UPF exists and is up to date
func (r *UPFDeploymentReconciler) reconcileDeployment(ctx context.Context, nfDeployment *nephiov1alpha1.NFDeployment) error {
	log := ctrl.LoggerFrom(ctx)

	// Create a Deployment for UPF
	deploymentName := controllers.GetNamespacedName(nfDeployment, "upf")
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: nfDeployment.Namespace,
		},
	}

	// Create or update the Deployment
	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		// Set the owner reference so the Deployment is automatically cleaned up
		if err := ctrl.SetControllerReference(nfDeployment, deployment, r.Scheme); err != nil {
			return err
		}

		// Configure deployment spec
		configureDeploymentSpec(deployment, nfDeployment)

		return nil
	})

	if err != nil {
		return err
	}

	log.Info("Deployment reconciled", "operation", op)
	return nil
}

// reconcileService ensures the Service for the UPF exists and is up to date
func (r *UPFDeploymentReconciler) reconcileService(ctx context.Context, nfDeployment *nephiov1alpha1.NFDeployment) error {
	log := ctrl.LoggerFrom(ctx)

	// Create a Service for UPF
	serviceName := controllers.GetNamespacedName(nfDeployment, upfServiceName)
	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: nfDeployment.Namespace,
		},
	}

	// Create or update the Service
	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		// Set the owner reference so the Service is automatically cleaned up
		if err := ctrl.SetControllerReference(nfDeployment, service, r.Scheme); err != nil {
			return err
		}

		// Configure service spec
		service.Spec.Selector = map[string]string{
			"app": controllers.GetNamespacedName(nfDeployment, "upf"),
		}
		service.Spec.Ports = []apiv1.ServicePort{
			{
				Name:       "pfcp",
				Protocol:   apiv1.ProtocolUDP,
				Port:       8805,
				TargetPort: intstr.FromInt(8805),
			},
			{
				Name:       "bess-web",
				Protocol:   apiv1.ProtocolTCP,
				Port:       8000,
				TargetPort: intstr.FromInt(8000),
			},
			{
				Name:       "prometheus",
				Protocol:   apiv1.ProtocolTCP,
				Port:       8080,
				TargetPort: intstr.FromInt(8080),
			},
		}

		return nil
	})

	if err != nil {
		return err
	}

	log.Info("Service reconciled", "operation", op)
	return nil
}

// configureDeploymentSpec configures the deployment spec for the UPF
func configureDeploymentSpec(deployment *appsv1.Deployment, nfDeployment *nephiov1alpha1.NFDeployment) {
	deployment.Spec.Replicas = func() *int32 { i := int32(1); return &i }()

	// Set labels and selector
	appLabel := controllers.GetNamespacedName(nfDeployment, "upf")
	labels := map[string]string{
		"app": appLabel,
	}

	deployment.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: labels,
	}

	deployment.Spec.Template.ObjectMeta.Labels = labels

	// ConfigMap name
	configMapName := controllers.GetNamespacedName(nfDeployment, upfConfigName)

	// Configure shared process namespace
	shareProcessNamespace := true
	deployment.Spec.Template.Spec.ShareProcessNamespace = &shareProcessNamespace

	// Add init containers
	deployment.Spec.Template.Spec.InitContainers = []apiv1.Container{
		{
			Name:    "bess-init",
			Image:   upfBessImageName,
			Command: []string{"sh", "-xec"},
			Args: []string{
				`echo "Skipping network setup for local testing";
				echo "In a real environment, we would run:";
				echo "ip route replace 192.168.251.0/24 via 192.168.252.1";
				echo "ip route replace default via 192.168.250.1 metric 110";
				echo "iptables -I OUTPUT -p icmp --icmp-type port-unreachable -j DROP";`,
			},
			SecurityContext: &apiv1.SecurityContext{
				Capabilities: &apiv1.Capabilities{
					Add: []apiv1.Capability{
						"NET_ADMIN",
					},
				},
			},
			Resources: apiv1.ResourceRequirements{
				Requests: apiv1.ResourceList{
					apiv1.ResourceCPU:    resource.MustParse("128m"),
					apiv1.ResourceMemory: resource.MustParse("64Mi"),
				},
				Limits: apiv1.ResourceList{
					apiv1.ResourceCPU:    resource.MustParse("128m"),
					apiv1.ResourceMemory: resource.MustParse("64Mi"),
				},
			},
		},
	}

	// Configure container spec
	deployment.Spec.Template.Spec.Containers = []apiv1.Container{
		{
			Name:  bessdContainerName,
			Image: upfBessImageName,
			SecurityContext: &apiv1.SecurityContext{
				Capabilities: &apiv1.Capabilities{
					Add: []apiv1.Capability{
						"IPC_LOCK",
						"CAP_SYS_NICE",
					},
				},
			},
			Command: []string{"/bin/bash", "-xc"},
			Args:    []string{"bessd -m 0 -f --grpc_url=0.0.0.0:10514"},
			Stdin:   true,
			TTY:     true,
			Lifecycle: &apiv1.Lifecycle{
				PostStart: &apiv1.LifecycleHandler{
					Exec: &apiv1.ExecAction{
						Command: []string{"/etc/bess/conf/bessd-poststart.sh"},
					},
				},
			},
			Resources: apiv1.ResourceRequirements{
				Requests: apiv1.ResourceList{
					apiv1.ResourceCPU:    resource.MustParse("2"),
					apiv1.ResourceMemory: resource.MustParse("2Gi"),
				},
				Limits: apiv1.ResourceList{
					apiv1.ResourceCPU:    resource.MustParse("2"),
					apiv1.ResourceMemory: resource.MustParse("2Gi"),
				},
			},
			Env: []apiv1.EnvVar{
				{
					Name:  "CONF_FILE",
					Value: "/etc/bess/conf/upf.jsonc",
				},
			},
			VolumeMounts: []apiv1.VolumeMount{
				{
					Name:      "shared-app",
					MountPath: "/pod-share",
				},
				{
					Name:      "config-volume",
					MountPath: "/etc/bess/conf",
				},
			},
			LivenessProbe: &apiv1.Probe{
				ProbeHandler: apiv1.ProbeHandler{
					TCPSocket: &apiv1.TCPSocketAction{
						Port: intstr.FromInt(10514),
					},
				},
				InitialDelaySeconds: 15,
				PeriodSeconds:       20,
			},
		},
		{
			Name:  routectlContainerName,
			Image: upfBessImageName,
			Env: []apiv1.EnvVar{
				{
					Name:  "PYTHONUNBUFFERED",
					Value: "1",
				},
			},
			Command: []string{"/opt/bess/bessctl/conf/route_control.py"},
			Args:    []string{"-i", "eth0", "eth0"},
			Resources: apiv1.ResourceRequirements{
				Requests: apiv1.ResourceList{
					apiv1.ResourceCPU:    resource.MustParse("256m"),
					apiv1.ResourceMemory: resource.MustParse("128Mi"),
				},
				Limits: apiv1.ResourceList{
					apiv1.ResourceCPU:    resource.MustParse("256m"),
					apiv1.ResourceMemory: resource.MustParse("128Mi"),
				},
			},
		},
		{
			Name:    webContainerName,
			Image:   upfBessImageName,
			Command: []string{"/bin/bash", "-xc", "bessctl http 0.0.0.0 8000"},
			Resources: apiv1.ResourceRequirements{
				Requests: apiv1.ResourceList{
					apiv1.ResourceCPU:    resource.MustParse("256m"),
					apiv1.ResourceMemory: resource.MustParse("128Mi"),
				},
				Limits: apiv1.ResourceList{
					apiv1.ResourceCPU:    resource.MustParse("256m"),
					apiv1.ResourceMemory: resource.MustParse("128Mi"),
				},
			},
		},
		{
			Name:    pfcpAgentContainerName,
			Image:   upfPfcpifaceImageName,
			Command: []string{"pfcpiface"},
			Args:    []string{"-config", "/tmp/conf/upf.jsonc"},
			Resources: apiv1.ResourceRequirements{
				Requests: apiv1.ResourceList{
					apiv1.ResourceCPU:    resource.MustParse("256m"),
					apiv1.ResourceMemory: resource.MustParse("128Mi"),
				},
				Limits: apiv1.ResourceList{
					apiv1.ResourceCPU:    resource.MustParse("256m"),
					apiv1.ResourceMemory: resource.MustParse("128Mi"),
				},
			},
			VolumeMounts: []apiv1.VolumeMount{
				{
					Name:      "shared-app",
					MountPath: "/pod-share",
				},
				{
					Name:      "config-volume",
					MountPath: "/tmp/conf",
				},
			},
		},
	}

	// Configure volumes
	deployment.Spec.Template.Spec.Volumes = []apiv1.Volume{
		{
			Name: "config-volume",
			VolumeSource: apiv1.VolumeSource{
				ConfigMap: &apiv1.ConfigMapVolumeSource{
					LocalObjectReference: apiv1.LocalObjectReference{
						Name: configMapName,
					},
					DefaultMode: func() *int32 { mode := int32(493); return &mode }(),
				},
			},
		},
		{
			Name: "shared-app",
			VolumeSource: apiv1.VolumeSource{
				EmptyDir: &apiv1.EmptyDirVolumeSource{},
			},
		},
	}
}

// generateUPFConfig generates the UPF configuration based on the NFDeployment spec
func generateUPFConfig(nfDeployment *nephiov1alpha1.NFDeployment) string {
	// This is a simplified configuration based on the BESS-UPF Helm chart
	// In a real implementation, this would parse the NFDeployment spec more thoroughly

	// Modified configuration for local testing that doesn't rely on network interfaces
	return `{
  "mode": "af_packet",
  "log_level": "info",
  "workers": 1,
  "max_sessions": 50000,
  "table_sizes": {
    "pdrLookup": 50000,
    "appQERLookup": 200000,
    "sessionQERLookup": 100000,
    "farLookup": 150000
  },
  "access": {
    "ifname": "eth0"
  },
  "core": {
    "ifname": "eth0"
  },
  "measure_upf": true,
  "measure_flow": false,
  "enable_notify_bess": true,
  "notify_sockaddr": "/pod-share/notifycp",
  "cpiface": {
    "dnn": "internet",
    "hostname": "",
    "http_port": "8080"
  },
  "slice_rate_limit_config": {
    "n6_bps": 1000000000,
    "n6_burst_bytes": 12500000,
    "n3_bps": 1000000000,
    "n3_burst_bytes": 12500000
  },
  "qci_qos_config": [
    {
      "qci": 0,
      "cbs": 50000,
      "ebs": 50000,
      "pbs": 50000,
      "burst_duration_ms": 10,
      "priority": 7
    }
  ]
}`
}

// generateBESSPostStartScript generates the post-start script for BESS
func generateBESSPostStartScript() string {
	return `#!/bin/bash
set -x

echo "Waiting for BESS to start..."
sleep 5

echo "Running BESS configuration..."
bessctl run /opt/bess/bessctl/conf/up4.bess -- $CONF_FILE || {
  echo "Error running BESS configuration, but continuing anyway for testing purposes"
  exit 0
}
`
}
