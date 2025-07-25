/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	. "github.com/MottainaiCI/ssh-compose/cmd/group"
	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"

	"github.com/spf13/cobra"
)

func newGroupCommand(config *specs.SshComposeConfig) *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "group [command] [OPTIONS]",
		Aliases: []string{"g"},
		Short:   "Execute specific operations for groups",
		Args:    cobra.NoArgs,
	}

	cmd.AddCommand(
		NewListCommand(config),
	)

	return cmd
}
