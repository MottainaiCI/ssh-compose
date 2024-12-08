/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package specs

import (
	"gopkg.in/yaml.v3"
)

func (g *SshCGroup) Init() {
	// Initialize Hooks array to reduce code checks.
	if g.Hooks == nil {
		g.Hooks = []SshCHook{}
	}

	if g.Config == nil {
		g.Config = make(map[string]string, 0)
	}

	for idx := range g.Nodes {
		g.Nodes[idx].Init()
	}
}

func GroupFromYaml(data []byte) (*SshCGroup, error) {
	ans := &SshCGroup{}
	if err := yaml.Unmarshal(data, ans); err != nil {
		return nil, err
	}

	return ans, nil
}

func (g *SshCGroup) GetName() string        { return g.Name }
func (g *SshCGroup) GetDescription() string { return g.Description }
func (g *SshCGroup) GetConnection() string  { return g.Connection }
func (g *SshCGroup) GetNodes() *[]SshCNode  { return &g.Nodes }

func (g *SshCGroup) GetHooks(event string) []SshCHook {
	return getHooks(&g.Hooks, event)
}

func (g *SshCGroup) GetHooks4Nodes(event string, nodes []string) []SshCHook {
	return getHooks4Nodes(&g.Hooks, event, nodes)
}

func (g *SshCGroup) ToProcess(groupsEnabled, groupsDisabled []string) bool {
	ans := false

	if len(groupsDisabled) > 0 {
		for _, gd := range groupsDisabled {
			if gd == g.Name {
				return false
			}
		}
	}

	if len(groupsEnabled) > 0 {
		for _, ge := range groupsEnabled {
			if ge == g.Name {
				ans = true
				break
			}
		}
	} else {
		ans = true
	}

	return ans
}

func (g *SshCGroup) AddHooks(h *SshCHooks) {
	if len(h.Hooks) > 0 {
		g.Hooks = append(g.Hooks, h.Hooks...)
	}
}

func (g *SshCGroup) PrependHooks(h *SshCHooks) {
	if len(h.Hooks) > 0 {
		g.Hooks = append(h.Hooks, g.Hooks...)
	}
}
