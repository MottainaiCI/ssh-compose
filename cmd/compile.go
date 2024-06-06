/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	"fmt"
	"os"

	loader "github.com/MottainaiCI/ssh-compose/pkg/loader"
	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"
	"github.com/MottainaiCI/ssh-compose/pkg/template"

	"github.com/spf13/cobra"
)

func NewCompileCommand(config *specs.SshComposeConfig) *cobra.Command {
	var sources []string
	var enabledGroups []string
	var disabledGroups []string
	var envs []string
	var renderEnvs []string
	var varsFiles []string

	var cmd = &cobra.Command{
		Use:     "compile [list-of-projects]",
		Short:   "Compiles files of selected projects.",
		Aliases: []string{"a"},
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("No project selected.")
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {

			// Create Instance
			composer, err := loader.NewSshCInstance(config)
			if err != nil {
				fmt.Println("error on setup instance", err.Error())
				os.Exit(1)
			}

			logger := composer.GetLogger()

			// We need set this before loading phase
			err = config.SetRenderEnvs(renderEnvs)
			if err != nil {
				logger.Fatal("Error on render specs: " + err.Error() + "\n")
			}

			err = composer.LoadEnvironments()
			if err != nil {
				logger.Fatal("Error on load environments:" + err.Error() + "\n")
			}

			composer.SetGroupsDisabled(disabledGroups)
			composer.SetGroupsEnabled(enabledGroups)

			opts := template.CompilerOpts{
				Sources:        sources,
				GroupsEnabled:  enabledGroups,
				GroupsDisabled: disabledGroups,
			}

			projects := args
			for _, proj := range projects {

				env := composer.GetEnvByProjectName(proj)
				if env == nil {
					logger.Fatal("Project " + proj + " not found")
				}

				if len(varsFiles) > 0 || len(envs) > 0 {

					pObj := env.GetProjectByName(proj)

					for _, varFile := range varsFiles {
						err := pObj.LoadEnvVarsFile(varFile)
						if err != nil {
							logger.Fatal(fmt.Sprintf(
								"Error on load additional envs var file %s: %s",
								varFile, err.Error()))
						}
					}

					if len(envs) > 0 {

						evars := specs.NewEnvVars()
						for _, e := range envs {
							err := evars.AddKVAggregated(e)
							if err != nil {
								logger.Fatal(err.Error())
							}
						}

						pObj.AddEnvironment(evars)
					}
				}

				logger.InfoC(
					logger.Aurora.Bold(fmt.Sprintf(
						">>> Compile files for project :right_arrow:%s :rocket:", proj)))

				err := template.CompileAllProjectFiles(env, proj, opts)
				if err != nil {
					logger.Fatal("Error on compile files of the project " +
						proj + ":" + err.Error() + "\n")
				}

			}

			logger.InfoC(":tada:All done!")
		},
	}

	flags := cmd.Flags()
	flags.StringSliceVarP(&sources, "source-file", "f", []string{},
		"Choice the list of the source file to compile. Default: all")

	flags.StringSliceVar(&disabledGroups, "disable-group", []string{},
		"Skip selected group from deploy.")
	flags.StringSliceVar(&enabledGroups, "enable-group", []string{},
		"Apply only selected groups.")
	flags.StringArrayVar(&envs, "env", []string{},
		"Append project environments in the format key=value.")
	flags.StringArrayVar(&renderEnvs, "render-env", []string{},
		"Append render engine environments in the format key=value.")
	flags.StringSliceVar(&varsFiles, "vars-file", []string{},
		"Add additional environments vars file.")

	return cmd
}
