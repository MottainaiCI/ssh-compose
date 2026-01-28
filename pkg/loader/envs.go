/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package loader

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"

	helpers "github.com/MottainaiCI/ssh-compose/pkg/helpers"
	helpers_render "github.com/MottainaiCI/ssh-compose/pkg/helpers/render"
	helpers_sec "github.com/MottainaiCI/ssh-compose/pkg/helpers/security"
	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"
)

func (i *SshCInstance) LoadEnvironments() error {
	var regexConfs = regexp.MustCompile(`.yml$|.yaml$`)

	if len(i.Config.GetEnvironmentDirs()) == 0 {
		return errors.New("No environment directories configured.")
	}

	for _, edir := range i.Config.GetEnvironmentDirs() {
		i.Logger.Debug("Checking directory", edir, "...")

		dirEntries, err := os.ReadDir(edir)
		if err != nil {
			i.Logger.Debug("Skip dir", edir, ":", err.Error())
			continue
		}

		// NOTE: Moving to os.ReadDir from ioutil.ReadDir
		//       the array returned is of DirEntry structs.
		for _, file := range dirEntries {
			if file.IsDir() {
				continue
			}

			if !regexConfs.MatchString(file.Name()) {
				i.Logger.Debug("File", file.Name(), "skipped.")
				continue
			}

			content, err := os.ReadFile(path.Join(edir, file.Name()))
			if err != nil {
				i.Logger.Debug("On read file", file.Name(), ":", err.Error())
				i.Logger.Debug("File", file.Name(), "skipped.")
				continue
			}

			if i.Config.IsEnableRenderEngine() {
				// Render file
				renderOut, err := helpers_render.RenderContentWithTemplates(string(content),
					i.Config.RenderValuesFile,
					i.Config.RenderDefaultFile,
					file.Name(),
					i.Config.RenderEnvsVars,
					i.Config.RenderTemplatesDirs,
				)
				if err != nil {
					i.Logger.Error("Error on render file", file.Name())
					return err
				}

				content = []byte(renderOut)
			}

			env, err := specs.EnvironmentFromYaml(content, path.Join(edir, file.Name()))
			if err != nil {
				i.Logger.Debug("On parse file", file.Name(), ":", err.Error())
				i.Logger.Debug("File", file.Name(), "skipped.")
				continue
			}

			err = i.loadExtraFiles(env)
			if err != nil {
				return err
			}

			i.AddEnvironment(*env)

			// Check for encrypted vars and decrypt it if possible
			err = i.decodeEncryptedEnvVars(env)
			if err != nil {
				return err
			}

			i.Logger.Debug("Loaded environment file " + env.File)

		}

	}

	return nil
}

func (i *SshCInstance) loadExtraFiles(env *specs.SshCEnvironment) error {
	envBaseDir, err := filepath.Abs(path.Dir(env.File))
	if err != nil {
		return err
	}

	i.Logger.Debug("For environment " + env.GetBaseFile() +
		" using base dir " + envBaseDir + ".")

	// Load external command
	if len(env.IncludeCommandsFiles) > 0 {

		for _, cfile := range env.IncludeCommandsFiles {

			if !helpers.Exists(path.Join(envBaseDir, cfile)) {
				i.Logger.Warning("For environment", env.GetBaseFile(),
					"included command file", cfile,
					"is not present.")
				continue
			}

			content, err := os.ReadFile(path.Join(envBaseDir, cfile))
			if err != nil {
				i.Logger.Debug("On read file", cfile, ":", err.Error())
				i.Logger.Debug("File", cfile, "skipped.")
				continue
			}

			if i.Config.IsEnableRenderEngine() {
				// Render file
				renderOut, err := helpers_render.RenderContentWithTemplates(string(content),
					i.Config.RenderValuesFile,
					i.Config.RenderDefaultFile,
					cfile,
					i.Config.RenderEnvsVars,
					i.Config.RenderTemplatesDirs,
				)
				if err != nil {
					return err
				}

				content = []byte(renderOut)
			}

			cmd, err := specs.CommandFromYaml(content)
			if err != nil {
				i.Logger.Debug("On parse file", cfile, ":", err.Error())
				i.Logger.Debug("File", cfile, "skipped.")
				continue
			}

			i.Logger.Debug("For environment " + env.GetBaseFile() +
				" add command " + cmd.GetName())

			env.AddCommand(cmd)
		}
	}

	for idx, proj := range env.Projects {

		// Load external groups files
		if len(proj.IncludeGroupFiles) > 0 {

			// Load external groups files
			for _, gfile := range proj.IncludeGroupFiles {

				if !helpers.Exists(path.Join(envBaseDir, gfile)) {
					i.Logger.Warning("For project", proj.Name, "included group file", gfile,
						"is not present.")
					continue
				}

				content, err := os.ReadFile(path.Join(envBaseDir, gfile))
				if err != nil {
					i.Logger.Debug("On read file", gfile, ":", err.Error())
					i.Logger.Debug("File", gfile, "skipped.")
					continue
				}

				if i.Config.IsEnableRenderEngine() {
					// Render file
					renderOut, err := helpers_render.RenderContentWithTemplates(string(content),
						i.Config.RenderValuesFile,
						i.Config.RenderDefaultFile,
						gfile,
						i.Config.RenderEnvsVars,
						i.Config.RenderTemplatesDirs,
					)
					if err != nil {
						return err
					}

					content = []byte(renderOut)
				}

				grp, err := specs.GroupFromYaml(content)
				if err != nil {
					i.Logger.Debug("On parse file", gfile, ":", err.Error())
					i.Logger.Debug("File", gfile, "skipped.")
					continue
				}

				i.Logger.Debug("For project " + proj.Name + " add group " + grp.Name)

				env.Projects[idx].AddGroup(grp)
			}

		} else {
			i.Logger.Debug("For project", proj.Name, "no includes for groups.")
		}

		if len(proj.IncludeEnvFiles) > 0 {
			// Load external env vars files
			for _, efile := range proj.IncludeEnvFiles {
				evars, err := i.loadEnvFile(envBaseDir, efile, &env.Projects[idx])
				if err != nil {
					return err
				} else if evars != nil {
					env.Projects[idx].AddEnvironment(evars)
				}
			}

		}

	}

	err = i.loadIncludeHooks(env)

	return err
}

func (i *SshCInstance) loadIncludeHooks(env *specs.SshCEnvironment) error {
	envBaseDir, err := filepath.Abs(path.Dir(env.File))
	if err != nil {
		return err
	}

	for idx, proj := range env.Projects {

		if len(proj.IncludeHooksFiles) > 0 {

			for _, hinclude := range proj.IncludeHooksFiles {

				hooks2prepend := []*specs.SshCHooks{}

				for _, hfile := range hinclude.GetFiles() {

					// Load project included hooks
					hf := path.Join(envBaseDir, hfile)
					hooks, err := i.getHooks(hfile, hf, &proj)
					if err != nil {
						return err
					}

					if hinclude.IncludeInAppend() {
						env.Projects[idx].AddHooks(hooks)
					} else {
						hooks2prepend = append(hooks2prepend, hooks)
					}

				}

				if len(hooks2prepend) > 0 {
					env.Projects[idx].PrependHooksList(hooks2prepend)
				}
			}

		} else {
			i.Logger.Debug("For project", proj.Name, "no includes for hooks.")
		}

		// Load groups included hooks
		for gidx, g := range env.Projects[idx].Groups {

			if len(g.IncludeHooksFiles) > 0 {

				for _, hinclude := range g.IncludeHooksFiles {

					hooks2prepend := []*specs.SshCHooks{}

					for _, hfile := range hinclude.GetFiles() {

						hf := path.Join(envBaseDir, hfile)
						hooks, err := i.getHooks(hfile, hf, &proj)
						if err != nil {
							return err
						}

						if hinclude.IncludeInAppend() {
							env.Projects[idx].Groups[gidx].AddHooks(hooks)
						} else {
							hooks2prepend = append(hooks2prepend, hooks)
						}
					}

					if len(hooks2prepend) > 0 {
						env.Projects[idx].Groups[gidx].PrependHooksList(hooks2prepend)
					}
				}

			}

			// Load nodes includes hooks
			for nidx, n := range g.Nodes {

				if len(n.IncludeHooksFiles) > 0 {

					hooks2prepend := []*specs.SshCHooks{}

					for _, hinclude := range n.IncludeHooksFiles {
						for _, hfile := range hinclude.GetFiles() {
							hf := path.Join(envBaseDir, hfile)
							hooks, err := i.getHooks(hfile, hf, &proj)
							if err != nil {
								return err
							}

							if hinclude.IncludeInAppend() {
								env.Projects[idx].Groups[gidx].Nodes[nidx].AddHooks(hooks)
							} else {
								hooks2prepend = append(hooks2prepend, hooks)
							}
						}
					}

					if len(hooks2prepend) > 0 {
						env.Projects[idx].Groups[gidx].Nodes[nidx].PrependHooksList(hooks2prepend)
					}
				}
			}

		}

	}

	return nil
}

func (i *SshCInstance) getHooks(hfile, hfileAbs string, proj *specs.SshCProject) (*specs.SshCHooks, error) {

	ans := &specs.SshCHooks{}

	if !helpers.Exists(hfileAbs) {
		i.Logger.Warning(
			"For project", proj.Name, "included hooks file", hfile,
			"is not present.")
		return ans, nil
	}

	content, err := os.ReadFile(hfileAbs)
	if err != nil {
		i.Logger.Debug("On read file", hfile, ":", err.Error())
		i.Logger.Debug("File", hfile, "skipped.")
		return ans, nil
	}

	if i.Config.IsEnableRenderEngine() {
		// Render file
		renderOut, err := helpers_render.RenderContentWithTemplates(string(content),
			i.Config.RenderValuesFile,
			i.Config.RenderDefaultFile,
			hfile,
			i.Config.RenderEnvsVars,
			i.Config.RenderTemplatesDirs,
		)
		if err != nil {
			return ans, err
		}

		content = []byte(renderOut)
	}

	hooks, err := specs.HooksFromYaml(content)
	if err != nil {
		i.Logger.Debug("On parse file", hfile, ":", err.Error())
		i.Logger.Debug("File", hfile, "skipped.")
		return ans, nil
	}

	ans = hooks

	i.Logger.Debug("For project", proj.Name, "add",
		len(ans.Hooks), "hooks.")

	return ans, nil
}

func (i *SshCInstance) loadEnvFile(envBaseDir, efile string, proj *specs.SshCProject) (*specs.SshCEnvVars, error) {
	if !helpers.Exists(path.Join(envBaseDir, efile)) {
		i.Logger.Warning("For project", proj.Name, "included env file", efile,
			"is not present.")
		return nil, nil
	}

	i.Logger.Debug("Loaded variables file " + efile)

	if path.Ext(efile) != ".yml" && path.Ext(efile) != ".yaml" {
		i.Logger.Warning("For project", proj.Name, "included env file", efile,
			"will be used only with template compiler")
		return nil, nil
	}

	content, err := os.ReadFile(path.Join(envBaseDir, efile))
	if err != nil {
		i.Logger.Debug("On read file", efile, ":", err.Error())
		i.Logger.Debug("File", efile, "skipped.")
		return nil, nil
	}

	if i.Config.IsEnableRenderEngine() {
		// Render file
		renderOut, err := helpers_render.RenderContentWithTemplates(string(content),
			i.Config.RenderValuesFile,
			i.Config.RenderDefaultFile,
			efile,
			i.Config.RenderEnvsVars,
			i.Config.RenderTemplatesDirs,
		)
		if err != nil {
			return nil, err
		}

		content = []byte(renderOut)
	}

	evars, err := specs.EnvVarsFromYaml(content)
	if err != nil {
		i.Logger.Debug("On parse file", efile, ":", err.Error())
		i.Logger.Debug("File", efile, "skipped.")
		return nil, nil
	}

	return evars, nil
}

func (i *SshCInstance) decodeEncryptedEnvVars(env *specs.SshCEnvironment) error {

	var err error
	keyBytes := []byte{}

	if i.Config.GetSecurity().Key != "" {
		keyBytes, err = base64.StdEncoding.DecodeString(i.Config.GetSecurity().Key)
		if err != nil {
			return fmt.Errorf("error on decode base64 key: %s", err.Error())
		}
	}

	for idx := range env.Projects {
		for eidx := range env.Projects[idx].Environments {

			if env.Projects[idx].Environments[eidx].Encrypted {

				if i.Config.GetSecurity().Key == "" {
					i.Logger.Warning("Found variables encrypted but no key available. Ignoring vars.")
					goto skipDecode
				}

				// Decode encrypted content.
				encryptedContent, err := base64.StdEncoding.DecodeString(
					env.Projects[idx].Environments[eidx].EncryptedContent,
				)
				if err != nil {
					i.Logger.Warning("ignoring error on decode base64 for %s: %s",
						env.Projects[idx].Environments[eidx].EncryptedContent,
						err.Error())
					continue
				}

				dkaOpts := helpers_sec.NewDKAOptsDefault()
				if i.Config.GetSecurity().DKAOpts != nil {
					if i.Config.GetSecurity().DKAOpts.TimeIterations != nil {
						dkaOpts.TimeIterations = *i.Config.GetSecurity().DKAOpts.TimeIterations
					}
					if i.Config.GetSecurity().DKAOpts.MemoryUsage != nil {
						dkaOpts.MemoryUsage = *i.Config.GetSecurity().DKAOpts.MemoryUsage
					}
					if i.Config.GetSecurity().DKAOpts.KeyLength != nil {
						dkaOpts.KeyLength = *i.Config.GetSecurity().DKAOpts.KeyLength
					}
					if i.Config.GetSecurity().DKAOpts.Parallelism != nil {
						dkaOpts.Parallelism = *i.Config.GetSecurity().DKAOpts.Parallelism
					}
				}
				decodedBytes, err := helpers_sec.Decrypt(encryptedContent, keyBytes, dkaOpts)
				if err != nil {
					i.Logger.Warning("ignoring error on decrypt content %s: %s",
						env.Projects[idx].Environments[eidx].EncryptedContent,
						err.Error())
					continue
				}
				// Render the decrypt content
				renderOut, err := helpers_render.RenderContentWithTemplates(string(decodedBytes),
					i.Config.RenderValuesFile,
					i.Config.RenderDefaultFile,
					"-",
					i.Config.RenderEnvsVars,
					i.Config.RenderTemplatesDirs,
				)
				if err != nil {
					i.Logger.Error("Error on render encrypted vars",
						string(decodedBytes),
					)
					return err
				}

				evars, err := specs.EnvVarsFromYaml([]byte(renderOut))
				if err != nil {
					i.Logger.Debug(fmt.Sprintf(
						"On parse decrypted vars content %s:\n%s",
						renderOut, err.Error()))
					continue
				}

				env.Projects[idx].Environments[eidx].EnvVars = evars.EnvVars

			}

		}

	}

skipDecode:

	return nil
}
