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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	parseIntBase        = 10
	parseIntBaseBitSize = 32
)

type Component struct {
	logger       *logrus.Entry
	Name         string                 `yaml:"name"`
	Service      *corev1.ServiceSpec    `yaml:"service,omitempty"`
	Deployment   *appsv1.DeploymentSpec `yaml:"deployment,omitempty"`
	componentEnv map[string]string
	// A pointer to the Env of the previous level
	envRef         *map[string]string
	configmapsRef  *[]corev1.ConfigMap
	image          string
	volumes        []corev1.Volume
	volumeMounts   []corev1.VolumeMount
	servicePorts   []corev1.ServicePort
	containerPorts []corev1.ContainerPort
}

func (c *Component) addEnv(envs map[string]*string) {
	// Deal with special circumstances

	for _, cf := range componentSpecialHandlers {
		cf(c)
	}

	for key, v := range envs {
		if _, ok := (*c.envRef)[key]; !ok {
			c.componentEnv[key] = *v
		}
	}
}

const (
	volumesSplitMinLen        = 2
	anonymousVolumeNamePrefix = "anonymous-volume"
	tmpfsVolumeNamePrefix     = "tmpfs-volume"
)

var HostPathType = corev1.HostPathDirectoryOrCreate

func (c *Component) fillVolumes(volumes []types.ServiceVolumeConfig) {
	_ = c.logger
	count := 1
	for _, v := range volumes {
		var volume corev1.Volume
		var volumeMount corev1.VolumeMount
		switch v.Type {
		case "volume":
			// Like this value: edgex-init:/edgex-init:ro,z
			name := v.Source
			volume = corev1.Volume{
				Name: name,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			}
			volumeMount = corev1.VolumeMount{
				Name:      name,
				MountPath: v.Target,
			}
		case "bind":
			// Like this value: /var/run/docker.sock:/var/run/docker.sock:z
			name := anonymousVolumeNamePrefix + strconv.FormatInt(int64(count), formatIntBase)
			count++
			volume = corev1.Volume{
				Name: name,
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: v.Source,
						Type: &HostPathType,
					},
				},
			}
			volumeMount = corev1.VolumeMount{
				Name:      name,
				MountPath: v.Target,
			}
		}
		c.volumes = append(c.volumes, volume)
		c.volumeMounts = append(c.volumeMounts, volumeMount)
	}
}

func (c *Component) fillTmpfs(tmpfs types.StringList) {
	logger := c.logger
	count := 1
	for _, tmpfsStr := range tmpfs {
		if tmpfsStr == "" {
			logger.Warningln("This is not a valid tmpfs", "value:", tmpfsStr)
			continue
		}

		name := tmpfsVolumeNamePrefix + strconv.FormatInt(int64(count), formatIntBase)
		count++
		volume := corev1.Volume{
			Name: name,
			// For tmpfs, we should set it to emptyDir
			// to prevent legacy configurations from being read when the component restarts
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
		volumeMount := corev1.VolumeMount{
			Name:      name,
			MountPath: tmpfsStr,
		}
		c.volumes = append(c.volumes, volume)
		c.volumeMounts = append(c.volumeMounts, volumeMount)
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

		name := strings.ToLower(v.Protocol) + "-" + strconv.FormatInt(int64(hostPort), 10)
		servicePort := corev1.ServicePort{
			Name:       name,
			Protocol:   corev1.Protocol(strings.ToUpper(v.Protocol)),
			Port:       int32(hostPort),
			TargetPort: intstr.FromInt(int(v.Target)),
		}
		containerPort := corev1.ContainerPort{
			Name:          name,
			Protocol:      corev1.Protocol(strings.ToUpper(v.Protocol)),
			ContainerPort: int32(v.Target),
		}
		c.servicePorts = append(c.servicePorts, servicePort)
		c.containerPorts = append(c.containerPorts, containerPort)
	}
}

func (c *Component) handleService() {
	if len(c.servicePorts) > 0 {
		c.Service = &corev1.ServiceSpec{
			Selector: map[string]string{"app": c.Name},
			Ports:    c.servicePorts,
		}
	} else {
		c.Service = nil
	}
}

func (c *Component) handleDeployment() {
	envs := []corev1.EnvVar{}
	for k, v := range c.componentEnv {
		envs = append(envs, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	efss := []corev1.EnvFromSource{}
	for _, configmap := range *c.configmapsRef {
		efs := corev1.EnvFromSource{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: configmap.Name,
				},
			},
		}
		efss = append(efss, efs)
	}

	container := corev1.Container{
		Name:            c.Name,
		Image:           c.image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Ports:           c.containerPorts,
		VolumeMounts:    c.volumeMounts,
		EnvFrom:         efss,
		Env:             envs,
	}

	c.Deployment = &appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": c.Name},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"app": c.Name},
			},
			Spec: corev1.PodSpec{
				Volumes:    c.volumes,
				Containers: []corev1.Container{container},
				Hostname:   c.Name,
			},
		},
	}
}
