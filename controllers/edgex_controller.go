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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	devicev1alpha1 "github.com/lwmqwer/EdgeX/api/v1alpha1"
	unitv1alpha1 "github.com/openyurtio/yurt-app-manager-api/pkg/yurtappmanager/apis/apps/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	udOwnerKey = ".metadata.controller"
	// LabelDesiredNodePool indicates which nodepool the node want to join
	LabelEdgeXDeployment = "www.edgexfoundry.org/deployment"

	LabelEdgeXService = "www.edgexfoundry.org/service"

	// name of finalizer
	FinalizerName = "www.edgexfoundry.org/finalizer"
)

// EdgeXReconciler reconciles a EdgeX object
type EdgeXReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var (
	CoreDeployment map[string][]devicev1alpha1.DeploymentTemplateSpec = make(map[string][]devicev1alpha1.DeploymentTemplateSpec)
	CoreServices   map[string][]devicev1alpha1.ServiceTemplateSpec    = make(map[string][]devicev1alpha1.ServiceTemplateSpec)
)

//+kubebuilder:rbac:groups=device.openyurt.io,resources=edgexes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=device.openyurt.io,resources=edgexes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=device.openyurt.io,resources=edgexes/finalizers,verbs=update

func (r *EdgeXReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	edgex := &devicev1alpha1.EdgeX{}
	unitedDeployment := &unitv1alpha1.UnitedDeployment{}
	service := &corev1.Service{}

	if err := r.Get(ctx, req.NamespacedName, edgex); err != nil {
		return ctrl.Result{}, err
	}

	// examine DeletionTimestamp to determine if object is under deletion
	if edgex.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !containsString(edgex.GetFinalizers(), FinalizerName) {
			controllerutil.AddFinalizer(edgex, FinalizerName)
			if err := r.Update(ctx, edgex); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if containsString(edgex.GetFinalizers(), FinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.cleanUpRelateResources(edgex); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(edgex, FinalizerName)
			if err := r.Update(ctx, edgex); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	for _, desireDeployment := range CoreDeployment[edgex.Spec.Version] {
		if err := r.createOrUpdateUnitDeployment(ctx, edgex, &desireDeployment, unitedDeployment); err != nil {
			return ctrl.Result{}, err
		}

		//edgex.Status.ComponetStatus = append(edgex.Status.ComponetStatus)
	}

	for _, desireservice := range CoreServices[edgex.Spec.Version] {
		if err := r.createOrUpdateService(ctx, edgex, &desireservice, service); err != nil {
			return ctrl.Result{}, err
		}
	}

	for _, desirecomponent := range edgex.Spec.AdditionalComponents {
		if desirecomponent.Deployment.Name != "" {
			if err := r.createOrUpdateUnitDeployment(ctx, edgex, &desirecomponent.Deployment, unitedDeployment); err != nil {
				return ctrl.Result{}, err
			}
		}

		if desirecomponent.Service.Name != "" {
			if err := r.createOrUpdateService(ctx, edgex, &desirecomponent.Service, service); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// Update status
	edgex.Status.Initialized = true

	return ctrl.Result{}, r.Status().Update(ctx, edgex)
}

func (r *EdgeXReconciler) cleanUpRelateResources(edgex *devicev1alpha1.EdgeX) error {
	//
	// delete any external resources associated with the cronJob
	//
	// Ensure that delete implementation is idempotent and safe to invoke
	// multiple times for same object.

	return nil
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func (r *EdgeXReconciler) createOrUpdateUnitDeployment(ctx context.Context,
	edgex *devicev1alpha1.EdgeX,
	dd *devicev1alpha1.DeploymentTemplateSpec,
	ud *unitv1alpha1.UnitedDeployment) error {
	if err := r.Get(ctx,
		types.NamespacedName{Namespace: edgex.Namespace,
			Name: dd.Name}, ud); err != nil {
		ud = &unitv1alpha1.UnitedDeployment{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				Name:        dd.Name,
				Namespace:   edgex.Namespace,
			},
			Spec: unitv1alpha1.UnitedDeploymentSpec{
				Selector: dd.Spec.Selector.DeepCopy(),
				WorkloadTemplate: unitv1alpha1.WorkloadTemplate{
					DeploymentTemplate: &unitv1alpha1.DeploymentTemplateSpec{ObjectMeta: *dd.Spec.Template.ObjectMeta.DeepCopy(),
						Spec: *dd.Spec.DeepCopy()},
				},
			},
		}
		pool := unitv1alpha1.Pool{Name: edgex.Spec.PoolName,
			Replicas: pointer.Int32Ptr(1)}
		pool.NodeSelectorTerm.MatchExpressions = append(pool.NodeSelectorTerm.MatchExpressions,
			corev1.NodeSelectorRequirement{Key: unitv1alpha1.LabelCurrentNodePool,
				Operator: corev1.NodeSelectorOpIn,
				Values:   []string{edgex.Spec.PoolName}})
		ud.Spec.Topology.Pools = append(ud.Spec.Topology.Pools, pool)
		if err := controllerutil.SetOwnerReference(edgex, ud, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(ctx, ud); err != nil {
			return err
		}
	} else {
		for _, pool := range ud.Spec.Topology.Pools {
			if pool.Name == edgex.Spec.PoolName {
				return nil
			}
		}
		pool := unitv1alpha1.Pool{Name: edgex.Spec.PoolName,
			Replicas: pointer.Int32Ptr(1)}
		pool.NodeSelectorTerm.MatchExpressions = append(pool.NodeSelectorTerm.MatchExpressions,
			corev1.NodeSelectorRequirement{Key: unitv1alpha1.LabelCurrentNodePool,
				Operator: corev1.NodeSelectorOpIn,
				Values:   []string{edgex.Spec.PoolName}})
		ud.Spec.Topology.Pools = append(ud.Spec.Topology.Pools, pool)
		if err := controllerutil.SetOwnerReference(edgex, ud, r.Scheme); err != nil {
			return err
		}
		if err := r.Update(ctx, ud); err != nil {
			return err
		}
	}
	return nil
}

func (r *EdgeXReconciler) createOrUpdateService(ctx context.Context,
	edgex *devicev1alpha1.EdgeX,
	ds *devicev1alpha1.ServiceTemplateSpec,
	s *corev1.Service) error {
	if err := r.Get(ctx,
		types.NamespacedName{Namespace: edgex.Namespace,
			Name: ds.Name}, s); err != nil {
		s = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				Name:        ds.Name,
				Namespace:   edgex.Namespace,
			},
			Spec: *ds.Spec.DeepCopy(),
		}
		for k, v := range ds.Annotations {
			s.Annotations[k] = v
		}
		for k, v := range ds.Labels {
			s.Labels[k] = v
		}
		if err := controllerutil.SetOwnerReference(edgex, s, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(ctx, s); err != nil {
			return err
		}
	} else {
		if err := controllerutil.SetOwnerReference(edgex, s, r.Scheme); err != nil {
			return err
		}
		if err := r.Update(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EdgeXReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &unitv1alpha1.UnitedDeployment{}, udOwnerKey, func(rawObj client.Object) []string {
		// grab the uniteddeployment object, extract the owner...
		ud := rawObj.(*unitv1alpha1.UnitedDeployment)
		owner := metav1.GetControllerOf(ud)
		if owner == nil {
			return nil
		}
		// ...make sure it's a EdgeX...
		if owner.APIVersion != devicev1alpha1.GroupVersion.String() || owner.Kind != "EdgeX" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&devicev1alpha1.EdgeX{}).
		//Owns(&unitv1alpha1.UnitedDeployment{}).
		Complete(r)
}
