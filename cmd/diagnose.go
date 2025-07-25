/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	. "github.com/MottainaiCI/ssh-compose/cmd/diagnose"
	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"

	"github.com/spf13/cobra"
)

func newDiagnoseCommand(config *specs.SshComposeConfig) *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "diagnose [command] [OPTIONS]",
		Aliases: []string{"g"},
		Short:   "Execute diagnose on loaded environments.",
		Args:    cobra.NoArgs,
	}

	cmd.AddCommand(
		NewVarsCommand(config),
	)

	return cmd
}
