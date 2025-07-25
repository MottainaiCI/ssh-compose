/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package specs

import (
	"github.com/jinzhu/copier"
	"gopkg.in/yaml.v3"
)

const (
	HookPreProject   = "pre-project"
	HookPreGroup     = "pre-group"
	HookPreNodeSync  = "pre-node-sync"
	HookPostNodeSync = "post-node-sync"
	HookPostGroup    = "post-group"
	HookPostProject  = "post-project"
	HookFinally      = "finally"
)

func getHooks(hooks *[]SshCHook, event string) []SshCHook {
	return getHooks4Nodes(hooks, event, []string{""})
}

func getHooks4Nodes(hooks *[]SshCHook, event string, nodes []string) []SshCHook {
	ans := []SshCHook{}

	if hooks != nil {
		for _, h := range *hooks {
			if h.Event == event {

				for _, node := range nodes {
					if (node == "" && h.Node != "host") || node == "*" {
						ans = append(ans, h)
						break
					} else {
						if node == h.Node {
							ans = append(ans, h)
							break
						}
					}
				}

			}
		}
	}

	return ans
}

func (h *SshCHook) For(node string) bool {
	if h.Node == "" || h.Node == "*" || h.Node == node {
		return true
	}
	return false
}

func (h *SshCHook) Clone() *SshCHook {
	ans := SshCHook{}
	copier.Copy(&ans, h)
	return &ans
}

func (h *SshCHook) SetNode(node string) {
	h.Node = node
}

func (h *SshCHook) HasPullResources() bool {
	if len(h.PullResources) > 0 {
		return true
	}
	return false
}

func (h *SshCHook) ToProcess(enabledFlags, disabledFlags []string) bool {
	ans := false

	if h.Disable && len(enabledFlags) == 0 {
		return false
	}

	if len(h.Flags) == 0 && len(enabledFlags) == 0 {
		return true
	}

	if len(disabledFlags) > 0 {
		// Check if the flag is present
		for _, df := range disabledFlags {
			if h.ContainsFlag(df) {
				return false
			}
		}
	}

	if len(enabledFlags) > 0 {
		for _, ef := range enabledFlags {
			if h.ContainsFlag(ef) {
				ans = true
				break
			}
		}
	} else {
		ans = true
	}

	return ans
}

func (h *SshCHook) ContainsFlag(flag string) bool {
	ans := false
	if len(h.Flags) > 0 {
		for _, f := range h.Flags {
			if f == flag {
				ans = true
				break
			}
		}
	}

	return ans
}

func FilterHooks4Node(hooks *[]SshCHook, nodes []string) []SshCHook {
	ans := []SshCHook{}

	if hooks != nil {
		for _, h := range *hooks {
			for _, node := range nodes {
				if h.For(node) {
					nh := h.Clone()
					nh.SetNode(node)
					ans = append(ans, *nh)
					break
				}
			}
		}
	}

	return ans
}

func HooksFromYaml(data []byte) (*SshCHooks, error) {
	ans := &SshCHooks{}
	if err := yaml.Unmarshal(data, ans); err != nil {
		return nil, err
	}

	return ans, nil
}
