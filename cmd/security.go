/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	. "github.com/MottainaiCI/ssh-compose/cmd/security"
	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"

	"github.com/spf13/cobra"
)

func newSecurityCommand(config *specs.SshComposeConfig) *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "security [command] [OPTIONS]",
		Aliases: []string{"se", "sec"},
		Short:   "Execute security operations.",
		Args:    cobra.NoArgs,
	}

	cmd.AddCommand(
		NewDecryptCommand(config),
		NewEncryptCommand(config),
		NewGenKeyCommand(config),
		NewRemotesCommand(config),
	)
	return cmd
}
