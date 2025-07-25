/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
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

func NewPushCommand(config *specs.SshComposeConfig) *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "push [node] [local-path] [remote-path]",
		Aliases: []string{"ps", "upload"},
		Args:    cobra.MinimumNArgs(3),
		Short:   "Sync files from local path to remote path.",
		Run: func(cmd *cobra.Command, args []string) {
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
			localPath := args[1]
			remotePath := args[2]

			remotes := composer.GetRemotes()
			logger := composer.GetLogger()

			if !remotes.HasRemote(remoteName) {
				logger.Fatal(fmt.Sprintf("Remote %s not found.", remoteName))
			}
			remote := remotes.GetRemote(remoteName)

			executor, err := ssh_executor.NewSshCExecutorFromRemote(remoteName, remote)
			if err != nil {
				logger.Fatal("Error on create executor:" + err.Error() + "\n")
			}
			err = executor.Setup()
			if err != nil {
				logger.Fatal("Error on setup executor:" + err.Error() + "\n")
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
				logger.Fatal("Error on setup sftp client on executor of the node " +
					remoteName + ": " + err.Error())
			}

			logger.InfoC(
				fmt.Sprintf("%s%s",
					":delivery_truck:",
					logger.Aurora.Bold(
						fmt.Sprintf(">>> [%s] Pusing %s -> %s ",
							remoteName, localPath, remotePath))))

			err = executor.RecursivePushFile(remoteName,
				localPath, remotePath, ensurePerms)
			if err != nil {
				logger.Fatal("Error on push files to " +
					remoteName + ": " + err.Error())
			}

			logger.InfoC(":tada:All done!")
		},
	}

	var flags = cmd.Flags()
	flags.Bool("ensure-perms", false,
		"Force sync of the local uid/gid and file modes to the remote copy.")

	return cmd
}
