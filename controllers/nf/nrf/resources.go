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
	"errors"

	"github.com/go-logr/logr"
	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	"github.com/RohitRathore1/sdcore-operator/controllers"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createDeployment(log logr.Logger, configMapVersion string, nrfDeployment *nephiov1alpha1.NFDeployment) (*appsv1.Deployment, error) {
	namespace := nrfDeployment.Namespace
	instanceName := nrfDeployment.Name
	spec := nrfDeployment.Spec

	var wrapperScriptMode int32 = 0777

	replicas, resourceRequirements, err := createResourceRequirements(spec)
	if err != nil {
		return nil, err
	}

	podAnnotations := make(map[string]string)
	podAnnotations[controllers.ConfigMapVersionAnnotation] = configMapVersion

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceName,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": instanceName,
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: podAnnotations,
					Labels: map[string]string{
						"name": instanceName,
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:            "nrf",
							Image:           controllers.NRFImage,
							ImagePullPolicy: apiv1.PullAlways,
							Ports: []apiv1.ContainerPort{
								{
									Name:          "nnrf",
									Protocol:      apiv1.ProtocolTCP,
									ContainerPort: 29510,
								},
							},
							Command: []string{
								"/bin/bash", "/config/wrapper.sh",
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									MountPath: "/config/",
									Name:      "nrf-volume",
								},
							},
							Resources: *resourceRequirements,
						},
					},
					Volumes: []apiv1.Volume{
						{
							Name: "nrf-volume",
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
										Name: instanceName,
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

	return deployment, nil
}

func createConfigMap(log logr.Logger, nrfDeployment *nephiov1alpha1.NFDeployment) (*apiv1.ConfigMap, error) {
	namespace := nrfDeployment.Namespace
	instanceName := nrfDeployment.Name

	configMap := &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceName,
			Namespace: namespace,
		},
		Data: map[string]string{
			"wrapper.sh": `#!/bin/bash
# Configuration script for NRF
set -x

# Create a config directory for NRF
mkdir -p /free5gc/config/nrfcfg.yaml

# Copy the default config
cp /free5gc/config/nrfcfg.conf /free5gc/config/nrfcfg.yaml

# Start the NRF
cd /free5gc
./bin/nrf -c config/nrfcfg.yaml

# Keep the container running
tail -f /dev/null
`,
		},
	}

	return configMap, nil
}

func createService(log logr.Logger, nrfDeployment *nephiov1alpha1.NFDeployment) (*apiv1.Service, error) {
	namespace := nrfDeployment.Namespace
	instanceName := nrfDeployment.Name

	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceName,
			Namespace: namespace,
		},
		Spec: apiv1.ServiceSpec{
			Selector: map[string]string{
				"name": instanceName,
			},
			Ports: []apiv1.ServicePort{
				{
					Name:     "nnrf",
					Protocol: apiv1.ProtocolTCP,
					Port:     29510,
				},
			},
		},
	}

	return service, nil
}

func createResourceRequirements(spec nephiov1alpha1.NFDeploymentSpec) (int32, *apiv1.ResourceRequirements, error) {
	var replicas int32 = 1

	// Calculate resource requirements from capacity parameters
	resources := apiv1.ResourceRequirements{
		Requests: apiv1.ResourceList{
			apiv1.ResourceCPU:    resource.MustParse("250m"),
			apiv1.ResourceMemory: resource.MustParse("256Mi"),
		},
		Limits: apiv1.ResourceList{
			apiv1.ResourceCPU:    resource.MustParse("500m"),
			apiv1.ResourceMemory: resource.MustParse("512Mi"),
		},
	}

	// If capacity parameters are specified, use them to adjust resources
	if len(spec.ParameterValues) > 0 {
		for _, param := range spec.ParameterValues {
			if param.Name == "capacity" {
				switch param.Value {
				case "small":
					resources.Requests[apiv1.ResourceCPU] = resource.MustParse("250m")
					resources.Requests[apiv1.ResourceMemory] = resource.MustParse("256Mi")
					resources.Limits[apiv1.ResourceCPU] = resource.MustParse("500m")
					resources.Limits[apiv1.ResourceMemory] = resource.MustParse("512Mi")
				case "medium":
					resources.Requests[apiv1.ResourceCPU] = resource.MustParse("500m")
					resources.Requests[apiv1.ResourceMemory] = resource.MustParse("512Mi")
					resources.Limits[apiv1.ResourceCPU] = resource.MustParse("1000m")
					resources.Limits[apiv1.ResourceMemory] = resource.MustParse("1Gi")
				case "large":
					resources.Requests[apiv1.ResourceCPU] = resource.MustParse("1000m")
					resources.Requests[apiv1.ResourceMemory] = resource.MustParse("1Gi")
					resources.Limits[apiv1.ResourceCPU] = resource.MustParse("2000m")
					resources.Limits[apiv1.ResourceMemory] = resource.MustParse("2Gi")
				default:
					return 0, nil, errors.New("invalid capacity value: " + param.Value)
				}
			}
		}
	}

	return replicas, &resources, nil
} 