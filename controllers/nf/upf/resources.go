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

package upf

import (
	"errors"

	"github.com/go-logr/logr"
	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	"github.com/nephio-project/sdcore/controllers"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createDeployment(log logr.Logger, configMapVersion string, upfDeployment *nephiov1alpha1.NFDeployment) (*appsv1.Deployment, error) {
	namespace := upfDeployment.Namespace
	instanceName := upfDeployment.Name
	spec := upfDeployment.Spec

	var wrapperScriptMode int32 = 0777

	replicas, resourceRequirements, err := createResourceRequirements(spec)
	if err != nil {
		return nil, err
	}

	// Create Kubernetes NetworkAttachmentDefinition networks
	networkAttachmentDefinitionNetworks, err := createNetworkAttachmentDefinitionNetworks(upfDeployment.Name, &spec)
	if err != nil {
		return nil, err
	}

	podAnnotations := make(map[string]string)
	podAnnotations[controllers.ConfigMapVersionAnnotation] = configMapVersion
	podAnnotations[controllers.NetworksAnnotation] = networkAttachmentDefinitionNetworks

	securityContext := &apiv1.SecurityContext{
		Capabilities: &apiv1.Capabilities{
			Add: []apiv1.Capability{"NET_ADMIN"},
		},
	}

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
							Name:            "upf",
							Image:           controllers.UPFImage,
							ImagePullPolicy: apiv1.PullAlways,
							SecurityContext: securityContext,
							Ports: []apiv1.ContainerPort{
								{
									Name:          "n4",
									Protocol:      apiv1.ProtocolUDP,
									ContainerPort: 8805,
								},
							},
							Command: []string{
								"/bin/bash", "/config/wrapper.sh",
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									MountPath: "/config/",
									Name:      "upf-volume",
								},
							},
							Resources: *resourceRequirements,
						},
					},
					Volumes: []apiv1.Volume{
						{
							Name: "upf-volume",
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

func createConfigMap(log logr.Logger, upfDeployment *nephiov1alpha1.NFDeployment) (*apiv1.ConfigMap, error) {
	namespace := upfDeployment.Namespace
	instanceName := upfDeployment.Name

	configMap := &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceName,
			Namespace: namespace,
		},
		Data: map[string]string{
			"wrapper.sh": `#!/bin/bash
# Configuration script for BESS-UPF
set -x

# Start the UPF
cd /opt/bess
./bessctl/bessctl run /opt/bess/bessctl/conf/up4.bess -- --no-core-id

# Keep the container running
tail -f /dev/null
`,
		},
	}

	return configMap, nil
}

func createService(log logr.Logger, upfDeployment *nephiov1alpha1.NFDeployment) (*apiv1.Service, error) {
	namespace := upfDeployment.Namespace
	instanceName := upfDeployment.Name

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
					Name:     "n4",
					Protocol: apiv1.ProtocolUDP,
					Port:     8805,
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
			apiv1.ResourceCPU:    resource.MustParse("500m"),
			apiv1.ResourceMemory: resource.MustParse("512Mi"),
		},
		Limits: apiv1.ResourceList{
			apiv1.ResourceCPU:    resource.MustParse("1000m"),
			apiv1.ResourceMemory: resource.MustParse("1Gi"),
		},
	}

	// If capacity parameters are specified, use them to adjust resources
	if len(spec.ParameterValues) > 0 {
		for _, param := range spec.ParameterValues {
			if param.Name == "capacity" {
				switch param.Value {
				case "small":
					resources.Requests[apiv1.ResourceCPU] = resource.MustParse("500m")
					resources.Requests[apiv1.ResourceMemory] = resource.MustParse("512Mi")
					resources.Limits[apiv1.ResourceCPU] = resource.MustParse("1000m")
					resources.Limits[apiv1.ResourceMemory] = resource.MustParse("1Gi")
				case "medium":
					resources.Requests[apiv1.ResourceCPU] = resource.MustParse("1000m")
					resources.Requests[apiv1.ResourceMemory] = resource.MustParse("1Gi")
					resources.Limits[apiv1.ResourceCPU] = resource.MustParse("2000m")
					resources.Limits[apiv1.ResourceMemory] = resource.MustParse("2Gi")
				case "large":
					resources.Requests[apiv1.ResourceCPU] = resource.MustParse("2000m")
					resources.Requests[apiv1.ResourceMemory] = resource.MustParse("2Gi")
					resources.Limits[apiv1.ResourceCPU] = resource.MustParse("4000m")
					resources.Limits[apiv1.ResourceMemory] = resource.MustParse("4Gi")
				default:
					return 0, nil, errors.New("invalid capacity value: " + param.Value)
				}
			}
		}
	}

	return replicas, &resources, nil
}

func createNetworkAttachmentDefinitionNetworks(name string, spec *nephiov1alpha1.NFDeploymentSpec) (string, error) {
	// This function would create CNI network attachment definitions based on the NFDeployment spec
	// For now, returning a simple example
	return `[
    {
        "name": "n4-net",
        "interface": "n4"
    },
    {
        "name": "n3-net",
        "interface": "n3"
    },
    {
        "name": "n6-net",
        "interface": "n6"
    }
]`, nil
} 