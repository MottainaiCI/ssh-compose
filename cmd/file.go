/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	. "github.com/MottainaiCI/ssh-compose/cmd/file"
	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"

	"github.com/spf13/cobra"
)

func newFileCommand(config *specs.SshComposeConfig) *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "file [command] [OPTIONS]",
		Aliases: []string{"f", "fi"},
		Short:   "Sync files from/to remote.",
		Args:    cobra.NoArgs,
	}

	cmd.AddCommand(
		NewPullCommand(config),
		NewPushCommand(config),
	)

	return cmd
}
