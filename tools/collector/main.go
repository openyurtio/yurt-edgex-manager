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

package main

import (
	"flag"
	"io/ioutil"
	"os"

	"github.com/openyurtio/yurt-edgex-manager/tools/collector/edgex"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

var (
	collectLog             = logrus.New()
	saveSectyConfigPath    = "../../EdgeXConfig/config.yaml"
	saveSectyConfigPathArm = "../../EdgeXConfig/config.yaml"
	saveNoSectyConfigPath  = "../../EdgeXConfig/config-nosecty.yaml"
	debug                  bool
	repo                   string
	amdArch                = "amd"
	armArch                = "arm"
	manifestPath           = "../../EdgeXConfig/manifest.yaml"
)

func main() {
	flag.BoolVar(&debug, "debug", false, "Start debug module")
	flag.UintVar(&edgex.UnifiedPort, "unified-port", 2000, "Unify ports of the edgex component")
	flag.StringVar(&repo, "repo", "openyurt", "repository name")

	flag.Parse()

	if debug {
		collectLog.SetLevel(logrus.DebugLevel)
	} else {
		collectLog.SetLevel(logrus.InfoLevel)
	}

	err := Run()
	if err != nil {
		collectLog.Errorln("Fail to collect edgex configuration:", err)
		return
	}
}

// Collect the edgex configuration and write it to the yaml file
func Run() error {
	logger := collectLog

	// Collect security version
	edgex.SetLog(logger.WithField("collect", "edgex"))

	versionsInfo, err := edgex.CollectVersionsInfo()
	if err != nil {
		return err
	}

	edgeXConfigAmd, err := edgex.CollectEdgeXConfig(versionsInfo, true, amdArch)
	if err != nil {
		return err
	}

	edgeXConfigArm, err := edgex.CollectEdgeXConfig(versionsInfo, true, armArch)

	err = edgex.CollectImages(edgeXConfigAmd, edgeXConfigArm)
	if err != nil {
		return err
	}

	edgex.ModifyImagesName(edgeXConfigAmd, repo)

	var oldManifest edgex.Manifest

	if _, err := os.Stat(manifestPath); err == nil {
		//file is exist
		manifestFile, err := ioutil.ReadFile(manifestPath)
		err = yaml.Unmarshal(manifestFile, &oldManifest)
		if err != nil {
			return err
		}
	} else {
		oldManifest = *edgex.NewManifest()
	}

	manifest := edgex.CollectVersionToManifest(edgeXConfigAmd.Versions, &oldManifest)

	manifestData, err := yaml.Marshal(manifest)
	if err != nil {
		logger.Errorln("Fail to parse report config to yaml:", err)
		return err
	}
	err = ioutil.WriteFile(manifestPath, manifestData, 0644)
	if err != nil {
		logger.Errorln("Fail to write report yaml:", err)
		return err
	}

	data, err := yaml.Marshal(edgeXConfigAmd)
	if err != nil {
		logger.Errorln("Fail to parse edgex config to yaml:", err)
		return err
	}

	err = ioutil.WriteFile(saveSectyConfigPath, data, 0644)
	if err != nil {
		logger.Errorln("Fail to write config yaml:", err)
		return err
	}

	// Collect no-security version
	edgex.SetLog(logger.WithField("collect", "edgex-nosecty"))

	edgeXConfigAmd, err = edgex.CollectEdgeXConfig(versionsInfo, false, amdArch)
	if err != nil {
		return err
	}

	edgex.ModifyImagesName(edgeXConfigAmd, repo)

	if manifest.Updated == "false" {
		manifest = edgex.CollectVersionToManifest(edgeXConfigAmd.Versions, &oldManifest)

		manifestData, err = yaml.Marshal(manifest)
		if err != nil {
			logger.Errorln("Fail to parse report config to yaml:", err)
			return err
		}
		err = ioutil.WriteFile(manifestPath, manifestData, 0644)
		if err != nil {
			logger.Errorln("Fail to write report yaml:", err)
			return err
		}
	}

	data, err = yaml.Marshal(edgeXConfigAmd)
	if err != nil {
		logger.Errorln("Fail to parse edgex-nosecty config to yaml:", err)
		return err
	}

	err = ioutil.WriteFile(saveNoSectyConfigPath, data, 0644)
	if err != nil {
		logger.Errorln("Fail to write nosecty-config yaml:", err)
		return err
	}

	return nil
}
