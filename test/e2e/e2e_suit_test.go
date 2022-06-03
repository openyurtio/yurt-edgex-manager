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

package e2e

import (
	"context"
	"flag"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openyurtio/yurt-edgex-manager/test/framework"
	"github.com/openyurtio/yurt-edgex-manager/test/framework/clustersetup"
	"sigs.k8s.io/yaml"
)

// Test suite flags
var (
	// configPath is the path to the e2e config file.
	configPath string

	// artifactFolder is the folder to store e2e test artifacts.
	artifactFolder string

	// skipCleanup prevents cleanup of test resources e.g. for debug purposes.
	skipCleanup bool
)

// Test suite global vars
var (
	ctx = context.Background()

	// e2eConfig to be used for this test, read from configPath.
	e2eConfig *framework.E2EConfig

	// TestBedCluster the cluster to be used for the e2e tests.
	TestBedCluster clustersetup.ClusterProvider
)

func init() {
	flag.StringVar(&configPath, "e2e.config", "", "path to the e2e config file")
	flag.StringVar(&artifactFolder, "e2e.artifacts-folder", "", "folder where e2e test artifact should be stored")
	flag.BoolVar(&skipCleanup, "e2e.skip-resource-cleanup", false, "if true, the resource cleanup after tests will be skipped")
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "yurt-edgex-manager Suite")
}

// Using a SynchronizedBeforeSuite for controlling how to create resources shared across ParallelNodes (~ginkgo threads).
var _ = SynchronizedBeforeSuite(func() []byte {
	// Before all ParallelNodes.

	Expect(configPath).To(BeAnExistingFile(), "Invalid test suite argument. e2e.config should be an existing file.")
	Expect(os.MkdirAll(artifactFolder, 0755)).To(Succeed(), "Invalid test suite argument. Can't create e2e.artifacts-folder %q", artifactFolder)

	Byf("Loading the e2e test configuration from %q", configPath)
	e2eConfig = loadE2EConfig(configPath)

	By("Setting up the bootstrap cluster")
	TestBedCluster = setupBootstrapCluster(e2eConfig)

	return nil
}, func(data []byte) {
	// Before each ParallelNode.
})

// Using a SynchronizedAfterSuite for controlling how to delete resources shared across ParallelNodes (~ginkgo threads).
// The bootstrap cluster is shared across all the tests, so it should be deleted only after all ParallelNodes completes.
var _ = SynchronizedAfterSuite(func() {
	// After each ParallelNode.
}, func() {
	// After all ParallelNodes.

	if !skipCleanup && TestBedCluster != nil {
		By("Tearing down the cluster testbed")
		TestBedCluster.Dispose(ctx)
	}
})

func loadE2EConfig(configPath string) *framework.E2EConfig {
	configData, err := os.ReadFile(configPath)
	Expect(err).ToNot(HaveOccurred(), "Failed to read the e2e test config file")
	Expect(configData).ToNot(BeEmpty(), "The e2e test config file should not be empty")

	config := &framework.E2EConfig{}
	Expect(yaml.Unmarshal(configData, config)).To(Succeed(), "Failed to convert the e2e test config file to yaml")

	config.Defaults()

	Expect(config.Validate()).To(Succeed(), "The e2e test config file is not valid")

	Expect(config).ToNot(BeNil(), "Failed to load E2E config from %s", configPath)

	return config
}

func setupBootstrapCluster(config *framework.E2EConfig) clustersetup.ClusterProvider {
	var cluster clustersetup.ClusterProvider

	cluster = clustersetup.CreateKindClusterAndLoadImages(ctx, clustersetup.CreateKindClusterAndLoadImagesInput{
		Name:               config.ManagementClusterName,
		KubernetesVersion:  config.GetVariable(KubernetesVersionManagement),
		RequiresDockerSock: true,
		Images:             config.Images,
		IPFamily:           config.GetVariable(IPFamily),
	})
	Expect(cluster).ToNot(BeNil(), "Failed to create a bootstrap cluster")

	kubeconfigPath := cluster.GetKubeconfigPath()
	Expect(kubeconfigPath).To(BeAnExistingFile(), "Failed to get the kubeconfig file for the bootstrap cluster")

	return cluster
}
