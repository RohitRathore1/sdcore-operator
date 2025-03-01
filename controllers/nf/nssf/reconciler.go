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
	"context"
	"time"

	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	"github.com/nephio-project/sdcore/controllers"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciles a NSSF NFDeployment resource
type NSSFDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NSSFDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("NFDeployment", req.NamespacedName, "NF", "NSSF")

	nfDeployment := new(nephiov1alpha1.NFDeployment)
	err := r.Client.Get(ctx, req.NamespacedName, nfDeployment)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("NSSF NFDeployment resource not found, ignoring because object must be deleted")
			return reconcile.Result{}, nil
		}
		log.Error(err, "Failed to get NSSF NFDeployment")
		return reconcile.Result{}, err
	}

	namespace := nfDeployment.Namespace

	configMapFound := false
	configMapName := nfDeployment.Name
	var configMapVersion string
	currentConfigMap := new(apiv1.ConfigMap)
	if err := r.Client.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: namespace}, currentConfigMap); err == nil {
		configMapFound = true
		configMapVersion = currentConfigMap.ResourceVersion
	}

	deploymentFound := false
	deploymentName := nfDeployment.Name
	currentDeployment := new(appsv1.Deployment)
	if err := r.Client.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: namespace}, currentDeployment); err == nil {
		deploymentFound = true
	}

	// If deployment exists, check if we need to update the status
	if deploymentFound {
		deployment := currentDeployment.DeepCopy()

		// TODO: implement status update logic
		
		// If configMap was updated, we should update the deployment to trigger a rolling update
		if currentDeployment.Spec.Template.Annotations[controllers.ConfigMapVersionAnnotation] != configMapVersion {
			log.Info("ConfigMap has been updated, rolling Deployment pods", "Deployment.namespace", currentDeployment.Namespace, "Deployment.name", currentDeployment.Name)
			currentDeployment.Spec.Template.Annotations[controllers.ConfigMapVersionAnnotation] = configMapVersion

			if err := r.Update(ctx, currentDeployment); err != nil {
				log.Error(err, "Failed to update Deployment", "Deployment.namespace", currentDeployment.Namespace, "Deployment.name", currentDeployment.Name)
				return reconcile.Result{}, err
			}
			return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
		}

		return reconcile.Result{}, nil
	}

	// Create or update configMap if needed
	if !configMapFound {
		configMap, err := createConfigMap(log, nfDeployment)
		if err != nil {
			log.Error(err, "Failed to create ConfigMap")
			return reconcile.Result{}, err
		}

		if err := r.Create(ctx, configMap); err != nil {
			log.Error(err, "Failed to create ConfigMap", "ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
			return reconcile.Result{}, err
		}
		configMapVersion = configMap.ResourceVersion
		log.Info("Created ConfigMap", "ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
	}

	// Create deployment if it doesn't exist
	if !deploymentFound {
		deployment, err := createDeployment(log, configMapVersion, nfDeployment)
		if err != nil {
			log.Error(err, "Failed to create Deployment")
			return reconcile.Result{}, err
		}

		if err := ctrl.SetControllerReference(nfDeployment, deployment, r.Scheme); err != nil {
			log.Error(err, "Failed to set controller reference for Deployment")
			return reconcile.Result{}, err
		}

		if err := r.Create(ctx, deployment); err != nil {
			log.Error(err, "Failed to create Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
			return reconcile.Result{}, err
		}
		log.Info("Created Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
	}

	// Create service if needed
	serviceName := nfDeployment.Name
	currentService := new(apiv1.Service)
	serviceFound := false
	if err := r.Client.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, currentService); err == nil {
		serviceFound = true
	}

	if !serviceFound {
		service, err := createService(log, nfDeployment)
		if err != nil {
			log.Error(err, "Failed to create Service")
			return reconcile.Result{}, err
		}

		if err := ctrl.SetControllerReference(nfDeployment, service, r.Scheme); err != nil {
			log.Error(err, "Failed to set controller reference for Service")
			return reconcile.Result{}, err
		}

		if err := r.Create(ctx, service); err != nil {
			log.Error(err, "Failed to create Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
			return reconcile.Result{}, err
		}
		log.Info("Created Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
	}

	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NSSFDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Watch NFDeployment with provider nssf.sdcore.io
		For(&nephiov1alpha1.NFDeployment{}).
		// Filter NFDeployments with provider nssf.sdcore.io
		WithEventFilter(controllers.ProviderFilter("nssf.sdcore.io")).
		Complete(r)
} 