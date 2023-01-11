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
	"bufio"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	collectLog           *logrus.Entry
	branchesURL          = "https://github.com/edgexfoundry/edgex-compose/branches/all"
	extractVersionRegexp = `branch="(.*?)"`
	singleArchPath       = "./config/singlearch_imagelist.txt"
	multiArchPath        = "./config/multiarch_imagelist.txt"
	repoName             = "openyurt/"
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

func CollectEdgeXConfig(versionsInfo []string, isSecurity bool, arch string) (*EdgeXConfig, error) {
	logger := collectLog
	logger.Infoln("Distributing version")

	edgeXConfig := newEdgeXConfig()

	for _, versionName := range versionsInfo {
		// The main branch is unstable. There is no need to synchronize the main branch
		if versionName == "main" {
			continue
		}

		version := newVersion(logger, versionName)
		err := version.catch(isSecurity, arch)
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

func CollectImages(edgexConfig, edgeXConfigArm *EdgeXConfig) error {
	fileSingleArch, err := os.Create(singleArchPath)
	if err != nil {
		return err
	}
	fileMutiArch, err := os.Create(multiArchPath)
	if err != nil {
		return err
	}
	defer fileSingleArch.Close()
	defer fileMutiArch.Close()

	writerSingleArch, writerMutiArch := bufio.NewWriter(fileSingleArch), bufio.NewWriter(fileMutiArch)
	versions, versionsArm := edgexConfig.Versions, edgeXConfigArm.Versions

	for i, version := range versions {
		components := version.Components
		newArray := make([]string, 0)
		for j := range components {
			imgSplit := strings.Split(versionsArm[i].Components[j].Image, ":")[0]
			newArray = append(newArray, imgSplit)
		}

		for _, component := range components {
			image := component.Image
			if !stringIsInArray(strings.Split(image, ":")[0], newArray) {
				writerSingleArch.WriteString(image + " ")
				imgArr := strings.Split(image, ":")
				imagePre := imgArr[:len(imgArr)-1]
				imagePre[len(imagePre)-1] = imagePre[len(imagePre)-1] + "-arm64"
				writerSingleArch.WriteString(" " + imagePre[0] + ":" + imgArr[len(imgArr)-1])
				writerSingleArch.WriteString("\n")
				writerSingleArch.Flush()
			} else {
				writerMutiArch.WriteString(image)
				writerMutiArch.WriteString("\n")
				writerMutiArch.Flush()
			}
		}
	}
	return err
}

func ModifyImagesName(edgexConfig *EdgeXConfig) {
	versions := edgexConfig.Versions
	for i, version := range versions {
		components := version.Components
		for j, component := range components {
			image := component.Image
			if strings.Contains(image, "/") {
				edgexConfig.Versions[i].Components[j].Image = repoName + strings.Split(image, "/")[1]
			} else {
				edgexConfig.Versions[i].Components[j].Image = repoName + image
			}
		}
	}

}
