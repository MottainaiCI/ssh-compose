/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	. "github.com/MottainaiCI/ssh-compose/cmd/project"
	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"

	"github.com/spf13/cobra"
)

func newProjectCommand(config *specs.SshComposeConfig) *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "project [command] [OPTIONS]",
		Aliases: []string{"p", "pro"},
		Short:   "Manage projects.",
		Args:    cobra.NoArgs,
	}

	cmd.AddCommand(
		NewListCommand(config),
	)

	return cmd
}
