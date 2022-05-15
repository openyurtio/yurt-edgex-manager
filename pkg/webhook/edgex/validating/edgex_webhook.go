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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/openyurtio/yurt-edgex-manager/api/v1alpha1"
	util "github.com/openyurtio/yurt-edgex-manager/controllers/utils"
)

type EdgeX struct {
	Client client.Reader
}

func (webhook *EdgeX) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.EdgeX{}).
		WithValidator(webhook).
		Complete()
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-device-openyurt-io-v1alpha1-edgex,mutating=false,failurePolicy=fail,sideEffects=None,groups=device.openyurt.io,resources=edgexes,verbs=create;update,versions=v1alpha1,name=vedgex.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &EdgeX{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type.
func (webhook *EdgeX) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	edgex, ok := obj.(*v1alpha1.EdgeX)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Cluster but got a %T", obj))
	}
	return webhook.validate(ctx, nil, edgex)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type.
func (webhook *EdgeX) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	newEdgex, ok := newObj.(*v1alpha1.EdgeX)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Cluster but got a %T", newObj))
	}
	oldEdgex, ok := oldObj.(*v1alpha1.EdgeX)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Cluster but got a %T", oldObj))
	}
	return webhook.validate(ctx, oldEdgex, newEdgex)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type.
func (webhook *EdgeX) ValidateDelete(_ context.Context, obj runtime.Object) error {
	return nil
}

var svcTypes = sets.NewString(string(corev1.ServiceTypeNodePort), string(corev1.ServiceTypeClusterIP), "")

func (webhook *EdgeX) validate(ctx context.Context, oldEdgex, newEdgex *v1alpha1.EdgeX) error {

	if !svcTypes.Has(string(newEdgex.Spec.ServiceType)) {
		return fmt.Errorf("serviceType should be in list: %v", svcTypes.List())
	}

	var edgexes v1alpha1.EdgeXList
	listOptions := client.MatchingFields{util.IndexerPathForNodepool: newEdgex.Spec.PoolName}
	if err := webhook.Client.List(ctx, &edgexes, listOptions); err != nil {
		klog.ErrorS(err, "fail to list the edgex")
		return err
	}

	if (oldEdgex == nil && len(edgexes.Items) != 0) || (oldEdgex != nil && len(edgexes.Items) > 1) {
		return fmt.Errorf("nodepool: %s already used by other edgex instance, namesapce: %s, name: %s.", newEdgex.Spec.PoolName, edgexes.Items[0].Namespace, edgexes.Items[0].Name)
	}

	return nil
}
