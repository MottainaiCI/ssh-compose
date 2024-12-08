/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package specs

import (
	"fmt"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/icza/dyno"
)

func (p *SshCProject) Init() {
	if p.Hooks == nil {
		p.Hooks = []SshCHook{}
	}

	for idx := range p.Groups {
		p.Groups[idx].Init()
	}
}

func (p *SshCProject) GetGroups() *[]SshCGroup { return &p.Groups }
func (p *SshCProject) GetDescription() string  { return p.Description }
func (p *SshCProject) GetName() string         { return p.Name }

func (p *SshCProject) AddGroup(grp *SshCGroup) {
	p.Groups = append(p.Groups, *grp)
}

func (p *SshCProject) AddEnvironment(e *SshCEnvVars) {
	p.Environments = append(p.Environments, *e)
}

func (p *SshCProject) GetGroupByName(name string) *SshCGroup {
	for idx := range p.Groups {
		if p.Groups[idx].Name == name {
			return &p.Groups[idx]
		}
	}
	return nil
}

func (p *SshCProject) GetEnvsMap() (map[string]string, error) {
	ans := map[string]string{}

	y, err := yaml.Marshal(p.Sanitize())
	if err != nil {
		return ans, fmt.Errorf("Error on convert project %s to yaml: %s",
			p.GetName(), err.Error())
	}
	pData, err := yaml.YAMLToJSON(y)
	if err != nil {
		return ans, fmt.Errorf("Error on convert project %s to json: %s",
			p.GetName(), err.Error())
	}
	ans["project"] = string(pData)

	for _, e := range p.Environments {
		for k, v := range e.EnvVars {

			// Bash doesn't support variable with dash.
			// I will convert dash with underscore.
			if strings.Contains(k, "-") {
				k = strings.ReplaceAll(k, "-", "_")
			}

			switch v.(type) {
			case int:
				ans[k] = fmt.Sprintf("%d", v.(int))
			case string:
				ans[k] = v.(string)
			default:
				m := dyno.ConvertMapI2MapS(v)
				y, err := yaml.Marshal(m)
				if err != nil {
					return ans, fmt.Errorf("Error on convert var %s to yaml: %s",
						k, err.Error())
				}

				data, err := yaml.YAMLToJSON(y)
				if err != nil {
					return ans, fmt.Errorf("Error on convert var %s to json: %s",
						k, err.Error())
				}
				ans[k] = string(data)
			}
		}
	}

	return ans, nil
}

func (p *SshCProject) GetHooks(event string) []SshCHook {
	return getHooks(&p.Hooks, event)
}

func (p *SshCProject) GetHooks4Nodes(event string, nodes []string) []SshCHook {
	return getHooks4Nodes(&p.Hooks, event, nodes)
}

func (p *SshCProject) Sanitize() *SshCProjectSanitized {
	return &SshCProjectSanitized{
		Name:              p.Name,
		Description:       p.Description,
		IncludeGroupFiles: p.IncludeGroupFiles,
		IncludeEnvFiles:   p.IncludeEnvFiles,
		Groups:            p.Groups,
		Hooks:             p.Hooks,
		ConfigTemplates:   p.ConfigTemplates,
	}
}

func (p *SshCProjectSanitized) GetName() string         { return p.Name }
func (p *SshCProjectSanitized) GetDescription() string  { return p.Description }
func (p *SshCProjectSanitized) GetGroups() *[]SshCGroup { return &p.Groups }

func (p *SshCProject) LoadEnvVarsFile(file string) error {
	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	evars, err := EnvVarsFromYaml(content)
	if err != nil {
		return err
	}

	p.AddEnvironment(evars)

	return nil
}

func (p *SshCProject) AddHooks(h *SshCHooks) {
	if len(h.Hooks) > 0 {
		p.Hooks = append(p.Hooks, h.Hooks...)
	}
}

func (p *SshCProject) PrependHooks(h *SshCHooks) {
	if len(h.Hooks) > 0 {
		p.Hooks = append(h.Hooks, p.Hooks...)
	}
}

func (p *SshCProject) PrependHooksList(list []*SshCHooks) {
	// Drop empty hooks
	hooks := []SshCHook{}

	if len(list) > 0 {
		for idx := range list {
			if len(list[idx].Hooks) > 0 {
				hooks = append(hooks, list[idx].Hooks...)
			}
		}

		p.Hooks = append(hooks, p.Hooks...)
	}
}
