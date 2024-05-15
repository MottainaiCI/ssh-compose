/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
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
			//search, _ := cmd.Flags().GetString("search")

			// Create Instance
			composer, err := loader.NewSshCInstance(config)
			if err != nil {
				fmt.Println("Error on setup sshc instance:" + err.Error() + "\n")
				os.Exit(1)
			}

			if jsonOutput {

				data, err := json.Marshal(composer.GetRemotes())
				if err != nil {
					fmt.Println("Error on decode projects ", err.Error())
					os.Exit(1)
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
				for name := range composer.GetRemotes().Remotes {
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
	flags.StringP("search", "s", "", "Regex filter to use with network name.")

	return cmd
}
