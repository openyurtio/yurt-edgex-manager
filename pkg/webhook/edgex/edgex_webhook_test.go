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
	"testing"

	corev1 "k8s.io/api/core/v1"

	v1 "github.com/openyurtio/api/apps/v1alpha1"
	"github.com/openyurtio/yurt-edgex-manager/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var defaultEdgeX = &v1alpha1.EdgeX{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test1",
		Namespace: "default",
	},
	Spec: v1alpha1.EdgeXSpec{
		PoolName: "beijing",
	},
}

func TestEdgeXDefaulter(t *testing.T) {
	webhook := &EdgeXHandler{}
	if err := webhook.Default(context.TODO(), defaultEdgeX); err != nil {
		t.Fatal(err)
	}
}

func TestEdgeXValidator(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	_ = v1.AddToScheme(scheme)

	beijingNodePool := &v1.NodePool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "beijing",
			Namespace: "default",
		},
		Spec: v1.NodePoolSpec{},
	}
	hangzhouNodePool := &v1.NodePool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hangzhou",
			Namespace: "default",
		},
		Spec: v1.NodePoolSpec{},
	}
	objs := []client.Object{beijingNodePool, hangzhouNodePool}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
	webhook := &EdgeXHandler{Client: client}

	// set default value
	if err := webhook.Default(context.TODO(), defaultEdgeX); err != nil {
		t.Fatal(err)
	}

	//validate edgex's version
	if err := webhook.ValidateCreate(context.TODO(), defaultEdgeX); err != nil {
		t.Fatal("edgex should create success", err)
	}
	EdgeX2 := defaultEdgeX.DeepCopy()
	EdgeX2.ObjectMeta.Name = "test2"
	EdgeX2.Spec.Version = "test"
	if err := webhook.ValidateCreate(context.TODO(), EdgeX2); err == nil {
		t.Fatal("edgex should create fail", err)
	}

	//validate edgex's ServiceType
	if err := webhook.ValidateCreate(context.TODO(), defaultEdgeX); err != nil {
		t.Fatal("edgex should create success", err)
	}
	EdgeX2.Spec.Version = "jakarta"
	EdgeX2.Spec.ServiceType = "test"
	if err := webhook.ValidateCreate(context.TODO(), EdgeX2); err == nil {
		t.Fatal("edgex should create fail", err)
	}

	//validate edgex's poolname
	EdgeX2.Spec.ServiceType = corev1.ServiceTypeClusterIP
	EdgeX2.Spec.PoolName = "hangzhou"
	if err := webhook.ValidateUpdate(context.TODO(), defaultEdgeX, EdgeX2); err != nil {
		t.Fatal("edgex should update success", err)
	}

	EdgeX2.Spec.PoolName = "shanghai"
	if err := webhook.ValidateUpdate(context.TODO(), defaultEdgeX, EdgeX2); err == nil {
		t.Fatal("edgex should update fail", err)
	}

	objs = append(objs, defaultEdgeX)
	client = fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
	webhook = &EdgeXHandler{Client: client}
	EdgeX2.Spec.PoolName = "beijing"
	if err := webhook.ValidateCreate(context.TODO(), EdgeX2); err == nil {
		t.Fatal("edgex should create fail", err)
	}

}
