/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	//loader "github.com/MottainaiCI/ssh-compose/pkg/loader"
	ssh_executor "github.com/MottainaiCI/ssh-compose/pkg/executor"
	loader "github.com/MottainaiCI/ssh-compose/pkg/loader"
	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"

	"github.com/spf13/cobra"
)

func NewTunnelCommand(config *specs.SshComposeConfig) *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "tunnel [remote]",
		Aliases: []string{"t", "tu"},
		Short:   "Open a tunnel to a remote.",
		Run: func(cmd *cobra.Command, args []string) {
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

			remoteName := ""

			if len(args) == 0 && remotes.GetDefault() == "" {
				logger.Fatal("No remote selected or default remote configured.")
				os.Exit(1)
			}

			if len(args) > 0 {
				remoteName = args[0]
			} else {
				remoteName = remotes.GetDefault()
			}

			if !remotes.HasRemote(remoteName) {
				logger.Fatal(fmt.Sprintf("Remote %s not found.", remoteName))
			}

			remote := remotes.GetRemote(remoteName)

			if !remote.GetTunLocalBind() {
				logger.Fatal(fmt.Sprintf("Remote %s without tun_local_bind option enabled.", remoteName))
			}

			if !remote.HasChain() {
				logger.Fatal(fmt.Sprintf("Remote %s without tunnel chain.", remoteName))
			}

			executor, err := ssh_executor.NewSshCExecutorFromRemote(remoteName, remote)
			if err != nil {
				logger.Fatal("Error on create executor:" + err.Error() + "\n")
			}
			// Create the context used to manage
			// all SSL sessions.
			executor.Ctx, executor.Cancel = context.WithCancel(context.Background())

			_, err = executor.BuildChain()
			if err != nil {
				logger.Fatal("Error on setup SSL tunnels chain: " + err.Error() + "\n")
			}
			// Manage dynamic resize of the window
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT)
			signal.Notify(sigs, syscall.SIGTERM)

			for true {
				sig := <-sigs

				if sig == syscall.SIGTERM || sig == syscall.SIGINT {
					logger.DebugC(
						fmt.Sprintf("Received signal '%s'. Exiting", sig))
					executor.Close()
					break
				} else {
					logger.DebugC(fmt.Sprintf("Received signal '%s'. Ignoring", sig))
				}

			}
		},
	}

	return cmd
}
