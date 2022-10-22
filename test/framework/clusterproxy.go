/*
Copyright 2022 Wuming Liu.

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

package framework

import (
	"context"
	"fmt"

	. "github.com/onsi/gomega"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/cluster-api/test/framework/exec"
	"sigs.k8s.io/controller-runtime/pkg/client"

	unitv1alpha1 "github.com/openyurtio/api/apps/v1alpha1"
	devicev1alpha1 "github.com/openyurtio/yurt-edgex-manager/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ClusterProxy defines the behavior of a type that acts as an intermediary with an existing Kubernetes cluster.
// It should work with any Kubernetes cluster, no matter if the Cluster was created by a bootstrap.ClusterProvider,
// by Cluster API (a workload cluster or a self-hosted cluster) or else.
type ClusterProxy interface {
	// GetKubeconfigPath returns the path to the kubeconfig file to be used to access the Kubernetes cluster.
	GetKubeconfigPath() string

	// GetScheme returns the scheme defining the types hosted in the Kubernetes cluster.
	// It is used when creating a controller-runtime client.
	GetScheme() *runtime.Scheme

	// GetClient returns a controller-runtime client to the Kubernetes cluster.
	GetClient() client.Client

	// Apply to apply YAML to the Kubernetes cluster, `kubectl apply`.
	Apply(ctx context.Context, resources string, args ...string) error

	// Apply to delete YAML to the Kubernetes cluster, `kubectl delete`.
	Delete(ctx context.Context, resources string, args ...string) error
}

// clusterProxy provides a base implementation of the ClusterProxy interface.
type clusterProxy struct {
	kubeconfigPath string
	scheme         *runtime.Scheme
}

// NewClusterProxy returns a clusterProxy given a KubeconfigPath and the scheme defining the types hosted in the cluster.
// If a kubeconfig file isn't provided, standard kubeconfig locations will be used (kubectl loading rules apply).
func NewClusterProxy(kubeconfigPath string) ClusterProxy {
	Expect(kubeconfigPath).NotTo(BeEmpty(), "kubeconfigPath is required for NewClusterProxy")

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	// Add the apps schemes.
	_ = appsv1.AddToScheme(scheme)

	// Add the api extensions (CRD) to the scheme.
	_ = apiextensionsv1beta.AddToScheme(scheme)
	_ = apiextensionsv1.AddToScheme(scheme)

	// Add rbac to the scheme.
	_ = rbacv1.AddToScheme(scheme)
	_ = unitv1alpha1.AddToScheme(scheme)

	_ = devicev1alpha1.AddToScheme(scheme)

	proxy := &clusterProxy{
		kubeconfigPath: kubeconfigPath,
		scheme:         scheme,
	}

	return proxy
}

// GetKubeconfigPath returns the path to the kubeconfig file for the cluster.
func (p *clusterProxy) GetKubeconfigPath() string {
	return p.kubeconfigPath
}

// GetScheme returns the scheme defining the types hosted in the cluster.
func (p *clusterProxy) GetScheme() *runtime.Scheme {
	return p.scheme
}

// GetClient returns a controller-runtime client for the cluster.
func (p *clusterProxy) GetClient() client.Client {
	config := p.GetRESTConfig()

	c, err := client.New(config, client.Options{Scheme: p.scheme})
	Expect(err).ToNot(HaveOccurred(), "Failed to get controller-runtime client")

	return c
}

func (p *clusterProxy) GetRESTConfig() *rest.Config {
	config, err := clientcmd.LoadFromFile(p.kubeconfigPath)
	Expect(err).ToNot(HaveOccurred(), "Failed to load Kubeconfig file from %q", p.kubeconfigPath)

	restConfig, err := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{}).ClientConfig()
	Expect(err).ToNot(HaveOccurred(), "Failed to get ClientConfig from %q", p.kubeconfigPath)

	restConfig.UserAgent = "cluster-api-e2e"
	return restConfig
}

// Apply wraps `kubectl apply ...` and prints the output so we can see what gets applied to the cluster.
func (p *clusterProxy) Apply(ctx context.Context, resources string, args ...string) error {
	Expect(ctx).NotTo(BeNil(), "ctx is required for Apply")
	Expect(resources).NotTo(BeNil(), "resources is required for Apply")

	return KubectlApply(ctx, p.kubeconfigPath, resources, args...)
}

// Delete wraps `kubectl delete ...` and prints the output so we can see what deletes applied to the cluster.
func (p *clusterProxy) Delete(ctx context.Context, resources string, args ...string) error {
	Expect(ctx).NotTo(BeNil(), "ctx is required for Delete")
	Expect(resources).NotTo(BeNil(), "resources is required for Delete")

	return KubectlDelete(ctx, p.kubeconfigPath, resources, args...)
}

func KubectlApply(ctx context.Context, kubeconfigPath string, resources string, args ...string) error {
	aargs := append([]string{"apply", "--kubeconfig", kubeconfigPath, "-f", resources}, args...)
	applyCmd := exec.NewCommand(
		exec.WithCommand("kubectl"),
		exec.WithArgs(aargs...),
	)
	stdout, stderr, err := applyCmd.Run(ctx)
	if err != nil {
		fmt.Println(string(stderr))
		return err
	}
	fmt.Println(string(stdout))
	return nil
}

func KubectlDelete(ctx context.Context, kubeconfigPath string, resources string, args ...string) error {
	aargs := append([]string{"delete", "--kubeconfig", kubeconfigPath, "-f", resources}, args...)
	deleteCmd := exec.NewCommand(
		exec.WithCommand("kubectl"),
		exec.WithArgs(aargs...),
	)
	stdout, stderr, err := deleteCmd.Run(ctx)
	if err != nil {
		fmt.Println(string(stderr))
		return err
	}
	fmt.Println(string(stdout))
	return nil
}
