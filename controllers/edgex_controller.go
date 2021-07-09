/*
Copyright 2021.

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
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	devicev1alpha1 "github.com/lwmqwer/EdgeX/api/v1alpha1"
	unitv1alpha1 "github.com/openyurtio/yurt-app-manager/pkg/yurtappmanager/apis/apps/v1alpha1"
)

const (
	// LabelDesiredNodePool indicates which nodepool the node want to join
	LabelEdgeXDeployment = "www.edgexfoundry.org/deployment"

	LabelEdgeXService = "www.edgexfoundry.org/service"
)

// EdgeXReconciler reconciles a EdgeX object
type EdgeXReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var (
	coreDeployment map[string][]devicev1alpha1.DeploymentTemplateSpec
	coreServices   map[string][]devicev1alpha1.ServiceTemplateSpec
)

//+kubebuilder:rbac:groups=device.openyurt.io,resources=edgexes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=device.openyurt.io,resources=edgexes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=device.openyurt.io,resources=edgexes/finalizers,verbs=update

func (r *EdgeXReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// your logic here
	var edgex *devicev1alpha1.EdgeX
	if err := r.Get(ctx, req.NamespacedName, edgex); err != nil {
		return ctrl.Result{}, err
	}

	// Get desire unit deployment and service
	desireDeployments := coreDeployment[edgex.Spec.Version]
	desireServices := coreServices[edgex.Spec.Version]

	for _, c := range edgex.Spec.AdditionalComponents {
		if c.Deployment != nil {
			desireDeployments = append(desireDeployments, c.Deployment)
		}
		if c.Service != nil {
			desireServices = append(desireServices, c.Service)
		}
	}

	// List current unit deployment and service
	var unitedDeploymentList *unitv1alpha1.UnitedDeploymentList
	if err := r.List(ctx, unitedDeploymentList, client.MatchingLabels(map[string]string{LabelEdgeXDeployment: "core"})); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var serviceList *corev1.ServiceList
	if err := r.List(ctx, serviceList, client.MatchingLabels(map[string]string{LabelEdgeXService: "core"})); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Create unit deployment and service

	for _, s := range desireServices {
		var ud unitv1alpha1.UnitedDeployment
		us.ss = s.
			r.Create(ctx, ud, opt)
	}
	for _, s := range desireDeployments {
		r.Create(ctx, s, opt)
	}

	// Update status
	edgex.Status.Initialized = true
	edgex.Status.ComponetStatus = append(edgex.Status.ComponetStatus)

	return ctrl.Result{}, r.Status().Update(ctx, edgex)
}

// SetupWithManager sets up the controller with the Manager.
func (r *EdgeXReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&devicev1alpha1.EdgeX{}).
		Complete(r)
}
