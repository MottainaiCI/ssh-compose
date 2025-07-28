/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	//loader "github.com/MottainaiCI/ssh-compose/pkg/loader"
	ssh_executor "github.com/MottainaiCI/ssh-compose/pkg/executor"
	loader "github.com/MottainaiCI/ssh-compose/pkg/loader"
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

			executor, err := ssh_executor.NewSshCExecutorFromRemote(remoteName, remote)
			if err != nil {
				logger.Fatal("Error on create executor:" + err.Error() + "\n")
			}
			err = executor.Setup()
			if err != nil {
				logger.Fatal("Error on setup executor: " + err.Error() + "\n")
			}
			defer executor.Close()

			term := os.Getenv("TERM")
			if term == "" {
				term = "linux"
			}

			session, restoreTermCb, err := executor.GetShellSessionWithTermSetup("my", term,
				os.Stdin, os.Stdout, os.Stderr)
			if err != nil {
				logger.Fatal("Error on get session:" + err.Error() + "\n")
			}
			defer restoreTermCb()

			// Manage dynamic resize of the window
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGWINCH)

			go ssh_executor.ResizeWindowHandler(sigs, os.Stdin, session)

			if !withoutEnvs {
				err = session.Setenv(fmt.Sprintf(
					"%s_VERSION", config.GetGeneral().EnvSessionPrefix),
					specs.SSH_COMPOSE_VERSION)
				if err != nil {
					logger.Debug("Error on set version env", err.Error())
				}

				if len(envs) > 0 {
					for _, env := range envs {
						e := specs.NewEnvVars()
						err := e.AddKVAggregated(env)
						if err != nil {
							logger.Debug(fmt.Sprintf(
								"Invalid env variable %s: %s", env, err.Error()))
							continue
						}

						for k, v := range e.EnvVars {
							err = session.Setenv(k, v.(string))
							if err != nil {
								logger.Debug(fmt.Sprintf(
									"error on set env variable %s: %s", env, err.Error()))
								continue
							}
						}

					}
				}
			}

			if err := session.Shell(); err != nil {
				logger.Fatal("failed to start shell: ", err)
			}

			if err = session.Wait(); err != nil {
				if e, ok := err.(*ssh.ExitError); ok {
					switch e.ExitStatus() {
					case 130:
						break
					default:
						logger.Fatal("failed to session wait: ", err)
					}
				}
			}
		},
	}

	pflags := cmd.Flags()
	pflags.Bool("without-envs", false,
		fmt.Sprintf("Avoid to set variables on session (ex %s_VERSION, etc.)",
			config.GetGeneral().EnvSessionPrefix),
	)
	pflags.StringSliceVar(&envs, "env", []string{},
		"Append environments in the format key=value\n(Only variables defined on AcceptEnv param of sshd are admitted)")

	return cmd
}
