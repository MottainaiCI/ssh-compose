/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package specs

import (
	"fmt"
	"path/filepath"

	"github.com/ghodss/yaml"
)

func (n *SshCNode) Init() {
	if n.Hooks == nil {
		n.Hooks = []SshCHook{}
	}
}

func (n *SshCNode) IsSourcePathRelative() bool {
	if filepath.IsAbs(n.SourceDir) {
		return false
	}
	return true
}

func (n *SshCNode) GetHooks(event string) []SshCHook {
	return getHooks(&n.Hooks, event)
}

func (n *SshCNode) GetAllHooks(event string) []SshCHook {
	return getHooks4Nodes(&n.Hooks, event, []string{"*"})
}

func (n *SshCNode) ToJson() (string, error) {
	y, err := yaml.Marshal(n)
	if err != nil {
		return "", fmt.Errorf("Error on convert node %s to yaml: %s",
			n.Name, err.Error())
	}

	data, err := yaml.YAMLToJSON(y)
	if err != nil {
		return "", fmt.Errorf("Error on convert node %s to json: %s",
			n.Name, err.Error())
	}

	return string(data), nil
}

func (n *SshCNode) GetName() string {
	return n.Name
}

func (n *SshCNode) AddHooks(h *SshCHooks) {
	if len(h.Hooks) > 0 {
		n.Hooks = append(n.Hooks, h.Hooks...)
	}
}
