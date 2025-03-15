package amf

import (
	"context"
	"time"

	"github.com/RohitRathore1/sdcore-operator/controllers"
	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// AMFDeploymentReconciler reconciles a NFDeployment resource for AMF
type AMFDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile handles the reconciliation loop for the AMF NFDeployment
func (r *AMFDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("AMFReconciler", req.NamespacedName)
	log.Info("Reconciling AMF NFDeployment")

	// Fetch the NFDeployment instance
	nfDeployment := &nephiov1alpha1.NFDeployment{}
	if err := r.Get(ctx, req.NamespacedName, nfDeployment); err != nil {
		log.Error(err, "Unable to fetch NFDeployment")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Verify this is an AMF deployment by checking the provider
	if !controllers.IsProviderSDCore(nfDeployment.Spec.Provider) {
		log.Info("NFDeployment is not for SDCore, ignoring", "provider", nfDeployment.Spec.Provider)
		return ctrl.Result{}, nil
	}

	// Reconcile ConfigMap
	configMapChanged, err := reconcileConfigMap(ctx, r.Client, nfDeployment)
	if err != nil {
		log.Error(err, "Failed to reconcile ConfigMap")
		return ctrl.Result{}, err
	}

	// Reconcile Deployment
	deploymentChanged, err := reconcileDeployment(ctx, r.Client, r.Scheme, nfDeployment)
	if err != nil {
		log.Error(err, "Failed to reconcile Deployment")
		return ctrl.Result{}, err
	}

	// Reconcile Service
	serviceChanged, err := reconcileService(ctx, r.Client, r.Scheme, nfDeployment)
	if err != nil {
		log.Error(err, "Failed to reconcile Service")
		return ctrl.Result{}, err
	}

	// Update status
	if err := updateStatus(ctx, r.Client, nfDeployment); err != nil {
		log.Error(err, "Failed to update NFDeployment status")
		return ctrl.Result{}, err
	}

	// If any resource changed, requeue after a short delay to allow resources to stabilize
	if configMapChanged || deploymentChanged || serviceChanged {
		log.Info("Resources changed, requeuing")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	log.Info("AMF NFDeployment reconciled successfully")
	return ctrl.Result{}, nil
}

// getDeployment gets the AMF deployment associated with the NFDeployment
func getDeployment(ctx context.Context, c client.Client, nfDeployment *nephiov1alpha1.NFDeployment) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	deploymentName := controllers.GetNamespacedName(nfDeployment, "amf")
	err := c.Get(ctx, client.ObjectKey{Namespace: nfDeployment.Namespace, Name: deploymentName}, deployment)
	return deployment, err
}
