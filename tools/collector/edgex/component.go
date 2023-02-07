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

	"github.com/compose-spec/compose-go/types"
	"github.com/sirupsen/logrus"
)

const (
	parseIntBase        = 10
	parseIntBaseBitSize = 32
)

type Component struct {
	logger       *logrus.Entry
	Name         string            `yaml:"name"`
	Image        string            `yaml:"image"`
	Volumes      []Volume          `yaml:"volumes,omitempty"`
	Ports        []Port            `yaml:"ports,omitempty"`
	ComponentEnv map[string]string `yaml:"componentEnv,omitempty"`
	// A pointer to the Env of the previous level
	envRef *map[string]string
}

type Volume struct {
	Name      string `yaml:"name"`
	HostPath  string `yaml:"hostPath"`
	MountPath string `yaml:"mountPath"`
}

type Port struct {
	Protocol   string `yaml:"protocol"`
	Port       int32  `yaml:"port"`
	TargetPort int32  `yaml:"targetPort"`
	NodePort   int32  `yaml:"nodePort,omitempty"`
}

func (c *Component) addEnv(envs map[string]*string) {
	for key, v := range envs {
		if _, ok := (*c.envRef)[key]; !ok {
			c.ComponentEnv[key] = *v
		}
	}
}

const (
	volumesSplitMinLen        = 2
	anonymousVolumeNamePrefix = "anonymous-volume"
)

func (c *Component) fillVolumes(volumes []types.ServiceVolumeConfig) {
	_ = c.logger
	for _, v := range volumes {
		var volume Volume
		switch v.Type {
		case "volume":
			// Like this value: edgex-init:/edgex-init:ro,z
			volume = Volume{
				Name:      v.Source,
				HostPath:  "",
				MountPath: v.Target,
			}
		case "bind":
			// Like this value: /var/run/docker.sock:/var/run/docker.sock:z
			volume = Volume{
				Name:      "",
				HostPath:  v.Source,
				MountPath: v.Target,
			}
		}
		c.Volumes = append(c.Volumes, volume)
	}
}

func (c *Component) fillTmpfs(tmpfs types.StringList) {
	logger := c.logger
	for _, tmpfsStr := range tmpfs {
		if tmpfsStr == "" {
			logger.Warningln("This is not a valid tmpfs", "value:", tmpfsStr)
			continue
		}
		volume := Volume{
			Name: "",
			// For tmpfs, we should set it to emptyDir
			// to prevent legacy configurations from being read when the component restarts
			HostPath:  "",
			MountPath: tmpfsStr,
		}
		c.Volumes = append(c.Volumes, volume)
	}
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

func (c *Component) fillPorts(ports []types.ServicePortConfig) {
	logger := c.logger
	for _, v := range ports {
		hostPort, err := strconv.ParseInt(v.Published, 10, 32)
		if err != nil {
			logger.Warningln("This is not a valid HostPort", "value", v.HostIP)
			continue
		}
		port := Port{
			Protocol:   strings.ToUpper(v.Protocol),
			Port:       int32(hostPort),
			TargetPort: int32(v.Target),
		}
		c.Ports = append(c.Ports, port)
	}
}
