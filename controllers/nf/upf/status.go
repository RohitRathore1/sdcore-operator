package upf

import (
	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createNfDeploymentStatus(deployment *appsv1.Deployment, nfDeployment *nephiov1alpha1.NFDeployment) (nephiov1alpha1.NFDeploymentStatus, bool) {
	observedGeneration := int32(deployment.ObjectMeta.Generation)
	status := nfDeployment.Status
	changed := false

	if status.ObservedGeneration != observedGeneration {
		status.ObservedGeneration = observedGeneration
		changed = true
	}

	if status.Conditions == nil {
		status.Conditions = []metav1.Condition{}
	}

	// Check for Available condition
	availableCondition := metav1.Condition{
		Type:               string(nephiov1alpha1.Available),
		Status:             metav1.ConditionFalse,
		Reason:             "DeploymentUnavailable",
		Message:            "Deployment is not available",
		LastTransitionTime: metav1.Now(),
	}

	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentAvailable && condition.Status == apiv1.ConditionTrue {
			availableCondition.Status = metav1.ConditionTrue
			availableCondition.Reason = "DeploymentAvailable"
			availableCondition.Message = "Deployment is available"
			break
		}
	}

	// Update the Available condition if it changed
	availableChanged := updateCondition(&status.Conditions, availableCondition)
	changed = changed || availableChanged

	// Check for Ready condition
	readyCondition := metav1.Condition{
		Type:               string(nephiov1alpha1.Ready),
		Status:             metav1.ConditionFalse,
		Reason:             "NotReady",
		Message:            "UPF is not ready",
		LastTransitionTime: metav1.Now(),
	}

	if availableCondition.Status == metav1.ConditionTrue &&
		deployment.Status.ReadyReplicas == deployment.Status.Replicas {
		readyCondition.Status = metav1.ConditionTrue
		readyCondition.Reason = "Ready"
		readyCondition.Message = "UPF is ready"
	}

	// Update the Ready condition if it changed
	readyChanged := updateCondition(&status.Conditions, readyCondition)
	changed = changed || readyChanged

	return status, changed
}

// updateCondition updates the condition in the list of conditions or adds it if not present
// Returns true if the condition was changed
func updateCondition(conditions *[]metav1.Condition, condition metav1.Condition) bool {
	for i, c := range *conditions {
		if c.Type == condition.Type {
			if c.Status != condition.Status || c.Reason != condition.Reason || c.Message != condition.Message {
				(*conditions)[i] = condition
				return true
			}
			return false
		}
	}

	// Condition not found, add it
	*conditions = append(*conditions, condition)
	return true
}
