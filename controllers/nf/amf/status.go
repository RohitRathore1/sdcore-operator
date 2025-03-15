package amf

import (
	"context"

	"github.com/RohitRathore1/sdcore-operator/controllers"
	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// updateStatus updates the status of the NFDeployment
func updateStatus(ctx context.Context, c client.Client, nfDeployment *nephiov1alpha1.NFDeployment) error {
	log := log.FromContext(ctx).WithValues("AMFStatus", nfDeployment.Name)

	// Get the deployment
	deploymentName := controllers.GetNamespacedName(nfDeployment, "amf")
	deployment := &appsv1.Deployment{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: nfDeployment.Namespace, Name: deploymentName}, deployment); err != nil {
		log.Error(err, "Failed to get deployment for status update")
		return err
	}

	// Update the status
	nfDeployment.Status.ObservedGeneration = int32(nfDeployment.Generation)

	// Update conditions
	ready := deployment.Status.ReadyReplicas > 0
	conditions := []metav1.Condition{}

	// Add a condition for the deployment
	if ready {
		conditions = append(conditions, metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             "DeploymentReady",
			Message:            "AMF deployment is ready",
		})
	} else {
		conditions = append(conditions, metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "DeploymentNotReady",
			Message:            "AMF deployment is not ready",
		})
	}

	nfDeployment.Status.Conditions = conditions

	// Update the NFDeployment status
	if err := c.Status().Update(ctx, nfDeployment); err != nil {
		log.Error(err, "Failed to update NFDeployment status")
		return err
	}

	return nil
}
