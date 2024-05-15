/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd_remote

import (
	"fmt"
	"os"

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

			remotes, err := specs.LoadRemotesConfig(
				config.GetGeneral().RemotesConfDir,
			)
			if err != nil {
				fmt.Println("Error:", err.Error())
				os.Exit(1)
			}

			if !remotes.HasRemote(remoteName) {
				fmt.Println(fmt.Sprintf("Remote %s not present.", remoteName))
				os.Exit(1)
			}

			remotes.DelRemote(remoteName)

			if remotes.GetDefault() == remoteName {
				remotes.SetDefault("")
			}

			// Write config
			err = remotes.Write()
			if err != nil {
				fmt.Println("error on update remote config file:", err.Error())
				os.Exit(1)
			}

			fmt.Println(fmt.Sprintf("Remote %s removed.", remoteName))
		},
	}

	return cmd
}
