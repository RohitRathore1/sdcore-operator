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

package nef

import (
	"errors"

	"github.com/go-logr/logr"
	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	"github.com/nephio-project/sdcore/controllers"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// createDeployment builds a deployment for the NEF
func createDeployment(log logr.Logger, configMapVersion string, nfDeployment *nephiov1alpha1.NFDeployment) (*appsv1.Deployment, error) {
	namespace := nfDeployment.Namespace
	name := nfDeployment.Name
	spec := nfDeployment.Spec

	// Process parameters
	replicas, resourceRequirements, err := createResourceRequirements(spec.ParameterValues)
	if err != nil {
		return nil, err
	}

	podAnnotations := map[string]string{
		controllers.ConfigMapVersionAnnotation: configMapVersion,
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
					"app": name,
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
					Annotations: podAnnotations,
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  "nef",
							Image: "registry.opennetworking.org/sdcore/nef:latest",
							Ports: []apiv1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 8000,
								},
								{
									Name:          "api",
									ContainerPort: 8080,
								},
							},
							Command: []string{
								"/bin/sh",
								"/etc/nef/wrapper.sh",
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      "config-volume",
									MountPath: "/etc/nef",
								},
							},
							Resources: *resourceRequirements,
						},
					},
					Volumes: []apiv1.Volume{
						{
							Name: "config-volume",
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
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

	return deployment, nil
}

// createConfigMap builds a ConfigMap for the NEF
func createConfigMap(log logr.Logger, nfDeployment *nephiov1alpha1.NFDeployment) (*apiv1.ConfigMap, error) {
	namespace := nfDeployment.Namespace
	name := nfDeployment.Name

	// Process parameters for DNS setting
	var dnsValue string
	for _, param := range nfDeployment.Spec.ParameterValues {
		if param.Name == "dns" {
			dnsValue = param.Value
		}
	}

	// Create wrapper script
	wrapperScript := `#!/bin/bash
set -e

# Configure NEF
NEF_CONFIG="/etc/nef/nefcfg.yaml"

# Make sure DNS is set
if [ -n "${DNS}" ]; then
  sed -i "s|DNS_SERVER|${DNS}|g" $NEF_CONFIG
else
  echo "Warning: DNS not set, using default"
  sed -i "s|DNS_SERVER|8.8.8.8|g" $NEF_CONFIG
fi

# Update config if the NRF address has been provided
if [ -n "${NRF_ADDR}" ]; then
  sed -i "s|NRF_ADDR|${NRF_ADDR}|g" $NEF_CONFIG
else
  echo "Warning: NRF_ADDR not set, using default"
  sed -i "s|NRF_ADDR|127.0.0.1:8000|g" $NEF_CONFIG
fi

# Start NEF
/bin/nef -nefcfg $NEF_CONFIG
`

	// Create NEF config
	nefConfig := `
info:
  version: 1.0.0
  description: NEF initial configuration

configuration:
  nefName: NEF
  sbi:
    scheme: http
    registerIPv4: nef-service
    bindingIPv4: 0.0.0.0
    port: 8000
  serviceNameList:
    - nnef-eventexposure
    - nnef-pfdmanagement
  nrfUri: http://NRF_ADDR
  supportedPlmnList:
    - mcc: 208
      mnc: 93
  apiList:
    - apiName: Nnef_EventExposure
      versions:
        - uri: /nef-event-exposure/v1
          version: 1.0.0
    - apiName: Nnef_PFDManagement
      versions:
        - uri: /nef-pfd-management/v1
          version: 1.0.0
  servingAreas:
    - areas:
        - areaCode: A001
          areaName: Area 1
  timeFormat: 2006-01-02 15:04:05
  defaultBdtRefId: BdtRefPolicy01
  apis:
    exposureAPI:
      enabled: true
      port: 8080
      basePath: /api/v1
  oauth:
    enabled: false
    url: http://auth-service:9090
    clientId: nef
    clientSecret: nef
    tokenExpiryMargin: 60

logger:
  NEF:
    debugLevel: info
    ReportCaller: true
  NRF:
    debugLevel: info
    ReportCaller: true
  OpenApi:
    debugLevel: info
    ReportCaller: true
  PathUtil:
    debugLevel: info
    ReportCaller: true
`

	// Create configMap
	configMap := &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{
			"wrapper.sh": wrapperScript,
			"nefcfg.yaml": nefConfig,
		},
	}

	return configMap, nil
}

// createService builds a Service for the NEF
func createService(log logr.Logger, nfDeployment *nephiov1alpha1.NFDeployment) (*apiv1.Service, error) {
	namespace := nfDeployment.Namespace
	name := nfDeployment.Name

	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-service",
			Namespace: namespace,
		},
		Spec: apiv1.ServiceSpec{
			Selector: map[string]string{
				"app": name,
			},
			Ports: []apiv1.ServicePort{
				{
					Name:     "http",
					Protocol: apiv1.ProtocolTCP,
					Port:     8000,
				},
				{
					Name:     "api",
					Protocol: apiv1.ProtocolTCP,
					Port:     8080,
				},
			},
		},
	}

	return service, nil
}

// createResourceRequirements calculates resource requirements based on capacity
func createResourceRequirements(parameterValues []nephiov1alpha1.ParameterValue) (int32, *apiv1.ResourceRequirements, error) {
	// Default to small if not specified
	capacity := "small"

	// Find capacity parameter
	for _, param := range parameterValues {
		if param.Name == "capacity" {
			capacity = param.Value
		}
	}

	var cpuRequest, memoryRequest resource.Quantity
	var cpuLimit, memoryLimit resource.Quantity
	var replicas int32 = 1

	// Set resource requirements based on capacity
	switch capacity {
	case "small":
		cpuRequest = resource.MustParse("100m")
		memoryRequest = resource.MustParse("128Mi")
		cpuLimit = resource.MustParse("200m")
		memoryLimit = resource.MustParse("256Mi")
	case "medium":
		cpuRequest = resource.MustParse("200m")
		memoryRequest = resource.MustParse("256Mi")
		cpuLimit = resource.MustParse("400m")
		memoryLimit = resource.MustParse("512Mi")
	case "large":
		cpuRequest = resource.MustParse("400m")
		memoryRequest = resource.MustParse("512Mi")
		cpuLimit = resource.MustParse("800m")
		memoryLimit = resource.MustParse("1Gi")
		replicas = 2
	default:
		return 0, nil, errors.New("invalid capacity value: " + capacity)
	}

	resourceRequirements := &apiv1.ResourceRequirements{
		Requests: apiv1.ResourceList{
			apiv1.ResourceCPU:    cpuRequest,
			apiv1.ResourceMemory: memoryRequest,
		},
		Limits: apiv1.ResourceList{
			apiv1.ResourceCPU:    cpuLimit,
			apiv1.ResourceMemory: memoryLimit,
		},
	}

	return replicas, resourceRequirements, nil
}