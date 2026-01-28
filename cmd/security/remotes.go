/*
Copyright © 2020-2026 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd_security

import (
	"fmt"
	"os"

	"github.com/MottainaiCI/ssh-compose/pkg/loader"
	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"

	"github.com/spf13/cobra"
)

func NewRemotesCommand(config *specs.SshComposeConfig) *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "remotes [OPTIONS]",
		Aliases: []string{"r", "re"},
		Short:   "Encrypt/Decrypt remotes file.",
		Run: func(cmd *cobra.Command, args []string) {
			var action string

			decrypt, _ := cmd.Flags().GetBool("decrypt")

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

			if remotes.Encrypted && !decrypt {
				fmt.Println("Remotes config file already encrypted. Nothing to do.")
				os.Exit(0)
			}

			if !remotes.Encrypted && decrypt {
				fmt.Println("Remotes config file already decrypted. Nothing to do.")
				os.Exit(0)
			}

			if decrypt {
				action = "decrypted"
			} else {
				action = "encrypted"
				if config.GetSecurity().Key == "" {
					logger.Fatal("Encryption key not configured")
				}
			}

			remotes.Encrypted = !decrypt

			// Write config
			err = remotes.Write(config)
			if err != nil {
				logger.Fatal("error on write remote config file:", err.Error())
			}

			logger.InfoC(fmt.Sprintf(":tada: Remotes config file %s.", action))
		},
	}

	pflags := cmd.Flags()
	pflags.Bool("decrypt", false, "Decrypt the remotes file (true) or encrypt (false)")

	return cmd
}
