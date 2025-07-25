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

func NewDelCommand(config *specs.SshComposeConfig) *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "del [remote-name]",
		Aliases: []string{"d", "rm"},
		Short:   "Add new remote endpoint.",
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("No remote name defined.")
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			remoteName := args[0]

			// Create Instance also if not really used but
			// contains the right setup of the logger and the
			// load of the remotes.
			composer, err := loader.NewSshCInstance(config)
			if err != nil {
				fmt.Println("error on setup instance", err.Error())
				os.Exit(1)
			}

			remotes := composer.GetRemotes()
			logger := composer.GetLogger()

			if !remotes.HasRemote(remoteName) {
				logger.Fatal(fmt.Sprintf("Remote %s not present.", remoteName))
			}

			remotes.DelRemote(remoteName)

			if remotes.GetDefault() == remoteName {
				remotes.SetDefault("")
			}

			// Write config
			err = remotes.Write()
			if err != nil {
				logger.Fatal("error on update remote config file:", err.Error())
			}

			logger.InfoC(fmt.Sprintf(":tada: Remote %s removed.", remoteName))
		},
	}

	return cmd
}
