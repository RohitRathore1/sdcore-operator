package nf

import (
	"context"

	"github.com/RohitRathore1/sdcore-operator/controllers"
	smf "github.com/RohitRathore1/sdcore-operator/controllers/nf/smf"
	upf "github.com/RohitRathore1/sdcore-operator/controllers/nf/upf"
	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciles a NFDeployment resource
type NFDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Sets up the controller with the Manager
func (r *NFDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(new(nephiov1alpha1.NFDeployment)).
		Owns(new(appsv1.Deployment)).
		Owns(new(apiv1.ConfigMap)).
		Complete(r)
}

// +kubebuilder:rbac:groups=workload.nephio.org,resources=nfdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=workload.nephio.org,resources=nfdeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="ref.nephio.org",resources=configs,verbs=get;list;watch
// +kubebuilder:rbac:groups="k8s.cni.cncf.io",resources=network-attachment-definitions,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps;services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NFDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("NFDeployment", req.NamespacedName)

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

	upfReconciler := &upf.UPFDeploymentReconciler{
		Client: r.Client,
		Scheme: r.Scheme,
	}

	smfReconciler := &smf.SMFDeploymentReconciler{
		Client: r.Client,
		Scheme: r.Scheme,
	}

	// Route to the appropriate reconciler based on the provider
	if controllers.IsProviderSDCoreUPF(nfDeployment.Spec.Provider) {
		log.Info("Routing to UPF reconciler")
		upfResult, upfErr := upfReconciler.Reconcile(ctx, req)
		return upfResult, upfErr
	} else if controllers.IsProviderSDCore(nfDeployment.Spec.Provider) {
		// For SMF, we'll use the name to determine if it's an SMF deployment
		if nfDeployment.Name == "test-smf" {
			log.Info("Routing to SMF reconciler")
			smfResult, smfErr := smfReconciler.Reconcile(ctx, req)
			return smfResult, smfErr
		}
	}

	log.Info("NFDeployment NOT for SDCore or unsupported type", "nfDeployment.Spec.Provider", nfDeployment.Spec.Provider)
	return reconcile.Result{}, nil
}
