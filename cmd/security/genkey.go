/*
Copyright © 2020-2026 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd_security

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"

	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"

	"github.com/spf13/cobra"
)

func NewGenKeyCommand(config *specs.SshComposeConfig) *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "genkey",
		Aliases: []string{"g", "gk"},
		Short:   "Generate an encryption key base64 encoded.",
		PreRun: func(cmd *cobra.Command, args []string) {
			lenKey, _ := cmd.Flags().GetUint64("length")
			if lenKey < 32 {
				fmt.Println("length of the key to small. Minimal 32 bytes.")
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			lenKey, _ := cmd.Flags().GetUint64("length")
			to, _ := cmd.Flags().GetString("to")

			key := make([]byte, lenKey)
			_, err := rand.Read(key)
			if err != nil {
				fmt.Println("Error on generate key: " + err.Error())
				os.Exit(1)
			}

			if to == "" {
				fmt.Println(base64.StdEncoding.EncodeToString(key))
			} else {
				err = os.WriteFile(to, []byte(base64.StdEncoding.EncodeToString(key)), 0644)
				if err != nil {
					fmt.Println(fmt.Sprintf("Error on write file %s: %s",
						to, err.Error()))
					os.Exit(1)
				}
			}
		},
	}

	pflags := cmd.Flags()
	pflags.Uint64P("length", "l", 64, "Define the length of the key")
	pflags.String("to", "", "Path of the keyfile to generate (stdout if not defined).")

	return cmd
}
