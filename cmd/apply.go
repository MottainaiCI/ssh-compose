/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	"fmt"
	"os"

	loader "github.com/MottainaiCI/ssh-compose/pkg/loader"
	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"

	"github.com/spf13/cobra"
)

func newApplyCommand(config *specs.SshComposeConfig) *cobra.Command {
	var enabledFlags []string
	var disabledFlags []string
	var enabledGroups []string
	var disabledGroups []string
	var envs []string
	var renderEnvs []string
	var varsFiles []string

	var cmd = &cobra.Command{
		Use:     "apply [list-of-projects]",
		Short:   "Deploy projects.",
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

			skipSync, _ := cmd.Flags().GetBool("skip-sync")
			skipCompile, _ := cmd.Flags().GetBool("skip-compile")

			composer.SetFlagsDisabled(disabledFlags)
			composer.SetFlagsEnabled(enabledFlags)
			composer.SetGroupsDisabled(disabledGroups)
			composer.SetGroupsEnabled(enabledGroups)
			composer.SetSkipSync(skipSync)
			composer.SetSkipCompile(skipCompile)

			projects := args[0:]

			for _, proj := range projects {

				logger.InfoC(
					logger.Aurora.Bold(fmt.Sprintf(">>> Applying project :right_arrow:%s :rocket:", proj)))

				env := composer.GetEnvByProjectName(proj)
				if env == nil {
					logger.Fatal("Project " + proj + " not found")
				}

				pObj := env.GetProjectByName(proj)
				for _, varFile := range varsFiles {
					err := pObj.LoadEnvVarsFile(varFile, config)
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

				err = composer.ApplyProject(proj)
				if err != nil {
					logger.Fatal(fmt.Sprintf(
						"Project %s failed. %s", proj, err.Error()))
				}

			}

			logger.InfoC(":tada:All done!")
		},
	}

	flags := cmd.Flags()
	flags.StringSliceVar(&enabledFlags, "enable-flag", []string{},
		"Run hooks of only specified flags.")
	flags.StringSliceVar(&disabledFlags, "disable-flag", []string{},
		"Disable execution of the hooks with the specified flags.")

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
	flags.Bool("skip-sync", false, "Disable sync of files.")
	flags.Bool("skip-compile", false, "Disable compile of templates.")

	return cmd
}
