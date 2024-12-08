/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package loader

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"regexp"

	helpers "github.com/MottainaiCI/ssh-compose/pkg/helpers"
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
				renderOut, err := helpers.RenderContentWithTemplates(string(content),
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
				renderOut, err := helpers.RenderContentWithTemplates(string(content),
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
					renderOut, err := helpers.RenderContentWithTemplates(string(content),
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
						env.Projects[idx].PrependHooks(hooks)
					}

				}
			}

		} else {
			i.Logger.Debug("For project", proj.Name, "no includes for hooks.")
		}

		// Load groups included hooks
		for gidx, g := range env.Projects[idx].Groups {

			if len(g.IncludeHooksFiles) > 0 {

				for _, hinclude := range g.IncludeHooksFiles {
					for _, hfile := range hinclude.GetFiles() {

						hf := path.Join(envBaseDir, hfile)
						hooks, err := i.getHooks(hfile, hf, &proj)
						if err != nil {
							return err
						}

						if hinclude.IncludeInAppend() {
							env.Projects[idx].Groups[gidx].AddHooks(hooks)
						} else {
							env.Projects[idx].Groups[gidx].PrependHooks(hooks)
						}
					}
				}

			}

			// Load nodes includes hooks
			for nidx, n := range g.Nodes {

				if len(n.IncludeHooksFiles) > 0 {

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
								env.Projects[idx].Groups[gidx].Nodes[nidx].PrependHooks(hooks)
							}
						}
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
		renderOut, err := helpers.RenderContentWithTemplates(string(content),
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
		renderOut, err := helpers.RenderContentWithTemplates(string(content),
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
