package upf

import (
	"context"
	"time"

	"github.com/RohitRathore1/sdcore-operator/controllers"
	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciles a UPF NFDeployment resource
type UPFDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *UPFDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("UPFDeployment", req.NamespacedName)

	// Get the NFDeployment resource
	nfDeployment := new(nephiov1alpha1.NFDeployment)
	err := r.Client.Get(ctx, req.NamespacedName, nfDeployment)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("NFDeployment resource not found, ignoring because object must be deleted")
			return reconcile.Result{}, nil
		}
		log.Error(err, "Failed to get NFDeployment")
		return reconcile.Result{}, err
	}

	// Verify that this is a UPF deployment
	if !controllers.IsProviderSDCoreUPF(nfDeployment.Spec.Provider) {
		log.Info("NFDeployment is not for SDCore UPF, ignoring",
			"Provider", nfDeployment.Spec.Provider)
		return reconcile.Result{}, nil
	}

	// Create or update ConfigMap
	if err := r.reconcileConfigMap(ctx, nfDeployment); err != nil {
		log.Error(err, "Failed to reconcile ConfigMap")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	// Create or update Deployment
	if err := r.reconcileDeployment(ctx, nfDeployment); err != nil {
		log.Error(err, "Failed to reconcile Deployment")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	// Create or update Service
	if err := r.reconcileService(ctx, nfDeployment); err != nil {
		log.Error(err, "Failed to reconcile Service")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	// Update status
	upfDeployment, err := r.getDeployment(ctx, nfDeployment)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Error(err, "Failed to get Deployment")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
		// Deployment not found yet, requeue
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	status, changed := createNfDeploymentStatus(upfDeployment, nfDeployment)
	if changed {
		nfDeployment.Status = status
		if err := r.Status().Update(ctx, nfDeployment); err != nil {
			log.Error(err, "Failed to update NFDeployment status")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
	}

	return ctrl.Result{}, nil
}

// getDeployment gets the UPF deployment for the NFDeployment
func (r *UPFDeploymentReconciler) getDeployment(ctx context.Context, nfDeployment *nephiov1alpha1.NFDeployment) (*appsv1.Deployment, error) {
	deployment := new(appsv1.Deployment)
	err := r.Client.Get(ctx, types.NamespacedName{
		Namespace: nfDeployment.Namespace,
		Name:      controllers.GetNamespacedName(nfDeployment, "upf"),
	}, deployment)
	return deployment, err
}
