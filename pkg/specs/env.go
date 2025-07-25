/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package specs

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func EnvVarsFromYaml(data []byte) (*SshCEnvVars, error) {
	ans := &SshCEnvVars{}
	if err := yaml.Unmarshal(data, ans); err != nil {
		return nil, err
	}
	return ans, nil
}

func NewEnvVars() *SshCEnvVars {
	return &SshCEnvVars{
		EnvVars: make(map[string]interface{}, 0),
	}
}

func (e *SshCEnvVars) AddKVAggregated(aggregatedEnv string) error {

	if aggregatedEnv == "" {
		return errors.New("Invalid key")
	}

	if strings.Index(aggregatedEnv, "=") < 0 {
		return errors.New(fmt.Sprintf("Invalid KV %s without =.", aggregatedEnv))
	}

	key := aggregatedEnv[0:strings.Index(aggregatedEnv, "=")]
	value := aggregatedEnv[strings.Index(aggregatedEnv, "=")+1:]

	e.EnvVars[key] = value

	return nil
}

func (e *SshCEnvVars) AddKV(key, value string) error {
	if key == "" {
		return errors.New("Invalid key")
	}

	if value == "" {
		return errors.New(fmt.Sprintf("Invalid value for key %s", key))
	}

	e.EnvVars[key] = value

	return nil
}
