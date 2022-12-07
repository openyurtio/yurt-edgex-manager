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
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

var (
	formatIntBase  = 10
	hostportLength = 4
	pageNotFound   = "404: Not Found"
	UnifiedPort    uint
)

func getPage(logger *logrus.Entry, url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		logger.Errorln("Failed to send request to edgex repo:", err)
		return "", err
	}
	defer resp.Body.Close()
	pageBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Errorln("Fail to read request body:", err)
		return "", err
	}
	pageStr := string(pageBytes)
	return pageStr, nil
}

func getPageWithRegex(logger *logrus.Entry, url, reStr string) ([]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		logger.Errorln("Failed to send request to edgex repo:", err)
		return nil, err
	}
	defer resp.Body.Close()
	pageBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Errorln("Fail to read request body:", err)
		return nil, err
	}
	pageStr := string(pageBytes)

	re := regexp.MustCompile(reStr)
	matches := re.FindAllStringSubmatch(pageStr, -1)

	results := make([]string, 0)
	for _, match := range matches {
		results = append(results, match[1])
	}
	return results, err
}

func loadEnv(logger *logrus.Entry, url string) (map[string]string, error) {
	content, err := getPage(logger, url)
	if err != nil {
		return nil, err
	} else if content == pageNotFound {
		return map[string]string{}, nil
	}

	envs, err := godotenv.Unmarshal(content)
	if err != nil {
		logger.Errorln("Fail to parse this env file:", err)
		return nil, err
	}

	return envs, nil
}
