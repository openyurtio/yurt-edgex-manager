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
	"os"
	"time"

	. "github.com/onsi/gomega"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

// Provides access to the configuration for an e2e test.

// LoadE2EConfig loads the configuration for the e2e test environment.
func LoadE2EConfig(ctx context.Context, configPath string) *E2EConfig {
	configData, err := os.ReadFile(configPath)
	Expect(err).ToNot(HaveOccurred(), "Failed to read the e2e test config file")
	Expect(configData).ToNot(BeEmpty(), "The e2e test config file should not be empty")

	config := &E2EConfig{}
	Expect(yaml.Unmarshal(configData, config)).To(Succeed(), "Failed to convert the e2e test config file to yaml")

	config.Defaults()

	Expect(config.Validate()).To(Succeed(), "The e2e test config file is not valid")

	return config
}

// E2EConfig defines the configuration of an e2e test environment.
type E2EConfig struct {
	// Name is the name of the Kind management cluster.
	// Defaults to test-[random generated suffix].
	ManagementClusterName string `json:"managementClusterName,omitempty"`

	// Images is a list of container images to load into the Kind cluster.
	Images []string `json:"images,omitempty"`

	// Variables to be added to the clusterctl config file
	// Please note that clusterctl read variables from OS environment variables as well, so you can avoid to hard code
	// sensitive data in the config file.
	Variables map[string]string `json:"variables,omitempty"`

	// Intervals to be used for long operations during tests
	Intervals map[string][]string `json:"intervals,omitempty"`
}

// Defaults assigns default values to the object. More specifically:
// - ManagementClusterName gets a default name if empty.
// - Providers version gets type KustomizeSource if not otherwise specified.
// - Providers file gets targetName = sourceName if not otherwise specified.
// - Images gets LoadBehavior = MustLoadImage if not otherwise specified.
func (c *E2EConfig) Defaults() {
	if c.ManagementClusterName == "" {
		c.ManagementClusterName = fmt.Sprintf("test-%s", RandomString(6))
	}
}

func errInvalidArg(format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	return errors.Errorf("invalid argument: %s", msg)
}

func errEmptyArg(argName string) error {
	return errInvalidArg("%s is empty", argName)
}

// Validate validates the configuration. More specifically:
// - ManagementClusterName should not be empty.
// - There should be one CoreProvider (cluster-api), one BootstrapProvider (kubeadm), one ControlPlaneProvider (kubeadm).
// - There should be one InfraProvider (pick your own).
// - Image should have name and loadBehavior be one of [mustload, tryload].
// - Intervals should be valid ginkgo intervals.
func (c *E2EConfig) Validate() error {
	// ManagementClusterName should not be empty.
	if c.ManagementClusterName == "" {
		return errEmptyArg("ManagementClusterName")
	}

	// Image should have name and loadBehavior be one of [mustload, tryload].
	for i, containerImage := range c.Images {
		if containerImage == "" {
			return errEmptyArg(fmt.Sprintf("Images[%d].Name=%q", i, containerImage))
		}
	}

	// Intervals should be valid ginkgo intervals.
	for k, intervals := range c.Intervals {
		switch len(intervals) {
		case 0:
			return errInvalidArg("Intervals[%s]=%q", k, intervals)
		case 1, 2:
		default:
			return errInvalidArg("Intervals[%s]=%q", k, intervals)
		}
		for _, i := range intervals {
			if _, err := time.ParseDuration(i); err != nil {
				return errInvalidArg("Intervals[%s]=%q", k, intervals)
			}
		}
	}
	return nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// GetIntervals returns the intervals to be applied to a Eventually operation.
// It searches for [spec]/[key] intervals first, and if it is not found, it searches
// for default/[key]. If also the default/[key] intervals are not found,
// ginkgo DefaultEventuallyTimeout and DefaultEventuallyPollingInterval are used.
func (c *E2EConfig) GetIntervals(spec, key string) []interface{} {
	intervals, ok := c.Intervals[fmt.Sprintf("%s/%s", spec, key)]
	if !ok {
		if intervals, ok = c.Intervals[fmt.Sprintf("default/%s", key)]; !ok {
			return nil
		}
	}
	intervalsInterfaces := make([]interface{}, len(intervals))
	for i := range intervals {
		intervalsInterfaces[i] = intervals[i]
	}
	return intervalsInterfaces
}

func (c *E2EConfig) HasVariable(varName string) bool {
	if _, ok := os.LookupEnv(varName); ok {
		return true
	}

	_, ok := c.Variables[varName]
	return ok
}

// GetVariable returns a variable from environment variables or from the e2e config file.
func (c *E2EConfig) GetVariable(varName string) string {
	if value, ok := os.LookupEnv(varName); ok {
		return value
	}

	value, ok := c.Variables[varName]
	Expect(ok).NotTo(BeFalse())
	return value
}
