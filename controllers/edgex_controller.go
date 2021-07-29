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
	"reflect"

	"github.com/go-logr/logr"
	devicev1alpha1 "github.com/lwmqwer/EdgeX/api/v1alpha1"
	unitv1alpha1 "github.com/openyurtio/yurt-app-manager-api/pkg/yurtappmanager/apis/apps/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// LabelDesiredNodePool indicates which nodepool the node want to join
	LabelEdgeXDeployment = "www.edgexfoundry.org/deployment"

	LabelEdgeXService = "www.edgexfoundry.org/service"
	// name of finalizer
	FinalizerName = "www.edgexfoundry.org/finalizer"
)

var (
	ControlledType     = &devicev1alpha1.EdgeX{}
	ControlledTypeName = reflect.TypeOf(ControlledType).Elem().Name()
	ControlledTypeGVK  = devicev1alpha1.GroupVersion.WithKind(ControlledTypeName)
)

// EdgeXReconciler reconciles a EdgeX object
type EdgeXReconciler struct {
	client.Client
	Logger logr.Logger
	Scheme *runtime.Scheme
}

var (
	CoreDeployment map[string][]devicev1alpha1.DeploymentTemplateSpec = make(map[string][]devicev1alpha1.DeploymentTemplateSpec)
	CoreServices   map[string][]devicev1alpha1.ServiceTemplateSpec    = make(map[string][]devicev1alpha1.ServiceTemplateSpec)
	CoreConfigMap  map[string]corev1.ConfigMap                        = make(map[string]corev1.ConfigMap)
)

//+kubebuilder:rbac:groups=device.openyurt.io,resources=edgexes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=device.openyurt.io,resources=edgexes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=device.openyurt.io,resources=edgexes/finalizers,verbs=update
//+kubebuilder:rbac:groups=device.openyurt.io,resources=edgexes/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps.openyurt.io,resources=uniteddeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.openyurt.io,resources=uniteddeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=configmaps;services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps/status;services/status,verbs=get;update;patch

func (r *EdgeXReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	edgex := &devicev1alpha1.EdgeX{}
	unitedDeployment := &unitv1alpha1.UnitedDeployment{}
	service := &corev1.Service{}

	if err := r.Get(ctx, req.NamespacedName, edgex); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
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
			if err := r.cleanUpRelateResources(ctx, edgex); err != nil {
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

	configmap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: CoreConfigMap[edgex.Spec.Version].Name,
		Namespace: edgex.Namespace},
		Data: make(map[string]string)}

	for k, v := range CoreConfigMap[edgex.Spec.Version].Data {
		configmap.Data[k] = v
	}

	_, err := controllerutil.CreateOrPatch(ctx, r.Client, configmap, func() error {
		return controllerutil.SetOwnerReference(edgex, configmap, r.Scheme)
	})

	if err != nil {
		return ctrl.Result{}, err
	}

	for _, desireDeployment := range CoreDeployment[edgex.Spec.Version] {
		if err := r.createOrUpdateUnitDeployment(ctx, edgex, &desireDeployment, unitedDeployment); err != nil {
			return ctrl.Result{}, err
		}

		//edgex.Status.ComponetStatus = append(edgex.Status.ComponetStatus)
	}

	for _, desireservice := range CoreServices[edgex.Spec.Version] {
		if err := r.createOrPatchService(ctx, edgex, &desireservice, service); err != nil {
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
			if err := r.createOrPatchService(ctx, edgex, &desirecomponent.Service, service); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// Update status
	edgex.Status.Initialized = true

	return ctrl.Result{}, r.Status().Update(ctx, edgex)
}

func (r *EdgeXReconciler) cleanUpRelateResources(ctx context.Context, edgex *devicev1alpha1.EdgeX) error {
	//
	// delete any external resources associated with the cronJob
	//
	// Ensure that delete implementation is idempotent and safe to invoke
	// multiple times for same object.
	ud := &unitv1alpha1.UnitedDeployment{}
	for _, desireDeployment := range CoreDeployment[edgex.Spec.Version] {

		err := r.Get(ctx, types.NamespacedName{Namespace: edgex.Namespace,
			Name: desireDeployment.Name}, ud)

		if err == nil {
			for i, pool := range ud.Spec.Topology.Pools {
				if pool.Name == edgex.Spec.PoolName {
					ud.Spec.Topology.Pools[i] = ud.Spec.Topology.Pools[len(ud.Spec.Topology.Pools)-1]
					ud.Spec.Topology.Pools = ud.Spec.Topology.Pools[:len(ud.Spec.Topology.Pools)-1]
				}
			}
			if err := r.Update(ctx, ud); err != nil {
				return err
			}
		}
	}

	for _, desirecomponent := range edgex.Spec.AdditionalComponents {
		if desirecomponent.Deployment.Name != "" {
			err := r.Get(ctx, types.NamespacedName{Namespace: edgex.Namespace,
				Name: desirecomponent.Deployment.Name}, ud)

			if err == nil {
				for i, pool := range ud.Spec.Topology.Pools {
					if pool.Name == edgex.Spec.PoolName {
						ud.Spec.Topology.Pools[i] = ud.Spec.Topology.Pools[len(ud.Spec.Topology.Pools)-1]
						ud.Spec.Topology.Pools = ud.Spec.Topology.Pools[:len(ud.Spec.Topology.Pools)-1]
					}
				}
				if err := r.Update(ctx, ud); err != nil {
					return err
				}
			}
		}
	}

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

func (r *EdgeXReconciler) createOrPatchService(ctx context.Context,
	edgex *devicev1alpha1.EdgeX,
	ds *devicev1alpha1.ServiceTemplateSpec,
	s *corev1.Service) error {

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

	_, err := controllerutil.CreateOrPatch(ctx, r.Client, s, func() error {
		return controllerutil.SetOwnerReference(edgex, s, r.Scheme)
	})

	return err
}

// SetupWithManager sets up the controller with the Manager.
func (r *EdgeXReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(ControlledType).
		Watches(
			&source.Kind{Type: &unitv1alpha1.UnitedDeployment{}},
			&handler.EnqueueRequestForOwner{OwnerType: ControlledType, IsController: false},
		).
		Watches(
			&source.Kind{Type: &corev1.Service{}},
			&handler.EnqueueRequestForOwner{OwnerType: ControlledType, IsController: false},
		).
		Watches(
			&source.Kind{Type: &corev1.ConfigMap{}},
			&handler.EnqueueRequestForOwner{OwnerType: ControlledType, IsController: false},
		).
		Complete(r)
}
