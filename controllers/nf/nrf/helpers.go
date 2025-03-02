/*
Copyright 2024 The Nephio Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nrf

import (
	"github.com/RohitRathore1/sdcore-operator/controllers"
	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ConfigMapVersionAnnotation is the annotation key for ConfigMap version
const ConfigMapVersionAnnotation = "nrf.sdcore.io/configmap-version"

// createConfigMap creates a ConfigMap for the NRF deployment
func createConfigMap(logger log.Logger, nfDeployment *nephiov1alpha1.NFDeployment) (*apiv1.ConfigMap, error) {
	capacity := "small"
	dns := "8.8.8.8"

	// Extract parameters from NFDeployment using parameterValues
	for _, param := range nfDeployment.Spec.ParameterValues {
		if param.Name == "capacity" {
			capacity = param.Value
		}
		if param.Name == "dns" {
			dns = param.Value
		}
	}

	logger.Info("Creating NRF ConfigMap", "capacity", capacity, "dns", dns)

	// Create NRF configuration
	nrfConfig := `
{
  "nrfName": "NRF",
  "capacity": "` + capacity + `",
  "dnsServer": "` + dns + `",
  "sbi": {
    "scheme": "http",
    "registerIPv4": "nrf-sbi",
    "bindingIPv4": "0.0.0.0",
    "port": 8000
  },
  "nfManagement": {
    "heartBeatTimer": 30
  },
  "mongodb": {
    "name": "free5gc",
    "url": "mongodb://mongodb:27017"
  },
  "logger": {
    "NRF": {
      "debugLevel": "info",
      "reportCaller": false
    }
  }
}
`

	configMap := &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nfDeployment.Name,
			Namespace: nfDeployment.Namespace,
			Labels: map[string]string{
				"app":     "nrf",
				"nephio":  "true",
				"sdcore":  "true",
				"nf-name": nfDeployment.Name,
			},
		},
		Data: map[string]string{
			"nrfcfg.json": nrfConfig,
		},
	}

	return configMap, nil
}

// createDeployment creates a Deployment for the NRF
func createDeployment(logger log.Logger, configMapVersion string, nfDeployment *nephiov1alpha1.NFDeployment) (*appsv1.Deployment, error) {
	replicas := int32(1)
	
	// Create annotations map if it doesn't exist
	annotations := map[string]string{}
	if configMapVersion != "" {
		annotations[ConfigMapVersionAnnotation] = configMapVersion
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nfDeployment.Name,
			Namespace: nfDeployment.Namespace,
			Labels: map[string]string{
				"app":     "nrf",
				"nephio":  "true",
				"sdcore":  "true",
				"nf-name": nfDeployment.Name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":     "nrf",
					"nf-name": nfDeployment.Name,
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":     "nrf",
						"nephio":  "true",
						"sdcore":  "true",
						"nf-name": nfDeployment.Name,
					},
					Annotations: annotations,
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  "nrf",
							Image: controllers.NRFImage,
							Ports: []apiv1.ContainerPort{
								{
									Name:          "sbi",
									ContainerPort: 8000,
									Protocol:      apiv1.ProtocolTCP,
								},
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/etc/nrf",
								},
							},
							Command: []string{
								"/bin/nrf",
							},
							Args: []string{
								"-c",
								"/etc/nrf/nrfcfg.json",
							},
						},
					},
					Volumes: []apiv1.Volume{
						{
							Name: "config",
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
										Name: nfDeployment.Name,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return deployment, nil
}

// createService creates a Service for the NRF
func createService(logger log.Logger, nfDeployment *nephiov1alpha1.NFDeployment) (*apiv1.Service, error) {
	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nfDeployment.Name,
			Namespace: nfDeployment.Namespace,
			Labels: map[string]string{
				"app":     "nrf",
				"nephio":  "true",
				"sdcore":  "true",
				"nf-name": nfDeployment.Name,
			},
		},
		Spec: apiv1.ServiceSpec{
			Selector: map[string]string{
				"app":     "nrf",
				"nf-name": nfDeployment.Name,
			},
			Ports: []apiv1.ServicePort{
				{
					Name:       "sbi",
					Port:       8000,
					TargetPort: intstr.FromInt(8000),
					Protocol:   apiv1.ProtocolTCP,
				},
			},
		},
	}

	return service, nil
} 