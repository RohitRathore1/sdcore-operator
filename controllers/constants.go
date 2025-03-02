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

package controllers

import (
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
)

// ProviderFilter creates a predicate.Funcs that filters NFDeployments by provider
func ProviderFilter(provider string) predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return filterObj(e.Object, provider)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return filterObj(e.ObjectNew, provider)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return filterObj(e.Object, provider)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return filterObj(e.Object, provider)
		},
	}
}

// filterObj returns true if the object is an NFDeployment with the specified provider
func filterObj(obj runtime.Object, provider string) bool {
	nfDeployment, ok := obj.(*nephiov1alpha1.NFDeployment)
	if !ok {
		return false
	}
	return nfDeployment.Spec.Provider == provider
}

// Container Registry for SD-Core components
const (
	// Container Registry for SD-Core images
	ImageRegistry = "omecproject"

	// Container image tags for SD-Core components
	InitImageTag       = "rel-1.1.2"
	AMFImageTag        = "rel-1.6.4"
	NRFImageTag        = "rel-1.6.3"
	SMFImageTag        = "rel-2.0.3"
	AUSFImageTag       = "rel-1.6.2"
	NSSFImageTag       = "rel-1.6.2"
	PCFImageTag        = "rel-1.6.2"
	UDRImageTag        = "rel-1.6.3"
	UDMImageTag        = "rel-1.6.2"
	WebUIImageTag      = "rel-1.8.3"
	SCTPLBImageTag     = "rel-1.6.1"
	MetricFuncImageTag = "rel-1.6.1"
	UPFAdapterImageTag = "rel-2.0.2"
	UPFImageTag        = "rel-1.0.0" // Placeholder for UPF

	// Container image names for SD-Core components
	InitImage       = ImageRegistry + "/pod-init:" + InitImageTag
	AMFImage        = ImageRegistry + "/5gc-amf:" + AMFImageTag
	NRFImage        = ImageRegistry + "/5gc-nrf:" + NRFImageTag
	SMFImage        = ImageRegistry + "/5gc-smf:" + SMFImageTag
	AUSFImage       = ImageRegistry + "/5gc-ausf:" + AUSFImageTag
	NSSFImage       = ImageRegistry + "/5gc-nssf:" + NSSFImageTag
	PCFImage        = ImageRegistry + "/5gc-pcf:" + PCFImageTag
	UDRImage        = ImageRegistry + "/5gc-udr:" + UDRImageTag
	UDMImage        = ImageRegistry + "/5gc-udm:" + UDMImageTag
	WebUIImage      = ImageRegistry + "/5gc-webui:" + WebUIImageTag
	SCTPLBImage     = ImageRegistry + "/sctplb:" + SCTPLBImageTag
	MetricFuncImage = ImageRegistry + "/metricfunc:" + MetricFuncImageTag
	UPFAdapterImage = ImageRegistry + "/upfadapter:" + UPFAdapterImageTag

	// Provider names for Network Functions
	AMFProvider  = "amf.sdcore.io"
	NRFProvider  = "nrf.sdcore.io"
	SMFProvider  = "smf.sdcore.io"
	UPFProvider  = "upf.sdcore.io"
	AUSFProvider = "ausf.sdcore.io"
	NSSFProvider = "nssf.sdcore.io"
	PCFProvider  = "pcf.sdcore.io"
	UDRProvider  = "udr.sdcore.io"
	UDMProvider  = "udm.sdcore.io"
) 