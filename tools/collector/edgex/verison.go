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
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	armVersion            = "arm64"
	fileSearchURLPrefix   = "https://github.com/edgexfoundry/edgex-compose/tree/"
	fileMatchRegexpPrefix = `href="/edgexfoundry/edgex-compose/blob/`
	dirMatchRegexpPrefix  = `href="/edgexfoundry/edgex-compose/tree/`
	rawVersionURLPrefix   = "https://raw.githubusercontent.com/edgexfoundry/edgex-compose/"
	selectedFilePrefix    = "docker-compose"
	selectedFilePrefixArm = "docker-compose-arm64"
	selectedFileSuffix    = ".yml"
	composeBuilder        = "compose-builder"
	envFile               = []string{"common.env", "device-common.env", "common-security.env", "common-sec-stage-gate.env"}
)

type EdgeXConfig struct {
	Versions []*Version `yaml:"versions"`
}

type Version struct {
	logger     *logrus.Entry
	env        map[string]string
	Name       string             `yaml:"versionName"`
	ConfigMaps []corev1.ConfigMap `yaml:"configMaps,omitempty"`
	Components []*Component       `yaml:"components,omitempty"`
}

func newVersion(logger *logrus.Entry, name string) *Version {
	return &Version{
		logger:     logger.WithField("version", name),
		Name:       name,
		ConfigMaps: []corev1.ConfigMap{},
		Components: []*Component{},
		env:        make(map[string]string),
	}
}

func newEdgeXConfig() *EdgeXConfig {
	edgeXConfig := &EdgeXConfig{
		Versions: make([]*Version, 0),
	}
	return edgeXConfig
}

func (v *Version) catch(isSecurity bool, arch string) error {
	logger := v.logger
	logger.Infoln("Start catching, version name:", v.Name)

	filenames, err := v.catchAllFilenames()
	if err != nil {
		return err
	}

	if ok := v.checkVersion(filenames); !ok {
		logger.Warningln("The current version cannot be adapted,", "version name:", v.Name)
		return ErrVersionNotAdapted
	}

	err = v.addEnv(isSecurity)
	if err != nil {
		return err
	}

	filename, ok := v.pickupFile(filenames, isSecurity, arch)
	if !ok {
		logger.Warningln("Configuration file is not found,", "version name:", v.Name)
		return ErrConfigFileNotFound
	}

	err = v.catchYML(filename)
	if err != nil {
		return err
	}

	return nil
}

func (v *Version) newComponent(name, image string) *Component {
	return &Component{
		logger:         v.logger.WithField("component", name),
		Name:           name,
		image:          image,
		volumes:        []corev1.Volume{},
		volumeMounts:   []corev1.VolumeMount{},
		servicePorts:   []corev1.ServicePort{},
		containerPorts: []corev1.ContainerPort{},
		componentEnv:   make(map[string]string),
		envRef:         &v.env,
		configmapsRef:  &v.ConfigMaps,
	}
}

func (v *Version) addEnv(isSecurity bool) error {
	logger := v.logger
	var env []string
	if isSecurity {
		env = envFile
	} else {
		env = envFile[0:2]
	}
	for _, file := range env {
		url := rawVersionURLPrefix + v.Name + "/" + composeBuilder + "/" + file
		envs, err := loadEnv(logger, url)
		if err != nil {
			logger.Errorln("Fail to load env:", err)
			return err
		}

		for key, value := range envs {
			v.env[key] = value
		}
	}
	return nil
}

func (v *Version) catchYML(filename string) error {
	logger := v.logger
	versionURL := rawVersionURLPrefix + v.Name + "/" + filename

	pageStr, err := getPage(logger, versionURL)
	if err != nil {
		return err
	}

	project, err := getProject(filename, pageStr)
	if err != nil {
		return err
	}

	v.handleConfigmap()

	for _, rawComponent := range project.Services {
		// Get the hostname and image information to create the component as basic information
		hostname := rawComponent.Hostname
		image := rawComponent.Image
		component := v.newComponent(hostname, image)

		// Collect information for each component
		component.addEnv(rawComponent.Environment)
		component.fillTmpfs(rawComponent.Tmpfs)
		component.fillVolumes(rawComponent.Volumes)
		component.fillPorts(rawComponent.Ports)

		component.handleService()
		component.handleDeployment()

		v.Components = append(v.Components, component)
	}
	return nil
}

func (v *Version) catchAllFilenames() ([]string, error) {
	logger := v.logger

	fileSearchURL := fileSearchURLPrefix + v.Name
	fileMatchRegexp := fileMatchRegexpPrefix + v.Name + `/(.*?)"`
	dirMatchRegexp := dirMatchRegexpPrefix + v.Name + `/(.*?)"`

	results, err := getPageWithRegex(v.logger, fileSearchURL, fileMatchRegexp)
	if err != nil {
		logger.Errorln("Fail to list all filename:", err)
		return nil, err
	}

	dirResults, err := getPageWithRegex(v.logger, fileSearchURL, dirMatchRegexp)
	if err != nil {
		logger.Errorln("Fail to list all directory:", err)
		return nil, err
	}
	results = append(results, dirResults...)

	return results, nil
}

func (v *Version) checkVersion(filenames []string) bool {
	for _, filename := range filenames {
		if filename == composeBuilder {
			return true
		}
	}
	return false
}

func (v *Version) pickupFile(filenames []string, isSecurity bool, arch string) (string, bool) {
	matchFile := selectedFilePrefix
	matchFileWithVersion := matchFile + "-" + v.Name

	matchFileArm := selectedFilePrefixArm
	matchFileWithVersionArm := matchFileArm + "-" + v.Name
	matchFileWithVer := selectedFilePrefix + "-" + v.Name + "-arm64"
	if !isSecurity {
		matchFile += "-no-secty"
		matchFileWithVersion += "-no-secty"
	}
	if arch == "amd" {
		matchFile += selectedFileSuffix
		matchFileWithVersion += selectedFileSuffix
		// match the configuration file with the version name or the configuration file named "docker-compose"
		for _, filename := range filenames {
			if filename == matchFile || filename == matchFileWithVersion {
				return filename, true
			}
		}
	} else if arch == "arm" {
		matchFileArm += selectedFileSuffix
		matchFileWithVersionArm += selectedFileSuffix
		matchFileWithVer += selectedFileSuffix
		// match the configuration file with the version name or the configuration file named "docker-compose-arm"
		for _, filename := range filenames {
			if filename == matchFileArm || filename == matchFileWithVersionArm || filename == matchFileWithVer {
				return filename, true
			}
		}
	}

	return "", false
}

func (v *Version) handleConfigmap() {
	configmap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Labels: make(map[string]string),
			Name:   "common-variable-" + v.Name,
		},
		Data: make(map[string]string),
	}

	// Deal with special circumstances
	for _, vf := range versionSpecialHandlers {
		vf(v)
	}

	for k, v := range v.env {
		configmap.Data[k] = v
	}
	v.ConfigMaps = append(v.ConfigMaps, configmap)
}
