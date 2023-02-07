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
	"strconv"
	"strings"
	"time"

	unitv1alpha1 "github.com/openyurtio/api/apps/v1alpha1"
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
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
	HostPathType       = corev1.HostPathDirectoryOrCreate
)

// EdgeXReconciler reconciles a EdgeX object
type EdgeXReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type EdgeXConfig struct {
	Versions []Version `yaml:"versions"`
}
type Version struct {
	Name       string            `yaml:"versionName"`
	Env        map[string]string `yaml:"env,omitempty"`
	Components []Component       `yaml:"components,omitempty"`
}
type Component struct {
	Name         string            `yaml:"name"`
	Image        string            `yaml:"image"`
	Volumes      []Volume          `yaml:"volumes,omitempty"`
	Ports        []Port            `yaml:"ports,omitempty"`
	ComponentEnv map[string]string `yaml:"componentEnv,omitempty"`
	// A pointer to the Env of the previous level
	envRef *map[string]string
}

type Volume struct {
	Name      string `yaml:"name"`
	HostPath  string `yaml:"hostPath"`
	MountPath string `yaml:"mountPath"`
}

type Port struct {
	Protocol   string `yaml:"protocol"`
	Port       int32  `yaml:"port"`
	TargetPort int32  `yaml:"targetPort"`
	NodePort   int32  `yaml:"nodePort,omitempty"`
}

var (
	SecurityComponents map[string][]Component       = make(map[string][]Component)
	NoSectyComponents  map[string][]Component       = make(map[string][]Component)
	SecurityEnv        map[string]map[string]string = make(map[string]map[string]string)
	NoSectyEnv         map[string]map[string]string = make(map[string]map[string]string)
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
	var desiredComponents []Component
	if edgex.Spec.Security {
		desiredComponents = SecurityComponents[edgex.Spec.Version]
	} else {
		desiredComponents = NoSectyComponents[edgex.Spec.Version]
	}
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
	configmap := &corev1.ConfigMap{}
	var env map[string]string
	if edgex.Spec.Security {
		env = SecurityEnv[edgex.Spec.Version]
	} else {
		env = NoSectyEnv[edgex.Spec.Version]
	}
	if env != nil {
		configmap.ObjectMeta = metav1.ObjectMeta{
			Labels:    make(map[string]string),
			Name:      ConfigMapName + "-" + edgex.Spec.Version,
			Namespace: edgex.Namespace,
		}
		configmap.Data = make(map[string]string)
		configmap.Labels[devicev1alpha2.LabelEdgeXGenerate] = LabelConfigmap

		for k, v := range env {
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
	if err := r.List(ctx, configmaplist, client.InNamespace(edgex.Namespace), client.MatchingLabels{devicev1alpha2.LabelEdgeXGenerate: LabelConfigmap}); err == nil {
		for _, c := range configmaplist.Items {
			if c.Name == configmap.Name {
				continue
			}
			r.removeOwner(ctx, edgex, &c)
		}
	}

	return true, nil
}

func (r *EdgeXReconciler) reconcileComponent(ctx context.Context, edgex *devicev1alpha2.EdgeX) (bool, error) {
	var desirecomponents []Component
	needcomponents := make(map[string]bool)
	var readycomponent int32

	efs := corev1.EnvFromSource{
		ConfigMapRef: &corev1.ConfigMapEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: ConfigMapName + "-" + edgex.Spec.Version,
			},
		},
	}

	if edgex.Spec.Security {
		desirecomponents = SecurityComponents[edgex.Spec.Version]
	} else {
		desirecomponents = NoSectyComponents[edgex.Spec.Version]
	}
	//TODO: handle edgex.Spec.Components

	defer func() {
		edgex.Status.ReadyComponentNum = readycomponent
		edgex.Status.UnreadyComponentNum = int32(len(desirecomponents)) - readycomponent
	}()

NextC:
	for _, desirecomponent := range desirecomponents {
		readyService := false
		readyDeployment := false
		needcomponents[desirecomponent.Name] = true
		serviceport := []corev1.ServicePort{}
		containerport := []corev1.ContainerPort{}
		handlePort(&serviceport, &containerport, desirecomponent.Ports)
		volumemount := []corev1.VolumeMount{}
		volume := []corev1.Volume{}
		handleVolume(&volumemount, &volume, desirecomponent.Volumes)
		envs := []corev1.EnvVar{}
		handleEnv(&envs, desirecomponent.ComponentEnv)

		ok, err := r.handleService(ctx, edgex, desirecomponent.Name, &serviceport)
		if !ok {
			return ok, err
		}
		readyService = true

		ud := &unitv1alpha1.YurtAppSet{}
		err = r.Get(
			ctx,
			types.NamespacedName{
				Namespace: edgex.Namespace,
				Name:      desirecomponent.Name},
			ud)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return false, err
			}
			container := corev1.Container{
				Name:            desirecomponent.Name,
				Image:           desirecomponent.Image,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Ports:           containerport,
				VolumeMounts:    volumemount,
				EnvFrom:         []corev1.EnvFromSource{efs},
				Env:             envs,
			}
			ok, err := r.handleDeployment(ctx, edgex, ud, desirecomponent.Name, &volume, &container)
			if !ok {
				return ok, err
			}
		} else {
			if _, ok := ud.Status.PoolReplicas[edgex.Spec.PoolName]; ok {
				if ud.Status.ReadyReplicas == ud.Status.Replicas {
					readyDeployment = true
					if readyDeployment && readyService {
						readycomponent++
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
			if _, ok := needcomponents[s.Name]; ok {
				continue
			}
			r.removeOwner(ctx, edgex, &s)
		}
	}

	/* Remove the deployment owner that we do not need */
	deploymentlist := &unitv1alpha1.YurtAppSetList{}
	if err := r.List(ctx, deploymentlist, client.InNamespace(edgex.Namespace), client.MatchingLabels{devicev1alpha2.LabelEdgeXGenerate: LabelDeployment}); err == nil {
		for _, s := range deploymentlist.Items {
			if _, ok := needcomponents[s.Name]; ok {
				continue
			}
			r.removeOwner(ctx, edgex, &s)
		}
	}

	return readycomponent == int32(len(desirecomponents)), nil
}
func handlePort(serviceports *[]corev1.ServicePort, containerports *[]corev1.ContainerPort, componentports []Port) {
	for _, port := range componentports {
		sp := corev1.ServicePort{
			Protocol:   corev1.Protocol(port.Protocol),
			Port:       port.Port,
			TargetPort: intstr.FromInt(int(port.TargetPort)),
			Name:       strings.ToLower(port.Protocol) + "-" + strconv.FormatInt(int64(port.Port), 10),
		}
		*serviceports = append(*serviceports, sp)

		cp := corev1.ContainerPort{
			Protocol:      corev1.Protocol(port.Protocol),
			ContainerPort: port.TargetPort,
			Name:          strings.ToLower(port.Protocol) + "-" + strconv.FormatInt(int64(port.Port), 10),
		}
		*containerports = append(*containerports, cp)
	}
}

func handleVolume(volumemounts *[]corev1.VolumeMount, volumes *[]corev1.Volume, componentvolumes []Volume) {
	for _, v := range componentvolumes {
		vm := corev1.VolumeMount{
			Name:      v.Name,
			MountPath: v.MountPath,
		}
		*volumemounts = append(*volumemounts, vm)
		var vs corev1.Volume
		if v.HostPath == "" {
			vs = corev1.Volume{
				Name: v.Name,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			}
		} else {
			vs = corev1.Volume{
				Name: v.Name,
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: v.HostPath,
						Type: &HostPathType,
					},
				},
			}
		}

		*volumes = append(*volumes, vs)
	}
}

func handleEnv(envs *[]corev1.EnvVar, componentenvs map[string]string) {
	for n, e := range componentenvs {
		env := corev1.EnvVar{
			Name:  n,
			Value: e,
		}
		*envs = append(*envs, env)
	}
}

func (r *EdgeXReconciler) handleService(ctx context.Context, edgex *devicev1alpha2.EdgeX, name string, serviceport *[]corev1.ServicePort) (bool, error) {
	if len(*serviceport) == 0 {
		return true, nil
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			Name:        name,
			Namespace:   edgex.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": name},
			Ports:    *serviceport,
		},
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
		return false, err
	}
	return true, nil
}

func (r *EdgeXReconciler) handleDeployment(ctx context.Context, edgex *devicev1alpha2.EdgeX, ud *unitv1alpha1.YurtAppSet, name string, volumes *[]corev1.Volume, container *corev1.Container) (bool, error) {
	ud = &unitv1alpha1.YurtAppSet{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			Name:        name,
			Namespace:   edgex.Namespace,
		},
		Spec: unitv1alpha1.YurtAppSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			WorkloadTemplate: unitv1alpha1.WorkloadTemplate{
				DeploymentTemplate: &unitv1alpha1.DeploymentTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": name},
					},
					Spec: v1.DeploymentSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": name},
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{"app": name},
							},
							Spec: corev1.PodSpec{
								Volumes:    *volumes,
								Containers: []corev1.Container{*container},
								Hostname:   name,
							},
						},
					},
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
		return false, err
	}
	if err := r.Create(ctx, ud); err != nil {
		return false, err
	}
	return true, nil
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
