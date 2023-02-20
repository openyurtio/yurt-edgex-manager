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
	"encoding/json"
	"reflect"
	"time"

	unitv1alpha1 "github.com/openyurtio/api/apps/v1alpha1"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
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

	devicev1alpha1 "github.com/openyurtio/yurt-edgex-manager/api/v1alpha1"
	devicev1alpha2 "github.com/openyurtio/yurt-edgex-manager/api/v1alpha2"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
)

const (
	LabelConfigmap  = "Configmap"
	LabelService    = "Service"
	LabelDeployment = "Deployment"

	AnnotationServiceTopologyKey           = "openyurt.io/topologyKeys"
	AnnotationServiceTopologyValueNodePool = "openyurt.io/nodepool"

	ConfigMapName = "common-variables"
)

var (
	ControlledType     = &devicev1alpha2.EdgeX{}
	ControlledTypeName = reflect.TypeOf(ControlledType).Elem().Name()
	ControlledTypeGVK  = devicev1alpha1.GroupVersion.WithKind(ControlledTypeName)
)

// EdgeXReconciler reconciles a EdgeX object
type EdgeXReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type EdgeXConfig struct {
	Versions []*Version `yaml:"versions"`
}

type Version struct {
	Name       string             `yaml:"versionName"`
	ConfigMaps []corev1.ConfigMap `yaml:"configMaps,omitempty"`
	Components []*Component       `yaml:"components,omitempty"`
}

type Component struct {
	Name       string                 `yaml:"name"`
	Service    *corev1.ServiceSpec    `yaml:"service,omitempty"`
	Deployment *appsv1.DeploymentSpec `yaml:"deployment,omitempty"`
}

var (
	SecurityComponents map[string][]*Component       = make(map[string][]*Component)
	NoSectyComponents  map[string][]*Component       = make(map[string][]*Component)
	SecurityConfigMaps map[string][]corev1.ConfigMap = make(map[string][]corev1.ConfigMap)
	NoSectyConfigMaps  map[string][]corev1.ConfigMap = make(map[string][]corev1.ConfigMap)
)

//+kubebuilder:rbac:groups=device.openyurt.io,resources=edgexes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=device.openyurt.io,resources=edgexes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=device.openyurt.io,resources=edgexes/finalizers,verbs=update
//+kubebuilder:rbac:groups=device.openyurt.io,resources=edgexes/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps.openyurt.io,resources=yurtappsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.openyurt.io,resources=yurtappsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=configmaps;services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps/status;services/status,verbs=get;update;patch

func (r *EdgeXReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)

	edgex := &devicev1alpha2.EdgeX{}
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
				devicev1alpha2.ConfigmapAvailableCondition,
				devicev1alpha2.ComponentAvailableCondition,
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

func (r *EdgeXReconciler) reconcileDelete(ctx context.Context, edgex *devicev1alpha2.EdgeX) (ctrl.Result, error) {

	ud := &unitv1alpha1.YurtAppSet{}
	var desiredComponents []*Component
	if edgex.Spec.Security {
		desiredComponents = SecurityComponents[edgex.Spec.Version]
	} else {
		desiredComponents = NoSectyComponents[edgex.Spec.Version]
	}

	additionalComponents, err := annotationToComponent(edgex.Annotations)
	if err != nil {
		return ctrl.Result{}, err
	}
	desiredComponents = append(desiredComponents, additionalComponents...)

	//TODO: handle edgex.Spec.Components

	for _, dc := range desiredComponents {
		if err := r.Get(
			ctx,
			types.NamespacedName{Namespace: edgex.Namespace, Name: dc.Name},
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

	controllerutil.RemoveFinalizer(edgex, devicev1alpha2.EdgexFinalizer)

	return ctrl.Result{}, nil
}

func (r *EdgeXReconciler) reconcileNormal(ctx context.Context, edgex *devicev1alpha2.EdgeX) (ctrl.Result, error) {
	controllerutil.AddFinalizer(edgex, devicev1alpha2.EdgexFinalizer)

	edgex.Status.Initialized = true

	if ok, err := r.reconcileConfigmap(ctx, edgex); !ok {
		if err != nil {
			conditions.MarkFalse(edgex, devicev1alpha2.ConfigmapAvailableCondition, devicev1alpha2.ConfigmapProvisioningFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
			return ctrl.Result{}, errors.Wrapf(err,
				"unexpected error while reconciling configmap for %s", edgex.Namespace+"/"+edgex.Name)
		}
		conditions.MarkFalse(edgex, devicev1alpha2.ConfigmapAvailableCondition, devicev1alpha2.ConfigmapProvisioningReason, clusterv1.ConditionSeverityInfo, "")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	conditions.MarkTrue(edgex, devicev1alpha2.ConfigmapAvailableCondition)

	if ok, err := r.reconcileComponent(ctx, edgex); !ok {
		if err != nil {
			conditions.MarkFalse(edgex, devicev1alpha2.ComponentAvailableCondition, devicev1alpha2.ComponentProvisioningFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
			return ctrl.Result{}, errors.Wrapf(err,
				"unexpected error while reconciling Component for %s", edgex.Namespace+"/"+edgex.Name)
		}
		conditions.MarkFalse(edgex, devicev1alpha2.ComponentAvailableCondition, devicev1alpha2.ComponentProvisioningReason, clusterv1.ConditionSeverityInfo, "")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	conditions.MarkTrue(edgex, devicev1alpha2.ComponentAvailableCondition)

	edgex.Status.Ready = true

	return ctrl.Result{}, nil
}

func (r *EdgeXReconciler) removeOwner(ctx context.Context, edgex *devicev1alpha2.EdgeX, obj client.Object) error {
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

func (r *EdgeXReconciler) reconcileConfigmap(ctx context.Context, edgex *devicev1alpha2.EdgeX) (bool, error) {
	var configmaps []corev1.ConfigMap
	needConfigMaps := make(map[string]struct{})

	if edgex.Spec.Security {
		configmaps = SecurityConfigMaps[edgex.Spec.Version]
	} else {
		configmaps = NoSectyConfigMaps[edgex.Spec.Version]
	}
	for _, configmap := range configmaps {
		// Supplement runtime information
		configmap.Namespace = edgex.Namespace
		configmap.Labels = make(map[string]string)
		configmap.Labels[devicev1alpha2.LabelEdgeXGenerate] = LabelConfigmap

		_, err := controllerutil.CreateOrUpdate(ctx, r.Client, &configmap, func() error {
			return controllerutil.SetOwnerReference(edgex, &configmap, r.Scheme)
		})

		if err != nil {
			return false, err
		}

		needConfigMaps[configmap.Name] = struct{}{}
	}

	configmaplist := &corev1.ConfigMapList{}
	if err := r.List(ctx, configmaplist, client.InNamespace(edgex.Namespace), client.MatchingLabels{devicev1alpha2.LabelEdgeXGenerate: LabelConfigmap}); err == nil {
		for _, c := range configmaplist.Items {
			if _, ok := needConfigMaps[c.Name]; !ok {
				r.removeOwner(ctx, edgex, &c)
			}
		}
	}

	return true, nil
}

func (r *EdgeXReconciler) reconcileComponent(ctx context.Context, edgex *devicev1alpha2.EdgeX) (bool, error) {
	var desireComponents []*Component
	needComponents := make(map[string]struct{})
	var readyComponent int32 = 0

	if edgex.Spec.Security {
		desireComponents = SecurityComponents[edgex.Spec.Version]
	} else {
		desireComponents = NoSectyComponents[edgex.Spec.Version]
	}

	additionalComponents, err := annotationToComponent(edgex.Annotations)
	if err != nil {
		return false, err
	}
	desireComponents = append(desireComponents, additionalComponents...)

	//TODO: handle edgex.Spec.Components

	defer func() {
		edgex.Status.ReadyComponentNum = readyComponent
		edgex.Status.UnreadyComponentNum = int32(len(desireComponents)) - readyComponent
	}()

NextC:
	for _, desireComponent := range desireComponents {
		readyService := false
		readyDeployment := false
		needComponents[desireComponent.Name] = struct{}{}

		if _, err := r.handleService(ctx, edgex, desireComponent); err != nil {
			return false, err
		}
		readyService = true

		ud := &unitv1alpha1.YurtAppSet{}
		err := r.Get(
			ctx,
			types.NamespacedName{
				Namespace: edgex.Namespace,
				Name:      desireComponent.Name},
			ud)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return false, err
			}
			_, err = r.handleYurtAppSet(ctx, edgex, desireComponent)
			if err != nil {
				return false, err
			}
		} else {
			if _, ok := ud.Status.PoolReplicas[edgex.Spec.PoolName]; ok {
				if ud.Status.ReadyReplicas == ud.Status.Replicas {
					readyDeployment = true
					if readyDeployment && readyService {
						readyComponent++
					}
				}
				continue NextC
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
			flag := false
			for _, up := range ud.Spec.Topology.Pools {
				if up.Name == pool.Name {
					flag = true
					break
				}
			}
			if !flag {
				ud.Spec.Topology.Pools = append(ud.Spec.Topology.Pools, pool)
			}
			if err := controllerutil.SetOwnerReference(edgex, ud, r.Scheme); err != nil {
				return false, err
			}
			if err := r.Update(ctx, ud); err != nil {
				return false, err
			}
		}
	}

	/* Remove the service owner that we do not need */
	servicelist := &corev1.ServiceList{}
	if err := r.List(ctx, servicelist, client.InNamespace(edgex.Namespace), client.MatchingLabels{devicev1alpha2.LabelEdgeXGenerate: LabelService}); err == nil {
		for _, s := range servicelist.Items {
			if _, ok := needComponents[s.Name]; !ok {
				r.removeOwner(ctx, edgex, &s)
			}
		}
	}

	/* Remove the yurtappset owner that we do not need */
	yurtappsetlist := &unitv1alpha1.YurtAppSetList{}
	if err := r.List(ctx, yurtappsetlist, client.InNamespace(edgex.Namespace), client.MatchingLabels{devicev1alpha2.LabelEdgeXGenerate: LabelDeployment}); err == nil {
		for _, s := range yurtappsetlist.Items {
			if _, ok := needComponents[s.Name]; !ok {
				r.removeOwner(ctx, edgex, &s)
			}
		}
	}

	return readyComponent == int32(len(desireComponents)), nil
}

func (r *EdgeXReconciler) handleService(ctx context.Context, edgex *devicev1alpha2.EdgeX, component *Component) (*corev1.Service, error) {
	// It is possible that the component does not need service.
	// Therefore, you need to be careful when calling this function.
	// It is still possible for service to be nil when there is no error!
	if component.Service == nil {
		return nil, nil
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			Name:        component.Name,
			Namespace:   edgex.Namespace,
		},
		Spec: *component.Service,
	}
	service.Labels[devicev1alpha2.LabelEdgeXGenerate] = LabelService
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
		return nil, err
	}
	return service, nil
}

func (r *EdgeXReconciler) handleYurtAppSet(ctx context.Context, edgex *devicev1alpha2.EdgeX, component *Component) (*unitv1alpha1.YurtAppSet, error) {
	ud := &unitv1alpha1.YurtAppSet{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			Name:        component.Name,
			Namespace:   edgex.Namespace,
		},
		Spec: unitv1alpha1.YurtAppSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": component.Name},
			},
			WorkloadTemplate: unitv1alpha1.WorkloadTemplate{
				DeploymentTemplate: &unitv1alpha1.DeploymentTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": component.Name},
					},
					Spec: *component.Deployment,
				},
			},
		},
	}

	ud.Labels[devicev1alpha2.LabelEdgeXGenerate] = LabelDeployment
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
	if err := controllerutil.SetControllerReference(edgex, ud, r.Scheme); err != nil {
		return nil, err
	}
	if err := r.Create(ctx, ud); err != nil {
		return nil, err
	}
	return ud, nil
}

// For version compatibility, v1alpha1's additionalservice and additionaldeployment are placed in
// v2alpha2's annotation, this function is to convert the annotation to component.
func annotationToComponent(annotation map[string]string) ([]*Component, error) {
	var components []*Component = []*Component{}
	var additionalDeployments []devicev1alpha1.DeploymentTemplateSpec = make([]devicev1alpha1.DeploymentTemplateSpec, 0)
	if _, ok := annotation["AdditionalDeployments"]; ok {
		err := json.Unmarshal([]byte(annotation["AdditionalDeployments"]), &additionalDeployments)
		if err != nil {
			return nil, err
		}
	}
	var additionalServices []devicev1alpha1.ServiceTemplateSpec = make([]devicev1alpha1.ServiceTemplateSpec, 0)
	if _, ok := annotation["AdditionalServices"]; ok {
		err := json.Unmarshal([]byte(annotation["AdditionalServices"]), &additionalServices)
		if err != nil {
			return nil, err
		}
	}
	if len(additionalDeployments) == 0 && len(additionalServices) == 0 {
		return components, nil
	}
	var services map[string]*corev1.ServiceSpec = make(map[string]*corev1.ServiceSpec)
	var usedServices map[string]struct{} = make(map[string]struct{})
	for _, additionalservice := range additionalServices {
		services[additionalservice.Name] = &additionalservice.Spec
	}
	for _, additionalDeployment := range additionalDeployments {
		var component Component
		component.Name = additionalDeployment.Name
		component.Deployment = &additionalDeployment.Spec
		service, ok := services[component.Name]
		if ok {
			component.Service = service
			usedServices[component.Name] = struct{}{}
		}
		components = append(components, &component)
	}
	if len(usedServices) < len(services) {
		for name, service := range services {
			_, ok := usedServices[name]
			if ok {
				continue
			}
			var component Component
			component.Name = name
			component.Service = service
			components = append(components, &component)
		}
	}

	return components, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EdgeXReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(ControlledType).
		Watches(
			&source.Kind{Type: &unitv1alpha1.YurtAppSet{}},
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
