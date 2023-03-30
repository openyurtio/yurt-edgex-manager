/*
Copyright 2022 The OpenYurt Authors.

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

package util

import (
	"context"
	"sync"

	"github.com/openyurtio/yurt-edgex-manager/api/v1alpha1"
	"github.com/openyurtio/yurt-edgex-manager/api/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	IndexerPathForNodepoolv1 = "spec.poolname"
	IndexerPathForNodepoolv2 = "spec.poolname"
)

var registerOnce sync.Once

func RegisterFieldIndexers(fi client.FieldIndexer) error {
	register := func(obj client.Object, path string) error {
		return fi.IndexField(context.TODO(), obj, path, func(rawObj client.Object) []string {
			switch t := rawObj.(type) {
			case *v1alpha1.EdgeX:
				return []string{t.Spec.PoolName}
			case *v1alpha2.EdgeX:
				return []string{t.Spec.PoolName}
			default:
				return []string{}
			}
		})
	}
	if err := register(&v1alpha1.EdgeX{}, IndexerPathForNodepoolv1); err != nil {
		return err
	}
	if err := register(&v1alpha2.EdgeX{}, IndexerPathForNodepoolv2); err != nil {
		return err
	}
	return nil
}
