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