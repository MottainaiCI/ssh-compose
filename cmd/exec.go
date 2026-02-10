/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	ssh_executor "github.com/MottainaiCI/ssh-compose/pkg/executor"
	"github.com/MottainaiCI/ssh-compose/pkg/helpers"
	loader "github.com/MottainaiCI/ssh-compose/pkg/loader"
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
			multipleCommands, _ := cmd.Flags().GetBool("multiple-commands")

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

			if remote.CiscoDevice {
				dSec, _ := cmd.Flags().GetInt("deadline-secs")
				ciscoEna, _ := cmd.Flags().GetBool("cisco-ena")
				opts := ssh_executor.NewCiscoCommandOpts(ciscoEna)
				opts.OverrideDeadlineSecs = dSec

				var outBuffer, errBuffer bytes.Buffer
				envs := make(map[string]string, 0)

				if multipleCommands {

					for i := 1; i < len(args); i++ {
						runArgs := args[i]

						_, err := executor.RunCommandWithOutputOnCiscoDeviceWithDS(
							remoteName,
							runArgs, envs,
							helpers.NewNopCloseWriter(&outBuffer),
							helpers.NewNopCloseWriter(&errBuffer),
							[]string{},
							opts,
						)
						if err != nil {
							logger.Fatal("error on execute command ", err.Error())
						}

						fmt.Print(outBuffer.String())
						outBuffer.Reset()
						errBuffer.Reset()
					}

				} else {
					runArgs := strings.Join(args[1:], " ")

					_, err := executor.RunCommandWithOutputOnCiscoDeviceWithDS(
						remoteName,
						runArgs, envs,
						helpers.NewNopCloseWriter(&outBuffer),
						helpers.NewNopCloseWriter(&errBuffer),
						[]string{},
						opts,
					)
					if err != nil {
						logger.Fatal("error on execute command ", err.Error())
					}

					fmt.Print(outBuffer.String())
				}

			} else {
				session, err := executor.GetSession("my-session")
				if err != nil {
					logger.Fatal("Error on get session:" + err.Error() + "\n")
				}

				if !withoutEnvs {
					err = session.Setenv(fmt.Sprintf("%s_VERSION",
						config.GetGeneral().EnvSessionPrefix),
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

				// set input and output
				session.Stdout = os.Stdout
				session.Stdin = os.Stdin
				session.Stderr = os.Stderr

				if multipleCommands {

					for i := 1; i < len(args); i++ {
						runArgs := args[i]

						logger.InfoC(
							logger.Aurora.Italic(
								logger.Aurora.BrightCyan(
									fmt.Sprintf(">>> [%s] - %s - :coffee:",
										remoteName, runArgs,
									))))

						err = session.Run(runArgs)
						if err != nil {
							logger.Fatal("error on execute command ", err.Error())
						}
					}

				} else {
					runArgs := strings.Join(args[1:], " ")

					logger.InfoC(
						logger.Aurora.Italic(
							logger.Aurora.BrightCyan(
								fmt.Sprintf(">>> [%s] - %s - :coffee:",
									remoteName, runArgs,
								))))

					err = session.Run(runArgs)
					if err != nil {
						logger.Fatal("error on execute command ", err.Error())
					}
				}
			}
		},
	}

	pflags := cmd.Flags()
	pflags.Bool("without-envs", false,
		"Avoid to set variables on session (ex SSH_COMPOSE_VERSION, etc.)")
	pflags.StringSliceVar(&envs, "env", []string{},
		"Append project environments in the format key=value.")
	pflags.Int("deadline-secs", 3,
		"Define the number of seconds wait for output. For cisco devices. Default 3.")
	pflags.Bool("cisco-ena", false,
		"The command requires cisca ena privileges (true) or not (false).")
	pflags.Bool("multiple-commands", false,
		"Sending multiple commands with multiple strings.")

	return cmd
}
