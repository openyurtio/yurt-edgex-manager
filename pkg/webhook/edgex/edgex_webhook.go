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

package v1alpha2

import (
	"context"
	"fmt"
	"strings"

	unitv1alpha1 "github.com/openyurtio/api/apps/v1alpha1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"gopkg.in/yaml.v2"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/openyurtio/yurt-edgex-manager/api/v1alpha2"
	util "github.com/openyurtio/yurt-edgex-manager/controllers/utils"
)

type Manifest struct {
	Updated       string   `yaml:"updated"`
	Count         int      `yaml:"count"`
	LatestVersion string   `yaml:"latestVersion"`
	Versions      []string `yaml:"versions"`
}

func NewManifest() *Manifest {
	manifest := &Manifest{
		Updated:       "false",
		Count:         0,
		LatestVersion: "",
		Versions:      make([]string, 0),
	}
	return manifest
}

// SetupWebhookWithManager sets up Cluster webhooks.
func (webhook *EdgeXHandler) SetupWebhookWithManager(mgr ctrl.Manager) error {

	err := webhook.initManifest(webhook.ManifestContent)
	if err != nil {
		return err
	}

	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha2.EdgeX{}).
		WithDefaulter(webhook).
		WithValidator(webhook).
		Complete()
}

func (webhook *EdgeXHandler) initManifest(manifestContent []byte) error {

	err := yaml.Unmarshal(manifestContent, &manifest)
	if err != nil {
		return fmt.Errorf("Error manifest edgeX configuration file %w", err)
	}
	return nil
}

var manifest = NewManifest()

//+kubebuilder:rbac:groups=apps.openyurt.io,resources=nodepools,verbs=list;watch

// Cluster implements a validating and defaulting webhook for Cluster.
type EdgeXHandler struct {
	Client          client.Client
	ManifestContent []byte
}

//+kubebuilder:webhook:path=/mutate-device-openyurt-io-v1alpha2-edgex,mutating=true,failurePolicy=fail,sideEffects=None,groups=device.openyurt.io,resources=edgexes,verbs=create;update,versions=v1alpha2,name=medgex.kb.io,admissionReviewVersions={"v1", "v2"}

var _ webhook.CustomDefaulter = &EdgeXHandler{}

//+kubebuilder:webhook:path=/validate-device-openyurt-io-v1alpha2-edgex,mutating=false,failurePolicy=fail,sideEffects=None,groups=device.openyurt.io,resources=edgexes,verbs=create;update,versions=v1alpha2,name=vedgex.kb.io,admissionReviewVersions={"v1", "v2"}

var _ webhook.CustomValidator = &EdgeXHandler{}

// Default satisfies the defaulting webhook interface.
func (webhook *EdgeXHandler) Default(ctx context.Context, obj runtime.Object) error {
	edgex, ok := obj.(*v1alpha2.EdgeX)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a EdgeX but got a %T", obj))
	}

	if edgex.Spec.Version == "" {
		edgex.Spec.Version = manifest.LatestVersion
	}

	return nil
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type.
func (webhook *EdgeXHandler) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	edgex, ok := obj.(*v1alpha2.EdgeX)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Cluster but got a %T", obj))
	}

	if allErrs := webhook.validate(ctx, edgex); len(allErrs) > 0 {
		return apierrors.NewInvalid(v1alpha2.GroupVersion.WithKind("EdgeX").GroupKind(), edgex.Name, allErrs)
	}

	return nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type.
func (webhook *EdgeXHandler) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	newEdgex, ok := newObj.(*v1alpha2.EdgeX)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a new Cluster but got a %T", newObj))
	}

	oldEdgex, ok := oldObj.(*v1alpha2.EdgeX)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a old Cluster but got a %T", newObj))
	}

	newErrorList := webhook.validate(ctx, newEdgex)
	oldErrorList := webhook.validate(ctx, oldEdgex)
	if allErrs := append(newErrorList, oldErrorList...); len(allErrs) > 0 {
		return apierrors.NewInvalid(v1alpha2.GroupVersion.WithKind("EdgeX").GroupKind(), newEdgex.Name, allErrs)
	}
	return nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type.
func (webhook *EdgeXHandler) ValidateDelete(_ context.Context, obj runtime.Object) error {
	return nil
}

// validate validates a EdgeX
func (webhook *EdgeXHandler) validate(ctx context.Context, edgex *v1alpha2.EdgeX) field.ErrorList {

	// verify the version
	if specErrs := webhook.validateEdgeXSpec(edgex); specErrs != nil {
		return specErrs
	}
	// verify that the poolname nodepool
	if nodePoolErrs := webhook.validateEdgeXWithNodePools(ctx, edgex); nodePoolErrs != nil {
		return nodePoolErrs
	}
	return nil
}

func (webhook *EdgeXHandler) validateEdgeXSpec(edgex *v1alpha2.EdgeX) field.ErrorList {
	for _, version := range manifest.Versions {
		if edgex.Spec.Version == version {
			return nil
		}
	}

	return field.ErrorList{
		field.Invalid(field.NewPath("spec", "version"), edgex.Spec.Version, "must be one of"+strings.Join(manifest.Versions, ",")),
	}
}

func (webhook *EdgeXHandler) validateEdgeXWithNodePools(ctx context.Context, edgex *v1alpha2.EdgeX) field.ErrorList {
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
	var edgexes v1alpha2.EdgeXList
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

	return nil

}
