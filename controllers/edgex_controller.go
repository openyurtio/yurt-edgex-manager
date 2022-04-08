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
	"time"

	devicev1alpha1 "github.com/lwmqwer/EdgeX/api/v1alpha1"
	unitv1alpha1 "github.com/openyurtio/yurt-app-manager-api/pkg/yurtappmanager/apis/apps/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
)

const (
	LabelConfigmap  = "Configmap"
	LabelService    = "Service"
	LabelDeployment = "Deployment"

	AnnotationServiceTopologyKey           = "openyurt.io/topologyKeys"
	AnnotationServiceTopologyValueNodePool = "openyurt.io/nodepool"
)

var (
	ControlledType     = &devicev1alpha1.EdgeX{}
	ControlledTypeName = reflect.TypeOf(ControlledType).Elem().Name()
	ControlledTypeGVK  = devicev1alpha1.GroupVersion.WithKind(ControlledTypeName)
)

// EdgeXReconciler reconciles a EdgeX object
type EdgeXReconciler struct {
	client.Client
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

func (r *EdgeXReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)

	edgex := &devicev1alpha1.EdgeX{}
	if err := r.Get(ctx, req.NamespacedName, edgex); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Create the patch helper.
	patchHelper, err := patch.NewHelper(edgex, r.Client)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(
			err,
			"failed to init patch helper for %s %s/%s",
			edgex.GroupVersionKind(),
			edgex.Namespace,
			edgex.Name)
	}

	// Always issue a patch when exiting this function so changes to the
	// resource are patched back to the API server.
	defer func() {
		// always update the readyCondition.
		conditions.SetSummary(edgex,
			conditions.WithConditions(
				devicev1alpha1.ConfigmapAvailableCondition,
				devicev1alpha1.DeploymentAvailableCondition,
				devicev1alpha1.ServiceAvailableCondition,
			),
		)

		if err := patchHelper.Patch(ctx, edgex); err != nil {
			reterr = kerrors.NewAggregate([]error{reterr, err})
		}

		if reterr != nil {
			logger.Error(reterr, "reconcile failed", "edgex", edgex.Namespace+"/"+edgex.Name)
		}
	}()

	// Handle deleted edgex
	if !edgex.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, edgex)
	}

	// Handle non-deleted edgex
	return r.reconcileNormal(ctx, edgex)
}

func (r *EdgeXReconciler) reconcileDelete(ctx context.Context, edgex *devicev1alpha1.EdgeX) (ctrl.Result, error) {

	ud := &unitv1alpha1.UnitedDeployment{}
	desiredeployments := append(CoreDeployment[edgex.Spec.Version], edgex.Spec.AdditionalDeployment...)
	for _, dd := range desiredeployments {

		if err := r.Get(
			ctx,
			types.NamespacedName{Namespace: edgex.Namespace, Name: dd.Name},
			ud); err != nil {
			continue
		}

		for i, pool := range ud.Spec.Topology.Pools {
			if pool.Name == edgex.Spec.PoolName {
				ud.Spec.Topology.Pools[i] = ud.Spec.Topology.Pools[len(ud.Spec.Topology.Pools)-1]
				ud.Spec.Topology.Pools = ud.Spec.Topology.Pools[:len(ud.Spec.Topology.Pools)-1]
			}
		}
		if err := r.Update(ctx, ud); err != nil {
			return ctrl.Result{}, err
		}
	}

	controllerutil.RemoveFinalizer(edgex, devicev1alpha1.EdgexFinalizer)

	return ctrl.Result{}, nil
}

func (r *EdgeXReconciler) reconcileNormal(ctx context.Context, edgex *devicev1alpha1.EdgeX) (ctrl.Result, error) {
	controllerutil.AddFinalizer(edgex, devicev1alpha1.EdgexFinalizer)

	edgex.Status.Initialized = true

	if ok, err := r.reconcileConfigmap(ctx, edgex); !ok {
		if err != nil {
			conditions.MarkFalse(edgex, devicev1alpha1.ConfigmapAvailableCondition, devicev1alpha1.ConfigmapProvisioningFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
			return ctrl.Result{}, errors.Wrapf(err,
				"unexpected error while reconciling configmap for %s", edgex.Namespace+"/"+edgex.Name)
		}
		conditions.MarkFalse(edgex, devicev1alpha1.ConfigmapAvailableCondition, devicev1alpha1.ConfigmapProvisioningReason, clusterv1.ConditionSeverityInfo, "")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	conditions.MarkTrue(edgex, devicev1alpha1.ConfigmapAvailableCondition)

	if ok, err := r.reconcileService(ctx, edgex); !ok {
		if err != nil {
			conditions.MarkFalse(edgex, devicev1alpha1.ServiceAvailableCondition, devicev1alpha1.ServiceProvisioningFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
			return ctrl.Result{}, errors.Wrapf(err,
				"unexpected error while reconciling Service for %s", edgex.Namespace+"/"+edgex.Name)
		}
		conditions.MarkFalse(edgex, devicev1alpha1.ServiceAvailableCondition, devicev1alpha1.ServiceProvisioningReason, clusterv1.ConditionSeverityInfo, "")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	conditions.MarkTrue(edgex, devicev1alpha1.ServiceAvailableCondition)

	if ok, err := r.reconcileDeployment(ctx, edgex); !ok {
		if err != nil {
			conditions.MarkFalse(edgex, devicev1alpha1.DeploymentAvailableCondition, devicev1alpha1.DeploymentProvisioningFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
			return ctrl.Result{}, errors.Wrapf(err,
				"unexpected error while reconciling deployment for %s", edgex.Namespace+"/"+edgex.Name)
		}
		conditions.MarkFalse(edgex, devicev1alpha1.DeploymentAvailableCondition, devicev1alpha1.DeploymentProvisioningReason, clusterv1.ConditionSeverityInfo, "")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	conditions.MarkTrue(edgex, devicev1alpha1.DeploymentAvailableCondition)

	edgex.Status.Ready = true

	return ctrl.Result{}, nil
}

func (r *EdgeXReconciler) removeOwner(ctx context.Context, edgex *devicev1alpha1.EdgeX, obj client.Object) error {
	owners := obj.GetOwnerReferences()
	for i, owner := range owners {
		if owner.UID == edgex.UID {
			owners[i] = owners[len(owners)-1]
			owners = owners[:len(owners)-1]

			if len(owners) == 0 {
				return r.Delete(ctx, obj)
			} else {
				obj.SetOwnerReferences(owners)
				return r.Update(ctx, obj)
			}
		}
	}
	return nil
}

func (r *EdgeXReconciler) reconcileConfigmap(ctx context.Context, edgex *devicev1alpha1.EdgeX) (bool, error) {
	configmap := &corev1.ConfigMap{}

	if _, ok := CoreConfigMap[edgex.Spec.Version]; ok {

		configmap.ObjectMeta = metav1.ObjectMeta{
			Labels:    make(map[string]string),
			Name:      CoreConfigMap[edgex.Spec.Version].Name,
			Namespace: edgex.Namespace,
		}
		configmap.Data = make(map[string]string)
		configmap.Labels[devicev1alpha1.LabelEdgeXGenerate] = LabelConfigmap

		for k, v := range CoreConfigMap[edgex.Spec.Version].Data {
			configmap.Data[k] = v
		}

		_, err := controllerutil.CreateOrUpdate(ctx, r.Client, configmap, func() error {
			return controllerutil.SetOwnerReference(edgex, configmap, r.Scheme)
		})

		if err != nil {
			return false, err
		}
	}

	configmaplist := &corev1.ConfigMapList{}
	if err := r.List(ctx, configmaplist, client.MatchingLabels{devicev1alpha1.LabelEdgeXGenerate: LabelConfigmap}); err == nil {
		for _, c := range configmaplist.Items {
			if c.Name == configmap.Name {
				continue
			}
			r.removeOwner(ctx, edgex, &c)
		}
	}

	return true, nil
}

func (r *EdgeXReconciler) reconcileService(ctx context.Context, edgex *devicev1alpha1.EdgeX) (bool, error) {
	desireservices := append(CoreServices[edgex.Spec.Version], edgex.Spec.AdditionalService...)
	needservices := make(map[string]bool)
	var readyservice int32

	defer func() {
		edgex.Status.ServiceReplicas = int32(len(desireservices))
		edgex.Status.ServiceReadyReplicas = readyservice
	}()

	for _, desireservice := range desireservices {
		needservices[desireservice.Name] = true
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				Name:        desireservice.Name,
				Namespace:   edgex.Namespace,
			},
			Spec: *desireservice.Spec.DeepCopy(),
		}
		for k, v := range desireservice.Annotations {
			service.Annotations[k] = v
		}
		for k, v := range desireservice.Labels {
			service.Labels[k] = v
		}
		service.Labels[devicev1alpha1.LabelEdgeXGenerate] = LabelService
		service.Annotations[AnnotationServiceTopologyKey] = AnnotationServiceTopologyValueNodePool
		_, err := controllerutil.CreateOrUpdate(
			ctx,
			r.Client,
			service,
			func() error {
				return controllerutil.SetOwnerReference(edgex, service, r.Scheme)
			},
		)

		if err != nil {
			return false, err
		}

		readyservice++
	}

	servicelist := &corev1.ServiceList{}
	if err := r.List(ctx, servicelist, client.MatchingLabels{devicev1alpha1.LabelEdgeXGenerate: LabelService}); err == nil {
		for _, s := range servicelist.Items {
			if _, ok := needservices[s.Name]; ok {
				continue
			}
			r.removeOwner(ctx, edgex, &s)
		}
	}

	return true, nil
}

func (r *EdgeXReconciler) reconcileDeployment(ctx context.Context, edgex *devicev1alpha1.EdgeX) (bool, error) {
	desiredeployments := append(CoreDeployment[edgex.Spec.Version], edgex.Spec.AdditionalDeployment...)
	needdeployments := make(map[string]bool)
	var readydeployment int32

	defer func() {
		edgex.Status.DeploymentReplicas = int32(len(desiredeployments))
		edgex.Status.DeploymentReadyReplicas = readydeployment
	}()

NextUD:
	for _, desireDeployment := range desiredeployments {
		needdeployments[desireDeployment.Name] = true

		ud := &unitv1alpha1.UnitedDeployment{}
		err := r.Get(
			ctx,
			types.NamespacedName{
				Namespace: edgex.Namespace,
				Name:      desireDeployment.Name},
			ud)

		if err != nil {
			if !apierrors.IsNotFound(err) {
				return false, err
			}
			desirespec := desireDeployment.Spec.DeepCopy()

			if edgex.Spec.ImageRegistry != "" {
				for i := range desirespec.Template.Spec.Containers {
					desirespec.Template.Spec.Containers[i].Image = edgex.Spec.ImageRegistry + "/" + desirespec.Template.Spec.Containers[i].Image
				}
			}

			ud = &unitv1alpha1.UnitedDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      make(map[string]string),
					Annotations: make(map[string]string),
					Name:        desireDeployment.Name,
					Namespace:   edgex.Namespace,
				},
				Spec: unitv1alpha1.UnitedDeploymentSpec{
					Selector: desireDeployment.Spec.Selector.DeepCopy(),
					WorkloadTemplate: unitv1alpha1.WorkloadTemplate{
						DeploymentTemplate: &unitv1alpha1.DeploymentTemplateSpec{
							ObjectMeta: *desireDeployment.Spec.Template.ObjectMeta.DeepCopy(),
							Spec:       *desirespec},
					},
				},
			}

			ud.Labels[devicev1alpha1.LabelEdgeXGenerate] = LabelDeployment
			pool := unitv1alpha1.Pool{
				Name:     edgex.Spec.PoolName,
				Replicas: pointer.Int32Ptr(1),
			}
			pool.NodeSelectorTerm.MatchExpressions = append(pool.NodeSelectorTerm.MatchExpressions,
				corev1.NodeSelectorRequirement{
					Key:      unitv1alpha1.LabelCurrentNodePool,
					Operator: corev1.NodeSelectorOpIn,
					Values:   []string{edgex.Spec.PoolName},
				})
			ud.Spec.Topology.Pools = append(ud.Spec.Topology.Pools, pool)
			if err := controllerutil.SetOwnerReference(edgex, ud, r.Scheme); err != nil {
				return false, err
			}
			if err := r.Create(ctx, ud); err != nil {
				return false, err
			}
		} else {
			if _, ok := ud.Status.PoolReplicas[edgex.Spec.PoolName]; ok {
				if ud.Status.ReadyReplicas == ud.Status.Replicas {
					readydeployment++
				}
				continue NextUD
			}

			pool := unitv1alpha1.Pool{
				Name:     edgex.Spec.PoolName,
				Replicas: pointer.Int32Ptr(1),
			}
			pool.NodeSelectorTerm.MatchExpressions = append(pool.NodeSelectorTerm.MatchExpressions,
				corev1.NodeSelectorRequirement{
					Key:      unitv1alpha1.LabelCurrentNodePool,
					Operator: corev1.NodeSelectorOpIn,
					Values:   []string{edgex.Spec.PoolName},
				})
			ud.Spec.Topology.Pools = append(ud.Spec.Topology.Pools, pool)
			if err := controllerutil.SetOwnerReference(edgex, ud, r.Scheme); err != nil {
				return false, err
			}
			if err := r.Update(ctx, ud); err != nil {
				return false, err
			}
		}
	}

	deploymentlist := &unitv1alpha1.UnitedDeploymentList{}
	if err := r.List(ctx, deploymentlist, client.MatchingLabels{devicev1alpha1.LabelEdgeXGenerate: LabelDeployment}); err == nil {
		for _, s := range deploymentlist.Items {
			if _, ok := needdeployments[s.Name]; ok {
				continue
			}
			r.removeOwner(ctx, edgex, &s)
		}
	}

	return readydeployment == int32(len(desiredeployments)), nil
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
