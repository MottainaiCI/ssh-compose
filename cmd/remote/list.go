/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd_remote

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	loader "github.com/MottainaiCI/ssh-compose/pkg/loader"
	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"

	tablewriter "github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func NewListCommand(config *specs.SshComposeConfig) *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"l", "li"},
		Short:   "List remotes availables.",
		Run: func(cmd *cobra.Command, args []string) {
			jsonOutput, _ := cmd.Flags().GetBool("json")
			labels, _ := cmd.Flags().GetStringArray("label")

			// Create Instance
			composer, err := loader.NewSshCInstance(config)
			if err != nil {
				fmt.Println("Error on setup sshc instance:" + err.Error() + "\n")
				os.Exit(1)
			}

			logger := composer.GetLogger()

			remotes := composer.GetRemotes().Remotes

			if len(labels) > 0 {
				remotes = make(map[string]*specs.Remote, 0)

				for name, remote := range composer.GetRemotes().Remotes {

					hasLabel := false
					for _, l := range labels {
						if remote.HasLabel(l) {
							hasLabel = true
							break
						}
					}

					if hasLabel {
						remotes[name] = remote
					}
				}
			}

			if jsonOutput {

				data, err := json.Marshal(remotes)
				if err != nil {
					logger.Fatal("Error on decode projects ", err.Error())
				}
				fmt.Println(string(data))

			} else {

				table := tablewriter.NewWriter(os.Stdout)
				table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
				table.SetCenterSeparator("|")
				table.SetHeader([]string{
					"Name", "URL", "AuthMethod", "User",
				})
				table.SetAutoWrapText(false)

				remoteNames := []string{}
				for name := range remotes {
					remoteNames = append(remoteNames, name)
				}
				sort.Strings(remoteNames)

				for _, remote := range remoteNames {
					r := composer.GetRemotes().GetRemote(remote)
					portstr := ":"
					if r.GetPort() > 0 {
						portstr += fmt.Sprintf("%d", r.GetPort())
					} else {
						portstr = ""
					}

					if remote == composer.GetRemotes().GetDefault() {
						remote = remote + " (default)"
					}

					table.Append([]string{
						remote,
						fmt.Sprintf("%s::%s%s", r.GetProtocol(), r.GetHost(), portstr),
						r.GetAuthMethod(),
						r.GetUser(),
					})
				}

				table.Render()
			}

		},
	}

	var flags = cmd.Flags()
	flags.Bool("json", false, "JSON output")
	flags.StringArray("label", []string{}, "Filter remotes with specific labels.")

	return cmd
}
