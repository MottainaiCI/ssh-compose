/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	. "github.com/MottainaiCI/ssh-compose/cmd/command"
	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"

	"github.com/spf13/cobra"
)

func newCommandCommand(config *specs.SshComposeConfig) *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "command [command] [OPTIONS]",
		Aliases: []string{"c"},
		Short:   "Execute specific commands operations.",
		Args:    cobra.NoArgs,
	}

	cmd.AddCommand(
		NewListCommand(config),
		NewRunCommand(config),
	)

	return cmd
}
