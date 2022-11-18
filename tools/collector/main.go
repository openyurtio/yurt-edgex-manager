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

	"github.com/openyurtio/yurt-edgex-manager/tools/collector/edgex"
	"gopkg.in/yaml.v3"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	collectLog     = ctrl.Log.WithName("collect")
	saveConfigPath = "../../EdgeXConfig/config.yaml"
)

func main() {
	flag.UintVar(&edgex.UnifiedPort, "unified-port", 2000, "Unify ports of the edgex component")

	flag.Parse()

	err := Run()
	if err != nil {
		collectLog.Error(err, "Fail to collect edgex configuration")
		return
	}
}

// Collect the edgex configuration and write it to the yaml file
func Run() error {
	logger := collectLog
	opts := zap.Options{
		Development: true,
	}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	versionsInfo, err := edgex.CollectVersionsInfo()
	if err != nil {
		return err
	}

	edgeXConfig, err := edgex.CollectEdgeXConfig(versionsInfo)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(edgeXConfig)
	if err != nil {
		logger.Error(err, "Fail to parse edgex config to yaml")
		return err
	}

	err = ioutil.WriteFile(saveConfigPath, data, 0644)
	if err != nil {
		logger.Error(err, "Fail to write yaml")
		return err
	}

	return nil
}
