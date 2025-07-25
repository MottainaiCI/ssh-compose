/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package specs

import (
	"os"

	"gopkg.in/yaml.v3"
)

func (c *SshCCommand) GetName() string                { return c.Name }
func (c *SshCCommand) GetDescription() string         { return c.Description }
func (c *SshCCommand) GetProject() string             { return c.Project }
func (c *SshCCommand) GetEnvs() SshCEnvVars           { return c.Envs }
func (c *SshCCommand) GetEnableFlags() []string       { return c.EnableFlags }
func (c *SshCCommand) GetDisableFlags() []string      { return c.DisableFlags }
func (c *SshCCommand) GetEnableGroups() []string      { return c.EnableGroups }
func (c *SshCCommand) GetDisableGroups() []string     { return c.DisableFlags }
func (c *SshCCommand) GetVarFiles() []string          { return c.VarFiles }
func (c *SshCCommand) GetSkipSync() bool              { return c.SkipSync }
func (c *SshCCommand) GetDestroy() bool               { return c.Destroy }
func (c *SshCCommand) SetEnableGroups(list []string)  { c.EnableGroups = list }
func (c *SshCCommand) SetDisableGroups(list []string) { c.DisableGroups = list }

func CommandFromYaml(data []byte) (*SshCCommand, error) {
	ans := &SshCCommand{}
	if err := yaml.Unmarshal(data, ans); err != nil {
		return nil, err
	}

	return ans, nil
}

func CommandFromFile(file string) (*SshCCommand, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return CommandFromYaml(content)
}
