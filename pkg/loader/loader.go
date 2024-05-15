/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package loader

import (
	log "github.com/MottainaiCI/ssh-compose/pkg/logger"
	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"
)

type SshCInstance struct {
	Config         *specs.SshComposeConfig
	Logger         *log.SshCLogger
	Environments   []specs.SshCEnvironment
	SkipSync       bool
	FlagsDisabled  []string
	FlagsEnabled   []string
	GroupsEnabled  []string
	GroupsDisabled []string

	Remotes *specs.RemotesConfig
}

func NewSshCInstance(config *specs.SshComposeConfig) (*SshCInstance, error) {
	var err error
	ans := &SshCInstance{
		Config:       config,
		Logger:       log.NewSshCLogger(config),
		Environments: make([]specs.SshCEnvironment, 0),
	}

	// Initialize logging
	if config.GetLogging().EnableLogFile && config.GetLogging().Path != "" {
		err = ans.Logger.InitLogger2File()
		if err != nil {
			ans.Logger.Fatal("Error on initialize logfile")
		}
	}
	ans.Logger.SetAsDefault()

	ans.Remotes, err = specs.LoadRemotesConfig(
		config.GetGeneral().RemotesConfDir,
	)

	if err == nil {
		ans.Remotes.Sanitize()
	}

	return ans, err
}

func (i *SshCInstance) GetRemotes() *specs.RemotesConfig  { return i.Remotes }
func (i *SshCInstance) SetRemotes(r *specs.RemotesConfig) { i.Remotes = r }

func (i *SshCInstance) AddEnvironment(env specs.SshCEnvironment) {
	i.Environments = append(i.Environments, env)
}

func (i *SshCInstance) GetEnvironments() *[]specs.SshCEnvironment {
	return &i.Environments
}

func (i *SshCInstance) SetSkipSync(v bool)          { i.SkipSync = v }
func (i *SshCInstance) GetSkipSync() bool           { return i.SkipSync }
func (i *SshCInstance) GetGroupsEnabled() []string  { return i.GroupsEnabled }
func (i *SshCInstance) GetGroupsDisabled() []string { return i.GroupsDisabled }
func (i *SshCInstance) SetGroupsEnabled(groups []string) {
	i.GroupsEnabled = groups
}
func (i *SshCInstance) SetGroupsDisabled(groups []string) {
	i.GroupsDisabled = groups
}

func (i *SshCInstance) GetFlagsEnabled() []string  { return i.FlagsEnabled }
func (i *SshCInstance) GetFlagsDisabled() []string { return i.FlagsDisabled }
func (i *SshCInstance) SetFlagsEnabled(flags []string) {
	i.FlagsEnabled = flags
}
func (i *SshCInstance) SetFlagsDisabled(flags []string) {
	i.FlagsDisabled = flags
}
func (i *SshCInstance) AddFlagEnabled(flag string) {
	i.FlagsEnabled = append(i.FlagsEnabled, flag)
}
func (i *SshCInstance) AddFlagDisabled(flag string) {
	i.FlagsDisabled = append(i.FlagsDisabled, flag)
}

func (i *SshCInstance) GetEnvByProjectName(name string) *specs.SshCEnvironment {
	for _, e := range i.Environments {
		for _, p := range e.Projects {
			if p.Name == name {
				return &e
			}
		}
	}

	return nil
}

func (i *SshCInstance) GetEntitiesByNodeName(name string) (*specs.SshCEnvironment, *specs.SshCProject, *specs.SshCGroup, *specs.SshCNode) {
	for _, e := range i.Environments {
		for _, p := range e.Projects {
			for _, g := range p.Groups {
				for _, n := range g.Nodes {
					if n.GetName() == name {
						return &e, &p, &g, &n
					}
				}
			}
		}
	}
	return nil, nil, nil, nil
}

func (i *SshCInstance) GetConfig() *specs.SshComposeConfig {
	return i.Config
}
