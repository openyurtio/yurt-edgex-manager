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
	"bytes"
	"strings"

	"github.com/go-logr/logr"
	"github.com/spf13/viper"
)

var (
	armVersion            = "arm64"
	fileSearchURLPrefix   = "https://github.com/edgexfoundry/edgex-compose/tree/"
	fileMatchRegexpPrefix = `href="/edgexfoundry/edgex-compose/blob/`
	dirMatchRegexpPrefix  = `href="/edgexfoundry/edgex-compose/tree/`
	rawVersionURLPrefix   = "https://raw.githubusercontent.com/edgexfoundry/edgex-compose/"
	selectedFilePrefix    = "docker-compose"
	selectedFileSuffix    = ".yml"
	composeBuilder        = "compose-builder"
	envFile               = []string{"common.env", "common-security.env", "common-sec-stage-gate.env", "device-common.env"}
)

type EdgeXConfig struct {
	UnifiedPort uint      `yaml:"unifiedPort"`
	Versions    []Version `yaml:"versions"`
}

type Version struct {
	logger     logr.Logger
	Name       string            `yaml:"versionName"`
	Env        map[string]string `yaml:"env,omitempty"`
	Components []Component       `yaml:"components,omitempty"`
}

func newVersion(logger logr.Logger, name string) *Version {
	return &Version{
		logger:     logger.WithName(name),
		Name:       name,
		Env:        make(map[string]string),
		Components: make([]Component, 0),
	}
}

func newEdgeXConfig() *EdgeXConfig {
	edgeXConfig := &EdgeXConfig{
		UnifiedPort: UnifiedPort,
		Versions:    make([]Version, 0),
	}
	return edgeXConfig
}

func (v *Version) catch() error {
	logger := v.logger
	logger.Info("Start catching", "version name", v.Name)

	filenames, err := v.catchAllFilenames()
	if err != nil {
		return err
	}

	if ok := v.checkVersion(filenames); !ok {
		logger.Info("The current version cannot be adapted", "version name", v.Name)
		return ErrVersionNotAdapted
	}

	err = v.addEnv()
	if err != nil {
		return err
	}

	filename, ok := v.pickupFile(filenames)
	if !ok {
		logger.Info("Configuration file is not found", "version name", v.Name)
		return ErrConfigFileNotFound
	}

	err = v.catchYML(filename)
	if err != nil {
		return err
	}

	v.repairPorts()

	return nil
}

func (v *Version) newComponent(name, image string) *Component {
	return &Component{
		Name:         name,
		Image:        image,
		Volumes:      make([]string, 0),
		ComponentEnv: make(map[string]string),
		IsSecurity:   true,
		envRef:       &v.Env,
	}
}

func (v *Version) addEnv() error {
	logger := v.logger

	for _, file := range envFile {
		url := rawVersionURLPrefix + v.Name + "/" + composeBuilder + "/" + file
		envs, err := loadEnv(logger, url)
		if err != nil {
			logger.Error(err, "Fail to load env")
			return err
		}

		for key, value := range envs {
			unifyPort(&key, &value)
			v.Env[key] = value
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

	viper.SetConfigType("yaml")
	err = viper.ReadConfig(bytes.NewBuffer([]byte(pageStr)))
	if err != nil {
		logger.Error(err, "Viper read config error")
		return err
	}

	components := viper.Get("services")
	for key, rawComponent := range components.(map[string]interface{}) {
		componentConfig := rawComponent.(map[string]interface{})
		// HACK: Some components do not have a hostname, need to check this problem.
		hostname, ok := componentConfig["hostname"].(string)
		if !ok {
			hostname = key
		}

		image, ok := componentConfig["image"].(string)
		if !ok {
			logger.Info("This is not a valid component", "component", hostname)
			continue
		}

		component := v.newComponent(hostname, image)
		envs, ok := componentConfig["environment"].(map[string]interface{})
		if ok {
			component.addEnv(envs)
		}

		volumes, ok := componentConfig["volumes"].([]interface{})
		if ok {
			_ = component.fillVolumes(volumes)
		}

		v.Components = append(v.Components, *component)
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
		logger.Error(err, "Fail to list all filename")
		return nil, err
	}

	dirResults, err := getPageWithRegex(v.logger, fileSearchURL, dirMatchRegexp)
	if err != nil {
		logger.Error(err, "Fail to list all directory")
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

func (v *Version) pickupFile(filenames []string) (string, bool) {
	// Fisrt match the configuration file with the version name
	for _, filename := range filenames {
		if strings.Contains(filename, selectedFilePrefix+"-"+v.Name+selectedFileSuffix) && !strings.Contains(filename, armVersion) {
			return filename, true
		}
	}

	// Then match the configuration file named "docker-compose"
	for _, filename := range filenames {
		if strings.Contains(filename, selectedFilePrefix+selectedFileSuffix) && !strings.Contains(filename, armVersion) {
			return filename, true
		}
	}
	return "", false
}

func (v *Version) repairPorts() {
	repairPort(&v.Env)
	for _, component := range v.Components {
		component.repairPorts()
	}
}
