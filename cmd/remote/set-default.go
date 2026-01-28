/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd_remote

import (
	"fmt"
	"os"

	loader "github.com/MottainaiCI/ssh-compose/pkg/loader"
	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"

	"github.com/spf13/cobra"
)

func NewSetDefaultCommand(config *specs.SshComposeConfig) *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "set-default [remote-name]",
		Aliases: []string{"sd", "default"},
		Short:   "Set the default endpoint.",
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("No remote name defined.")
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			remoteName := args[0]

			// Create Instance
			composer, err := loader.NewSshCInstance(config)
			if err != nil {
				fmt.Println("Error on setup sshc instance:" + err.Error() + "\n")
				os.Exit(1)
			}

			remotes := composer.GetRemotes()
			logger := composer.GetLogger()

			if !remotes.HasRemote(remoteName) {
				logger.Fatal(fmt.Sprintf("Remote %s not present.", remoteName))
			}

			remotes.SetDefault(remoteName)

			// Write config
			err = remotes.Write(config)
			if err != nil {
				logger.Fatal("error on update remote config file:", err.Error())
			}

			logger.InfoC(fmt.Sprintf(":tada: Remote %s set as default.", remoteName))
		},
	}

	return cmd
}
