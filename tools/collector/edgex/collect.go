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
)

var (
	collectLog           *logrus.Entry
	branchesURL          = "https://github.com/edgexfoundry/edgex-compose/branches/all"
	extractVersionRegexp = `branch="(.*?)"`
)

func SetLog(logger *logrus.Entry) {
	collectLog = logger
}

func CollectVersionsInfo() ([]string, error) {
	logger := collectLog
	logger.Infoln("Collecting versions")

	branches, err := getPageWithRegex(logger, branchesURL, extractVersionRegexp)
	if err != nil {
		return nil, err
	}

	return branches, nil
}

func CollectEdgeXConfig(versionsInfo []string, isSecurity bool) (*EdgeXConfig, error) {
	logger := collectLog
	logger.Infoln("Distributing version")

	edgeXConfig := newEdgeXConfig()

	for _, versionName := range versionsInfo {
		// The main branch is unstable. There is no need to synchronize the main branch
		if versionName == "main" {
			continue
		}

		version := newVersion(logger, versionName)
		err := version.catch(isSecurity)
		if err != nil && err == ErrConfigFileNotFound {
			logger.Warningln("The configuration file for this version could not be found,", "version:", versionName)
			continue
		} else if err != nil && err == ErrVersionNotAdapted {
			logger.Warningln("The configuration file of this version cannot be captured")
			continue
		}
		edgeXConfig.Versions = append(edgeXConfig.Versions, *version)
	}

	return edgeXConfig, nil
}
