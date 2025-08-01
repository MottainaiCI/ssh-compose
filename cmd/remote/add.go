/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd_remote

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/MottainaiCI/ssh-compose/pkg/helpers"
	loader "github.com/MottainaiCI/ssh-compose/pkg/loader"
	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"

	"github.com/spf13/cobra"
)

func NewAddCommand(config *specs.SshComposeConfig) *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "add [remote-name] [flags]",
		Aliases: []string{"a", "i"},
		Short:   "Add new remote endpoint.",
		PreRun: func(cmd *cobra.Command, args []string) {
			authMethod, _ := cmd.Flags().GetString("auth-method")
			host, _ := cmd.Flags().GetString("host")
			user, _ := cmd.Flags().GetString("user")
			pass, _ := cmd.Flags().GetString("pass")
			privatekeyFile, _ := cmd.Flags().GetString("privatekey-file")
			privatekeyRaw, _ := cmd.Flags().GetString("privatekey-raw")

			if authMethod != "" && authMethod == specs.AuthMethodPassword {
				if user == "" || pass == "" {
					fmt.Println("Used --auth-method=password without --user|--pass")
					os.Exit(1)
				}
			}

			if authMethod != "" && authMethod == specs.AuthMethodPublickey {
				if privatekeyFile == "" && privatekeyRaw == "" {
					fmt.Println("Used --auth-method=publickey without --privatekey-file or --privatekey-raw")
					os.Exit(1)
				}
			}

			if len(args) == 0 {
				fmt.Println("No remote name defined.")
				os.Exit(1)
			}

			if host == "" {
				fmt.Println("Invalid or missing --host parameter")
				os.Exit(1)
			}

		},
		Run: func(cmd *cobra.Command, args []string) {
			authMethod, _ := cmd.Flags().GetString("auth-method")
			host, _ := cmd.Flags().GetString("host")
			port, _ := cmd.Flags().GetString("port")
			user, _ := cmd.Flags().GetString("user")
			pass, _ := cmd.Flags().GetString("pass")
			privatekeyFile, _ := cmd.Flags().GetString("privatekey-file")
			privatekeyFilePass, _ := cmd.Flags().GetString("privatekey-pass")
			privatekeyRaw, _ := cmd.Flags().GetString("privatekey-raw")
			protocol, _ := cmd.Flags().GetString("protocol")
			defaultRemote, _ := cmd.Flags().GetBool("default")

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

			if remotes.HasRemote(remoteName) {
				logger.Fatal(fmt.Sprintf("Remote %s already present.", remoteName))
			}

			portNum := 22
			if port != "" {
				portNum, err = strconv.Atoi(port)
				if err != nil {
					logger.Fatal("Invalid value for port")
				}
			}

			remote := specs.NewRemote(host, protocol, authMethod, portNum)
			remote.SetUser(user)
			remote.SetPass(pass)
			remote.SetPrivateKeyFile(privatekeyFile)
			remote.SetPrivateKeyPass(privatekeyFilePass)

			if privatekeyRaw != "" {
				// The file could be defined as relative path or abs path.
				// The relative path is based on the path of the config file.
				file := privatekeyRaw
				if !strings.HasPrefix(file, "/") {
					configdir, err := remotes.GetAbsConfigDir()
					if err != nil {
						logger.Fatal("Error on retrieve abs path of the config", err.Error())
					}
					file = filepath.Join(configdir, file)
				}

				if helpers.Exists(file) {
					logger.Fatal(fmt.Sprintf(
						"The file %s doesn't exist.", file))
				}

				// Read the file
				data, err := os.ReadFile(file)
				if err != nil {
					logger.Fatal(fmt.Sprintf(
						"error on read file %s: %s", file, err.Error()))
				}

				remote.SetPrivateKeyRaw(string(data))
			}

			remotes.AddRemote(remoteName, remote)
			if defaultRemote {
				remotes.SetDefault(remoteName)
			}

			// Write config
			err = remotes.Write()
			if err != nil {
				logger.Fatal("error on update remote config file:", err.Error())
			}

			logger.InfoC(fmt.Sprintf(":tada: Remote %s created.", remoteName))
		},
	}

	var flags = cmd.Flags()
	flags.Bool("default", false, "Set the new remote as default endpoint.")
	flags.String("protocol", "tcp", "Define the protocol to use: tcp|tcp4|tcp6")
	flags.String("auth-method", "password", "Define the auth-method: password|publickey")
	flags.String("host", "", "Define the host of the remote.")
	flags.String("port", "", "Define the port of the remote.")
	flags.String("user", "", "Define the user to use for the remote.")
	flags.String("pass", "", "Define the password to use for the remote.")
	flags.String("privatekey-file", "", "Define the private key file path for the remote.")
	flags.String("privatekey-pass", "", "Define the password of the private key file for the remote.")
	flags.String("privatekey-raw", "", "Define the path of the file to read with the private key for the remote.")

	return cmd
}
