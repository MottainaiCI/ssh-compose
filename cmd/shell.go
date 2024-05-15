/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	"fmt"
	"os"

	//loader "github.com/MottainaiCI/ssh-compose/pkg/loader"
	ssh_executor "github.com/MottainaiCI/ssh-compose/pkg/executor"
	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

func NewShellCommand(config *specs.SshComposeConfig) *cobra.Command {
	var envs []string

	var cmd = &cobra.Command{
		Use:     "shell [remote]",
		Aliases: []string{"s", "sh"},
		Short:   "Open a shell to a remote.",
		Run: func(cmd *cobra.Command, args []string) {
			withoutEnvs, _ := cmd.Flags().GetBool("without-envs")

			remotes, err := specs.LoadRemotesConfig(
				config.GetGeneral().RemotesConfDir,
			)
			if err != nil {
				fmt.Println("Error:", err.Error())
				os.Exit(1)
			}

			remoteName := ""

			if len(args) == 0 && remotes.GetDefault() == "" {
				fmt.Println("No remote selected or default remote configured.")
				os.Exit(1)
			}

			if len(args) > 0 {
				remoteName = args[0]
			} else {
				remoteName = remotes.GetDefault()
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

			term := os.Getenv("TERM")
			if term == "" {
				term = "linux"
			}

			session, restoreTermCb, err := executor.GetShellSessionWithTermSetup("my", term,
				os.Stdin, os.Stdout, os.Stderr)
			if err != nil {
				fmt.Println("Error on get session:" + err.Error() + "\n")
				os.Exit(1)
			}
			defer restoreTermCb()

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

			if err := session.Shell(); err != nil {
				fmt.Println("failed to start shell: ", err)
				os.Exit(1)
			}

			if err = session.Wait(); err != nil {
				if e, ok := err.(*ssh.ExitError); ok {
					switch e.ExitStatus() {
					case 130:
						break
					default:
						fmt.Println("failed to session wait: ", err)
						os.Exit(1)
					}
				}
			}
		},
	}

	pflags := cmd.Flags()
	pflags.Bool("without-envs", false,
		"Avoid to set variables on session (ex SSH_COMPOSE_VERSION, etc.)")
	pflags.StringSliceVar(&envs, "env", []string{},
		"Append environments in the format key=value\n(Only variables defined on AcceptEnv param of sshd are admitted)")

	return cmd
}
