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

package nf

import (
	"context"

	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	amf "github.com/RohitRathore1/sdcore-operator/controllers/nf/amf"
	smf "github.com/RohitRathore1/sdcore-operator/controllers/nf/smf"
	upf "github.com/RohitRathore1/sdcore-operator/controllers/nf/upf"
	nrf "github.com/RohitRathore1/sdcore-operator/controllers/nf/nrf"
	ausf "github.com/RohitRathore1/sdcore-operator/controllers/nf/ausf"
	udm "github.com/RohitRathore1/sdcore-operator/controllers/nf/udm"
	udr "github.com/RohitRathore1/sdcore-operator/controllers/nf/udr"
	pcf "github.com/RohitRathore1/sdcore-operator/controllers/nf/pcf"
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
	amfReconciler := &amf.AMFDeploymentReconciler{
		Client: r.Client,
		Scheme: r.Scheme,
	}
	smfReconciler := &smf.SMFDeploymentReconciler{
		Client: r.Client,
		Scheme: r.Scheme,
	}
	nrfReconciler := &nrf.NRFDeploymentReconciler{
		Client: r.Client,
		Scheme: r.Scheme,
	}
	ausfReconciler := &ausf.AUSFDeploymentReconciler{
		Client: r.Client,
		Scheme: r.Scheme,
	}
	udmReconciler := &udm.UDMDeploymentReconciler{
		Client: r.Client,
		Scheme: r.Scheme,
	}
	udrReconciler := &udr.UDRDeploymentReconciler{
		Client: r.Client,
		Scheme: r.Scheme,
	}
	pcfReconciler := &pcf.PCFDeploymentReconciler{
		Client: r.Client,
		Scheme: r.Scheme,
	}

	switch nfDeployment.Spec.Provider {
	case "upf.sdcore.io":
		upfresult, _ := upfReconciler.Reconcile(ctx, req)
		return upfresult, nil
	case "smf.sdcore.io":
		smfresult, _ := smfReconciler.Reconcile(ctx, req)
		return smfresult, nil
	case "amf.sdcore.io":
		amfresult, _ := amfReconciler.Reconcile(ctx, req)
		return amfresult, nil
	case "nrf.sdcore.io":
		nrfresult, _ := nrfReconciler.Reconcile(ctx, req)
		return nrfresult, nil
	case "ausf.sdcore.io":
		ausfresult, _ := ausfReconciler.Reconcile(ctx, req)
		return ausfresult, nil
	case "udm.sdcore.io":
		udmresult, _ := udmReconciler.Reconcile(ctx, req)
		return udmresult, nil
	case "udr.sdcore.io":
		udrresult, _ := udrReconciler.Reconcile(ctx, req)
		return udrresult, nil
	case "pcf.sdcore.io":
		pcfresult, _ := pcfReconciler.Reconcile(ctx, req)
		return pcfresult, nil
	default:
		log.Info("NFDeployment NOT for sdcore", "nfDeployment.Spec.Provider", nfDeployment.Spec.Provider)
		return reconcile.Result{}, nil
	}
} 