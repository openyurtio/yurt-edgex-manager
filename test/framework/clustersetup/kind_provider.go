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

package clustersetup

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/docker/docker/client"
	kindv1 "sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
	kind "sigs.k8s.io/kind/pkg/cluster"
	kindnodes "sigs.k8s.io/kind/pkg/cluster/nodes"
	kindnodesutils "sigs.k8s.io/kind/pkg/cluster/nodeutils"
)

const (
	// DefaultNodeImageRepository is the default node image repository to be used for testing.
	DefaultNodeImageRepository = "kindest/node"

	// DefaultNodeImageVersion is the default Kubernetes version to be used for creating a kind cluster.
	DefaultNodeImageVersion = "v1.24.6"
)

func Logf(format string, a ...interface{}) {
	fmt.Fprintf(GinkgoWriter, "INFO: "+format+"\n", a...)
}

// KindClusterOption is a NewKindClusterProvider option.
type KindClusterOption interface {
	apply(*KindClusterProvider)
}

type kindClusterOptionAdapter func(*KindClusterProvider)

func (adapter kindClusterOptionAdapter) apply(kindClusterProvider *KindClusterProvider) {
	adapter(kindClusterProvider)
}

// WithNodeImage implements a New Option that instruct the kindClusterProvider to use a specific node image / Kubernetes version.
func WithNodeImage(image string) KindClusterOption {
	return kindClusterOptionAdapter(func(k *KindClusterProvider) {
		k.nodeImage = image
	})
}

// WithDockerSockMount implements a New Option that instruct the kindClusterProvider to mount /var/run/docker.sock into
// the new kind cluster.
func WithDockerSockMount() KindClusterOption {
	return kindClusterOptionAdapter(func(k *KindClusterProvider) {
		k.withDockerSock = true
	})
}

// WithIPv6Family implements a New Option that instruct the kindClusterProvider to set the IPFamily to IPv6 in
// the new kind cluster.
func WithIPv6Family() KindClusterOption {
	return kindClusterOptionAdapter(func(k *KindClusterProvider) {
		k.ipFamily = kindv1.IPv6Family
	})
}

// NewKindClusterProvider returns a ClusterProvider that can create a kind cluster.
func NewKindClusterProvider(name string, options ...KindClusterOption) *KindClusterProvider {
	Expect(name).ToNot(BeEmpty(), "name is required for NewKindClusterProvider")

	clusterProvider := &KindClusterProvider{
		name: name,
	}
	for _, option := range options {
		option.apply(clusterProvider)
	}
	return clusterProvider
}

// KindClusterProvider implements a ClusterProvider that can create a kind cluster.
type KindClusterProvider struct {
	name           string
	withDockerSock bool
	kubeconfigPath string
	nodeImage      string
	ipFamily       kindv1.ClusterIPFamily
}

// Create a Kubernetes cluster using kind.
func (k *KindClusterProvider) Create(ctx context.Context) {
	Expect(ctx).NotTo(BeNil(), "ctx is required for Create")

	// Sets the kubeconfig path to a temp file.
	// NB. the ClusterProvider is responsible for the cleanup of this file
	f, err := os.CreateTemp("", "e2e-kind")
	Expect(err).ToNot(HaveOccurred(), "Failed to create kubeconfig file for the kind cluster %q", k.name)
	k.kubeconfigPath = f.Name()

	// Creates the kind cluster
	k.createKindCluster()
}

// createKindCluster calls the kind library taking care of passing options for:
// - use a dedicated kubeconfig file (test should not alter the user environment)
// - if required, mount /var/run/docker.sock.
func (k *KindClusterProvider) createKindCluster() {
	kindCreateOptions := []kind.CreateOption{
		kind.CreateWithKubeconfigPath(k.kubeconfigPath),
	}

	cfg := &kindv1.Cluster{
		TypeMeta: kindv1.TypeMeta{
			APIVersion: "kind.x-k8s.io/v1alpha4",
			Kind:       "Cluster",
		},
		Nodes: []kindv1.Node{
			{
				Labels: make(map[string]string),
			},
			{
				Role:   kindv1.WorkerRole,
				Labels: make(map[string]string),
			},
			{
				Role:   kindv1.WorkerRole,
				Labels: make(map[string]string),
			},
		},
	}

	cfg.Nodes[0].Labels["openyurt.io/is-edge-worker"] = "false"
	cfg.Nodes[1].Labels["openyurt.io/is-edge-worker"] = "true"
	cfg.Nodes[2].Labels["openyurt.io/is-edge-worker"] = "true"
	cfg.Nodes[1].Labels["apps.openyurt.io/desired-nodepool"] = "beijing"
	cfg.Nodes[2].Labels["apps.openyurt.io/desired-nodepool"] = "hangzhou"

	if k.ipFamily == kindv1.IPv6Family {
		cfg.Networking.IPFamily = kindv1.IPv6Family
	}
	kindv1.SetDefaultsCluster(cfg)

	if k.withDockerSock {
		setDockerSockConfig(cfg)
	}

	kindCreateOptions = append(kindCreateOptions, kind.CreateWithV1Alpha4Config(cfg))

	nodeImage := fmt.Sprintf("%s:%s", DefaultNodeImageRepository, DefaultNodeImageVersion)
	if k.nodeImage != "" {
		nodeImage = k.nodeImage
	}
	kindCreateOptions = append(kindCreateOptions, kind.CreateWithNodeImage(nodeImage))

	err := kind.NewProvider().Create(k.name, kindCreateOptions...)
	Expect(err).ToNot(HaveOccurred(), "Failed to create the kind cluster %q")
}

// setDockerSockConfig returns a kind config for mounting /var/run/docker.sock into the kind node.
func setDockerSockConfig(cfg *kindv1.Cluster) {
	for _, node := range cfg.Nodes {
		node.ExtraMounts = []kindv1.Mount{
			{
				HostPath:      "/var/run/docker.sock",
				ContainerPath: "/var/run/docker.sock",
			},
		}
	}
}

// GetKubeconfigPath returns the path to the kubeconfig file for the cluster.
func (k *KindClusterProvider) GetKubeconfigPath() string {
	return k.kubeconfigPath
}

// Dispose the kind cluster and its kubeconfig file.
func (k *KindClusterProvider) Dispose(ctx context.Context) {
	Expect(ctx).NotTo(BeNil(), "ctx is required for Dispose")

	if err := kind.NewProvider().Delete(k.name, k.kubeconfigPath); err != nil {
		Logf("Deleting the kind cluster %q failed. You may need to remove this by hand.", k.name)
	}
	if err := os.Remove(k.kubeconfigPath); err != nil {
		Logf("Deleting the kubeconfig file %q file. You may need to remove this by hand.", k.kubeconfigPath)
	}
}

// CreateKindClusterAndLoadImagesInput is the input for CreateKindClusterAndLoadImages.
type CreateKindClusterAndLoadImagesInput struct {
	// Name of the cluster.
	Name string

	// KubernetesVersion of the cluster.
	KubernetesVersion string

	// RequiresDockerSock defines if the cluster requires the docker sock.
	RequiresDockerSock bool

	// Images to be loaded in the cluster.
	Images []string

	// IPFamily is either ipv4 or ipv6. Default is ipv4.
	IPFamily string
}

// CreateKindBootstrapClusterAndLoadImages returns a new Kubernetes cluster with pre-loaded images.
func CreateKindClusterAndLoadImages(ctx context.Context, input CreateKindClusterAndLoadImagesInput) ClusterProvider {
	Expect(ctx).NotTo(BeNil(), "ctx is required for CreateKindBootstrapClusterAndLoadImages")
	Expect(input.Name).ToNot(BeEmpty(), "Invalid argument. Name can't be empty when calling CreateKindBootstrapClusterAndLoadImages")

	Logf("Creating a kind cluster with name %q", input.Name)

	options := []KindClusterOption{}
	if input.KubernetesVersion != "" {
		options = append(options, WithNodeImage(fmt.Sprintf("%s:%s", DefaultNodeImageRepository, input.KubernetesVersion)))
	}
	if input.RequiresDockerSock {
		options = append(options, WithDockerSockMount())
	}
	if input.IPFamily == "IPv6" {
		options = append(options, WithIPv6Family())
	}

	clusterProvider := NewKindClusterProvider(input.Name, options...)
	Expect(clusterProvider).ToNot(BeNil(), "Failed to create a kind cluster")

	clusterProvider.Create(ctx)
	Expect(clusterProvider.GetKubeconfigPath()).To(BeAnExistingFile(), "The kubeconfig file for the kind cluster with name %q does not exists at %q as expected", input.Name, clusterProvider.GetKubeconfigPath())

	Logf("The kubeconfig file for the kind cluster is %s", clusterProvider.kubeconfigPath)

	err := LoadImagesToKindCluster(ctx, LoadImagesToKindClusterInput{
		Name:   input.Name,
		Images: input.Images,
	})
	if err != nil {
		clusterProvider.Dispose(ctx)
		Expect(err).NotTo(HaveOccurred()) // re-surface the error to fail the test
	}

	return clusterProvider
}

// LoadImagesToKindClusterInput is the input for LoadImagesToKindCluster.
type LoadImagesToKindClusterInput struct {
	// Name of the cluster
	Name string

	// Images to be loaded in the cluster (this is kind specific)
	Images []string
}

// LoadImagesToKindCluster provides a utility for loading images into a kind cluster.
func LoadImagesToKindCluster(ctx context.Context, input LoadImagesToKindClusterInput) error {
	if ctx == nil {
		return errors.New("ctx is required for LoadImagesToKindCluster")
	}
	if input.Name == "" {
		return errors.New("Invalid argument. Name can't be empty when calling LoadImagesToKindCluster")
	}

	for _, image := range input.Images {
		Logf("Loading image: %q", image)
		if err := loadImage(ctx, input.Name, image); err != nil {
			return errors.Wrapf(err, "Failed to load image %q into the kind cluster %q", image, input.Name)
		}
	}
	return nil
}

// LoadImage will put a local image onto the kind node.
func loadImage(ctx context.Context, cluster, image string) error {
	// Save the image into a tar
	dir, err := os.MkdirTemp("", "image-tar")
	if err != nil {
		return errors.Wrap(err, "failed to create tempdir")
	}
	defer os.RemoveAll(dir)
	imageTarPath := filepath.Join(dir, "image.tar")

	err = save(ctx, image, imageTarPath)
	if err != nil {
		return err
	}

	// Gets the nodes in the cluster
	provider := kind.NewProvider()
	nodeList, err := provider.ListInternalNodes(cluster)
	if err != nil {
		return err
	}

	// Load the image on the selected nodes
	for _, node := range nodeList {
		if err := load(imageTarPath, node); err != nil {
			return err
		}
	}

	return nil
}

// copied from kind https://github.com/kubernetes-sigs/kind/blob/v0.7.0/pkg/cmd/kind/load/docker-image/docker-image.go#L168
// save saves image to dest, as in `docker save`.
func save(ctx context.Context, image, dest string) error {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return errors.Wrap(err, "failed to get Docker client")
	}

	reader, err := dockerClient.ImageSave(ctx, []string{image})
	if err != nil {
		err = fmt.Errorf("unable to read image data: %v", err)
		return errors.Wrapf(err, "error saving image %q to %q", image, dest)
	}
	defer reader.Close()

	tar, err := os.Create(dest)
	if err != nil {
		err = fmt.Errorf("failed to create destination file %q: %v", dest, err)
		return errors.Wrapf(err, "error saving image %q to %q", image, dest)
	}
	defer tar.Close()

	_, err = io.Copy(tar, reader)
	if err != nil {
		err = fmt.Errorf("failure writing image data to file: %v", err)
		return errors.Wrapf(err, "error saving image %q to %q", image, dest)
	}

	return nil
}

// copied from kind https://github.com/kubernetes-sigs/kind/blob/v0.7.0/pkg/cmd/kind/load/docker-image/docker-image.go#L158
// loads an image tarball onto a node.
func load(imageTarName string, node kindnodes.Node) error {
	f, err := os.Open(filepath.Clean(imageTarName))
	if err != nil {
		return errors.Wrap(err, "failed to open image")
	}
	defer f.Close()
	return kindnodesutils.LoadImageArchive(node, f)
}
