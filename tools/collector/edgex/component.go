/*
Copyright 2022.

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

package edgex

import (
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	volumesSplitMinLen        = 2
	anonymousVolumeNamePrefix = "anonymous_volume"
)

type Component struct {
	logger       *logrus.Logger
	Name         string            `yaml:"name"`
	Image        string            `yaml:"image"`
	Volumes      []Volume          `yaml:"volumns,omitempty"`
	ComponentEnv map[string]string `yaml:"componentEnv,omitempty"`
	// TODO: We need to crawl another no-security file and mark which components are not secure
	IsSecurity bool `yaml:"isSecurity"`
	// A pointer to the Env of the previous level
	envRef *map[string]string
}

type Volume struct {
	Name      string `yaml:"name"`
	HostPath  string `yaml:"hostPath"`
	MountPath string `yaml:"mountPath"`
}

func (c *Component) addEnv(envs map[string]interface{}) {
	for key, v := range envs {
		var value string
		key = strings.ToUpper(key)
		switch rawValue := v.(type) {
		case int:
			value = strconv.FormatInt(int64(rawValue), formatIntBase)
		case string:
			value = rawValue
		}
		if _, ok := (*c.envRef)[key]; !ok {
			unifyPort(&key, &value)
			c.ComponentEnv[key] = value
		}
	}
}

func (c *Component) fillVolumes(volumes []interface{}) {
	logger := c.logger
	for _, v := range volumes {
		volumeStr, ok := v.(string)
		if volumeStr == "" || !ok {
			logger.Warningln("This is not a valid volume", "volume:", v)
			continue
		}
		infos := strings.Split(volumeStr, ":")
		if len(infos) < volumesSplitMinLen {
			logger.Warningln("This is not a valid volume", "volume:", v)
			continue
		}
		if volumeStr[0] == '/' {
			// Like this value: /var/run/docker.sock:/var/run/docker.sock:z
			volume := Volume{
				Name:      "",
				HostPath:  infos[0],
				MountPath: infos[1],
			}
			c.Volumes = append(c.Volumes, volume)
		} else {
			// edgex-init:/edgex-init:ro,z
			volume := Volume{
				Name:      infos[0],
				HostPath:  infos[1],
				MountPath: infos[1],
			}
			c.Volumes = append(c.Volumes, volume)
		}
	}
	c.repairVolumes()
}

func (c *Component) repairPorts() {
	repairPort(&c.ComponentEnv)
}

func (c *Component) repairVolumes() {
	count := 1
	for i := range c.Volumes {
		if c.Volumes[i].Name == "" {
			c.Volumes[i].Name = anonymousVolumeNamePrefix + strconv.FormatInt(int64(count), formatIntBase)
			count++
		}
	}
}
