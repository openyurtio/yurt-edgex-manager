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
)

type Component struct {
	Name         string            `yaml:"name"`
	Image        string            `yaml:"image"`
	Volumes      []string          `yaml:"volumns,omitempty"`
	ComponentEnv map[string]string `yaml:"componentEnv,omitempty"`
	// TODO: We need to crawl another no-security file and mark which components are not secure
	IsSecurity bool `yaml:"isSecurity"`
	// A pointer to the Env of the previous level
	envRef *map[string]string
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
			unifyPort(&key, &value)
			c.ComponentEnv[key] = value
		}
	}
}

func (c *Component) fillVolumes(volumes []interface{}) error {
	// TODO: Read volumes information
	return nil
}

func (c *Component) repairPorts() {
	repairPort(&c.ComponentEnv)
}
