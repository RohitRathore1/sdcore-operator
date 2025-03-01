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

package nssf

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

// createDeployment builds a deployment for the NSSF
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
							Name:  "nssf",
							Image: "registry.opennetworking.org/sdcore/nssf:latest",
							Ports: []apiv1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 8000,
								},
							},
							Command: []string{
								"/bin/sh",
								"/etc/nssf/wrapper.sh",
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      "config-volume",
									MountPath: "/etc/nssf",
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

// createConfigMap builds a ConfigMap for the NSSF
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

# Configure NSSF
NSSF_CONFIG="/etc/nssf/nssfcfg.yaml"

# Make sure DNS is set
if [ -n "${DNS}" ]; then
  sed -i "s|DNS_SERVER|${DNS}|g" $NSSF_CONFIG
else
  echo "Warning: DNS not set, using default"
  sed -i "s|DNS_SERVER|8.8.8.8|g" $NSSF_CONFIG
fi

# Update config if the NRF address has been provided
if [ -n "${NRF_ADDR}" ]; then
  sed -i "s|NRF_ADDR|${NRF_ADDR}|g" $NSSF_CONFIG
else
  echo "Warning: NRF_ADDR not set, using default"
  sed -i "s|NRF_ADDR|127.0.0.1:8000|g" $NSSF_CONFIG
fi

# Start NSSF
/bin/nssf -nssfcfg $NSSF_CONFIG
`

	// Create NSSF config
	nssfConfig := `
info:
  version: 1.0.0
  description: NSSF initial configuration

configuration:
  nssfName: NSSF
  sbi:
    scheme: http
    registerIPv4: nssf-service
    bindingIPv4: 0.0.0.0
    port: 8000
  serviceNameList:
    - nnssf-nsselection
    - nnssf-nssaiavailability
  nrfUri: http://NRF_ADDR
  supportedPlmnList:
    - mcc: 208
      mnc: 93
  supportedNssaiInPlmnList:
    - plmnId:
        mcc: 208
        mnc: 93
      supportedSnssaiList:
        - sst: 1
          sd: 010203
        - sst: 1
          sd: 112233
        - sst: 2
          sd: 000003
  nsiList:
    - snssai:
        sst: 1
        sd: 010203
      nsiInformationList:
        - nrfId: http://NRF_ADDR
          nsiId: 22
    - snssai:
        sst: 1
        sd: 112233
      nsiInformationList:
        - nrfId: http://NRF_ADDR
          nsiId: 23
    - snssai:
        sst: 2
        sd: 000003
      nsiInformationList:
        - nrfId: http://NRF_ADDR
          nsiId: 24
  amfList:
    - nfId: 469de254-2fe5-4ca0-8381-af3f500af77c
      supportedNssaiAvailabilityData:
        - tai:
            plmnId:
              mcc: 208
              mnc: 93
            tac: 1
          supportedSnssaiList:
            - sst: 1
              sd: 010203
            - sst: 1
              sd: 112233
            - sst: 2
              sd: 000003
        - tai:
            plmnId:
              mcc: 208
              mnc: 93
            tac: 2
          supportedSnssaiList:
            - sst: 1
              sd: 010203
    - nfId: fbe604a8-27b2-417e-bd7c-8a7be2691f8d
      supportedNssaiAvailabilityData:
        - tai:
            plmnId:
              mcc: 208
              mnc: 93
            tac: 3
          supportedSnssaiList:
            - sst: 1
              sd: 010203
            - sst: 1
              sd: 112233
        - tai:
            plmnId:
              mcc: 208
              mnc: 93
            tac: 4
          supportedSnssaiList:
            - sst: 1
              sd: 010203
  taList:
    - tai:
        plmnId:
          mcc: 208
          mnc: 93
        tac: 1
      accessType: 3GPP_ACCESS
      supportedSnssaiList:
        - sst: 1
          sd: 010203
        - sst: 1
          sd: 112233
        - sst: 2
          sd: 000003
    - tai:
        plmnId:
          mcc: 208
          mnc: 93
        tac: 2
      accessType: 3GPP_ACCESS
      supportedSnssaiList:
        - sst: 1
          sd: 010203
    - tai:
        plmnId:
          mcc: 208
          mnc: 93
        tac: 3
      accessType: 3GPP_ACCESS
      supportedSnssaiList:
        - sst: 1
          sd: 010203
        - sst: 1
          sd: 112233
  mappingListFromPlmn:
    - operatorName: NTT DoCoMo
      homePlmnId:
        mcc: 440
        mnc: 10
      mappingOfSnssai:
        - servingSnssai:
            sst: 1
            sd: 010203
          homeSnssai:
            sst: 1
            sd: 001001
        - servingSnssai:
            sst: 1
            sd: 112233
          homeSnssai:
            sst: 2
            sd: 001001
        - servingSnssai:
            sst: 2
            sd: 000003
          homeSnssai:
            sst: 1
            sd: 001001

logger:
  NSSF:
    debugLevel: info
    ReportCaller: true
  NRF:
    debugLevel: info
    ReportCaller: true
  PATH_SWITCH:
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
			"wrapper.sh":  wrapperScript,
			"nssfcfg.yaml": nssfConfig,
		},
	}

	return configMap, nil
}

// createService builds a Service for the NSSF
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