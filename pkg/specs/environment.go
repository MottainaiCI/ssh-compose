/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package specs

import (
	"errors"
	"path"

	"gopkg.in/yaml.v3"
)

func EnvironmentFromYaml(data []byte, file string) (*SshCEnvironment, error) {
	ans := &SshCEnvironment{}
	if err := yaml.Unmarshal(data, ans); err != nil {
		return nil, err
	}
	ans.File = file

	if ans.Commands == nil {
		ans.Commands = []SshCCommand{}
	}
	if ans.IncludeCommandsFiles == nil {
		ans.IncludeCommandsFiles = []string{}
	}

	for idx := range ans.Projects {
		ans.Projects[idx].Init()
	}

	return ans, nil
}

func (e *SshCEnvironment) GetProjectByName(pName string) *SshCProject {
	for idx, p := range e.Projects {
		if p.Name == pName {
			return &e.Projects[idx]
		}
	}

	return nil
}

func (e *SshCEnvironment) GetProjects() *[]SshCProject {
	return &e.Projects
}

func (e *SshCEnvironment) GetCommands() *[]SshCCommand {
	return &e.Commands
}

func (e *SshCEnvironment) GetCommand(name string) (*SshCCommand, error) {
	for idx, cmd := range e.Commands {
		if cmd.Name == name {
			return &e.Commands[idx], nil
		}
	}

	return nil, errors.New("Command + " + name + " not available.")
}

func (e *SshCEnvironment) AddCommand(cmd *SshCCommand) {
	e.Commands = append(e.Commands, *cmd)
}

func (e *SshCEnvironment) GetBaseFile() string {
	ans := ""
	if e.File != "" {
		ans = path.Base(e.File)
	}

	return ans
}
