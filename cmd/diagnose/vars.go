/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd_diagnose

import (
	"fmt"
	"os"

	"github.com/MottainaiCI/ssh-compose/pkg/loader"
	"github.com/MottainaiCI/ssh-compose/pkg/specs"
	"github.com/MottainaiCI/ssh-compose/pkg/template"

	yamlgo "github.com/ghodss/yaml"
	"github.com/icza/dyno"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func NewVarsCommand(config *specs.SshComposeConfig) *cobra.Command {
	var renderEnvs []string
	var envs []string

	var cmd = &cobra.Command{
		Use:     "vars [project]",
		Aliases: []string{"d"},
		Short:   "Dump variables of the project.",
		Args:    cobra.MaximumNArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("Missing project name param")
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			jsonFormat, _ := cmd.Flags().GetBool("json")

			composer, err := loader.NewSshCInstance(config)
			if err != nil {
				fmt.Println("error on setup instance", err.Error())
				os.Exit(1)
			}

			logger := composer.GetLogger()

			// We need set this before loading phase
			err = config.SetRenderEnvs(renderEnvs)
			if err != nil {
				logger.Fatal("Error on render specs: " + err.Error() + "\n")
			}

			err = composer.LoadEnvironments()
			if err != nil {
				logger.Fatal("Error on load environments:" + err.Error() + "\n")
			}

			pName := args[0]

			env := composer.GetEnvByProjectName(pName)
			if env == nil {
				fmt.Println("Project " + pName + " not found")
				os.Exit(1)
			}

			proj := env.GetProjectByName(pName)

			if len(envs) > 0 {
				evars := specs.NewEnvVars()
				for _, e := range envs {
					err := evars.AddKVAggregated(e)
					if err != nil {
						logger.Fatal(
							fmt.Sprintf(
								"Error on elaborate var string %s: %s",
								e, err.Error(),
							))
					}
				}

				proj.AddEnvironment(evars)
			}

			compiler, err := template.NewProjectTemplateCompiler(env, proj)
			if err != nil {
				logger.Fatal("Error on initialize compiler: " + err.Error())
			}

			var out string

			if jsonFormat {

				// TODO: Found the issue present on yaml/yamlgo libs
				m := dyno.ConvertMapI2MapS(*compiler.GetVars()).(map[string]interface{})
				y, err := yaml.Marshal(m)
				//y, err := yaml.Marshal(*proj)
				if err != nil {
					logger.Fatal("Error on convert vars to yaml: " + err.Error())
				}
				//data, err := json.Marshal(proj)
				data, err := yamlgo.YAMLToJSON(y)
				if err != nil {
					logger.Fatal("Error on convert vars to JSON: " + err.Error())
				}

				out = string(data)

			} else {
				m := dyno.ConvertMapI2MapS(*compiler.GetVars())
				data, err := yaml.Marshal(m)
				if err != nil {
					logger.Fatal("Error on convert vars to yaml: " + err.Error())
				}

				out = string(data)
			}

			fmt.Println(string(out))
		},
	}

	flags := cmd.Flags()
	flags.Bool("json", false, "Dump variables in JSON format.")
	flags.StringArrayVar(&renderEnvs, "render-env", []string{},
		"Append render engine environments in the format key=value.")
	flags.StringArrayVar(&envs, "env", []string{},
		"Append project environments in the format key=value.")

	return cmd
}
