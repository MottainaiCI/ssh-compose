/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	. "github.com/MottainaiCI/ssh-compose/cmd/remote"
	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"

	"github.com/spf13/cobra"
)

func newRemoteCommand(config *specs.SshComposeConfig) *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "remote [command] [OPTIONS]",
		Aliases: []string{"r", "re"},
		Short:   "Manage remotes.",
		Args:    cobra.NoArgs,
	}

	cmd.AddCommand(
		NewListCommand(config),
		NewAddCommand(config),
		NewDelCommand(config),
		NewSetDefaultCommand(config),
	)

	return cmd
}
