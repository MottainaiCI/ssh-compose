/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package loader

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	ssh_executor "github.com/MottainaiCI/ssh-compose/pkg/executor"
	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"
	"github.com/MottainaiCI/ssh-compose/pkg/template"
)

func (i *SshCInstance) GetNodeHooks4Event(event string, proj *specs.SshCProject, group *specs.SshCGroup, node *specs.SshCNode) []specs.SshCHook {

	// Retrieve project hooks
	projHooks := proj.GetHooks4Nodes(event, []string{"*"})
	projHooks = specs.FilterHooks4Node(&projHooks, []string{node.GetName(), "host"})

	// Retrieve group hooks
	groupHooks := group.GetHooks4Nodes(event, []string{"*"})
	groupHooks = specs.FilterHooks4Node(&groupHooks, []string{node.GetName(), "host"})

	ans := projHooks
	ans = append(ans, groupHooks...)
	ans = append(ans, node.GetAllHooks(event)...)

	return ans
}

func (i *SshCInstance) ApplyProject(projectName string) error {

	env := i.GetEnvByProjectName(projectName)
	if env == nil {
		return errors.New("No environment found for project " + projectName)
	}

	proj := env.GetProjectByName(projectName)
	if proj == nil {
		return errors.New("No project found with name " + projectName)
	}

	// Get only host hooks. All other hooks are handled by group and node.
	preProjHooks := proj.GetHooks4Nodes(specs.HookPreProject, []string{"host"})
	postProjHooks := proj.GetHooks4Nodes(specs.HookPostProject, []string{"*", "host"})

	// Execute pre-project hooks
	i.Logger.Debug(fmt.Sprintf(
		"[%s] Running %d %s hooks... ", projectName,
		len(preProjHooks), specs.HookPreProject))
	err := i.ProcessHooks(&preProjHooks, proj, nil, env, nil)
	if err != nil {
		return err
	}

	compiler, err := template.NewProjectTemplateCompiler(env, proj)
	if err != nil {
		return err
	}

	// Compiler project files
	err = template.CompileProjectFiles(proj, compiler, template.CompilerOpts{})
	if err != nil {
		return err
	}

	for _, grp := range proj.Groups {

		if !grp.ToProcess(i.GroupsEnabled, i.GroupsDisabled) {
			i.Logger.Debug("Skipped group ", grp.Name)
			continue
		}

		err := i.ApplyGroup(&grp, proj, env, compiler)
		if err != nil {
			return err
		}

	}

	// Execute post-project hooks
	i.Logger.Debug(fmt.Sprintf(
		"[%s] Running %d %s hooks... ", projectName,
		len(preProjHooks), specs.HookPostProject))
	err = i.ProcessHooks(&postProjHooks, proj, nil, env, nil)
	if err != nil {
		return err
	}

	return nil
}

func (i *SshCInstance) cleanupExecutorMap() {
	if len(i.executorMap) > 0 {
		for _, executor := range i.executorMap {
			executor.Close()
		}

		i.executorMap = make(map[string]*ssh_executor.SshCExecutor, 0)
	}
}

func (i *SshCInstance) getExecutor(node, endpoint string) (*ssh_executor.SshCExecutor, error) {
	ans, ok := i.executorMap[node]

	if ok {
		return ans, nil
	}

	// Retrieve the node from remotes
	if !i.Remotes.HasRemote(endpoint) {
		return nil, fmt.Errorf(
			"error on retrieve the remote with name %s for node %s",
			endpoint, node)
	}

	remote := i.Remotes.GetRemote(endpoint)
	executor, err := ssh_executor.NewSshCExecutorFromRemote(endpoint, remote)
	if err != nil {
		return nil, fmt.Errorf(
			"error on create executor from remote %s (node %s): %s",
			endpoint, node, err.Error())
	}
	err = executor.Setup()
	if err != nil {
		return nil, fmt.Errorf(
			"error on setup executor for node %s: %s",
			node, err.Error())
	}
	executor.ConfigDir, _ = i.Remotes.GetAbsConfigDir()

	i.executorMap[node] = executor

	return executor, nil
}

func (i *SshCInstance) ProcessHooks(hooks *[]specs.SshCHook, proj *specs.SshCProject,
	group *specs.SshCGroup, env *specs.SshCEnvironment, targetNode *specs.SshCNode) error {
	envBaseAbs, err := filepath.Abs(filepath.Dir(env.File))
	if err != nil {
		return err
	}

	var res int
	nodes := []specs.SshCNode{}
	storeVar := false

	cleanUp := func() {
	}
	defer cleanUp()

	if len(*hooks) <= 0 {
		return nil
	}

	pullNodeResources := func(h *specs.SshCHook, node string) error {

		if node == "host" {
			return fmt.Errorf("hook with host node could be used for pull!")
		}

		var nodeEntity *specs.SshCNode = nil

		_, _, _, nodeEntity = i.GetEntitiesByNodeName(node)
		if nodeEntity == nil {
			return fmt.Errorf("error on retrieve node entity of the node %s", node)
		}

		executor, err := i.getExecutor(node, nodeEntity.Endpoint)
		if err != nil {
			i.Logger.Error("Error on retrieve executor of the node " +
				node + ": " + err.Error())
			return err
		}
		// TODO: propagate sftp client options
		err = executor.SetupSftp()
		if err != nil {
			i.Logger.Error("Error on setup sftp client on executor of the node " +
				node + ": " + err.Error())
			return err
		}

		nPullResources := len(h.PullResources)
		i.Logger.InfoC(
			i.Logger.Aurora.Bold(
				i.Logger.Aurora.BrightCyan(
					fmt.Sprintf(">>> [%s] Pulling %d resources... - :bus:",
						node, nPullResources))))

		// TODO: Propagate this options from config
		ensurePerms := false

		for idx, resource := range h.PullResources {

			var targetPath string

			if filepath.IsAbs(resource.Destination) {
				targetPath = resource.Destination
			} else {
				targetPath = filepath.Join(envBaseAbs, "pull", node, resource.Destination)
			}

			if strings.HasSuffix(resource.Destination, "/") {
				targetPath += "/"
			}

			err = executor.RecursivePullFile(node,
				resource.Source, targetPath, !h.PullKeepSourcePath, ensurePerms)
			if err != nil {
				i.Logger.Debug("Error on pull from sourcePath " + resource.Source +
					" to dest " + targetPath)
				i.Logger.Error("Error on pull " + resource.Source + ": " + err.Error())
				return err
			}

			i.Logger.InfoC(
				i.Logger.Aurora.BrightCyan(
					fmt.Sprintf(">>> [%s] - [%2d/%2d] %s - :check_mark:",
						node, idx+1, nPullResources, resource.Source)))
		}

		return nil
	}

	runSingleCmd := func(h *specs.SshCHook, node, cmds string) error {
		var executor *ssh_executor.SshCExecutor
		var err error

		envs, err := proj.GetEnvsMap()
		if err != nil {
			return err
		}
		if _, ok := envs["HOME"]; !ok {
			envs["HOME"] = "/"
		}

		if node != "host" {
			var nodeEntity *specs.SshCNode = nil

			_, _, _, nodeEntity = i.GetEntitiesByNodeName(node)
			if nodeEntity != nil {
				json, err := nodeEntity.ToJson()
				if err != nil {
					return err
				}
				envs["node"] = json

				if nodeEntity.Labels != nil && len(nodeEntity.Labels) > 0 {
					for k, v := range nodeEntity.Labels {
						envs[k] = v
					}
				}

				executor, err = i.getExecutor(node, nodeEntity.Endpoint)
				if err != nil {
					return err
				}

			} else {
				return fmt.Errorf("error on retrieve executor of the node %s", node)
			}

		} else {
			// POST: node == host
			// NOTE: I use a fake executor. Probably we
			//       could create a specific executor for the Host
			//       in the near future.

			executor = ssh_executor.NewSshCExecutor(
				"host", "127.0.0.1", 22)
			executor.ConfigDir, _ = i.Remotes.GetAbsConfigDir()

			// NOTE: I don't need to run executor.Setup() for host node.
		}

		if h.Out2Var != "" || h.Err2Var != "" {
			storeVar = true
		} else {
			storeVar = false
		}

		if h.Node == "host" {
			if storeVar {
				res, err = executor.RunHostCommandWithOutput4Var(cmds, h.Out2Var, h.Err2Var, &envs, h.Entrypoint)
			} else {

				if i.Config.GetLogging().RuntimeCmdsOutput {
					emitter := executor.GetEmitter()
					res, err = executor.RunHostCommandWithOutput(
						cmds, envs,
						(emitter.(*ssh_executor.SshCEmitter)).GetHostWriterStdout(),
						(emitter.(*ssh_executor.SshCEmitter)).GetHostWriterStderr(),
						h.Entrypoint,
					)
				} else {
					res, err = executor.RunHostCommand(cmds, envs, h.Entrypoint)
				}
			}
		} else {

			if storeVar {
				res, err = executor.RunCommandWithOutput4Var(node, cmds, h.Out2Var, h.Err2Var, &envs, h.Entrypoint)
			} else {
				if i.Config.GetLogging().RuntimeCmdsOutput {

					emitter := executor.GetEmitter()

					if executor.CiscoDevice {

						// Cisco device seems doesn't support multi SSH sessions.
						// We need to use a pty ssh session and write the hooks
						// command in the same session.
						// In this case we lose support of the return status of
						// a command. We need to process the output but is
						// at the moment just ignore. We run the command
						// without check response.

						// NOTE: The ena options need to be enable at the first hook
						//       and atm we don't support exiting from the ena mode.
						ciscoOpts := ssh_executor.NewCiscoCommandOpts(h.CiscoEna)

						res, err = executor.RunCommandWithOutputOnCiscoDeviceWithDS(
							node, cmds, envs,
							(emitter.(*ssh_executor.SshCEmitter)).GetSshWriterStdout(),
							(emitter.(*ssh_executor.SshCEmitter)).GetSshWriterStderr(),
							h.Entrypoint,
							ciscoOpts)

					} else {

						res, err = executor.RunCommandWithOutput(
							node, cmds, envs,
							(emitter.(*ssh_executor.SshCEmitter)).GetSshWriterStdout(),
							(emitter.(*ssh_executor.SshCEmitter)).GetSshWriterStderr(),
							h.Entrypoint)

					}
				} else {
					res, err = executor.RunCommand(
						node, cmds, envs, h.Entrypoint,
					)
				}
			}

		}

		if err != nil {
			i.Logger.Error("Error " + err.Error())
			return err
		}

		if res != 0 {
			i.Logger.Error(fmt.Sprintf("Command result wrong (%d). Exiting.", res))
			return errors.New("Error on execute command: " + cmds)
		}

		if storeVar {
			if len(proj.Environments) == 0 {
				proj.AddEnvironment(&specs.SshCEnvVars{EnvVars: make(map[string]interface{}, 0)})
			}
			if h.Out2Var != "" {
				proj.Environments[len(proj.Environments)-1].EnvVars[h.Out2Var] = envs[h.Out2Var]
			}
			if h.Err2Var != "" {
				proj.Environments[len(proj.Environments)-1].EnvVars[h.Err2Var] = envs[h.Err2Var]
			}
		}

		return nil
	}

	// Retrieve list of nodes
	if group != nil {
		nodes = group.Nodes
	} else {
		for _, g := range proj.Groups {
			nodes = append(nodes, g.Nodes...)
		}
	}

	for _, h := range *hooks {

		// Check if hooks must be processed
		if !h.ToProcess(i.FlagsEnabled, i.FlagsDisabled) {
			i.Logger.Debug("Skipped hooks ", h)
			continue
		}

		if h.HasPullResources() {

			switch h.Node {
			case "", "*":
				if targetNode != nil {
					err := pullNodeResources(&h, targetNode.GetName())
					if err != nil {
						return err
					}
				} else {
					for _, node := range nodes {
						err := pullNodeResources(&h, node.GetName())
						if err != nil {
							return err
						}
					}
				}

			default:
				err := pullNodeResources(&h, h.Node)
				if err != nil {
					return err
				}
			}

		} else if h.Commands != nil && len(h.Commands) > 0 {

			for _, cmds := range h.Commands {
				switch h.Node {
				case "", "*":
					if targetNode != nil {
						err := runSingleCmd(&h, targetNode.GetName(), cmds)
						if err != nil {
							return err
						}
					} else {
						for _, node := range nodes {
							err := runSingleCmd(&h, node.GetName(), cmds)
							if err != nil {
								return err
							}
						}
					}

				default:

					err := runSingleCmd(&h, h.Node, cmds)
					if err != nil {
						return err
					}
				}

			}

		}
	}

	return nil
}

func (i *SshCInstance) ApplyGroup(group *specs.SshCGroup, proj *specs.SshCProject, env *specs.SshCEnvironment, compiler template.SshCTemplateCompiler) error {

	// Retrieve pre-group hooks from project
	preGroupHooks := proj.GetHooks4Nodes(specs.HookPreGroup, []string{"*", "host"})
	// Retrieve pre-group hooks from group
	preGroupHooks = append(preGroupHooks, group.GetHooks4Nodes(specs.HookPreGroup, []string{"*", "host"})...)

	// Run pre-group hooks
	i.Logger.Debug(fmt.Sprintf(
		"[%s - %s] Running %d %s hooks... ", proj.Name, group.Name,
		len(preGroupHooks), specs.HookPreGroup))
	err := i.ProcessHooks(&preGroupHooks, proj, group, env, nil)
	if err != nil {
		return err
	}

	// We need reload variables updated from out2var/err2var hooks.
	compiler.InitVars()

	// Compile group templates
	err = template.CompileGroupFiles(group, compiler, template.CompilerOpts{})
	if err != nil {
		return err
	}

	defer i.cleanupExecutorMap()

	// TODO: implement parallel creation
	for _, node := range group.Nodes {

		finallyHooks := i.GetNodeHooks4Event(specs.HookFinally, proj, group, &node)

		err = i.ApplyNode(&node, group, proj, env, compiler)

		// Run finally hooks
		errFinally := i.ProcessHooks(&finallyHooks, proj, group, env, &node)
		if errFinally != nil {
			return errFinally
		}

		if err != nil {
			return err
		}

	}

	// Retrieve post-group hooks from project
	postGroupHooks := proj.GetHooks4Nodes(specs.HookPostGroup, []string{"*", "host"})
	postGroupHooks = append(postGroupHooks, group.GetHooks4Nodes(specs.HookPostGroup, []string{"*", "host"})...)

	// Execute post-group hooks
	i.Logger.Debug(fmt.Sprintf(
		"[%s - %s] Running %d %s hooks... ", proj.Name, group.Name,
		len(postGroupHooks), specs.HookPostGroup))
	err = i.ProcessHooks(&postGroupHooks, proj, group, env, nil)

	return err
}

func (i *SshCInstance) ApplyCommand(c *specs.SshCCommand, proj *specs.SshCProject, envs []string, varfiles []string) error {

	if c == nil {
		return errors.New("Invalid command")
	}

	if proj == nil {
		return errors.New("Invalid project")
	}

	env := i.GetEnvByProjectName(proj.GetName())
	if env == nil {
		return errors.New(fmt.Sprintf("No environment found for project " + proj.GetName()))
	}

	envBaseDir, err := filepath.Abs(filepath.Dir(env.File))
	if err != nil {
		return err
	}

	// Load envs from commands.
	if len(c.VarFiles) > 0 {
		for _, varFile := range c.VarFiles {

			envs, err := i.loadEnvFile(envBaseDir, varFile, proj)
			if err != nil {
				return errors.New(
					fmt.Sprintf(
						"Error on load additional envs var file %s: %s",
						varFile, err.Error()),
				)
			}

			proj.AddEnvironment(envs)

		}
	}

	if len(c.Envs.EnvVars) > 0 {
		proj.AddEnvironment(&c.Envs)
	}

	if len(c.IncludeHooksFiles) > 0 {

		for _, hinclude := range c.IncludeHooksFiles {

			for _, hfile := range hinclude.GetFiles() {
				// Load project included hooks
				hf := path.Join(envBaseDir, hfile)
				hooks, err := i.getHooks(hfile, hf, proj)
				if err != nil {
					return err
				}

				if hinclude.IncludeInAppend() {
					proj.AddHooks(hooks)
				} else {
					proj.PrependHooks(hooks)
				}
			}
		}
	}

	if len(envs) > 0 {
		evars := specs.NewEnvVars()
		for _, e := range envs {
			err := evars.AddKVAggregated(e)
			if err != nil {
				return errors.New(
					fmt.Sprintf(
						"Error on elaborate var string %s: %s",
						e, err.Error(),
					))
			}
		}

		proj.AddEnvironment(evars)
	}

	i.SetFlagsDisabled(c.DisableFlags)
	i.SetFlagsEnabled(c.EnableFlags)
	i.SetGroupsDisabled(c.DisableGroups)
	i.SetGroupsEnabled(c.EnableGroups)
	i.SetSkipSync(c.SkipSync)
	i.SetSkipCompile(c.SkipCompile)

	return nil
}
