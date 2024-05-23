/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd_file

import (
	"fmt"
	"os"

	ssh_executor "github.com/MottainaiCI/ssh-compose/pkg/executor"
	loader "github.com/MottainaiCI/ssh-compose/pkg/loader"
	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"

	"github.com/spf13/cobra"
)

func NewPullCommand(config *specs.SshComposeConfig) *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "pull [node] [remote-path] [local-path]",
		Aliases: []string{"pu", "pl", "fetch", "download"},
		Args:    cobra.MinimumNArgs(3),
		Short:   "Sync files from remote to local path.",
		Run: func(cmd *cobra.Command, args []string) {
			localAsTarget, _ := cmd.Flags().GetBool("local-as-target")
			ensurePerms, _ := cmd.Flags().GetBool("ensure-perms")

			// Create Instance also if not really used but
			// contains the right setup of the logger and the
			// load of the remotes.
			composer, err := loader.NewSshCInstance(config)
			if err != nil {
				fmt.Println("error on setup instance", err.Error())
				os.Exit(1)
			}

			remoteName := args[0]
			remotePath := args[1]
			localPath := args[2]

			remotes := composer.GetRemotes()
			logger := composer.GetLogger()

			if !remotes.HasRemote(remoteName) {
				fmt.Println(fmt.Sprintf("Remote %s not found.", remoteName))
				os.Exit(1)
			}
			remote := remotes.GetRemote(remoteName)

			executor, err := ssh_executor.NewSshCExecutorFromRemote(remoteName, remote)
			if err != nil {
				fmt.Println("Error on create executor:" + err.Error() + "\n")
				os.Exit(1)
			}
			err = executor.Setup()
			if err != nil {
				fmt.Println("Error on setup executor:" + err.Error() + "\n")
				os.Exit(1)
			}
			defer executor.Close()

			logger.InfoC(
				fmt.Sprintf("%s%s",
					":satellite: ",
					logger.Aurora.Bold(
						fmt.Sprintf("Using remote:\t%s",
							remoteName))))

			err = executor.SetupSftp()
			if err != nil {
				fmt.Println("Error on setup sftp client on executor of the node " +
					remoteName + ": " + err.Error())
				os.Exit(1)
			}

			logger.InfoC(
				fmt.Sprintf("%s%s",
					":delivery_truck:",
					logger.Aurora.Bold(
						fmt.Sprintf(">>> [%s] Pulling %s -> %s ",
							remoteName, remotePath, localPath))))

			err = executor.RecursivePullFile(remoteName,
				remotePath, localPath, localAsTarget, ensurePerms)
			if err != nil {
				fmt.Println("Error on pull files from " +
					remoteName + ": " + err.Error())
				os.Exit(1)
			}

			logger.InfoC(":tada:All done!")
		},
	}

	var flags = cmd.Flags()
	flags.Bool("local-as-target", true,
		"Using the local path as target path instead of append all remote path.")
	flags.Bool("ensure-perms", false,
		"Force sync of the remote uid/gid and file modes on local copy.")

	return cmd
}
