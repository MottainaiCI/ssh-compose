/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package loader

import (
	"errors"
	"fmt"

	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"
)

func (i *SshCInstance) Validate(ignoreError bool) error {
	var ans error = nil
	mproj := make(map[string]int, 0)
	mnodes := make(map[string]int, 0)
	mgroups := make(map[string]int, 0)
	mcommands := make(map[string]int, 0)
	dupProjs := 0
	dupNodes := 0
	dupGroups := 0
	dupCommands := 0
	wrongHooks := 0

	// Check for duplicated project name
	for _, env := range i.Environments {

		for _, cmd := range env.Commands {
			if _, isPresent := mcommands[cmd.Name]; isPresent {
				if !ignoreError {
					return errors.New("Duplicated command " + cmd.Name)
				}

				i.Logger.Warning("Found duplicated command " + cmd.Name)
				dupCommands++

			} else {
				mcommands[cmd.Name] = 1
			}

			if cmd.Project == "" {
				if !ignoreError {
					return errors.New("Command " + cmd.Name + " with an empty project")
				}

				i.Logger.Warning("Command " + cmd.Name + " with an empty project.")
			}

			if !cmd.ApplyAlias {
				msg := fmt.Sprintf("Command %s with apply_alias disable. Not yet supported.",
					cmd.Name)

				if !ignoreError {
					return errors.New(msg)
				}

				i.Logger.Warning(msg)
			}

		}

		for _, proj := range env.Projects {

			if _, isPresent := mproj[proj.Name]; isPresent {
				if !ignoreError {
					return errors.New("Duplicated project " + proj.Name)
				}

				i.Logger.Warning("Found duplicated project " + proj.Name)

				dupProjs++

			} else {
				mproj[proj.Name] = 1
			}

			// Check groups
			for _, grp := range proj.Groups {

				if _, isPresent := mgroups[grp.Name]; isPresent {
					if !ignoreError {
						return errors.New("Duplicated group " + grp.Name)
					}

					i.Logger.Warning("Found duplicated group " + grp.Name)

					dupGroups++

				} else {
					mgroups[grp.Name] = 1
				}

				// Check group's hooks events
				if len(grp.Hooks) > 0 {
					for _, h := range grp.Hooks {
						if h.Event != specs.HookPreNodeSync &&
							h.Event != specs.HookPostNodeSync &&
							h.Event != specs.HookPreGroup &&
							h.Event != specs.HookPostGroup {

							wrongHooks++

							i.Logger.Warning("Found invalid hook of type " + h.Event +
								" on group " + grp.Name)

							if !ignoreError {
								return errors.New("Invalid hook " + h.Event + " on group " + grp.Name)
							}
						}

					}
				}

				for _, node := range grp.Nodes {

					if _, isPresent := mnodes[node.GetName()]; isPresent {
						if !ignoreError {
							return errors.New("Duplicated node " + node.GetName())
						}

						i.Logger.Warning("Found duplicated node " + node.GetName())

						dupNodes++

					} else {
						mnodes[node.GetName()] = 1
					}

					if len(node.Hooks) > 0 {
						for _, h := range node.Hooks {
							if h.Node != "" && h.Node != "host" {
								i.Logger.Warning("Invalid hook on node " + node.GetName() +
									" with node field valorized.")
								wrongHooks++
								if !ignoreError {
									return errors.New("Invalid hook on node " + node.GetName())
								}
							}

							if h.Event != specs.HookPreNodeSync &&
								h.Event != specs.HookPostNodeSync {

								wrongHooks++

								i.Logger.Warning("Found invalid hook of type " + h.Event +
									" on node " + node.GetName())

								if !ignoreError {
									return errors.New("Invalid hook " + h.Event + " on node " + node.GetName())
								}
							}
						}

					}

				}

			}
		}

		return nil
	}

	return ans
}
