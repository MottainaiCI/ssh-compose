/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd_command

import (
	"errors"
	"fmt"
	"os"

	loader "github.com/MottainaiCI/ssh-compose/pkg/loader"
	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"

	"github.com/spf13/cobra"
)

func ApplyCommand(c *specs.SshCCommand, composer *loader.SshCInstance,
	proj *specs.SshCProject, envs []string, varsfiles []string) error {

	err := composer.ApplyCommand(c, proj, envs, varsfiles)
	if err != nil {
		return err
	}

	err = composer.ApplyProject(proj.GetName())
	if err != nil {
		return errors.New(
			fmt.Sprintf(
				"Error on apply project %s: %s",
				proj.GetName(), err.Error()),
		)
	}

	return nil
}

func NewRunCommand(config *specs.SshComposeConfig) *cobra.Command {
	var commandFiles []string
	var envs []string
	var renderEnvs []string
	var varsFiles []string
	var enabledGroups []string
	var disabledGroups []string

	var cmd = &cobra.Command{
		Use:     "run <project> <command>",
		Aliases: []string{"r"},
		Short:   "Run a command of environment commands.",
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				fmt.Println("Invalid argument. You need <project> and <command>.")
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {

			// Create Instance
			composer, err := loader.NewSshCInstance(config)
			if err != nil {
				fmt.Println("Error on setup sshc instance:" + err.Error() + "\n")
				os.Exit(1)
			}

			logger := composer.GetLogger()

			pname := args[0]
			cname := args[1]

			// We need set this before loading phase
			err = config.SetRenderEnvs(renderEnvs)
			if err != nil {
				logger.Fatal("Error on render specs: " + err.Error() + "\n")
			}

			err = composer.LoadEnvironments()
			if err != nil {
				logger.Fatal("Error on load environments:" + err.Error() + "\n")
			}

			env := composer.GetEnvByProjectName(pname)
			if env == nil {
				logger.Fatal("No project found with name " + pname)
			}

			// Load runtime commands
			if len(commandFiles) > 0 {
				for _, f := range commandFiles {
					c, err := specs.CommandFromFile(f)
					if err != nil {
						logger.Fatal(fmt.Sprintf(
							"Error on load command file %s: %s",
							f, err.Error()))
					}

					// Check if the command is already present.
					// NOTE: If this is slow it's better move to a map.
					ce, _ := env.GetCommand(c.Name)
					if ce != nil {
						cmds := []specs.SshCCommand{}

						for _, cmd := range env.Commands {
							if cmd.Name != c.Name {
								cmds = append(cmds, cmd)
							}
						}
						env.Commands = cmds
					}
					env.AddCommand(c)
				}
			}

			command, err := env.GetCommand(cname)
			if err != nil {
				logger.Fatal(
					"No command available with name " + cname +
						" on project " + pname)
			}

			command.SetDisableGroups(disabledGroups)
			command.SetEnableGroups(enabledGroups)

			err = ApplyCommand(command, composer,
				env.GetProjectByName(pname),
				envs, varsFiles,
			)
			if err != nil {
				logger.Fatal(err.Error())
			}

			logger.InfoC(":tada:All done!")
		},
	}

	var flags = cmd.Flags()
	flags.StringSliceVar(&disabledGroups, "disable-group", []string{},
		"Skip selected group from deploy.")
	flags.StringSliceVar(&enabledGroups, "enable-group", []string{},
		"Apply only selected groups.")
	flags.StringArrayVar(&renderEnvs, "render-env", []string{},
		"Append render engine environments in the format key=value.")
	flags.StringArrayVar(&envs, "env", []string{},
		"Append project environments in the format key=value.")
	flags.StringSliceVar(&varsFiles, "vars-file", []string{},
		"Add additional environments vars file.")
	flags.StringSliceVar(&commandFiles, "command-file", []string{},
		"Add additional commands file.")

	return cmd
}
