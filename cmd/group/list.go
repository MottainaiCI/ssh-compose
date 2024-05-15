/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd_group

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
		Use:     "list <project>",
		Aliases: []string{"l"},
		Short:   "list of groups available in the project.",
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("No project selected.")
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {

			jsonOutput, _ := cmd.Flags().GetBool("json")
			search, _ := cmd.Flags().GetString("search")

			// Create Instance
			composer, err := loader.NewSshCInstance(config)
			if err != nil {
				fmt.Println("Error on setup sshc instance:" + err.Error() + "\n")
				os.Exit(1)
			}

			err = composer.LoadEnvironments()
			if err != nil {
				fmt.Println("Error on load environments:" + err.Error() + "\n")
				os.Exit(1)
			}

			project := args[0]
			env := composer.GetEnvByProjectName(project)
			if env == nil {
				fmt.Println("Project not found")
				os.Exit(1)
			}

			proj := env.GetProjectByName(project)
			groups := *proj.GetGroups()

			if search != "" {
				ngroups := []specs.SshCGroup{}

				for _, g := range groups {
					res := helpers.RegexEntry(search, []string{g.GetName()})
					if len(res) > 0 {
						ngroups = append(ngroups, g)
					}
				}

				groups = ngroups
			}

			if jsonOutput {

				data, _ := json.Marshal(groups)
				fmt.Println(string(data))
			} else {

				table := tablewriter.NewWriter(os.Stdout)
				table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
				table.SetCenterSeparator("|")
				table.SetHeader([]string{
					"Group Name", "Description", "# Nodes",
				})
				table.SetAutoWrapText(false)

				for _, g := range groups {
					table.Append([]string{
						g.GetName(),
						g.GetDescription(),
						fmt.Sprintf("%d", len(*g.GetNodes())),
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
