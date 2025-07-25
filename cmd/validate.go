/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/MottainaiCI/ssh-compose/pkg/loader"
	"github.com/MottainaiCI/ssh-compose/pkg/specs"

	"github.com/spf13/cobra"
)

func newValidateCommand(config *specs.SshComposeConfig) *cobra.Command {
	var renderEnvs []string

	var cmd = &cobra.Command{
		Use:     "validate",
		Aliases: []string{"v", "va"},
		Short:   "Validate environments.",
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			ignoreError, _ := cmd.Flags().GetBool("ignore-errors")

			// Create Instance also if not really used but
			// contains the right setup of the logger and the
			// load of the remotes.
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

			err = composer.Validate(ignoreError)
			if err != nil {
				logger.Fatal(err.Error())
			}

			logger.InfoC(fmt.Sprintf("%s%s",
				":rainbow: ",
				logger.Aurora.Bold("The environments are good!")))
		},
	}

	pflags := cmd.Flags()
	pflags.BoolP("ignore-errors", "i", false, "Ignore errors and print duplicate.")
	pflags.StringArrayVar(&renderEnvs, "render-env", []string{},
		"Append render engine environments in the format key=value.")

	return cmd
}
