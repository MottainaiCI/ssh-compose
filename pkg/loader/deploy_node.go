/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package loader

import (
	"fmt"
	"path/filepath"
	"strings"

	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"
	"github.com/MottainaiCI/ssh-compose/pkg/template"
)

func (i *SshCInstance) ApplyNode(node *specs.SshCNode,
	group *specs.SshCGroup, proj *specs.SshCProject, env *specs.SshCEnvironment,
	compiler template.SshCTemplateCompiler) error {

	syncSourceDir := ""
	envBaseAbs, err := filepath.Abs(filepath.Dir(env.File))
	if err != nil {
		return err
	}

	// Retrieve pre-node-sync hooks of the node from project
	preSyncHooks := i.GetNodeHooks4Event(specs.HookPreNodeSync, proj, group, node)

	// Run pre-node-sync hooks
	err = i.ProcessHooks(&preSyncHooks, proj, group, env, node)
	if err != nil {
		return err
	}

	// We need reload variables updated from out2var/err2var hooks.
	compiler.InitVars()

	// Compile node templates
	err = template.CompileNodeFiles(*node, compiler, template.CompilerOpts{})
	if err != nil {
		return err
	}

	if len(node.SyncResources) > 0 && !i.SkipSync {
		if node.SourceDir != "" {
			if node.IsSourcePathRelative() {
				syncSourceDir = filepath.Join(envBaseAbs, node.SourceDir)
			} else {
				syncSourceDir = node.SourceDir
			}
		} else {
			// Use env file directory
			syncSourceDir = envBaseAbs
		}

		executor, err := i.getExecutor(node.GetName(), node.Endpoint)
		if err != nil {
			i.Logger.Error("Error on retrieve executor of the node " +
				node.GetName() + ": " + err.Error())
			return err
		}
		// TODO: propagate sftp client options
		err = executor.SetupSftp()
		if err != nil {
			i.Logger.Error("Error on setup sftp client on executor of the node " +
				node.GetName() + ": " + err.Error())
			return err
		}

		i.Logger.Debug(i.Logger.Aurora.Bold(
			i.Logger.Aurora.BrightCyan(
				">>> [" + node.GetName() + "] Using sync source basedir " +
					syncSourceDir)))

		nResources := len(node.SyncResources)
		i.Logger.InfoC(
			i.Logger.Aurora.Bold(
				i.Logger.Aurora.BrightCyan(
					fmt.Sprintf(">>> [%s] Syncing %d resources... - :bus:",
						node.GetName(), nResources))))

		for idx, resource := range node.SyncResources {

			var sourcePath string

			if filepath.IsAbs(resource.Source) {
				sourcePath = resource.Source
			} else {
				sourcePath = filepath.Join(syncSourceDir, resource.Source)
			}

			i.Logger.DebugC(
				i.Logger.Aurora.Italic(
					i.Logger.Aurora.BrightCyan(
						fmt.Sprintf(">>> [%s] %s => %s",
							node.GetName(), resource.Source,
							resource.Destination))))

			// TODO: Propagate this options from config
			ensurePerms := false

			if strings.HasSuffix(resource.Source, "/") {
				sourcePath += "/"
			}

			err = executor.RecursivePushFile(node.GetName(),
				sourcePath, resource.Destination, ensurePerms)
			if err != nil {
				i.Logger.Debug("Error on sync from sourcePath " + sourcePath +
					" to dest " + resource.Destination)
				i.Logger.Error("Error on sync " + resource.Source + ": " + err.Error())
				return err
			}

			i.Logger.InfoC(
				i.Logger.Aurora.BrightCyan(
					fmt.Sprintf(">>> [%s] - [%2d/%2d] %s - :check_mark:",
						node.GetName(), idx+1, nResources, resource.Destination)))
		}

	}

	// Retrieve post-node-sync hooks of the node from project
	postSyncHooks := i.GetNodeHooks4Event(specs.HookPostNodeSync, proj, group, node)

	// Run post-node-sync hooks
	err = i.ProcessHooks(&postSyncHooks, proj, group, env, node)
	if err != nil {
		return err
	}

	return nil
}
