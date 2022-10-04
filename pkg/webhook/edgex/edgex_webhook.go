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

package validating

import (
	"context"
	"fmt"

	unitv1alpha1 "github.com/openyurtio/api/apps/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/openyurtio/yurt-edgex-manager/api/v1alpha1"
	util "github.com/openyurtio/yurt-edgex-manager/controllers/utils"
)

// SetupWebhookWithManager sets up Cluster webhooks.
func (webhook *EdgeXHandler) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.EdgeX{}).
		WithDefaulter(webhook).
		WithValidator(webhook).
		Complete()
}

//+kubebuilder:rbac:groups=apps.openyurt.io,resources=nodepools,verbs=list;watch

// Cluster implements a validating and defaulting webhook for Cluster.
type EdgeXHandler struct {
	Client client.Client
}

//+kubebuilder:webhook:path=/mutate-device-openyurt-io-v1alpha1-edgex,mutating=true,failurePolicy=fail,sideEffects=None,groups=device.openyurt.io,resources=edgexes,verbs=create;update,versions=v1alpha1,name=medgex.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &EdgeXHandler{}

//+kubebuilder:webhook:path=/validate-device-openyurt-io-v1alpha1-edgex,mutating=false,failurePolicy=fail,sideEffects=None,groups=device.openyurt.io,resources=edgexes,verbs=create;update,versions=v1alpha1,name=vedgex.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &EdgeXHandler{}

// Default satisfies the defaulting webhook interface.
func (webhook *EdgeXHandler) Default(ctx context.Context, obj runtime.Object) error {
	edgex, ok := obj.(*v1alpha1.EdgeX)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a EdgeX but got a %T", obj))
	}
	//set the default version
	if edgex.Spec.Version == "" {
		edgex.Spec.Version = "jakarta"
	}
	//set the default ServiceType
	if edgex.Spec.ServiceType == "" {
		edgex.Spec.ServiceType = corev1.ServiceTypeClusterIP
	}
	return nil
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type.
func (webhook *EdgeXHandler) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	edgex, ok := obj.(*v1alpha1.EdgeX)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Cluster but got a %T", obj))
	}
	if allErrs := webhook.validate(ctx, edgex); len(allErrs) > 0 {
		return apierrors.NewInvalid(v1alpha1.GroupVersion.WithKind("EdgeX").GroupKind(), edgex.Name, allErrs)
	}
	return nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type.
func (webhook *EdgeXHandler) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	newEdgex, ok := newObj.(*v1alpha1.EdgeX)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Cluster but got a %T", newObj))
	}
	oldEdgex, ok := oldObj.(*v1alpha1.EdgeX)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Cluster but got a %T", oldObj))
	}
	newErrorList := webhook.validate(ctx, newEdgex)
	oldErrorList := webhook.validate(ctx, oldEdgex)

	if allErrs := append(newErrorList, oldErrorList...); len(allErrs) > 0 {
		return apierrors.NewInvalid(v1alpha1.GroupVersion.WithKind("EdgeX").GroupKind(), newEdgex.Name, allErrs)
	}
	return nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type.
func (webhook *EdgeXHandler) ValidateDelete(_ context.Context, obj runtime.Object) error {
	return nil
}

// validate validates a EdgeX
func (webhook *EdgeXHandler) validate(ctx context.Context, edgex *v1alpha1.EdgeX) field.ErrorList {
	// verify the version
	if !(edgex.Spec.Version == "jakarta" || edgex.Spec.Version == "hanoi") {
		return field.ErrorList{
			field.Invalid(field.NewPath("spec", "version"), edgex.Spec.Version, "must be one of jakarta, hanoi"),
		}
	}
	// verify that the poolname is a right nodepool name
	nodePools := &unitv1alpha1.NodePoolList{}
	if err := webhook.Client.List(ctx, nodePools); err != nil {
		return field.ErrorList{
			field.Invalid(field.NewPath("spec", "poolName"), edgex.Spec.PoolName, "can not list nodepools, cause"+err.Error()),
		}
	}
	ok := false
	for _, nodePool := range nodePools.Items {
		if nodePool.ObjectMeta.Name == edgex.Spec.PoolName {
			ok = true
			break
		}
	}
	if !ok {
		return field.ErrorList{
			field.Invalid(field.NewPath("spec", "poolName"), edgex.Spec.PoolName, "can not find the nodePool"),
		}
	}
	// verify that no other edgex in the nodepool
	var edgexes v1alpha1.EdgeXList
	listOptions := client.MatchingFields{util.IndexerPathForNodepool: edgex.Spec.PoolName}
	if err := webhook.Client.List(ctx, &edgexes, listOptions); err != nil {
		return field.ErrorList{
			field.Invalid(field.NewPath("spec", "poolName"), edgex.Spec.PoolName, "can not list edgexes, cause"+err.Error()),
		}
	}
	for _, other := range edgexes.Items {
		if edgex.Name != other.Name {
			return field.ErrorList{
				field.Invalid(field.NewPath("spec", "poolName"), edgex.Spec.PoolName, "already used by other edgex instance,"),
			}
		}
	}
	// verify the ServiceType
	if !(edgex.Spec.ServiceType == corev1.ServiceTypeClusterIP || edgex.Spec.ServiceType == corev1.ServiceTypeNodePort) {
		return field.ErrorList{
			field.Invalid(field.NewPath("spec", "serviceType"), edgex.Spec.ServiceType, "must be one of ClusterIP, NodePort"),
		}
	}
	return nil
}
