package controllers

import (
	"fmt"
	"strings"

	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
)

// GetNamespacedName returns a namespaced name for a deployment
func GetNamespacedName(nfDeployment *nephiov1alpha1.NFDeployment, suffix string) string {
	return fmt.Sprintf("%s-%s", nfDeployment.Name, suffix)
}

// IsProviderSDCoreUPF returns true if the provider is sdcore UPF
func IsProviderSDCoreUPF(provider string) bool {
	return strings.EqualFold(provider, "upf.sdcore.io")
}
