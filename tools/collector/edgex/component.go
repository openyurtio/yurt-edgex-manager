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
	parseIntBase        = 10
	parseIntBaseBitSize = 32
)

type Component struct {
	logger       *logrus.Entry
	Name         string            `yaml:"name"`
	Image        string            `yaml:"image"`
	Volumes      []Volume          `yaml:"volumns,omitempty"`
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
			c.ComponentEnv[key] = value
		}
	}
}

const (
	volumesSplitMinLen        = 2
	anonymousVolumeNamePrefix = "anonymous-volume"
)

func (c *Component) fillVolumes(volumes []interface{}) {
	logger := c.logger
	for _, v := range volumes {
		volumeStr, ok := v.(string)
		if volumeStr == "" || !ok {
			logger.Warningln("This is not a valid volume", "value:", v)
			continue
		}
		infos := strings.Split(volumeStr, ":")
		if len(infos) < volumesSplitMinLen {
			logger.Warningln("This is not a valid volume", "value:", v)
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
			// Like this value: edgex-init:/edgex-init:ro,z
			volume := Volume{
				Name: infos[0],
				// For non-mapped volumes, we should set it to emptyDir
				// to prevent legacy configurations from being read when the component restarts
				HostPath:  "",
				MountPath: infos[1],
			}
			c.Volumes = append(c.Volumes, volume)
		}
	}
	c.repairVolumes()
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

const (
	portsSplitIgnoreProtocol = 1
	portsSplitWithProtocol   = 2

	reflectSamePort       = 1
	reflectPortOutCluster = 2
	reflectPortInCluster  = 3
)

func (c *Component) fillPorts(ports []interface{}) {
	logger := c.logger
	for _, v := range ports {
		portStr, ok := v.(string)
		if portStr == "" || !ok {
			logger.Warningln("This is not a valid port", "value:", v)
			continue
		}

		port := Port{}

		// Parse protocol and supplement default tcp protocol
		portAndProtocol := strings.Split(portStr, "/")
		if len(portAndProtocol) == portsSplitIgnoreProtocol {
			port.Protocol = "TCP"
		} else if len(portAndProtocol) == portsSplitWithProtocol {
			port.Protocol = strings.ToUpper(portAndProtocol[portsSplitWithProtocol-1])
		} else {
			logger.Warningln("This is not a valid port", "value:", v)
			continue
		}

		portInfo := strings.Split(portAndProtocol[0], ":")

		if len(portInfo) == reflectSamePort {
			portNum, err := strconv.ParseInt(portInfo[0], parseIntBase, parseIntBaseBitSize)
			if err != nil {
				logger.Warningln("This is not a valid port", "value:", v)
				continue
			}
			port.Port = int32(portNum)
			port.TargetPort = int32(portNum)
		} else if len(portInfo) == reflectPortOutCluster {
			portNumInCluster, err := strconv.ParseInt(portInfo[0], parseIntBase, parseIntBaseBitSize)
			if err != nil {
				logger.Warningln("This is not a valid port", "value:", v)
				continue
			}
			portNumOutCluster, err := strconv.ParseInt(portInfo[1], parseIntBase, parseIntBaseBitSize)
			if err != nil {
				logger.Warningln("This is not a valid port", "value:", v)
				continue
			}
			port.Port = int32(portNumInCluster)
			port.TargetPort = int32(portNumInCluster)
			port.NodePort = int32(portNumOutCluster)
		} else if len(portInfo) == reflectPortInCluster {
			portNumPort, err := strconv.ParseInt(portInfo[1], parseIntBase, parseIntBaseBitSize)
			if err != nil {
				logger.Warningln("This is not a valid port", "value:", v)
				continue
			}
			portNumTargetPort, err := strconv.ParseInt(portInfo[2], parseIntBase, parseIntBaseBitSize)
			if err != nil {
				logger.Warningln("This is not a valid port", "value:", v)
				continue
			}
			port.Port = int32(portNumPort)
			port.TargetPort = int32(portNumTargetPort)
		} else {
			logger.Warningln("This is not a valid port", "value:", v)
			continue
		}
		c.Ports = append(c.Ports, port)
	}
}
