/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	ssh_executor "github.com/MottainaiCI/ssh-compose/pkg/executor"
	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"

	"github.com/spf13/cobra"
)

func NewExecCommand(config *specs.SshComposeConfig) *cobra.Command {
	var envs []string

	var cmd = &cobra.Command{
		Use:     "exec [remote] [command] -- [command-flags]",
		Aliases: []string{"e", "exec"},
		Short:   "Execute a command to a node or a list of nodes.",
		Args:    cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			withoutEnvs, _ := cmd.Flags().GetBool("without-envs")

			remoteName := args[0]

			remotes, err := specs.LoadRemotesConfig(
				config.GetGeneral().RemotesConfDir,
			)
			if err != nil {
				fmt.Println("Error:", err.Error())
				os.Exit(1)
			}

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

			session, err := executor.GetSession("my-session")
			if err != nil {
				fmt.Println("Error on get session:" + err.Error() + "\n")
				os.Exit(1)
			}

			if !withoutEnvs {
				err = session.Setenv("SSH_COMPOSE_VERSION", specs.SSH_COMPOSE_VERSION)
				if err != nil {
					fmt.Println("ERR on set env", err.Error())
				}

				if len(envs) > 0 {
					for _, env := range envs {
						e := specs.NewEnvVars()
						err := e.AddKVAggregated(env)
						if err != nil {
							fmt.Println(fmt.Sprintf(
								"Invalid env variable %s: %s", env, err.Error()))
							continue
						}

						for k, v := range e.EnvVars {
							err = session.Setenv(k, v.(string))
							if err != nil {
								fmt.Println(fmt.Sprintf(
									"error on set env variable %s: %s", env, err.Error()))
								continue
							}
						}

					}
				}
			}

			// set input and output
			session.Stdout = os.Stdout
			session.Stdin = os.Stdin
			session.Stderr = os.Stderr

			runArgs := strings.Join(args[1:], " ")

			fmt.Println("Running: ", runArgs)
			err = session.Run(runArgs)
			if err != nil {
				fmt.Println("error on execute command ", err.Error())
				os.Exit(1)
			}
		},
	}

	pflags := cmd.Flags()
	pflags.Bool("without-envs", false,
		"Avoid to set variables on session (ex SSH_COMPOSE_VERSION, etc.)")
	pflags.StringSliceVar(&envs, "env", []string{},
		"Append project environments in the format key=value.")

	return cmd
}
