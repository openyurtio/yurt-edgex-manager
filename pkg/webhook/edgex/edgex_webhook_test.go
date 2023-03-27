package v1alpha2

import (
	"context"
	"io/ioutil"
	"testing"

	v1 "github.com/openyurtio/api/apps/v1alpha1"
	"github.com/openyurtio/yurt-edgex-manager/api/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var defaultEdgeX = &v1alpha2.EdgeX{

	ObjectMeta: metav1.ObjectMeta{
		Name:      "test1",
		Namespace: "default",
	},
	Spec: v1alpha2.EdgeXSpec{
		PoolName: "beijing",
	},
}

func TestEdgeXDefaulter(t *testing.T) {
	webhook := &EdgeXHandler{}
	manifestPath := "../../../EdgeXConfig/manifest.yaml"
	manifestContent, err := ioutil.ReadFile(manifestPath)

	if err != nil {
		t.Fatal(err)
	}

	if err := webhook.initManifest(manifestContent); err != nil {
		t.Fatal(err)
	}
	if err := webhook.Default(context.TODO(), defaultEdgeX); err != nil {
		t.Fatal(err)
	}
}

func TestEdgeXValidator(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = v1alpha2.AddToScheme(scheme)
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

	manifestPath := "../../../EdgeXConfig/manifest.yaml"
	manifestContent, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := webhook.initManifest(manifestContent); err != nil {
		t.Fatal(err)
	}

	// set default value
	if err := webhook.Default(context.TODO(), defaultEdgeX); err != nil {
		t.Fatal(err)
	}

	//validate edgex's version
	// defaultEdgeX.Spec.Version = "testing"
	// if err := webhook.ValidateCreate(context.TODO(), defaultEdgeX); err == nil {
	// 	t.Fatal("edgex should create fail", err)
	// }

	// defaultEdgeX.Spec.Version = "levski"
	if err := webhook.ValidateCreate(context.TODO(), defaultEdgeX); err != nil {
		t.Fatal("edgex should create success", err)
	}

	//validate edgex's poolname
	EdgeX2 := defaultEdgeX.DeepCopy()
	EdgeX2.ObjectMeta.Name = "test2"
	EdgeX2.Spec.Version = "jakarta"
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
