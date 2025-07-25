/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd_project

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/MottainaiCI/ssh-compose/pkg/helpers"
	loader "github.com/MottainaiCI/ssh-compose/pkg/loader"
	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"

	tablewriter "github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func NewListCommand(config *specs.SshComposeConfig) *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"l", "li"},
		Short:   "List all loaded projects.",
		Run: func(cmd *cobra.Command, args []string) {
			projects := []specs.SshCProjectSanitized{}

			jsonOutput, _ := cmd.Flags().GetBool("json")
			search, _ := cmd.Flags().GetString("search")

			// Create Instance
			composer, err := loader.NewSshCInstance(config)
			if err != nil {
				fmt.Println("Error on setup sshc instance:" + err.Error() + "\n")
				os.Exit(1)
			}

			logger := composer.GetLogger()

			err = composer.LoadEnvironments()
			if err != nil {
				logger.Fatal("Error on load environments:" + err.Error() + "\n")
			}

			for _, e := range *composer.GetEnvironments() {
				for _, p := range *e.GetProjects() {
					if search != "" {
						res := helpers.RegexEntry(search, []string{p.GetName()})
						if len(res) > 0 {
							projects = append(projects, *p.Sanitize())
						}
					} else {
						projects = append(projects, *p.Sanitize())
					}
				}
			}

			if jsonOutput {

				data, err := json.Marshal(projects)
				if err != nil {
					logger.Fatal("Error on decode projects ", err.Error())
				}
				fmt.Println(string(data))

			} else {

				table := tablewriter.NewWriter(os.Stdout)
				table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
				table.SetCenterSeparator("|")
				table.SetHeader([]string{
					"Project Name", "Description", "# Groups",
				})
				table.SetAutoWrapText(false)

				for _, p := range projects {

					table.Append([]string{
						p.GetName(),
						p.GetDescription(),
						fmt.Sprintf("%d", len(*p.GetGroups())),
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
